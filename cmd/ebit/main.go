package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/0ya-sh0/chip-8/internal/ebit"
	"github.com/0ya-sh0/chip-8/internal/vm"
	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
	chip8   *vm.Chip8VM
	adapter *ebit.EbitEngineAdapter
}

func (g *Game) Update() error {
	// Ebitengine update loop runs at 60Hz
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.adapter.Render(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 960, 480 // Scaled resolution
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: ./chip-8 row.chip8\n")
		os.Exit(1)
	}
	romPath := os.Args[1]

	adapter := ebit.NewEbitenAdapter()
	chip8 := vm.NewChip8VM(adapter, adapter, adapter)
	if err := chip8.LoadROMFromFile(romPath); err != nil {
		fmt.Printf("Failed to load ROM: %v\n", err)
		return
	}

	// Start CPU loop in background goroutine
	go chip8.Start(context.Background())

	ebiten.SetWindowSize(960, 480)
	ebiten.SetWindowTitle("CHIP-8 Emulator (Ebitengine GUI)")

	game := &Game{chip8: chip8, adapter: adapter}
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}

}
