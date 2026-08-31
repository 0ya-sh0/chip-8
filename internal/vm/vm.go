package vm

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"sync/atomic"
	"time"
)

// Machine constants.
const (
	// MemorySize is the CHIP-8 address space. It is a power of two, so
	// addresses can be wrapped with a mask instead of bounds-checked.
	MemorySize = 4096
	addrMask   = MemorySize - 1

	// ProgramStart is where ROMs are loaded and execution begins. Everything
	// below it was interpreter scratch space on original hardware.
	ProgramStart = 0x200

	// FontStart is where the built-in hex font lives. The address is
	// convention, not spec; FX29 just has to agree with it.
	FontStart  = 0x050
	FontHeight = 5

	// NumRegisters is the count of general-purpose registers. VF doubles as the
	// carry/collision flag, so ROMs must not rely on keeping a value there.
	NumRegisters = 16
	VF           = 0xF

	// StackDepth caps subroutine nesting. The COSMAC VIP allowed 12; 16 is the
	// common modern choice. A bounded stack turns a runaway ROM into an error
	// instead of unbounded memory growth.
	StackDepth = 16

	// TimerRate is the fixed 60Hz rate of the delay and sound timers. It is the
	// one hard timing number in the CHIP-8 spec.
	TimerRate    = 60
	timerTick    = time.Second / TimerRate
	DefaultCPUHz = 700
	maxROMSize   = MemorySize - ProgramStart

	// schedulerTick is how often the interpreter wakes, independent of the
	// instruction rate. Driving one instruction per timer tick caps throughput
	// at whatever the platform's timer resolution allows - measured at roughly
	// 1kHz on a host without a TSC clocksource - so instructions are instead
	// issued in small batches sized by elapsed time. That keeps the configured
	// rate accurate well past the timer's own limit.
	schedulerTick = 2 * time.Millisecond

	// maxCatchUpBatch bounds one batch. Without it a long stall would bank
	// thousands of instructions and release them in a burst that stalls things
	// further - the classic catch-up spiral. Exceeding it abandons the backlog,
	// so the emulator runs briefly slow rather than seizing.
	maxCatchUpBatch = 64
)

// Errors returned by [Chip8VM.Start] and the ROM loaders.
var (
	// ErrUnknownOpcode means the program counter reached a word that does not
	// decode to an instruction, which usually means execution ran into data.
	ErrUnknownOpcode = errors.New("vm: unknown opcode")
	// ErrStackOverflow means subroutine nesting exceeded StackDepth.
	ErrStackOverflow = errors.New("vm: call stack overflow")
	// ErrStackUnderflow means RET was executed with no matching CALL.
	ErrStackUnderflow = errors.New("vm: call stack underflow")
	// ErrROMTooLarge means the ROM does not fit between ProgramStart and the
	// top of memory.
	ErrROMTooLarge = errors.New("vm: rom too large")
)

var fontSet = [16 * FontHeight]uint8{
	0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
	0x20, 0x60, 0x20, 0x20, 0x70, // 1
	0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
	0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
	0x90, 0x90, 0xF0, 0x10, 0x10, // 4
	0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
	0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
	0xF0, 0x10, 0x20, 0x40, 0x40, // 7
	0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
	0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
	0xF0, 0x90, 0xF0, 0x90, 0x90, // A
	0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
	0xF0, 0x80, 0x80, 0x80, 0xF0, // C
	0xE0, 0x90, 0x90, 0x90, 0xE0, // D
	0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
	0xF0, 0x80, 0xF0, 0x80, 0x80, // F
}

// Chip8VM is a CHIP-8 interpreter. All state is owned by the goroutine running
// [Chip8VM.Start]; the providers are the only concurrency boundary.
type Chip8VM struct {
	ram    [MemorySize]uint8
	greg   [NumRegisters]uint8
	idxreg uint16
	pc     uint16

	buf   FrameBuffer
	stack [StackDepth]uint16
	sp    uint8

	// Timers are stored as deadlines rather than counters decremented at 60Hz.
	// Nothing outside the VM can observe the intermediate values, so simulating
	// the tick would be pure overhead - and would need its own clock, which is
	// what made the original scheduler unfair.
	delayTimerEnd time.Time
	soundTimerEnd time.Time
	// soundPlaying tracks whether PlaySound has been emitted, so the VM emits
	// edges and providers do not each have to deduplicate.
	soundPlaying bool

	key     KeyboardProvider
	display DisplayProvider
	sound   SoundProvider

	cpuHz int
	now   func() time.Time
	rng   *rand.Rand

	metrics metrics
}

type metrics struct {
	opcodes      [NumOpcodeTypes]atomic.Uint64
	instructions atomic.Uint64
	draws        atomic.Uint64
	drawSamples  atomic.Uint64
	drawNanos    atomic.Uint64
	drawMaxNanos atomic.Uint64
	// startedAt is Unix nanoseconds, written once by the interpreter goroutine
	// and read by whatever goroutine serves Stats. A plain time.Time here is a
	// data race - one the race detector reliably reports.
	startedAt atomic.Int64
}

// drawSampleMask selects which draws are timed: one in every 64.
//
// Timing a call costs two clock reads, and time.Now() is only cheap when the
// kernel exposes a TSC-backed vDSO clock. On a host stuck on HPET it is a ~3us
// MMIO read - several times the cost of the draw being measured, which would
// make the instrumentation the dominant term in its own metric. Sampling keeps
// the average and maximum meaningful at a fraction of the overhead.
const drawSampleMask = 15

// Option configures a [Chip8VM].
type Option func(*Chip8VM)

// WithCPURate sets the instruction rate in Hz. CHIP-8 has no specified clock
// speed; the COSMAC VIP managed roughly 1500-2000 instructions per second and
// most ROMs are happy between 500 and 1000.
//
// This is a gameplay knob, not a latency fix. It does shorten the interval
// between a ROM's key polls, since a keypad scan costs a fixed number of
// instructions - but it speeds the whole game up to get there. Dropped input is
// fixed in [KeyLatch] instead, so the two concerns stay independent.
func WithCPURate(hz int) Option {
	return func(v *Chip8VM) {
		if hz > 0 {
			v.cpuHz = hz
		}
	}
}

// WithClock replaces the time source, making timer behaviour deterministic in
// tests.
func WithClock(now func() time.Time) Option {
	return func(v *Chip8VM) { v.now = now }
}

// WithRandom replaces the random source used by CXNN, so tests can assert on
// exact register values.
func WithRandom(r *rand.Rand) Option {
	return func(v *Chip8VM) { v.rng = r }
}

// NewChip8VM returns a VM wired to the given providers.
func NewChip8VM(key KeyboardProvider, display DisplayProvider, sound SoundProvider, opts ...Option) *Chip8VM {
	v := &Chip8VM{
		key:     key,
		display: display,
		sound:   sound,
		pc:      ProgramStart,
		cpuHz:   DefaultCPUHz,
		now:     time.Now,
		rng:     rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
	}
	copy(v.ram[FontStart:], fontSet[:])
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// LoadROM copies a ROM image to ProgramStart. It reports [ErrROMTooLarge]
// rather than silently truncating.
func (vm *Chip8VM) LoadROM(data []byte) error {
	if len(data) > maxROMSize {
		return fmt.Errorf("%w: %d bytes, limit %d", ErrROMTooLarge, len(data), maxROMSize)
	}
	copy(vm.ram[ProgramStart:], data)
	return nil
}

// LoadROMFromFile reads a ROM image from disk and loads it.
func (vm *Chip8VM) LoadROMFromFile(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("vm: read rom: %w", err)
	}
	return vm.LoadROM(data)
}

// Start runs the interpreter until ctx is cancelled or the ROM faults.
//
// It returns ctx.Err() on cancellation and a wrapped sentinel error otherwise,
// so callers can distinguish "the user quit" from "this ROM is broken".
func (vm *Chip8VM) Start(ctx context.Context) error {
	// The buzzer is stopped on every exit path, including a panic. Without
	// this, a VM that faults mid-beep leaves the sound on forever, because the
	// only other StopSound call lives on the instruction path.
	defer func() {
		if vm.soundPlaying {
			vm.soundPlaying = false
			vm.sound.StopSound()
		}
	}()

	ticker := time.NewTicker(schedulerTick)
	defer ticker.Stop()

	vm.metrics.startedAt.Store(vm.now().UnixNano())
	last := vm.now()

	// credit is the number of instructions owed, carried across ticks as a
	// fraction so the long-run rate stays exact even when a tick period is not
	// a whole number of instructions.
	var credit float64

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			now := vm.now()
			credit += now.Sub(last).Seconds() * float64(vm.cpuHz)
			last = now
			if credit > maxCatchUpBatch {
				credit = maxCatchUpBatch
			}

			vm.updateTimers()

			for credit >= 1 {
				credit--
				op := ParseOpcode([2]uint8{vm.mem(vm.pc), vm.mem(vm.pc + 1)})
				if op.OpcodeType == OpNA {
					return fmt.Errorf("%w: %02X%02X at 0x%03X",
						ErrUnknownOpcode, vm.mem(vm.pc), vm.mem(vm.pc+1), vm.pc)
				}
				if err := vm.exec(op); err != nil {
					return err
				}
			}
		}
	}
}

// updateTimers emits the buzzer-stop edge when the sound timer expires. The
// delay timer needs no upkeep: FX07 derives its value from the deadline.
func (vm *Chip8VM) updateTimers() {
	if vm.soundPlaying && !vm.now().Before(vm.soundTimerEnd) {
		vm.soundPlaying = false
		vm.sound.StopSound()
	}
}

// mem reads RAM with the address wrapped into range. CHIP-8 ROMs can compute
// out-of-range indices (I is 16 bits addressing 12 bits of memory); wrapping
// keeps a buggy ROM from panicking the whole server.
func (vm *Chip8VM) mem(addr uint16) uint8 { return vm.ram[addr&addrMask] }

func (vm *Chip8VM) setMem(addr uint16, val uint8) { vm.ram[addr&addrMask] = val }

// skip advances past the next instruction, for the conditional-skip opcodes.
func (vm *Chip8VM) skip() { vm.pc = (vm.pc + 4) & addrMask }

func (vm *Chip8VM) next() { vm.pc = (vm.pc + 2) & addrMask }

func (vm *Chip8VM) exec(oc ParsedOpcode) error {
	vm.metrics.opcodes[oc.OpcodeType].Add(1)
	vm.metrics.instructions.Add(1)

	switch oc.OpcodeType {
	case Op0NNN:
		// Calls an RCA 1802 routine on real hardware. Modern ROMs do not use
		// it and every modern interpreter ignores it.

	case Op00E0:
		// Display: clear the screen.
		vm.buf = FrameBuffer{}
		vm.display.Clear()

	case Op00EE:
		// Flow: return from subroutine.
		if vm.sp == 0 {
			return ErrStackUnderflow
		}
		vm.sp--
		vm.pc = vm.stack[vm.sp]
		return nil

	case Op1NNN:
		// Flow: jump to NNN.
		vm.pc = oc.NNN()
		return nil

	case Op2NNN:
		// Flow: call subroutine at NNN.
		if int(vm.sp) >= StackDepth {
			return ErrStackOverflow
		}
		vm.stack[vm.sp] = (vm.pc + 2) & addrMask
		vm.sp++
		vm.pc = oc.NNN()
		return nil

	case Op3XNN:
		// Cond: skip next instruction if VX == NN.
		if vm.greg[oc.X()] == oc.NN() {
			vm.skip()
			return nil
		}

	case Op4XNN:
		// Cond: skip next instruction if VX != NN.
		if vm.greg[oc.X()] != oc.NN() {
			vm.skip()
			return nil
		}

	case Op5XY0:
		// Cond: skip next instruction if VX == VY.
		if vm.greg[oc.X()] == vm.greg[oc.Y()] {
			vm.skip()
			return nil
		}

	case Op9XY0:
		// Cond: skip next instruction if VX != VY.
		if vm.greg[oc.X()] != vm.greg[oc.Y()] {
			vm.skip()
			return nil
		}

	case Op6XNN:
		// Const: VX = NN.
		vm.greg[oc.X()] = oc.NN()

	case Op7XNN:
		// Const: VX += NN. Carry is explicitly not affected.
		vm.greg[oc.X()] += oc.NN()

	case Op8XY0:
		vm.greg[oc.X()] = vm.greg[oc.Y()]

	case Op8XY1:
		vm.greg[oc.X()] |= vm.greg[oc.Y()]

	case Op8XY2:
		vm.greg[oc.X()] &= vm.greg[oc.Y()]

	case Op8XY3:
		vm.greg[oc.X()] ^= vm.greg[oc.Y()]

	case Op8XY4:
		// VX += VY, VF = carry. The flag is written after the result so that
		// "ADD VF, VY" leaves the flag, not the sum.
		sum := uint16(vm.greg[oc.X()]) + uint16(vm.greg[oc.Y()])
		vm.greg[oc.X()] = uint8(sum)
		vm.greg[VF] = b2u(sum > 0xFF)

	case Op8XY5:
		// VX -= VY, VF = 1 when there is no borrow.
		vx, vy := vm.greg[oc.X()], vm.greg[oc.Y()]
		vm.greg[oc.X()] = vx - vy
		vm.greg[VF] = b2u(vx >= vy)

	case Op8XY7:
		// VX = VY - VX, VF = 1 when there is no borrow.
		vx, vy := vm.greg[oc.X()], vm.greg[oc.Y()]
		vm.greg[oc.X()] = vy - vx
		vm.greg[VF] = b2u(vy >= vx)

	case Op8XY6:
		// VX >>= 1, VF = shifted-out bit. CHIP-48/SUPER-CHIP semantics: VY is
		// ignored. The original COSMAC VIP shifted VY into VX instead; most
		// modern ROMs expect this variant.
		vx := vm.greg[oc.X()]
		vm.greg[oc.X()] = vx >> 1
		vm.greg[VF] = vx & 0x01

	case Op8XYE:
		// VX <<= 1, VF = shifted-out bit. Same quirk choice as 8XY6.
		vx := vm.greg[oc.X()]
		vm.greg[oc.X()] = vx << 1
		vm.greg[VF] = vx >> 7

	case OpANNN:
		// MEM: I = NNN.
		vm.idxreg = oc.NNN()

	case OpBNNN:
		// Flow: jump to NNN + V0. Original COSMAC VIP semantics; SUPER-CHIP
		// instead used XNN + VX.
		vm.pc = (oc.NNN() + uint16(vm.greg[0])) & addrMask
		return nil

	case OpCXNN:
		vm.greg[oc.X()] = uint8(vm.rng.UintN(256)) & oc.NN()

	case OpDXYN:
		vm.draw(oc)

	case OpEX9E:
		// Cond: skip if the key in VX is down.
		if vm.key.IsKeyPressed(vm.greg[oc.X()] & 0x0F) {
			vm.skip()
			return nil
		}

	case OpEXA1:
		// Cond: skip if the key in VX is up.
		if !vm.key.IsKeyPressed(vm.greg[oc.X()] & 0x0F) {
			vm.skip()
			return nil
		}

	case OpFX0A:
		// Blocks until a key is down. Re-executing the same instruction is how
		// the wait is implemented, so the program counter must not advance.
		key, pressed := vm.key.GetPressedKey()
		if !pressed {
			return nil
		}
		vm.greg[oc.X()] = key

	case OpFX07:
		// Reconstruct the 60Hz counter from the deadline. Rounding up means a
		// partially elapsed tick still counts, matching a decrementing timer.
		var remaining uint8
		if d := vm.delayTimerEnd.Sub(vm.now()); d > 0 {
			remaining = uint8((d + timerTick - 1) / timerTick)
		}
		vm.greg[oc.X()] = remaining

	case OpFX15:
		// Setting the timer to zero must clear a running one, so the deadline
		// is always assigned - only the provider call is conditional.
		vm.delayTimerEnd = vm.now().Add(time.Duration(vm.greg[oc.X()]) * timerTick)

	case OpFX18:
		n := vm.greg[oc.X()]
		vm.soundTimerEnd = vm.now().Add(time.Duration(n) * timerTick)
		if n > 0 && !vm.soundPlaying {
			vm.soundPlaying = true
			vm.sound.PlaySound()
		}

	case OpFX1E:
		vm.idxreg += uint16(vm.greg[oc.X()])

	case OpFX29:
		// I points at the 5-byte font sprite for the low nibble of VX.
		vm.idxreg = FontStart + uint16(vm.greg[oc.X()]&0x0F)*FontHeight

	case OpFX33:
		// BCD: store the three decimal digits of VX at I, I+1, I+2.
		vx := vm.greg[oc.X()]
		vm.setMem(vm.idxreg, vx/100)
		vm.setMem(vm.idxreg+1, vx/10%10)
		vm.setMem(vm.idxreg+2, vx%10)

	case OpFX55:
		// Store V0..VX at I onwards. Modern semantics: I is left unmodified.
		for i := uint16(0); i <= uint16(oc.X()); i++ {
			vm.setMem(vm.idxreg+i, vm.greg[i])
		}

	case OpFX65:
		// Load V0..VX from I onwards. Modern semantics: I is left unmodified.
		for i := uint16(0); i <= uint16(oc.X()); i++ {
			vm.greg[i] = vm.mem(vm.idxreg + i)
		}

	default:
		return fmt.Errorf("%w: %s", ErrUnknownOpcode, oc.OpcodeType)
	}

	vm.next()
	return nil
}

// draw implements DXYN. The sprite origin wraps around the screen, but the
// sprite body clips at the edges rather than wrapping.
func (vm *Chip8VM) draw(oc ParsedOpcode) {
	x0 := int(vm.greg[oc.X()] % ScreenWidth)
	y0 := int(vm.greg[oc.Y()] % ScreenHeight)

	vm.greg[VF] = 0
	for row := 0; row < int(oc.N()); row++ {
		y := y0 + row
		if y >= ScreenHeight {
			break
		}
		sprite := vm.mem(vm.idxreg + uint16(row))
		for col := 0; col < 8; col++ {
			x := x0 + col
			if x >= ScreenWidth {
				break
			}
			if sprite&(0x80>>col) == 0 {
				continue
			}
			if vm.buf[y][x] {
				// A lit pixel being turned off is a collision.
				vm.greg[VF] = 1
			}
			vm.buf[y][x] = !vm.buf[y][x]
		}
	}

	// A slow Draw stalls the whole interpreter, and a stalled interpreter stops
	// sampling the keypad, so this is the number to watch. See drawSampleMask
	// for why only some calls are timed.
	n := vm.metrics.draws.Add(1)
	// Offset by one so the very first draw is always sampled; otherwise a
	// short-lived session reports no timing at all.
	sample := n&drawSampleMask == 1

	var start time.Time
	if sample {
		start = vm.now()
	}
	vm.display.Draw(vm.buf)
	if !sample {
		return
	}

	elapsed := uint64(vm.now().Sub(start))
	vm.metrics.drawSamples.Add(1)
	vm.metrics.drawNanos.Add(elapsed)
	for {
		cur := vm.metrics.drawMaxNanos.Load()
		if elapsed <= cur || vm.metrics.drawMaxNanos.CompareAndSwap(cur, elapsed) {
			break
		}
	}
}

// b2u converts a condition to the 0/1 a flag register expects.
func b2u(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

// Stats is a snapshot of VM instrumentation.
type Stats struct {
	// Instructions executed, and the rate actually achieved. If EffectiveHz is
	// well below the configured rate, something on the instruction path is
	// blocking - which also means the keypad is being sampled less often.
	Instructions uint64
	ConfiguredHz int
	EffectiveHz  float64

	// Draws is the DXYN count. DrawAvg and DrawMax measure time spent inside
	// the DisplayProvider and are derived from a sample of draws, not all of
	// them; DrawSamples says how many.
	Draws       uint64
	DrawSamples uint64
	DrawAvg     time.Duration
	DrawMax     time.Duration
	DrawsPerSec float64

	// OpcodeCounts is indexed by [OpcodeType].
	OpcodeCounts [NumOpcodeTypes]uint64

	Uptime time.Duration
}

// Stats returns a snapshot of VM instrumentation. It is safe to call from any
// goroutine.
func (vm *Chip8VM) Stats() Stats {
	s := Stats{
		Instructions: vm.metrics.instructions.Load(),
		ConfiguredHz: vm.cpuHz,
		Draws:        vm.metrics.draws.Load(),
		DrawSamples:  vm.metrics.drawSamples.Load(),
		DrawMax:      time.Duration(vm.metrics.drawMaxNanos.Load()),
	}
	for i := range s.OpcodeCounts {
		s.OpcodeCounts[i] = vm.metrics.opcodes[i].Load()
	}
	if s.DrawSamples > 0 {
		s.DrawAvg = time.Duration(vm.metrics.drawNanos.Load() / s.DrawSamples)
	}
	if started := vm.metrics.startedAt.Load(); started != 0 {
		s.Uptime = vm.now().Sub(time.Unix(0, started))
		if secs := s.Uptime.Seconds(); secs > 0 {
			s.EffectiveHz = float64(s.Instructions) / secs
			s.DrawsPerSec = float64(s.Draws) / secs
		}
	}
	return s
}
