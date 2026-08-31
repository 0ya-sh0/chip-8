package terminal

import (
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0ya-sh0/chip-8/internal/vm"
)

// keyMap maps QWERTY characters to the CHIP-8 hex keypad.
var keyMap = map[byte]uint8{
	'1': 0x1, '2': 0x2, '3': 0x3, '4': 0xC,
	'q': 0x4, 'w': 0x5, 'e': 0x6, 'r': 0xD,
	'a': 0x7, 's': 0x8, 'd': 0x9, 'f': 0xE,
	'z': 0xA, 'x': 0x0, 'c': 0xB, 'v': 0xF,
}

// repeatGrace is how long a key is held down after its last byte arrives.
//
// It must exceed the terminal's autorepeat *period* (typically 30ms at 33
// repeats/sec) so that a held key does not flicker between repeats.
const repeatGrace = 60 * time.Millisecond

// TerminalKeyboard reads raw bytes from a cbreak-mode terminal.
//
// A terminal in cbreak mode is edge-triggered and one-directional: it delivers
// a byte when a key goes down and delivers nothing at all when it comes up.
// There is no way to know a key was released, so releases have to be inferred
// from silence - see repeatGrace.
//
// This is fundamentally lossy and no amount of tuning fixes it. Autorepeat
// sends the first byte, then pauses for the typematic delay (500-660ms on a
// typical Linux desktop) before repeating, so a genuinely held key reads as
// released during that gap. A terminal that supports the kitty keyboard
// protocol, or reading evdev directly, would provide real key-up events; the
// WebSocket and Ebitengine backends already do, which is why only this backend
// has to guess.
type TerminalKeyboard struct {
	// KeyLatch provides IsKeyPressed/GetPressedKey. The latch's minimum-hold
	// window still helps here: it guarantees a single tap stays observable long
	// enough for the VM to sample it.
	*vm.KeyLatch

	mu       sync.Mutex
	lastByte [vm.NumKeys]time.Time

	closed   atomic.Bool
	done     chan struct{}
	closeOne sync.Once
}

func NewTerminalKeyboard() (*TerminalKeyboard, error) {
	if err := setRawMode(true); err != nil {
		return nil, err
	}

	tk := &TerminalKeyboard{
		KeyLatch: vm.NewKeyLatch(vm.DefaultMinHold),
		done:     make(chan struct{}),
	}
	go tk.readLoop()
	go tk.releaseLoop()
	return tk, nil
}

// readLoop turns incoming bytes into key-down edges.
//
// It blocks in a read that no context can interrupt, so it exits when stdin
// returns an error or when the process does. That is acceptable for a
// foreground CLI whose lifetime is the process; pretending otherwise with a
// select/default around a blocking read would just be misleading.
func (tk *TerminalKeyboard) readLoop() {
	buf := make([]byte, 16)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return
		}
		if tk.closed.Load() {
			return
		}
		now := time.Now()
		for _, c := range buf[:n] {
			key, ok := keyMap[c]
			if !ok {
				continue
			}
			tk.mu.Lock()
			tk.lastByte[key] = now
			tk.mu.Unlock()
			// Press is idempotent while the key is already down, so a stream of
			// autorepeat bytes does not disturb the press timestamp.
			tk.KeyLatch.Press(key)
		}
	}
}

// releaseLoop infers key-up from the absence of further bytes.
func (tk *TerminalKeyboard) releaseLoop() {
	tick := time.NewTicker(repeatGrace / 4)
	defer tick.Stop()

	for {
		select {
		case <-tk.done:
			return
		case now := <-tick.C:
			tk.mu.Lock()
			for key := range tk.lastByte {
				last := tk.lastByte[key]
				if !last.IsZero() && now.Sub(last) > repeatGrace {
					tk.lastByte[key] = time.Time{}
					tk.KeyLatch.Release(uint8(key))
				}
			}
			tk.mu.Unlock()
		}
	}
}

// Close restores the terminal. It is safe to call more than once.
func (tk *TerminalKeyboard) Close() {
	tk.closeOne.Do(func() {
		tk.closed.Store(true)
		close(tk.done)
		_ = setRawMode(false)
	})
}

// setRawMode toggles cbreak/-echo via stty. The -F flag is Linux-only, so the
// macOS form is tried as a fallback.
func setRawMode(on bool) error {
	args := []string{"cbreak", "-echo"}
	if !on {
		args = []string{"-cbreak", "echo"}
	}

	cmd := exec.Command("stty", append([]string{"-F", "/dev/tty"}, args...)...)
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err == nil {
		return nil
	}

	cmd = exec.Command("stty", args...)
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

var _ vm.KeyboardProvider = (*TerminalKeyboard)(nil)
