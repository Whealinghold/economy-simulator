package economysim

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// packOneCommodities are the nine commodities of the pack "one" in canonical
// order (spec-ledger.md 3.4.3), with the shelf life and durability that the
// ledger table reads. Only the three fields newLedger reads are set.
func packOneCommodities() []Commodity {
	ids := []string{"fuel", "mineral", "prepared", "refined", "seed", "staple", "tool_a", "tool_b", "waste"}
	out := make([]Commodity, len(ids))
	for i, id := range ids {
		out[i].ID = id
	}
	out[2].ShelfLife = 4   // prepared
	out[5].ShelfLife = 700 // staple
	out[6].Durability = 10 // tool_a
	out[7].Durability = 8  // tool_b
	return out
}

// mustLedger builds the ledger of a configuration or stops the test.
func mustLedger(t *testing.T, cfg Config) *ledger.Ledger {
	t.Helper()
	l, err := newLedger(cfg)
	if err != nil {
		t.Fatalf("newLedger: %v", err)
	}
	return l
}

// readSpec reads a file of the repository by a path relative to the package.
func readSpec(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// TestNewLedger_CommodityTable verifies that the ledger is built with the
// shelf life and durability of each commodity in commodity-ID order: a lot of
// a spoiling commodity is born in the current tick, an item that wears starts
// at its full durability, and any other lot has neither.
//
// Requirement: FR-LED-013
func TestNewLedger_CommodityTable(t *testing.T) {
	t.Parallel()
	cfg := Config{
		Commodities: []Commodity{{ID: "a", ShelfLife: 3}, {ID: "b", Durability: 5}, {ID: "c"}},
		Regions:     []Region{{ID: "north"}},
	}
	l := mustLedger(t, cfg)
	agent, err := l.Open(ledger.KindAgent, 0, 0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := l.EndConstruction(); err != nil {
		t.Fatalf("EndConstruction: %v", err)
	}
	l.AdvanceTick()
	l.AdvanceTick() // the current tick is 2
	for c := range cfg.Commodities {
		from, ok := l.Boundary(ledger.Initial, c)
		if !ok {
			t.Fatalf("Boundary(Initial, %d) = none, want an account", c)
		}
		if err := l.Move(from, agent, c, 1, ledger.Initial); err != nil {
			t.Fatalf("Move(commodity %d): %v", c, err)
		}
	}
	got, err := l.AppendLots(nil, agent)
	if err != nil {
		t.Fatalf("AppendLots: %v", err)
	}
	// a spoils: born in tick 2. b wears: born 0, durability 5. c neither.
	want := []ledger.Lot{
		{Commodity: 0, Quantity: 1, Born: 2},
		{Commodity: 1, Quantity: 1, Durability: 5},
		{Commodity: 2, Quantity: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("lots of the agent = %+v, want %+v", got, want)
	}
}

// TestNewLedger_Refusals verifies that a configuration the ledger cannot use is
// returned as ErrInvalidConfig, wrapped, and no ledger is built.
//
// Guard: the root hands the ledger numbers it derived from a validated
// configuration, so these cannot occur through a world; the test keeps the
// error wrapped, so a caller can still match it.
func TestNewLedger_Refusals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  Config
	}{
		{"no region", Config{Commodities: []Commodity{{ID: "a"}}}},
		{"negative shelf life", Config{Commodities: []Commodity{{ID: "a", ShelfLife: -1}}, Regions: []Region{{ID: "r"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			l, err := newLedger(tt.cfg)
			if !errors.Is(err, ledger.ErrInvalidConfig) {
				t.Errorf("newLedger(%s) error = %v, want ErrInvalidConfig", tt.name, err)
			}
			if l != nil {
				t.Errorf("newLedger(%s) returned a ledger with its error", tt.name)
			}
		})
	}
}

// TestNewLedger_BoundaryAccountOrder verifies that the boundary accounts of
// the built-in group are created in the order of spec-ledger.md 3.4.2 (money
// first, then commodity-major, the goods reasons in registry order), that
// their count is the B of 3.4.1 for the pack "one" and that the account after
// them is ID B, which is the estate of its region. The expected numbers are
// read from the spec's worked count, not written here.
//
// Requirement: FR-LED-052, FR-LED-008
func TestNewLedger_BoundaryAccountOrder(t *testing.T) {
	t.Parallel()
	spec := readSpec(t, "doc/specs/spec-ledger.md")

	// "B = 1 + 7 * 9 = 64": nine commodities, so B is 64.
	m := regexp.MustCompile(`B = 1 \+ 7 \* (\d+) = (\d+)`).FindAllStringSubmatch(spec, -1)
	if len(m) == 0 {
		t.Fatalf("worked count of B not found in spec-ledger.md 3.4.3")
	}
	last := m[len(m)-1]
	wantC, _ := strconv.Atoi(last[1])
	wantB, _ := strconv.Atoi(last[2])

	cfg := Config{Commodities: packOneCommodities(), Regions: []Region{{ID: "north"}}}
	if len(cfg.Commodities) != wantC {
		t.Fatalf("pack one has %d commodities, spec-ledger.md 3.4.3 says %d", len(cfg.Commodities), wantC)
	}
	l := mustLedger(t, cfg)
	if got := l.Len(); got != wantB {
		t.Errorf("accounts after newLedger = %d, want B = %d", got, wantB)
	}
	for id := 0; id < l.Len(); id++ {
		v, err := l.Account(id)
		if err != nil || v.Kind != ledger.KindBoundary {
			t.Errorf("Account(%d) = %+v, %v, want a boundary account", id, v, err)
		}
	}

	// Money first: account 0 is initial money.
	if id, ok := l.Boundary(ledger.Initial, ledger.Money); !ok || id != 0 {
		t.Errorf("Boundary(Initial, Money) = %d, %v, want 0, true", id, ok)
	}
	// Commodity-major, 7 reasons each: reason r of commodity c is 1 + 7c + r.
	order := []ledger.Reason{ledger.Initial, ledger.RecipeInput, ledger.RecipeOutput,
		ledger.Disposal, ledger.Consumption, ledger.Spoilage, ledger.Wear}
	for c := range cfg.Commodities {
		for r, reason := range order {
			want := 1 + 7*c + r
			if id, ok := l.Boundary(reason, c); !ok || id != want {
				t.Errorf("Boundary(%d, commodity %d) = %d, %v, want %d", reason, c, id, ok, want)
			}
		}
	}

	// The worked examples of 3.4.3: "`recipe_output` of `staple` is
	// `1 + 35 + 2 = 38`".
	exRe := regexp.MustCompile("`([a-z_]+)` of `([a-z_]+)` is `1 \\+ (\\d+) \\+ (\\d+) = (\\d+)`")
	examples := exRe.FindAllStringSubmatch(spec, -1)
	if len(examples) < 2 {
		t.Fatalf("worked examples of 3.4.3 not found: got %d", len(examples))
	}
	for _, e := range examples {
		reason, ok := ledgerReason(Reason(e[1]))
		c := slices.IndexFunc(cfg.Commodities, func(x Commodity) bool { return x.ID == e[2] })
		want, _ := strconv.Atoi(e[5])
		if !ok || c < 0 {
			t.Errorf("example %q names an unknown reason or commodity", e[0])
			continue
		}
		if id, found := l.Boundary(reason, c); !found || id != want {
			t.Errorf("Boundary(%s, %s) = %d, %v, want %d (spec)", e[1], e[2], id, found, want)
		}
	}

	// The estate of the region is the next account, ID B.
	if id, err := l.Open(ledger.KindEstate, 0, 0); err != nil || id != wantB {
		t.Errorf("first Open after the boundary accounts = %d, %v, want %d", id, err, wantB)
	}
}

// TestReasonNames_RoundTrip verifies that every built-in ledger reason has a
// distinct, non-empty name, that the name maps back to the same reason, that
// the declared reasons have no name until their module appends one (want_consumption has),
// and that a
// Reason that is not a ledger reason maps to nothing.
//
// Requirement: FR-LED-004
func TestReasonNames_RoundTrip(t *testing.T) {
	t.Parallel()
	seen := map[Reason]ledger.Reason{}
	for r := ledger.Reason(0); r < ledger.NumReasons; r++ {
		name, ok := reasonName(r)
		if r >= ledger.LoanOut && r <= ledger.LoanInterest {
			if ok {
				t.Errorf("reasonName(%d) = %q, want none for a declared reason", r, name)
			}
			continue
		}
		if !ok || name == "" {
			t.Errorf("reasonName(%d) = %q, %v, want a name", r, name, ok)
			continue
		}
		if prev, dup := seen[name]; dup {
			t.Errorf("reasons %d and %d share the name %q", prev, r, name)
		}
		seen[name] = r
		if back, ok := ledgerReason(name); !ok || back != r {
			t.Errorf("ledgerReason(%q) = %d, %v, want %d", name, back, ok, r)
		}
	}
	for _, r := range []Reason{ReasonOverflow, ReasonUnknownAgent, "", "loan_out"} {
		if got, ok := ledgerReason(r); ok {
			t.Errorf("ledgerReason(%q) = %d, true, want none", r, got)
		}
	}
	if _, ok := reasonName(ledger.NumReasons + 3); ok {
		t.Errorf("reasonName of a value above the table = true, want false")
	}
}

// TestReasonNames_EventConstantsHaveLedgerRows verifies, from the source of
// event.go, that every Reason constant declared in the two ledger groups (the
// faucets and sinks, and the internal reasons) maps to a ledger reason, and that
// every built-in ledger reason has a constant in them.
//
// Guard: a Reason constant added to the ledger groups with no row in the ledger
// would name a flow the ledger cannot move, and a row added to the ledger with
// no constant would have a BoundaryFlow row with no name. Either would surface
// only at the first faucet or sink that used it.
func TestReasonNames_EventConstantsHaveLedgerRows(t *testing.T) {
	t.Parallel()
	f, err := parser.ParseFile(token.NewFileSet(), "event.go", nil, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse event.go: %v", err)
	}
	var values []string
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.CONST || g.Doc == nil {
			continue
		}
		doc := g.Doc.Text()
		if !strings.HasPrefix(doc, "The ledger reasons of the flows") && !strings.HasPrefix(doc, "The internal ledger reasons") && !strings.HasPrefix(doc, "The declared ledger reasons") {
			continue
		}
		for _, s := range g.Specs {
			vs := s.(*ast.ValueSpec)
			for _, v := range vs.Values {
				lit, ok := v.(*ast.BasicLit)
				if !ok {
					t.Fatalf("constant %s of a ledger group is not a string literal", vs.Names[0].Name)
				}
				u, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("unquote %s: %v", lit.Value, err)
				}
				values = append(values, u)
			}
		}
	}
	named := 0
	for r := ledger.Reason(0); r < ledger.NumReasons; r++ {
		if _, ok := reasonName(r); ok {
			named++
		}
	}
	if len(values) != named {
		t.Errorf("event.go declares %d ledger reason constants, the ledger has %d named reasons", len(values), named)
	}
	for _, v := range values {
		if _, ok := ledgerReason(Reason(v)); !ok {
			t.Errorf("Reason constant %q has no ledger row", v)
		}
	}
	for r := range int(ledger.NumReasons) {
		name, ok := reasonName(ledger.Reason(r))
		if !ok {
			if r >= numBuiltInReasons && ledger.Reason(r) <= ledger.LoanInterest {
				continue // a loan row: unnamed until the credit module appends one
			}
			t.Errorf("ledger reason %d has no Reason name", r)
			continue
		}
		if !slices.Contains(values, string(name)) {
			t.Errorf("ledger reason %d (%q) has no Reason constant in event.go", r, name)
		}
	}
}

// scriptedFlows runs a small script over a two-region, two-commodity ledger
// and returns the ledger. The construction phase moves initial money and goods
// (the bucket); after EndConstruction a tick moves faucets, sinks and an
// internal trade.
func scriptedFlows(t *testing.T) *ledger.Ledger {
	t.Helper()
	cfg := Config{
		Commodities: []Commodity{{ID: "c0"}, {ID: "c1"}},
		Regions:     []Region{{ID: "r0"}, {ID: "r1"}},
	}
	l := mustLedger(t, cfg)
	open := func(owner, region int) int {
		id, err := l.Open(ledger.KindAgent, owner, region)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		return id
	}
	a0, b0, a1 := open(0, 0), open(1, 0), open(2, 1)
	boundary := func(r ledger.Reason, c int) int {
		id, ok := l.Boundary(r, c)
		if !ok {
			t.Fatalf("Boundary(%d, %d) not found", r, c)
		}
		return id
	}
	do := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("script: %v", err)
		}
	}
	// Construction: the bucket.
	do(l.Transfer(boundary(ledger.Initial, ledger.Money), a0, 100, ledger.Initial))
	do(l.Move(boundary(ledger.Initial, 0), a0, 0, 4, ledger.Initial))
	do(l.Move(boundary(ledger.Initial, 1), a1, 1, 6, ledger.Initial))
	do(l.EndConstruction())
	// The tick.
	do(l.Transfer(a0, b0, 10, ledger.Trade))
	do(l.Move(boundary(ledger.RecipeOutput, 1), a0, 1, 9, ledger.RecipeOutput))
	do(l.Move(a0, boundary(ledger.RecipeInput, 0), 0, 3, ledger.RecipeInput))
	do(l.Move(a0, boundary(ledger.Consumption, 1), 1, 2, ledger.Consumption))
	do(l.Move(a1, boundary(ledger.Spoilage, 1), 1, 1, ledger.Spoilage))
	do(l.Transfer(boundary(ledger.Initial, ledger.Money), a1, 5, ledger.Initial))
	return l
}

// TestBoundaryFlows_Rows verifies the rows built from scripted totals: only
// faucets and sinks (the internal trade is left out), in ascending region, then
// reason in registry order, money before goods, then commodity, with
// NoCommodity for money and a magnitude for the total, and that the
// construction bucket gives the same form with Tick -1.
//
// Requirement: FR-LED-031
func TestBoundaryFlows_Rows(t *testing.T) {
	t.Parallel()
	l := scriptedFlows(t)

	// The tick: region 0 has recipe_input c0 3 (sink), recipe_output c1 9
	// (faucet) and consumption c1 2 (sink); region 1 has initial money 5
	// (faucet) and spoilage c1 1 (sink). The trade of 10 is internal.
	got := boundaryFlows(nil, l.AppendFlows(nil, false), 7, nil)
	want := []BoundaryFlow{
		{Tick: 7, Region: 0, Reason: ReasonRecipeInput, Kind: KindSink, Commodity: 0, Total: 3},
		{Tick: 7, Region: 0, Reason: ReasonRecipeOutput, Kind: KindFaucet, Commodity: 1, Total: 9},
		{Tick: 7, Region: 0, Reason: ReasonConsumption, Kind: KindSink, Commodity: 1, Total: 2},
		{Tick: 7, Region: 1, Reason: ReasonInitial, Kind: KindFaucet, Commodity: NoCommodity, Total: 5},
		{Tick: 7, Region: 1, Reason: ReasonSpoilage, Kind: KindSink, Commodity: 1, Total: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("boundaryFlows(tick totals, 7, nil) =\n %+v\nwant\n %+v", got, want)
	}

	// The bucket: initial money 100 and 4 units of c0 in region 0, 6 units of
	// c1 in region 1, money before goods.
	gotBucket := boundaryFlows(nil, l.AppendFlows(nil, true), constructionTick, nil)
	wantBucket := []BoundaryFlow{
		{Tick: -1, Region: 0, Reason: ReasonInitial, Kind: KindFaucet, Commodity: NoCommodity, Total: 100},
		{Tick: -1, Region: 0, Reason: ReasonInitial, Kind: KindFaucet, Commodity: 0, Total: 4},
		{Tick: -1, Region: 1, Reason: ReasonInitial, Kind: KindFaucet, Commodity: 1, Total: 6},
	}
	if !reflect.DeepEqual(gotBucket, wantBucket) {
		t.Errorf("boundaryFlows(bucket, -1, nil) =\n %+v\nwant\n %+v", gotBucket, wantBucket)
	}

	// A call appends: the rows already in dst stay in front.
	prefix := []BoundaryFlow{{Tick: 1, Total: 1}}
	got = boundaryFlows(prefix, l.AppendFlows(nil, true), -1, nil)
	if len(got) != 1+len(wantBucket) || got[0] != prefix[0] {
		t.Errorf("boundaryFlows appended to a prefix = %+v, want the prefix then %d rows", got, len(wantBucket))
	}
}

// TestBoundaryFlows_NothingMoved verifies that totals that are all zero, and
// totals of internal reasons only, give no row.
//
// Requirement: FR-LED-031
func TestBoundaryFlows_NothingMoved(t *testing.T) {
	t.Parallel()
	l := mustLedger(t, Config{Commodities: []Commodity{{ID: "c"}}, Regions: []Region{{ID: "r"}}})
	if got := boundaryFlows(nil, l.AppendFlows(nil, false), 0, nil); len(got) != 0 {
		t.Errorf("boundaryFlows of empty totals = %+v, want none", got)
	}
	internal := []ledger.Flow{
		{Region: 0, Reason: ledger.Trade, Commodity: ledger.Money, Total: 5},
		{Region: 0, Reason: ledger.Trade, Commodity: 0, Total: 5},
		{Region: 0, Reason: ledger.EstateMoney, Commodity: ledger.Money, Total: 5},
		{Region: 0, Reason: ledger.LoanOut, Commodity: ledger.Money, Total: 5},
	}
	if got := boundaryFlows(nil, internal, 0, nil); len(got) != 0 {
		t.Errorf("boundaryFlows of internal reasons = %+v, want none", got)
	}
}

// TestBoundaryFlows_NoAllocationWithRoom verifies that building the rows into a
// slice with room allocates nothing, so the world can reuse one slice every
// tick, and that it allocates when the slice has none (so the test measures
// something).
//
// Requirement: NFR-LED-001
func TestBoundaryFlows_NoAllocationWithRoom(t *testing.T) {
	l := scriptedFlows(t) // not parallel: AllocsPerRun must not run beside other tests
	flows := l.AppendFlows(nil, false)
	dst := make([]BoundaryFlow, 0, 16)
	if got := testing.AllocsPerRun(100, func() { dst = boundaryFlows(dst[:0], flows, 7, nil) }); got != 0 {
		t.Errorf("boundaryFlows with room allocated %v times per run, want 0", got)
	}
	if len(dst) != 5 {
		t.Errorf("rows = %d, want 5", len(dst))
	}
	var tight []BoundaryFlow
	if got := testing.AllocsPerRun(10, func() { tight = boundaryFlows(nil, flows, 7, nil) }); got == 0 || len(tight) != 5 {
		t.Errorf("boundaryFlows with no room allocated %v times for %d rows, want more than 0 and 5", got, len(tight))
	}
}

// scriptedWear runs a small script over a ledger of one plain and two durable
// commodities: wear in the construction phase, then wear in a tick.
func scriptedWear(t *testing.T) *ledger.Ledger {
	t.Helper()
	cfg := Config{
		Commodities: []Commodity{{ID: "c0"}, {ID: "c1", Durability: 10}, {ID: "c2", Durability: 6}},
		Regions:     []Region{{ID: "r0"}},
	}
	l := mustLedger(t, cfg)
	a, err := l.Open(ledger.KindAgent, 0, 0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	do := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("script: %v", err)
		}
	}
	initial := func(c int) int {
		id, ok := l.Boundary(ledger.Initial, c)
		if !ok {
			t.Fatalf("Boundary(initial, %d) not found", c)
		}
		return id
	}
	do(l.Move(initial(1), a, 1, 3, ledger.Initial))
	do(l.Move(initial(2), a, 2, 1, ledger.Initial))
	do(l.Wear(a, 1, 4)) // construction: 4 units of c1
	do(l.EndConstruction())
	do(l.Wear(a, 1, 3))         // the tick: the worn item of c1 loses 3
	do(l.WearItems(a, 1, 2, 2)) // and the two most worn items of c1 lose 2 each
	do(l.Wear(a, 2, 9))         // c2's one item has 6 left: it is retired with 6 counted
	return l
}

// TestWearApplied_Rows verifies the rows built from the ledger's wear units:
// only commodities with units above 0, in ascending commodity order, with the
// durability actually removed (an item retired with less left than asked
// counts what it had), no region, and the construction bucket in the same form
// with Tick -1.
//
// Requirement: FR-LED-016, FR-OBS-092
func TestWearApplied_Rows(t *testing.T) {
	t.Parallel()
	l := scriptedWear(t)

	// The tick: c1 loses 3 (one item) and 2 + 2 (two items) = 7; c2's item had
	// 6 left and the wear asked for 9, so 6 are counted; c0 does not wear.
	got := wearApplied(nil, l, 3, 7, false)
	want := []WearApplied{
		{Tick: 7, Commodity: 1, Units: 7},
		{Tick: 7, Commodity: 2, Units: 6},
	}
	if !slices.Equal(got, want) {
		t.Errorf("wearApplied(tick, 7) = %+v, want %+v", got, want)
	}

	// The construction bucket: one item of c1 lost 4.
	gotBucket := wearApplied(nil, l, 3, constructionTick, true)
	wantBucket := []WearApplied{{Tick: -1, Commodity: 1, Units: 4}}
	if !slices.Equal(gotBucket, wantBucket) {
		t.Errorf("wearApplied(bucket, -1) = %+v, want %+v", gotBucket, wantBucket)
	}

	// The tick's rows are cleared with the totals at the end of the tick; the
	// bucket stays.
	l.EndTick()
	if got := wearApplied(nil, l, 3, 8, false); len(got) != 0 {
		t.Errorf("wearApplied after EndTick = %+v, want none", got)
	}
	if got := wearApplied(nil, l, 3, constructionTick, true); !slices.Equal(got, wantBucket) {
		t.Errorf("wearApplied(bucket) after EndTick = %+v, want %+v", got, wantBucket)
	}
}

// TestWearApplied_Prefix verifies that the rows are appended after what dst
// already holds, so a world can build the BoundaryFlow and WearApplied rows of a
// tick in one slice order, and that a ledger with no wear adds none.
//
// Guard: a function that reset dst would drop the caller's rows without an error.
func TestWearApplied_Prefix(t *testing.T) {
	t.Parallel()
	l := scriptedWear(t)
	prefix := []WearApplied{{Tick: 7, Commodity: 0, Units: 1}}
	got := wearApplied(prefix, l, 3, 7, false)
	if len(got) != 3 || got[0] != prefix[0] || got[1].Commodity != 1 || got[2].Commodity != 2 {
		t.Errorf("wearApplied(prefix, ...) = %+v, want the prefix then the rows of c1 and c2", got)
	}
	if got := wearApplied(nil, mustLedger(t, Config{
		Commodities: []Commodity{{ID: "c0"}}, Regions: []Region{{ID: "r0"}},
	}), 1, 0, false); len(got) != 0 {
		t.Errorf("wearApplied of a ledger with no wear = %+v, want none", got)
	}
}

// TestWearApplied_NoAllocationWithRoom verifies that the rows are built with no
// allocation when dst has room, as boundaryFlows is.
//
// Guard: the world builds these rows every tick, and the no-op observer path
// must not allocate (observability-and-logging.md 8.2).
func TestWearApplied_NoAllocationWithRoom(t *testing.T) {
	l := scriptedWear(t)
	dst := make([]WearApplied, 0, 8)
	if got := testing.AllocsPerRun(100, func() { dst = wearApplied(dst[:0], l, 3, 7, false) }); got != 0 {
		t.Errorf("wearApplied with room allocated %v times per run, want 0", got)
	}
}

// TestWearApplied_RetirementIsNotTheRate verifies the contract that separates
// WearApplied from BoundaryFlow under the wear reason: the boundary counts the
// items retired, the event the durability removed, so over the script the
// units of a commodity are at least its retired items and, for c1 whose items
// all survive, the boundary has no row while the event does.
//
// Guard: the two are easy to confuse (spec-observability.md 4.4.1). A change
// that fed the wear units from the retirement, or the reverse, would make
// economysim.commodity.wear and economysim.commodity.flow{reason=wear} equal and
// lose the lumpy-versus-smooth distinction that the metric exists for.
func TestWearApplied_RetirementIsNotTheRate(t *testing.T) {
	t.Parallel()
	l := scriptedWear(t)
	retired := map[CommodityID]int64{}
	for _, f := range boundaryFlows(nil, l.AppendFlows(nil, false), 7, nil) {
		if f.Reason == ReasonWear {
			retired[f.Commodity] += f.Total
		}
	}
	// c2's one item was used up (1 item, 6 units); c1's three items survive.
	if want := (map[CommodityID]int64{2: 1}); !maps.Equal(retired, want) {
		t.Errorf("items retired under wear = %v, want %v", retired, want)
	}
	units := map[CommodityID]int64{}
	for _, w := range wearApplied(nil, l, 3, 7, false) {
		units[w.Commodity] = w.Units
	}
	if want := (map[CommodityID]int64{1: 7, 2: 6}); !maps.Equal(units, want) {
		t.Errorf("wear units = %v, want %v", units, want)
	}
	for c, n := range retired {
		if units[c] < n {
			t.Errorf("commodity %d: wear units %d below items retired %d", c, units[c], n)
		}
	}
}
