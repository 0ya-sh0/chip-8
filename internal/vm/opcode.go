package vm

// OpcodeType identifies a decoded CHIP-8 instruction.
//
// It is a defined integer type rather than a string so that the compiler
// rejects invalid values, and so that per-opcode bookkeeping can use a plain
// array index instead of a map lookup on the hot path.
type OpcodeType uint8

const (
	OpNA OpcodeType = iota // not a valid instruction
	Op0NNN
	Op00E0
	Op00EE
	Op1NNN
	Op2NNN
	Op3XNN
	Op4XNN
	Op5XY0
	Op6XNN
	Op7XNN
	Op8XY0
	Op8XY1
	Op8XY2
	Op8XY3
	Op8XY4
	Op8XY5
	Op8XY6
	Op8XY7
	Op8XYE
	Op9XY0
	OpANNN
	OpBNNN
	OpCXNN
	OpDXYN
	OpEX9E
	OpEXA1
	OpFX07
	OpFX0A
	OpFX15
	OpFX18
	OpFX1E
	OpFX29
	OpFX33
	OpFX55
	OpFX65

	// NumOpcodeTypes is the exclusive upper bound of OpcodeType. It is the
	// length of any array keyed by opcode.
	NumOpcodeTypes
)

var opcodeNames = [NumOpcodeTypes]string{
	OpNA:   "NA",
	Op0NNN: "0NNN",
	Op00E0: "00E0",
	Op00EE: "00EE",
	Op1NNN: "1NNN",
	Op2NNN: "2NNN",
	Op3XNN: "3XNN",
	Op4XNN: "4XNN",
	Op5XY0: "5XY0",
	Op6XNN: "6XNN",
	Op7XNN: "7XNN",
	Op8XY0: "8XY0",
	Op8XY1: "8XY1",
	Op8XY2: "8XY2",
	Op8XY3: "8XY3",
	Op8XY4: "8XY4",
	Op8XY5: "8XY5",
	Op8XY6: "8XY6",
	Op8XY7: "8XY7",
	Op8XYE: "8XYE",
	Op9XY0: "9XY0",
	OpANNN: "ANNN",
	OpBNNN: "BNNN",
	OpCXNN: "CXNN",
	OpDXYN: "DXYN",
	OpEX9E: "EX9E",
	OpEXA1: "EXA1",
	OpFX07: "FX07",
	OpFX0A: "FX0A",
	OpFX15: "FX15",
	OpFX18: "FX18",
	OpFX1E: "FX1E",
	OpFX29: "FX29",
	OpFX33: "FX33",
	OpFX55: "FX55",
	OpFX65: "FX65",
}

// String implements [fmt.Stringer].
func (t OpcodeType) String() string {
	if t >= NumOpcodeTypes {
		return "INVALID"
	}
	return opcodeNames[t]
}

// ParsedOpcode is a decoded 16-bit instruction. Nibbles holds the four 4-bit
// fields in big-endian order, so Nibbles[0] is the high nibble of the first
// byte.
type ParsedOpcode struct {
	OpcodeType OpcodeType
	Nibbles    [4]uint8
}

// The accessors below name the operand fields the way the CHIP-8 reference
// does, so instruction bodies read like the spec instead of like bit twiddling.

// X is the register index in the second nibble.
func (o ParsedOpcode) X() uint8 { return o.Nibbles[1] }

// Y is the register index in the third nibble.
func (o ParsedOpcode) Y() uint8 { return o.Nibbles[2] }

// N is the 4-bit immediate in the low nibble.
func (o ParsedOpcode) N() uint8 { return o.Nibbles[3] }

// NN is the 8-bit immediate in the low byte.
func (o ParsedOpcode) NN() uint8 { return o.Nibbles[2]<<4 | o.Nibbles[3] }

// NNN is the 12-bit address in the low three nibbles.
func (o ParsedOpcode) NNN() uint16 {
	return uint16(o.Nibbles[1])<<8 | uint16(o.Nibbles[2])<<4 | uint16(o.Nibbles[3])
}

// ParseOpcode decodes a big-endian instruction word. An unrecognised encoding
// yields OpNA with the nibbles still populated, so callers such as the
// disassembler can render it as data.
func ParseOpcode(opcode [2]uint8) ParsedOpcode {
	nibbles := splitNibbles(opcode)

	return ParsedOpcode{
		OpcodeType: decode(nibbles),
		Nibbles:    nibbles,
	}
}

func decode(n [4]uint8) OpcodeType {
	switch n[0] {
	case 0x0:
		switch {
		case n[1] == 0x0 && n[2] == 0xE && n[3] == 0x0:
			return Op00E0
		case n[1] == 0x0 && n[2] == 0xE && n[3] == 0xE:
			return Op00EE
		default:
			return Op0NNN
		}
	case 0x1:
		return Op1NNN
	case 0x2:
		return Op2NNN
	case 0x3:
		return Op3XNN
	case 0x4:
		return Op4XNN
	case 0x5:
		if n[3] == 0x0 {
			return Op5XY0
		}
	case 0x6:
		return Op6XNN
	case 0x7:
		return Op7XNN
	case 0x8:
		switch n[3] {
		case 0x0:
			return Op8XY0
		case 0x1:
			return Op8XY1
		case 0x2:
			return Op8XY2
		case 0x3:
			return Op8XY3
		case 0x4:
			return Op8XY4
		case 0x5:
			return Op8XY5
		case 0x6:
			return Op8XY6
		case 0x7:
			return Op8XY7
		case 0xE:
			return Op8XYE
		}
	case 0x9:
		if n[3] == 0x0 {
			return Op9XY0
		}
	case 0xA:
		return OpANNN
	case 0xB:
		return OpBNNN
	case 0xC:
		return OpCXNN
	case 0xD:
		return OpDXYN
	case 0xE:
		switch {
		case n[2] == 0x9 && n[3] == 0xE:
			return OpEX9E
		case n[2] == 0xA && n[3] == 0x1:
			return OpEXA1
		}
	case 0xF:
		switch n[2]<<4 | n[3] {
		case 0x07:
			return OpFX07
		case 0x0A:
			return OpFX0A
		case 0x15:
			return OpFX15
		case 0x18:
			return OpFX18
		case 0x1E:
			return OpFX1E
		case 0x29:
			return OpFX29
		case 0x33:
			return OpFX33
		case 0x55:
			return OpFX55
		case 0x65:
			return OpFX65
		}
	}
	return OpNA
}

func splitNibbles(opcode [2]uint8) [4]uint8 {
	return [4]uint8{
		(opcode[0] & 0xF0) >> 4,
		opcode[0] & 0x0F,
		(opcode[1] & 0xF0) >> 4,
		opcode[1] & 0x0F,
	}
}
