package ebit

import (
	"log"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"

	"github.com/0ya-sh0/chip-8/internal/vm"
)

// EbitEngineAdapter bridges the VM to Ebitengine.
//
// Two goroutines meet here: the VM goroutine calls Draw/PlaySound/StopSound and
// polls the keypad, while Ebitengine's game loop calls Render and PollInput.
// Every field shared between them is therefore synchronised - the framebuffer
// by a mutex, the keypad by [vm.KeyLatch].
type EbitEngineAdapter struct {
	// KeyLatch is embedded so the adapter satisfies vm.KeyboardProvider using
	// the same edge-to-level logic as every other backend.
	*vm.KeyLatch
	keys *vm.KeyLatch

	mu  sync.Mutex
	buf vm.FrameBuffer

	offscreenImg *ebiten.Image
	audioCtx     *audio.Context
	audioPlayer  *audio.Player
}

func NewEbitenAdapter() *EbitEngineAdapter {
	audioCtx, audioPlayer, err := setupAudio()
	if err != nil {
		// Audio is not essential; a silent emulator still plays.
		log.Printf("ebit: audio unavailable: %v", err)
	}
	latch := vm.NewKeyLatch(vm.DefaultMinHold)
	return &EbitEngineAdapter{
		KeyLatch:     latch,
		keys:         latch,
		offscreenImg: ebiten.NewImage(vm.ScreenWidth, vm.ScreenHeight),
		audioCtx:     audioCtx,
		audioPlayer:  audioPlayer,
	}
}
