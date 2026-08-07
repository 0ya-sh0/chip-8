package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0ya-sh0/chip-8/internal/asm"
)

func main() {
	mode := flag.String("m", "disasm", "Mode: 'asm' (assemble) or 'disasm' (disassemble)")
	input := flag.String("i", "", "Input file path (.asm or .ch8)")
	output := flag.String("o", "", "Output file path (optional)")

	flag.Parse()

	if *input == "" {
		fmt.Println("Usage: go run cmd/casm/main.go -m [asm|disasm] -i <input-file> [-o <output-file>]")
		os.Exit(1)
	}

	data, err := os.ReadFile(*input)
	if err != nil {
		fmt.Printf("Error reading input file: %v\n", err)
		os.Exit(1)
	}

	switch strings.ToLower(*mode) {
	case "disasm":
		asmText, err := asm.Disassemble(data)
		if err != nil {
			fmt.Printf("Disassembly error: %v\n", err)
			os.Exit(1)
		}

		outPath := *output
		if outPath == "" {
			outPath = strings.TrimSuffix(*input, filepath.Ext(*input)) + ".asm"
		}

		if err := os.WriteFile(outPath, []byte(asmText), 0644); err != nil {
			fmt.Printf("Error writing assembly file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully disassembled %s -> %s\n", *input, outPath)

	case "asm":
		romBytes, err := asm.Assemble(string(data))
		if err != nil {
			fmt.Printf("Assembly error: %v\n", err)
			os.Exit(1)
		}

		outPath := *output
		if outPath == "" {
			outPath = strings.TrimSuffix(*input, filepath.Ext(*input)) + ".ch8"
		}

		if err := os.WriteFile(outPath, romBytes, 0644); err != nil {
			fmt.Printf("Error writing ROM file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully assembled %s -> %s (%d bytes)\n", *input, outPath, len(romBytes))

	default:
		fmt.Printf("Unknown mode: %s. Use 'asm' or 'disasm'.\n", *mode)
		os.Exit(1)
	}
}
