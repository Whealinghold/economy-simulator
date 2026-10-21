package economysim

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Whealinghold/economy-sim/internal/beliefs"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// World is the simulation that a Config defines: it owns all simulation state
// (the configuration it was built from, the ledger, the agents, the random
// source and the observers) and is stepped by Step and Run.
//
// A World is NOT safe for concurrent use: serialise every call, including the
// readers. Two worlds share nothing, so two worlds may be stepped on two
// goroutines. The library starts no goroutine, and an observer, a strategy or
// any other extension MUST NOT call back into the World: a call made from
// inside a step returns ErrInStep and changes nothing.
//
// The zero World is not usable; NewWorld is its constructor. A world whose
// step failed is failed: every call with an error result returns ErrWorldFailed,
// and Err returns the cause.
type World struct {
	// cfg is the effective configuration: a private canonical copy of the one
	// given, with WithTargetSpread applied. It is never written after NewWorld.
	cfg  Config
	hash [32]byte

	// obs, log and audit are the options that are not data. obs.supplied is
	// read at every site that would build an event (observability-and-logging.md
	// 4.5); with no logger given log discards.
	obs   observers
	log   *slog.Logger
	audit bool

	// ext is what the configured names resolved to, once, after Validate.
	ext extensions

	// ledger holds all money and goods. shocks is the shock slot of the yield
	// factors; the field name is read by the test seam (plan 1-12). The tables
	// below are derived from cfg and are never saved.
	ledger  *ledger.Ledger
	recipes recipeTable
	roles   roleTable
	days    dayTable
	yields  yieldTable
	shocks  yieldShocks

	// rnd is the one random source (random.go).
	rnd *stream

	// estates holds the estate pseudo-agent of every region (estates.go).
	// agents is the agent table, indexed by AgentID: the first len(Regions)
	// entries are the estate pseudo-agents, which are not households and hold
	// the zero agent (weight 0), then the households in creation order. live
	// counts what is in it. markets is the record of every market. bp is
	// beliefParams of the configuration. All but the tables derived from cfg
	// are saved state (D145) or scratch.
	estates *estates
	agents  []agent
	live    liveCounts
	markets marketTable
	bp      beliefs.Params

	// The scratch of one tick, allocated at construction and reused (D154): it
	// is never saved and never shrinks. runnable is the recipes some agent has
	// on its role's list, per region, for the cost anchors of P1.2.
	clr      *clearScratch
	receipts receiptTally
	prod     *productionScratch
	refused  *refusalTally
	welfare  welfareScratch
	reads    readBuf
	views    *viewScratch
	// tk, pr, oc and ex are the scratch of the entry and phase 1 (phase_start.go), of
	// phase 2 (phase_production.go), of phases 5 and 6 (phase_outcomes.go) and of the entry
	// step of phase 7 (phase_exit.go).
	tk       tickScratch
	pr       productionTick
	oc       outcomeTick
	ex       exitTick
	tally    *entryTally
	runnable [][]RecipeID
	// seedWage is the starting wage of each region, 0 where the warm start gives
	// none (spec-viability.md 8.2.5, 3c): the cost anchor reads it while the pooled
	// wage is unknown. Derived from cfg and never saved.
	seedWage []int64

	// state is what a checkpoint saves of the world itself (D145); the tick
	// counter is the ledger's.
	state worldState

	// lotBuf is the scratch of Snapshot's lot scan; never saved.
	lotBuf []ledger.Lot

	// failed is the error of the step that failed the world, set before
	// TickEnded is emitted. stepping is the in-step flag. Neither is state and
	// neither is saved (spec-world.md 3.5, 3.6.2).
	failed   error
	stepping bool
}

// worldState is what the world itself saves, in one place for the checkpoint
// (spec-world.md 12; D145): the effective seed, the next agent ID, the runs of
// the bounded Warns the world keeps (D156) and the spawner's state, the renewal
// carry and the bit "had an agent". The tick counter and the construction flag
// are the ledger's. The slices indexed by (region, role) are at region * roles +
// role, and every run is a diagnostic that changes no result.
type worldState struct {
	seed      uint64
	nextAgent AgentID
	// unfedRun, floorRun, extinctRun, belowFloorRun and clampRun are the runs of
	// the Warns "households unfed", "work floor binding", "role extinct", "group
	// below floor" and "welfare record clamped"; unfundedRun, by region, is that
	// of "entry unfunded". The run of "price beliefs at bound" is the market's.
	unfedRun, floorRun, extinctRun, belowFloorRun, clampRun []int64
	unfundedRun                                             []int64
	// renewalCarry and hadAgent are the spawner's state of each (region, role).
	renewalCarry []int64
	hadAgent     []bool
}

// newWorldState returns the state of a world with the seed, before any agent.
func newWorldState(seed uint64, regions, roles int) worldState {
	n := regions * roles
	return worldState{
		seed:          seed,
		unfedRun:      make([]int64, n),
		floorRun:      make([]int64, n),
		extinctRun:    make([]int64, n),
		belowFloorRun: make([]int64, n),
		clampRun:      make([]int64, n),
		unfundedRun:   make([]int64, regions),
		renewalCarry:  make([]int64, n),
		hadAgent:      make([]bool, n),
	}
}

// extensions are the extensions the configured names select: the clearing, the
// strategy of each role indexed by RoleID, and the spawner.
type extensions struct {
	clearing   Clearing
	strategies []Strategy
	spawner    Spawner
}

// The values of the reason key of the "run ended" line: the closed set of that
// one line, with no event field (observability-and-logging.md 3.3.2).
const (
	runCompleted = "completed"
	runCancelled = "cancelled"
	runFailed    = "failed"
)

// NewWorld builds a world at tick 0 from cfg and the options, or returns an
// error and no world (spec-world.md 3.2, 5.1). It applies the options in the
// order given, copies the configuration into canonical order and applies
// WithTargetSpread, validates the result in full, resolves the configured
// names of the clearing, the strategies and the spawner against the registered
// extensions, takes the configuration hash, derives the tables, builds the
// random source and opens the ledger, then (construct.go) opens the estate
// accounts, creates the household agents, takes their creation draws, audits
// with WithAudit and logs and emits what it built. The configuration the caller
// holds is not kept or changed. The only random words taken are the two creation
// words of each agent.
func NewWorld(cfg Config, opts ...Option) (*World, error) {
	w, err := newShell(cfg, opts)
	if err != nil {
		return nil, err
	}
	if err := w.construct(w.cfg.Warnings()); err != nil {
		return nil, err
	}
	return w, nil
}

// newShell is steps 1 to 6 of the construction: the options, the canonical
// copy, Validate, the name checks, the hash, the derived tables, the stream
// and the ledger with its boundary accounts. The world has no estate account
// and no agent yet; construct (construct.go) builds them. It is split from
// NewWorld so a test can stop the construction between two of its steps.
func newShell(cfg Config, opts []Option) (*World, error) {
	o, err := applyOptions(opts)
	if err != nil {
		return nil, err
	}
	c := cfg.canonical()
	if o.targetSpreadGiven {
		for i := range c.Roles {
			c.Roles[i].TargetSpread = o.targetSpread
		}
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	ext, err := resolveExtensions(&c, &o)
	if err != nil {
		return nil, err
	}
	l, err := newLedger(c)
	// Excluded from coverage (testing.md 9.5): Validate has accepted the
	// configuration, so the ledger opens its built-in accounts.
	if err != nil {
		return nil, err
	}
	return &World{
		cfg:     c,
		hash:    HashConfig(c),
		obs:     o.observers,
		log:     o.logger,
		audit:   o.audit,
		ext:     ext,
		ledger:  l,
		recipes: newRecipeTable(&c),
		roles:   newRoleTable(&c),
		days:    newDayTable(c.Calendar),
		yields:  newYieldTable(&c),
		shocks:  yieldShocks{},
		rnd:     newStream(o.seed),
		state:   newWorldState(o.seed, len(c.Regions), len(c.Roles)),
	}, nil
}

// resolveExtensions runs the name checks that need the options, after
// Validate succeeded (spec-world.md 3.2.2): the clearing, the strategy of every
// role in ID order, then the spawner. The first unknown name is returned as a
// *ValidationError with the path of the field.
func resolveExtensions(c *Config, o *options) (extensions, error) {
	var ext extensions
	var err error
	if ext.clearing, err = resolveClearing(o, c.World.Clearing, []string{"world", "clearing"}); err != nil {
		return extensions{}, err
	}
	ext.strategies = make([]Strategy, len(c.Roles))
	for i := range c.Roles {
		path := []string{"roles", c.Roles[i].ID, "strategy"}
		if ext.strategies[i], err = resolveStrategy(o, c.Roles[i].Strategy, path); err != nil {
			return extensions{}, err
		}
	}
	if ext.spawner, err = resolveSpawner(o, c.World.Spawner, []string{"world", "spawner"}); err != nil {
		return extensions{}, err
	}
	return ext, nil
}

// Err returns the error that failed the world, or nil on a healthy one. It
// works on a failed world and from inside a step.
func (w *World) Err() error { return w.failed }

// Tick returns the number of steps completed since tick 0: the tick the next
// Step will run. After a failed step it is the tick that step was running. It
// works on a failed world and from inside a step.
func (w *World) Tick() int64 { return w.ledger.Tick() }

// Config returns a deep copy of the effective configuration: the one given to
// NewWorld in canonical order, with WithTargetSpread applied. Changing the copy
// changes nothing in the world.
func (w *World) Config() Config { return w.cfg.canonical() }

// ConfigHash returns the hash of the effective configuration (HashConfig of
// Config()). The seed, the logger, the observers and the audit option are not
// in it; WithTargetSpread and the registered names are.
func (w *World) ConfigHash() [32]byte { return w.hash }

// guard is P0.1: ErrInStep if a step is running, else ErrWorldFailed wrapping
// the cause if the world failed, else nil. A running step is checked first so a
// call made from an observer of the failing step's TickEnded still gets
// ErrInStep.
func (w *World) guard() error {
	if w.stepping {
		return ErrInStep
	}
	if w.failed != nil {
		return fmt.Errorf("%w: %w", ErrWorldFailed, w.failed)
	}
	return nil
}

// Audit checks the invariants of the world: the identities of the ledger
// (spec-ledger.md 12.4). It returns ErrInStep or ErrWorldFailed under the
// guards, an error that wraps ErrAuditFailed and names the tick when an
// identity is false, and nil otherwise. It changes nothing and draws no random
// number.
func (w *World) Audit() error {
	if err := w.guard(); err != nil {
		return err
	}
	return auditFailure(w.Tick(), w.ledger.Audit())
}

// auditFailure maps the result of the ledger's audit at tick to the world's
// error: nil to nil, else an error that wraps ErrAuditFailed and the ledger's
// error and names the tick (spec-world.md 3.3.3). It is pure, so the mapping is
// tested alone (D190).
func auditFailure(tick int64, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("audit at tick %d: %w: %w", tick, ErrAuditFailed, err)
}

// Step runs one tick through the eight phases of spec-world.md 6 and, on
// success, advances the tick by one. It takes no context: it is short and
// CPU-bound.
//
// A step that returns an error leaves the world failed: every later call that
// has an error result returns ErrWorldFailed (spec-world.md 3.5). A call made
// while a step runs, by an observer for instance, returns ErrInStep. A panic
// that leaves the step is not recovered; it also fails the world, so a caller
// that recovers it gets ErrWorldFailed, and the step emits nothing then.
func (w *World) Step() error {
	finished := false
	defer func() {
		if !finished {
			w.abandonStep()
		}
	}()
	if err := w.beginStep(); err != nil {
		// A refused call: the flag and the failure belong to the step or the
		// failure that refused it, and are left as they are.
		finished = true
		return err
	}
	err := w.endStep(w.positions())
	finished = true
	return err
}

// abandonStep is the deferred half of Step when a panic is leaving it. It does
// not call recover and emits nothing; it marks the world failed, unless the
// step already failed it, and clears the in-step flag (spec-world.md 3.5.4).
func (w *World) abandonStep() {
	if w.failed == nil {
		w.failed = fmt.Errorf("step at tick %d panicked", w.Tick())
	}
	w.stepping = false
}

// beginStep is the part of a step before the positions of the straight line:
// P0.1 (the guards and the in-step flag), P0.2a (the end of the construction
// phase and the bucket's events, p02aBucket) and P0.3 (TickStarted). A replay of
// the step calls it, then the positions, then endStep.
func (w *World) beginStep() error {
	if err := w.guard(); err != nil {
		return err
	}
	w.stepping = true
	w.p02aBucket()
	if w.obs.supplied {
		w.obs.deliver(TickStarted{Tick: w.Tick()})
	}
	return nil
}

// endStep is the part of a step after the straight line: P8.5 (a non-nil err
// fails the world before TickEnded is emitted, with Failed true) and P8.6 (on
// success the tick advances, the flow totals are cleared by Ledger.EndTick and
// the welfare scratch is cleared, D158, D161). The in-step flag is cleared as its
// last act. It returns err, so Step can return it.
func (w *World) endStep(err error) error {
	if err != nil {
		w.failed = err
	}
	if w.obs.supplied {
		w.obs.deliver(TickEnded{Tick: w.Tick(), Failed: err != nil})
	}
	w.logTickEnded()
	if err == nil {
		w.ledger.EndTick()
		for slot := range w.welfare.slots {
			w.welfare.reset(slot)
		}
	}
	w.stepping = false
	return err
}

// Run steps the world ticks times and returns the number of steps that
// completed (spec-world.md 3.4). It checks ctx before every tick, the first
// included, and never inside a tick, so a cancelled Run leaves the world at a
// tick boundary and usable; the error then wraps ctx.Err and the number
// completed. A failed step ends the Run with the step's error wrapped and the
// world failed. Run emits RunStarted before its first tick and RunEnded before
// it returns, and logs "run started" and "run ended" at Info. A negative ticks
// is an error that wraps ErrOutOfBounds, and a failed or running world returns
// ErrWorldFailed or ErrInStep; none of the three emits an event or a line.
func (w *World) Run(ctx context.Context, ticks int64) (completed int64, err error) {
	if gerr := w.guard(); gerr != nil {
		return 0, gerr
	}
	if ticks < 0 {
		return 0, fmt.Errorf("ticks %d below 0: %w", ticks, ErrOutOfBounds)
	}
	w.runStarted(ctx, ticks)
	for completed < ticks {
		if cerr := ctx.Err(); cerr != nil {
			err = fmt.Errorf("run cancelled after %d ticks: %w", completed, cerr)
			break
		}
		if serr := w.Step(); serr != nil {
			err = fmt.Errorf("run failed after %d ticks: %w", completed, serr)
			break
		}
		completed++
	}
	w.runEnded(ctx, completed, err)
	return completed, err
}

// runStarted emits RunStarted and logs "run started".
func (w *World) runStarted(ctx context.Context, ticks int64) {
	if w.obs.supplied {
		w.obs.deliver(RunStarted{Tick: w.Tick(), Ticks: ticks})
	}
	if w.log.Enabled(ctx, slog.LevelInfo) {
		w.log.LogAttrs(ctx, slog.LevelInfo, "run started", slog.Int64(logkeys.Ticks, ticks))
	}
}

// runEnded emits RunEnded and logs "run ended". An error of the step failed the
// run, any other error is a cancellation: the two are never both true. It is
// not reached when a step panics.
func (w *World) runEnded(ctx context.Context, completed int64, err error) {
	failed := err != nil && w.failed != nil
	cancelled := err != nil && !failed
	if w.obs.supplied {
		w.obs.deliver(RunEnded{Tick: w.Tick(), Completed: completed, Cancelled: cancelled, Failed: failed})
	}
	if w.log.Enabled(ctx, slog.LevelInfo) {
		reason := runCompleted
		switch {
		case failed:
			reason = runFailed
		case cancelled:
			reason = runCancelled
		}
		w.log.LogAttrs(ctx, slog.LevelInfo, "run ended",
			slog.Int64(logkeys.Ticks, completed), slog.String(logkeys.Reason, reason))
	}
}
