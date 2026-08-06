package vm

import (
	"context"
	"os"
)

type Chip8VM struct {
	ram    [4096]uint8
	greg   [16]uint8
	idxreg uint16
	pc     uint16
	// buf[col - x][row - y]
	buf     [64][32]bool
	key     KeyboardProvider
	display DisplayProvider
}

func NewChip8VM(key KeyboardProvider, display DisplayProvider) *Chip8VM {
	return &Chip8VM{key: key, display: display, pc: 512}
}

func (vm *Chip8VM) LoadROMFromFile(filepath string) error {
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
	for {
		if int(vm.pc) >= len(vm.ram)-1 {
			return
		}
		parsedOpcode := parseOpcode([2]uint8{vm.ram[vm.pc], vm.ram[vm.pc+1]})
		if parsedOpcode.opcodeType == OpNA {
			return
		}
		vm.exec(parsedOpcode)
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
	default:
		panic("unimplemented: " + oc.opcodeType)
	}
	vm.pc += 2
}
