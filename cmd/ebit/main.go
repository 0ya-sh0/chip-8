// Command chip8-gui runs a CHIP-8 ROM in an Ebitengine window.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/0ya-sh0/chip-8/internal/ebit"
	"github.com/0ya-sh0/chip-8/internal/vm"
	"github.com/hajimehoshi/ebiten/v2"
)

const scale = 15

// Game adapts the emulator to Ebitengine's game loop.
type Game struct {
	adapter *ebit.EbitEngineAdapter
	cancel  context.CancelFunc
	done    <-chan error
}

// Update runs at 60Hz on Ebitengine's goroutine. Input is sampled here because
// Ebitengine's key state is only valid inside Update; the VM reads the latch
// the adapter fills, which is safe to poll from its own goroutine.
func (g *Game) Update() error {
	g.adapter.PollInput()

	select {
	case err := <-g.done:
		if err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		return ebiten.Termination
	default:
		return nil
	}
}

func (g *Game) Draw(screen *ebiten.Image) { g.adapter.Render(screen) }

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return vm.ScreenWidth * scale, vm.ScreenHeight * scale
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "chip8-gui:", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return errors.New("usage: chip8-gui <rom.ch8>")
	}

	adapter := ebit.NewEbitenAdapter()
	machine := vm.NewChip8VM(adapter, adapter, adapter)
	if err := machine.LoadROMFromFile(os.Args[1]); err != nil {
		return err
	}

	// The VM runs on its own goroutine; cancelling on return guarantees it does
	// not outlive the window.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- machine.Start(ctx) }()

	ebiten.SetWindowSize(vm.ScreenWidth*scale, vm.ScreenHeight*scale)
	ebiten.SetWindowTitle("CHIP-8")

	if err := ebiten.RunGame(&Game{adapter: adapter, cancel: cancel, done: done}); err != nil {
		return err
	}
	return nil
}
