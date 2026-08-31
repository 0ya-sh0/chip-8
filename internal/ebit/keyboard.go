package ebit

import (
	"github.com/0ya-sh0/chip-8/internal/vm"
	"github.com/hajimehoshi/ebiten/v2"
)

// keyMappings maps host keys to the CHIP-8 hex keypad in the conventional
// 4x4 layout.
var keyMappings = map[ebiten.Key]uint8{
	ebiten.Key1: 0x1, ebiten.Key2: 0x2, ebiten.Key3: 0x3, ebiten.Key4: 0xC,
	ebiten.KeyQ: 0x4, ebiten.KeyW: 0x5, ebiten.KeyE: 0x6, ebiten.KeyR: 0xD,
	ebiten.KeyA: 0x7, ebiten.KeyS: 0x8, ebiten.KeyD: 0x9, ebiten.KeyF: 0xE,
	ebiten.KeyZ: 0xA, ebiten.KeyX: 0x0, ebiten.KeyC: 0xB, ebiten.KeyV: 0xF,
}

// PollInput samples the host keyboard and feeds edges into the latch.
//
// It must be called from Ebitengine's Update, and only from there: input state
// is refreshed once per tick and reading it from another goroutine is both
// racy and meaningless. The VM polls the latch instead, which is safe for
// concurrent use, so the two run at their own rates without touching each
// other's state.
func (e *EbitEngineAdapter) PollInput() {
	for hostKey, chipKey := range keyMappings {
		if ebiten.IsKeyPressed(hostKey) {
			e.keys.Press(chipKey)
		} else {
			e.keys.Release(chipKey)
		}
	}
}

var _ vm.KeyboardProvider = (*EbitEngineAdapter)(nil)
