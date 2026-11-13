package economysim

import (
	"errors"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// treasuryConfig is regionMill (regions away 0 and home 1, estates are agents 0 and
// 1) with a polity of two governed jurisdictions, crown (0, seat home, 50 000) and
// inland (1, a child of crown, seat away, 5 000), and a grouping, march (2). The
// treasury agents are 2 (crown) and 3 (inland); the households start at 4.
func treasuryConfig() Config {
	c := regionMill()
	c.Modules.Polity = &PolityConfig{
		Jurisdictions: []Jurisdiction{
			{ID: "crown", Seat: "home", StartMoney: 50_000},
			{ID: "inland", Parent: "crown", Seat: "away", StartMoney: 5_000},
			{ID: "march", Parent: "crown"},
		},
		Members: []Member{{Region: "away", Jurisdiction: "inland"}, {Region: "home", Jurisdiction: "crown"}},
	}
	return c
}

// TestPolity_ConstructionOrder verifies the orders of spec-polity.md 5.2 and the
// worked IDs of 12.1: the treasury accounts follow the estates' and precede every
// agent's, in ascending JurisdictionID, in the seat's region with the
// JurisdictionID as owner; the pseudo-agents take the IDs after the estates, so the
// households start g later; and the StartMoney is in the construction bucket from
// the boundary of initial.
//
// Arithmetic: 2 regions give estate agents 0 and 1; g = 2 governed jurisdictions
// (march is a grouping) give 2 and 3; firstHousehold = 2 + 0 + 2 = 4. The ledger
// accounts: estates, then crown's treasury, then inland's, then the households'.
//
// Requirement: FR-POL-015, FR-POL-016, FR-POL-017, FR-WLD-030, FR-WLD-031, FR-WLD-076
func TestPolity_ConstructionOrder(t *testing.T) {
	t.Parallel()
	w := newMill(t, treasuryConfig(), WithAudit())
	if w.firstHousehold != 4 {
		t.Fatalf("firstHousehold = %d, want 4 (2 estates + 2 treasuries)", w.firstHousehold)
	}
	first := w.estates.accounts[0] // the boundary accounts come before the estates
	checkTreasury(t, w, 0, treasuryWant{2, true, first + 2, 1, 50_000})
	checkTreasury(t, w, 1, treasuryWant{3, true, first + 3, 0, 5_000})
	checkTreasury(t, w, 2, treasuryWant{agent: NoAgent})
	if len(w.estates.accounts) != 2 || first+4 >= w.ledger.Len() {
		t.Fatalf("estates %d, accounts %d, want 2 estates, then 2 treasuries, then agents", len(w.estates.accounts), w.ledger.Len())
	}
	if v, err := w.ledger.Account(first + 4); err != nil || v.Kind != ledger.KindAgent {
		t.Errorf("account %d = %+v, %v, want the first agent account, right after the treasuries", first+4, v, err)
	}
	// The start money is initial money, issued by the boundary of initial: the 55 000
	// of the two treasuries and at least nothing else is asserted here, the agents'
	// own start money being the same reason.
	if got := w.boundaryFlowMoney(ledger.Initial); got < 55_000 {
		t.Errorf("initial money issued = %d, want at least the 55 000 of the treasuries", got)
	}
}

// treasuryWant is what a governed jurisdiction's treasury must be: its pseudo-agent,
// whether it has one, its ledger account, the account's region and its money.
type treasuryWant struct {
	agent   AgentID
	ok      bool
	account int
	region  int
	money   int64
}

// checkTreasury checks the agent and the account of jurisdiction j against want.
func checkTreasury(t *testing.T, w *World, j int, want treasuryWant) {
	t.Helper()
	got, ok := w.TreasuryAgent(JurisdictionID(j))
	if got != want.agent || ok != want.ok {
		t.Errorf("TreasuryAgent(%d) = %d, %v, want %d, %v", j, got, ok, want.agent, want.ok)
	}
	if !ok {
		return
	}
	v, err := w.ledger.Account(want.account)
	if err != nil || v.Kind != ledger.KindTreasury || v.Owner != j || v.Region != want.region || v.Balance != want.money || v.Closed {
		t.Errorf("account %d = %+v, %v, want an open Treasury of jurisdiction %d in region %d holding %d", want.account, v, err, j, want.region, want.money)
	}
}

// boundaryFlowMoney is the money the boundary of reason r has issued since the
// world was built: the negated balance of its boundary account.
func (w *World) boundaryFlowMoney(r ledger.Reason) int64 {
	b, ok := w.ledger.Boundary(r, ledger.Money)
	if !ok {
		return 0
	}
	v, _ := w.ledger.Account(b)
	return -v.Balance
}

// TestPolity_NeutralWithoutSeat verifies that a grouping creates no account and no
// pseudo-agent and shifts no ID: g is 0, firstHousehold is the number of regions,
// and TreasuryAgent answers none for every jurisdiction.
//
// Requirement: FR-POL-019
func TestPolity_NeutralWithoutSeat(t *testing.T) {
	t.Parallel()
	c := treasuryConfig()
	for i := range c.Modules.Polity.Jurisdictions {
		c.Modules.Polity.Jurisdictions[i].Seat, c.Modules.Polity.Jurisdictions[i].StartMoney = "", 0
	}
	w := newMill(t, c, WithAudit())
	off := newMill(t, regionMill(), WithAudit())
	if w.firstHousehold != 2 || w.ledger.Len() != off.ledger.Len() {
		t.Errorf("firstHousehold = %d and %d accounts, want 2 and %d: a grouping creates nothing", w.firstHousehold, w.ledger.Len(), off.ledger.Len())
	}
	for j := range 3 {
		if a, ok := w.TreasuryAgent(JurisdictionID(j)); ok || a != NoAgent {
			t.Errorf("TreasuryAgent(%d) = %d, %v, want NoAgent, false", j, a, ok)
		}
	}
}

// TestPolity_TreasuryAgentIsNone verifies the answers of TreasuryAgent that are not a
// governed jurisdiction: an unknown or negative j, and a world without the module.
//
// Requirement: FR-POL-016
func TestPolity_TreasuryAgentIsNone(t *testing.T) {
	t.Parallel()
	w := newMill(t, treasuryConfig())
	for _, j := range []JurisdictionID{-1, 3, 1 << 40} {
		if a, ok := w.TreasuryAgent(j); ok || a != NoAgent {
			t.Errorf("TreasuryAgent(%d) = %d, %v, want NoAgent, false", j, a, ok)
		}
	}
	if a, ok := newMill(t, regionMill()).TreasuryAgent(0); ok || a != NoAgent {
		t.Errorf("TreasuryAgent(0) with the module off = %d, %v, want NoAgent, false", a, ok)
	}
}

// TestPolity_TreasuryRefusals verifies that a treasury's ID is treated as a
// pseudo-agent's by the calls that name an agent (spec-polity.md 5.3.3,
// spec-integration.md 3.4.3, 4.5.1): Relocate is ErrNotRelocatable, which the
// existing id < firstHousehold test already gives once firstHousehold has grown,
// and Direct and RemoveAgent are ErrUnknownAgent. A transfer that would take the
// account below 0 is refused and the account is never closed.
//
// Requirement: FR-POL-015, FR-POL-029, FR-POL-031
func TestPolity_TreasuryRefusals(t *testing.T) {
	t.Parallel()
	w := newMill(t, treasuryConfig(), WithAudit())
	for _, id := range []AgentID{2, 3} {
		if err := w.Relocate(id, 0); !errors.Is(err, ErrNotRelocatable) {
			t.Errorf("Relocate(%d) error = %v, want ErrNotRelocatable", id, err)
		}
		if err := w.RemoveAgent(id); !errors.Is(err, ErrUnknownAgent) {
			t.Errorf("RemoveAgent(%d) error = %v, want ErrUnknownAgent", id, err)
		}
		if err := w.Direct(id, Directive{}); !errors.Is(err, ErrUnknownAgent) {
			t.Errorf("Direct(%d) error = %v, want ErrUnknownAgent", id, err)
		}
	}
	tr, boundary := w.polity.tr[0].account, w.estates.accounts[0]
	if err := w.ledger.Transfer(tr, boundary, 50_001, ledger.EstateMoney); err == nil {
		t.Error("a transfer of more than the balance out of a treasury succeeded, want it refused")
	}
	if v, _ := w.ledger.Account(tr); v.Balance != 50_000 || v.Closed {
		t.Errorf("treasury after a refused transfer = %+v, want 50 000 and open", v)
	}
}

// TestPolity_Audit verifies that the audit runs with two treasuries and a non-zero
// StartMoney, and that it fails when a governed jurisdiction's account is not the
// open Treasury account of its seat.
//
// Requirement: FR-POL-029, FR-LED-025
func TestPolity_Audit(t *testing.T) {
	t.Parallel()
	w := newMill(t, treasuryConfig(), WithAudit())
	if err := w.Audit(); err != nil {
		t.Fatalf("Audit() error = %v, want nil", err)
	}
	w.polity.tr[1].seat = 1 // the account sits in region 0
	if err := w.Audit(); !errors.Is(err, ErrAuditFailed) {
		t.Errorf("Audit() with a treasury changed error = %v, want ErrAuditFailed", err)
	}
}

// TestPolity_AttachTreasuries verifies the check of a restore (spec-polity.md 11,
// FR-POL-030): the attach accepts the ledger the construction made and rejects, as
// ErrCheckpointCorrupt, an account that is missing, of another kind, owned by another
// jurisdiction, in another region or closed.
//
// Requirement: FR-POL-030
func TestPolity_AttachTreasuries(t *testing.T) {
	t.Parallel()
	for name, mutate := range map[string]func(p *polity){
		"wrong seat":    func(p *polity) { p.tr[0].seat = 0 },
		"wrong owner":   func(p *polity) { p.tr[0].j = 2 },
		"swapped order": func(p *polity) { p.tr[0].j, p.tr[1].j = p.tr[1].j, p.tr[0].j },
		"more than the ledger": func(p *polity) {
			p.tr = append(p.tr, treasury{j: 2, seat: 0})
			p.tr = append(p.tr, treasury{j: 2, seat: 0})
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			w := newMill(t, treasuryConfig())
			if err := w.attachTreasuries(); err != nil {
				t.Fatalf("attachTreasuries() on the ledger the construction made error = %v, want nil", err)
			}
			mutate(w.polity)
			if err := w.attachTreasuries(); !errors.Is(err, ErrCheckpointCorrupt) {
				t.Errorf("attachTreasuries() error = %v, want ErrCheckpointCorrupt", err)
			}
		})
	}
}

// TestPolity_ConstructionOrderWithOutside verifies the order with the external
// module on (spec-polity.md 12.2): the counterparties' IDs and Outside accounts come
// first, then the treasury's agent ID and account, then the households.
//
// Arithmetic: extConfig has 2 regions (estates 0, 1) and 2 counterparties (2, 3); one
// governed jurisdiction takes agent 4, so firstHousehold = 2 + 2 + 1 = 5, and its
// account is the 5th after the first estate account: 2 estates and 2 Outside accounts
// come before it.
//
// Requirement: FR-POL-015, FR-POL-016, FR-WLD-030
func TestPolity_ConstructionOrderWithOutside(t *testing.T) {
	t.Parallel()
	c := extConfig()
	c.Modules.Polity = &PolityConfig{
		Jurisdictions: []Jurisdiction{{ID: "realm", Seat: "lowland", StartMoney: 700}},
		Members:       []Member{{Region: "coast", Jurisdiction: "realm"}, {Region: "lowland", Jurisdiction: "realm"}},
	}
	w := newMill(t, c, WithAudit(), WithStrategy("script", &extScript{plan: map[AgentID][]Intent{}}))
	if w.firstHousehold != 5 {
		t.Fatalf("firstHousehold = %d, want 5", w.firstHousehold)
	}
	if a, ok := w.TreasuryAgent(0); !ok || a != 4 {
		t.Errorf("TreasuryAgent(0) = %d, %v, want 4, true", a, ok)
	}
	first := w.estates.accounts[0]
	for i, kind := range []ledger.Kind{ledger.KindEstate, ledger.KindEstate, ledger.KindOutside, ledger.KindOutside, ledger.KindTreasury, ledger.KindAgent} {
		if v, err := w.ledger.Account(first + i); err != nil || v.Kind != kind {
			t.Errorf("account %d after the boundaries = %+v, %v, want kind %d", i, v, err, kind)
		}
	}
	if v, _ := w.ledger.Account(first + 4); v.Balance != 700 || v.Region != 1 {
		t.Errorf("the treasury account = %+v, want 700 in region lowland (1)", v)
	}
}

// TestPolity_TreasuryIsNotAnAgent verifies that a treasury pseudo-agent is not an
// agent or a household (spec-polity.md 5.3.3): no AgentSpawned or AgentBankrupted
// event names it, the agent and household counts are those of the world without the
// module, and the live agent table holds no weight at its slot.
//
// Requirement: FR-POL-016, FR-POL-031
func TestPolity_TreasuryIsNotAnAgent(t *testing.T) {
	t.Parallel()
	rec := &evLog{}
	w := newMill(t, treasuryConfig(), WithObserver(rec), WithAudit())
	base := newMill(t, regionMill())
	if w.live.agents != base.live.agents || w.live.households != base.live.households {
		t.Errorf("agents %d, households %d, want %d and %d of the world without the module", w.live.agents, w.live.households, base.live.agents, base.live.households)
	}
	for _, e := range rec.events {
		if s, ok := e.(AgentSpawned); ok && s.Agent < AgentID(w.firstHousehold) {
			t.Errorf("AgentSpawned names the pseudo-agent %d, want none", s.Agent)
		}
	}
	for id := 2; id < w.firstHousehold; id++ {
		if w.agents[id].weight != 0 {
			t.Errorf("agents[%d].weight = %d, want 0: a treasury has no weight", id, w.agents[id].weight)
		}
	}
}

// TestPolity_NoRegionBound verifies that the module registers no region_bound
// refusal (spec-polity.md 5.3.2, N-090): a treasury is neither away nor committed,
// and Relocate refuses it as a pseudo-agent.
//
// Guard: a later change that registered an obligation for the module would stop a
// household relocating for no reason of its own.
func TestPolity_NoRegionBound(t *testing.T) {
	t.Parallel()
	with, without := newMill(t, treasuryConfig()), newMill(t, regionMill())
	if len(with.bounds.rows) != len(without.bounds.rows) {
		t.Errorf("%d region-bound obligations with the module, %d without, want equal", len(with.bounds.rows), len(without.bounds.rows))
	}
}

// TestPolity_IDShift verifies the consequence of spec-polity.md 5.2.3: every
// household's agent ID is g higher in a world with g governed jurisdictions. The two
// worlds' totals are not compared (POL-A1: a paired comparison across a governed and
// an ungoverned world is not valid, the IDs being tie-breaks).
//
// Guard: the shift is the price of a treasury and is stated, not hidden.
func TestPolity_IDShift(t *testing.T) {
	t.Parallel()
	with, without := newMill(t, treasuryConfig()), newMill(t, regionMill())
	if got, want := len(with.agents)-len(without.agents), 2; got != want {
		t.Fatalf("%d agent slots more than the world without the module, want %d", got, want)
	}
	for i := without.firstHousehold; i < len(without.agents); i++ {
		if a, b := without.agents[i], with.agents[i+2]; a.role != b.role || a.weight != b.weight {
			t.Errorf("household %d (role %d, weight %d) is at ID %d with role %d, weight %d, want the same household shifted by 2", i, a.role, a.weight, i+2, b.role, b.weight)
		}
	}
}
