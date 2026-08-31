package vm

import (
	"testing"
	"time"
)

// benchProviders is a do-nothing provider set, so benchmarks measure the
// interpreter rather than whatever backend happens to be attached.
type benchProviders struct{}

func (benchProviders) IsKeyPressed(uint8) bool      { return false }
func (benchProviders) GetPressedKey() (uint8, bool) { return 0, false }
func (benchProviders) Clear()                       {}
func (benchProviders) Draw(FrameBuffer)             {}
func (benchProviders) PlaySound()                   {}
func (benchProviders) StopSound()                   {}

func BenchmarkParseOpcode(b *testing.B) {
	words := [][2]uint8{
		{0x00, 0xE0}, {0x12, 0x34}, {0x6A, 0xBC}, {0x8A, 0xB4},
		{0xDA, 0xB5}, {0xFA, 0x33}, {0xEA, 0x9E}, {0xAB, 0xCD},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ParseOpcode(words[i%len(words)])
	}
}

// BenchmarkExecArithmetic measures the cost of an instruction that touches no
// provider. Individual opcodes are far too fast to instrument with time.Now
// (~20-50ns per call), which is why per-opcode cost is measured here instead.
func BenchmarkExecArithmetic(b *testing.B) {
	p := benchProviders{}
	v := NewChip8VM(p, p, p)
	op := ParseOpcode([2]uint8{0x81, 0x24}) // ADD V1, V2

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.pc = ProgramStart
		_ = v.exec(op)
	}
}

// BenchmarkExecDraw is the expensive instruction: a full 8x15 sprite blit plus
// the provider call.
func BenchmarkExecDraw(b *testing.B) {
	p := benchProviders{}
	v := NewChip8VM(p, p, p)
	for i := 0; i < 15; i++ {
		v.ram[0x300+i] = 0b1010_1010
	}
	v.idxreg = 0x300
	op := ParseOpcode([2]uint8{0xD0, 0x1F}) // DRW V0, V1, 15

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.pc = ProgramStart
		_ = v.exec(op)
	}
}

func BenchmarkFrameBufferPack(b *testing.B) {
	var fb FrameBuffer
	for y := range fb {
		for x := range fb[y] {
			fb[y][x] = (x+y)%3 == 0
		}
	}

	b.ReportAllocs()
	b.SetBytes(PackedFrameSize)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fb.Pack()
	}
}

// The next two benchmarks compare the two ways a DisplayProvider can publish a
// frame to a slower consumer. Draw is called thousands of times per second, so
// the difference is per-frame allocation, not speed.

func BenchmarkFramePublishCopy(b *testing.B) {
	var src, dst FrameBuffer
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst = src
		// Feeding a byte back prevents the compiler from proving the copy dead
		// and deleting it, which would report an impossible sub-nanosecond cost.
		src[0][0] = dst[ScreenHeight-1][ScreenWidth-1]
	}
}

func BenchmarkFramePublishPointer(b *testing.B) {
	var src FrameBuffer
	var sink *FrameBuffer
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		frame := src // escapes, so each publish heap-allocates a fresh 2KB frame
		sink = &frame
	}
	_ = sink
}

func BenchmarkKeyLatchPoll(b *testing.B) {
	latch := NewKeyLatch(DefaultMinHold)
	latch.Press(0x5)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = latch.IsKeyPressed(0x5)
	}
}

// BenchmarkClockNow calibrates time.Now on this host. It is not a benchmark of
// this code, but every duration measured anywhere in the project is built out
// of two of these calls, so it is the baseline the others must be read against.
//
// Expect ~25ns where the kernel exposes a TSC-backed vDSO clock. If it reports
// microseconds, check:
//
//	cat /sys/devices/system/clocksource/clocksource0/current_clocksource
//
// An hpet or acpi_pm clocksource makes every time.Now a slow MMIO read.
func BenchmarkClockNow(b *testing.B) {
	var t time.Time
	for i := 0; i < b.N; i++ {
		t = time.Now()
	}
	_ = t
}
