package ebit

import (
	"bytes"

	"github.com/0ya-sh0/chip-8/internal/vm"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

const (
	sampleRate = 44100
	frequency  = 440 // A4 note (440 Hz)
)

func setupAudio() (*audio.Context, *audio.Player, error) {
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
		return nil, nil, err
	}

	return audioCtx, player, nil
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

var _ vm.SoundProvider = (*EbitEngineAdapter)(nil)
