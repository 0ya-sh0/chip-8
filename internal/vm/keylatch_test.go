package vm

import (
	"sync"
	"testing"
	"time"
)

func newTestLatch(minHold time.Duration) (*KeyLatch, *fakeClock) {
	clock := newFakeClock()
	return NewKeyLatch(minHold, WithLatchClock(clock.Now)), clock
}

// simulateTap runs a tap against a poller ticking every pollGap and reports
// whether any poll observed it. Both the tap and the polls advance the same
// fake clock, so the timing is exact rather than approximate.
func simulateTap(latch *KeyLatch, clock *fakeClock, key uint8, offset, tapLen, pollGap, runFor time.Duration) bool {
	const step = time.Millisecond

	var (
		elapsed           time.Duration
		nextPoll          time.Duration
		pressed, released bool
		seen              bool
	)
	for elapsed <= runFor {
		if !pressed && elapsed >= offset {
			latch.Press(key)
			pressed = true
		}
		if pressed && !released && elapsed >= offset+tapLen {
			latch.Release(key)
			released = true
		}
		if elapsed >= nextPoll {
			if latch.IsKeyPressed(key) {
				seen = true
			}
			nextPoll += pollGap
		}
		clock.Advance(step)
		elapsed += step
	}
	return seen
}

// This is the defect the latch exists for: a tap shorter than the interval
// between two polls used to be invisible, so players had to hold keys.
//
// The guarantee is minHold >= pollGap, and it is swept across every phase
// offset because a tap that lands just after a poll is the worst case and a
// fixed offset could pass by luck.
func TestShortTapSurvivesASlowPoller(t *testing.T) {
	t.Parallel()

	const (
		minHold = 120 * time.Millisecond
		tapLen  = 20 * time.Millisecond
		pollGap = 100 * time.Millisecond
	)

	for offset := time.Duration(0); offset < pollGap; offset += time.Millisecond {
		latch, clock := newTestLatch(minHold)
		if !simulateTap(latch, clock, 0x5, offset, tapLen, pollGap, offset+minHold+pollGap) {
			t.Fatalf("tap at offset %s was dropped; minHold=%s must cover pollGap=%s",
				offset, minHold, pollGap)
		}
		if got := latch.Stats().Unobserved; got != 0 {
			t.Fatalf("offset %s: Stats().Unobserved = %d, want 0", offset, got)
		}
	}
}

// Beyond minHold, a press that no poll has seen yet stays pending rather than
// expiring. This is what makes taps survive the long gaps a ROM produces while
// it is busy redrawing, without having to size minHold for the worst case.
func TestTapsSurviveGapsFarLongerThanMinHold(t *testing.T) {
	t.Parallel()

	const (
		minHold = 30 * time.Millisecond
		tapLen  = 10 * time.Millisecond
		pollGap = 250 * time.Millisecond // well past minHold, well under maxHold
	)

	for offset := time.Duration(0); offset < pollGap; offset += 5 * time.Millisecond {
		latch, clock := newTestLatch(minHold)
		if !simulateTap(latch, clock, 0x5, offset, tapLen, pollGap, offset+2*pollGap) {
			t.Fatalf("tap at offset %s was dropped despite a %s poll gap", offset, pollGap)
		}
	}
}

// The guarantee is bounded, though: past maxHold a pending press is abandoned,
// because delivering a keystroke a second late is worse than losing it.
func TestPendingPressesExpireAtMaxHold(t *testing.T) {
	t.Parallel()

	latch, clock := newTestLatch(20 * time.Millisecond)

	latch.Press(0x5)
	clock.Advance(10 * time.Millisecond)
	latch.Release(0x5)

	clock.Advance(DefaultMaxHold / 2)
	if !latch.IsKeyPressed(0x5) {
		t.Fatal("press abandoned before maxHold")
	}

	latch2, clock2 := newTestLatch(20 * time.Millisecond)
	latch2.Press(0x5)
	clock2.Advance(10 * time.Millisecond)
	latch2.Release(0x5)
	clock2.Advance(DefaultMaxHold + 50*time.Millisecond)
	if latch2.IsKeyPressed(0x5) {
		t.Error("press still pending past maxHold")
	}
}

func TestLongHoldReleasesImmediately(t *testing.T) {
	t.Parallel()

	latch, clock := newTestLatch(100 * time.Millisecond)

	latch.Press(0x3)
	// Held far longer than minHold.
	for i := 0; i < 10; i++ {
		clock.Advance(500 * time.Millisecond)
		if !latch.IsKeyPressed(0x3) {
			t.Fatalf("key read as up while still held (iteration %d)", i)
		}
	}

	latch.Release(0x3)
	// The deadline is anchored to the press, which is long past, so there is no
	// tail. Anchoring to the release instead would leave the key down for
	// another minHold and make every hold overshoot.
	if latch.IsKeyPressed(0x3) {
		t.Error("key still down immediately after releasing a long hold")
	}
}

// Browsers and terminals both re-send key-down while a key is held. If those
// repeats reset the press timestamp, every long hold gains a minHold tail.
func TestAutorepeatDoesNotExtendTheTail(t *testing.T) {
	t.Parallel()

	latch, clock := newTestLatch(100 * time.Millisecond)

	latch.Press(0x1)
	clock.Advance(500 * time.Millisecond)
	for i := 0; i < 20; i++ {
		latch.Press(0x1) // autorepeat
		clock.Advance(30 * time.Millisecond)
	}
	latch.Release(0x1)

	if latch.IsKeyPressed(0x1) {
		t.Error("key still down after release; autorepeat reset the press time")
	}
	if got := latch.Stats().Presses; got != 1 {
		t.Errorf("Stats().Presses = %d, want 1; repeats were counted as new presses", got)
	}
}

func TestTapExpiresAfterMinHold(t *testing.T) {
	t.Parallel()

	const minHold = 100 * time.Millisecond
	latch, clock := newTestLatch(minHold)

	latch.Press(0x9)
	clock.Advance(10 * time.Millisecond)
	latch.Release(0x9)

	clock.Advance(80 * time.Millisecond) // 90ms since press
	if !latch.IsKeyPressed(0x9) {
		t.Error("key expired before minHold elapsed")
	}
	clock.Advance(20 * time.Millisecond) // 110ms since press
	if latch.IsKeyPressed(0x9) {
		t.Error("key still down after minHold elapsed")
	}
}

func TestGetPressedKeyPrefersMostRecent(t *testing.T) {
	t.Parallel()

	latch, clock := newTestLatch(200 * time.Millisecond)

	latch.Press(0x2)
	clock.Advance(10 * time.Millisecond)
	latch.Press(0xE)

	// Both are down. Returning the lowest index would make rolling from one key
	// to the next feel stuck on the first.
	if got, ok := latch.GetPressedKey(); !ok || got != 0xE {
		t.Errorf("GetPressedKey() = %#x, %v; want 0xE, true", got, ok)
	}
}

func TestReleaseAllClearsHeldKeys(t *testing.T) {
	t.Parallel()

	latch, clock := newTestLatch(50 * time.Millisecond)

	latch.Press(0x1)
	latch.Press(0x2)
	clock.Advance(500 * time.Millisecond)

	// Simulates the tab losing focus: the key-ups will never arrive.
	latch.ReleaseAll()
	if _, ok := latch.GetPressedKey(); ok {
		t.Error("keys still held after ReleaseAll")
	}
}

// The drop metric has to separate two very different situations, or it reads
// as broken on any ROM that uses only a few of the sixteen keys.
func TestDropAccountingSeparatesUnpolledKeys(t *testing.T) {
	t.Parallel()

	t.Run("key the rom never reads counts as unpolled", func(t *testing.T) {
		t.Parallel()
		latch, clock := newTestLatch(10 * time.Millisecond)

		for i := 0; i < 3; i++ {
			latch.Press(0x4)
			clock.Advance(5 * time.Millisecond)
			latch.Release(0x4)
			clock.Advance(50 * time.Millisecond)
		}
		latch.Press(0x4) // retires the third press for accounting

		got := latch.Stats()
		if got.Unpolled != 3 {
			t.Errorf("Unpolled = %d, want 3", got.Unpolled)
		}
		if got.Unobserved != 0 {
			t.Errorf("Unobserved = %d, want 0; nothing ever read this key", got.Unobserved)
		}
	})

	t.Run("key the rom reads only past maxHold counts as unobserved", func(t *testing.T) {
		t.Parallel()
		latch, clock := newTestLatch(10 * time.Millisecond)

		// The ROM does read this key - but never within maxHold of a press.
		latch.IsKeyPressed(0x4)

		for i := 0; i < 3; i++ {
			latch.Press(0x4)
			clock.Advance(5 * time.Millisecond)
			latch.Release(0x4)
			clock.Advance(DefaultMaxHold + 50*time.Millisecond)
			latch.IsKeyPressed(0x4)
		}
		latch.Press(0x4)

		got := latch.Stats()
		if got.Unobserved != 3 {
			t.Errorf("Unobserved = %d, want 3", got.Unobserved)
		}
		if got.Unpolled != 0 {
			t.Errorf("Unpolled = %d, want 0", got.Unpolled)
		}
	})
}

func TestOutOfRangeKeysAreIgnored(t *testing.T) {
	t.Parallel()

	latch, _ := newTestLatch(0)
	latch.Press(0xFF)
	latch.Release(0xFF)
	if latch.IsKeyPressed(0xFF) {
		t.Error("out-of-range key reported as pressed")
	}
}

// The latch is written by input goroutines and read by the VM goroutine, so it
// must be race-free. Run with -race for this to mean anything.
func TestConcurrentAccess(t *testing.T) {
	t.Parallel()

	latch := NewKeyLatch(DefaultMinHold)

	var wg sync.WaitGroup
	stop := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			key := uint8(i % NumKeys)
			latch.Press(key)
			latch.Release(key)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			latch.IsKeyPressed(uint8(i % NumKeys))
			latch.GetPressedKey()
			latch.Stats()
		}
	}()

	time.Sleep(20 * time.Millisecond)
	close(stop)
	wg.Wait()
}
