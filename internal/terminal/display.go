package teminal

import (
	"bytes"
	"fmt"
	"os"

	"github.com/0ya-sh0/chip-8/internal/vm"
)

// TerminalDisplay renders the CHIP-8 framebuffer to the terminal using ANSI escape codes.
type TerminalDisplay struct{}

func NewTerminalDisplay() *TerminalDisplay {
	// Clear screen and hide terminal cursor
	fmt.Print("\033[2J\033[?25l")
	return &TerminalDisplay{}
}

// Clear clears the terminal screen and moves the cursor to top-left.
func (t *TerminalDisplay) Clear() {
	fmt.Print("\033[2J\033[H")
}

// Draw renders the 64x32 buffer to the terminal.
func (t *TerminalDisplay) Draw(data [64][32]bool) {
	var buf bytes.Buffer

	// Move cursor to top-left (1,1) without clearing, reducing flicker
	buf.WriteString("\033[H")

	for y := 0; y < 32; y++ {
		for x := 0; x < 64; x++ {
			if data[x][y] {
				// Use two block characters per pixel to maintain a square 1:1 aspect ratio
				buf.WriteString("██")
			} else {
				buf.WriteString("  ")
			}
		}
		buf.WriteString("\n")
	}

	os.Stdout.Write(buf.Bytes())
}

// Close restores the terminal cursor when exiting.
func (t *TerminalDisplay) Close() {
	fmt.Print("\033[?25h")
}

var _ vm.DisplayProvider = (*TerminalDisplay)(nil)
