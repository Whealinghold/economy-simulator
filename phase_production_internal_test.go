package economysim

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// evLog is a recording Observer of the internal tests.
type evLog struct{ events []Event }

func (r *evLog) Observe(e Event) { r.events = append(r.events, e) }

// of returns the events of one type, in order.
func of[T Event](r *evLog) []T {
	var out []T
	for _, e := range r.events {
		if v, ok := e.(T); ok {
			out = append(out, v)
		}
	}
	return out
}

// logRec is one line captured by capture.
type logRec struct {
	level slog.Level
	msg   string
	attrs map[string]int64
	strs  map[string]string
}

// capture is a slog.Handler that keeps every line with its integer attributes.
type capture struct{ lines *[]logRec }

func newCapture() capture { return capture{lines: new([]logRec)} }

func (capture) Enabled(context.Context, slog.Level) bool { return true }
func (c capture) Handle(_ context.Context, r slog.Record) error {
	l := logRec{level: r.Level, msg: r.Message, attrs: map[string]int64{}, strs: map[string]string{}}
	r.Attrs(func(a slog.Attr) bool {
		switch a.Value.Kind() {
		case slog.KindInt64:
			l.attrs[a.Key] = a.Value.Int64()
		case slog.KindString:
			l.strs[a.Key] = a.Value.String()
		case slog.KindBool:
			l.attrs[a.Key] = map[bool]int64{false: 0, true: 1}[a.Value.Bool()]
		}
		return true
	})
	*c.lines = append(*c.lines, l)
	return nil
}
func (c capture) WithAttrs([]slog.Attr) slog.Handler { return c }
func (c capture) WithGroup(string) slog.Handler      { return c }

// lines returns the captured lines with the message, and forgets them all.
func (c capture) take(msg string) []logRec {
	var out []logRec
	for _, l := range *c.lines {
		if l.msg == msg {
			out = append(out, l)
		}
	}
	*c.lines = nil
	return out
}

// tapStrategy is a consumer-style Strategy that wraps the default: onWork may
// replace the request of Work, and every call of Work is passed to it first.
type tapStrategy struct {
	onWork func(v *AgentView) (JobRequest, bool)
}

func (s *tapStrategy) Work(v *AgentView, rng *rand.Rand) JobRequest {
	if s.onWork != nil {
		if req, ok := s.onWork(v); ok {
			return req
		}
	}
	return DefaultStrategy{}.Work(v, rng)
}

func (*tapStrategy) Trade(v *AgentView, rng *rand.Rand, p *TradePlan) {
	DefaultStrategy{}.Trade(v, rng, p)
}

func (*tapStrategy) Revise(o *OutcomeView, rng *rand.Rand) Belief {
	return DefaultStrategy{}.Revise(o, rng)
}

var _ Strategy = (*tapStrategy)(nil)

// millConfig is the scripted chain of the production tests: one region, one
// role "miller" with one recipe "mill" that takes 2 "grain" and gives 3 "flour"
// in duration ticks, and households households of weight 1 (Scale 1 and
// MinAgents 10 give min(10, households) agents). A start takes 2 grain from an
// agent that holds 10, and the grain is also the upkeep, so the default
// strategy keeps (10 - 1) / 2 = 4 executions of food and starts the 1 it wants.
// The commodity IDs are the canonical order: flour 0, grain 1.
func millConfig(duration, households int64) Config {
	c := shellConfig()
	c.Commodities = []Commodity{
		{ID: "grain", ReferencePrice: 1_000, Raw: true},
		{ID: "flour", ReferencePrice: 3_000},
	}
	c.Recipes = []Recipe{{
		ID: "mill", Duration: duration, TraineeYield: 5_000,
		Inputs:  []Item{{Commodity: "grain", Quantity: 2}},
		Outputs: []Output{{Commodity: "flour", Quantity: 3, CostShare: 10_000}},
	}}
	// OutputCoverTicks 0 gives an output target of 0, which never halts a recipe,
	// so a stock of flour that builds up does not stop the scripted jobs.
	c.World.OutputCoverTicks = 0
	c.Roles[0].ID = "miller"
	c.Roles[0].Recipes = []string{"mill"}
	c.Roles[0].StartStock = []Item{{Commodity: "grain", Quantity: 10}}
	c.Groups = []Group{{Region: "home", Role: "miller", Households: households}}
	return c
}

const (
	flour = 0
	grain = 1
)

// newMill builds a world of c with the observer and options, failing the test
// on an error.
func newMill(t *testing.T, c Config, opts ...Option) *World {
	t.Helper()
	w, err := NewWorld(c, opts...)
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	return w
}

// tickInspecting runs one step by calling the positions of the straight line one
// by one, as the replay of the step does, and calls inspect after P4.1, before
// the outcomes: the welfare scratch of the tick and the flow totals can be read
// there, because P6.3 resets the scratch (D158) and P8.6 clears the totals. The
// replay is checked against Step by TestTickInspecting_ReplayEqualsStep.
func tickInspecting(t *testing.T, w *World, inspect func()) {
	t.Helper()
	if err := w.beginStep(); err != nil {
		t.Fatalf("beginStep() at tick %d error = %v, want nil", w.Tick(), err)
	}
	err := replayTo41(w)
	if inspect != nil {
		inspect()
	}
	if err == nil {
		err = replayFrom51(w)
	}
	if err := w.endStep(err); err != nil {
		t.Fatalf("step at tick %d error = %v, want nil", w.Tick(), err)
	}
}

// replayTo41 calls the positions of the straight line up to and including P4.1.
func replayTo41(w *World) error {
	w.p02()
	if err := w.p11Moment(); err != nil {
		return err
	}
	w.p12Anchors()
	w.p21Complete()
	w.p23Snapshot()
	if err := w.p25Work(); err != nil {
		return err
	}
	w.p26Ration()
	if err := w.p27Start(); err != nil {
		return err
	}
	w.p28aHalted()
	for _, f := range []func() error{w.p32Trade, w.p34Estates, w.p41Clear} {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}

// replayFrom51 calls the positions of the straight line after P4.1.
func replayFrom51(w *World) error {
	w.p51Window()
	if err := w.p53Markdown(); err != nil {
		return err
	}
	w.p54Outcomes()
	if err := w.p61Upkeep(); err != nil {
		return err
	}
	w.p63Welfare()
	w.p64Shortfall()
	if err := w.p81Age(); err != nil {
		return err
	}
	w.p82Flows()
	return w.p84Audit()
}

// TestTickInspecting_ReplayEqualsStep verifies that the replay of the helper
// above gives the world a step gives: the same balances, holdings and beliefs
// after 5 ticks of a scripted market, so the tests that read the scratch through
// it read the real step.
//
// Requirement: FR-WLD-004
func TestTickInspecting_ReplayEqualsStep(t *testing.T) {
	t.Parallel()
	trade := func(v *AgentView, p *TradePlan) {
		side, limit := Bid, Price(1_100)
		if v.Agent == 2 {
			side, limit = Ask, 900
		}
		p.Intents = append(p.Intents, Intent{Commodity: grain, Side: side, Quantity: 1, Limit: limit})
	}
	a, b := newTrader(t, tradeConfig(2), trade), newTrader(t, tradeConfig(2), trade)
	for range 5 {
		step(t, a, 1)
		tickInspecting(t, b, nil)
	}
	for id := AgentID(1); id <= 2; id++ {
		ma, _ := balance(t, a, id)
		mb, _ := balance(t, b, id)
		if ma != mb || held(t, a, id, grain) != held(t, b, id, grain) || a.agents[id].beliefs[0] != b.agents[id].beliefs[0] {
			t.Errorf("agent %d: replay gave money %d, grain %d, belief %+v; Step gave %d, %d, %+v",
				id, mb, held(t, b, id, grain), b.agents[id].beliefs[0], ma, held(t, a, id, grain), a.agents[id].beliefs[0])
		}
	}
}

// step runs n steps of w and fails the test on an error.
func step(t *testing.T, w *World, n int) {
	t.Helper()
	for range n {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d error = %v, want nil", w.Tick(), err)
		}
	}
}

// flowOf returns the total of the flow of a reason and a commodity in the
// current tick's totals.
func flowOf(w *World, r ledger.Reason, commodity int) int64 {
	for _, f := range w.ledger.AppendFlows(nil, false) {
		if f.Reason == r && f.Commodity == commodity {
			return f.Total
		}
	}
	return 0
}

// held returns the units of a commodity an agent holds.
func held(t *testing.T, w *World, id AgentID, commodity int) int64 {
	t.Helper()
	n, _, err := w.ledger.Holding(w.agents[id].account, commodity)
	if err != nil {
		t.Fatalf("Holding(agent %d, commodity %d) error = %v, want nil", id, commodity, err)
	}
	return n
}

// TestRotationAt verifies the rotation of spec-world.md 6.1.4: for n agents a
// walk at tick t starts at the position t mod n and wraps, so each agent has the
// first turn once in n ticks.
//
// Arithmetic: n = 3 and ticks 0 to 3 start at 0, 1, 2 and 0 (t mod 3), so the
// visits are [0 1 2], [1 2 0], [2 0 1] and [0 1 2]; one agent is always index 0.
//
// Requirement: FR-WLD-043, FR-PRO-016
func TestRotationAt(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		n    int
		tick int64
		want []int
	}{
		{"tick 0 of 3", 3, 0, []int{0, 1, 2}},
		{"tick 1 of 3", 3, 1, []int{1, 2, 0}},
		{"tick 2 of 3", 3, 2, []int{2, 0, 1}},
		{"tick 3 wraps to the start", 3, 3, []int{0, 1, 2}},
		{"one agent", 1, 7, []int{0}},
		{"a large tick: 1 000 000 007 mod 4 = 3", 4, 1_000_000_007, []int{3, 0, 1, 2}},
	} {
		got := make([]int, 0, tt.n)
		for k := range tt.n {
			got = append(got, rotationAt(tt.n, tt.tick, k))
		}
		if !slices.Equal(got, tt.want) {
			t.Errorf("%s: rotationAt(%d, %d, 0..%d) = %v, want %v", tt.name, tt.n, tt.tick, tt.n-1, got, tt.want)
		}
	}
}

// TestStep_OneTickJobStartsAndCompletesInItsStartTick verifies a recipe of
// duration 1 over two ticks: each of the two agents starts one execution and
// completes it in the same tick, the rotation starts at t mod 2, the
// RecipeExecuted fields equal the arithmetic, and the flow totals of the tick
// are the recipes' arithmetic.
//
// Arithmetic: a household of weight 1 starts k = 1 execution (it holds 10 grain,
// keeps 1 for upkeep and mill takes 2). The input is 1 * 2 = 2 grain and the
// output is q * k * E / 10 000 = 3 * 1 * 10 000 / 10 000 = 3 flour (a residue
// below 10 000 adds nothing). Two agents make 4 grain of recipe_input and 6
// flour of recipe_output in a tick. The agent list is [1, 2], so tick 0 starts
// at index 0 (agents 1, 2) and tick 1 at index 1 (agents 2, 1).
//
// Requirement: FR-PRO-014, FR-PRO-016, FR-PRO-017, FR-WLD-043, FR-WLD-044, FR-OBS-014
func TestStep_OneTickJobStartsAndCompletesInItsStartTick(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := newMill(t, millConfig(1, 2), WithObserver(rec), WithAudit())
	for tick := range int64(2) {
		tickInspecting(t, w, func() {
			if in, out := flowOf(w, ledger.RecipeInput, grain), flowOf(w, ledger.RecipeOutput, flour); in != 4 || out != 6 {
				t.Errorf("tick %d: flows recipe_input grain = %d, recipe_output flour = %d, want 4 and 6", tick, in, out)
			}
		})
		if err := w.Audit(); err != nil {
			t.Fatalf("Audit() after tick %d = %v, want nil", tick, err)
		}
	}
	got := of[RecipeExecuted](rec)
	if len(got) != 4 {
		t.Fatalf("RecipeExecuted events = %d, want 4 (two agents, two ticks)", len(got))
	}
	for i, want := range []struct {
		tick  int64
		agent AgentID
	}{{0, 1}, {0, 2}, {1, 2}, {1, 1}} {
		e := got[i]
		wantEvent := RecipeExecuted{
			Tick: want.tick, Agent: want.agent, Region: 0, Recipe: 0, StartTick: want.tick, Executions: 1,
			Outputs: []Amount{{Commodity: flour, Quantity: 3}}, EffectiveYield: 10_000,
		}
		if e.Tick != wantEvent.Tick || e.Agent != wantEvent.Agent || e.Region != wantEvent.Region || e.Recipe != wantEvent.Recipe ||
			e.StartTick != wantEvent.StartTick || e.Executions != wantEvent.Executions || e.EffectiveYield != wantEvent.EffectiveYield ||
			!slices.Equal(e.Outputs, wantEvent.Outputs) || len(e.Disposed) != 0 || len(e.Overflow) != 0 {
			t.Errorf("RecipeExecuted #%d = %+v, want %+v", i, e, wantEvent)
		}
	}
	// After two ticks: 10 - 2 * 2 - 2 * 1 = 4 grain (the inputs and the upkeep of
	// 1 each tick, eaten at P6.1) and 2 * 3 = 6 flour each.
	for _, id := range []AgentID{1, 2} {
		if g, f := held(t, w, id, grain), held(t, w, id, flour); g != 4 || f != 6 {
			t.Errorf("agent %d holds %d grain and %d flour after 2 ticks, want 4 and 6", id, g, f)
		}
	}
}

// TestStep_JobOfDurationThreeCompletesInSPlusTwo verifies the timing of a job
// of duration 3 over six ticks: it starts in tick s, completes in s + 2 with no
// event in between, its pending output survives a tick and is created only at
// the completion, the inputs are taken once at the start, a completing agent
// starts nothing in the tick it completes, and the next job starts in s + 3.
//
// Arithmetic: duration 3 started in tick 0 runs in ticks 0, 1 and 2 (remaining
// 3, 2 and 1 after the tick's step... 3 -> 2 at the start tick, 1 at tick 1, 0 at
// tick 2). The next start is tick 3 and completes in tick 5. Each start takes 2
// grain from each of two agents (4 in all, at ticks 0 and 3) and each completion
// creates 3 flour for each (6 in all, at ticks 2 and 5).
//
// Requirement: FR-PRO-015, FR-PRO-029, FR-PRO-041, FR-WLD-004
func TestStep_JobOfDurationThreeCompletesInSPlusTwo(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := newMill(t, millConfig(3, 2), WithObserver(rec))
	wantIn := []int64{4, 0, 0, 4, 0, 0}
	wantOut := []int64{0, 0, 6, 0, 0, 6}
	wantPending := []int64{3, 3, 0, 3, 3, 0}
	for tick := range 6 {
		tickInspecting(t, w, func() {
			if in, out := flowOf(w, ledger.RecipeInput, grain), flowOf(w, ledger.RecipeOutput, flour); in != wantIn[tick] || out != wantOut[tick] {
				t.Errorf("tick %d: recipe_input = %d, recipe_output = %d, want %d and %d", tick, in, out, wantIn[tick], wantOut[tick])
			}
			for i, ws := range w.welfare.slots {
				if i >= 1 && !ws.jobSet {
					t.Errorf("tick %d: agent %d not marked working, want a working tick on every tick of a job", tick, i)
				}
			}
		})
		for _, id := range []AgentID{1, 2} {
			pending := int64(0)
			if j := &w.agents[id].job; j.executions > 0 {
				pending = j.pending[0]
			}
			if pending != wantPending[tick] {
				t.Errorf("after tick %d: pending flour of agent %d = %d, want %d", tick, id, pending, wantPending[tick])
			}
		}
		for i, s := range w.welfare.slots {
			if s != (flows{}) {
				t.Errorf("after tick %d: welfare scratch of slot %d = %+v, want zero (the clear of P8.6)", tick, i, s)
			}
		}
	}
	var seen [][2]int64
	for _, e := range of[RecipeExecuted](rec) {
		seen = append(seen, [2]int64{e.Tick, e.StartTick})
	}
	if want := [][2]int64{{2, 0}, {2, 0}, {5, 3}, {5, 3}}; !slices.Equal(seen, want) {
		t.Errorf("(tick, start tick) of RecipeExecuted = %v, want %v: complete in s + 2", seen, want)
	}
}

// TestMarkWorking_OncePerAgentPerTickOnEveryPath verifies that the working tick
// is set once per agent per tick whatever the path of the job, and that its
// executions are the k of the job (FR-WEL-017, the role-with-recipes half): a
// job started and completed in one tick (duration 1), started and then
// completed in the next (duration 2), and continued (duration 3), over eight
// ticks. markWorking panics on a second call for a slot, so a double call fails
// the test by panic; the wear of the executions started is recorded at the start.
//
// Arithmetic: one household runs k = 1 execution in every tick of every job, so
// the scratch of each agent reads k = 1, jobSet and not own use at each tick.
//
// Requirement: FR-WEL-017, FR-WEL-003, FR-PRO-014
func TestMarkWorking_OncePerAgentPerTickOnEveryPath(t *testing.T) {
	t.Parallel()
	for _, d := range []int64{1, 2, 3} {
		c := millConfig(d, 2)
		c.Roles[0].StartStock = []Item{{Commodity: "grain", Quantity: 100}} // enough for every start
		w := newMill(t, c)
		for tick := range 8 {
			tickInspecting(t, w, func() {
				for _, id := range []AgentID{1, 2} {
					if k, own, traded := w.welfare.workRead(int(id)); k != 1 || own || traded || !w.welfare.slots[id].jobSet {
						t.Errorf("duration %d, tick %d, agent %d: working k = %d, own use %t, traded %t, want k 1 and a working tick", d, tick, id, k, own, traded)
					}
				}
			})
		}
	}
}

// TestStep_WorkIsCalledForEveryFreeAgentBeforeAnyStart verifies the two passes
// of spec-world.md 6.4.1: a recording Strategy that wraps the default sees no
// recipe input taken in any call of Work of a tick, although the starts of the
// tick take 4 grain, and sees the agents in the rotation order. A job in progress
// is not passed to Work.
//
// Arithmetic: with duration 2, tick 0 starts both jobs; tick 1 completes them
// (Work is not called: the agents completed a job), tick 2 calls Work again. In
// ticks 0 and 2 the input flow is 0 at every call and 4 grain (2 agents * 2) at
// the end of the tick; ticks 0 and 2 have agents [1, 2] and [1, 2] (t mod 2 = 0).
//
// Requirement: FR-WLD-046, FR-PRO-016, FR-AGT-006
func TestStep_WorkIsCalledForEveryFreeAgentBeforeAnyStart(t *testing.T) {
	t.Parallel()
	var w *World
	type call struct {
		tick  int64
		agent AgentID
		input int64
	}
	var calls []call
	s := &tapStrategy{onWork: func(v *AgentView) (JobRequest, bool) {
		calls = append(calls, call{w.Tick(), v.Agent, flowOf(w, ledger.RecipeInput, grain)})
		return JobRequest{}, false
	}}
	c := millConfig(2, 2)
	c.Roles[0].Strategy = "tap"
	w = newMill(t, c, WithStrategy("tap", s))
	var endInputs []int64
	for range 3 {
		tickInspecting(t, w, func() { endInputs = append(endInputs, flowOf(w, ledger.RecipeInput, grain)) })
	}
	want := []call{{0, 1, 0}, {0, 2, 0}, {2, 1, 0}, {2, 2, 0}}
	if !slices.Equal(calls, want) {
		t.Errorf("calls of Work (tick, agent, recipe input so far) = %v, want %v", calls, want)
	}
	if want := []int64{4, 0, 4}; !slices.Equal(endInputs, want) {
		t.Errorf("recipe input at the end of ticks 0 to 2 = %v, want %v", endInputs, want)
	}
}

// TestWork_ViewIsBuiltFromTheStateAfterP21 verifies what the view that Work
// receives holds: the agent's own stock as of the call (a job completed in an
// earlier tick has put its output there), the published price of the market,
// which is the seed price at tick 0 and the price the previous clearing
// published after it, the unit cost, which is the cost anchor of P1.2 for a
// commodity a local recipe makes and 0 otherwise, and that the anchor is the one
// computed once at P1.2. A strategy that writes to its view changes no state and
// no later view.
//
// Arithmetic: tick 0 writes to its view and requests no job, so the stock of tick 1
// is the untouched 10 grain less the 1 of upkeep eaten at P6.1 of tick 0, 9, and 0 flour. Mill costs 2 grain * 1 000 + 1 tick * wage 100 = 2 100 and makes 3
// flour at E = 10 000, so the anchor is ceil(2 100 * 10 000 / (3 * 10 000)) = 700
// (spec-production.md 9.1). When the published grain price is 1 200 it is
// ceil((2 * 1 200 + 100) / 3) = ceil(833.33) = 834. Grain is made by no recipe,
// so its unit cost is 0.
//
// Requirement: FR-AGT-024, FR-AGT-073, FR-PRO-048, FR-WLD-042
func TestWork_ViewIsBuiltFromTheStateAfterP21(t *testing.T) {
	t.Parallel()
	var w *World
	type seen struct {
		stockFlour, stockGrain, priceGrain, costFlour, costGrain int64
		sameAnchor                                               bool
	}
	var got []seen
	s := &tapStrategy{onWork: func(v *AgentView) (JobRequest, bool) {
		var sn seen
		for _, c := range v.Commodities {
			switch c.Commodity {
			case flour:
				sn.stockFlour, sn.costFlour = c.Stock, int64(c.UnitCost)
			case grain:
				sn.stockGrain, sn.priceGrain, sn.costGrain = c.Stock, c.Market.Price, int64(c.UnitCost)
			}
		}
		sn.sameAnchor = sn.costFlour == w.prod.unitCost(0, flour)
		got = append(got, sn)
		if w.Tick() > 0 {
			return JobRequest{}, false
		}
		// Tick 0 writes to the view and asks for no job: nothing may change.
		for i := range v.Commodities {
			v.Commodities[i].Stock, v.Commodities[i].UnitCost = 0, 0
		}
		return JobRequest{NoJob: true}, true
	}}
	c := millConfig(1, 1)
	c.Roles[0].Strategy = "tap"
	w = newMill(t, c, WithStrategy("tap", s))
	w.seedWage[0] = 100
	step(t, w, 1)
	// A market with no order publishes its last price: here the price a trade
	// would have published is set.
	w.markets.reports[w.markets.index(0, grain)].Price = 1_200
	step(t, w, 1)
	want := []seen{
		{0, 10, 1_000, 700, 0, true}, // tick 0: the seed prices, an empty flour account
		{0, 9, 1_200, 834, 0, true},  // tick 1: the write of tick 0 is gone, 1 grain eaten, the new price
	}
	if !slices.Equal(got, want) {
		t.Errorf("view of Work at ticks 0 and 1 = %+v, want %+v", got, want)
	}
}

// TestWork_RefusalsAndTheHaltAreCountedFromTheViewAsFilled verifies the counts of
// P2.5 and the aggregates of P2.8a for a hand-built world: a recipe that the
// halt skips (ProductionHalted, with the skipped row), a request off the role's
// list (RecipeRefused not_on_role, before any other check, although the recipe
// could not run either), and a household that cannot start (RecipeRefused
// no_input, counted for the recipe the work report names blocked). The aggregates
// come in ascending order, ProductionHalted before RecipeRefused, and a tick
// with no refusal has no RecipeRefused. A strategy that writes to its view leaves
// the halt rows equal to the default strategy's.
//
// Arithmetic: a miller of weight 1 holding 9 flour has OutputTarget =
// ceil(1 * 3 * 10 000 * 1 / (10 000 * 1)) = 3 flour, and the halt acts when stock
// * 10 000 >= 30 000 * 3, that is 90 000 >= 90 000, so both of the 2 households
// are halted: Free 2, Halted 2, one skipped row (mill, 2 households).
//
// Requirement: FR-AGT-072, FR-OBS-016, FR-OBS-095, FR-PRO-048, FR-WLD-046
func TestWork_RefusalsAndTheHaltAreCountedFromTheViewAsFilled(t *testing.T) {
	t.Parallel()
	haltConfig := func() Config {
		c := millConfig(1, 2)
		c.World.OutputCoverTicks = 1
		c.Roles[0].StartStock = []Item{{Commodity: "grain", Quantity: 10}, {Commodity: "flour", Quantity: 9}}
		return c
	}
	halted := ProductionHalted{Tick: 0, Region: 0, Role: 0, Free: 2, Halted: 2, Skipped: []HaltRow{{Recipe: 0, Households: 2}}}
	t.Run("the default strategy", func(t *testing.T) {
		t.Parallel()
		rec := &evLog{}
		w := newMill(t, haltConfig(), WithObserver(rec))
		step(t, w, 1)
		if got := of[ProductionHalted](rec); len(got) != 1 || !haltEqual(got[0], halted) {
			t.Errorf("ProductionHalted = %+v, want one equal to %+v", got, halted)
		}
		if got := of[RecipeRefused](rec); len(got) != 0 {
			t.Errorf("RecipeRefused = %+v, want none: the halt is not a refusal", got)
		}
		if n := len(of[RecipeExecuted](rec)); n != 0 {
			t.Errorf("RecipeExecuted = %d, want 0: a halted recipe starts nothing", n)
		}
	})
	t.Run("a strategy that writes to its view", func(t *testing.T) {
		t.Parallel()
		rec := &evLog{}
		s := &tapStrategy{onWork: func(v *AgentView) (JobRequest, bool) {
			for i := range v.Commodities {
				v.Commodities[i].Stock = 0
			}
			for i := range v.Recipes {
				v.Recipes[i].MaxExecutions = 5
			}
			return JobRequest{NoJob: true}, true
		}}
		c := haltConfig()
		c.Roles[0].Strategy = "tap"
		w := newMill(t, c, WithObserver(rec), WithStrategy("tap", s))
		step(t, w, 1)
		if got := of[ProductionHalted](rec); len(got) != 1 || !haltEqual(got[0], halted) {
			t.Errorf("ProductionHalted = %+v, want one equal to %+v: the counts read the view as filled", got, halted)
		}
	})
	t.Run("a request off the role's list", func(t *testing.T) {
		t.Parallel()
		// A second recipe, "bake", is on no role's list and needs 100 grain, which no
		// agent holds: not_on_role is the first check and wins over no_input.
		c := millConfig(1, 2)
		c.Recipes = append(c.Recipes, Recipe{
			ID: "bake", Duration: 1, TraineeYield: 5_000,
			Inputs:  []Item{{Commodity: "grain", Quantity: 100}},
			Outputs: []Output{{Commodity: "flour", Quantity: 1, CostShare: 10_000}},
		})
		c.Roles = append(c.Roles, DefaultRole())
		c.Roles[1].ID = "baker"
		c.Roles[1].Recipes = []string{"bake"}
		c.Roles[1].Closed = true
		cfg := c.canonical()
		bake, _ := cfg.recipeID("bake")
		s := &tapStrategy{onWork: func(*AgentView) (JobRequest, bool) { return JobRequest{Recipe: bake}, true }}
		c.Roles[0].Strategy = "tap"
		rec := &evLog{}
		lg := newCapture()
		w := newMill(t, c, WithObserver(rec), WithStrategy("tap", s), WithLogger(slog.New(lg)))
		step(t, w, 1)
		want := RecipeRefused{Tick: 0, Region: 0, Recipe: bake, NotOnRole: 2}
		if got := of[RecipeRefused](rec); len(got) != 1 || got[0] != want {
			t.Errorf("RecipeRefused = %+v, want one equal to %+v", got, want)
		}
		lines := lg.take("recipe not started")
		if len(lines) != 2 || lines[0].level != slog.LevelDebug || lines[0].attrs["recipe"] != int64(bake) || lines[0].attrs["agent"] != 1 {
			t.Errorf("Debug lines 'recipe not started' = %+v, want 2 at Debug for recipe %d, the first for agent 1", lines, bake)
		}
	})
	t.Run("blocked for lack of input, then only in the tick with a refusal", func(t *testing.T) {
		t.Parallel()
		// 3 grain: tick 0 keeps 1 for upkeep and has room for (3 - 1) / 2 = 1
		// execution, so the household starts; 3 - 2 = 1 grain is left, mill needs
		// 2, so in tick 1 the first recipe that cannot run is mill, blocked for
		// no_input, and each of the 2 households is counted once.
		c := millConfig(1, 2)
		c.Roles[0].StartStock = []Item{{Commodity: "grain", Quantity: 3}}
		rec := &evLog{}
		w := newMill(t, c, WithObserver(rec))
		step(t, w, 1)
		if got := of[RecipeRefused](rec); len(got) != 0 {
			t.Errorf("RecipeRefused at tick 0 = %+v, want none: both started", got)
		}
		step(t, w, 1)
		want := RecipeRefused{Tick: 1, Region: 0, Recipe: 0, NoInput: 2}
		if got := of[RecipeRefused](rec); len(got) != 1 || got[0] != want {
			t.Errorf("RecipeRefused = %+v, want one equal to %+v", got, want)
		}
	})
}

// haltEqual compares two ProductionHalted values.
func haltEqual(a, b ProductionHalted) bool {
	return a.Tick == b.Tick && a.Region == b.Region && a.Role == b.Role && a.Free == b.Free && a.Halted == b.Halted &&
		a.Rested == b.Rested && a.NoRecipe == b.NoRecipe && slices.Equal(a.Skipped, b.Skipped)
}

// TestWork_FloorCountAndWarnRun verifies the work floor at P2.5: the households
// of the free agents that the floor holds to work are counted from the work
// report of the view as filled, the run of the Warn "work floor binding"
// advances for the cell every tick, and the Warn is logged at the tenth tick of a
// run and every tenth after with the region, the role and the households held,
// never before, and the run resets at once on a tick without the condition.
//
// Arithmetic: a view with ratio 20 000 against the target 15 000 and no working
// tick in a window of 10 with MinWorkShare 4 000 is held by the floor (0 working
// ticks * 10 000 < 4 000 * 10 held), so the 3 households of an agent of weight 3
// are floor-held, and 2 * 3 > 3 / 2 (written without the product), so the
// condition holds for a role of 3 households.
//
// Requirement: FR-AGT-072, FR-WEL-033
func TestWork_FloorCountAndWarnRun(t *testing.T) {
	t.Parallel()
	lg := newCapture()
	c := millConfig(1, 3)
	c.World.Scale, c.World.MinAgents = 3, 1
	w := newMill(t, c, WithLogger(slog.New(lg)))
	if len(w.agents) != 2 || w.agents[1].weight != 3 {
		t.Fatalf("agents = %d, weight %d, want the estate and one agent of 3 households", len(w.agents), w.agents[1].weight)
	}
	a := &w.agents[1]
	w.buildRotation()
	w.refreshViewCtx()
	var v AgentView
	if err := w.fillAgentView(&v, a); err != nil {
		t.Fatalf("fillAgentView() error = %v, want nil", err)
	}
	v.HasRatio, v.Ratio, v.TargetRatio, v.WorkingTicks, v.Window, v.MinWorkShare = true, 20_000, 15_000, 0, 10, 4_000
	w.countWork(0, 0, a, &v)
	if got := w.pr.cells[0]; got.free != 3 || got.floorHeld != 3 || got.households != 3 {
		t.Errorf("cell after countWork = %+v, want 3 free, 3 floor-held, 3 households", got)
	}
	for tick := 1; tick <= 20; tick++ {
		w.floorWarns()
		lines := lg.take("work floor binding")
		switch {
		case tick%10 == 0 && (len(lines) != 1 || lines[0].level != slog.LevelWarn || lines[0].attrs["households"] != 3 ||
			lines[0].attrs["region"] != 0 || lines[0].attrs["role"] != 0):
			t.Errorf("tick %d of the run: lines = %+v, want one Warn with region 0, role 0, households 3", tick, lines)
		case tick%10 != 0 && len(lines) != 0:
			t.Errorf("tick %d of the run: lines = %+v, want none before the tenth tick and between the tenths", tick, lines)
		}
	}
	w.pr.cells[0].floorHeld = 0
	w.floorWarns()
	if got := w.state.floorRun[0]; got != 0 {
		t.Errorf("run after a tick without the condition = %d, want 0", got)
	}
}

// TestWork_PanicLeavesTheWorldFailed verifies that a panic inside a position, a
// consumer's Work, leaves the world failed, emits no TickEnded and clears the
// in-step flag, so the next call returns ErrWorldFailed and not ErrInStep.
//
// Requirement: FR-WLD-006, FR-WLD-012
func TestWork_PanicLeavesTheWorldFailed(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	s := &tapStrategy{onWork: func(*AgentView) (JobRequest, bool) { panic("consumer panic") }}
	c := millConfig(1, 2)
	c.Roles[0].Strategy = "tap"
	w := newMill(t, c, WithObserver(rec), WithStrategy("tap", s))
	func() {
		defer func() {
			if v := recover(); v != "consumer panic" {
				t.Errorf("Step() panicked with %v, want the consumer's panic", v)
			}
		}()
		_ = w.Step()
	}()
	if got := len(of[TickEnded](rec)); got != 0 {
		t.Errorf("TickEnded events = %d after a panic, want 0", got)
	}
	if w.stepping || w.Err() == nil {
		t.Errorf("stepping = %t, Err = %v, want the flag cleared and the world failed", w.stepping, w.Err())
	}
	if err := w.Step(); !errors.Is(err, ErrWorldFailed) {
		t.Errorf("Step() after the panic = %v, want ErrWorldFailed", err)
	}
}

// TestFinishJob_OverflowWarnIsOnePerRecipePerTick verifies the completion of a
// job whose output has no room: the part above MaxQuantity is disposed, RecipeExecuted
// lists it in Disposed and Overflow, and the Warn "recipe output disposed" is
// logged once per recipe per tick although two agents overflow, with the
// commodity and the quantity of the first.
//
// Arithmetic: an agent holding 999 999 999 flour (MaxQuantity less 1) completes 3
// flour; the room is 1 000 000 000 - 999 999 999 = 1, so 1 is created and 3 - 1 =
// 2 are disposed, all of them overflow (no discard output). OutputCoverTicks 0
// gives a target of 0, which never halts, so the recipe runs with that stock.
//
// Requirement: FR-PRO-029, FR-OBS-096
func TestFinishJob_OverflowWarnIsOnePerRecipePerTick(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	lg := newCapture()
	w := newMill(t, millConfig(1, 2), WithObserver(rec), WithLogger(slog.New(lg)))
	// The role's start stock is bounded far below MaxQuantity, so the flour is
	// moved in during the construction phase.
	for _, id := range []AgentID{1, 2} {
		if err := fundInitial(w.ledger, w.agents[id].account, 0, []itemRow{{commodity: flour, quantity: MaxQuantity - 1}}); err != nil {
			t.Fatalf("fundInitial(agent %d) error = %v, want nil", id, err)
		}
	}
	lg.take("") // forget the construction lines
	step(t, w, 1)
	got := of[RecipeExecuted](rec)
	if len(got) != 2 {
		t.Fatalf("RecipeExecuted = %d, want 2", len(got))
	}
	for _, e := range got {
		if want := []Amount{{Commodity: flour, Quantity: 2}}; !slices.Equal(e.Disposed, want) || !slices.Equal(e.Overflow, want) ||
			!slices.Equal(e.Outputs, []Amount{{Commodity: flour, Quantity: 3}}) {
			t.Errorf("RecipeExecuted of agent %d: Outputs %v, Disposed %v, Overflow %v, want 3 created, 2 disposed, 2 overflow", e.Agent, e.Outputs, e.Disposed, e.Overflow)
		}
	}
	lines := lg.take("recipe output disposed")
	if len(lines) != 1 || lines[0].level != slog.LevelWarn || lines[0].attrs["recipe"] != 0 || lines[0].attrs["commodity"] != flour || lines[0].attrs["quantity"] != 2 {
		t.Errorf("Warn lines 'recipe output disposed' = %+v, want one for recipe 0, commodity %d, quantity 2", lines, flour)
	}
}

// TestStep_RegionOfEventsIsTheRegionOfTheAccount verifies over a world of two
// regions that the Region of every RecipeExecuted equals the region of the
// agent's ledger account at the event, and that each region has its own
// rotation: the agents of a region are listed in ascending ID and the walk of a
// region starts at t mod n_r of its own count.
//
// Arithmetic: the canonical order of the regions is far (0), home (1). Far has 3
// agents (IDs 2, 3, 4 after the two estates 0, 1) and home 2 (IDs 5, 6); at tick 1
// the walk of far starts at 1 mod 3 = 1 (3, 4, 2) and that of home at 1 mod 2 = 1
// (6, 5).
//
// Requirement: FR-OBS-018, FR-WLD-043
func TestStep_RegionOfEventsIsTheRegionOfTheAccount(t *testing.T) {
	t.Parallel()
	c := millConfig(1, 2)
	far := DefaultRegion()
	far.ID = "far"
	c.Regions = []Region{far, c.Regions[0]} // canonical order sorts by ID: far, home
	c.Groups = []Group{{Region: "home", Role: "miller", Households: 2}, {Region: "far", Role: "miller", Households: 3}}
	rec := &evLog{}
	w := newMill(t, c, WithObserver(rec))
	step(t, w, 2)
	byTick := map[int64][]AgentID{}
	for _, e := range of[RecipeExecuted](rec) {
		v, err := w.ledger.Account(w.agents[e.Agent].account)
		if err != nil {
			t.Fatalf("Account(agent %d) error = %v, want nil", e.Agent, err)
		}
		if int(e.Region) != v.Region {
			t.Errorf("RecipeExecuted of agent %d has Region %d, want %d, the region of its account", e.Agent, e.Region, v.Region)
		}
		byTick[e.Tick] = append(byTick[e.Tick], e.Agent)
	}
	// far is region 0 with agents 2, 3, 4 and home region 1 with agents 5, 6.
	if want := []AgentID{2, 3, 4, 5, 6}; !slices.Equal(byTick[0], want) {
		t.Errorf("agents of tick 0 in the order of the regions and the rotation = %v, want %v", byTick[0], want)
	}
	if want := []AgentID{3, 4, 2, 6, 5}; !slices.Equal(byTick[1], want) {
		t.Errorf("agents of tick 1 = %v, want %v (far from 1 mod 3, home from 1 mod 2)", byTick[1], want)
	}
}

// TestStep_StepAllocatesNothingWithTheDefaultObserver verifies the allocation
// limit of the production phase (observability-and-logging.md 4.5, 8.2): with the
// default logger and observer a step of the scripted chain, which starts and
// completes jobs, wears nothing and counts a halt for no household, allocates
// nothing once the scratch has grown. The step includes the trade and the
// clearing (plan-phase-1-11.md Phase 5), so the clearing is a hand-written one
// that matches nothing and allocates nothing: the default AverageClearing builds
// a new list of offers and trades on every call, by the contract of the Clearing
// interface, and its cost is the baseline of the full step of row 12, not a
// call-site cost of this row.
//
// Requirement: NFR-OBS-001
//
// Guard: an event or a log attribute built without the flag allocates on every
// tick of a long run, and only a test notices.
func TestStep_StepAllocatesNothingWithTheDefaultObserver(t *testing.T) {
	c := millConfig(2, 4)
	c.Roles[0].StartStock = []Item{{Commodity: "grain", Quantity: 1_000}}
	c.World.Clearing = "nomatch"
	w := newMill(t, c, WithClearing("nomatch", noMatch{}))
	step(t, w, 6)
	if got := testing.AllocsPerRun(20, func() {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
	}); got != 0 {
		t.Errorf("allocations per Step = %v, want 0 with the default observer and logger", got)
	}
}

// noMatch is a Clearing that matches nothing and allocates nothing.
type noMatch struct{}

func (noMatch) Clear(_, _ []Order, _ *rand.Rand) []Trade { return nil }

// TestWork_ARequestThatCannotStartIsRefusedForItsReason verifies the request
// check of P2.5 for a consumer Strategy that asks for a recipe on the role's list
// that the agent cannot start: the request is not kept, the world counts a
// RecipeRefused with no_input for the recipe asked, and nothing starts.
//
// Arithmetic: the role lists "mill" (2 grain) and "bake" (100 grain); each
// household holds 10 grain, so bake could start floor(10 / 100) = 0 executions
// and both households of the region are refused no_input for bake. The default
// work report would pick mill, which can run, so no other recipe is counted.
//
// Requirement: FR-PRO-048, FR-AGT-072
func TestWork_ARequestThatCannotStartIsRefusedForItsReason(t *testing.T) {
	t.Parallel()
	c := millConfig(1, 2)
	c.Recipes = append(c.Recipes, Recipe{
		ID: "bake", Duration: 1, TraineeYield: 5_000,
		Inputs:  []Item{{Commodity: "grain", Quantity: 100}},
		Outputs: []Output{{Commodity: "flour", Quantity: 1, CostShare: 10_000}},
	})
	c.Roles[0].Recipes = []string{"mill", "bake"}
	c.Roles[0].Strategy = "tap"
	bake, _ := c.canonical().recipeID("bake")
	s := &tapStrategy{onWork: func(*AgentView) (JobRequest, bool) { return JobRequest{Recipe: bake, Executions: 1}, true }}
	rec := &evLog{}
	w := newMill(t, c, WithObserver(rec), WithStrategy("tap", s))
	step(t, w, 1)
	want := RecipeRefused{Tick: 0, Region: 0, Recipe: bake, NoInput: 2}
	if got := of[RecipeRefused](rec); len(got) != 1 || got[0] != want {
		t.Errorf("RecipeRefused = %+v, want one equal to %+v", got, want)
	}
	if n := len(of[RecipeExecuted](rec)); n != 0 {
		t.Errorf("RecipeExecuted = %d, want 0: a refused request starts nothing", n)
	}
}

// TestWork_RestedHouseholdsAreCounted verifies that free households whose work
// share is 0 are counted in ProductionHalted.Rested, not in Halted or NoRecipe.
//
// Arithmetic: each agent's history holds one record with an income of 1 000 000
// against a cost of 1, a ratio of 10 000 * 1 000 000 / 1 (plus the padding of the
// missing ticks), far above the target of 15 000 plus the rest band of 500, so
// the share is 0 (welfare.WorkShare). MinWorkShare 0 turns the floor off, and the
// agents hold money and are not hungry, so the survival rule does not apply. Both
// households of weight 1 rest: Free 2, Rested 2.
//
// Requirement: FR-OBS-016, FR-AGT-072
func TestWork_RestedHouseholdsAreCounted(t *testing.T) {
	t.Parallel()
	c := millConfig(1, 2)
	c.Roles[0].MinWorkShare = 0
	rec := &evLog{}
	w := newMill(t, c, WithObserver(rec))
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		w.agents[i].history.Push(welfare.Record{Income: 1_000_000, Cost: 1, Working: true, WorkHH: 1})
	}
	step(t, w, 1)
	got := of[ProductionHalted](rec)
	if len(got) != 1 || got[0].Free != 2 || got[0].Rested != 2 || got[0].Halted != 0 || got[0].NoRecipe != 0 {
		t.Errorf("ProductionHalted = %+v, want one with Free 2, Rested 2, Halted 0, NoRecipe 0", got)
	}
}

// TestWork_EventOnlyCountersAreBuiltOnlyWithAnObserver verifies the flag of
// observability rule 4.5 for the counters of P2.5: with no observer supplied the
// halt, rest, no-recipe and skip counts that exist only to build ProductionHalted
// stay 0, while the free and floor-held counts that the Warn reads are filled; with
// an observer the halt and skip counts are filled.
//
// Arithmetic: the halt config of the refusal test halts both households of weight
// 1 (OutputTarget 3 flour, 9 flour held plus the halt rule): Free 2, Halted 2, one
// skipped row of 2 households for the recipe.
//
// Requirement: NFR-OBS-003
// Guard: an event-only counter built without an observer costs work on the default
// path; the call-site allocation test would not notice a counter that fits in
// scratch the world already holds.
func TestWork_EventOnlyCountersAreBuiltOnlyWithAnObserver(t *testing.T) {
	t.Parallel()
	haltConfig := func() Config {
		c := millConfig(1, 2)
		c.World.OutputCoverTicks = 1
		c.Roles[0].StartStock = []Item{{Commodity: "grain", Quantity: 10}, {Commodity: "flour", Quantity: 9}}
		return c
	}
	for _, tt := range []struct {
		name             string
		opts             []Option
		wantHalted, skip int64
	}{
		{"no observer", nil, 0, 0},
		{"an observer", []Option{WithObserver(&evLog{})}, 2, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := newMill(t, haltConfig(), tt.opts...)
			if err := w.beginStep(); err != nil {
				t.Fatalf("beginStep() error = %v, want nil", err)
			}
			w.p02()
			if err := w.p11Moment(); err != nil {
				t.Fatalf("p11Moment() error = %v, want nil", err)
			}
			w.p12Anchors()
			w.p21Complete()
			if err := w.p25Work(); err != nil {
				t.Fatalf("p25Work() error = %v, want nil", err)
			}
			var free, halted, skips int64
			for _, c := range w.pr.cells {
				free += c.free
				halted += c.halted
			}
			for _, n := range w.pr.skips {
				skips += n
			}
			if free != 2 {
				t.Errorf("free households counted = %d, want 2: the Warn reads them with or without an observer", free)
			}
			if halted != tt.wantHalted || skips != tt.skip {
				t.Errorf("halted, skipped households counted = %d, %d, want %d, %d", halted, skips, tt.wantHalted, tt.skip)
			}
		})
	}
}

// TestWorkDraws verifies which Work implementations take a random word, so the
// world reseeds the stream for them and skips the reseed for the ones that take
// none: the default strategy takes none, a consumer's strategy does, and a
// Directed takes one only when its fallback does (a Directed with no fallback
// falls back to the default).
//
// Requirement: FR-WLD-062
// Guard: reseeding for a strategy that takes no word, or not reseeding for one
// that does, moves the stream of every later draw and breaks the golden output.
func TestWorkDraws(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		s    Strategy
		want bool
	}{
		{"default", DefaultStrategy{}, false},
		{"consumer", &tapStrategy{}, true},
		{"directed with no fallback", Directed{}, false},
		{"directed over the default", Directed{Fallback: DefaultStrategy{}}, false},
		{"directed over a consumer", Directed{Fallback: &tapStrategy{}}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := workDraws(tt.s); got != tt.want {
				t.Errorf("workDraws(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestP28a_HaltAndNoRecipeRowsAreCountedPerRegionAndRole verifies the aggregates of
// P2.8a on the two-region world of removalConfig when the output target is met: one
// ProductionHalted for each (region, role) with free households, in ascending
// region and then role order, each with the households of its own cell.
//
// Arithmetic: every miller holds 9 flour per household and OutputCoverTicks 1, so
// the halt acts for all of them (as in TestWork_RefusalsAndTheHaltAreCounted...).
// Home miller: agents 2 (weight 2) and 3 (weight 1), Free 3, Halted 3, one skipped
// row of the mill recipe for 3 households. Home weaver: agent 4 (weight 1), whose
// role has no recipe: Free 1, NoRecipe 1, nothing halted. Yonder miller: agent 5
// (weight 2), Free 2, Halted 2, skipped row for 2 households. Yonder's weaver has no
// agent and no event.
//
// Requirement: FR-OBS-016, FR-AGT-072, FR-WLD-046
// Guard: the cell of a (region, role) is region * roles + role and the skip row of
// a recipe is indexed by cell * recipes + recipe; a scaled index puts one region's
// households in another's event.
func TestP28a_HaltAndNoRecipeRowsAreCountedPerRegionAndRole(t *testing.T) {
	t.Parallel()
	c := removalConfig()
	c.World.OutputCoverTicks = 1
	c.Roles[0].StartStock = []Item{{Commodity: "grain", Quantity: 10}, {Commodity: "flour", Quantity: 9}, {Commodity: "pan", Quantity: 1}}
	rec := &evLog{}
	w := newMill(t, c, WithObserver(rec))
	rec.events = nil
	step(t, w, 1)
	want := []ProductionHalted{
		{Tick: 0, Region: 0, Role: 0, Free: 3, Halted: 3, Skipped: []HaltRow{{Recipe: 0, Households: 3}}},
		{Tick: 0, Region: 0, Role: 1, Free: 1, NoRecipe: 1},
		{Tick: 0, Region: 1, Role: 0, Free: 2, Halted: 2, Skipped: []HaltRow{{Recipe: 0, Households: 2}}},
	}
	got := of[ProductionHalted](rec)
	if len(got) != len(want) {
		t.Fatalf("ProductionHalted events = %+v, want %d", got, len(want))
	}
	for i := range want {
		if !haltEqual(got[i], want[i]) {
			t.Errorf("ProductionHalted[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestP28a_RefusalsAreCountedPerRegionAndRecipe verifies that RecipeRefused is
// emitted once for each region in which a request was refused, in ascending region
// order, with the households refused in that region and no other.
//
// Arithmetic: a consumer strategy asks every miller for the recipe "bake" (100
// grain, on the role's list), which no household can start: floor(10 grain * weight
// / 100) = 0 executions for weights 2 and 1. Home has agents 2 (weight 2) and 3
// (weight 1): NoInput 3 for bake. Yonder has agent 5 (weight 2): NoInput 2. The
// weaver requests nothing.
//
// Requirement: FR-PRO-048, FR-OBS-095
// Guard: the refusal tally is indexed by region and recipe; a scaled index would
// report yonder's refusals under home.
func TestP28a_RefusalsAreCountedPerRegionAndRecipe(t *testing.T) {
	t.Parallel()
	c := removalConfig()
	c.Recipes = append(c.Recipes, Recipe{
		ID: "bake", Duration: 1, TraineeYield: 5_000,
		Inputs:  []Item{{Commodity: "grain", Quantity: 100}},
		Outputs: []Output{{Commodity: "flour", Quantity: 1, CostShare: 10_000}},
	})
	c.Roles[0].Recipes = []string{"mill", "bake"}
	c.Roles[0].Strategy = "tap"
	bake, _ := c.canonical().recipeID("bake")
	s := &tapStrategy{onWork: func(*AgentView) (JobRequest, bool) { return JobRequest{Recipe: bake, Executions: 1}, true }}
	rec := &evLog{}
	w := newMill(t, c, WithObserver(rec), WithStrategy("tap", s))
	rec.events = nil
	step(t, w, 1)
	want := []RecipeRefused{
		{Tick: 0, Region: 0, Recipe: bake, NoInput: 3},
		{Tick: 0, Region: 1, Recipe: bake, NoInput: 2},
	}
	if got := of[RecipeRefused](rec); !slices.Equal(got, want) {
		t.Errorf("RecipeRefused events = %+v, want %+v", got, want)
	}
}

// TestRefuse_CountsOnlyARefusalOfARecipeOfTheConfiguration verifies the guard of the
// refusal tally: a refusal is counted for the households of the agent when an
// observer was supplied, the reason is not empty and the recipe is one of the
// configuration, and it is not counted for a recipe below 0, a recipe at the count
// of recipes, an empty reason, or when no observer was supplied; the Debug line is
// written for every refusal.
//
// Arithmetic: the mill world has 1 recipe, so the valid ID is 0 and -1 and 1 are
// the edges just outside. Agent 2 has weight 2 in the removal world, so one counted
// refusal is 2 households.
//
// Requirement: FR-PRO-048, FR-OBS-016
// Guard: a recipe outside the configuration has no row in the tally; counting it
// would index past the table.
func TestRefuse_CountsOnlyARefusalOfARecipeOfTheConfiguration(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	lg := newCapture()
	w := newMill(t, removalConfig(), WithObserver(rec), WithLogger(slog.New(lg)))
	a := &w.agents[2]
	if a.weight != 2 {
		t.Fatalf("agent 2 weight = %d, want 2", a.weight)
	}
	for _, tt := range []struct {
		recipe RecipeID
		reason Reason
	}{{-1, ReasonNoInput}, {1, ReasonNoInput}, {0, ""}} {
		w.refuse(a, 0, tt.recipe, tt.reason)
	}
	w.refuse(a, 0, 0, ReasonNoInput)
	var got []RecipeRefused
	w.refused.flush(0, func(e RecipeRefused) { got = append(got, e) })
	if want := []RecipeRefused{{Tick: 0, Region: 0, Recipe: 0, NoInput: 2}}; !slices.Equal(got, want) {
		t.Errorf("refusals flushed = %+v, want %+v: only the last call is a counted refusal", got, want)
	}
	if n := len(lg.take("recipe not started")); n != 4 {
		t.Errorf("Debug lines \"recipe not started\" = %d, want 4, one for each call", n)
	}

	quiet := newMill(t, removalConfig())
	quiet.refuse(&quiet.agents[2], 0, 0, ReasonNoInput)
	var none []RecipeRefused
	quiet.refused.flush(0, func(e RecipeRefused) { none = append(none, e) })
	if len(none) != 0 {
		t.Errorf("refusals flushed with no observer = %+v, want none: the tally is built only under the flag", none)
	}
}
