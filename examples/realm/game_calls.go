package main

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file is the rest of the Game: the calls of spec-integration.md 4.3, 5 and 6
// by name (the directive, the harvest and road shocks, the reads of the world and a
// region) and the ring of the player's own events.

// maxDirectiveItems caps each list of a directive. The library bounds a directive
// too; this keeps a request small before it gets there.
const maxDirectiveItems = 64

// defaultFactorBP is the length factor of an edge that no shock changed, in basis
// points: a road request that omits the factor means it.
const defaultFactorBP = 10_000

// names holds the pack's identifiers by typed ID: the slice position is the ID.
type names struct {
	regions, roles, goods, recipes []string
}

func namesOf(c economysim.Config) names {
	var n names
	for _, r := range c.Regions {
		n.regions = append(n.regions, r.ID)
	}
	for _, r := range c.Roles {
		n.roles = append(n.roles, r.ID)
	}
	for _, c := range c.Commodities {
		n.goods = append(n.goods, c.ID)
	}
	for _, r := range c.Recipes {
		n.recipes = append(n.recipes, r.ID)
	}
	return n
}

// nameAt returns s[i], or the decimal ID when the world gave an ID the pack does not
// list, so that a response never fails on a lookup.
func nameAt(s []string, i int) string {
	if i < 0 || i >= len(s) {
		return strconv.Itoa(i)
	}
	return s[i]
}

func (n names) region(id economysim.RegionID) string       { return nameAt(n.regions, int(id)) }
func (n names) role(id economysim.RoleID) string           { return nameAt(n.roles, int(id)) }
func (n names) commodity(id economysim.CommodityID) string { return nameAt(n.goods, int(id)) }
func (n names) recipe(id economysim.RecipeID) string       { return nameAt(n.recipes, int(id)) }

// JobSpec is the job of a directive, by recipe name; NoJob asks for none.
type JobSpec struct {
	NoJob      bool   `json:"no_job"`
	Recipe     string `json:"recipe"`
	Executions int64  `json:"executions"`
}

// IntentSpec is one order of a directive: Side is "bid" or "ask", Limit the price
// per unit in minor units.
type IntentSpec struct {
	Commodity string `json:"commodity"`
	Side      string `json:"side"`
	Quantity  int64  `json:"quantity"`
	Limit     int64  `json:"limit"`
}

// DiscardSpec is a quantity of a commodity to dispose of.
type DiscardSpec struct {
	Commodity string `json:"commodity"`
	Quantity  int64  `json:"quantity"`
}

// DirectiveSpec is the directive of the player for the next step, by name.
type DirectiveSpec struct {
	Job      *JobSpec      `json:"job"`
	Intents  []IntentSpec  `json:"intents"`
	Discards []DiscardSpec `json:"discards"`
	Hold     []string      `json:"hold"`
	Sticky   bool          `json:"sticky"`
}

// HarvestSpec is the yield factor of a recipe in a region, in basis points: 10 000 is
// no shock, 0 a failed harvest.
type HarvestSpec struct {
	Region string `json:"region"`
	Recipe string `json:"recipe"`
	BP     int64  `json:"bp"`
}

// RoadSpec is the status of the road from one region to another: Closed, and the
// length factor in basis points (omitted or 0 means 10 000, no change).
type RoadSpec struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Closed   bool   `json:"closed"`
	FactorBP int64  `json:"factor_bp"`
}

// WarningInfo is a warning of the configuration the world was built from.
type WarningInfo struct {
	Rule  string   `json:"rule"`
	Path  []string `json:"path"`
	Count int64    `json:"count"`
}

// RegionInfo is one region of the world: its households and agents.
type RegionInfo struct {
	Name       string `json:"name"`
	Households int64  `json:"households"`
	Agents     int64  `json:"agents"`
	Outside    bool   `json:"outside"`
}

// WorldInfo is what GET /v1/world shows: the configuration hash and warnings, the
// regions, and the slots a shock holds (the harvest factors and the roads set).
type WorldInfo struct {
	ConfigHash string        `json:"config_hash"`
	Warnings   []WarningInfo `json:"warnings"`
	Regions    []RegionInfo  `json:"regions"`
	Harvest    []HarvestSpec `json:"harvest"`
	Roads      []RoadSpec    `json:"roads"`
}

// Direct leaves the directive in the player's mailbox for the next step
// (spec-integration.md 4.3): a name the pack does not define is ErrUnknownName, a
// list past maxDirectiveItems or a side that is neither "bid" nor "ask" ErrBadInput,
// no player ErrNoPlayer; the world's own refusals carry their sentinels.
func (g *Game) Direct(d DirectiveSpec) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.hasPlayer {
		return ErrNoPlayer
	}
	dir, err := g.directive(d)
	if err != nil {
		return err
	}
	if err := g.w.Direct(g.player, dir); err != nil {
		return fmt.Errorf("direct the player: %w", err)
	}
	return nil
}

// directive turns the named spec into the world's directive.
func (g *Game) directive(d DirectiveSpec) (economysim.Directive, error) {
	if len(d.Intents) > maxDirectiveItems || len(d.Discards) > maxDirectiveItems || len(d.Hold) > maxDirectiveItems {
		return economysim.Directive{}, fmt.Errorf("a directive list longer than %d: %w", maxDirectiveItems, ErrBadInput)
	}
	out := economysim.Directive{Sticky: d.Sticky}
	var err error
	if out.Job, err = g.job(d.Job); err != nil {
		return out, err
	}
	if out.Intents, err = g.intents(d.Intents); err != nil {
		return out, err
	}
	if out.Discards, err = g.discards(d.Discards); err != nil {
		return out, err
	}
	for _, h := range d.Hold {
		c, ok := g.goods[h]
		if !ok {
			return out, fmt.Errorf("commodity %q: %w", h, ErrUnknownName)
		}
		out.Hold = append(out.Hold, c)
	}
	return out, nil
}

// job turns the named job of a directive into the world's, or nil for none.
func (g *Game) job(j *JobSpec) (*economysim.JobRequest, error) {
	if j == nil {
		return nil, nil
	}
	recipe, ok := g.recipes[j.Recipe]
	if !ok && !j.NoJob {
		return nil, fmt.Errorf("recipe %q: %w", j.Recipe, ErrUnknownName)
	}
	return &economysim.JobRequest{NoJob: j.NoJob, Recipe: recipe, Executions: j.Executions}, nil
}

// discards turns the named discards of a directive into the world's.
func (g *Game) discards(specs []DiscardSpec) ([]economysim.Discard, error) {
	var out []economysim.Discard
	for _, dc := range specs {
		c, ok := g.goods[dc.Commodity]
		if !ok {
			return nil, fmt.Errorf("commodity %q: %w", dc.Commodity, ErrUnknownName)
		}
		out = append(out, economysim.Discard{Commodity: c, Quantity: economysim.Quantity(dc.Quantity)})
	}
	return out, nil
}

// intents turns the named orders of a directive into the world's.
func (g *Game) intents(specs []IntentSpec) ([]economysim.Intent, error) {
	var out []economysim.Intent
	for _, in := range specs {
		c, ok := g.goods[in.Commodity]
		if !ok {
			return nil, fmt.Errorf("commodity %q: %w", in.Commodity, ErrUnknownName)
		}
		side, err := parseSide(in.Side)
		if err != nil {
			return nil, err
		}
		out = append(out, economysim.Intent{
			Commodity: c, Side: side, Quantity: economysim.Quantity(in.Quantity), Limit: economysim.Price(in.Limit),
		})
	}
	return out, nil
}

func parseSide(s string) (economysim.Side, error) {
	switch s {
	case "bid":
		return economysim.Bid, nil
	case "ask":
		return economysim.Ask, nil
	}
	return economysim.Bid, fmt.Errorf("side %q, want bid or ask: %w", s, ErrBadInput)
}

// SetHarvest sets the yield factor of each (region, recipe) between steps
// (spec-integration.md 5.2); all or none. A factor of 10 000 clears the slot.
func (g *Game) SetHarvest(specs []HarvestSpec) error {
	if len(specs) > maxDirectiveItems {
		return fmt.Errorf("%d harvest shocks, at most %d: %w", len(specs), maxDirectiveItems, ErrBadInput)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	changes := make([]economysim.YieldShock, 0, len(specs))
	for _, s := range specs {
		region, ok := g.regions[s.Region]
		if !ok {
			return fmt.Errorf("region %q: %w", s.Region, ErrUnknownName)
		}
		recipe, ok := g.recipes[s.Recipe]
		if !ok {
			return fmt.Errorf("recipe %q: %w", s.Recipe, ErrUnknownName)
		}
		changes = append(changes, economysim.YieldShock{Region: region, Recipe: recipe, BP: s.BP})
	}
	if err := g.w.SetYieldShocks(changes); err != nil {
		return fmt.Errorf("set the harvest: %w", err)
	}
	return nil
}

// SetRoads sets the status of each road between steps (spec-integration.md 5.2,
// spec-regions.md 7); all or none. A road the pack does not define is refused by the
// world with ErrUnknownEdge.
func (g *Game) SetRoads(specs []RoadSpec) error {
	if len(specs) > maxDirectiveItems {
		return fmt.Errorf("%d road changes, at most %d: %w", len(specs), maxDirectiveItems, ErrBadInput)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	changes := make([]economysim.EdgeStatus, 0, len(specs))
	for _, s := range specs {
		from, ok := g.regions[s.From]
		if !ok {
			return fmt.Errorf("region %q: %w", s.From, ErrUnknownName)
		}
		to, ok := g.regions[s.To]
		if !ok {
			return fmt.Errorf("region %q: %w", s.To, ErrUnknownName)
		}
		factor := s.FactorBP
		if factor == 0 {
			factor = defaultFactorBP
		}
		changes = append(changes, economysim.EdgeStatus{From: from, To: to, Closed: s.Closed, FactorBP: factor})
	}
	if err := g.w.SetEdgeStatuses(changes); err != nil {
		return fmt.Errorf("set the roads: %w", err)
	}
	return nil
}

// Info returns the world as GET /v1/world shows it.
func (g *Game) Info() (WorldInfo, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	snap, err := g.w.Snapshot()
	if err != nil {
		return WorldInfo{}, fmt.Errorf("snapshot: %w", err)
	}
	hash := g.w.ConfigHash()
	info := WorldInfo{ConfigHash: hex.EncodeToString(hash[:]), Warnings: []WarningInfo{}, Regions: []RegionInfo{}}
	for _, w := range g.w.Warnings() {
		info.Warnings = append(info.Warnings, WarningInfo{Rule: w.Rule, Path: w.Path, Count: w.Count})
	}
	for _, r := range snap.Regions {
		info.Regions = append(info.Regions, RegionInfo{Name: g.names.region(r.Region), Households: r.Households, Agents: r.Agents, Outside: r.Outside})
	}
	if info.Harvest, info.Roads, err = g.slots(); err != nil {
		return WorldInfo{}, err
	}
	return info, nil
}

// Slots returns the harvest factors and the roads in force, by name.
func (g *Game) Slots() ([]HarvestSpec, []RoadSpec, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.slots()
}

// slots is Slots with g.mu held.
func (g *Game) slots() ([]HarvestSpec, []RoadSpec, error) {
	slots, err := g.w.Slots()
	if err != nil {
		return nil, nil, fmt.Errorf("slots: %w", err)
	}
	harvest, roads := []HarvestSpec{}, []RoadSpec{}
	for _, y := range slots.YieldShocks {
		harvest = append(harvest, HarvestSpec{Region: g.names.region(y.Region), Recipe: g.names.recipe(y.Recipe), BP: y.BP})
	}
	for _, e := range slots.EdgeStatuses {
		roads = append(roads, RoadSpec{From: g.names.region(e.From), To: g.names.region(e.To), Closed: e.Closed, FactorBP: e.FactorBP})
	}
	return harvest, roads, nil
}

// RegionState returns the state of a region by name.
func (g *Game) RegionState(region string) (economysim.RegionState, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	r, ok := g.regions[region]
	if !ok {
		return economysim.RegionState{}, fmt.Errorf("region %q: %w", region, ErrUnknownName)
	}
	s, err := g.w.Region(r)
	if err != nil {
		return economysim.RegionState{}, fmt.Errorf("read the region: %w", err)
	}
	return s, nil
}

// Events returns the player's own events of tick since and later, oldest first.
func (g *Game) Events(since int64) []PlayerEvent { return g.ring.since(since) }

// ringCap is the most events the ring keeps; the oldest are dropped.
const ringCap = 1024

// PlayerEvent is an order, a rejection or a trade of the player.
type PlayerEvent struct {
	Tick      int64
	Kind      string // "order", "rejected" or "trade"
	Region    economysim.RegionID
	Commodity economysim.CommodityID
	Side      string // "bid" or "ask": the player's side
	Price     economysim.Price
	Quantity  economysim.Quantity
	Reason    string
}

// eventRing is the Observer of the game: it keeps the player's own events in a
// bounded ring. The world calls it on the stepping goroutine, so it takes its own
// lock and never calls back into the world (architecture.md 12.5). A nil ring does
// nothing, so a Game built without one still works.
type eventRing struct {
	mu       sync.Mutex
	player   economysim.AgentID
	watching bool
	events   []PlayerEvent
}

var _ economysim.Observer = (*eventRing)(nil)

func (r *eventRing) watch(id economysim.AgentID, on bool) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.player, r.watching = id, on
	r.mu.Unlock()
}

func (r *eventRing) reset() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.events = nil
	r.mu.Unlock()
}

func (r *eventRing) since(tick int64) []PlayerEvent {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []PlayerEvent{}
	for _, e := range r.events {
		if e.Tick >= tick {
			out = append(out, e)
		}
	}
	return out
}

// Observe records the events that name the player and ignores every other.
func (r *eventRing) Observe(e economysim.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.watching {
		return
	}
	switch e := e.(type) {
	case economysim.OrderPlaced:
		if e.Agent == r.player {
			r.add(PlayerEvent{Tick: e.Tick, Kind: "order", Region: e.Region, Commodity: e.Commodity, Side: e.Side.String(), Price: e.Limit, Quantity: e.Quantity})
		}
	case economysim.OrderRejected:
		if e.Agent == r.player {
			r.add(PlayerEvent{Tick: e.Tick, Kind: "rejected", Region: e.Region, Commodity: e.Commodity, Side: e.Side.String(), Reason: string(e.Reason)})
		}
	case economysim.TradeExecuted:
		t := PlayerEvent{Tick: e.Tick, Kind: "trade", Region: e.Region, Commodity: e.Commodity, Price: e.Price, Quantity: e.Quantity}
		if e.Buyer == r.player {
			t.Side = economysim.Bid.String()
			r.add(t)
		}
		if e.Seller == r.player {
			t.Side = economysim.Ask.String()
			r.add(t)
		}
	}
}

// add appends under the lock and drops the oldest event past ringCap.
func (r *eventRing) add(e PlayerEvent) {
	if len(r.events) == ringCap {
		copy(r.events, r.events[1:])
		r.events = r.events[:ringCap-1]
	}
	r.events = append(r.events, e)
}
