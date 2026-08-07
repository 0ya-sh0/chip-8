package asm_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/0ya-sh0/chip-8/internal/asm"
)

func TestE2ERoundTripAllROMs(t *testing.T) {
	// Relative paths from internal/asm to ROM directories
	dirs := []string{
		"../../test-roms",
		"../../test-games",
	}

	var romFiles []string

	// Discover all .ch8 files across both directories
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("failed to read directory %s: %v", dir, err)
		}

		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".ch8" {
				romFiles = append(romFiles, filepath.Join(dir, entry.Name()))
			}
		}
	}

	if len(romFiles) == 0 {
		t.Fatal("no .ch8 files found in test directories")
	}

	// Test round-trip fidelity for every discovered ROM
	for _, romPath := range romFiles {
		romPath := romPath
		fileName := filepath.Base(romPath)

		t.Run(fileName, func(t *testing.T) {
			originalBytes, err := os.ReadFile(romPath)
			if err != nil {
				t.Fatalf("failed to read ROM file %s: %v", romPath, err)
			}

			// 1. Disassemble original binary ROM to assembly text
			asmText, err := asm.Disassemble(originalBytes)
			if err != nil {
				t.Fatalf("disassembly failed for %s: %v", fileName, err)
			}

			// 2. Re-assemble disassembled text back into binary ROM
			reassembledBytes, err := asm.Assemble(asmText)
			if err != nil {
				t.Fatalf("assembly failed for %s: %v\nAssembly Output:\n%s", fileName, err, asmText)
			}

			// 3. Verify byte-for-byte equivalence
			if !bytes.Equal(originalBytes, reassembledBytes) {
				t.Errorf("round-trip mismatch for %s\nOriginal length: %d bytes | Reassembled length: %d bytes",
					fileName, len(originalBytes), len(reassembledBytes))

				// Print first byte mismatch location to aid debugging
				maxLen := len(originalBytes)
				if len(reassembledBytes) < maxLen {
					maxLen = len(reassembledBytes)
				}
				for i := 0; i < maxLen; i++ {
					if originalBytes[i] != reassembledBytes[i] {
						addr := 0x0200 + i
						t.Errorf("First mismatch at byte index %d (RAM addr 0x%04X): original 0x%02X != reassembled 0x%02X",
							i, addr, originalBytes[i], reassembledBytes[i])
						break
					}
				}
			}
		})
	}
}
