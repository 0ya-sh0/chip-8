package keyboard

import (
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/0ya-sh0/chip-8/internal/vm"
)

// Keymap: QWERTY -> CHIP-8 Hex Keypad (0x0 - 0xF)
var keyMap = map[byte]uint8{
	'1': 0x1, '2': 0x2, '3': 0x3, '4': 0xC,
	'q': 0x4, 'w': 0x5, 'e': 0x6, 'r': 0xD,
	'a': 0x7, 's': 0x8, 'd': 0x9, 'f': 0xE,
	'z': 0xA, 'x': 0x0, 'c': 0xB, 'v': 0xF,
}

type TerminalKeyboard struct {
	mu           sync.RWMutex
	lastPressed  [16]time.Time
	holdDuration time.Duration
	stopChan     chan struct{}
}

func NewTerminalKeyboard() (*TerminalKeyboard, error) {
	// Put terminal in raw mode so keypresses don't echo and don't require Enter
	cmd := exec.Command("stty", "-F", "/dev/tty", "cbreak", "-echo")
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		// Fallback for macOS where -F is not supported
		cmd = exec.Command("stty", "cbreak", "-echo")
		cmd.Stdin = os.Stdin
		_ = cmd.Run()
	}

	tk := &TerminalKeyboard{
		holdDuration: 150 * time.Millisecond,
		stopChan:     make(chan struct{}),
	}

	go tk.listenInput()
	return tk, nil
}

func (tk *TerminalKeyboard) listenInput() {
	buf := make([]byte, 1)
	for {
		select {
		case <-tk.stopChan:
			return
		default:
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				time.Sleep(10 * time.Millisecond)
				continue
			}

			char := buf[0]
			if hexKey, ok := keyMap[char]; ok {
				tk.mu.Lock()
				tk.lastPressed[hexKey] = time.Now()
				tk.mu.Unlock()
			}
		}
	}
}

// IsKeyPressed returns true if the key was pressed within the last 150ms.
func (tk *TerminalKeyboard) IsKeyPressed(key uint8) bool {
	if key > 0xF {
		return false
	}

	tk.mu.RLock()
	defer tk.mu.RUnlock()

	return time.Since(tk.lastPressed[key]) < tk.holdDuration
}

// GetPressedKey implements Pattern 1 non-blocking polling for OpFX0A.
func (tk *TerminalKeyboard) GetPressedKey() (uint8, bool) {
	tk.mu.RLock()
	defer tk.mu.RUnlock()

	now := time.Now()
	for key := uint8(0); key <= 0xF; key++ {
		if now.Sub(tk.lastPressed[key]) < tk.holdDuration {
			return key, true
		}
	}
	return 0, false
}

// Close restores the terminal settings when exiting.
func (tk *TerminalKeyboard) Close() {
	close(tk.stopChan)
	cmd := exec.Command("stty", "-F", "/dev/tty", "-cbreak", "echo")
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("stty", "-cbreak", "echo")
		cmd.Stdin = os.Stdin
		_ = cmd.Run()
	}
}

var _ vm.KeyboardProvider = (*TerminalKeyboard)(nil)
