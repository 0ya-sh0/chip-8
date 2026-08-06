package vm

const (
	OpNA   = "NA"
	Op00E0 = "00E0"
	Op00EE = "00EE"
	Op0NNN = "0NNN"
	Op1NNN = "1NNN"
	Op2NNN = "2NNN"
	Op3XNN = "3XNN"
	Op4XNN = "4XNN"
	Op5XY0 = "5XY0"
	Op6XNN = "6XNN"
	Op7XNN = "7XNN"
	Op8XY0 = "8XY0"
	Op8XY1 = "8XY1"
	Op8XY2 = "8XY2"
	Op8XY3 = "8XY3"
	Op8XY4 = "8XY4"
	Op8XY5 = "8XY5"
	Op8XY6 = "8XY6"
	Op8XY7 = "8XY7"
	Op8XYE = "8XYE"
	Op9XY0 = "9XY0"
	OpANNN = "ANNN"
	OpBNNN = "BNNN"
	OpCXNN = "CXNN"
	OpDXYN = "DXYN"
	OpEX9E = "EX9E"
	OpEXA1 = "EXA1"
	OpFX07 = "FX07"
	OpFX0A = "FX0A"
	OpFX15 = "FX15"
	OpFX18 = "FX18"
	OpFX1E = "FX1E"
	OpFX29 = "FX29"
	OpFX33 = "FX33"
	OpFX55 = "FX55"
	OpFX65 = "FX65"
)

type parsedOpcode struct {
	opcodeType string
	nibbles    [4]uint8
}

func parseOpcode(opcode [2]uint8) parsedOpcode {
	nibbles := splitNibbles(opcode)
	opcodeType := OpNA

	switch nibbles[0] {
	case 0x00:
		if nibbles[1] == 0x00 && nibbles[2] == 0x0E && nibbles[3] == 0x00 {
			opcodeType = Op00E0
		} else if nibbles[1] == 0x00 && nibbles[2] == 0x0E && nibbles[3] == 0x0E {
			opcodeType = Op00EE
		} else {
			opcodeType = Op0NNN
		}
	case 0x01:
		opcodeType = Op1NNN
	case 0x02:
		opcodeType = Op2NNN
	case 0x03:
		opcodeType = Op3XNN
	case 0x04:
		opcodeType = Op4XNN
	case 0x05:
		if nibbles[3] == 0x00 {
			opcodeType = Op5XY0
		}
	case 0x06:
		opcodeType = Op6XNN
	case 0x07:
		opcodeType = Op7XNN
	case 0x08:
		switch nibbles[3] {
		case 0x00:
			opcodeType = Op8XY0
		case 0x01:
			opcodeType = Op8XY1
		case 0x02:
			opcodeType = Op8XY2
		case 0x03:
			opcodeType = Op8XY3
		case 0x04:
			opcodeType = Op8XY4
		case 0x05:
			opcodeType = Op8XY5
		case 0x06:
			opcodeType = Op8XY6
		case 0x07:
			opcodeType = Op8XY7
		case 0x0E:
			opcodeType = Op8XYE
		}
	case 0x09:
		if nibbles[3] == 0x00 {
			opcodeType = Op9XY0
		}
	case 0x0A:
		opcodeType = OpANNN
	case 0x0B:
		opcodeType = OpBNNN
	case 0x0C:
		opcodeType = OpCXNN
	case 0x0D:
		opcodeType = OpDXYN
	case 0x0E:
		if nibbles[2] == 0x09 && nibbles[3] == 0x0E {
			opcodeType = OpEX9E
		} else if nibbles[2] == 0x0A && nibbles[3] == 0x01 {
			opcodeType = OpEXA1
		}
	case 0x0F:
		switch nibbles[2] {
		case 0x00:
			if nibbles[3] == 0x07 {
				opcodeType = OpFX07
			} else if nibbles[3] == 0x0A {
				opcodeType = OpFX0A
			}
		case 0x01:
			switch nibbles[3] {
			case 0x05:
				opcodeType = OpFX15
			case 0x08:
				opcodeType = OpFX18
			case 0x0E:
				opcodeType = OpFX1E
			}
		case 0x02:
			if nibbles[3] == 0x09 {
				opcodeType = OpFX29
			}
		case 0x03:
			if nibbles[3] == 0x03 {
				opcodeType = OpFX33
			}
		case 0x05:
			if nibbles[3] == 0x05 {
				opcodeType = OpFX55
			}
		case 0x06:
			if nibbles[3] == 0x05 {
				opcodeType = OpFX65
			}
		}
	}

	return parsedOpcode{
		opcodeType: opcodeType,
		nibbles:    nibbles,
	}
}

func splitNibbles(opcode [2]uint8) [4]uint8 {
	var result [4]uint8
	result[0] = (opcode[0] & 0xF0) >> 4
	result[1] = opcode[0] & 0x0F
	result[2] = (opcode[1] & 0xF0) >> 4
	result[3] = opcode[1] & 0x0F
	return result
}
