package ebit

import (
	"image/color"
	"log"

	"github.com/0ya-sh0/chip-8/internal/vm"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"bytes"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

const (
	sampleRate = 44100
	frequency  = 440 // A4 note (440 Hz)
)

type EbitEngineAdapter struct {
	buf          [64][32]bool
	offscreenImg *ebiten.Image
	audioCtx     *audio.Context
	audioPlayer  *audio.Player
}

func NewEbitenAdapter() *EbitEngineAdapter {
	audioCtx := audio.NewContext(sampleRate)

	// 2. Generate 1 second of 440 Hz square wave PCM data (16-bit stereo)
	pcm := make([]byte, sampleRate*4) // 4 bytes per sample (2 for L, 2 for R)
	period := sampleRate / frequency

	for i := 0; i < sampleRate; i++ {
		var val int16 = 3000 // Comfortably low volume (max int16 is 32767)
		if (i % period) < (period / 2) {
			val = -3000
		}

		b1 := byte(val)
		b2 := byte(val >> 8)

		idx := i * 4
		// Left channel
		pcm[idx] = b1
		pcm[idx+1] = b2
		// Right channel
		pcm[idx+2] = b1
		pcm[idx+3] = b2
	}

	// 3. Wrap in an infinite loop stream and create player
	loopStream := audio.NewInfiniteLoop(bytes.NewReader(pcm), int64(len(pcm)))
	player, err := audioCtx.NewPlayer(loopStream)
	if err != nil {
		log.Printf("failed to initialize audio player: %v", err)
	}
	return &EbitEngineAdapter{offscreenImg: ebiten.NewImage(64, 32), audioCtx: audioCtx, audioPlayer: player}
}

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

// PlaySound implements [vm.SoundProvider].
func (e *EbitEngineAdapter) PlaySound() {
	if e.audioPlayer != nil && !e.audioPlayer.IsPlaying() {
		e.audioPlayer.Play()
	}
}

// StopSound implements [vm.SoundProvider].
func (e *EbitEngineAdapter) StopSound() {
	if e.audioPlayer != nil && e.audioPlayer.IsPlaying() {
		e.audioPlayer.Pause()
	}
}

// Clear implements [vm.DisplayProvider].
func (e *EbitEngineAdapter) Clear() {
	e.buf = [64][32]bool{}
}

// Draw implements [vm.DisplayProvider].
func (e *EbitEngineAdapter) Draw(buf [64][32]bool) {
	e.buf = buf
}

// Close implements [vm.KeyboardProvider].
func (e *EbitEngineAdapter) Close() {
	panic("unimplemented")
}

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
var _ vm.DisplayProvider = (*EbitEngineAdapter)(nil)
var _ vm.SoundProvider = (*EbitEngineAdapter)(nil)
