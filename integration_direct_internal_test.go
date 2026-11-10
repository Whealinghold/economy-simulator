package economysim

import (
	"bytes"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"testing"
)

// directConfig is extConfig with a role that runs the recipe "make", a second
// recipe "other" that no role lists, and a commodity "ore" that the role does not
// trade, so that every refusal of spec-integration.md 4.3.2 has a case. The role
// trader's commodities are bread (0, the upkeep), grain (1), ore (2) and salt (3), in canonical order.
func directConfig() Config {
	c := extConfig()
	ore := DefaultCommodity()
	ore.ID, ore.ReferencePrice, ore.Raw = "ore", 30, true
	c.Commodities = append(c.Commodities, ore)
	c.Recipes = append(c.Recipes, Recipe{ID: "other", Duration: 1, TraineeYield: 5_000,
		Outputs: []Output{{Commodity: "ore", Quantity: 1, CostShare: 10_000}}})
	c.Roles[0].Recipes = []string{"make"}
	return c
}

// Commodities and recipes of directConfig by ID.
const (
	dcBread CommodityID = iota
	dcGrain
	dcOre
	dcSalt
)

const (
	rMake  RecipeID = 0
	rOther RecipeID = 1
)

// directWorld is the world of directConfig with a directed external agent and a
// scripted one ("script" is a registered strategy, so not directed) in lowland, a
// removed external agent, and the observer log. It returns the IDs: the directed
// agent, the scripted one, the removed one and a household of the configuration.
func directWorld(t *testing.T, opts ...Option) (w *World, rec *evLog, directed, scripted, removed, household AgentID) {
	t.Helper()
	w, _, rec = extWorld(t, directConfig(), opts...)
	var err error
	if directed, err = w.AddAgent(NewAgent{Region: regLowland, Role: 0, Money: 100_000}); err != nil {
		t.Fatalf("AddAgent(directed) = %v, want nil", err)
	}
	if scripted, err = w.AddAgent(NewAgent{Region: regLowland, Role: 0, Strategy: "script"}); err != nil {
		t.Fatalf("AddAgent(script) = %v, want nil", err)
	}
	if removed, err = w.AddAgent(NewAgent{Region: regLowland, Role: 0}); err != nil {
		t.Fatalf("AddAgent(removed) = %v, want nil", err)
	}
	if err = w.RemoveAgent(removed); err != nil {
		t.Fatalf("RemoveAgent() = %v, want nil", err)
	}
	return w, rec, directed, scripted, removed, AgentID(w.firstHousehold)
}

// TestDirect_Errors verifies the checks of spec-integration.md 4.3.2 in their
// order: the agent first (a pseudo-agent or an ID never issued, a removed agent,
// a household, a strategy that is not Directed), then the directive row by row,
// each with its sentinel and, for an argument, a *ValidationError whose path names
// it. A refused call leaves the mailbox, and so the checkpoint, as it was, and
// emits nothing.
//
// Requirement: FR-INT-025, FR-INT-005, FR-INT-002
func TestDirect_Errors(t *testing.T) {
	t.Parallel()
	ok := func() Directive { return Directive{} }
	with := func(f func(d *Directive)) Directive { d := ok(); f(&d); return d }
	bad := Intent{Commodity: dcGrain, Side: Bid, Quantity: 1, Limit: 10}
	intent := func(f func(in *Intent)) Directive {
		in := bad
		f(&in)
		return Directive{Intents: []Intent{in}}
	}
	argTests := []struct {
		name string
		d    Directive
		want error
		path string // the last segment of the path of the error
	}{
		{"recipe below range", with(func(d *Directive) { d.Job = &JobRequest{Recipe: RecipeID(-1)} }), ErrUnknownRecipe, "recipe"},
		{"recipe above range", with(func(d *Directive) { d.Job = &JobRequest{Recipe: 9} }), ErrUnknownRecipe, "recipe"},
		{"executions above the weight", with(func(d *Directive) { d.Job = &JobRequest{Recipe: rMake, Executions: 2} }), ErrOutOfBounds, "executions"},
		{"executions below 0", with(func(d *Directive) { d.Job = &JobRequest{Recipe: rMake, Executions: -1} }), ErrOutOfBounds, "executions"},
		{"recipe not on the role", with(func(d *Directive) { d.Job = &JobRequest{Recipe: rOther} }), ErrNotOnRole, "recipe"},
		{"intent commodity unknown", intent(func(in *Intent) { in.Commodity = 9 }), ErrUnknownCommodity, "commodity"},
		{"intent commodity below range", intent(func(in *Intent) { in.Commodity = NoCommodity }), ErrUnknownCommodity, "commodity"},
		{"intent side invalid", intent(func(in *Intent) { in.Side = 2 }), ErrOutOfBounds, "side"},
		{"intent purpose invalid", intent(func(in *Intent) { in.Purpose = 9 }), ErrOutOfBounds, "purpose"},
		{"intent quantity 0", intent(func(in *Intent) { in.Quantity = 0 }), ErrOutOfBounds, "quantity"},
		{"intent quantity above MaxQuantity", intent(func(in *Intent) { in.Quantity = MaxQuantity + 1 }), ErrOutOfBounds, "quantity"},
		{"intent limit 0", intent(func(in *Intent) { in.Limit = 0 }), ErrOutOfBounds, "limit"},
		{"intent limit above MaxPrice", intent(func(in *Intent) { in.Limit = MaxPrice + 1 }), ErrOutOfBounds, "limit"},
		{"intent target below 0", intent(func(in *Intent) { in.Target = -1 }), ErrOutOfBounds, "target"},
		{"intent reserve above MaxQuantity", intent(func(in *Intent) { in.Reserve = MaxQuantity + 1 }), ErrOutOfBounds, "reserve"},
		{"intent unit cost below 0", intent(func(in *Intent) { in.UnitCost = -1 }), ErrOutOfBounds, "unit_cost"},
		{"intent commodity twice", with(func(d *Directive) { d.Intents = []Intent{bad, bad} }), ErrDuplicateID, "commodity"},
		{"intent commodity not tradable", intent(func(in *Intent) { in.Commodity = dcOre }), ErrNotTradable, "commodity"},
		{"discard commodity unknown", with(func(d *Directive) { d.Discards = []Discard{{Commodity: 9, Quantity: 1}} }), ErrUnknownCommodity, "commodity"},
		{"discard quantity 0", with(func(d *Directive) { d.Discards = []Discard{{Commodity: dcGrain}} }), ErrOutOfBounds, "quantity"},
		{"discard commodity twice", with(func(d *Directive) {
			d.Discards = []Discard{{Commodity: dcGrain, Quantity: 1}, {Commodity: dcGrain, Quantity: 2}}
		}), ErrDuplicateID, "commodity"},
		{"hold commodity unknown", with(func(d *Directive) { d.Hold = []CommodityID{9} }), ErrUnknownCommodity, "hold[0]"},
		{"hold commodity twice", with(func(d *Directive) { d.Hold = []CommodityID{dcGrain, dcSalt, dcGrain} }), ErrDuplicateID, "hold[2]"},
		// The order of the table: the job before the intents, the intents' fields and
		// repeats before their tradability, the intents before the discards.
		{"job first", with(func(d *Directive) {
			d.Job, d.Intents = &JobRequest{Recipe: 9}, []Intent{{Commodity: 9}}
		}), ErrUnknownRecipe, "recipe"},
		{"intent fields before tradable", with(func(d *Directive) {
			d.Intents = []Intent{{Commodity: dcOre, Side: Bid, Quantity: 1, Limit: 10}, {Commodity: dcGrain, Side: Bid, Quantity: 0, Limit: 10}}
		}), ErrOutOfBounds, "quantity"},
		{"intents before discards", with(func(d *Directive) {
			d.Intents, d.Discards = []Intent{{Commodity: dcOre, Side: Bid, Quantity: 1, Limit: 10}}, []Discard{{Commodity: 9}}
		}), ErrNotTradable, "commodity"},
	}
	for _, tt := range argTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, rec, id, _, _, _ := directWorld(t)
			if err := w.Direct(id, Directive{Hold: []CommodityID{dcSalt}, Sticky: true}); err != nil {
				t.Fatalf("Direct(held) = %v, want nil", err)
			}
			before, n := bytesOf(t, w), len(rec.events)
			err := w.Direct(id, tt.d)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Direct(%+v) = %v, want %v", tt.d, err, tt.want)
			}
			var ve *ValidationError
			if !errors.As(err, &ve) || ve.Path[0] != "d" || ve.Path[len(ve.Path)-1] != tt.path {
				t.Errorf("Direct error %v is not a ValidationError with a path from d ending %q", err, tt.path)
			}
			if !bytes.Equal(bytesOf(t, w), before) || len(rec.events) != n {
				t.Errorf("a refused Direct changed the world: bytes equal %v, %d events (was %d)", bytes.Equal(bytesOf(t, w), before), len(rec.events), n)
			}
		})
	}
	// The agent before the arguments: a directive that is wrong in every row is
	// refused for the agent first.
	rubbish := Directive{Job: &JobRequest{Recipe: 9}, Intents: []Intent{{Commodity: 9}}}
	agentTests := []struct {
		name string
		pick func(directed, scripted, removed, household AgentID) AgentID
		want error
	}{
		{"never issued", func(_, _, _, _ AgentID) AgentID { return 999 }, ErrUnknownAgent},
		{"below 0", func(_, _, _, _ AgentID) AgentID { return NoAgent }, ErrUnknownAgent},
		{"an estate", func(_, _, _, _ AgentID) AgentID { return 0 }, ErrUnknownAgent},
		{"a counterparty", func(_, _, _, _ AgentID) AgentID { return 2 }, ErrUnknownAgent},
		{"removed", func(_, _, r, _ AgentID) AgentID { return r }, ErrAgentRemoved},
		{"household", func(_, _, _, h AgentID) AgentID { return h }, ErrNotExternal},
		{"not directed", func(_, s, _, _ AgentID) AgentID { return s }, ErrNotDirected},
	}
	for _, tt := range agentTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _, d, s, r, h := directWorld(t)
			before := bytesOf(t, w)
			if err := w.Direct(tt.pick(d, s, r, h), rubbish); !errors.Is(err, tt.want) {
				t.Errorf("Direct(%s, a directive wrong in every row) = %v, want %v", tt.name, err, tt.want)
			}
			if !bytes.Equal(bytesOf(t, w), before) {
				t.Error("a refused Direct changed the checkpoint")
			}
		})
	}
	t.Run("no job needs no recipe", func(t *testing.T) {
		t.Parallel()
		w, _, id, _, _, _ := directWorld(t)
		if err := w.Direct(id, Directive{Job: &JobRequest{NoJob: true, Recipe: 9, Executions: -5}}); err != nil {
			t.Errorf("Direct(a request for no job) = %v, want nil: the other fields are ignored", err)
		}
	})
	t.Run("a recipe of the role's list at the bounds", func(t *testing.T) {
		t.Parallel()
		w, _, id, _, _, _ := directWorld(t)
		for _, e := range []int64{0, 1} {
			if err := w.Direct(id, Directive{Job: &JobRequest{Recipe: rMake, Executions: e}}); err != nil {
				t.Errorf("Direct(recipe on the list, executions %d) = %v, want nil (0 to the weight 1)", e, err)
			}
		}
	})
	t.Run("the error names the call and the agent", func(t *testing.T) {
		t.Parallel()
		w, _, id, _, _, _ := directWorld(t)
		err := w.Direct(id, Directive{Hold: []CommodityID{9}})
		if err == nil || !strings.HasPrefix(err.Error(), "Direct: agent ") {
			t.Errorf("Direct error = %v, want it to begin with the call's name and the agent", err)
		}
	})
}

// TestDirect_ReplaceAndEmpty verifies that the mailbox holds one directive: a
// second call replaces the first and does not merge, a directive with nothing in
// it (a Sticky flag alone is nothing) empties the mailbox, and the mailbox is a
// copy, so a caller that changes its slices afterwards changes nothing.
//
// Requirement: FR-INT-026
func TestDirect_ReplaceAndEmpty(t *testing.T) {
	t.Parallel()
	w, _, id, _, _, _ := directWorld(t)
	first := Directive{Intents: []Intent{{Commodity: dcGrain, Side: Bid, Quantity: 2, Limit: 10}}, Hold: []CommodityID{dcSalt}}
	second := Directive{Discards: []Discard{{Commodity: dcGrain, Quantity: 1}}, Sticky: true}
	for _, d := range []Directive{first, second} {
		if err := w.Direct(id, d); err != nil {
			t.Fatalf("Direct(%+v) = %v, want nil", d, err)
		}
	}
	got := w.agents[id].directive
	if got == nil || len(got.Intents) != 0 || len(got.Hold) != 0 || !slices.Equal(got.Discards, second.Discards) || !got.Sticky {
		t.Errorf("mailbox after two calls = %+v, want the second directive alone, %+v", got, second)
	}
	second.Discards[0].Quantity = 99
	if got.Discards[0].Quantity != 1 {
		t.Errorf("a change to the caller's slice reached the mailbox: quantity %d, want 1", got.Discards[0].Quantity)
	}
	for _, empty := range []Directive{{}, {Sticky: true}, {Intents: []Intent{}, Hold: []CommodityID{}}} {
		if err := w.Direct(id, first); err != nil {
			t.Fatalf("Direct(first) = %v, want nil", err)
		}
		if err := w.Direct(id, empty); err != nil {
			t.Fatalf("Direct(%+v) = %v, want nil", empty, err)
		}
		if w.agents[id].directive != nil {
			t.Errorf("mailbox after Direct(%+v) = %+v, want empty", empty, w.agents[id].directive)
		}
	}
}

// bidsAt returns the OrderPlaced events of agent id at limit price limit, the
// mark of a directive that acted.
func bidsAt(rec *evLog, id AgentID, limit Price) []OrderPlaced {
	return slices.DeleteFunc(of[OrderPlaced](rec), func(o OrderPlaced) bool { return o.Agent != id || o.Limit != limit })
}

// TestDirect_OneTickUnlessSticky verifies that a directive acts in the next step
// and is gone after it, that a Sticky one stays until another call replaces or
// empties it, that P1.9 fixes the directive of the tick (a call after the step
// acts in the step after), and that Direct emits no event and logs one Debug line.
//
// The directive bids 3 grain at 77, a limit nothing else in the world uses.
//
// Requirement: FR-INT-027, FR-INT-052
func TestDirect_OneTickUnlessSticky(t *testing.T) {
	t.Parallel()
	const limit Price = 77
	d := func(sticky bool) Directive {
		return Directive{Intents: []Intent{{Commodity: dcGrain, Side: Bid, Quantity: 3, Limit: limit}}, Sticky: sticky}
	}
	tests := []struct {
		name   string
		sticky bool
		want   []int // orders at the limit in steps 1, 2 and 3
	}{
		{"one tick", false, []int{1, 0, 0}},
		{"sticky", true, []int{1, 1, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newCapture()
			lines := h.lines
			w, rec, id, _, _, _ := directWorld(t, WithLogger(slog.New(h)))
			n := len(rec.events)
			if err := w.Direct(id, d(tt.sticky)); err != nil {
				t.Fatalf("Direct() = %v, want nil", err)
			}
			if len(rec.events) != n {
				t.Errorf("Direct emitted %d events, want none (spec-world.md 8.3)", len(rec.events)-n)
			}
			if got := slices.IndexFunc(*lines, func(l logRec) bool {
				return l.msg == "directive set" && l.level.String() == "DEBUG" && l.attrs["agent"] == int64(id)
			}); got < 0 {
				t.Errorf("no Debug line \"directive set\" with agent %d in %v", id, *lines)
			}
			for i, want := range tt.want {
				rec.events = rec.events[:0]
				if err := w.Step(); err != nil {
					t.Fatalf("Step %d = %v, want nil", i+1, err)
				}
				if got := len(bidsAt(rec, id, limit)); got != want {
					t.Errorf("step %d: %d orders at %d, want %d", i+1, got, limit, want)
				}
			}
			if gone := w.agents[id].directive == nil; gone == tt.sticky {
				t.Errorf("mailbox empty after three steps = %v, want %v", gone, !tt.sticky)
			}
		})
	}
	t.Run("a call after the step acts in the next", func(t *testing.T) {
		t.Parallel()
		w, rec, id, _, _, _ := directWorld(t)
		if err := w.Step(); err != nil {
			t.Fatalf("Step() = %v, want nil", err)
		}
		rec.events = rec.events[:0]
		if err := w.Direct(id, d(false)); err != nil {
			t.Fatalf("Direct() = %v, want nil", err)
		}
		if err := w.Step(); err != nil {
			t.Fatalf("Step() = %v, want nil", err)
		}
		if got := len(bidsAt(rec, id, limit)); got != 1 {
			t.Errorf("%d orders at %d in the step after the call, want 1", got, limit)
		}
	})
}

// TestDirect_AffordableNotChecked verifies that the call does not check what the
// tick can change (spec-integration.md 3.2.1): a bid the agent cannot pay is
// accepted by Direct and rejected by the step with insufficient_funds.
//
// Requirement: FR-INT-010
func TestDirect_AffordableNotChecked(t *testing.T) {
	t.Parallel()
	w, rec, _, _, _, _ := directWorld(t)
	poor, err := w.AddAgent(NewAgent{Region: regLowland, Role: 0})
	if err != nil {
		t.Fatalf("AddAgent() = %v, want nil", err)
	}
	if err := w.Direct(poor, Directive{Intents: []Intent{{Commodity: dcGrain, Side: Bid, Quantity: 5, Limit: 77}}}); err != nil {
		t.Fatalf("Direct(a bid the agent cannot pay) = %v, want nil", err)
	}
	if err := w.Step(); err != nil {
		t.Fatalf("Step() = %v, want nil", err)
	}
	var rejected []OrderRejected
	for _, r := range of[OrderRejected](rec) {
		if r.Agent == poor && r.Commodity == dcGrain && r.Side == Bid {
			rejected = append(rejected, r)
		}
	}
	if len(rejected) != 1 || rejected[0].Reason != ReasonInsufficientFunds {
		t.Errorf("rejected bids of the agent with no money = %+v, want one with reason insufficient_funds", rejected)
	}
}

// TestDirect_NoJobLeavesWork verifies that a directive with intents and no job
// request leaves Work to the fallback (spec-integration.md 4.3.3): the agent runs
// the recipe the fallback chooses, as an agent with no directive does, while a
// request for no job runs nothing.
//
// Requirement: FR-INT-028
func TestDirect_NoJobLeavesWork(t *testing.T) {
	t.Parallel()
	ran := func(d *Directive) int {
		w, rec, id, _, _, _ := directWorld(t)
		if d != nil {
			if err := w.Direct(id, *d); err != nil {
				t.Fatalf("Direct(%+v) = %v, want nil", d, err)
			}
		}
		if err := w.Step(); err != nil {
			t.Fatalf("Step() = %v, want nil", err)
		}
		n := 0
		for _, e := range of[RecipeExecuted](rec) {
			if e.Agent == id {
				n++
			}
		}
		return n
	}
	intents := &Directive{Intents: []Intent{{Commodity: dcGrain, Side: Bid, Quantity: 1, Limit: 10}}}
	noJob := &Directive{Job: &JobRequest{NoJob: true}}
	none := ran(nil)
	if none == 0 {
		t.Fatal("an agent with no directive ran no recipe, so the fallback's Work cannot be observed")
	}
	if got := ran(intents); got != none {
		t.Errorf("recipes run with intents and no job = %d, want %d, those of the fallback", got, none)
	}
	if got := ran(noJob); got != 0 {
		t.Errorf("recipes run with a request for no job = %d, want 0", got)
	}
}

// TestPositions_P19 verifies the position P1.9: a world that never held an
// external agent records no P1.9 line, one with an external agent records it every
// tick. (The restore leg waits for the restore of an external agent, Phase 8.)
//
// Requirement: FR-WLD-040, FR-INT-027
func TestPositions_P19(t *testing.T) {
	t.Parallel()
	count := func(lines []string) int {
		n := 0
		for _, l := range lines {
			if strings.Contains(l, " P1.9 ") {
				n++
			}
		}
		return n
	}
	w, _, _ := extWorld(t, directConfig())
	lines, err := positionHashes(w, 3)
	if err != nil {
		t.Fatalf("positionHashes() = %v, want nil", err)
	}
	if got := count(lines); got != 0 {
		t.Errorf("a world with no external agent recorded %d P1.9 lines, want 0", got)
	}
	if _, err := w.AddAgent(NewAgent{Region: regLowland, Role: 0}); err != nil {
		t.Fatalf("AddAgent() = %v, want nil", err)
	}
	if lines, err = positionHashes(w, 3); err != nil || count(lines) != 3 {
		t.Errorf("a world with an external agent recorded %d P1.9 lines (error %v), want 3", count(lines), err)
	}
}

// TestIntegration_ExternalsAreSortedAndRebuilt verifies the list P1.9 visits: it is
// ascending whatever the order agents were watched in, a removed agent leaves it at
// the next P1.9, and a world that had its last external removed stops recording the
// position after that step.
//
// Guard: the order of the list is the order of P1.9's effects, and a stale entry
// would make a restored world differ from its original.
func TestIntegration_ExternalsAreSortedAndRebuilt(t *testing.T) {
	t.Parallel()
	var s integrationState
	for _, slot := range []int32{9, 4, 7, 4} {
		s.watch(slot)
	}
	if want := []int32{4, 7, 9}; !slices.Equal(s.externals, want) || !s.visit {
		t.Errorf("externals after watching 9, 4, 7, 4 = %v (visit %v), want %v (true)", s.externals, s.visit, want)
	}
	w, _, id, _, removed, _ := directWorld(t)
	if err := w.Step(); err != nil {
		t.Fatalf("Step() = %v, want nil", err)
	}
	if slices.Contains(w.integ.externals, int32(removed)) || !slices.Contains(w.integ.externals, int32(id)) {
		t.Errorf("externals = %v, want the live external %d and not the removed %d", w.integ.externals, id, removed)
	}
	for _, a := range w.integ.externals {
		if a2 := &w.agents[a]; !a2.external {
			t.Errorf("slot %d in externals is not an external agent", a)
		}
	}
	want := slices.Clone(w.integ.externals)
	w.rebuildExternals()
	if !slices.Equal(w.integ.externals, want) {
		t.Errorf("rebuildExternals gave %v, want %v", w.integ.externals, want)
	}
	for _, slot := range want {
		if err := w.RemoveAgent(AgentID(slot)); err != nil {
			t.Fatalf("RemoveAgent(%d) = %v, want nil", slot, err)
		}
	}
	if err := w.Step(); err != nil {
		t.Fatalf("Step() = %v, want nil", err)
	}
	if w.integ.visit || len(w.integ.externals) != 0 {
		t.Errorf("after removing every external: visit %v, externals %v, want false, none", w.integ.visit, w.integ.externals)
	}
}

// TestP19Directives_AllocatesNothing verifies that the position itself allocates
// nothing once the world is built, with a sticky and a one-tick directive in the
// mailboxes (a non-sticky one is dropped, the pointer moves, no copy is made).
// Not parallel: testing.AllocsPerRun panics in a parallel test.
//
// Requirement: NFR-INT-002
func TestP19Directives_AllocatesNothing(t *testing.T) {
	w, _, id, _, _, _ := directWorld(t)
	other, err := w.AddAgent(NewAgent{Region: regLowland, Role: 0})
	if err != nil {
		t.Fatalf("AddAgent() = %v, want nil", err)
	}
	sticky, oneTick := &Directive{Hold: []CommodityID{dcGrain}, Sticky: true}, &Directive{Hold: []CommodityID{dcSalt}}
	got := testing.AllocsPerRun(50, func() {
		w.agents[id].directive, w.agents[other].directive = sticky, oneTick
		w.p19Directives()
	})
	if got != 0 {
		t.Errorf("allocations of a P1.9 pass = %v, want 0", got)
	}
}
