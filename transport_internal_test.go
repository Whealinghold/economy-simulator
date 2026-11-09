package economysim

import (
	"errors"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// transportSeed is the seed of these tests; the failure messages name it
// (testing.md 6.2).
const transportSeed = 20261109

// withTransport switches the transport module on in c with one self-carried mode.
func withTransport(c Config) Config {
	c.Modules.Transport = &Transport{Modes: []Mode{{ID: "foot", Speed: 20, UnitMass: 40, UnitBulk: 30, Crew: DefaultCrew}}}
	return c
}

// stocked returns the first commodity the agent account holds and how many units.
func stocked(t *testing.T, w *World, account int) (commodity int, held int64) {
	t.Helper()
	for c := range w.cfg.Commodities {
		h, _, err := w.ledger.Holding(account, c)
		if err != nil {
			t.Fatalf("seed %d: Holding(%d, %d) error = %v, want nil", transportSeed, account, c, err)
		}
		if h > 0 {
			return c, h
		}
	}
	t.Fatalf("seed %d: agent account %d holds no goods", transportSeed, account)
	return 0, 0
}

// TestLedger_TransitReasons verifies FR-LED-055 for the transport rows on a
// world with the module on: transit_out is goods from an agent to a Transit
// account, transit_in goods from a Transit account to an agent or an estate,
// haul_use and transit_use goods sinks of an agent, and every other pairing is
// ErrReasonMismatch, money included; with the module off none of the four is
// accepted.
//
// Requirement: FR-TRN-030, FR-LED-055
func TestLedger_TransitReasons(t *testing.T) {
	t.Parallel()
	w := newMill(t, withTransport(exitConfig()), WithSeed(transportSeed))
	agent := w.agents[1].account
	other := w.agents[2].account
	estate := w.estates.accounts[0]
	transit, err := w.ledger.Open(ledger.KindTransit, 0, 0)
	if err != nil {
		t.Fatalf("seed %d: Open(transit) error = %v, want nil", transportSeed, err)
	}
	c, held := stocked(t, w, agent)
	boundary, ok := w.ledger.Boundary(ledger.HaulUse, c)
	if !ok {
		t.Fatalf("seed %d: no haul_use boundary account for commodity %d", transportSeed, c)
	}
	mustMove := func(from, to int, r ledger.Reason, qty int64) {
		t.Helper()
		if err := w.ledger.Move(from, to, c, qty, r); err != nil {
			t.Fatalf("seed %d: Move(%d, %d, reason %d) error = %v, want nil", transportSeed, from, to, r, err)
		}
	}
	mustMove(agent, transit, ledger.TransitOut, 1)
	mustMove(transit, other, ledger.TransitIn, 1)
	mustMove(other, transit, ledger.TransitOut, 1)
	mustMove(transit, estate, ledger.TransitIn, 1)
	mustMove(agent, boundary, ledger.HaulUse, 1)
	if tb, ok := w.ledger.Boundary(ledger.TransitUse, c); !ok {
		t.Fatalf("seed %d: no transit_use boundary account for commodity %d", transportSeed, c)
	} else {
		mustMove(agent, tb, ledger.TransitUse, 1)
	}
	refused := []struct {
		name     string
		from, to int
		r        ledger.Reason
	}{
		{"transit_out to an estate", agent, estate, ledger.TransitOut},
		{"transit_out from a transit account", transit, agent, ledger.TransitOut},
		{"transit_in from an agent", agent, other, ledger.TransitIn},
		{"transit_in into a transit account", agent, transit, ledger.TransitIn},
		{"haul_use to a transit account", agent, transit, ledger.HaulUse},
		{"transit_use to an estate", agent, estate, ledger.TransitUse},
	}
	for _, tt := range refused {
		if err := w.ledger.Move(tt.from, tt.to, c, 1, tt.r); !errors.Is(err, ledger.ErrReasonMismatch) {
			t.Errorf("seed %d: Move(%s) error = %v, want ErrReasonMismatch", transportSeed, tt.name, err)
		}
	}
	if err := w.ledger.Transfer(agent, transit, 1, ledger.TransitOut); !errors.Is(err, ledger.ErrReasonMismatch) {
		t.Errorf("seed %d: Transfer(transit_out, money) error = %v, want ErrReasonMismatch: the rows move goods only", transportSeed, err)
	}
	if held < 4 {
		t.Errorf("seed %d: the agent held %d units of commodity %d, the moves above need 4", transportSeed, held, c)
	}

	off := newMill(t, exitConfig(), WithSeed(transportSeed))
	oc, _ := stocked(t, off, off.agents[1].account)
	if err := off.ledger.Move(off.agents[1].account, off.estates.accounts[0], oc, 1, ledger.TransitIn); err == nil {
		t.Errorf("seed %d: Move(transit_in) with the module off succeeded, want a refusal", transportSeed)
	}
}

// TestConstruct_BoundaryAccountsRow3a verifies the creation position of the
// boundary accounts of haul_use and transit_use (spec-world.md 5.2 row 3a, N-226):
// with the module on they are opened after the built-in group, goods
// commodity-major, haul_use before transit_use, and before the estate accounts,
// which shift by twice the commodity count; with the module off no account is
// opened for them and the first estate account is where Stage 1 puts it.
//
// Requirement: FR-TRN-030
func TestConstruct_BoundaryAccountsRow3a(t *testing.T) {
	t.Parallel()
	off := newMill(t, exitConfig(), WithSeed(transportSeed))
	on := newMill(t, withTransport(exitConfig()), WithSeed(transportSeed))
	n := len(on.cfg.Commodities)
	base := off.estates.accounts[0]
	if got := on.estates.accounts[0]; got != base+2*n {
		t.Errorf("seed %d: first estate account = %d with the module on, want %d + 2*%d from the module off", transportSeed, got, base, n)
	}
	for c := range n {
		haul, hok := on.ledger.Boundary(ledger.HaulUse, c)
		use, uok := on.ledger.Boundary(ledger.TransitUse, c)
		if !hok || !uok || haul != base+2*c || use != base+2*c+1 {
			t.Errorf("seed %d: boundary accounts of commodity %d = %d, %d (%v, %v), want %d and %d", transportSeed, c, haul, use, hok, uok, base+2*c, base+2*c+1)
		}
		if _, ok := off.ledger.Boundary(ledger.HaulUse, c); ok {
			t.Errorf("seed %d: haul_use has a boundary account for commodity %d with the module off", transportSeed, c)
		}
	}
	if _, ok := on.ledger.Boundary(ledger.TransitOut, 0); ok {
		t.Errorf("seed %d: transit_out has a boundary account, want none: it is internal", transportSeed)
	}
}
