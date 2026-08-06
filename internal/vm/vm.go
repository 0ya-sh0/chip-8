package vm

import (
	"context"
	"math/rand"
	"os"
	"time"
)

var fontSet = [80]uint8{
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

type Chip8VM struct {
	ram    [4096]uint8
	greg   [16]uint8
	idxreg uint16
	pc     uint16
	// buf[col - x][row - y]
	buf        [64][32]bool
	stack      []uint16
	soundTimer uint8
	delayTimer uint8
	key        KeyboardProvider
	display    DisplayProvider
}

func NewChip8VM(key KeyboardProvider, display DisplayProvider) *Chip8VM {
	return &Chip8VM{key: key, display: display, pc: 512}
}

func (vm *Chip8VM) LoadROMFromFile(filepath string) error {
	// Load font set into reserved memory 0x050 - 0x09F
	copy(vm.ram[0x050:], fontSet[:])
	data, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	for i, ptr := 0, 512; ptr < len(vm.ram) && i < len(data); ptr++ {
		vm.ram[ptr] = data[i]
		i++
	}
	return nil
}

func (vm *Chip8VM) Start(ctx context.Context) {
	cpuTick := time.NewTicker(time.Second / 500)
	delayTick := time.NewTicker(time.Second / 60)

	defer cpuTick.Stop()
	defer delayTick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cpuTick.C:
			if int(vm.pc) >= len(vm.ram)-1 {
				return
			}
			parsedOpcode := parseOpcode([2]uint8{vm.ram[vm.pc], vm.ram[vm.pc+1]})
			if parsedOpcode.opcodeType == OpNA {
				return
			}
			vm.exec(parsedOpcode)
		case <-delayTick.C:
			if vm.delayTimer > 0 {
				vm.delayTimer -= 1
			}
			if vm.soundTimer > 0 {
				vm.soundTimer -= 1
			}
		}
	}
}

func (vm *Chip8VM) exec(oc parsedOpcode) {
	// fmt.Printf("%X => %+v\n", [2]uint8{vm.ram[vm.pc], vm.ram[vm.pc+1]}, oc)
	switch oc.opcodeType {
	case Op00E0:
		for col := 0; col < 64; col++ {
			for row := 0; row < 32; row++ {
				vm.buf[col][row] = false
			}
		}
		vm.display.Clear()
	case Op6XNN:
		// Const: Sets VX to NN
		vm.greg[oc.nibbles[1]] = (oc.nibbles[2]<<4 + oc.nibbles[3])
	case OpANNN:
		// MEM: Sets I to the address NNN
		vm.idxreg = uint16(oc.nibbles[1])<<8 + uint16(oc.nibbles[2])<<4 + uint16(oc.nibbles[3])
	case OpDXYN:
		// Display: draw(Vx, Vy, N)
		xCoord := vm.greg[oc.nibbles[1]] % 64
		yCoord := vm.greg[oc.nibbles[2]] % 32
		N := oc.nibbles[3]
		flipBit := false
		for i := 0; i < int(N) && int(yCoord)+i < 32; i++ {
			rowByte := vm.ram[vm.idxreg+uint16(i)]
			var rowData [8]bool

			rowData[0] = (rowByte>>7)&0x01 == 1
			rowData[1] = (rowByte>>6)&0x01 == 1
			rowData[2] = (rowByte>>5)&0x01 == 1
			rowData[3] = (rowByte>>4)&0x01 == 1
			rowData[4] = (rowByte>>3)&0x01 == 1
			rowData[5] = (rowByte>>2)&0x01 == 1
			rowData[6] = (rowByte>>1)&0x01 == 1
			rowData[7] = (rowByte>>0)&0x01 == 1

			for col := xCoord; col < 64 && col < (xCoord+8); col++ {
				row := int(yCoord) + i
				if rowData[col-xCoord] {
					if vm.buf[col][row] {
						flipBit = true
					}
					vm.buf[col][row] = !vm.buf[col][row]
				}
			}
		}
		if flipBit {
			vm.greg[15] = 1
		} else {
			vm.greg[15] = 0
		}
		vm.display.Draw(vm.buf)
	case Op1NNN:
		// Flow: Jumps to address NNN
		vm.pc = uint16(oc.nibbles[1])<<8 + uint16(oc.nibbles[2])<<4 + uint16(oc.nibbles[3])
		return
	case Op7XNN:
		// Const: Adds NN to VX (carry flag is not changed).
		vm.greg[oc.nibbles[1]] += oc.nibbles[2]<<4 + oc.nibbles[3]
	case Op3XNN:
		// Cond: Skips the next instruction if VX equals NN (usually the next instruction is a jump to skip a code block)
		if vm.greg[oc.nibbles[1]] == oc.nibbles[2]<<4+oc.nibbles[3] {
			vm.pc += 4
			return
		}
	case Op4XNN:
		// Cond: Skips the next instruction if VX does not equal NN (usually the next instruction is a jump to skip a code block)
		if vm.greg[oc.nibbles[1]] != oc.nibbles[2]<<4+oc.nibbles[3] {
			vm.pc += 4
			return
		}
	case Op5XY0:
		// Cond: Skips the next instruction if VX equals VY (usually the next instruction is a jump to skip a code block)
		if vm.greg[oc.nibbles[1]] == vm.greg[oc.nibbles[2]] {
			vm.pc += 4
			return
		}
	case Op9XY0:
		// Cond: Skips the next instruction if VX does not equal VY. (Usually the next instruction is a jump to skip a code block)
		if vm.greg[oc.nibbles[1]] != vm.greg[oc.nibbles[2]] {
			vm.pc += 4
			return
		}
	case Op2NNN:
		// Flow: Calls subroutine at NNN
		vm.stack = append(vm.stack, vm.pc+2)
		vm.pc = uint16(oc.nibbles[1])<<8 + uint16(oc.nibbles[2])<<4 + uint16(oc.nibbles[3])
		return
	case Op00EE:
		// Flow: return
		vm.pc = vm.stack[len(vm.stack)-1]
		vm.stack = vm.stack[:len(vm.stack)-1]
		return
	case Op8XY0:
		// Sets VX to the value of VY
		x, y := oc.nibbles[1], oc.nibbles[2]
		vm.greg[x] = vm.greg[y]

	case Op8XY1:
		// Sets VX to VX OR VY
		x, y := oc.nibbles[1], oc.nibbles[2]
		vm.greg[x] |= vm.greg[y]

	case Op8XY2:
		// Sets VX to VX AND VY
		x, y := oc.nibbles[1], oc.nibbles[2]
		vm.greg[x] &= vm.greg[y]

	case Op8XY3:
		// Sets VX to VX XOR VY
		x, y := oc.nibbles[1], oc.nibbles[2]
		vm.greg[x] ^= vm.greg[y]

	case Op8XY4:
		// Adds VY to VX. VF is set to 1 when there's an overflow (> 255), otherwise 0.
		x, y := oc.nibbles[1], oc.nibbles[2]
		vx, vy := vm.greg[x], vm.greg[y]
		sum := uint16(vx) + uint16(vy)

		var flag uint8 = 0
		if sum > 255 {
			flag = 1
		}
		vm.greg[x] = uint8(sum)
		vm.greg[15] = flag

	case Op8XY5:
		// VX = VX - VY. VF is set to 1 if VX >= VY (NO borrow), otherwise 0.
		x, y := oc.nibbles[1], oc.nibbles[2]
		vx, vy := vm.greg[x], vm.greg[y]

		var flag uint8 = 0
		if vx >= vy {
			flag = 1
		}
		vm.greg[x] = vx - vy
		vm.greg[15] = flag

	case Op8XY6:
		// Shifts VX right by 1. VF is set to the LSB of VX prior to shift.
		x := oc.nibbles[1]
		vx := vm.greg[x]
		flag := vx & 0x01

		vm.greg[x] = vx >> 1
		vm.greg[15] = flag

	case Op8XY7:
		// VX = VY - VX. VF is set to 1 if VY >= VX (NO borrow), otherwise 0.
		x, y := oc.nibbles[1], oc.nibbles[2]
		vx, vy := vm.greg[x], vm.greg[y]

		var flag uint8 = 0
		if vy >= vx {
			flag = 1
		}
		vm.greg[x] = vy - vx
		vm.greg[15] = flag

	case Op8XYE:
		// Shifts VX left by 1. VF is set to the MSB of VX prior to shift.
		x := oc.nibbles[1]
		vx := vm.greg[x]
		flag := (vx >> 7) & 0x01

		vm.greg[x] = vx << 1
		vm.greg[15] = flag
	case OpFX65:
		// Fills from V0 to VX (including VX) with values from memory, starting at address I. The offset from I is increased by 1 for each value read, but I itself is left unmodified.
		x := oc.nibbles[1]
		for ix := 0; ix <= int(x); ix++ {
			vm.greg[ix] = vm.ram[vm.idxreg+uint16(ix)]
		}
	case OpFX55:
		// Stores from V0 to VX (including VX) in memory, starting at address I. The offset from I is increased by 1 for each value written, but I itself is left unmodified.
		x := oc.nibbles[1]
		for ix := 0; ix <= int(x); ix++ {
			vm.ram[vm.idxreg+uint16(ix)] = vm.greg[ix]
		}
	case OpFX33:
		// BCD: Stores Binary-Coded Decimal representation of VX in memory at I, I+1, I+2.
		x := oc.nibbles[1]
		vx := vm.greg[x]

		vm.ram[vm.idxreg] = vx / 100
		vm.ram[vm.idxreg+1] = (vx / 10) % 10
		vm.ram[vm.idxreg+2] = vx % 10
	case OpFX1E:
		// Sets I = I + VX
		x := oc.nibbles[1]
		vm.idxreg += uint16(vm.greg[x])
	case OpCXNN:
		x := oc.nibbles[1]
		nn := (oc.nibbles[2] << 4) | oc.nibbles[3]
		vm.greg[x] = uint8(rand.Intn(256)) & nn
	case OpEX9E:
		x := oc.nibbles[1]
		if vm.key.IsKeyPressed(vm.greg[x]) {
			vm.pc += 4
			return
		}
	case OpEXA1:
		x := oc.nibbles[1]
		if !vm.key.IsKeyPressed(vm.greg[x]) {
			vm.pc += 4
			return
		}
	case OpFX0A:
		x := oc.nibbles[1]
		key, pressed := vm.key.GetPressedKey()
		if !pressed {
			// No key pressed: return early WITHOUT advancing vm.pc.
			// Timers in Start() continue ticking while CPU repeatedly checks this opcode.
			return
		}
		vm.greg[x] = key
	case OpFX07:
		vm.greg[oc.nibbles[1]] = vm.delayTimer
	case OpFX15:
		vm.delayTimer = vm.greg[oc.nibbles[1]]
	case OpFX18:
		vm.soundTimer = vm.greg[oc.nibbles[1]]
	case OpFX29:
		// Fonts start at address 0x050, and each character is 5 bytes tall
		vm.idxreg = 0x050 + uint16(vm.greg[oc.nibbles[1]]&0x0F)*5
	default:
		panic("unimplemented: " + oc.opcodeType)
	}
	vm.pc += 2
}
