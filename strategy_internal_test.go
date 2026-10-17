package economysim

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"math/rand/v2"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// fakeStrategy is a hand-written Strategy for the tests of Directed
// (architecture.md 10.3): it returns what it was built with and records its
// calls.
type fakeStrategy struct {
	job     JobRequest
	intents []Intent
	discard []Discard
	belief  Belief
	calls   struct{ work, trade, revise int }
	rng     struct{ work, trade, revise *rand.Rand } // the source each method last received
}

func newFake() *fakeStrategy { return &fakeStrategy{} }

func (f *fakeStrategy) Work(_ *AgentView, rng *rand.Rand) JobRequest {
	f.calls.work++
	f.rng.work = rng
	return f.job
}

func (f *fakeStrategy) Trade(_ *AgentView, rng *rand.Rand, p *TradePlan) {
	f.calls.trade++
	f.rng.trade = rng
	p.Intents = append(p.Intents, f.intents...)
	p.Discards = append(p.Discards, f.discard...)
}

func (f *fakeStrategy) Revise(_ *OutcomeView, rng *rand.Rand) Belief {
	f.calls.revise++
	f.rng.revise = rng
	return f.belief
}

// TestWithStrategy_Rules verifies each rule of the registration: a name that
// is not an ID is ErrInvalidID, a built-in or repeated name is ErrDuplicateID
// and a nil extension is ErrNilExtension, checked in that order, and a
// registered strategy is found under its name.
//
// Requirement: FR-AGT-030, FR-WLD-002
func TestWithStrategy_Rules(t *testing.T) {
	t.Parallel()
	s := newFake()
	tests := []struct {
		name string
		opts []Option
		want error
	}{
		{"valid", []Option{WithStrategy("custom", s)}, nil},
		{"empty name", []Option{WithStrategy("", s)}, ErrInvalidID},
		{"upper case name", []Option{WithStrategy("A", s)}, ErrInvalidID},
		{"leading digit", []Option{WithStrategy("1x", s)}, ErrInvalidID},
		{"name too long", []Option{WithStrategy(strings.Repeat("a", MaxIDLength+1), s)}, ErrInvalidID},
		{"built-in default", []Option{WithStrategy("default", s)}, ErrDuplicateID},
		{"built-in directed", []Option{WithStrategy("directed", s)}, ErrDuplicateID},
		{"given twice", []Option{WithStrategy("custom", s), WithStrategy("custom", s)}, ErrDuplicateID},
		{"nil extension", []Option{WithStrategy("custom", nil)}, ErrNilExtension},
		{"invalid name wins over nil", []Option{WithStrategy("Bad", nil)}, ErrInvalidID},
		{"built-in name wins over nil", []Option{WithStrategy("default", nil)}, ErrDuplicateID},
		{"duplicate wins over nil", []Option{WithStrategy("custom", s), WithStrategy("custom", nil)}, ErrDuplicateID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			o, err := applyOptions(tt.opts)
			if tt.want == nil && err != nil || tt.want != nil && !errors.Is(err, tt.want) {
				t.Fatalf("applyOptions(%s) = %v, want %v", tt.name, err, tt.want)
			}
			if tt.want == nil {
				if got, ok := o.strategies.find("custom"); !ok || got != Strategy(s) {
					t.Errorf("strategies.find(custom) = %v, %v, want the registered strategy", got, ok)
				}
			}
		})
	}
}

// wrapsDirected holds a Directed in a named field, as a consumer's wrapper
// does.
type wrapsDirected struct{ inner Directed }

func (w wrapsDirected) Work(v *AgentView, r *rand.Rand) JobRequest { return w.inner.Work(v, r) }
func (w wrapsDirected) Trade(v *AgentView, r *rand.Rand, p *TradePlan) {
	w.inner.Trade(v, r, p)
}
func (w wrapsDirected) Revise(o *OutcomeView, r *rand.Rand) Belief { return w.inner.Revise(o, r) }

// TestIsDirected verifies that a Directed is directed and that a wrapper that
// holds one in a field, a pointer to one and another strategy are not.
//
// Requirement: FR-AGT-030
func TestIsDirected(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s    Strategy
		want bool
	}{
		{"zero value", Directed{}, true},
		{"with fallback", Directed{Fallback: newFake()}, true},
		{"wrapper holding one", wrapsDirected{}, false},
		{"other strategy", newFake(), false},
		{"nil", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isDirected(tt.s); got != tt.want {
				t.Errorf("isDirected(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestDirected_MethodSet verifies that Directed has exactly the three methods
// of Strategy and no others.
//
// Requirement: FR-AGT-019
func TestDirected_MethodSet(t *testing.T) {
	t.Parallel()
	typ := reflect.TypeOf(Directed{})
	if typ.NumMethod() != 3 {
		t.Fatalf("Directed has %d methods, want 3", typ.NumMethod())
	}
	for i, want := range []string{"Revise", "Trade", "Work"} {
		if got := typ.Method(i).Name; got != want {
			t.Errorf("Directed method %d = %q, want %q", i, got, want)
		}
	}
	if iface := reflect.TypeOf((*Strategy)(nil)).Elem(); iface.NumMethod() != 3 {
		t.Errorf("Strategy has %d methods, want 3", iface.NumMethod())
	}
}

// TestDirected_Work verifies that the directive's job wins, a nil job falls
// back, a NoJob request requests nothing without asking the fallback, and a
// nil fallback requests nothing without a directive.
//
// Requirement: FR-AGT-054, FR-AGT-055
func TestDirected_Work(t *testing.T) {
	t.Parallel()
	fb := JobRequest{Recipe: 4, Executions: 9}
	tests := []struct {
		name      string
		fallback  bool
		directive *Directive
		want      JobRequest
		calls     int
	}{
		{"directive job wins", true, &Directive{Job: &JobRequest{Recipe: 2, Executions: 3}}, JobRequest{Recipe: 2, Executions: 3}, 0},
		{"nil job falls back", true, &Directive{}, fb, 1},
		{"no directive falls back", true, nil, fb, 1},
		{"NoJob requests nothing", true, &Directive{Job: &JobRequest{NoJob: true}}, JobRequest{NoJob: true}, 0},
		{"nil fallback without directive", false, nil, JobRequest{NoJob: true}, 0},
		{"nil fallback with nil job", false, &Directive{}, JobRequest{NoJob: true}, 0},
		{"nil fallback with job", false, &Directive{Job: &JobRequest{Recipe: 1}}, JobRequest{Recipe: 1}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFake()
			f.job = fb
			d := Directed{}
			if tt.fallback {
				d.Fallback = f
			}
			v := &AgentView{}
			if tt.directive != nil {
				v.HasDirective, v.Directive = true, *tt.directive
			}
			rng := rand.New(rand.NewPCG(1, 2))
			if got := d.Work(v, rng); got != tt.want {
				t.Errorf("Directed.Work(%s) = %+v, want %+v", tt.name, got, tt.want)
			}
			if f.calls.work != tt.calls {
				t.Errorf("fallback Work called %d times, want %d", f.calls.work, tt.calls)
			}
			if tt.calls == 1 && f.rng.work != rng {
				t.Errorf("fallback Work received rng %p, want the world's %p", f.rng.work, rng)
			}
		})
	}
}

func bid(c CommodityID, q Quantity) Intent {
	return Intent{Commodity: c, Side: Bid, Quantity: q, Limit: 10}
}

// TestDirected_Trade verifies the plan of Directed: the fallback's plan, less
// the intents of the commodities in Hold, with the directive's intent in place
// of the fallback's for the same commodity, the directive's other intents in
// canonical commodity order and its discards appended. Intents are told apart
// by their Quantity.
//
// Requirement: FR-AGT-054, FR-AGT-055
func TestDirected_Trade(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		fallback *fakeStrategy // nil means a nil Fallback
		pre      []Intent      // already in the plan before the call
		dir      *Directive
		want     []Intent
		wantDisc []Discard
	}{
		{
			name:     "no directive leaves the plan",
			fallback: &fakeStrategy{intents: []Intent{bid(1, 1), bid(3, 2)}},
			want:     []Intent{bid(1, 1), bid(3, 2)},
		},
		{
			// Fallback plans 1, 3, 5. The directive replaces 3 with 30, adds 4
			// (between 3 and 5) and holds 1, which removes the fallback's 1.
			name:     "replace, add in order and hold",
			fallback: &fakeStrategy{intents: []Intent{bid(1, 1), bid(3, 2), bid(5, 3)}},
			dir:      &Directive{Intents: []Intent{bid(3, 30), bid(4, 40)}, Hold: []CommodityID{1}},
			want:     []Intent{bid(3, 30), bid(4, 40), bid(5, 3)},
		},
		{
			// Hold removes only the fallback's intent: the directive's own
			// intent for the held commodity 2 survives, in the fallback's place.
			name:     "directive intent for a held commodity survives",
			fallback: &fakeStrategy{intents: []Intent{bid(2, 1)}},
			dir:      &Directive{Intents: []Intent{bid(2, 20)}, Hold: []CommodityID{2}},
			want:     []Intent{bid(2, 20)},
		},
		{
			name:     "directive intent for a held commodity the fallback did not plan is added",
			fallback: &fakeStrategy{intents: []Intent{bid(1, 1)}},
			dir:      &Directive{Intents: []Intent{bid(2, 20)}, Hold: []CommodityID{2}},
			want:     []Intent{bid(1, 1), bid(2, 20)},
		},
		{
			name:     "intents before and after the fallback's range",
			fallback: &fakeStrategy{intents: []Intent{bid(3, 1)}},
			dir:      &Directive{Intents: []Intent{bid(5, 50), bid(1, 10)}},
			want:     []Intent{bid(1, 10), bid(3, 1), bid(5, 50)},
		},
		{
			// Two directive intents for one commodity the fallback did not
			// plan are both added, adjacent, so that the admission refuses the
			// second as duplicate_order instead of it vanishing here.
			name:     "repeated directive intent stays for the admission",
			fallback: &fakeStrategy{intents: []Intent{bid(1, 1)}},
			dir:      &Directive{Intents: []Intent{bid(2, 20), bid(2, 21)}},
			want:     []Intent{bid(1, 1), bid(2, 20), bid(2, 21)},
		},
		{
			name:     "discards are appended after the fallback's",
			fallback: &fakeStrategy{discard: []Discard{{Commodity: 1, Quantity: 1}}},
			dir:      &Directive{Discards: []Discard{{Commodity: 2, Quantity: 2}}},
			want:     nil,
			wantDisc: []Discard{{Commodity: 1, Quantity: 1}, {Commodity: 2, Quantity: 2}},
		},
		{
			name: "nil fallback plans only the directive",
			dir:  &Directive{Intents: []Intent{bid(4, 4), bid(2, 2)}, Hold: []CommodityID{9}},
			want: []Intent{bid(2, 2), bid(4, 4)},
		},
		{
			name: "nil fallback without directive plans nothing",
			want: nil,
		},
		{
			// An intent already in the plan is not the fallback's: neither Hold
			// nor a replacement touches it.
			name:     "intents already in the plan are left alone",
			fallback: &fakeStrategy{intents: []Intent{bid(3, 1)}},
			pre:      []Intent{bid(3, 99)},
			dir:      &Directive{Intents: []Intent{bid(3, 30)}, Hold: []CommodityID{3}},
			want:     []Intent{bid(3, 99), bid(3, 30)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := Directed{}
			if tt.fallback != nil {
				d.Fallback = tt.fallback
			}
			v := &AgentView{}
			if tt.dir != nil {
				v.HasDirective, v.Directive = true, *tt.dir
			}
			p := &TradePlan{Intents: append([]Intent(nil), tt.pre...)}
			rng := rand.New(rand.NewPCG(1, 2))
			d.Trade(v, rng, p)
			if !reflect.DeepEqual(emptyToNil(p.Intents), emptyToNil(tt.want)) {
				t.Errorf("Directed.Trade(%s) intents = %+v, want %+v", tt.name, p.Intents, tt.want)
			}
			if !reflect.DeepEqual(emptyToNilD(p.Discards), emptyToNilD(tt.wantDisc)) {
				t.Errorf("Directed.Trade(%s) discards = %+v, want %+v", tt.name, p.Discards, tt.wantDisc)
			}
			if tt.fallback != nil && tt.fallback.calls.trade != 1 {
				t.Errorf("fallback Trade called %d times, want 1", tt.fallback.calls.trade)
			}
			if tt.fallback != nil && tt.fallback.rng.trade != rng {
				t.Errorf("fallback Trade received rng %p, want the world's %p", tt.fallback.rng.trade, rng)
			}
		})
	}
}

func emptyToNil(in []Intent) []Intent {
	if len(in) == 0 {
		return nil
	}
	return in
}

func emptyToNilD(in []Discard) []Discard {
	if len(in) == 0 {
		return nil
	}
	return in
}

// TestDirected_TradeDoesNotChangeTheDirective verifies that Directed reads the
// directive without writing it, so the same view gives the same plan twice.
//
// Guard: the directive may be Sticky and so read again next tick; a Trade that
// compacted or sorted the directive's slices in place would change what the
// player asked for.
func TestDirected_TradeDoesNotChangeTheDirective(t *testing.T) {
	t.Parallel()
	dir := Directive{Intents: []Intent{bid(5, 5), bid(2, 2)}, Hold: []CommodityID{7, 1}, Discards: []Discard{{Commodity: 1, Quantity: 1}}}
	want := Directive{Intents: []Intent{bid(5, 5), bid(2, 2)}, Hold: []CommodityID{7, 1}, Discards: []Discard{{Commodity: 1, Quantity: 1}}}
	f := newFake()
	f.intents = []Intent{bid(1, 1), bid(5, 9)}
	v := &AgentView{HasDirective: true, Directive: dir}
	var first, second TradePlan
	Directed{Fallback: f}.Trade(v, nil, &first)
	Directed{Fallback: f}.Trade(v, nil, &second)
	if !reflect.DeepEqual(v.Directive, want) {
		t.Errorf("directive after Trade = %+v, want it unchanged %+v", v.Directive, want)
	}
	if !reflect.DeepEqual(first, second) {
		t.Errorf("two calls gave %+v and %+v, want equal plans", first, second)
	}
}

// TestDirected_Revise verifies that Revise is the fallback's, and that a nil
// fallback keeps the belief.
//
// Requirement: FR-AGT-054
func TestDirected_Revise(t *testing.T) {
	t.Parallel()
	in := Belief{Centre: 100, Width: 500}
	out := Belief{Centre: 120, Width: 400}
	t.Run("fallback decides", func(t *testing.T) {
		t.Parallel()
		f := newFake()
		f.belief = out
		rng := rand.New(rand.NewPCG(1, 2))
		if got := (Directed{Fallback: f}).Revise(&OutcomeView{Belief: in}, rng); got != out || f.calls.revise != 1 {
			t.Errorf("Directed.Revise = %+v after %d fallback calls, want %+v after 1", got, f.calls.revise, out)
		}
		if f.rng.revise != rng {
			t.Errorf("fallback Revise received rng %p, want the world's %p", f.rng.revise, rng)
		}
	})
	t.Run("nil fallback keeps the belief", func(t *testing.T) {
		t.Parallel()
		if got := (Directed{}).Revise(&OutcomeView{Belief: in}, nil); got != in {
			t.Errorf("Directed.Revise with nil Fallback = %+v, want %+v", got, in)
		}
	})
}

// TestDirected_TradeAllocatesNothingOnAWarmedPlan verifies that Directed.Trade
// adds no allocation of its own when the plan's slices have room.
//
// Requirement: NFR-AGT-001
func TestDirected_TradeAllocatesNothingOnAWarmedPlan(t *testing.T) {
	f := newFake()
	f.intents = []Intent{bid(1, 1), bid(3, 2), bid(5, 3)}
	v := &AgentView{HasDirective: true, Directive: Directive{
		Intents: []Intent{bid(3, 30), bid(4, 40)}, Hold: []CommodityID{1},
		Discards: []Discard{{Commodity: 2, Quantity: 2}},
	}}
	d := Directed{Fallback: f}
	p := &TradePlan{Intents: make([]Intent, 0, 16), Discards: make([]Discard, 0, 16)}
	allocs := testing.AllocsPerRun(100, func() {
		p.Intents, p.Discards = p.Intents[:0], p.Discards[:0]
		d.Trade(v, nil, p)
	})
	// The fake's own append is within the plan's capacity, so the whole call
	// allocates nothing.
	if allocs != 0 {
		t.Errorf("Directed.Trade allocated %v times per run on a warmed plan, want 0", allocs)
	}
}

// TestStrategyFiles_Contract verifies that strategy.go and agent.go import no
// clock, no global or crypto randomness and no sync or goroutine primitive, and
// start no goroutine.
//
// Guard: architecture.md 7.3 and 12.2. A strategy file that read the clock or
// started a goroutine would break the reproducibility of every run; the module
// guard of determinism_guard_test.go covers the calls, this one the imports and
// go statements of the two files that define the extension point.
func TestStrategyFiles_Contract(t *testing.T) {
	t.Parallel()
	banned := []string{"time", "crypto/rand", "math/rand", "sync", "os"}
	for _, name := range []string{"strategy.go", "agent.go"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			src, err := os.ReadFile(name)
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}
			f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.SkipObjectResolution)
			if err != nil {
				t.Fatalf("parse %s: %v", name, err)
			}
			for _, imp := range f.Imports {
				path, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					t.Fatalf("unquote %s: %v", imp.Path.Value, err)
				}
				for _, b := range banned {
					if path == b {
						t.Errorf("%s imports %q, which architecture.md 7.3 and 12.2 do not allow here", name, path)
					}
				}
			}
			ast.Inspect(f, func(n ast.Node) bool {
				if g, ok := n.(*ast.GoStmt); ok {
					t.Errorf("%s starts a goroutine at %v", name, g.Pos())
				}
				return true
			})
		})
	}
}
