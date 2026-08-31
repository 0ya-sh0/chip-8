package main

import "github.com/0ya-sh0/chip-8/internal/vm"

// Wire protocol.
//
// The two directions carry different kinds of data, so they use the two
// WebSocket message types rather than inventing a tag inside a single JSON
// envelope:
//
//   - Binary frames are packed framebuffers: 256 bytes, one bit per pixel, in
//     scanline order. Sending [64][32]bool as JSON costs ~10KB for the same
//     2048 pixels, which at 60fps is 600KB/s of avoidable traffic.
//   - Text frames are JSON control messages, which are rare and benefit from
//     being self-describing.
//
// The client distinguishes them with `ws.binaryType = "arraybuffer"` and a
// typeof check in onmessage.
const (
	msgSoundPlay   = "sound.play"
	msgSoundStop   = "sound.stop"
	msgDisplayInit = "display.init"
	msgError       = "error"
)

// controlMessage is a server-to-client JSON message.
type controlMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	// Width and Height are sent once on connect so the client does not have to
	// hardcode the geometry to unpack binary frames.
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
}

// clientMessage is a client-to-server JSON message.
type clientMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

const (
	clientKeyDown = "key.down"
	clientKeyUp   = "key.up"
	clientBlur    = "focus.lost"
)

// keypadCodes maps the hex labels the client sends to CHIP-8 keypad indices.
//
// A lookup miss must be reported rather than defaulted: an unmapped label
// silently becoming key 0 is invisible in testing and produces phantom presses.
var keypadCodes = map[string]uint8{
	"0": 0x0, "1": 0x1, "2": 0x2, "3": 0x3,
	"4": 0x4, "5": 0x5, "6": 0x6, "7": 0x7,
	"8": 0x8, "9": 0x9,
	"A": 0xA, "B": 0xB, "C": 0xC, "D": 0xD, "E": 0xE, "F": 0xF,
	"a": 0xA, "b": 0xB, "c": 0xC, "d": 0xD, "e": 0xE, "f": 0xF,
}

func lookupKey(label string) (uint8, bool) {
	code, ok := keypadCodes[label]
	return code, ok
}

func displayInitMessage() controlMessage {
	return controlMessage{
		Type:   msgDisplayInit,
		Width:  vm.ScreenWidth,
		Height: vm.ScreenHeight,
	}
}
