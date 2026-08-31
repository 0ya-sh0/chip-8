package main

import (
	"context"
	"errors"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0ya-sh0/chip-8/internal/vm"
	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

const (
	// frameInterval is the client-facing refresh rate. The CHIP-8 display was
	// scanned out at 60Hz by dedicated hardware; DXYN only mutates the
	// framebuffer. Emitting a frame per DXYN instead would be roughly 40x more
	// traffic than the client can use.
	frameInterval = time.Second / 60

	// WebSocket keepalive. Without these a client that vanishes without a close
	// frame (laptop lid, dropped wifi) leaves the session running forever.
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = pongWait * 9 / 10

	// controlBuffer sizes the must-deliver control queue. It only ever holds
	// sound and lifecycle events, so it is generous relative to real traffic.
	controlBuffer = 64

	// maxClientMessage caps inbound frames. Key events are tiny; anything
	// larger is a bug or an attack.
	maxClientMessage = 512
)

var errControlBacklog = errors.New("client is not draining control messages")

// session owns one WebSocket connection and the VM behind it.
//
// Goroutine layout, all supervised by a single errgroup so that the death of
// any one tears down the rest:
//
//	readLoop   - owns conn reads, translates key events into the latch
//	writeLoop  - sole owner of conn writes; paces frames at 60Hz
//	vmLoop     - runs the interpreter
//	closer     - closes the conn on cancellation to unblock the blocking reads
//
// The VM goroutine never performs I/O: Draw stores into a buffer under a mutex
// and returns. That matters because key state is only sampled between
// instructions, so any blocking on the instruction path shows up as dropped
// input, not just as lag.
type session struct {
	// KeyLatch supplies IsKeyPressed/GetPressedKey, so session satisfies
	// vm.KeyboardProvider by embedding rather than by reimplementing the
	// edge-to-level logic per backend.
	*vm.KeyLatch

	conn    *websocket.Conn
	machine *vm.Chip8VM
	rom     ROM

	// frame is the latest rendered framebuffer. A mutex-protected copy beats an
	// atomic.Pointer swap here: Draw is called thousands of times per second
	// and publishing a pointer would heap-allocate a fresh 2KB frame each time,
	// nearly all of which the 60Hz reader discards.
	mu       sync.Mutex
	frame    vm.FrameBuffer
	frameGen uint64

	control chan controlMessage

	cancel context.CancelCauseFunc
	stats  sessionStats
}

type sessionStats struct {
	framesSent     atomic.Uint64
	framesSkipped  atomic.Uint64
	bytesSent      atomic.Uint64
	controlSent    atomic.Uint64
	controlDropped atomic.Uint64
	unknownKeys    atomic.Uint64
	keyEvents      atomic.Uint64
}

func newSession(conn *websocket.Conn, rom ROM, cpuHz int, minHold, maxHold time.Duration) *session {
	s := &session{
		KeyLatch: vm.NewKeyLatch(minHold, vm.WithMaxHold(maxHold)),
		conn:     conn,
		rom:      rom,
		control:  make(chan controlMessage, controlBuffer),
	}
	s.machine = vm.NewChip8VM(s, s, s, vm.WithCPURate(cpuHz))
	return s
}

// run drives the session until the client disconnects, the ROM faults, or the
// parent context is cancelled.
func (s *session) run(parent context.Context) error {
	ctx, cancel := context.WithCancelCause(parent)
	defer cancel(nil)
	s.cancel = cancel

	if err := s.machine.LoadROMFromFile(s.rom.Path); err != nil {
		return err
	}

	s.sendControl(displayInitMessage())

	g, ctx := errgroup.WithContext(ctx)

	// Closing the connection is what unblocks readLoop, which is otherwise
	// parked in a blocking read that no context can interrupt.
	g.Go(func() error {
		<-ctx.Done()
		_ = s.conn.Close()
		return nil
	})
	g.Go(func() error { return s.readLoop(ctx) })
	g.Go(func() error { return s.writeLoop(ctx) })
	g.Go(func() error { return s.machine.Start(ctx) })

	err := g.Wait()
	if cause := context.Cause(ctx); cause != nil && !errors.Is(cause, context.Canceled) {
		err = cause
	}
	return normalizeShutdown(err)
}

// normalizeShutdown folds the several ways a healthy session ends into nil, so
// callers only log genuine faults.
func normalizeShutdown(err error) error {
	switch {
	case err == nil,
		errors.Is(err, context.Canceled),
		errors.Is(err, net.ErrClosed),
		websocket.IsCloseError(err,
			websocket.CloseNormalClosure,
			websocket.CloseGoingAway,
			websocket.CloseNoStatusReceived):
		return nil
	}
	return err
}

// fail records the first real cause of shutdown and unwinds the group.
func (s *session) fail(err error) {
	if s.cancel != nil {
		s.cancel(err)
	}
}

// readLoop owns all reads from the connection.
func (s *session) readLoop(ctx context.Context) error {
	s.conn.SetReadLimit(maxClientMessage)
	if err := s.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		return err
	}
	s.conn.SetPongHandler(func(string) error {
		return s.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		var msg clientMessage
		if err := s.conn.ReadJSON(&msg); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		s.handleClientMessage(msg)
	}
}

func (s *session) handleClientMessage(msg clientMessage) {
	switch msg.Type {
	case clientKeyDown, clientKeyUp:
		code, ok := lookupKey(msg.Data)
		if !ok {
			// Never default to key 0: a typo in the client's binding table
			// would otherwise show up as the game reacting to a key nobody
			// pressed, with nothing logged anywhere.
			s.stats.unknownKeys.Add(1)
			return
		}
		s.stats.keyEvents.Add(1)
		if msg.Type == clientKeyDown {
			// Press is idempotent while held, so browser key autorepeat
			// (keydown fires ~30x/sec on a held key) cannot disturb the latch.
			s.KeyLatch.Press(code)
		} else {
			s.KeyLatch.Release(code)
		}
	case clientBlur:
		// A key held when the tab loses focus never receives its keyup.
		s.KeyLatch.ReleaseAll()
	}
}

// writeLoop is the sole writer of the connection. gorilla/websocket permits
// only one concurrent writer, and funnelling frames, control messages and pings
// through one goroutine enforces that by construction.
func (s *session) writeLoop(ctx context.Context) error {
	frames := time.NewTicker(frameInterval)
	defer frames.Stop()
	pings := time.NewTicker(pingPeriod)
	defer pings.Stop()

	var lastGen uint64
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case msg := <-s.control:
			if err := s.writeJSON(msg); err != nil {
				return err
			}

		case <-frames.C:
			frame, gen := s.snapshot()
			if gen == lastGen {
				// Nothing changed since the last send. Skipping is free
				// bandwidth; the client already has this exact image.
				s.stats.framesSkipped.Add(1)
				continue
			}
			lastGen = gen
			if err := s.writeFrame(&frame); err != nil {
				return err
			}

		case <-pings.C:
			deadline := time.Now().Add(writeWait)
			if err := s.conn.WriteControl(websocket.PingMessage, nil, deadline); err != nil {
				return err
			}
		}
	}
}

func (s *session) writeFrame(frame *vm.FrameBuffer) error {
	packed := frame.Pack()
	if err := s.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}
	if err := s.conn.WriteMessage(websocket.BinaryMessage, packed); err != nil {
		return err
	}
	s.stats.framesSent.Add(1)
	s.stats.bytesSent.Add(uint64(len(packed)))
	return nil
}

func (s *session) writeJSON(msg controlMessage) error {
	if err := s.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}
	if err := s.conn.WriteJSON(msg); err != nil {
		return err
	}
	s.stats.controlSent.Add(1)
	return nil
}

func (s *session) snapshot() (vm.FrameBuffer, uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.frame, s.frameGen
}

// sendControl queues a control message without blocking the caller, which may
// be the VM goroutine.
//
// Control messages are must-deliver: silently dropping a sound.stop leaves the
// buzzer running on the client forever. So a full queue is treated as a dead
// client and fails the session rather than being discarded.
func (s *session) sendControl(msg controlMessage) {
	select {
	case s.control <- msg:
	default:
		s.stats.controlDropped.Add(1)
		s.fail(errControlBacklog)
	}
}

// --- vm.DisplayProvider ---

// Draw implements [vm.DisplayProvider]. It must not block: see the session doc
// comment.
func (s *session) Draw(fb vm.FrameBuffer) {
	s.mu.Lock()
	s.frame = fb
	s.frameGen++
	s.mu.Unlock()
}

// Clear implements [vm.DisplayProvider].
func (s *session) Clear() {
	s.mu.Lock()
	s.frame = vm.FrameBuffer{}
	s.frameGen++
	s.mu.Unlock()
}

// --- vm.SoundProvider ---

// PlaySound implements [vm.SoundProvider]. The VM guarantees this is an edge,
// so no deduplication is needed here.
func (s *session) PlaySound() { s.sendControl(controlMessage{Type: msgSoundPlay}) }

// StopSound implements [vm.SoundProvider].
func (s *session) StopSound() { s.sendControl(controlMessage{Type: msgSoundStop}) }

// Stats reports everything measured about this session.
func (s *session) Stats() SessionStats {
	vmStats := s.machine.Stats()
	counts := make(map[string]uint64, vm.NumOpcodeTypes)
	for i, n := range vmStats.OpcodeCounts {
		if n > 0 {
			counts[vm.OpcodeType(i).String()] = n
		}
	}
	return SessionStats{
		ROM:            s.rom.Name,
		Uptime:         vmStats.Uptime.String(),
		Instructions:   vmStats.Instructions,
		ConfiguredHz:   vmStats.ConfiguredHz,
		EffectiveHz:    round1(vmStats.EffectiveHz),
		Draws:          vmStats.Draws,
		DrawsPerSec:    round1(vmStats.DrawsPerSec),
		DrawAvg:        vmStats.DrawAvg.String(),
		DrawMax:        vmStats.DrawMax.String(),
		OpcodeCounts:   counts,
		Keyboard:       s.KeyLatch.Stats(),
		FramesSent:     s.stats.framesSent.Load(),
		FramesSkipped:  s.stats.framesSkipped.Load(),
		BytesSent:      s.stats.bytesSent.Load(),
		ControlSent:    s.stats.controlSent.Load(),
		ControlDropped: s.stats.controlDropped.Load(),
		KeyEvents:      s.stats.keyEvents.Load(),
		UnknownKeys:    s.stats.unknownKeys.Load(),
	}
}

func logSessionEnd(rom string, err error) {
	if err != nil {
		log.Printf("session %s ended: %v", rom, err)
		return
	}
	log.Printf("session %s ended cleanly", rom)
}

var _ vm.KeyboardProvider = (*session)(nil)
var _ vm.DisplayProvider = (*session)(nil)
var _ vm.SoundProvider = (*session)(nil)
