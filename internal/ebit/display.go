package ebit

import (
	"image/color"

	"github.com/0ya-sh0/chip-8/internal/vm"
	"github.com/hajimehoshi/ebiten/v2"
)

// Render draws the latest framebuffer. It runs on Ebitengine's game loop
// goroutine, so it takes a snapshot under the mutex rather than reading the
// shared buffer while the VM mutates it.
func (e *EbitEngineAdapter) Render(screen *ebiten.Image) {
	e.mu.Lock()
	frame := e.buf
	e.mu.Unlock()

	e.offscreenImg.Clear()
	for y := 0; y < vm.ScreenHeight; y++ {
		for x := 0; x < vm.ScreenWidth; x++ {
			if frame[y][x] {
				e.offscreenImg.Set(x, y, color.White)
			}
		}
	}

	opts := &ebiten.DrawImageOptions{}
	scale := float64(screen.Bounds().Dx()) / vm.ScreenWidth
	opts.GeoM.Scale(scale, scale)
	screen.DrawImage(e.offscreenImg, opts)
}

// Clear implements [vm.DisplayProvider].
func (e *EbitEngineAdapter) Clear() {
	e.mu.Lock()
	e.buf = vm.FrameBuffer{}
	e.mu.Unlock()
}

// Draw implements [vm.DisplayProvider]. It only stores the frame; scaling and
// rasterising happen on the render goroutine so the VM is never blocked.
func (e *EbitEngineAdapter) Draw(buf vm.FrameBuffer) {
	e.mu.Lock()
	e.buf = buf
	e.mu.Unlock()
}

var _ vm.DisplayProvider = (*EbitEngineAdapter)(nil)
