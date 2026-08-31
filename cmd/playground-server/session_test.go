package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0ya-sh0/chip-8/internal/vm"
	"github.com/gorilla/websocket"
)

// drawLoopROM draws the font glyph for 0 once, then spins:
//
//	0x200: A050  LD I, 0x050    ; font glyph "0"
//	0x202: D005  DRW V0, V0, 5
//	0x204: 1204  JP 0x204       ; spin without redrawing
//
// It must not redraw in the loop: DRW is XOR, so a redrawing loop would leave
// the screen blank half the time and make the assertion below a coin flip.
var drawLoopROM = []byte{0xA0, 0x50, 0xD0, 0x05, 0x12, 0x04}

func newTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "loop.ch8"), drawLoopROM, 0o600); err != nil {
		t.Fatalf("write rom: %v", err)
	}

	catalog, err := discoverROMs([]string{dir})
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(catalog) != 1 {
		t.Fatalf("catalog has %d roms, want 1", len(catalog))
	}

	srv := newServer(catalog, 2000, 50*time.Millisecond, vm.DefaultMaxHold)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /game", srv.handleList)
	mux.HandleFunc("GET /game/{romid}", srv.handlePlay)
	mux.HandleFunc("GET /stats", srv.handleStats)

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts, "ws" + strings.TrimPrefix(ts.URL, "http")
}

func TestSessionStreamsBinaryFrames(t *testing.T) {
	ts, wsURL := newTestServer(t)
	_ = ts

	conn, _, err := websocket.DefaultDialer.Dial(wsURL+"/game/0", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}

	// The first message is the JSON handshake carrying the geometry, so the
	// client never has to hardcode it to unpack the binary frames.
	msgType, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read init: %v", err)
	}
	if msgType != websocket.TextMessage {
		t.Fatalf("init message type = %d, want text", msgType)
	}
	var init controlMessage
	if err := json.Unmarshal(data, &init); err != nil {
		t.Fatalf("unmarshal init: %v", err)
	}
	if init.Type != msgDisplayInit || init.Width != vm.ScreenWidth || init.Height != vm.ScreenHeight {
		t.Fatalf("init = %+v, want %s %dx%d", init, msgDisplayInit, vm.ScreenWidth, vm.ScreenHeight)
	}

	// Frames arrive as binary messages of exactly one packed bitmap.
	msgType, data, err = conn.ReadMessage()
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	if msgType != websocket.BinaryMessage {
		t.Fatalf("frame message type = %d, want binary", msgType)
	}
	if len(data) != vm.PackedFrameSize {
		t.Fatalf("frame is %d bytes, want %d", len(data), vm.PackedFrameSize)
	}
	if allZero(data) {
		t.Error("frame is blank; the ROM should have drawn a font glyph")
	}
}

func TestSessionAcceptsKeyEventsAndRejectsUnknownLabels(t *testing.T) {
	ts, wsURL := newTestServer(t)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL+"/game/0", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	for _, msg := range []clientMessage{
		{Type: clientKeyDown, Data: "A"},
		{Type: clientKeyDown, Data: "A"}, // autorepeat
		{Type: clientKeyUp, Data: "A"},
		{Type: clientKeyDown, Data: "nonsense"},
	} {
		if err := conn.WriteJSON(msg); err != nil {
			t.Fatalf("write %v: %v", msg, err)
		}
	}

	stats := waitForStats(t, ts.URL, func(s SessionStats) bool {
		return s.KeyEvents >= 3 && s.UnknownKeys >= 1
	})

	// The three valid events are counted, and the two key-downs collapse into a
	// single press because the latch ignores autorepeat.
	if stats.Keyboard.Presses != 1 {
		t.Errorf("Keyboard.Presses = %d, want 1", stats.Keyboard.Presses)
	}
	// An unmapped label must be counted and discarded, never defaulted to key 0.
	if stats.UnknownKeys != 1 {
		t.Errorf("UnknownKeys = %d, want 1", stats.UnknownKeys)
	}
}

// Every session starts four goroutines. If any of them outlive the connection,
// each page reload leaks a running VM - which is how a dev session ends up with
// a dozen interpreters competing for the CPU.
func TestSessionLeavesNoGoroutinesBehind(t *testing.T) {
	_, wsURL := newTestServer(t)

	// One connect/disconnect first, so lazily-initialised runtime goroutines
	// are not counted as leaks.
	openAndClose(t, wsURL)
	waitForGoroutines(t, runtime.NumGoroutine(), 2*time.Second)
	baseline := runtime.NumGoroutine()

	for i := 0; i < 10; i++ {
		openAndClose(t, wsURL)
	}

	if !waitForGoroutines(t, baseline, 5*time.Second) {
		buf := make([]byte, 1<<16)
		n := runtime.Stack(buf, true)
		t.Fatalf("goroutines did not return to baseline %d (now %d) after 10 sessions\n%s",
			baseline, runtime.NumGoroutine(), buf[:n])
	}
}

func openAndClose(t *testing.T, wsURL string) {
	t.Helper()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL+"/game/0", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	// Read the init message and one frame, so the session is genuinely running
	// before it is torn down.
	for i := 0; i < 2; i++ {
		if _, _, err := conn.ReadMessage(); err != nil {
			t.Fatalf("read: %v", err)
		}
	}
	conn.Close()
}

func waitForGoroutines(t *testing.T, target int, timeout time.Duration) bool {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		// Allowance for httptest's own per-connection bookkeeping settling.
		if runtime.NumGoroutine() <= target+2 {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

func waitForStats(t *testing.T, baseURL string, ready func(SessionStats) bool) SessionStats {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("timed out waiting for stats")
		default:
		}

		resp, err := http.Get(baseURL + "/stats")
		if err != nil {
			t.Fatalf("get stats: %v", err)
		}
		var payload struct {
			Sessions []SessionStats `json:"sessions"`
		}
		err = json.NewDecoder(resp.Body).Decode(&payload)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("decode stats: %v", err)
		}
		if len(payload.Sessions) > 0 && ready(payload.Sessions[0]) {
			return payload.Sessions[0]
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func allZero(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}
