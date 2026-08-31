package vm

// Display geometry. CHIP-8 is a fixed 64x32 monochrome display.
const (
	ScreenWidth  = 64
	ScreenHeight = 32

	// PackedFrameSize is the length of the bitmap produced by
	// [FrameBuffer.Pack]: one bit per pixel.
	PackedFrameSize = ScreenWidth * ScreenHeight / 8
)

// FrameBuffer is the CHIP-8 display, indexed [y][x].
//
// Row-major ordering matches how every consumer walks it (scanline order) and
// makes a row of pixels contiguous, which is what [FrameBuffer.Pack] needs.
type FrameBuffer [ScreenHeight][ScreenWidth]bool

// Pack encodes the framebuffer as a 1bpp bitmap in scanline order, MSB first
// within each byte. A 2048-pixel display is 256 bytes.
//
// This is the wire format: sending one bool per pixel costs 8x more as raw
// bytes and roughly 40x more once JSON-encoded.
func (fb *FrameBuffer) Pack() []byte {
	out := make([]byte, PackedFrameSize)
	i := 0
	for y := range fb {
		row := &fb[y]
		for x := 0; x < ScreenWidth; x += 8 {
			var b byte
			for bit := 0; bit < 8; bit++ {
				if row[x+bit] {
					b |= 0x80 >> bit
				}
			}
			out[i] = b
			i++
		}
	}
	return out
}

// KeyboardProvider reports the state of the 16-key hex keypad.
//
// Implementations are polled from the VM goroutine at instruction rate and
// updated from whatever goroutine owns the real input device, so they must be
// safe for concurrent use. See [KeyLatch], which implements the tricky part.
type KeyboardProvider interface {
	// GetPressedKey returns any key currently considered down. It is used by
	// FX0A and must not block.
	GetPressedKey() (key uint8, pressed bool)

	// IsKeyPressed reports whether the given key (0x0-0xF) is considered down.
	IsKeyPressed(key uint8) bool
}

// DisplayProvider receives framebuffer updates.
//
// Draw is called from the VM goroutine once per DXYN, which can be thousands of
// times per second. Implementations must return promptly: store the frame and
// let a separate goroutine do any I/O. Blocking here stalls emulation and,
// because key state is only sampled between instructions, silently drops input.
type DisplayProvider interface {
	Clear()
	Draw(FrameBuffer)
}

// SoundProvider is driven by edges, not levels: the VM calls PlaySound exactly
// once when the buzzer starts and StopSound exactly once when it stops.
// Implementations do not need to deduplicate.
type SoundProvider interface {
	PlaySound()
	StopSound()
}
