package ebit

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

type EbitEngineAdapter struct {
	buf          [64][32]bool
	offscreenImg *ebiten.Image
	audioCtx     *audio.Context
	audioPlayer  *audio.Player
}

func NewEbitenAdapter() *EbitEngineAdapter {
	audioCtx, audioPlayer, err := setupAudio()
	if err != nil {
		log.Printf("failed to initialize audio player: %v", err)
	}
	return &EbitEngineAdapter{offscreenImg: ebiten.NewImage(64, 32), audioCtx: audioCtx, audioPlayer: audioPlayer}
}

// Close implements [vm.KeyboardProvider], [vm.DisplayProvider].
func (e *EbitEngineAdapter) Close() {
}
