package asm

import (
	"bytes"
	"testing"
)

func TestAssembleAndDisassembleRoundTrip(t *testing.T) {
	originalAsm := `
CLS
LD V0, 0x0A
LD V1, 0x0F
LD I, 0x0200
DRW V0, V1, 5
ADD V0, 0x01
SE V0, V1
JP 0x0200
`

	romBytes, err := Assemble(originalAsm)
	if err != nil {
		t.Fatalf("failed to assemble: %v", err)
	}

	disassembled, err := Disassemble(romBytes)
	if err != nil {
		t.Fatalf("failed to disassemble: %v", err)
	}

	// Re-assemble the disassembled output to verify 100% round-trip fidelity
	reAssembledBytes, err := Assemble(disassembled)
	if err != nil {
		t.Fatalf("failed to re-assemble disassembled text: %v", err)
	}

	if !bytes.Equal(romBytes, reAssembledBytes) {
		t.Errorf("round-trip failed!\nOriginal ROM: %X\nReassembled ROM: %X", romBytes, reAssembledBytes)
	}
}

func TestAssemblerWithLabels(t *testing.T) {
	source := `
START:
    CLS
    LD V0, 0x05
LOOP:
    ADD V0, 0x01
    JP LOOP
`

	rom, err := Assemble(source)
	if err != nil {
		t.Fatalf("failed to assemble labeled program: %v", err)
	}

	// Expected opcodes:
	// 0x0200: CLS         -> 00 E0
	// 0x0202: LD V0, 0x05 -> 60 05
	// 0x0204: ADD V0, 1   -> 70 01
	// 0x0206: JP 0x0204   -> 12 04
	expected := []byte{
		0x00, 0xE0,
		0x60, 0x05,
		0x70, 0x01,
		0x12, 0x04,
	}

	if !bytes.Equal(rom, expected) {
		t.Errorf("expected %X, got %X", expected, rom)
	}
}
