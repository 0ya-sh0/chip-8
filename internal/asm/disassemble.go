package asm

import (
	"fmt"
	"strings"

	"github.com/0ya-sh0/chip-8/internal/vm"
)

// Disassemble converts a binary CHIP-8 ROM into a formatted assembly string.
func Disassemble(rom []byte) (string, error) {
	if len(rom) == 0 {
		return "", fmt.Errorf("rom is empty")
	}

	var builder strings.Builder
	address := 0x0200 // CHIP-8 ROMs start at address 0x0200 (512)

	for i := 0; i < len(rom); i += 2 {
		// Handle dangling byte if ROM length is odd
		if i+1 >= len(rom) {
			builder.WriteString(fmt.Sprintf("0x%04X:  %02X    DATA 0x%02X\n", address+i, rom[i], rom[i]))
			break
		}

		rawOpcode := [2]uint8{rom[i], rom[i+1]}
		parsed := vm.ParseOpcode(rawOpcode)
		mnemonic := FormatOpcode(parsed, rawOpcode)

		builder.WriteString(fmt.Sprintf("0x%04X:  %02X%02X  %s\n", address+i, rawOpcode[0], rawOpcode[1], mnemonic))
	}

	return builder.String(), nil
}

// FormatOpcode converts a vm.ParsedOpcode into its exact assembly syntax string.
func FormatOpcode(op vm.ParsedOpcode, raw [2]uint8) string {
	x, y := op.X(), op.Y()
	nn, nnn, nibble := op.NN(), op.NNN(), op.N()

	switch op.OpcodeType {
	case vm.Op00E0:
		return "CLS"
	case vm.Op00EE:
		return "RET"
	case vm.Op0NNN:
		return fmt.Sprintf("SYS 0x%03X", nnn)
	case vm.Op1NNN:
		return fmt.Sprintf("JP 0x%03X", nnn)
	case vm.Op2NNN:
		return fmt.Sprintf("CALL 0x%03X", nnn)
	case vm.Op3XNN:
		return fmt.Sprintf("SE V%X, 0x%02X", x, nn)
	case vm.Op4XNN:
		return fmt.Sprintf("SNE V%X, 0x%02X", x, nn)
	case vm.Op5XY0:
		return fmt.Sprintf("SE V%X, V%X", x, y)
	case vm.Op6XNN:
		return fmt.Sprintf("LD V%X, 0x%02X", x, nn)
	case vm.Op7XNN:
		return fmt.Sprintf("ADD V%X, 0x%02X", x, nn)
	case vm.Op8XY0:
		return fmt.Sprintf("LD V%X, V%X", x, y)
	case vm.Op8XY1:
		return fmt.Sprintf("OR V%X, V%X", x, y)
	case vm.Op8XY2:
		return fmt.Sprintf("AND V%X, V%X", x, y)
	case vm.Op8XY3:
		return fmt.Sprintf("XOR V%X, V%X", x, y)
	case vm.Op8XY4:
		return fmt.Sprintf("ADD V%X, V%X", x, y)
	case vm.Op8XY5:
		return fmt.Sprintf("SUB V%X, V%X", x, y)
	case vm.Op8XY6:
		if y == 0 {
			return fmt.Sprintf("SHR V%X", x)
		}
		return fmt.Sprintf("SHR V%X, V%X", x, y)
	case vm.Op8XY7:
		return fmt.Sprintf("SUBN V%X, V%X", x, y)
	case vm.Op8XYE:
		if y == 0 {
			return fmt.Sprintf("SHL V%X", x)
		}
		return fmt.Sprintf("SHL V%X, V%X", x, y)
	case vm.Op9XY0:
		return fmt.Sprintf("SNE V%X, V%X", x, y)
	case vm.OpANNN:
		return fmt.Sprintf("LD I, 0x%03X", nnn)
	case vm.OpBNNN:
		return fmt.Sprintf("JP V0, 0x%03X", nnn)
	case vm.OpCXNN:
		return fmt.Sprintf("RND V%X, 0x%02X", x, nn)
	case vm.OpDXYN:
		return fmt.Sprintf("DRW V%X, V%X, %d", x, y, nibble)
	case vm.OpEX9E:
		return fmt.Sprintf("SKP V%X", x)
	case vm.OpEXA1:
		return fmt.Sprintf("SKNP V%X", x)
	case vm.OpFX07:
		return fmt.Sprintf("LD V%X, DT", x)
	case vm.OpFX0A:
		return fmt.Sprintf("LD V%X, K", x)
	case vm.OpFX15:
		return fmt.Sprintf("LD DT, V%X", x)
	case vm.OpFX18:
		return fmt.Sprintf("LD ST, V%X", x)
	case vm.OpFX1E:
		return fmt.Sprintf("ADD I, V%X", x)
	case vm.OpFX29:
		return fmt.Sprintf("LD F, V%X", x)
	case vm.OpFX33:
		return fmt.Sprintf("LD B, V%X", x)
	case vm.OpFX55:
		return fmt.Sprintf("LD [I], V%X", x)
	case vm.OpFX65:
		return fmt.Sprintf("LD V%X, [I]", x)
	default:
		return fmt.Sprintf("DATA 0x%02X%02X", raw[0], raw[1])
	}
}
