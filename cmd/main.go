package main

import (
	"context"

	"github.com/0ya-sh0/chip-8/internal/display"
	"github.com/0ya-sh0/chip-8/internal/vm"
)

func main() {
	vm := vm.NewChip8VM(nil, &display.LoggerDisplay{})
	vm.LoadROMFromFile("/home/yash/coding/chip-8/e2e/1-chip8-logo.ch8")
	vm.Start(context.Background())
}
