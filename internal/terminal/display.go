package terminal

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/0ya-sh0/chip-8/internal/vm"
)

// RefreshInterval matches the 60Hz scanout of the original hardware.
const RefreshInterval = time.Second / 60

// TerminalDisplay renders the CHIP-8 framebuffer with ANSI escape codes.
//
// Writing to stdout is a blocking syscall and Draw runs on the VM goroutine, so
// the display coalesces: Draw only stores the frame, and [TerminalDisplay.Run]
// repaints at 60Hz. Painting on every DXYN would put a terminal write on the
// instruction path - which stalls emulation and, because key state is only
// sampled between instructions, silently drops keypresses.
type TerminalDisplay struct {
	mu    sync.Mutex
	frame vm.FrameBuffer
	gen   uint64

	out      *bufio.Writer
	done     chan struct{}
	closeOne sync.Once
}

func NewTerminalDisplay() *TerminalDisplay {
	// Clear the screen and hide the cursor.
	fmt.Print("\033[2J\033[?25l")
	return &TerminalDisplay{
		out:  bufio.NewWriterSize(os.Stdout, 16*1024),
		done: make(chan struct{}),
	}
}

// Clear implements [vm.DisplayProvider].
func (t *TerminalDisplay) Clear() {
	t.mu.Lock()
	t.frame = vm.FrameBuffer{}
	t.gen++
	t.mu.Unlock()
}

// Draw implements [vm.DisplayProvider]. It stores and returns; see the type doc
// for why it must not paint.
func (t *TerminalDisplay) Draw(frame vm.FrameBuffer) {
	t.mu.Lock()
	t.frame = frame
	t.gen++
	t.mu.Unlock()
}

// Run repaints until Close is called. Run it on its own goroutine.
func (t *TerminalDisplay) Run() {
	tick := time.NewTicker(RefreshInterval)
	defer tick.Stop()

	var last uint64
	for {
		select {
		case <-t.done:
			return
		case <-tick.C:
			t.mu.Lock()
			frame, gen := t.frame, t.gen
			t.mu.Unlock()
			if gen == last {
				// Unchanged since the last paint; redrawing would only flicker.
				continue
			}
			last = gen
			t.paint(&frame)
		}
	}
}

func (t *TerminalDisplay) paint(frame *vm.FrameBuffer) {
	// Home the cursor rather than clearing the screen, which would flicker.
	t.out.WriteString("\033[H")
	for y := 0; y < vm.ScreenHeight; y++ {
		for x := 0; x < vm.ScreenWidth; x++ {
			// Two cells per pixel keeps the aspect ratio roughly square.
			if frame[y][x] {
				t.out.WriteString("██")
			} else {
				t.out.WriteString("  ")
			}
		}
		t.out.WriteByte('\n')
	}
	t.out.Flush()
}

// Close stops the repaint loop and restores the cursor. It is safe to call
// more than once.
func (t *TerminalDisplay) Close() {
	t.closeOne.Do(func() {
		close(t.done)
		fmt.Print("\033[?25h")
	})
}

var _ vm.DisplayProvider = (*TerminalDisplay)(nil)
