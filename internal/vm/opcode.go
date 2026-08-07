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

type ParsedOpcode struct {
	OpcodeType string
	Nibbles    [4]uint8
}

func ParseOpcode(opcode [2]uint8) ParsedOpcode {
	Nibbles := splitNibbles(opcode)
	OpcodeType := OpNA

	switch Nibbles[0] {
	case 0x00:
		if Nibbles[1] == 0x00 && Nibbles[2] == 0x0E && Nibbles[3] == 0x00 {
			OpcodeType = Op00E0
		} else if Nibbles[1] == 0x00 && Nibbles[2] == 0x0E && Nibbles[3] == 0x0E {
			OpcodeType = Op00EE
		} else {
			OpcodeType = Op0NNN
		}
	case 0x01:
		OpcodeType = Op1NNN
	case 0x02:
		OpcodeType = Op2NNN
	case 0x03:
		OpcodeType = Op3XNN
	case 0x04:
		OpcodeType = Op4XNN
	case 0x05:
		if Nibbles[3] == 0x00 {
			OpcodeType = Op5XY0
		}
	case 0x06:
		OpcodeType = Op6XNN
	case 0x07:
		OpcodeType = Op7XNN
	case 0x08:
		switch Nibbles[3] {
		case 0x00:
			OpcodeType = Op8XY0
		case 0x01:
			OpcodeType = Op8XY1
		case 0x02:
			OpcodeType = Op8XY2
		case 0x03:
			OpcodeType = Op8XY3
		case 0x04:
			OpcodeType = Op8XY4
		case 0x05:
			OpcodeType = Op8XY5
		case 0x06:
			OpcodeType = Op8XY6
		case 0x07:
			OpcodeType = Op8XY7
		case 0x0E:
			OpcodeType = Op8XYE
		}
	case 0x09:
		if Nibbles[3] == 0x00 {
			OpcodeType = Op9XY0
		}
	case 0x0A:
		OpcodeType = OpANNN
	case 0x0B:
		OpcodeType = OpBNNN
	case 0x0C:
		OpcodeType = OpCXNN
	case 0x0D:
		OpcodeType = OpDXYN
	case 0x0E:
		if Nibbles[2] == 0x09 && Nibbles[3] == 0x0E {
			OpcodeType = OpEX9E
		} else if Nibbles[2] == 0x0A && Nibbles[3] == 0x01 {
			OpcodeType = OpEXA1
		}
	case 0x0F:
		switch Nibbles[2] {
		case 0x00:
			if Nibbles[3] == 0x07 {
				OpcodeType = OpFX07
			} else if Nibbles[3] == 0x0A {
				OpcodeType = OpFX0A
			}
		case 0x01:
			switch Nibbles[3] {
			case 0x05:
				OpcodeType = OpFX15
			case 0x08:
				OpcodeType = OpFX18
			case 0x0E:
				OpcodeType = OpFX1E
			}
		case 0x02:
			if Nibbles[3] == 0x09 {
				OpcodeType = OpFX29
			}
		case 0x03:
			if Nibbles[3] == 0x03 {
				OpcodeType = OpFX33
			}
		case 0x05:
			if Nibbles[3] == 0x05 {
				OpcodeType = OpFX55
			}
		case 0x06:
			if Nibbles[3] == 0x05 {
				OpcodeType = OpFX65
			}
		}
	}

	return ParsedOpcode{
		OpcodeType: OpcodeType,
		Nibbles:    Nibbles,
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
