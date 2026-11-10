package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

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
	h, r, err := s.game.Slots()
	if err != nil {
		return nil, err
	}
	return slotsView{tickBody{s.game.Tick()}, h, r}, nil
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
	// ReadTimeout would expire the connection's background read during a long step and
	// net/http then cancels the request context; the step has no body left to time out.
	_ = http.NewResponseController(w).SetReadDeadline(time.Time{})
	done, err := s.advance(r.Context(), n)
	tick := s.tick.Load()
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
	b, tick, err := func() ([]byte, int64, error) {
		s.mu.RLock()
		defer s.mu.RUnlock()
		b, err := s.game.Checkpoint()
		return b, s.game.Tick(), err
	}()
	if err != nil {
		s.fail(w, err)
		return
	}
	if save {
		if err := writeAtomic(s.state, b); err != nil {
			s.log.LogAttrs(r.Context(), slog.LevelError, "checkpoint not saved", slog.Int64("tick", tick), slog.Any("error", err))
			s.fail(w, fmt.Errorf("save the checkpoint: %w", ErrSaveFailed)) // the path stays in the log
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
	var merr error
	if fi, err := os.Stat(path); err == nil { // keep the mode an operator gave the file (CreateTemp makes 0600)
		merr = f.Chmod(fi.Mode().Perm())
	}
	_, werr := f.Write(b)
	serr := f.Sync() // a crash after the rename must not leave an empty state file
	cerr := f.Close()
	if err := errors.Join(merr, werr, serr, cerr); err != nil {
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
	// The token is taken before the read, so at most one 32 MiB body is in memory.
	if !s.acquire() {
		s.fail(w, ErrBusy)
		return
	}
	defer s.release()
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRestoreBody))
	if err != nil {
		s.fail(w, bodyError(err))
		return
	}
	var perr error
	var tick int64
	func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		err = s.game.Restore(b)
		_, perr = s.game.Player()
		tick = s.game.Tick()
		s.tick.Store(tick)
	}()
	if err != nil {
		s.fail(w, err)
		return
	}
	s.log.LogAttrs(r.Context(), slog.LevelInfo, "state restored", slog.Int64("tick", tick), slog.Int("bytes", len(b)))
	s.reply(w, http.StatusOK, restoredBody{tickBody{tick}, perr == nil})
}
