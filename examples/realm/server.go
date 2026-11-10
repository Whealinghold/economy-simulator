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
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	saveMu  sync.Mutex // orders the saves of the state file
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
	s.mux = s.routes()
	return s, nil
}

// loadState restores the state file if there is one. A file that exists and is not a
// checkpoint is an error: starting fresh over it would overwrite it at the next save.
func (s *Server) loadState() error {
	if s.state == "" {
		return nil
	}
	b, err := os.ReadFile(s.state)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read the state file: %w", err)
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
		return h
	}
	return strings.Trim(host, "[]")
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
		s.hosts[h] = true
	}
}

// ServeHTTP implements http.Handler. It refuses a write whose Origin header names
// another host (a web page driving the server from the user's browser, CSRF) before
// any route runs.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if p := recover(); p != nil { // a library panic must not drop the connection with no status
			s.log.LogAttrs(r.Context(), slog.LevelError, "handler panicked", slog.String("panic", clip(fmt.Sprint(p))))
			s.reply(w, http.StatusInternalServerError, errorBody{tickBody{s.game.Tick()}, "ErrInternal", "internal error"})
		}
	}()
	if s.hosts != nil && !s.hosts[hostname(r.Host)] {
		s.fail(w, fmt.Errorf("host %q: %w", clip(r.Host), ErrForbidden))
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		if o := r.Header.Get("Origin"); o != "" {
			if u, err := url.Parse(o); err != nil || u.Host != r.Host {
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
		s.mu.Lock()
		_, err := s.game.AdvanceContext(ctx, 1)
		s.mu.Unlock()
		s.release()
		if err != nil {
			if ctx.Err() == nil {
				s.log.LogAttrs(ctx, slog.LevelError, "paced step failed", slog.Any("error", err))
			}
			return
		}
	}
}

// read wraps a GET handler: it holds the read lock, so the tick passed in is the tick
// of everything the handler reads.
func (s *Server) read(f func(*http.Request, tickBody) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		body, err := f(r, tickBody{s.game.Tick()})
		s.mu.RUnlock() // the reply may block on a slow client: never under the lock
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
		s.mu.Lock()
		body, err := f(w, r)
		s.mu.Unlock()
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
	tick := s.game.Tick()
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

// money is a price or an amount with its text in the ladder of the preset.
type money struct {
	Value int64  `json:"value"`
	Text  string `json:"text"`
}

func (s *Server) amount(v int64) money { return money{v, s.money.Format(v)} }

type worldView struct {
	tickBody
	WorldInfo
}

func (s *Server) world(_ *http.Request, t tickBody) (any, error) {
	info, err := s.game.Info()
	if err != nil {
		return nil, err
	}
	return worldView{t, info}, nil
}

type marketView struct {
	tickBody
	Region       string `json:"region"`
	Commodity    string `json:"commodity"`
	Price        money  `json:"price"`
	Age          int64  `json:"age"`
	Volume       int64  `json:"volume"`
	UnmatchedBid int64  `json:"unmatched_bid"`
	UnmatchedAsk int64  `json:"unmatched_ask"`
	BestBid      *money `json:"best_bid,omitempty"`
	BestAsk      *money `json:"best_ask,omitempty"`
}

func (s *Server) market(r *http.Request, t tickBody) (any, error) {
	q := r.URL.Query()
	m, err := s.game.Market(q.Get("region"), q.Get("commodity"))
	if err != nil {
		return nil, err
	}
	v := marketView{
		tickBody: t, Region: q.Get("region"), Commodity: q.Get("commodity"), Price: s.amount(int64(m.Price)),
		Age: m.Age, Volume: int64(m.Volume), UnmatchedBid: int64(m.UnmatchedBid), UnmatchedAsk: int64(m.UnmatchedAsk),
	}
	if m.HasBid {
		b := s.amount(int64(m.BestBid))
		v.BestBid = &b
	}
	if m.HasAsk {
		a := s.amount(int64(m.BestAsk))
		v.BestAsk = &a
	}
	return v, nil
}

type roleView struct {
	Role       string `json:"role"`
	Agents     int64  `json:"agents"`
	Households int64  `json:"households"`
	Unfed      int64  `json:"unfed"`
}

type marketRowView struct {
	Commodity    string `json:"commodity"`
	Price        money  `json:"price"`
	Volume       int64  `json:"volume"`
	HeldByAgents int64  `json:"held_by_agents"`
}

type regionStateView struct {
	tickBody
	Region     string          `json:"region"`
	Households int64           `json:"households"`
	Agents     int64           `json:"agents"`
	Money      money           `json:"money_households"`
	Roles      []roleView      `json:"roles"`
	Markets    []marketRowView `json:"markets"`
}

func (s *Server) region(r *http.Request, t tickBody) (any, error) {
	name := r.URL.Query().Get("region")
	st, err := s.game.RegionState(name)
	if err != nil {
		return nil, err
	}
	n := s.game.names
	v := regionStateView{
		tickBody: t, Region: name, Households: st.Region.Households, Agents: st.Region.Agents,
		Money: s.amount(int64(st.Region.MoneyHouseholds)), Roles: []roleView{}, Markets: []marketRowView{},
	}
	for _, ro := range st.Roles {
		v.Roles = append(v.Roles, roleView{n.role(ro.Role), ro.Agents, ro.Households, ro.Unfed})
	}
	for _, m := range st.Markets {
		v.Markets = append(v.Markets, marketRowView{n.commodity(m.State.Commodity), s.amount(int64(m.State.Price)), int64(m.State.Volume), int64(m.HeldByAgents)})
	}
	return v, nil
}

type holdingView struct {
	Commodity string `json:"commodity"`
	Quantity  int64  `json:"quantity"`
}

type playerView struct {
	tickBody
	ID       int64         `json:"id"`
	Region   string        `json:"region"`
	Role     string        `json:"role"`
	Mortal   bool          `json:"mortal"`
	Money    money         `json:"money"`
	Hunger   int64         `json:"hunger"`
	Unfed    int64         `json:"unfed"`
	HasJob   bool          `json:"has_job"`
	Holdings []holdingView `json:"holdings"`
}

func (s *Server) getPlayer(_ *http.Request, t tickBody) (any, error) {
	a, err := s.game.Player()
	if err != nil {
		return nil, err
	}
	n := s.game.names
	v := playerView{
		tickBody: t, ID: int64(a.ID), Region: n.region(a.Region), Role: n.role(a.Role), Mortal: a.Mortal,
		Money: s.amount(int64(a.Money)), Hunger: a.Hunger, Unfed: a.Unfed, HasJob: a.HasJob, Holdings: []holdingView{},
	}
	for _, h := range a.Holdings {
		v.Holdings = append(v.Holdings, holdingView{n.commodity(h.Commodity), int64(h.Quantity)})
	}
	return v, nil
}

type addedBody struct {
	tickBody
	ID int64 `json:"id"`
}

func (s *Server) addPlayer(w http.ResponseWriter, r *http.Request) (any, error) {
	var spec PlayerSpec
	if err := decode(w, r, &spec); err != nil {
		return nil, err
	}
	id, err := s.game.AddPlayer(spec)
	if err != nil {
		return nil, err
	}
	return addedBody{tickBody{s.game.Tick()}, int64(id)}, nil
}

type acceptedBody struct {
	tickBody
	AppliesToTick int64 `json:"applies_to_tick"`
}

// direct leaves a directive for the next step, which runs at the tick the response
// carries: one tick of latency (spec-integration.md 3.6).
func (s *Server) direct(w http.ResponseWriter, r *http.Request) (any, error) {
	var spec DirectiveSpec
	if err := decode(w, r, &spec); err != nil {
		return nil, err
	}
	if err := s.game.Direct(spec); err != nil {
		return nil, err
	}
	t := s.game.Tick()
	return acceptedBody{tickBody{t}, t}, nil
}

type harvestBody struct {
	Shocks []HarvestSpec `json:"shocks"`
}

type roadBody struct {
	Roads []RoadSpec `json:"roads"`
}

type slotsView struct {
	tickBody
	Harvest []HarvestSpec `json:"harvest"`
	Roads   []RoadSpec    `json:"roads"`
}

// slots answers a shock with the slots the world holds after it.
func (s *Server) slots() (any, error) {
	info, err := s.game.Info()
	if err != nil {
		return nil, err
	}
	return slotsView{tickBody{s.game.Tick()}, info.Harvest, info.Roads}, nil
}

func (s *Server) harvest(w http.ResponseWriter, r *http.Request) (any, error) {
	var b harvestBody
	if err := decode(w, r, &b); err != nil {
		return nil, err
	}
	if err := s.game.SetHarvest(b.Shocks); err != nil {
		return nil, err
	}
	return s.slots()
}

func (s *Server) road(w http.ResponseWriter, r *http.Request) (any, error) {
	var b roadBody
	if err := decode(w, r, &b); err != nil {
		return nil, err
	}
	if err := s.game.SetRoads(b.Roads); err != nil {
		return nil, err
	}
	return s.slots()
}

type eventView struct {
	Tick      int64  `json:"tick"`
	Kind      string `json:"kind"`
	Region    string `json:"region"`
	Commodity string `json:"commodity"`
	Side      string `json:"side"`
	Price     *money `json:"price,omitempty"`
	Quantity  int64  `json:"quantity,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type eventsView struct {
	tickBody
	Events []eventView `json:"events"`
}

// events lists the player's own events from tick since (default 0) on.
func (s *Server) events(r *http.Request, t tickBody) (any, error) {
	since := int64(0)
	if q := r.URL.Query().Get("since"); q != "" {
		v, err := strconv.ParseInt(q, 10, 64)
		if err != nil || v < 0 {
			return nil, fmt.Errorf("since %q, want a tick of 0 or more: %w", clip(q), ErrBadInput)
		}
		since = v
	}
	n := s.game.names
	out := eventsView{tickBody: t, Events: []eventView{}}
	for _, e := range s.game.Events(since) {
		ev := eventView{
			Tick: e.Tick, Kind: e.Kind, Region: n.region(e.Region), Commodity: n.commodity(e.Commodity),
			Side: e.Side, Quantity: int64(e.Quantity), Reason: e.Reason,
		}
		if e.Price > 0 {
			p := s.amount(int64(e.Price))
			ev.Price = &p
		}
		out.Events = append(out.Events, ev)
	}
	return out, nil
}

type stepBody struct {
	tickBody
	Completed int64 `json:"completed"`
}

// step runs n ticks (default 1, at most maxStep) with the world locked. It stops at a
// tick boundary when the client goes away.
func (s *Server) step(w http.ResponseWriter, r *http.Request) {
	n := int64(1)
	if q := r.URL.Query().Get("n"); q != "" {
		v, err := strconv.ParseInt(q, 10, 64)
		if err != nil || v < 0 || v > s.maxStep {
			s.fail(w, fmt.Errorf("n %q, want 0 to %d: %w", clip(q), s.maxStep, ErrBadInput))
			return
		}
		n = v
	}
	if !s.acquire() {
		s.fail(w, ErrBusy)
		return
	}
	defer s.release()
	s.mu.Lock()
	done, err := s.game.AdvanceContext(r.Context(), n)
	tick := s.game.Tick()
	s.mu.Unlock()
	if err != nil {
		s.fail(w, err)
		return
	}
	s.reply(w, http.StatusOK, stepBody{tickBody{tick}, done})
}

// checkpoint answers with the encoded checkpoint and, with save=true, also writes it
// to the -state file (atomically: a temporary file in the same directory, renamed).
func (s *Server) checkpoint(w http.ResponseWriter, r *http.Request) {
	save := r.URL.Query().Get("save") == "true"
	if save && s.state == "" {
		s.fail(w, fmt.Errorf("save=true: %w", ErrNoState))
		return
	}
	if save {
		s.saveMu.Lock() // saves run one at a time, so an older checkpoint never lands after a newer one
		defer s.saveMu.Unlock()
	}
	// Encode under the lock, then write the file and the response without it, so a
	// slow reader of the response cannot hold up a step.
	s.mu.RLock()
	b, err := s.game.Checkpoint()
	tick := s.game.Tick()
	s.mu.RUnlock()
	if err != nil {
		s.fail(w, err)
		return
	}
	if save {
		if err := writeAtomic(s.state, b); err != nil {
			s.fail(w, fmt.Errorf("save the checkpoint: %w", err))
			return
		}
		s.log.LogAttrs(r.Context(), slog.LevelInfo, "checkpoint saved", slog.Int64("tick", tick), slog.Int("bytes", len(b)))
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Realm-Tick", strconv.FormatInt(tick, 10))
	_, _ = w.Write(b) // the client may be gone; nothing to do with the error
}

// writeAtomic writes b to path through a temporary file of the same directory.
func writeAtomic(path string, b []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp*")
	if err != nil {
		return err
	}
	_, werr := f.Write(b)
	serr := f.Sync() // a crash after the rename must not leave an empty state file
	cerr := f.Close()
	if err := errors.Join(werr, serr, cerr); err != nil {
		return errors.Join(err, os.Remove(f.Name()))
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return errors.Join(err, os.Remove(f.Name()))
	}
	if d, err := os.Open(filepath.Dir(path)); err == nil { // best effort: persist the rename
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

type restoredBody struct {
	tickBody
	Player bool `json:"player"`
}

// restore replaces the world by the checkpoint in the body. A body that is not a
// checkpoint is refused before anything changes (Game.Restore keeps the old world on
// any error).
func (s *Server) restore(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRestoreBody))
	if err != nil {
		s.fail(w, bodyError(err))
		return
	}
	if !s.acquire() {
		s.fail(w, ErrBusy)
		return
	}
	defer s.release()
	s.mu.Lock()
	err = s.game.Restore(b)
	_, perr := s.game.Player()
	tick := s.game.Tick()
	s.mu.Unlock()
	if err != nil {
		s.fail(w, err)
		return
	}
	s.log.LogAttrs(r.Context(), slog.LevelInfo, "state restored", slog.Int64("tick", tick), slog.Int("bytes", len(b)))
	s.reply(w, http.StatusOK, restoredBody{tickBody{tick}, perr == nil})
}
