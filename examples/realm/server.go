package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/denomination"
)

// ErrBodyTooLarge is a request body past the cap of its route.
var ErrBodyTooLarge = errors.New("request body too large")

// ErrNoState is a checkpoint save asked for when the server has no -state path. The
// path is a flag of the process, never a field of a request.
var ErrNoState = errors.New("no state path")

// ErrForbidden is a state-changing request that a web page of another origin sent.
var ErrForbidden = errors.New("cross-origin request")

// ErrSaveFailed is a checkpoint that could not be written to the -state file: a fault
// of the server's disk, not of the request.
var ErrSaveFailed = errors.New("cannot write the state file")

// ErrBusy is a step or restore asked for while another runs: steps do not queue, because
// each holds the world for up to maxStepTicks ticks.
var ErrBusy = errors.New("a step or restore is already running")

const (
	// maxStepTicks is the most ticks one POST /v1/step may run: a step is about 60 ms,
	// and the world is locked for all of them.
	maxStepTicks = 1000
	// maxJSONBody caps the JSON body of a request.
	maxJSONBody = 64 << 10
	// maxRestoreBody caps the body of POST /v1/restore: the checkpoint of the realm is
	// about 2 MB at tick 50 and grows slowly with the run.
	maxRestoreBody = 32 << 20
	// maxMessage clips the text of an error, which echoes the name the client sent.
	maxMessage = 256
)

// refusals maps the sentinels the routes return to the status and the name the
// response carries (the name is what a client matches on). It is read-only. A refusal
// of a library call that is not listed is 422 ErrRefused; a failed or busy world is
// 500, because bad input never causes it.
var refusals = []struct {
	err    error
	name   string
	status int
}{
	{ErrUnknownName, "ErrUnknownName", http.StatusNotFound},
	{ErrNoPlayer, "ErrNoPlayer", http.StatusNotFound},
	{ErrPlayerExists, "ErrPlayerExists", http.StatusConflict},
	{ErrBadInput, "ErrBadInput", http.StatusBadRequest},
	{ErrNoState, "ErrNoState", http.StatusBadRequest},
	{ErrBodyTooLarge, "ErrBodyTooLarge", http.StatusRequestEntityTooLarge},
	{ErrForbidden, "ErrForbidden", http.StatusForbidden},
	{ErrBusy, "ErrBusy", http.StatusTooManyRequests},
	{economysim.ErrCheckpointCorrupt, "ErrCheckpointCorrupt", http.StatusBadRequest},
	{economysim.ErrCheckpointVersion, "ErrCheckpointVersion", http.StatusBadRequest},
	{economysim.ErrCheckpointConfig, "ErrCheckpointConfig", http.StatusBadRequest},
	{economysim.ErrCheckpointExtension, "ErrCheckpointExtension", http.StatusBadRequest},
	{economysim.ErrOutOfBounds, "ErrOutOfBounds", http.StatusBadRequest},
	{economysim.ErrOverflow, "ErrOverflow", http.StatusBadRequest},
	{economysim.ErrOutsideNode, "ErrOutsideNode", http.StatusBadRequest},
	{economysim.ErrUnknownEdge, "ErrUnknownEdge", http.StatusBadRequest},
	{economysim.ErrNotTradable, "ErrNotTradable", http.StatusBadRequest},
	{economysim.ErrNotOnRole, "ErrNotOnRole", http.StatusBadRequest},
	{economysim.ErrNotDirected, "ErrNotDirected", http.StatusBadRequest},
	{economysim.ErrDuplicateID, "ErrDuplicateID", http.StatusBadRequest},
	{ErrSaveFailed, "ErrSaveFailed", http.StatusInternalServerError},
	{economysim.ErrWorldFailed, "ErrWorldFailed", http.StatusInternalServerError},
	{economysim.ErrInStep, "ErrInStep", http.StatusInternalServerError},
	{context.Canceled, "ErrCancelled", http.StatusServiceUnavailable},
	{context.DeadlineExceeded, "ErrCancelled", http.StatusServiceUnavailable},
}

// tickBody is the field every response carries.
type tickBody struct {
	Tick int64 `json:"tick"`
}

type errorBody struct {
	tickBody
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Server is the HTTP face of one Game. A world is not safe for concurrent use
// (architecture.md 12.1): the Game serialises every call, and the server's own
// read-write lock keeps a response consistent, so that the tick of a response is the
// tick of the data in it (reads share the lock, a step, a restore, a shock and an
// added player take it alone).
type Server struct {
	mu      sync.RWMutex
	game    *Game
	money   *denomination.Formatter
	state   string
	log     *slog.Logger
	maxStep int64
	mux     *http.ServeMux
	stepper chan struct{} // one token: the step or restore in progress
	hosts   map[string]bool
	saveMu  sync.Mutex   // orders the saves of the state file
	tick    atomic.Int64 // last tick seen under mu: error replies read it and never wait on the game
}

var _ http.Handler = (*Server)(nil)

// NewServer loads the pack, builds the world for the seed and, when state names a
// file that exists, restores the checkpoint in it. state is also where a checkpoint
// is saved on request; "" means neither. A nil logger discards.
func NewServer(seed uint64, state string, log *slog.Logger) (*Server, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}
	return newServer(cfg, seed, state, log)
}

// newServer is NewServer over a configuration Load returned, so that tests share one
// solve.
func newServer(cfg economysim.Config, seed uint64, state string, log *slog.Logger) (*Server, error) {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	f, err := denomination.NewFormatter(denomination.RPG())
	if err != nil {
		return nil, fmt.Errorf("build the price formatter: %w", err)
	}
	ring := &eventRing{}
	g, err := NewGameFrom(cfg, seed, economysim.WithObserver(ring))
	if err != nil {
		return nil, err
	}
	g.ring = ring
	s := &Server{game: g, money: f, state: state, log: log, maxStep: maxStepTicks, stepper: make(chan struct{}, 1)}
	if err := s.loadState(); err != nil {
		return nil, err
	}
	s.tick.Store(g.Tick())
	s.mux = s.routes()
	return s, nil
}

// loadState restores the state file if there is one. A file that exists and is not a
// checkpoint is an error: starting fresh over it would overwrite it at the next save.
func (s *Server) loadState() error {
	if s.state == "" {
		return nil
	}
	f, err := os.Open(s.state)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read the state file: %w", err)
	}
	defer f.Close()
	// The same cap as POST /v1/restore: a wrong -state path must not fill the memory.
	b, err := io.ReadAll(io.LimitReader(f, maxRestoreBody+1))
	if err != nil {
		return fmt.Errorf("read the state file: %w", err)
	}
	if len(b) > maxRestoreBody {
		return fmt.Errorf("the state file is over %d bytes: %w", maxRestoreBody, ErrBodyTooLarge)
	}
	if err := s.game.Restore(b); err != nil {
		return fmt.Errorf("restore the state file: %w", err)
	}
	s.log.LogAttrs(context.Background(), slog.LevelInfo, "state restored", slog.Int64("tick", s.game.Tick()), slog.Int("bytes", len(b)))
	return nil
}

// routes builds the router. Method and path patterns do the dispatch; a path no
// route has is answered in JSON like every other refusal.
func (s *Server) routes() *http.ServeMux {
	m := http.NewServeMux()
	m.HandleFunc("GET /v1/world", s.read(s.world))
	m.HandleFunc("GET /v1/market", s.read(s.market))
	m.HandleFunc("GET /v1/region", s.read(s.region))
	m.HandleFunc("GET /v1/player", s.read(s.getPlayer))
	m.HandleFunc("GET /v1/events", s.read(s.events))
	m.HandleFunc("POST /v1/checkpoint", s.checkpoint)
	m.HandleFunc("POST /v1/player", s.write(http.StatusCreated, s.addPlayer))
	m.HandleFunc("POST /v1/player/directive", s.write(http.StatusOK, s.direct))
	m.HandleFunc("POST /v1/shock/harvest", s.write(http.StatusOK, s.harvest))
	m.HandleFunc("POST /v1/shock/road", s.write(http.StatusOK, s.road))
	m.HandleFunc("POST /v1/step", s.step)
	m.HandleFunc("POST /v1/restore", s.restore)
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.fail(w, fmt.Errorf("no route %s %s: %w", r.Method, clip(r.URL.Path), ErrUnknownName))
	})
	return m
}

// hostname is host without its port.
func hostname(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		return strings.ToLower(h)
	}
	return strings.ToLower(strings.Trim(host, "[]"))
}

// acquire takes the token of the one long operation (a step or a restore) without
// waiting; release gives it back.
func (s *Server) acquire() bool {
	select {
	case s.stepper <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *Server) release() { <-s.stepper }

// AllowHosts limits the Host header of every request to hosts (host names without a
// port), the defence against DNS rebinding. Without a call every Host is accepted.
func (s *Server) AllowHosts(hosts ...string) {
	s.hosts = map[string]bool{}
	for _, h := range hosts {
		s.hosts[strings.ToLower(h)] = true // hostname lowercases the request side
	}
}

// ServeHTTP implements http.Handler. It refuses a write whose Origin header names
// another host (a web page driving the server from the user's browser, CSRF) before
// any route runs.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if p := recover(); p != nil {
			if p == http.ErrAbortHandler {
				panic(p) // net/http's own signal to drop the connection: not ours to answer
			}
			// a library panic must not drop the connection with no status
			s.log.LogAttrs(r.Context(), slog.LevelError, "handler panicked", slog.String("panic", clip(fmt.Sprint(p))))
			s.reply(w, http.StatusInternalServerError, errorBody{tickBody{s.tick.Load()}, "ErrInternal", "internal error"})
		}
	}()
	if s.hosts != nil && !s.hosts[hostname(r.Host)] {
		s.fail(w, fmt.Errorf("host %q: %w", clip(r.Host), ErrForbidden))
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		if o := r.Header.Get("Origin"); o != "" {
			if u, err := url.Parse(o); err != nil || !strings.EqualFold(u.Host, r.Host) {
				s.fail(w, fmt.Errorf("origin %q: %w", clip(o), ErrForbidden))
				return
			}
		}
	}
	s.mux.ServeHTTP(w, r)
}

// Pace steps the world once for every value that arrives on ticks, until ctx is done
// or a step fails. The caller owns the clock (main passes a time.Ticker's channel;
// a test passes its own), so the server reads no wall-clock time.
func (s *Server) Pace(ctx context.Context, ticks <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
		}
		if !s.acquire() {
			continue // a step or restore runs: skip this tick of the clock
		}
		_, err := s.paced(ctx)
		s.release()
		if err != nil {
			if ctx.Err() == nil {
				s.log.LogAttrs(ctx, slog.LevelError, "paced step failed", slog.Any("error", err))
			}
			return
		}
	}
}

// paced is one paced step; a library panic is an error here, not a crash of the process
// (handlers get the same from ServeHTTP).
func (s *Server) paced(ctx context.Context) (n int64, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("paced step panicked: %s", clip(fmt.Sprint(p)))
		}
	}()
	return s.advance(ctx, 1)
}

// advance runs n ticks under the write lock, released even if the library panics, and
// records the tick for error replies.
func (s *Server) advance(ctx context.Context, n int64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer func() { s.tick.Store(s.game.Tick()) }()
	return s.game.AdvanceContext(ctx, n)
}

// read wraps a GET handler: it holds the read lock, so the tick passed in is the tick
// of everything the handler reads.
func (s *Server) read(f func(*http.Request, tickBody) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := func() (any, error) { // deferred unlock: a panic must not leave the lock held
			s.mu.RLock()
			defer s.mu.RUnlock()
			return f(r, tickBody{s.game.Tick()})
		}() // the reply may block on a slow client: never under the lock
		if err != nil {
			s.fail(w, err)
			return
		}
		s.reply(w, http.StatusOK, body)
	}
}

// write wraps a POST handler with a JSON body that changes the world: it holds the
// write lock.
func (s *Server) write(status int, f func(http.ResponseWriter, *http.Request) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read the body before taking the lock: a client that sends it slowly must
		// not hold the world (and every read) for the length of the read timeout.
		b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxJSONBody))
		if err != nil {
			s.fail(w, bodyError(err))
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(b))
		body, err := func() (any, error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			return f(w, r)
		}()
		if err != nil {
			s.fail(w, err)
			return
		}
		s.reply(w, status, body)
	}
}

// reply writes v as JSON. The body is encoded before the header goes out, so an
// encoding failure is still a 500.
func (s *Server) reply(w http.ResponseWriter, status int, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		s.log.LogAttrs(context.Background(), slog.LevelError, "encode the response", slog.Any("error", err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(b, '\n')) // the client may be gone; nothing to do with the error
}

// fail answers an error: the status and name of its sentinel, or 422 ErrRefused for a
// refusal of the library that the table does not list. Only a 5xx is logged above
// Debug; a refusal is the client's.
func (s *Server) fail(w http.ResponseWriter, err error) {
	status, name := http.StatusUnprocessableEntity, "ErrRefused"
	for _, r := range refusals {
		if errors.Is(err, r.err) {
			status, name = r.status, r.name
			break
		}
	}
	ctx := context.Background()
	tick := s.tick.Load()
	level := slog.LevelDebug
	if status >= http.StatusInternalServerError {
		level = slog.LevelError
	}
	s.log.LogAttrs(ctx, level, "request refused", slog.Int64("tick", tick), slog.String("reason", name), slog.String("error", clip(err.Error())))
	s.reply(w, status, errorBody{tickBody{tick}, name, clip(err.Error())})
}

// clip cuts text to maxMessage bytes, which an echoed name could otherwise inflate.
func clip(s string) string {
	if len(s) > maxMessage {
		return s[:maxMessage]
	}
	return s
}

// decode reads the JSON body into v: at most maxJSONBody bytes, no unknown field, no
// second value.
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxJSONBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return bodyError(err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return fmt.Errorf("data after the JSON value: %w", ErrBadInput)
	}
	return nil
}

// bodyError maps a failed read or decode of a body to the sentinel of the refusal.
func bodyError(err error) error {
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		return fmt.Errorf("body past %d bytes: %w", tooBig.Limit, ErrBodyTooLarge)
	}
	return fmt.Errorf("read the body: %v: %w", err, ErrBadInput)
}
