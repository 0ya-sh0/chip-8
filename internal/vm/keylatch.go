package vm

import (
	"sync"
	"time"
)

// NumKeys is the size of the CHIP-8 hex keypad.
const NumKeys = 16

// DefaultMinHold is the minimum time a press stays observable. It must exceed
// the worst-case interval between two key polls, which is set by how many
// instructions a ROM executes between its EX9E/EXA1 checks. Measure the real
// distribution with [KeyLatch.Stats] before changing it.
const DefaultMinHold = 120 * time.Millisecond

// DefaultMaxHold caps how long an unobserved press is kept alive.
//
// minHold alone is not enough. A ROM scans the keypad one key at a time, so any
// single key is only polled once per full sweep - for a 700Hz interpreter
// spending ~5 instructions per key that is over 100ms - and the sweep pauses
// entirely while the game redraws. Rather than sizing minHold for the worst
// sweep (which would leave every key stuck down for a second), a press that no
// poll has seen yet stays pending until one does, bounded by this.
const DefaultMaxHold = time.Second

// observeGrace keeps a press readable for a couple of frames after its first
// sighting, so a game loop that tests the same key twice gets the same answer
// both times.
const observeGrace = 2 * time.Second / 60

// KeyLatch turns key *edges* into a level the VM can poll without losing short
// presses.
//
// The problem it solves: the VM only observes the keypad when it executes
// EX9E/EXA1/FX0A, which for a typical ROM is once per game loop - tens to
// hundreds of milliseconds apart. A 50ms tap that begins and ends between two
// polls is invisible, so the player has to hold keys unnaturally long.
//
// Real hardware did not have this problem because its CPU polled the keypad
// continuously; the gap is an artefact of emulation, so the fix is deliberately
// not spec-faithful.
//
// The fix keeps two facts per key: whether it is physically down, and when the
// current press began. A press reads as down until the key is released *and*
// minHold has elapsed since the press started. Anchoring the deadline to the
// press (not the release) means long holds release immediately - only short
// taps get padded.
//
// KeyLatch is safe for concurrent use: input goroutines call Press/Release
// while the VM goroutine polls.
type KeyLatch struct {
	mu      sync.Mutex
	minHold time.Duration
	maxHold time.Duration
	now     func() time.Time
	keys    [NumKeys]keyState

	// Instrumentation. The poll gap is the number that determines whether
	// minHold is large enough, so it is measured rather than guessed.
	polls      uint64
	maxPollGap time.Duration
	gapBuckets [len(gapBucketBounds) + 1]uint64
	presses    uint64
	unobserved uint64
	unpolled   uint64
	observeSum time.Duration
	observeMax time.Duration
	observeQty uint64
}

type keyState struct {
	down      bool
	pressedAt time.Time
	releaseAt time.Time
	// observed reports whether any poll saw the current or most recent press.
	observed bool
	// polls counts how many times this specific key has been sampled. A press
	// on a key the ROM never reads is not a dropped input - most ROMs use only
	// a handful of the sixteen keys - so the two cases are separated.
	polls uint64
	// lastPoll times the gap between samples *of this key*. The global rate at
	// which the VM touches the keypad is not the relevant number: a ROM scans
	// one key per iteration, so an individual key is read far less often than
	// the keypad as a whole.
	lastPoll time.Time
}

// gapBucketBounds are the upper bounds, in milliseconds, of the poll-gap
// histogram.
var gapBucketBounds = [...]time.Duration{1, 2, 5, 10, 20, 50, 100, 200, 500}

// LatchOption configures a [KeyLatch].
type LatchOption func(*KeyLatch)

// WithLatchClock replaces the time source, which makes latch behaviour
// deterministic in tests.
func WithLatchClock(now func() time.Time) LatchOption {
	return func(l *KeyLatch) { l.now = now }
}

// WithMaxHold caps how long an unobserved press stays pending. Setting it equal
// to minHold disables the hold-until-observed behaviour entirely, which is
// useful for measuring what that behaviour is worth on a given ROM.
func WithMaxHold(d time.Duration) LatchOption {
	return func(l *KeyLatch) {
		if d >= l.minHold {
			l.maxHold = d
		}
	}
}

// NewKeyLatch returns a latch that keeps presses observable for at least
// minHold. Passing zero uses [DefaultMinHold].
func NewKeyLatch(minHold time.Duration, opts ...LatchOption) *KeyLatch {
	if minHold <= 0 {
		minHold = DefaultMinHold
	}
	maxHold := DefaultMaxHold
	if minHold > maxHold {
		maxHold = minHold
	}
	l := &KeyLatch{minHold: minHold, maxHold: maxHold, now: time.Now}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Press records a key-down edge. Repeated presses without an intervening
// release are ignored, which makes the latch immune to OS/browser key
// autorepeat - without this, a held key would keep resetting the press
// timestamp and every hold would gain a spurious minHold tail on release.
func (l *KeyLatch) Press(key uint8) {
	if key >= NumKeys {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	k := &l.keys[key]
	if k.down {
		return
	}
	if k.pressedAt != (time.Time{}) && !k.observed {
		// Retire the previous press. Only blame the latch when the ROM
		// actually reads this key; otherwise the press had no audience.
		if k.polls > 0 {
			l.unobserved++
		} else {
			l.unpolled++
		}
	}
	now := l.now()
	k.down = true
	k.pressedAt = now
	k.releaseAt = time.Time{}
	k.observed = false
	l.presses++
}

// Release records a key-up edge. The key stays observable until minHold has
// elapsed since the press began.
func (l *KeyLatch) Release(key uint8) {
	if key >= NumKeys {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	k := &l.keys[key]
	if !k.down {
		return
	}
	k.down = false
	k.releaseAt = k.pressedAt.Add(l.minHold)
}

// ReleaseAll cancels every key. Call it when input focus is lost, otherwise a
// key held at that moment never receives its key-up and sticks forever.
//
// Unlike Release this discards pending presses outright rather than keeping
// them alive for a later poll: losing focus means the player is no longer
// playing, and delivering a keystroke after they have alt-tabbed away is worse
// than dropping it.
func (l *KeyLatch) ReleaseAll() {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := range l.keys {
		k := &l.keys[i]
		k.down = false
		k.pressedAt = time.Time{}
		k.releaseAt = time.Time{}
		// Marked observed so the cancellation is not later counted as a drop.
		k.observed = true
	}
}

// IsKeyPressed implements [KeyboardProvider].
func (l *KeyLatch) IsKeyPressed(key uint8) bool {
	if key >= NumKeys {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.isDown(key, l.now())
}

// GetPressedKey implements [KeyboardProvider]. When several keys qualify it
// returns the most recently pressed one, which is what a player expects when
// rolling from one key to the next.
func (l *KeyLatch) GetPressedKey() (uint8, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()

	best, found := uint8(0), false
	var bestAt time.Time
	for i := uint8(0); i < NumKeys; i++ {
		if !l.isDown(i, now) {
			continue
		}
		if !found || l.keys[i].pressedAt.After(bestAt) {
			best, bestAt, found = i, l.keys[i].pressedAt, true
		}
	}
	return best, found
}

// isDown must be called with l.mu held.
func (l *KeyLatch) isDown(key uint8, now time.Time) bool {
	k := &l.keys[key]
	l.recordPoll(k, now)

	if k.down {
		l.markObserved(k, now)
		return true
	}
	if k.pressedAt.IsZero() {
		return false
	}

	// The key is released. It stays readable until minHold has elapsed - but if
	// no poll has seen this press yet, keep it pending up to maxHold rather than
	// letting it expire unobserved. That is the difference between "the player
	// tapped too fast" and "the game was busy when they tapped".
	deadline := k.releaseAt
	if !k.observed {
		if limit := k.pressedAt.Add(l.maxHold); limit.After(deadline) {
			deadline = limit
		}
	}
	if !now.Before(deadline) {
		return false
	}

	if !k.observed && !now.Before(k.releaseAt) {
		// First sighting of a press that outlived minHold. Extend it slightly so
		// the answer is stable across one game loop instead of flickering.
		k.releaseAt = now.Add(observeGrace)
	}
	l.markObserved(k, now)
	return true
}

// markObserved must be called with l.mu held.
func (l *KeyLatch) markObserved(k *keyState, now time.Time) {
	if k.observed {
		return
	}
	k.observed = true
	d := now.Sub(k.pressedAt)
	l.observeSum += d
	l.observeQty++
	if d > l.observeMax {
		l.observeMax = d
	}
}

// recordPoll must be called with l.mu held. Gaps are measured per key, since
// that is what determines whether a press can be missed.
func (l *KeyLatch) recordPoll(k *keyState, now time.Time) {
	l.polls++
	k.polls++
	if !k.lastPoll.IsZero() {
		gap := now.Sub(k.lastPoll)
		if gap > l.maxPollGap {
			l.maxPollGap = gap
		}
		l.gapBuckets[bucketFor(gap)]++
	}
	k.lastPoll = now
}

func bucketFor(gap time.Duration) int {
	for i, bound := range gapBucketBounds {
		if gap < bound*time.Millisecond {
			return i
		}
	}
	return len(gapBucketBounds)
}

// KeyStats is a snapshot of latch instrumentation.
type KeyStats struct {
	// Polls is how many times the VM sampled the keypad.
	Polls uint64
	// PollGapP50 and PollGapP99 approximate the interval between samples from
	// the histogram; PollGapMax is exact. If P99 approaches or exceeds MinHold,
	// presses are being dropped.
	PollGapP50 time.Duration
	PollGapP99 time.Duration
	PollGapMax time.Duration
	// Presses counted. Unobserved is the number that expired without any poll
	// seeing them despite the ROM reading that key - the drop rate that
	// matters, and it should be zero. Unpolled counts presses on keys the ROM
	// never reads at all, which is normal.
	Presses    uint64
	Unobserved uint64
	Unpolled   uint64
	// ObserveLatency is press-to-first-observation.
	ObserveLatencyAvg time.Duration
	ObserveLatencyMax time.Duration
	MinHold           time.Duration
	MaxHold           time.Duration
}

// Stats returns a snapshot of latch instrumentation.
func (l *KeyLatch) Stats() KeyStats {
	l.mu.Lock()
	defer l.mu.Unlock()

	var avg time.Duration
	if l.observeQty > 0 {
		avg = l.observeSum / time.Duration(l.observeQty)
	}
	return KeyStats{
		Polls:             l.polls,
		PollGapP50:        l.percentile(0.50),
		PollGapP99:        l.percentile(0.99),
		PollGapMax:        l.maxPollGap,
		Presses:           l.presses,
		Unobserved:        l.unobserved,
		Unpolled:          l.unpolled,
		ObserveLatencyAvg: avg,
		ObserveLatencyMax: l.observeMax,
		MinHold:           l.minHold,
		MaxHold:           l.maxHold,
	}
}

// percentile must be called with l.mu held. It returns the upper bound of the
// bucket containing the requested quantile, so it over-estimates within a
// bucket - fine for deciding whether minHold is in the right order of
// magnitude, which is all it is used for.
func (l *KeyLatch) percentile(q float64) time.Duration {
	var total uint64
	for _, c := range l.gapBuckets {
		total += c
	}
	if total == 0 {
		return 0
	}
	target := uint64(q * float64(total))
	var seen uint64
	for i, c := range l.gapBuckets {
		seen += c
		if seen >= target {
			if i >= len(gapBucketBounds) {
				return l.maxPollGap
			}
			return gapBucketBounds[i] * time.Millisecond
		}
	}
	return l.maxPollGap
}

var _ KeyboardProvider = (*KeyLatch)(nil)
