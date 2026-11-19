package economysim_test

import (
	"fmt"
	"reflect"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The end-to-end tests of the institutions module over a whole run of pack one, with the
// rights and the legal prices both binding (PR C): a body whose places are fewer than its
// eligible households, a right of the body that refuses the others, and a legal price that
// lowers a limit. They are the proof from outside that the module's readers refuse, clamp,
// tell, conserve and repeat (spec-institutions.md 5, 6, 7, 11).

// institutionsLegalDoc is the institutions tables over pack one that bind. The guild holds
// the makers of north, 55 households, for 40 places, so 15 are outsiders; make_tools and
// sell_tools bar them from making and selling tool_a in north; assize caps the staple of
// north at 9 000 bp of its reference price 96, which is 96 * 9 000 / 10 000 = 86 (truncated),
// below the price the market runs at (about 95), so it lowers bids.
const institutionsLegalDoc = `
[modules.institutions]

[modules.institutions.body.guild]
roles = ["maker"]
regions = ["north"]
places = 40

[modules.institutions.right.make_tools]
body = "guild"
activity = "produce"
subject = "make_tool_a"
region = "north"

[modules.institutions.right.sell_tools]
body = "guild"
activity = "sell"
subject = "tool_a"
region = "north"

[modules.institutions.control.assize]
commodity = "staple"
region = "north"
cap_bp = 9000
`

// legalRun is what one observed run of institutionsLegalDoc kept.
type legalRun struct {
	events    []economysim.Event
	refused   economysim.RightRefused // the sum of every RightRefused, Tick and ids unset
	rejected  int64                   // OrderRejected with the reason not_permitted
	lowered   int64                   // bids lowered, the sum of PriceControlled.Bids
	trades    int64                   // trades of the staple
	topTrade  economysim.Price        // the highest price of a staple trade
	worldHash [32]byte
}

// runLegal runs the world of cfg for ticks from the seed, with the audit, and sums the
// events of the module. The staple is the commodity of index 5 of pack one.
func runLegal(t testing.TB, cfg economysim.Config, seed uint64, ticks int) legalRun {
	t.Helper()
	var r legalRun
	obs := economysim.ObserverFunc(func(e economysim.Event) {
		r.events = append(r.events, e)
		switch e := e.(type) {
		case economysim.RightRefused:
			r.refused.Produce += e.Produce
			r.refused.Sell += e.Sell
			r.refused.Buy += e.Buy
		case economysim.OrderRejected:
			if e.Reason == economysim.ReasonNotPermitted {
				r.rejected++
			}
		case economysim.PriceControlled:
			r.lowered += e.Bids
		case economysim.TradeExecuted:
			if e.Commodity == 5 {
				r.trades++
				r.topTrade = max(r.topTrade, e.Price)
			}
		}
	})
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithObserver(obs), economysim.WithAudit())
	if err != nil {
		t.Fatalf("NewWorld() seed %d error = %v, want nil", seed, err)
	}
	for range ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() seed %d error = %v, want nil", seed, err)
		}
	}
	r.worldHash = economysim.HashWorld(w)
	return r
}

// TestEndToEnd_InstitutionsRightsAndLegalPrices verifies, over whole runs of pack one with
// the module on and a body, a right and a legal price in force, what a consumer sees: the
// right refuses (RightRefused counts the refused sells and starts, and the refused orders are
// OrderRejected with the reason not_permitted, the same orders counted twice), the legal
// price lowers (PriceControlled tells the bids lowered and no staple trade is above the cap
// of 86), the audit passes every tick (money and goods conserved: it checks the ledger after
// each step), and two runs of one seed agree event for event and in the state hash.
//
// Arithmetic: the cap is 96 * 9 000 / 10 000 = 86; a run that never lowered a bid or never
// refused an order would pass the bounds vacuously, so each seed must show both.
//
// Requirement: FR-ORG-020, FR-ORG-030, FR-ORG-031, FR-ORG-040, FR-ORG-041, FR-ORG-053, FR-ORG-055
func TestEndToEnd_InstitutionsRightsAndLegalPrices(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadPackBytes(t, []byte(institutionsLegalDoc), packOne...)
	for _, seed := range seedList(invariantSeeds(institutionsWorldSeeds)) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			a := runLegal(t, cfg, seed, 60)
			t.Logf("seed %d: refused %+v, not_permitted rejections %d, bids lowered %d, staple trades %d, top price %d", seed, a.refused, a.rejected, a.lowered, a.trades, a.topTrade)
			if a.refused.Produce == 0 || a.refused.Sell == 0 {
				t.Errorf("seed %d: RightRefused counts = %+v, want starts and sells refused to the 15 outsiders of the guild", seed, a.refused)
			}
			if got := a.refused.Sell + a.refused.Buy; got != a.rejected {
				t.Errorf("seed %d: RightRefused sell and buy orders = %d, OrderRejected not_permitted = %d, want the same refused orders", seed, got, a.rejected)
			}
			if a.lowered == 0 || a.trades == 0 {
				t.Errorf("seed %d: bids lowered = %d, staple trades = %d, want both above 0 so the cap binds", seed, a.lowered, a.trades)
			}
			if a.topTrade > 86 {
				t.Errorf("seed %d: highest staple trade price = %d, want at most the cap of 86", seed, a.topTrade)
			}
			b := runLegal(t, cfg, seed, 60)
			if !reflect.DeepEqual(a.events, b.events) || a.worldHash != b.worldHash {
				t.Errorf("seed %d: two runs of one seed differ in their events or in the state hash", seed)
			}
		})
	}
}

// TestInvariants_InstitutionsLegalPrices verifies the invariants of testing.md 6.3 over the
// world of institutionsLegalDoc, seed by seed and after every tick: goods and money are
// conserved with exact equality, no balance or stock is negative, no trade has a missing
// party, every clearing price lies between the matched limits (a clamped order is placed at
// its replaced limit, which the event shows), and a second world of the same seed has the
// same first tick.
//
// Requirement: FR-ORG-031, FR-ORG-041, FR-ORG-052, FR-REF-028
//
// Guard: a refused order that kept its reservation, or a clamp that reserved the old limit
// and traded at the new, would break conservation or the price bounds only in a world where
// both bind, which no pack of the table is.
func TestInvariants_InstitutionsLegalPrices(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadPackBytes(t, []byte(institutionsLegalDoc), packOne...)
	for _, seed := range seedList(invariantSeeds(institutionsWorldSeeds)) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			runInvariants(t, cfg, seed, 60, nil)
		})
	}
}

// TestEndToEnd_InstitutionsOffLeavesNoTrace verifies that the module off is the world
// before the module existed: the run of pack one emits no event of the module and no
// rejection for not_permitted, a world with the module present and empty is the same world
// (snapshot for snapshot, apart from the hash and the list of modules), and the module's
// bodies and rules, once declared, are what changes the run.
//
// Requirement: FR-ORG-001, FR-ORG-029
func TestEndToEnd_InstitutionsOffLeavesNoTrace(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	off := loadValidPack(t, packOne...)
	empty := off
	empty.Modules.Institutions = &economysim.Institutions{}
	on := loadPackBytes(t, []byte(institutionsLegalDoc), packOne...)
	for _, seed := range seedList(invariantSeeds(institutionsWorldSeeds)) {
		a, b := runLegal(t, off, seed, 60), runLegal(t, empty, seed, 60)
		for _, r := range []legalRun{a, b} {
			for _, e := range r.events {
				switch e.(type) {
				case economysim.MembershipChanged, economysim.RightRefused, economysim.PriceControlled:
					t.Errorf("seed %d: event %T of the module in a world without bodies, want none", seed, e)
				}
			}
			if r.rejected != 0 {
				t.Errorf("seed %d: %d orders rejected as not_permitted without a right, want 0", seed, r.rejected)
			}
		}
		if a.trades != b.trades || a.topTrade != b.topTrade {
			t.Errorf("seed %d: staple trades %d at most %d (off) against %d at most %d (empty module), want the same", seed, a.trades, a.topTrade, b.trades, b.topTrade)
		}
		if c := runLegal(t, on, seed, 60); c.worldHash == a.worldHash {
			t.Errorf("seed %d: the state hash with a body, a right and a control equals that of the world without the module, want the module to move the run", seed)
		}
	}
}
