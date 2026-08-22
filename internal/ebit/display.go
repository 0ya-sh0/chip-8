package ebit

import (
	"image/color"

	"github.com/0ya-sh0/chip-8/internal/vm"
	"github.com/hajimehoshi/ebiten/v2"
)

func (e *EbitEngineAdapter) Render(screen *ebiten.Image) {
	e.offscreenImg.Clear()
	for x := 0; x < 64; x++ {
		for y := 0; y < 32; y++ {
			if e.buf[x][y] {
				e.offscreenImg.Set(x, y, color.White)
			}
		}
	}

	// Scale 64x32 buffer up to fit the main window
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(15, 15) // 64x15 = 960 width, 32x15 = 480 height
	screen.DrawImage(e.offscreenImg, opts)
}

// Clear implements [vm.DisplayProvider].
func (e *EbitEngineAdapter) Clear() {
	e.buf = vm.FrameBuffer{}
}

// Draw implements [vm.DisplayProvider].
func (e *EbitEngineAdapter) Draw(buf vm.FrameBuffer) {
	e.buf = buf
}

var _ vm.DisplayProvider = (*EbitEngineAdapter)(nil)
