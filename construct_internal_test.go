package economysim

import (
	"errors"
	"math/bits"
	"math/rand/v2"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/ordering"
)

// TestGroupWeights verifies the pure derivation of the agents and weights of a
// group: ceil(H / S) agents, raised to min(K, H), weights that sum to H and
// differ by at most 1, the first H mod a agents taking the extra household
// (spec-scale.md 3.2). The expected weights are worked out in each row.
//
// Requirement: FR-SCL-001, FR-SCL-002, FR-SCL-003, FR-SCL-004, FR-SCL-005
func TestGroupWeights(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		households, scale int64
		minAgents         int64
		want              []int64
	}{
		// H 1, S 1: one agent of weight 1 (the floor min(K, H) is 1).
		{"one household", 1, 1, 10, []int64{1}},
		// H 10, S 4: ceil(10 / 4) = 3 agents; 10 = 3 * 3 + 1, so the first agent takes
		// the extra household: 4, 3, 3.
		{"ten over scale four", 10, 4, 1, []int64{4, 3, 3}},
		// H 2 below K 3: ceil(2 / 100) = 1, raised to min(3, 2) = 2 agents of 1.
		{"fewer households than the floor", 2, 100, 3, []int64{1, 1}},
		// H 40, S 40, K 1: one agent holding all forty households.
		{"one agent holds the group", 40, 40, 1, []int64{40}},
		// H 7, S 100, K 3: ceil(7 / 100) = 1, raised to min(3, 7) = 3 agents; 7 = 3 * 2 + 1:
		// 3, 2, 2.
		{"floor raises the count", 7, 100, 3, []int64{3, 2, 2}},
		// H 100, S 1, K 10: ceil(100 / 1) = 100 agents of weight 1 (the floor is below).
		{"scale one gives one household each", 100, 1, 10, slices.Repeat([]int64{1}, 100)},
		// H 10, S 3, K 1: ceil(10 / 3) = 4 agents; 10 = 4 * 2 + 2: 3, 3, 2, 2.
		{"two extras", 10, 3, 1, []int64{3, 3, 2, 2}},
		// No household or no scale is a group validation rejects: no agents.
		{"no household", 0, 1, 1, nil},
		{"no scale", 5, 0, 1, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := groupWeights(tt.households, tt.scale, tt.minAgents)
			if !slices.Equal(got, tt.want) {
				t.Errorf("groupWeights(%d, %d, %d) = %v, want %v", tt.households, tt.scale, tt.minAgents, got, tt.want)
			}
			if tt.want == nil {
				return
			}
			var sum int64
			for _, w := range got {
				sum += w
			}
			if lo, hi := slices.Min(got), slices.Max(got); sum != tt.households || hi-lo > 1 || lo < 1 {
				t.Errorf("groupWeights(%d, %d, %d) = %v: sum %d, spread %d, min %d, want sum %d, spread at most 1, min at least 1",
					tt.households, tt.scale, tt.minAgents, got, sum, hi-lo, lo, tt.households)
			}
		})
	}
}

// TestFinishConstruction verifies the mapping of the audit of construction
// (step 9): no failure is no error, and a failure of the ledger's audit is an
// error that wraps ErrAuditFailed and the ledger's error. The mapping is a pure
// function because the real audit cannot be made to fail from the root: every
// ledger operation conserves and Validate rejects the rest (D190).
//
// Requirement: FR-WLD-036
func TestFinishConstruction(t *testing.T) {
	t.Parallel()
	if err := finishConstruction(nil); err != nil {
		t.Errorf("finishConstruction(nil) = %v, want nil", err)
	}
	synthetic := &ledger.AuditError{Account: 3, Commodity: ledger.Money, Claim: -1}
	err := finishConstruction(synthetic)
	var got *ledger.AuditError
	if !errors.Is(err, ErrAuditFailed) || !errors.As(err, &got) || got != synthetic {
		t.Errorf("finishConstruction(audit error) = %v, want an error wrapping ErrAuditFailed and the audit error", err)
	}
}

// mulHigh is the high word of w * n, the world's bounded draw (spec-world.md
// 7.2.1), written out here without the function under test.
func mulHigh(w, n uint64) int64 {
	hi, _ := bits.Mul64(w, n)
	return int64(hi)
}

// TestNewWorld_ConstructionTakesOnlyTheCreationWords verifies that construction
// takes exactly the two creation words of each household agent and nothing else
// (FR-WLD-034, 062, 064): after NewWorld the PCG is in the state of a fresh PCG
// keyed (siteConstruction, 0, last agent) and advanced by two words, with and
// without an observer, a logger and the audit; and the target ratio and the
// residue of the first agent are the two words of its own stream, recomputed
// here from the stream key (spec-world.md 7.1.5) by the stateless mix of
// internal/ordering.
//
// Requirement: FR-WLD-034, FR-WLD-062, FR-WLD-064, FR-WLD-066, FR-WLD-076
func TestNewWorld_ConstructionTakesOnlyTheCreationWords(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		seed uint64
		opts []Option
	}{
		{"seed 0", 0, nil},
		{"seed 42", 42, nil},
		{"seed 2^64-1 with an observer and the audit", 1<<64 - 1, []Option{WithObserver(ObserverFunc(func(Event) {})), WithAudit()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := internalShell(t, append([]Option{WithSeed(tt.seed)}, tt.opts...)...)
			regions := len(w.cfg.Regions)
			last := w.agents[len(w.agents)-1].id
			hi, lo := ordering.StreamKey(tt.seed, siteConstruction, 0, int64(last))
			fresh := rand.NewPCG(hi, lo)
			fresh.Uint64()
			fresh.Uint64()
			want, err := fresh.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary(fresh) = %v", err)
			}
			got, err := w.rnd.pcg.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary(world) = %v", err)
			}
			if !slices.Equal(got, want) {
				t.Errorf("seed %d: PCG state after NewWorld = %x, want a PCG keyed (siteConstruction, 0, %d) advanced by two words, %x", tt.seed, got, last, want)
			}

			// The first agent: word 1 gives the target ratio in [lo, hi] and word 2 the
			// residue in [0, 10 000). Ratio 15 000 and spread 0 give the range [15 000,
			// 15 000]: the draw takes its word and the target is 15 000 whatever it is.
			first := &w.agents[regions]
			role := w.cfg.Roles[first.role]
			rlo := role.TargetRatio * (10_000 - role.TargetSpread) / 10_000
			rhi := role.TargetRatio * (10_000 + role.TargetSpread) / 10_000
			fh, fl := ordering.StreamKey(tt.seed, siteConstruction, 0, int64(first.id))
			p := rand.NewPCG(fh, fl)
			w1, w2 := p.Uint64(), p.Uint64()
			wantTarget, wantResidue := rlo+mulHigh(w1, uint64(rhi-rlo+1)), mulHigh(w2, 10_000)
			if first.target != wantTarget {
				t.Errorf("seed %d: first agent target %d, want %d (word %#x)", tt.seed, first.target, wantTarget, w1)
			}
			// The shell role has no recipe, so there is no residue to read: the second
			// word is taken all the same (the PCG state above) and TestDrawCreation and
			// the pack tests read the residues of a role with recipes.
			if len(first.residues) > 0 && first.residues[0] != wantResidue {
				t.Errorf("seed %d: first agent residue %d, want %d (word %#x)", tt.seed, first.residues[0], wantResidue, w2)
			}
		})
	}
}

// TestNewWorld_StateAndScratch verifies what construction leaves in the world:
// the next agent ID after the households, the counters, the bit "had an agent"
// of the (region, role) of the group, the runs at 0, the table of agents with
// the estate pseudo-agent first, and the scratch of D154 allocated once.
//
// Requirement: FR-WLD-032, FR-WLD-033, NFR-WLD-003
func TestNewWorld_StateAndScratch(t *testing.T) {
	t.Parallel()
	cfg := shellConfig()
	cfg.World.Scale, cfg.World.MinAgents = 4, 1 // ten households over scale 4: weights 4, 3, 3
	w, err := NewWorld(cfg)
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	if len(w.agents) != 4 || w.agents[0].weight != 0 {
		t.Fatalf("agent table = %d entries with weight %d at 0, want the estate pseudo-agent (weight 0) and three households", len(w.agents), w.agents[0].weight)
	}
	var weights []int64
	for _, a := range w.agents[1:] {
		weights = append(weights, a.weight)
	}
	if !slices.Equal(weights, []int64{4, 3, 3}) {
		t.Errorf("weights = %v, want [4 3 3]: 10 households over 3 agents, the first taking the extra", weights)
	}
	if w.state.nextAgent != 4 || w.live.agents != 3 || w.live.households != 10 || w.live.inRole[0] != 3 || !w.state.hadAgent[0] {
		t.Errorf("state = next %d, live %+v, hadAgent %v, want next 4, 3 agents, 10 households, 3 in the role, true",
			w.state.nextAgent, w.live, w.state.hadAgent)
	}
	for _, run := range [][]int64{w.state.unfedRun, w.state.floorRun, w.state.extinctRun, w.state.belowFloorRun, w.state.clampRun, w.state.unfundedRun, w.state.renewalCarry} {
		if len(run) != 1 || run[0] != 0 {
			t.Errorf("a run or carry = %v, want one cell at 0", run)
		}
	}
	if !w.ledger.InConstruction() {
		t.Errorf("InConstruction() = false after NewWorld, want true until the first Step (FR-WLD-035)")
	}
	if w.estates == nil || w.clr == nil || w.prod == nil || w.refused == nil || w.views == nil || w.tally == nil || len(w.runnable) != 1 {
		t.Errorf("scratch missing after construction: estates %v, clr %v, prod %v, refused %v, views %v, tally %v, runnable %d",
			w.estates != nil, w.clr != nil, w.prod != nil, w.refused != nil, w.views != nil, w.tally != nil, len(w.runnable))
	}
	// The estate account is the first account after the boundary accounts, so the
	// estate pseudo-agent 0 owns it and the first household's is the next.
	if v, _ := w.ledger.Account(w.estates.account(0)); v.Kind != ledger.KindEstate {
		t.Errorf("estate account kind = %v, want KindEstate", v.Kind)
	}
	if w.agents[1].account != w.estates.account(0)+1 {
		t.Errorf("first household account = %d, want %d, the account after the estate", w.agents[1].account, w.estates.account(0)+1)
	}
}

// TestNewWorld_SeedPriceIsTheWarmEntryElseTheReference verifies the seed price
// rule (spec-beliefs.md 8.1, 8.4, FR-BEL-024, FR-VIA-031, FR-VIA-042): a market
// with an entry in WarmStart.Prices publishes it and its agents' beliefs start
// at it; a market with none takes the commodity's reference price. Two
// commodities, a seed for the first: 1 500 against a reference of 1 000, and the
// second at its reference 2 000.
//
// Requirement: FR-BEL-024, FR-VIA-031, FR-VIA-042, FR-CLR-020
func TestNewWorld_SeedPriceIsTheWarmEntryElseTheReference(t *testing.T) {
	t.Parallel()
	cfg := shellConfig()
	salt := DefaultCommodity()
	salt.ID, salt.ReferencePrice, salt.Raw = "salt", 2_000, true
	cfg.Commodities = append(cfg.Commodities, salt) // grain 1 000, salt 2 000
	cfg.World.Upkeep = []Item{{Commodity: "grain", Quantity: 1}, {Commodity: "salt", Quantity: 1}}
	cfg.Regions[0].ReferenceBasket = 3_000 // the upkeep at the reference prices: 1 000 + 2 000
	cfg.WarmStart = &WarmStart{Prices: []SeedPrice{{Region: "home", Commodity: "grain", Price: 1_500}}}
	w, err := NewWorld(cfg)
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	if got := w.markets.reports[0].Price; got != 1_500 {
		t.Errorf("market (home, grain) price = %d, want the seed 1500", got)
	}
	if got := w.markets.reports[1].Price; got != 2_000 {
		t.Errorf("market (home, salt) price = %d, want the reference 2000", got)
	}
	a := w.agents[1]
	var centres []int64
	for _, b := range a.beliefs {
		centres = append(centres, b.Centre)
	}
	if !slices.Equal(centres, []int64{1_500, 2_000}) {
		t.Errorf("belief centres = %v, want [1500 2000]: the seed price of each market, grain then salt", centres)
	}
}

// TestConstruction_TextGuards verifies by text search the two structural
// properties of the construction that no run can show (FR-WLD-062, FR-VIA-021):
// the construction stream is keyed only in the creation draws of construct.go,
// where each key is one agent's, and no file of NewWorld's path calls the
// solver.
//
// Requirement: FR-WLD-062, FR-VIA-021, FR-VIA-047
//
// Guard: a draw at a second place would move the golden output without any
// test of one run failing, and a solve at construction would make a rule after
// tick 0 depend on the table (FR-VIA-047).
func TestConstruction_TextGuards(t *testing.T) {
	t.Parallel()
	read := func(name string) string {
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		return string(b)
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read directory: %v", err)
	}
	for _, e := range entries {
		n := e.Name()
		if !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") || n == "random.go" {
			continue
		}
		src := read(n)
		if n != "construct.go" && strings.Contains(src, "siteConstruction") {
			t.Errorf("%s names siteConstruction, want only construct.go (the creation draws) and the constant's home random.go", n)
		}
	}
	if got := strings.Count(read("construct.go"), "keyed(siteConstruction"); got != 1 {
		t.Errorf("construct.go keys siteConstruction %d times, want once, in draws", got)
	}
	for _, n := range []string{"world.go", "construct.go", "construct_warm.go", "market.go", "tick.go"} {
		if strings.Contains(read(n), "SolveSteadyState") {
			t.Errorf("%s calls SolveSteadyState, want no solver on NewWorld's path (FR-VIA-021)", n)
		}
	}
}

// invariantMessage runs f and returns the message of the invariant panic it raised,
// or "" when it did not panic.
func invariantMessage(f func()) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			msg, _ = r.(string)
		}
	}()
	f()
	return ""
}

// TestGroupPlans_AnUnknownRegionOrRoleIsAViolatedInvariant verifies that a group
// naming a region or a role the configuration does not have, which validation
// rejects, makes groupPlans panic with an invariant message when either name is
// unknown (not only when both are), and that a valid configuration yields one plan
// per group, in order.
//
// Requirement: FR-WLD-030
// Guard: a plan built from an unknown name would index the world's tables at -1.
func TestGroupPlans_AnUnknownRegionOrRoleIsAViolatedInvariant(t *testing.T) {
	t.Parallel()
	base := removalConfig().canonical()
	if plans := groupPlans(&base); len(plans) != len(base.Groups) {
		t.Fatalf("groupPlans(valid) = %d plans, want %d", len(plans), len(base.Groups))
	}
	for _, tt := range []struct {
		name   string
		mutate func(g *Group)
	}{
		{"unknown region", func(g *Group) { g.Region = "nowhere" }},
		{"unknown role", func(g *Group) { g.Role = "nobody" }},
		{"both unknown", func(g *Group) { g.Region, g.Role = "nowhere", "nobody" }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := removalConfig().canonical()
			tt.mutate(&c.Groups[0])
			if msg := invariantMessage(func() { groupPlans(&c) }); !strings.Contains(msg, "invariant") || !strings.Contains(msg, "unknown region or role") {
				t.Errorf("groupPlans(%s) panicked with %q, want an invariant panic naming the unknown region or role", tt.name, msg)
			}
		})
	}
}

// TestMustBuild verifies that a refusal of the ledger during construction is a
// violated invariant that names the step and carries the ledger's reason, and
// that no refusal does nothing.
//
// Requirement: FR-WLD-030
func TestMustBuild(t *testing.T) {
	t.Parallel()
	mustBuild(nil, "a step")
	msg := invariantMessage(func() { mustBuild(errors.New("no room"), "the opening of an account") })
	if !strings.Contains(msg, "invariant") || !strings.Contains(msg, "the ledger refused the opening of an account at construction: no room") {
		t.Errorf("mustBuild(error) panicked with %q, want an invariant panic naming the step and the reason", msg)
	}
}

// TestCreateAgent_TheIDSpaceMustBeDense verifies that creating an agent whose next
// ID is not the next slot of the agent table is a violated invariant, in either
// direction, and that the table is left as it was.
//
// Arithmetic: the removal world has 2 estates and 4 households, so the table has 6
// slots and the next ID is 6; setting it to 5 or to 7 breaks the dense space.
//
// Requirement: FR-WLD-031
func TestCreateAgent_TheIDSpaceMustBeDense(t *testing.T) {
	t.Parallel()
	for _, next := range []AgentID{5, 7} {
		w := newMill(t, removalConfig())
		b := &builder{w: w, plans: groupPlans(&w.cfg), seeds: make([]beliefSeed, 3)}
		n := len(w.agents)
		w.state.nextAgent = next
		msg := invariantMessage(func() { b.createAgent(&b.plans[0], 0, 1) })
		if !strings.Contains(msg, "the agent ID space is not dense") || len(w.agents) != n {
			t.Errorf("createAgent with next ID %d over %d slots: panic %q, agents %d, want the dense-space invariant and %d agents", next, n, msg, len(w.agents), n)
		}
	}
}

// TestDraws_EachAgentTakesItsCreationWordsFromItsOwnStream verifies that the target
// ratio and the residue of every household agent are the two words of the stream
// keyed by (construction site, tick 0, the agent's ID), whatever the order of the
// agents and whatever the other agents drew: a fresh stream of the seed, keyed for
// the agent, reproduces both.
//
// Requirement: FR-WLD-034, FR-WLD-066
// Guard: an agent drawing from the end of the previous agent's stream would change
// its target when a group before it grew, which a keyed stream exists to prevent.
func TestDraws_EachAgentTakesItsCreationWordsFromItsOwnStream(t *testing.T) {
	t.Parallel()
	const seed = 4242
	w := newMill(t, removalConfig(), WithSeed(seed))
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		a := &w.agents[i]
		role := &w.cfg.Roles[a.role]
		ref := agent{residues: make([]int64, len(a.residues))}
		s := newStream(seed)
		s.keyed(siteConstruction, 0, int64(a.id))
		drawCreation(s, &ref, role.TargetRatio, role.TargetSpread)
		if a.target != ref.target || !slices.Equal(a.residues, ref.residues) {
			t.Errorf("seed %d: agent %d target %d, residues %v, want %d and %v from its own keyed stream", seed, a.id, a.target, a.residues, ref.target, ref.residues)
		}
	}
}
