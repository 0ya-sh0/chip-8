package main

import (
	"encoding/json"
	"testing"

	"github.com/0ya-sh0/chip-8/internal/vm"
)

func testFrame() vm.FrameBuffer {
	var fb vm.FrameBuffer
	for y := range fb {
		for x := range fb[y] {
			fb[y][x] = (x+y)%3 == 0
		}
	}
	return fb
}

// BenchmarkFrameEncodeJSONBools reproduces the original wire format: the whole
// [32][64]bool array marshalled inside a JSON envelope. Compare its ns/op,
// B/op and the reported bytes against BenchmarkFrameEncodePacked - that ratio
// is the cost of sending one bool per pixel.
func BenchmarkFrameEncodeJSONBools(b *testing.B) {
	fb := testFrame()
	msg := map[string]any{"type": "display.draw", "data": fb}

	out, err := json.Marshal(msg)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(out)))
	b.ReportMetric(float64(len(out)), "wirebytes")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := json.Marshal(msg); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFrameEncodePacked is the current format: a 1bpp bitmap sent as a
// binary WebSocket message, with no envelope at all.
func BenchmarkFrameEncodePacked(b *testing.B) {
	fb := testFrame()

	b.SetBytes(vm.PackedFrameSize)
	b.ReportMetric(float64(vm.PackedFrameSize), "wirebytes")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fb.Pack()
	}
}

// BenchmarkDrawStore measures what the VM goroutine actually pays per DXYN.
// This is the number that matters for input latency: everything spent here is
// time the interpreter is not sampling the keypad.
func BenchmarkDrawStore(b *testing.B) {
	s := &session{}
	fb := testFrame()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Draw(fb)
	}
}
