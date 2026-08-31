// Command playground-server serves CHIP-8 ROMs to a browser client over
// WebSocket: it runs the interpreter, streams packed framebuffers at 60Hz, and
// feeds keyboard edges back into the VM.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"

	// Registers /debug/pprof handlers on http.DefaultServeMux. Blank import is
	// the documented way to enable them; keep the listener bound to localhost,
	// as these endpoints expose process memory and allow expensive dumps.
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/0ya-sh0/chip-8/internal/vm"
	"github.com/gorilla/websocket"
)

// ROM is a playable ROM in the catalog.
type ROM struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Path string `json:"-"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "playground-server:", err)
		os.Exit(1)
	}
}

// run holds the real body of main so that deferred cleanup still executes on
// the error path; calling os.Exit from main would skip every defer.
func run() error {
	var (
		addr    = flag.String("addr", "localhost:9000", "listen address")
		cpuHz   = flag.Int("cpu", vm.DefaultCPUHz, "interpreter speed in instructions per second")
		minHold = flag.Duration("min-hold", vm.DefaultMinHold, "minimum time a keypress stays observable")
		maxHold = flag.Duration("max-hold", vm.DefaultMaxHold, "maximum time an unobserved keypress stays pending; set equal to -min-hold to disable")
		dirs    = flag.String("roms", "test-roms,test-games,more-roms", "comma-separated ROM directories")
	)
	flag.Parse()

	catalog, err := discoverROMs(strings.Split(*dirs, ","))
	if err != nil {
		return err
	}
	if len(catalog) == 0 {
		return errors.New("no .ch8 ROMs found; check -roms")
	}
	log.Printf("loaded %d roms, cpu=%dHz min-hold=%s max-hold=%s", len(catalog), *cpuHz, *minHold, *maxHold)

	srv := newServer(catalog, *cpuHz, *minHold, *maxHold)

	// Cancelled on SIGINT/SIGTERM. Sessions inherit this context, so a shutdown
	// signal unwinds every VM and connection rather than leaving them running.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /game", srv.handleList)
	mux.HandleFunc("GET /game/{romid}", srv.handlePlay)
	mux.HandleFunc("GET /stats", srv.handleStats)
	// pprof registered itself on DefaultServeMux; forward to it.
	mux.Handle("/debug/", http.DefaultServeMux)

	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 5 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("listening on http://%s", *addr)
		errCh <- httpSrv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		log.Println("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpSrv.Shutdown(shutdownCtx)
	}
}

// server holds immutable configuration plus the live session registry.
type server struct {
	catalog []ROM
	cpuHz   int
	minHold time.Duration
	maxHold time.Duration

	mu       sync.Mutex
	sessions map[uint64]*session
	nextID   atomic.Uint64
}

func newServer(catalog []ROM, cpuHz int, minHold, maxHold time.Duration) *server {
	return &server{
		catalog:  catalog,
		cpuHz:    cpuHz,
		minHold:  minHold,
		maxHold:  maxHold,
		sessions: make(map[uint64]*session),
	}
}

var upgrader = websocket.Upgrader{
	// The client is served from a different origin in development. Tighten this
	// before exposing the server beyond localhost.
	CheckOrigin: func(*http.Request) bool { return true },
}

func (s *server) handleList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.catalog)
}

func (s *server) handlePlay(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("romid"))
	if err != nil || id < 0 || id >= len(s.catalog) {
		http.Error(w, "unknown rom id", http.StatusNotFound)
		return
	}
	rom := s.catalog[id]

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade has already written an error response.
		log.Printf("upgrade: %v", err)
		return
	}

	sess := newSession(conn, rom, s.cpuHz, s.minHold, s.maxHold)
	id64 := s.register(sess)
	defer s.unregister(id64)

	log.Printf("session %d started: %s", id64, rom.Name)
	// r.Context() is cancelled when the client disconnects or the server shuts
	// down, so every goroutine the session owns is reachable from here.
	logSessionEnd(rom.Name, sess.run(r.Context()))
}

func (s *server) handleStats(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	ids := make([]uint64, 0, len(s.sessions))
	for id := range s.sessions {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	out := make([]SessionStats, 0, len(ids))
	for _, id := range ids {
		st := s.sessions[id].Stats()
		st.ID = id
		out = append(out, st)
	}
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"sessions": out,
		"count":    len(out),
	})
}

func (s *server) register(sess *session) uint64 {
	id := s.nextID.Add(1)
	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()
	return id
}

func (s *server) unregister(id uint64) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

// SessionStats is the JSON shape returned by /stats.
type SessionStats struct {
	ID     uint64 `json:"id"`
	ROM    string `json:"rom"`
	Uptime string `json:"uptime"`

	// Instructions and EffectiveHz are the guardrail: if EffectiveHz sits well
	// below ConfiguredHz the interpreter is being starved, which also means the
	// keypad is sampled less often than intended.
	Instructions uint64  `json:"instructions"`
	ConfiguredHz int     `json:"configured_hz"`
	EffectiveHz  float64 `json:"effective_hz"`

	Draws        uint64            `json:"draws"`
	DrawsPerSec  float64           `json:"draws_per_sec"`
	DrawAvg      string            `json:"draw_avg"`
	DrawMax      string            `json:"draw_max"`
	OpcodeCounts map[string]uint64 `json:"opcode_counts"`

	Keyboard vm.KeyStats `json:"keyboard"`

	FramesSent     uint64 `json:"frames_sent"`
	FramesSkipped  uint64 `json:"frames_skipped"`
	BytesSent      uint64 `json:"bytes_sent"`
	ControlSent    uint64 `json:"control_sent"`
	ControlDropped uint64 `json:"control_dropped"`
	KeyEvents      uint64 `json:"key_events"`
	UnknownKeys    uint64 `json:"unknown_keys"`
}

// discoverROMs walks the given directories and returns a stable, sorted
// catalog. Missing directories are skipped; unreadable ones are an error, since
// silently serving a short list is worse than failing loudly.
func discoverROMs(dirs []string) ([]ROM, error) {
	var roms []ROM
	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("read rom dir %q: %w", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".ch8") {
				continue
			}
			info, err := entry.Info()
			if err != nil || info.Size() == 0 {
				continue
			}
			roms = append(roms, ROM{
				Name: strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())),
				Path: filepath.Join(dir, entry.Name()),
			})
		}
	}

	sort.Slice(roms, func(i, j int) bool { return roms[i].Path < roms[j].Path })
	for i := range roms {
		roms[i].ID = i
	}
	return roms, nil
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func round1(f float64) float64 { return math.Round(f*10) / 10 }
