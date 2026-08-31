// Command chip8 runs a CHIP-8 ROM in the terminal.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/0ya-sh0/chip-8/internal/terminal"
	"github.com/0ya-sh0/chip-8/internal/vm"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "chip8:", err)
		os.Exit(1)
	}
}

// run exists so that deferred terminal restoration still happens on the error
// path; os.Exit from main would skip every defer and leave the tty in cbreak
// mode with the cursor hidden.
func run() error {
	if len(os.Args) < 2 {
		return errors.New("usage: chip8 <rom.ch8>")
	}
	romPath := os.Args[1]

	disp := terminal.NewTerminalDisplay()
	defer disp.Close()
	go disp.Run()

	kb, err := terminal.NewTerminalKeyboard()
	if err != nil {
		return fmt.Errorf("initialise keyboard: %w", err)
	}
	defer kb.Close()

	machine := vm.NewChip8VM(kb, disp, terminal.NewTerminalSound())
	if err := machine.LoadROMFromFile(romPath); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// A clean quit surfaces as context.Canceled; anything else is a real fault
	// and deserves a non-zero exit status.
	if err := machine.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
