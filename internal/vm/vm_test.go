package vm

import (
	"context"
	"errors"
	"math/rand/v2"
	"sync"
	"testing"
	"time"
)

// fakeClock is a manually advanced time source. Timer behaviour is a function
// of time, so injecting the clock is what turns "wait 200ms and hope" into a
// deterministic assertion.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// stubProviders records provider traffic so tests can assert on edges.
type stubProviders struct {
	pressed map[uint8]bool
	anyKey  func() (uint8, bool)

	draws, clears, plays, stops int
	lastFrame                   FrameBuffer
}

func newStubProviders() *stubProviders {
	return &stubProviders{pressed: map[uint8]bool{}}
}

func (s *stubProviders) IsKeyPressed(key uint8) bool { return s.pressed[key] }

func (s *stubProviders) GetPressedKey() (uint8, bool) {
	if s.anyKey != nil {
		return s.anyKey()
	}
	for k, down := range s.pressed {
		if down {
			return k, true
		}
	}
	return 0, false
}

func (s *stubProviders) Clear()              { s.clears++ }
func (s *stubProviders) Draw(fb FrameBuffer) { s.draws++; s.lastFrame = fb }
func (s *stubProviders) PlaySound()          { s.plays++ }
func (s *stubProviders) StopSound()          { s.stops++ }

func newTestVM(t *testing.T) (*Chip8VM, *stubProviders, *fakeClock) {
	t.Helper()
	stub := newStubProviders()
	clock := newFakeClock()
	v := NewChip8VM(stub, stub, stub,
		WithClock(clock.Now),
		WithRandom(rand.New(rand.NewPCG(1, 2))),
	)
	return v, stub, clock
}

// step assembles one instruction at the current PC and executes it.
func step(t *testing.T, v *Chip8VM, hi, lo uint8) error {
	t.Helper()
	v.ram[v.pc] = hi
	v.ram[v.pc+1] = lo
	op := ParseOpcode([2]uint8{hi, lo})
	if op.OpcodeType == OpNA {
		t.Fatalf("test wrote an undecodable word %02X%02X", hi, lo)
	}
	return v.exec(op)
}

func TestArithmeticAndFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(v *Chip8VM)
		hi, lo  uint8
		wantReg map[uint8]uint8
	}{
		{
			name:    "8XY4 sets carry on overflow",
			setup:   func(v *Chip8VM) { v.greg[1], v.greg[2] = 200, 100 },
			hi:      0x81,
			lo:      0x24,
			wantReg: map[uint8]uint8{1: 44, VF: 1},
		},
		{
			name:    "8XY4 clears carry without overflow",
			setup:   func(v *Chip8VM) { v.greg[1], v.greg[2] = 1, 2 },
			hi:      0x81,
			lo:      0x24,
			wantReg: map[uint8]uint8{1: 3, VF: 0},
		},
		{
			// VF is both the destination and the flag here. The flag must win,
			// which is why the result is stored before the flag.
			name:    "8XY4 into VF keeps the flag not the sum",
			setup:   func(v *Chip8VM) { v.greg[VF], v.greg[2] = 200, 100 },
			hi:      0x8F,
			lo:      0x24,
			wantReg: map[uint8]uint8{VF: 1},
		},
		{
			name:    "8XY5 sets no-borrow flag",
			setup:   func(v *Chip8VM) { v.greg[1], v.greg[2] = 5, 3 },
			hi:      0x81,
			lo:      0x25,
			wantReg: map[uint8]uint8{1: 2, VF: 1},
		},
		{
			name:    "8XY5 borrows",
			setup:   func(v *Chip8VM) { v.greg[1], v.greg[2] = 3, 5 },
			hi:      0x81,
			lo:      0x25,
			wantReg: map[uint8]uint8{1: 254, VF: 0},
		},
		{
			name:    "8XY7 reverse subtract",
			setup:   func(v *Chip8VM) { v.greg[1], v.greg[2] = 3, 5 },
			hi:      0x81,
			lo:      0x27,
			wantReg: map[uint8]uint8{1: 2, VF: 1},
		},
		{
			name:    "8XY6 shifts right and captures LSB",
			setup:   func(v *Chip8VM) { v.greg[1] = 0b0000_0101 },
			hi:      0x81,
			lo:      0x26,
			wantReg: map[uint8]uint8{1: 0b0000_0010, VF: 1},
		},
		{
			name:    "8XYE shifts left and captures MSB",
			setup:   func(v *Chip8VM) { v.greg[1] = 0b1000_0001 },
			hi:      0x81,
			lo:      0x2E,
			wantReg: map[uint8]uint8{1: 0b0000_0010, VF: 1},
		},
		{
			name:    "7XNN wraps without touching carry",
			setup:   func(v *Chip8VM) { v.greg[1], v.greg[VF] = 0xFF, 0 },
			hi:      0x71,
			lo:      0x02,
			wantReg: map[uint8]uint8{1: 1, VF: 0},
		},
		{
			name:    "FX33 stores BCD digits",
			setup:   func(v *Chip8VM) { v.greg[1], v.idxreg = 195, 0x300 },
			hi:      0xF1,
			lo:      0x33,
			wantReg: map[uint8]uint8{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v, _, _ := newTestVM(t)
			tt.setup(v)
			if err := step(t, v, tt.hi, tt.lo); err != nil {
				t.Fatalf("exec: %v", err)
			}
			for reg, want := range tt.wantReg {
				if got := v.greg[reg]; got != want {
					t.Errorf("V%X = %d, want %d", reg, got, want)
				}
			}
			if v.pc != ProgramStart+2 {
				t.Errorf("pc = %#x, want %#x", v.pc, ProgramStart+2)
			}
		})
	}
}

func TestBCD(t *testing.T) {
	t.Parallel()

	v, _, _ := newTestVM(t)
	v.greg[1], v.idxreg = 195, 0x300
	if err := step(t, v, 0xF1, 0x33); err != nil {
		t.Fatalf("exec: %v", err)
	}
	for i, want := range []uint8{1, 9, 5} {
		if got := v.ram[0x300+i]; got != want {
			t.Errorf("ram[0x%03X] = %d, want %d", 0x300+i, got, want)
		}
	}
}

func TestSkipInstructions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(v *Chip8VM, s *stubProviders)
		hi, lo   uint8
		wantSkip bool
	}{
		{"3XNN equal skips", func(v *Chip8VM, _ *stubProviders) { v.greg[1] = 0x42 }, 0x31, 0x42, true},
		{"3XNN unequal falls through", func(v *Chip8VM, _ *stubProviders) { v.greg[1] = 0x00 }, 0x31, 0x42, false},
		{"4XNN unequal skips", func(v *Chip8VM, _ *stubProviders) { v.greg[1] = 0x00 }, 0x41, 0x42, true},
		{"5XY0 equal skips", func(v *Chip8VM, _ *stubProviders) { v.greg[1], v.greg[2] = 7, 7 }, 0x51, 0x20, true},
		{"9XY0 unequal skips", func(v *Chip8VM, _ *stubProviders) { v.greg[1], v.greg[2] = 7, 8 }, 0x91, 0x20, true},
		{
			"EX9E skips when key down",
			func(v *Chip8VM, s *stubProviders) { v.greg[1] = 0xA; s.pressed[0xA] = true },
			0xE1, 0x9E, true,
		},
		{
			"EXA1 skips when key up",
			func(v *Chip8VM, s *stubProviders) { v.greg[1] = 0xA },
			0xE1, 0xA1, true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v, stub, _ := newTestVM(t)
			tt.setup(v, stub)
			if err := step(t, v, tt.hi, tt.lo); err != nil {
				t.Fatalf("exec: %v", err)
			}
			want := uint16(ProgramStart + 2)
			if tt.wantSkip {
				want = ProgramStart + 4
			}
			if v.pc != want {
				t.Errorf("pc = %#x, want %#x", v.pc, want)
			}
		})
	}
}

func TestCallAndReturn(t *testing.T) {
	t.Parallel()

	v, _, _ := newTestVM(t)
	if err := step(t, v, 0x24, 0x00); err != nil { // CALL 0x400
		t.Fatalf("call: %v", err)
	}
	if v.pc != 0x400 {
		t.Fatalf("pc after call = %#x, want 0x400", v.pc)
	}
	if v.sp != 1 || v.stack[0] != ProgramStart+2 {
		t.Fatalf("stack = %v sp = %d, want return address %#x", v.stack[:1], v.sp, ProgramStart+2)
	}

	if err := step(t, v, 0x00, 0xEE); err != nil { // RET
		t.Fatalf("ret: %v", err)
	}
	if v.pc != ProgramStart+2 {
		t.Errorf("pc after ret = %#x, want %#x", v.pc, ProgramStart+2)
	}
	if v.sp != 0 {
		t.Errorf("sp after ret = %d, want 0", v.sp)
	}
}

// A malformed ROM must produce an error, not a panic that takes down a server
// hosting many sessions.
func TestStackFaultsAreErrorsNotPanics(t *testing.T) {
	t.Parallel()

	t.Run("underflow", func(t *testing.T) {
		t.Parallel()
		v, _, _ := newTestVM(t)
		err := step(t, v, 0x00, 0xEE)
		if !errors.Is(err, ErrStackUnderflow) {
			t.Fatalf("err = %v, want ErrStackUnderflow", err)
		}
	})

	t.Run("overflow", func(t *testing.T) {
		t.Parallel()
		v, _, _ := newTestVM(t)
		for i := 0; i < StackDepth; i++ {
			v.pc = ProgramStart
			if err := step(t, v, 0x24, 0x00); err != nil {
				t.Fatalf("call %d: %v", i, err)
			}
		}
		v.pc = ProgramStart
		if err := step(t, v, 0x24, 0x00); !errors.Is(err, ErrStackOverflow) {
			t.Fatalf("err = %v, want ErrStackOverflow", err)
		}
	})
}

// I is 16 bits but memory is 12, so a ROM can trivially index past the end.
// Wrapping keeps that a ROM bug rather than a process-killing panic.
func TestOutOfRangeIndexRegisterWraps(t *testing.T) {
	t.Parallel()

	v, _, _ := newTestVM(t)
	v.idxreg = 0xFFF
	v.greg[0] = 0xAB

	if err := step(t, v, 0xF0, 0x55); err != nil { // LD [I], V0
		t.Fatalf("FX55: %v", err)
	}
	if got := v.ram[0xFFF]; got != 0xAB {
		t.Errorf("ram[0xFFF] = %#x, want 0xAB", got)
	}

	v.pc = ProgramStart
	v.idxreg = 0xFFE
	if err := step(t, v, 0xF3, 0x55); err != nil { // writes 0xFFE..0x1001
		t.Fatalf("FX55 across the end: %v", err)
	}
}

func TestDrawCollisionAndClipping(t *testing.T) {
	t.Parallel()

	t.Run("collision sets VF and erases", func(t *testing.T) {
		t.Parallel()
		v, stub, _ := newTestVM(t)
		v.ram[0x300] = 0b1000_0000
		v.idxreg = 0x300
		v.greg[0], v.greg[1] = 0, 0

		if err := step(t, v, 0xD0, 0x11); err != nil { // DRW V0, V1, 1
			t.Fatalf("draw: %v", err)
		}
		if !v.buf[0][0] {
			t.Fatal("pixel not lit after first draw")
		}
		if v.greg[VF] != 0 {
			t.Errorf("VF = %d after first draw, want 0", v.greg[VF])
		}

		v.pc = ProgramStart
		if err := step(t, v, 0xD0, 0x11); err != nil {
			t.Fatalf("redraw: %v", err)
		}
		if v.buf[0][0] {
			t.Error("pixel still lit after XOR redraw")
		}
		if v.greg[VF] != 1 {
			t.Errorf("VF = %d after collision, want 1", v.greg[VF])
		}
		if stub.draws != 2 {
			t.Errorf("Draw called %d times, want 2", stub.draws)
		}
	})

	t.Run("origin wraps but sprite clips", func(t *testing.T) {
		t.Parallel()
		v, _, _ := newTestVM(t)
		v.ram[0x300] = 0xFF
		v.idxreg = 0x300
		// 64 wraps to column 0; 32 wraps to row 0.
		v.greg[0], v.greg[1] = ScreenWidth, ScreenHeight

		if err := step(t, v, 0xD0, 0x11); err != nil {
			t.Fatalf("draw: %v", err)
		}
		for x := 0; x < 8; x++ {
			if !v.buf[0][x] {
				t.Fatalf("pixel [0][%d] not lit; origin should wrap to 0,0", x)
			}
		}

		// A sprite at the right edge must clip, not wrap onto the next row.
		v.pc = ProgramStart
		v.buf = FrameBuffer{}
		v.greg[0], v.greg[1] = ScreenWidth-4, 0
		if err := step(t, v, 0xD0, 0x11); err != nil {
			t.Fatalf("edge draw: %v", err)
		}
		for x := 0; x < ScreenWidth-4; x++ {
			if v.buf[0][x] {
				t.Fatalf("pixel [0][%d] lit; sprite wrapped instead of clipping", x)
			}
		}
	})
}

func TestDelayTimerReadsBackAsCountdown(t *testing.T) {
	t.Parallel()

	v, _, clock := newTestVM(t)
	v.greg[1] = 60 // one second at 60Hz

	if err := step(t, v, 0xF1, 0x15); err != nil { // LD DT, V1
		t.Fatalf("FX15: %v", err)
	}

	for _, tc := range []struct {
		advance time.Duration
		want    uint8
	}{
		{0, 60},
		{500 * time.Millisecond, 30},
		{490 * time.Millisecond, 1},
		{20 * time.Millisecond, 0},
		{time.Second, 0},
	} {
		clock.Advance(tc.advance)
		v.pc = ProgramStart
		if err := step(t, v, 0xF2, 0x07); err != nil { // LD V2, DT
			t.Fatalf("FX07: %v", err)
		}
		if got := v.greg[2]; got != tc.want {
			t.Errorf("after %s total, DT = %d, want %d", tc.advance, got, tc.want)
		}
	}
}

func TestSoundEmitsExactlyOneEdgePerBeep(t *testing.T) {
	t.Parallel()

	t.Run("zero never starts the buzzer", func(t *testing.T) {
		t.Parallel()
		v, stub, _ := newTestVM(t)
		v.greg[1] = 0
		if err := step(t, v, 0xF1, 0x18); err != nil {
			t.Fatalf("FX18: %v", err)
		}
		if stub.plays != 0 {
			t.Errorf("PlaySound called %d times for ST=0, want 0", stub.plays)
		}
	})

	t.Run("re-arming does not replay", func(t *testing.T) {
		t.Parallel()
		v, stub, clock := newTestVM(t)
		v.greg[1] = 60
		if err := step(t, v, 0xF1, 0x18); err != nil {
			t.Fatalf("FX18: %v", err)
		}
		clock.Advance(100 * time.Millisecond)
		v.pc = ProgramStart
		if err := step(t, v, 0xF1, 0x18); err != nil {
			t.Fatalf("FX18 again: %v", err)
		}
		if stub.plays != 1 {
			t.Errorf("PlaySound called %d times, want 1", stub.plays)
		}
	})

	t.Run("zero silences a running beep", func(t *testing.T) {
		t.Parallel()
		v, stub, clock := newTestVM(t)
		v.greg[1] = 60
		if err := step(t, v, 0xF1, 0x18); err != nil {
			t.Fatalf("FX18: %v", err)
		}
		clock.Advance(100 * time.Millisecond)

		v.greg[1] = 0
		v.pc = ProgramStart
		if err := step(t, v, 0xF1, 0x18); err != nil {
			t.Fatalf("FX18 zero: %v", err)
		}
		v.updateTimers()
		if stub.stops != 1 {
			t.Errorf("StopSound called %d times, want 1", stub.stops)
		}
	})

	t.Run("expiry stops once", func(t *testing.T) {
		t.Parallel()
		v, stub, clock := newTestVM(t)
		v.greg[1] = 6 // 100ms
		if err := step(t, v, 0xF1, 0x18); err != nil {
			t.Fatalf("FX18: %v", err)
		}
		clock.Advance(50 * time.Millisecond)
		v.updateTimers()
		if stub.stops != 0 {
			t.Fatalf("stopped early after 50ms")
		}
		clock.Advance(60 * time.Millisecond)
		for i := 0; i < 5; i++ {
			v.updateTimers()
		}
		if stub.stops != 1 {
			t.Errorf("StopSound called %d times, want exactly 1", stub.stops)
		}
	})
}

// A VM that faults or is cancelled mid-beep must not leave the buzzer running:
// StopSound lives on the instruction path, which stops running at shutdown.
func TestShutdownStopsSound(t *testing.T) {
	t.Parallel()

	stub := newStubProviders()
	v := NewChip8VM(stub, stub, stub, WithCPURate(5000))
	// FX18 V0 with V0=255, then an endless jump back to itself.
	if err := v.LoadROM([]byte{0x60, 0xFF, 0xF0, 0x18, 0x12, 0x02}); err != nil {
		t.Fatalf("load: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := v.Start(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Start returned %v, want DeadlineExceeded", err)
	}
	if stub.plays == 0 {
		t.Fatal("buzzer never started; test ROM did not run")
	}
	if stub.stops != 1 {
		t.Errorf("StopSound called %d times after shutdown, want 1", stub.stops)
	}
}

func TestStartReportsWhyItStopped(t *testing.T) {
	t.Parallel()

	stub := newStubProviders()
	v := NewChip8VM(stub, stub, stub, WithCPURate(5000))
	// 0x5123 is not a valid instruction (5XY0 requires a zero low nibble).
	if err := v.LoadROM([]byte{0x51, 0x23}); err != nil {
		t.Fatalf("load: %v", err)
	}

	err := v.Start(context.Background())
	if !errors.Is(err, ErrUnknownOpcode) {
		t.Fatalf("Start returned %v, want ErrUnknownOpcode", err)
	}
}

func TestLoadROMRejectsOversizedImages(t *testing.T) {
	t.Parallel()

	stub := newStubProviders()
	v := NewChip8VM(stub, stub, stub)
	if err := v.LoadROM(make([]byte, maxROMSize+1)); !errors.Is(err, ErrROMTooLarge) {
		t.Fatalf("err = %v, want ErrROMTooLarge", err)
	}
	if err := v.LoadROM(make([]byte, maxROMSize)); err != nil {
		t.Fatalf("exactly-full ROM rejected: %v", err)
	}
}

func TestFontIsLoadedAndAddressable(t *testing.T) {
	t.Parallel()

	v, _, _ := newTestVM(t)
	v.greg[1] = 0xA
	if err := step(t, v, 0xF1, 0x29); err != nil {
		t.Fatalf("FX29: %v", err)
	}
	want := uint16(FontStart + 0xA*FontHeight)
	if v.idxreg != want {
		t.Fatalf("I = %#x, want %#x", v.idxreg, want)
	}
	// The 'A' glyph starts with 0xF0.
	if got := v.ram[want]; got != 0xF0 {
		t.Errorf("font byte = %#x, want 0xF0", got)
	}
}

func TestRegisterStoreLoadRoundTrip(t *testing.T) {
	t.Parallel()

	v, _, _ := newTestVM(t)
	v.idxreg = 0x400
	for i := uint8(0); i <= 5; i++ {
		v.greg[i] = i * 3
	}
	if err := step(t, v, 0xF5, 0x55); err != nil { // LD [I], V5
		t.Fatalf("FX55: %v", err)
	}
	if v.idxreg != 0x400 {
		t.Errorf("I = %#x after FX55, want unmodified 0x400", v.idxreg)
	}

	for i := range v.greg {
		v.greg[i] = 0
	}
	v.pc = ProgramStart
	if err := step(t, v, 0xF5, 0x65); err != nil { // LD V5, [I]
		t.Fatalf("FX65: %v", err)
	}
	for i := uint8(0); i <= 5; i++ {
		if got := v.greg[i]; got != i*3 {
			t.Errorf("V%X = %d, want %d", i, got, i*3)
		}
	}
	if v.greg[6] != 0 {
		t.Errorf("V6 = %d, want 0; FX65 loaded past VX", v.greg[6])
	}
}

// FX0A must not advance the program counter while it waits, or the wait
// silently becomes a one-shot read.
func TestWaitForKeyBlocksWithoutAdvancing(t *testing.T) {
	t.Parallel()

	v, stub, _ := newTestVM(t)
	if err := step(t, v, 0xF1, 0x0A); err != nil {
		t.Fatalf("FX0A: %v", err)
	}
	if v.pc != ProgramStart {
		t.Fatalf("pc = %#x while waiting, want %#x", v.pc, ProgramStart)
	}

	stub.pressed[0x7] = true
	if err := step(t, v, 0xF1, 0x0A); err != nil {
		t.Fatalf("FX0A: %v", err)
	}
	if v.pc != ProgramStart+2 {
		t.Errorf("pc = %#x after key, want %#x", v.pc, ProgramStart+2)
	}
	if v.greg[1] != 0x7 {
		t.Errorf("V1 = %#x, want 0x7", v.greg[1])
	}
}

func TestStatsCountOpcodes(t *testing.T) {
	t.Parallel()

	v, _, _ := newTestVM(t)
	for i := 0; i < 3; i++ {
		v.pc = ProgramStart
		if err := step(t, v, 0x60, 0x01); err != nil {
			t.Fatalf("exec: %v", err)
		}
	}
	stats := v.Stats()
	if got := stats.OpcodeCounts[Op6XNN]; got != 3 {
		t.Errorf("Op6XNN count = %d, want 3", got)
	}
	if stats.Instructions != 3 {
		t.Errorf("Instructions = %d, want 3", stats.Instructions)
	}
}
