package main

import (
	"context"
	"fmt"
	"os"

	"github.com/0ya-sh0/chip-8/internal/display"
	"github.com/0ya-sh0/chip-8/internal/vm"
)

func main() {
	vm := vm.NewChip8VM(nil, &display.TerminalDisplay{})
	if len(os.Args) < 2 {
		fmt.Printf("Usage: ./chip-8 row.chip8\n")
		os.Exit(1)
	}
	vm.LoadROMFromFile(os.Args[1])
	vm.Start(context.Background())
}
