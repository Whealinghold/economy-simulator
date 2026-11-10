package main

import (
	"cmp"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"sync"

	economysim "github.com/Whealinghold/economy-sim"
)

// ErrUnknownName is returned for a region, role or commodity name the pack does not
// define.
var ErrUnknownName = errors.New("unknown name")

// ErrBadInput is returned for a value the game refuses before it reaches the world:
// a negative amount or tick count.
var ErrBadInput = errors.New("bad input")

// ErrPlayerExists is returned by AddPlayer when the game already has its player.
var ErrPlayerExists = errors.New("the player already exists")

// headerLen is the size of the player ID that Checkpoint puts before the encoded checkpoint.
const headerLen = 8

// ErrNoPlayer is returned by Player when no player has been added.
var ErrNoPlayer = errors.New("no player")

// PlayerSpec is what AddPlayer takes: names from the pack, not typed IDs.
type PlayerSpec struct {
	// Region and Role are the region and role names of the pack.
	Region, Role string
	// Money is the starting money in minor units.
	Money int64
	// Goods maps a commodity name to a starting quantity.
	Goods map[string]int64
	// Mortal says the failure ladder may act on the player.
	Mortal bool
}

// Game owns one world behind a mutex: a world is not safe for concurrent use
// (architecture.md 12.1, 12.6), so every method locks, calls the world and unlocks
// (D685). It holds the configuration, which a restore needs, and the lookups from
// the pack's names to the typed IDs, built once from the world's configuration
// (spec-integration.md 3.4.2).
type Game struct {
	mu        sync.Mutex
	cfg       economysim.Config
	restore   []economysim.Option
	w         *economysim.World
	regions   map[string]economysim.RegionID
	roles     map[string]economysim.RoleID
	goods     map[string]economysim.CommodityID
	player    economysim.AgentID
	hasPlayer bool
}

// NewGame loads the pack (which runs the start-up solve) and builds the world with
// the given seed. extra options (an observer, the audit) are given to the world and
// again to every restore.
func NewGame(seed uint64, extra ...economysim.Option) (*Game, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}
	return NewGameFrom(cfg, seed, extra...)
}

// NewGameFrom builds a game from a configuration Load returned, so that several
// games share one solve.
func NewGameFrom(cfg economysim.Config, seed uint64, extra ...economysim.Option) (*Game, error) {
	restore := append(RestoreOptions(nil, false), extra...)
	w, err := economysim.NewWorld(cfg, append([]economysim.Option{economysim.WithSeed(seed)}, restore...)...)
	if err != nil {
		return nil, fmt.Errorf("build the world: %w", err)
	}
	g := &Game{cfg: cfg, restore: restore, w: w}
	g.index()
	return g, nil
}

// index builds the name lookups from the world's canonical configuration, whose
// slice positions are the typed IDs.
func (g *Game) index() {
	c := g.w.Config()
	g.regions = make(map[string]economysim.RegionID, len(c.Regions))
	for i, r := range c.Regions {
		g.regions[r.ID] = economysim.RegionID(i)
	}
	g.roles = make(map[string]economysim.RoleID, len(c.Roles))
	for i, r := range c.Roles {
		g.roles[r.ID] = economysim.RoleID(i)
	}
	g.goods = make(map[string]economysim.CommodityID, len(c.Commodities))
	for i, c := range c.Commodities {
		g.goods[c.ID] = economysim.CommodityID(i)
	}
}

// Tick returns the world's tick.
func (g *Game) Tick() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.w.Tick()
}

// Advance runs n ticks and returns the ticks completed. A step is CPU-bound and
// the game has no clock, so the call takes no context (architecture.md 12.3 lets
// short CPU-bound calls go without; the caller bounds n).
func (g *Game) Advance(n int64) (int64, error) {
	if n < 0 {
		return 0, fmt.Errorf("advance %d ticks: %w", n, ErrBadInput)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	done, err := g.w.Run(context.Background(), n)
	if err != nil {
		return done, fmt.Errorf("advance: %w", err)
	}
	return done, nil
}

// AddPlayer adds the player, an external agent of weight 1, before or between
// steps. A name the pack does not define is ErrUnknownName, a negative amount
// ErrBadInput, a second player ErrPlayerExists; anything else the world refuses is
// wrapped with its own sentinel.
func (g *Game) AddPlayer(p PlayerSpec) (economysim.AgentID, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.hasPlayer {
		return economysim.NoAgent, ErrPlayerExists
	}
	region, ok := g.regions[p.Region]
	if !ok {
		return economysim.NoAgent, fmt.Errorf("region %q: %w", p.Region, ErrUnknownName)
	}
	role, ok := g.roles[p.Role]
	if !ok {
		return economysim.NoAgent, fmt.Errorf("role %q: %w", p.Role, ErrUnknownName)
	}
	if p.Money < 0 {
		return economysim.NoAgent, fmt.Errorf("money %d: %w", p.Money, ErrBadInput)
	}
	stock, err := g.stock(p.Goods)
	if err != nil {
		return economysim.NoAgent, err
	}
	id, err := g.w.AddAgent(economysim.NewAgent{
		Region: region, Role: role, Money: economysim.Money(p.Money), Stock: stock, Mortal: p.Mortal,
	})
	if err != nil {
		return economysim.NoAgent, fmt.Errorf("add the player: %w", err)
	}
	g.player, g.hasPlayer = id, true
	return id, nil
}

// stock turns named quantities into the world's stock items, in the order of the
// commodity IDs (a map has none, architecture.md 7.5).
func (g *Game) stock(goods map[string]int64) ([]economysim.StockItem, error) {
	items := make([]economysim.StockItem, 0, len(goods))
	for name, q := range goods {
		c, ok := g.goods[name]
		if !ok {
			return nil, fmt.Errorf("commodity %q: %w", name, ErrUnknownName)
		}
		if q < 1 {
			return nil, fmt.Errorf("quantity %d of %q: %w", q, name, ErrBadInput)
		}
		items = append(items, economysim.StockItem{Commodity: c, Quantity: economysim.Quantity(q)})
	}
	slices.SortFunc(items, func(a, b economysim.StockItem) int { return cmp.Compare(a.Commodity, b.Commodity) })
	return items, nil
}

// Player returns the state of the player, or ErrNoPlayer.
func (g *Game) Player() (economysim.AgentState, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.hasPlayer {
		return economysim.AgentState{}, ErrNoPlayer
	}
	s, err := g.w.Agent(g.player)
	if err != nil {
		return economysim.AgentState{}, fmt.Errorf("read the player: %w", err)
	}
	return s, nil
}

// Market returns the market of a commodity in a region, by name.
func (g *Game) Market(region, commodity string) (economysim.MarketState, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	r, ok := g.regions[region]
	if !ok {
		return economysim.MarketState{}, fmt.Errorf("region %q: %w", region, ErrUnknownName)
	}
	c, ok := g.goods[commodity]
	if !ok {
		return economysim.MarketState{}, fmt.Errorf("commodity %q: %w", commodity, ErrUnknownName)
	}
	m, err := g.w.Market(r, c)
	if err != nil {
		return economysim.MarketState{}, fmt.Errorf("read the market: %w", err)
	}
	return m, nil
}

// Snapshot returns the world's snapshot.
func (g *Game) Snapshot() (economysim.Snapshot, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	s, err := g.w.Snapshot()
	if err != nil {
		return economysim.Snapshot{}, fmt.Errorf("snapshot: %w", err)
	}
	return s, nil
}

// Audit runs the world's audit: nil when the ledger, the books and the modules
// agree (the world must have been built with WithAudit for the per-step audit; this
// call works on any world).
func (g *Game) Audit() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.w.Audit()
}

// Checkpoint returns the encoded checkpoint of the world at the tick boundary, behind
// an 8-byte header with the ID of the player (NoAgent if there is none): the library
// saves the agent but not who the consumer treats as the player, and a game that
// restores the bytes in a new process must find them again.
func (g *Game) Checkpoint() ([]byte, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	c, err := g.w.Checkpoint()
	if err != nil {
		return nil, fmt.Errorf("checkpoint: %w", err)
	}
	id := economysim.NoAgent
	if g.hasPlayer {
		id = g.player
	}
	b, err := c.AppendBinary(binary.BigEndian.AppendUint64(nil, uint64(id)))
	if err != nil {
		return nil, fmt.Errorf("encode the checkpoint: %w", err)
	}
	return b, nil
}

// Restore replaces the world by the one the encoded checkpoint holds, built with
// the options the game was made with (extensions are not saved, spec-integration.md
// 8). On any error the game is unchanged. The player is the agent of the header, if
// the restored world has it as an external agent, and none otherwise.
func (g *Game) Restore(b []byte) error {
	if len(b) < headerLen {
		return fmt.Errorf("restore %d bytes, want a header of %d: %w", len(b), headerLen, ErrBadInput)
	}
	player := economysim.AgentID(int64(binary.BigEndian.Uint64(b)))
	var c economysim.Checkpoint
	if err := c.UnmarshalBinary(b[headerLen:]); err != nil {
		return fmt.Errorf("decode the checkpoint: %w", err)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	w, err := economysim.RestoreWorld(g.cfg, c, g.restore...)
	if err != nil {
		return fmt.Errorf("restore the world: %w", err)
	}
	g.w = w
	g.hasPlayer = false
	if player != economysim.NoAgent {
		s, err := w.Agent(player)
		g.player, g.hasPlayer = player, err == nil && s.External
	}
	return nil
}
