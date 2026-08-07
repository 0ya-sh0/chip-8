package asm

import (
	"strings"
	"testing"
)

func TestDisassemble(t *testing.T) {
	rom := []byte{
		0x00, 0xE0, // CLS
		0x60, 0x0A, // LD V0, 0x0A
		0x12, 0x00, // JP 0x200
	}

	result, err := Disassemble(rom)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedLines := []string{
		"0x0200:  00E0  CLS",
		"0x0202:  600A  LD V0, 0x0A",
		"0x0204:  1200  JP 0x200",
	}

	for _, expected := range expectedLines {
		if !strings.Contains(result, expected) {
			t.Errorf("expected output to contain %q, but got:\n%s", expected, result)
		}
	}
}
