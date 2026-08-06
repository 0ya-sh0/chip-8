package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/0ya-sh0/chip-8/internal/display"
	"github.com/0ya-sh0/chip-8/internal/keyboard"
	"github.com/0ya-sh0/chip-8/internal/sound"
	"github.com/0ya-sh0/chip-8/internal/vm"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: ./chip-8 row.chip8\n")
		os.Exit(1)
	}
	romPath := os.Args[1]

	disp := display.NewTerminalDisplay()
	defer disp.Close()

	kb, err := keyboard.NewTerminalKeyboard()
	if err != nil {
		fmt.Printf("Failed to initialize keyboard: %v\n", err)
		return
	}
	defer kb.Close()

	chip8 := vm.NewChip8VM(kb, disp, sound.NewTerminalSound())
	if err := chip8.LoadROMFromFile(romPath); err != nil {
		fmt.Printf("Failed to load ROM: %v\n", err)
		return
	}

	// 4. Set up Graceful Shutdown Context (Ctrl+C / SIGINT)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 5. Start the CPU loop
	chip8.Start(ctx)
}
