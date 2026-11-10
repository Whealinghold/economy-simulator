package economysim

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// The tests of the external module's world plumbing (plan 2-16 Phase 5): the
// construction order, the node that is not a market, the refusal rows, the module
// off, the audit and the guard against reads of the realm. They run on shipConfig
// (three regions: a_home 0, b_far 1 and c_isle 2) with c_isle made the outside
// node, joined to a_home by edges both ways, so a_home is its one gate.

// The agents of extShipConfig: the estates are 0 to 2, the counterparties follow
// (the buyer of the gate a_home is 3, its seller 4) and the households from 5.
const (
	extBuyer      AgentID  = 3
	extSeller     AgentID  = 4
	extHousehold  AgentID  = 5
	extFirstHouse          = 5
	extIsle       RegionID = 2
)

// extShipConfig returns shipConfig with c_isle the outside node: it buys flour
// (5 a tick, bid 5 % below the mid of 3 000) from the gate and sells grain (10 a
// tick, ask 5 % above 1 000). The lines are written out of order on purpose.
func extShipConfig(c *Config) {
	c.Edges = append(c.Edges, Edge{From: "c_isle", To: "a_home", Distance: 30}, Edge{From: "a_home", To: "c_isle", Distance: 30})
	out := DefaultOutsideLine("flour", 3_000)
	out.Buy = OutsideSide{MarginBP: 500, Capacity: 5}
	in := DefaultOutsideLine("grain", 1_000)
	in.Sell = OutsideSide{MarginBP: 500, Capacity: 10}
	c.Modules.External = &External{Nodes: []OutsideNode{{Region: "c_isle", Lines: []OutsideLine{out, in}}}}
}

// TestExternal_ConstructionOrder verifies the order of construction of
// spec-external-sector.md 4.2.2 to 4.2.4, 8.6: the estates take agent IDs 0 to 2,
// the counterparties 3 (the buyer) and 4 (the seller), the households start at 5;
// the Outside accounts follow the estate accounts and precede the agents'; the
// boundary accounts of the six reasons come money first, then goods by commodity,
// after the built-in group and before the other modules' (the demand module's
// want_consumption account follows them); the counterparties are not agents: no
// AgentSpawned, in neither count; and the construction bucket has no outside_*
// reason.
//
// Requirement: FR-EXT-005, FR-EXT-011, FR-EXT-012, FR-EXT-018, FR-EXT-045
func TestExternal_ConstructionOrder(t *testing.T) {
	t.Parallel()
	for _, demand := range []bool{false, true} {
		name := "demand off"
		if demand {
			name = "demand on"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			w, rec := newShipWorld(t, func(c *Config) {
				extShipConfig(c)
				if demand {
					c.Modules.Demand = &Demand{}
				}
			}, nil)
			x := w.external
			if w.firstHousehold != extFirstHouse || len(x.cps) != 2 {
				t.Fatalf("firstHousehold = %d with %d counterparties, want %d with 2", w.firstHousehold, len(x.cps), extFirstHouse)
			}
			if cp := x.cps[0]; cp.agent != extBuyer || !cp.buy || cp.node != extIsle || cp.gate != 0 || cp.scaleBP != 10_000 {
				t.Errorf("counterparty 0 = %+v, want the buyer (agent 3) of gate a_home at scale 10000", cp)
			}
			if cp := x.cps[1]; cp.agent != extSeller || cp.buy {
				t.Errorf("counterparty 1 = %+v, want the seller (agent 4)", cp)
			}
			if w.agents[extBuyer].weight != 0 || w.agents[extSeller].weight != 0 || w.agents[extHousehold].weight < 1 {
				t.Errorf("weights of the counterparties and the first household = %d, %d, %d, want 0, 0 and at least 1",
					w.agents[extBuyer].weight, w.agents[extSeller].weight, w.agents[extHousehold].weight)
			}
			// Accounts: estate 0..2, then the two Outside accounts, then the agents.
			last := w.estates.accounts[len(w.estates.accounts)-1]
			for i, cp := range x.cps {
				v, err := w.ledger.Account(cp.account)
				if err != nil || cp.account != last+1+i || v.Kind != ledger.KindOutside || v.Owner != int(cp.agent) || v.Region != 0 {
					t.Errorf("counterparty %d account %d = %+v (%v), want an Outside account %d owned by the agent in the gate region", i, cp.account, v, err, last+1+i)
				}
			}
			if got := w.agents[extHousehold].account; got != last+3 {
				t.Errorf("account of the first household = %d, want %d: right after the Outside accounts", got, last+3)
			}
			b := outsideBoundaryIDs(t, w)
			if !slices.IsSorted(b) || slices.Compact(slices.Clone(b))[len(b)-1] != b[len(b)-1] || len(b) != 9 {
				t.Errorf("boundary account IDs of the six reasons = %v, want 9 ascending: 3 money, then 3 for each of the 2 commodities", b)
			}
			initial, _ := w.ledger.Boundary(ledger.Initial, ledger.Money)
			disposal, _ := w.ledger.Boundary(ledger.Disposal, int(commodityOf(t, w, "grain")))
			if b[0] < initial || b[0] < disposal || b[len(b)-1] > w.estates.accounts[0] {
				t.Errorf("outside boundary IDs %v are not between the built-in group (%d, %d) and the estates (%d)", b, initial, disposal, w.estates.accounts[0])
			}
			if demand {
				want, ok := w.ledger.Boundary(ledger.WantConsumption, int(commodityOf(t, w, "grain")))
				if !ok || want < b[len(b)-1] {
					t.Errorf("want_consumption boundary %d (%v) is not after the outside group ending at %d", want, ok, b[len(b)-1])
				}
			}
			if n := len(of[AgentSpawned](rec)); n != 4 {
				t.Errorf("AgentSpawned events = %d, want 4 (the households only)", n)
			}
			s, err := w.Snapshot()
			if err != nil {
				t.Fatalf("Snapshot() = %v, want nil", err)
			}
			if s.Agents != 4 || s.Households != 4 {
				t.Errorf("snapshot Agents, Households = %d, %d, want 4, 4: the counterparties are in neither count", s.Agents, s.Households)
			}
			for _, f := range s.Construction {
				if strings.HasPrefix(string(f.Reason), "outside_") {
					t.Errorf("construction bucket has the flow %+v, want none under an outside_ reason", f)
				}
			}
		})
	}
}

// outsideBoundaryIDs returns the IDs of the boundary accounts of the six reasons
// of the module: the three money reasons, then the three goods reasons of each
// commodity in order.
func outsideBoundaryIDs(t *testing.T, w *World) []int {
	t.Helper()
	var ids []int
	add := func(r ledger.Reason, thing int) {
		id, ok := w.ledger.Boundary(r, thing)
		if !ok {
			t.Fatalf("no boundary account for reason %d and thing %d", r, thing)
		}
		ids = append(ids, id)
	}
	for _, r := range []ledger.Reason{ledger.OutsideBudget, ledger.OutsideUnspent, ledger.OutsideReceipts} {
		add(r, ledger.Money)
	}
	for k := range w.cfg.Commodities {
		if k > 1 { // flour and grain are the only commodities of the fixture with a line
			break
		}
		for _, r := range []ledger.Reason{ledger.OutsideOffer, ledger.OutsideUnsold, ledger.OutsidePurchases} {
			add(r, k)
		}
	}
	return ids
}

// TestExternal_NodeHasNoMarket verifies that an outside node has an estate and
// dense market slots but no market in effect (spec-external-sector.md 3.1.2,
// D526): no MarketCleared for it, its RegionRow is marked Outside with the rest
// zero, and the snapshot has no Markets or Roles row for it.
//
// Requirement: FR-EXT-004
func TestExternal_NodeHasNoMarket(t *testing.T) {
	t.Parallel()
	w, rec := newShipWorld(t, extShipConfig, nil)
	for range 3 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() = %v, want nil", err)
		}
	}
	cleared := of[MarketCleared](rec)
	if len(cleared) == 0 {
		t.Fatal("no MarketCleared event in three steps, want the markets of the other regions")
	}
	for _, e := range cleared {
		if e.Region == extIsle {
			t.Errorf("MarketCleared for the outside node: %+v, want none", e)
		}
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() = %v, want nil", err)
	}
	if want := (RegionRow{Region: extIsle, Outside: true}); s.Regions[extIsle] != want {
		t.Errorf("Regions[node] = %+v, want %+v", s.Regions[extIsle], want)
	}
	nr, nc := len(w.cfg.Roles), len(w.cfg.Commodities)
	if len(s.Roles) != 2*nr || len(s.Markets) != 2*nc {
		t.Errorf("snapshot has %d role rows and %d market rows, want %d and %d: two regions with a market", len(s.Roles), len(s.Markets), 2*nr, 2*nc)
	}
	for _, r := range s.Roles {
		if r.Region == extIsle {
			t.Errorf("Roles row for the node: %+v, want none", r)
		}
	}
	for _, m := range s.Markets {
		if m.State.Region == extIsle {
			t.Errorf("Markets row for the node: %+v, want none", m)
		}
	}
	for _, e := range w.cfg.Warnings() {
		if e.Rule == "region_without_agents" && len(e.Path) > 1 && e.Path[1] == "c_isle" {
			t.Errorf("region_without_agents for the node: %v, want none", e)
		}
	}
}

// TestExternal_RefusalRows verifies the rows 5a of spec-regions.md 6.1.3 and 6a of
// spec-transport.md 5.4: a relocation into the node is outside_node with
// ErrOutsideNode and changes nothing, after same_region and unknown_region; a
// shipment to it is outside_destination between unknown_region and no_route and
// changes nothing; Routes toward it reports no route.
//
// Requirement: FR-EXT-034, FR-TRN-047, FR-REG-058
func TestExternal_RefusalRows(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, extShipConfig, nil)
	hh := shipHome + 2 // the first household of the fixture is agent 5; shipHome = 3 is a counterparty here
	if hh != extHousehold {
		t.Fatalf("fixture household = %d, want %d", hh, extHousehold)
	}
	before := w.ledger.Len()
	reloc := []struct {
		name string
		id   AgentID
		to   RegionID
		want Reason
		err  error
	}{
		{"into the node", hh, extIsle, ReasonOutsideNode, ErrOutsideNode},
		{"into an unknown region first", hh, 9, ReasonUnknownRegion, ErrUnknownRegion},
		{"into its own region first", hh, 0, ReasonSameRegion, ErrSameRegion},
		{"into another region", hh, 1, "", nil},
	}
	for _, tt := range reloc {
		reason, err := w.relocateRefusal(tt.id, tt.to, nil)
		if reason != tt.want || !errors.Is(err, tt.err) && (err != nil || tt.err != nil) {
			t.Errorf("relocateRefusal(%s) = %q, %v, want %q, %v", tt.name, reason, err, tt.want, tt.err)
		}
	}
	if err := w.relocate(hh, extIsle, CauseConsumer); !errors.Is(err, ErrOutsideNode) {
		t.Errorf("relocate into the node = %v, want ErrOutsideNode", err)
	}
	if w.regionOf(&w.agents[hh]) != 0 || w.ledger.Len() != before {
		t.Errorf("a refused relocation changed the agent's region or the accounts")
	}
	ships := []struct {
		name string
		to   RegionID
		want Reason
	}{
		{"to the node", extIsle, ReasonOutsideDestination},
		{"to an unknown region", 9, ReasonUnknownRegion},
		{"to a region with a path", 1, ""},
	}
	for _, tt := range ships {
		if _, reason := ask(w, hh, 0, ShipmentRequest{Mode: modeFoot, To: tt.to, ToSelf: true}); reason != tt.want && tt.want != "" {
			t.Errorf("ask(%s) refusal = %q, want %q", tt.name, reason, tt.want)
		}
	}
	if err := w.Step(); err != nil {
		t.Fatalf("Step() = %v, want nil", err)
	}
	for _, rows := range w.transport.tables.routes {
		for _, r := range rows {
			if r.To == extIsle && r.Has {
				t.Errorf("route row to the node = %+v, want no route", r)
			}
		}
	}
}

// TestExternal_Off verifies that the module off leaves no trace in a world: no
// state, no Outside account, no boundary account of its reasons, the first
// household right after the estates, and the same accounts as the configuration
// without the module (spec-external-sector.md 3.1.1).
//
// Requirement: FR-EXT-001
func TestExternal_Off(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, nil, nil)
	if w.external != nil || w.firstHousehold != len(w.cfg.Regions) || len(w.outside) != len(w.cfg.Regions) {
		t.Fatalf("module off: external %v, firstHousehold %d, outside %v, want nil, %d and all false", w.external, w.firstHousehold, w.outside, len(w.cfg.Regions))
	}
	if slices.Contains(w.outside, true) {
		t.Errorf("outside marks = %v, want all false", w.outside)
	}
	for i := range w.ledger.Len() {
		if v, _ := w.ledger.Account(i); v.Kind == ledger.KindOutside {
			t.Errorf("account %d is an Outside account, want none", i)
		}
	}
	if _, ok := w.ledger.Boundary(ledger.OutsideBudget, ledger.Money); ok {
		t.Errorf("boundary account of outside_budget exists, want none while the module is off")
	}
	on, _ := newShipWorld(t, extShipConfig, nil)
	// 3 more boundary accounts of money, 3 for each of 2 commodities, 2 Outside accounts.
	if got, want := on.ledger.Len()-w.ledger.Len(), 3+3*2+2; got != want {
		t.Errorf("accounts with the module on less off = %d, want %d", got, want)
	}
}

// TestExternal_AuditDetects verifies the audit of the module (spec-external-sector.md
// 8.5, FR-EXT-050): a counterparty account that holds money, a cumulative unit count
// that disagrees with the boundary balance, and a boundary balance of the wrong sign
// are each reported by Audit, naming the node.
//
// Requirement: FR-EXT-050
func TestExternal_AuditDetects(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		want   string // a word of the error
		tamper func(t *testing.T, w *World)
	}{
		// The ledger's own sectoral identity also reports money and goods held by the
		// outside sector, and runs first; the module's check names the node.
		{"money in a counterparty account", "c_isle", func(t *testing.T, w *World) {
			src, _ := w.ledger.Boundary(ledger.OutsideBudget, ledger.Money)
			if err := w.ledger.Transfer(src, w.external.cps[0].account, 7, ledger.OutsideBudget); err != nil {
				t.Fatalf("Transfer() = %v, want nil", err)
			}
		}},
		{"goods in a counterparty account", "c_isle", func(t *testing.T, w *World) {
			grain := int(commodityOf(t, w, "grain"))
			src, _ := w.ledger.Boundary(ledger.OutsideOffer, grain)
			if err := w.ledger.Move(src, w.external.cps[1].account, grain, 2, ledger.OutsideOffer); err != nil {
				t.Fatalf("Move() = %v, want nil", err)
			}
		}},
		{"a carry out of its range", "c_isle", func(t *testing.T, w *World) { w.external.rows[0].carry = 10_000 }},
		{"a drift beyond the bound", "c_isle", func(t *testing.T, w *World) { w.external.lines[0].drift = 1 }},
		{"a cumulative unit count the boundary does not match", "c_isle", func(t *testing.T, w *World) { w.external.rows[0].units = 3 }},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _ := newShipWorld(t, extShipConfig, nil)
			if err := w.Audit(); err != nil {
				t.Fatalf("Audit() before the tamper = %v, want nil", err)
			}
			tt.tamper(t, w)
			if err := w.Audit(); err == nil {
				t.Errorf("Audit() after %s = nil, want an error", tt.name)
			}
			err := w.external.audit(w)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("external.audit after %s = %v, want an error that mentions %q", tt.name, err, tt.want)
			}
		})
	}
}

// TestExternal_NoRealmReads verifies that the functions of the module's tick read
// nothing of the realm: no AgentView, no Snapshot, no balance and no market report
// (spec-external-sector.md 11.3, FR-EXT-023, FR-EXT-033).
//
// Requirement: FR-EXT-033
// Guard: the order of a counterparty is a pure function of the tick, the line state
// and the edge status; a read of a price, a balance or an aggregate of the realm
// would make the outside a policy (D532). The check is by name, over the functions
// of external.go that run in the tick; the validator and the audit read what they
// must.
func TestExternal_NoRealmReads(t *testing.T) {
	t.Parallel()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "external.go", nil, 0)
	if err != nil {
		t.Fatalf("parse external.go: %v", err)
	}
	tickFuncs := []string{"p18Walk", "p35Orders", "p42Sweep", "lineOrder", "wantUnits", "sideQuote"}
	forbidden := []string{"AgentView", "Snapshot", "Balance", "reports", "unitCost", "price"}
	seen := 0
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || !slices.Contains(tickFuncs, fn.Name.Name) {
			continue
		}
		seen++
		ast.Inspect(fn, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && slices.Contains(forbidden, id.Name) {
				t.Errorf("%s reads %s, want no read of the realm", fn.Name.Name, id.Name)
			}
			return true
		})
	}
	t.Logf("checked %d tick functions of external.go", seen)
}

// TestExternal_RestoreAttachesAccounts verifies that a restore finds the Outside
// accounts of the counterparties and the first household again, and that the
// restored world continues exactly as the original does (the walk and the other
// state of the module join the checkpoint in tag 18, a later phase).
//
// Requirement: FR-EXT-016, FR-EXT-011
func TestExternal_RestoreAttachesAccounts(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, extShipConfig, nil)
	for range 3 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() = %v, want nil", err)
		}
	}
	cp, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() = %v, want nil", err)
	}
	c := shipConfig()
	extShipConfig(&c)
	r, err := RestoreWorld(c, cp, WithStrategy("script", &shipScript{}))
	if err != nil {
		t.Fatalf("RestoreWorld() = %v, want nil", err)
	}
	if r.firstHousehold != w.firstHousehold || !slices.Equal(r.external.cps, w.external.cps) {
		t.Errorf("restored firstHousehold %d and counterparties %+v, want %d and %+v", r.firstHousehold, r.external.cps, w.firstHousehold, w.external.cps)
	}
	for _, x := range []*World{w, r} {
		if err := x.Step(); err != nil {
			t.Fatalf("Step() after the restore = %v, want nil", err)
		}
	}
	a, errA := w.Checkpoint()
	b, errB := r.Checkpoint()
	if errA != nil || errB != nil || !reflect.DeepEqual(a, b) {
		t.Errorf("checkpoints after one more step differ (errors %v, %v), want equal", errA, errB)
	}
}
