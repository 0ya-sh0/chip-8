package vm

import "testing"

func TestParseOpcode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		word  [2]uint8
		want  OpcodeType
		x, y  uint8
		n, nn uint8
		nnn   uint16
	}{
		{name: "clear screen", word: [2]uint8{0x00, 0xE0}, want: Op00E0, y: 0xE, nn: 0xE0, nnn: 0x0E0},
		{name: "return", word: [2]uint8{0x00, 0xEE}, want: Op00EE, y: 0xE, n: 0xE, nn: 0xEE, nnn: 0x0EE},
		{name: "sys call", word: [2]uint8{0x01, 0x23}, want: Op0NNN, x: 1, y: 2, n: 3, nn: 0x23, nnn: 0x123},
		{name: "jump", word: [2]uint8{0x12, 0x34}, want: Op1NNN, x: 2, y: 3, n: 4, nn: 0x34, nnn: 0x234},
		{name: "call", word: [2]uint8{0x2A, 0xBC}, want: Op2NNN, x: 0xA, y: 0xB, n: 0xC, nn: 0xBC, nnn: 0xABC},
		{name: "skip eq imm", word: [2]uint8{0x35, 0xFF}, want: Op3XNN, x: 5, y: 0xF, n: 0xF, nn: 0xFF, nnn: 0x5FF},
		{name: "skip ne imm", word: [2]uint8{0x41, 0x00}, want: Op4XNN, x: 1, nn: 0x00, nnn: 0x100},
		{name: "skip eq reg", word: [2]uint8{0x51, 0x20}, want: Op5XY0, x: 1, y: 2, nn: 0x20, nnn: 0x120},
		{name: "load imm", word: [2]uint8{0x60, 0x0A}, want: Op6XNN, y: 0, n: 0xA, nn: 0x0A, nnn: 0x00A},
		{name: "add imm", word: [2]uint8{0x71, 0x01}, want: Op7XNN, x: 1, n: 1, nn: 0x01, nnn: 0x101},
		{name: "alu mov", word: [2]uint8{0x81, 0x20}, want: Op8XY0, x: 1, y: 2, nn: 0x20, nnn: 0x120},
		{name: "alu or", word: [2]uint8{0x81, 0x21}, want: Op8XY1, x: 1, y: 2, n: 1, nn: 0x21, nnn: 0x121},
		{name: "alu and", word: [2]uint8{0x81, 0x22}, want: Op8XY2, x: 1, y: 2, n: 2, nn: 0x22, nnn: 0x122},
		{name: "alu xor", word: [2]uint8{0x81, 0x23}, want: Op8XY3, x: 1, y: 2, n: 3, nn: 0x23, nnn: 0x123},
		{name: "alu add", word: [2]uint8{0x81, 0x24}, want: Op8XY4, x: 1, y: 2, n: 4, nn: 0x24, nnn: 0x124},
		{name: "alu sub", word: [2]uint8{0x81, 0x25}, want: Op8XY5, x: 1, y: 2, n: 5, nn: 0x25, nnn: 0x125},
		{name: "alu shr", word: [2]uint8{0x81, 0x26}, want: Op8XY6, x: 1, y: 2, n: 6, nn: 0x26, nnn: 0x126},
		{name: "alu subn", word: [2]uint8{0x81, 0x27}, want: Op8XY7, x: 1, y: 2, n: 7, nn: 0x27, nnn: 0x127},
		{name: "alu shl", word: [2]uint8{0x81, 0x2E}, want: Op8XYE, x: 1, y: 2, n: 0xE, nn: 0x2E, nnn: 0x12E},
		{name: "skip ne reg", word: [2]uint8{0x91, 0x20}, want: Op9XY0, x: 1, y: 2, nn: 0x20, nnn: 0x120},
		{name: "set index", word: [2]uint8{0xA2, 0x00}, want: OpANNN, x: 2, nnn: 0x200},
		{name: "jump plus v0", word: [2]uint8{0xB1, 0x23}, want: OpBNNN, x: 1, y: 2, n: 3, nn: 0x23, nnn: 0x123},
		{name: "random", word: [2]uint8{0xC3, 0x0F}, want: OpCXNN, x: 3, n: 0xF, nn: 0x0F, nnn: 0x30F},
		{name: "draw", word: [2]uint8{0xD1, 0x25}, want: OpDXYN, x: 1, y: 2, n: 5, nn: 0x25, nnn: 0x125},
		{name: "skip if key", word: [2]uint8{0xE1, 0x9E}, want: OpEX9E, x: 1, y: 9, n: 0xE, nn: 0x9E, nnn: 0x19E},
		{name: "skip if not key", word: [2]uint8{0xE1, 0xA1}, want: OpEXA1, x: 1, y: 0xA, n: 1, nn: 0xA1, nnn: 0x1A1},
		{name: "read delay", word: [2]uint8{0xF1, 0x07}, want: OpFX07, x: 1, n: 7, nn: 0x07, nnn: 0x107},
		{name: "wait key", word: [2]uint8{0xF1, 0x0A}, want: OpFX0A, x: 1, n: 0xA, nn: 0x0A, nnn: 0x10A},
		{name: "set delay", word: [2]uint8{0xF1, 0x15}, want: OpFX15, x: 1, y: 1, n: 5, nn: 0x15, nnn: 0x115},
		{name: "set sound", word: [2]uint8{0xF1, 0x18}, want: OpFX18, x: 1, y: 1, n: 8, nn: 0x18, nnn: 0x118},
		{name: "add index", word: [2]uint8{0xF1, 0x1E}, want: OpFX1E, x: 1, y: 1, n: 0xE, nn: 0x1E, nnn: 0x11E},
		{name: "font", word: [2]uint8{0xF1, 0x29}, want: OpFX29, x: 1, y: 2, n: 9, nn: 0x29, nnn: 0x129},
		{name: "bcd", word: [2]uint8{0xF1, 0x33}, want: OpFX33, x: 1, y: 3, n: 3, nn: 0x33, nnn: 0x133},
		{name: "store regs", word: [2]uint8{0xF1, 0x55}, want: OpFX55, x: 1, y: 5, n: 5, nn: 0x55, nnn: 0x155},
		{name: "load regs", word: [2]uint8{0xF1, 0x65}, want: OpFX65, x: 1, y: 6, n: 5, nn: 0x65, nnn: 0x165},

		// Encodings with no instruction must decode to OpNA rather than being
		// mistaken for a neighbouring opcode.
		{name: "bad 5XY nonzero", word: [2]uint8{0x51, 0x23}, want: OpNA, x: 1, y: 2, n: 3, nn: 0x23, nnn: 0x123},
		{name: "bad 9XY nonzero", word: [2]uint8{0x91, 0x21}, want: OpNA, x: 1, y: 2, n: 1, nn: 0x21, nnn: 0x121},
		{name: "bad 8XY subcode", word: [2]uint8{0x81, 0x28}, want: OpNA, x: 1, y: 2, n: 8, nn: 0x28, nnn: 0x128},
		{name: "bad EX suffix", word: [2]uint8{0xE1, 0x9F}, want: OpNA, x: 1, y: 9, n: 0xF, nn: 0x9F, nnn: 0x19F},
		{name: "bad FX suffix", word: [2]uint8{0xF1, 0x99}, want: OpNA, x: 1, y: 9, n: 9, nn: 0x99, nnn: 0x199},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ParseOpcode(tt.word)
			if got.OpcodeType != tt.want {
				t.Errorf("ParseOpcode(%02X%02X) type = %v, want %v",
					tt.word[0], tt.word[1], got.OpcodeType, tt.want)
			}
			if got.X() != tt.x {
				t.Errorf("X() = %#x, want %#x", got.X(), tt.x)
			}
			if got.Y() != tt.y {
				t.Errorf("Y() = %#x, want %#x", got.Y(), tt.y)
			}
			if got.N() != tt.n {
				t.Errorf("N() = %#x, want %#x", got.N(), tt.n)
			}
			if got.NN() != tt.nn {
				t.Errorf("NN() = %#x, want %#x", got.NN(), tt.nn)
			}
			if got.NNN() != tt.nnn {
				t.Errorf("NNN() = %#x, want %#x", got.NNN(), tt.nnn)
			}
		})
	}
}

// TestParseOpcodeTotal checks that decoding never panics and never yields a
// type outside the enum, for every one of the 65536 possible words.
func TestParseOpcodeTotal(t *testing.T) {
	t.Parallel()

	for hi := 0; hi < 256; hi++ {
		for lo := 0; lo < 256; lo++ {
			got := ParseOpcode([2]uint8{uint8(hi), uint8(lo)})
			if got.OpcodeType >= NumOpcodeTypes {
				t.Fatalf("ParseOpcode(%02X%02X) = %d, out of range", hi, lo, got.OpcodeType)
			}
			if got.OpcodeType.String() == "INVALID" {
				t.Fatalf("ParseOpcode(%02X%02X) has no name", hi, lo)
			}
		}
	}
}

func TestOpcodeTypeString(t *testing.T) {
	t.Parallel()

	if got := Op8XY4.String(); got != "8XY4" {
		t.Errorf("Op8XY4.String() = %q, want %q", got, "8XY4")
	}
	if got := OpcodeType(NumOpcodeTypes).String(); got != "INVALID" {
		t.Errorf("out-of-range String() = %q, want %q", got, "INVALID")
	}
	// Every declared opcode must have a name, or stats output would show
	// blank keys.
	for i := OpcodeType(0); i < NumOpcodeTypes; i++ {
		if i.String() == "" {
			t.Errorf("OpcodeType(%d) has an empty name", i)
		}
	}
}
