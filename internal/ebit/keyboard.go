package ebit

import (
	"github.com/0ya-sh0/chip-8/internal/vm"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// --- KeyboardProvider Mapping ---
var keyMappings = map[ebiten.Key]uint8{
	ebiten.Key1: 0x1, ebiten.Key2: 0x2, ebiten.Key3: 0x3, ebiten.Key4: 0xC,
	ebiten.KeyQ: 0x4, ebiten.KeyW: 0x5, ebiten.KeyE: 0x6, ebiten.KeyR: 0xD,
	ebiten.KeyA: 0x7, ebiten.KeyS: 0x8, ebiten.KeyD: 0x9, ebiten.KeyF: 0xE,
	ebiten.KeyZ: 0xA, ebiten.KeyX: 0x0, ebiten.KeyC: 0xB, ebiten.KeyV: 0xF,
}

// GetPressedKey implements [vm.KeyboardProvider].
func (e *EbitEngineAdapter) GetPressedKey() (key uint8, pressed bool) {
	for ebKey, chipKey := range keyMappings {
		if inpututil.IsKeyJustPressed(ebKey) || ebiten.IsKeyPressed(ebKey) {
			return chipKey, true
		}
	}
	return 0, false
}

// IsKeyPressed implements [vm.KeyboardProvider].
func (e *EbitEngineAdapter) IsKeyPressed(key uint8) bool {
	for ebKey, chipKey := range keyMappings {
		if chipKey == key && ebiten.IsKeyPressed(ebKey) {
			return true
		}
	}
	return false
}

var _ vm.KeyboardProvider = (*EbitEngineAdapter)(nil)
