package economysim

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"strconv"
	"strings"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the relocation of an agent's account to another region
// (spec-regions.md 6): the query of the refusals, the one path that moves an
// account, and the table of the region-bound obligations. The ledger changes the
// region of the account (internal/ledger/relocate.go); what the move does to the
// agent is only the counters of its (region, role) cell.

// regionBound is one registered obligation (spec-regions.md 6.5.2): its error,
// made once so that the refusal path allocates nothing, and the test that the
// agent holds an instance of it that skip does not name.
type regionBound struct {
	err *RegionBoundError
	// held reports whether the agent holds an instance of the obligation that skip
	// does not name by key. It names each instance through skipHas.
	held func(a *agent, skip []string) bool
}

// regionBounds is the table of obligations, in registration order, which is
// module order. Its zero value is the empty table.
type regionBounds struct{ rows []regionBound }

// add registers the obligation name with its test. The table is closed at
// construction, so a name given twice is a violated invariant of the module that
// registers it.
func (b *regionBounds) add(name string, held func(a *agent, skip []string) bool) {
	// A name with a ':' would read as an instance key in skip, and a nil test
	// would panic in the refusal path, far from the module that registered it.
	if name == "" || strings.Contains(name, ":") || held == nil {
		invariant("the region-bound obligation " + strconv.Quote(name) + " has no name, a name with a colon, or no test")
	}
	for _, r := range b.rows {
		if r.err.Obligation == name {
			invariant("the region-bound obligation " + name + " is registered twice")
		}
	}
	b.rows = append(b.rows, regionBound{err: &RegionBoundError{Obligation: name}, held: held})
}

// refusal returns the error of the first obligation in registration order that
// the agent holds, skipping the ones skip names by name.
func (b *regionBounds) refusal(a *agent, skip []string) (*RegionBoundError, bool) {
	for _, r := range b.rows {
		if !skipHas(skip, r.err.Obligation, -1) && r.held(a, skip) {
			return r.err, true
		}
	}
	return nil, false
}

// skipHas reports whether skip names the obligation name, by name alone or by the
// instance key "name:key" with the key in canonical
// decimal (spec-regions.md 6.5.2). A key below 0 matches the bare name only,
// because no decimal form equals it. It allocates nothing.
func skipHas(skip []string, name string, key int64) bool {
	for _, s := range skip {
		if s == name {
			return true
		}
		n := len(name)
		if len(s) > n && s[n] == ':' && s[:n] == name && decimalIs(s[n+1:], key) {
			return true
		}
	}
	return false
}

// decimalIs reports whether s is the canonical decimal form of n: digits only,
// no sign and no leading zero, and a value that fits in int64.
func decimalIs(s string, n int64) bool {
	if s == "" || len(s) > 1 && s[0] == '0' {
		return false
	}
	var v int64
	for i := 0; i < len(s); i++ {
		// A byte below '0' wraps to a value above 9, so one test rejects both sides.
		d := s[i] - '0'
		if d > 9 || v > (math.MaxInt64-int64(d))/10 {
			return false
		}
		v = v*10 + int64(d)
	}
	return v == n
}

// RelocationRefusal returns the reason (a Reason constant) and the error of the
// first refusal of spec-regions.md 6.1.3 that moving agent a to region to would
// meet, or ("", nil) if none would. The obligations named in skip, by name or by
// instance key ("name:number"), are not tested; no other refusal can be skipped.
// It changes nothing, draws nothing and allocates nothing, and works on a failed
// world and from inside a step; its error is the refusal it reports, never the
// failure of the world (spec-world.md FR-WLD-006, 007).
func (w *World) RelocationRefusal(a AgentID, to RegionID, skip ...string) (string, error) {
	reason, err := w.relocateRefusal(a, to, skip)
	return string(reason), err
}

// relocateRefusal runs the checks of 6.1.3 in the order of its table. The first four are
// the root's own reads of the region and of the agent table, the pseudo-agents
// (an ID below the region count) being the estates, which hold no account of an
// agent; the ledger answers the same region and the reservation; the obligations
// of the modules come last.
func (w *World) relocateRefusal(id AgentID, to RegionID, skip []string) (Reason, error) {
	switch {
	case to < 0 || int(to) >= len(w.cfg.Regions):
		return ReasonUnknownRegion, ErrUnknownRegion
	case id < 0 || id >= w.state.nextAgent:
		return ReasonUnknownAgent, ErrUnknownAgent
	case int(id) < len(w.cfg.Regions):
		return ReasonNotRelocatable, ErrNotRelocatable
	}
	a := &w.agents[id]
	if a.weight < 1 {
		return ReasonClosedAccount, ErrClosedAccount
	}
	// The ledger answers rows 5 and 6 together. Row 5a, outside_node (the external
	// module, spec-regions.md 6.1.3), goes between them: its change splits the
	// answer at ReasonSameRegion and tests the destination there.
	if reason, err := ledgerRefusal(w.ledger.CheckRelocate(a.account, int(to))); err != nil {
		return reason, err
	}
	if e, ok := w.relocationRefusal(a, skip); ok {
		return ReasonRegionBound, e
	}
	return "", nil
}

// ledgerRefusal maps the ledger's refusal of a relocation to its reason and the
// root's error. The ledger's sentinels are matched by errors.Is, and the root
// returns its own bare sentinels so that the query allocates nothing. The root
// has already checked the region, the agent and the account, so any other error
// is a violated invariant. A sum of the pending relocation totals past int64 has
// no row in the table of 6.1.3; it is refused as ReasonOverflow, after the
// reservation.
func ledgerRefusal(err error) (Reason, error) {
	switch {
	case err == nil:
		return "", nil
	case errors.Is(err, ledger.ErrSameRegion):
		return ReasonSameRegion, ErrSameRegion
	case errors.Is(err, ledger.ErrOpenReservation):
		return ReasonOpenReservation, ErrOpenReservation
	case errors.Is(err, ledger.ErrBalanceOverflow):
		return ReasonOverflow, ErrOverflow
	}
	invariant("the ledger refused a relocation the world's own checks passed: " + err.Error())
	return "", nil
}

// relocate moves the account of agent id to region to, if no refusal of 6.1.3
// holds: it is the query followed by moveAgent, so it returns exactly the error
// the query names (spec-regions.md 6.1.1, FR-REG-059). The cause is carried to
// the event and the log. The refusal is logged at Debug and returned.
func (w *World) relocate(id AgentID, to RegionID, cause Cause) error {
	reason, err := w.relocateRefusal(id, to, nil)
	if err != nil {
		if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
			w.log.LogAttrs(ctx, slog.LevelDebug, "relocation refused",
				slog.Int64(logkeys.Agent, int64(id)), slog.Int64(logkeys.ToRegion, int64(to)), slog.String(logkeys.Reason, string(reason)))
		}
		return err
	}
	return w.moveAgent(&w.agents[id], to, cause)
}

// moveAgent is the one path that changes the region of an agent's account
// (spec-regions.md 6.2, architecture.md 2.10.4): the ledger moves the account and
// totals what it holds, the counters of the (region, role) cell follow, and the
// event and the Debug line come last. It does not check the refusals of 6.1.3 that
// the caller owns: relocate checks all of them, and TRN and MIG ask
// RelocationRefusal with their own skip first. The agent's role, weight, job,
// beliefs and the rest of its state are not touched (6.3). The error is the
// ledger's refusal, mapped as the query maps it. Only the live counters of the
// source and target cells change; an extinction of the source cell is seen by the
// next exit phase, as for any other departure between phases.
func (w *World) moveAgent(a *agent, to RegionID, cause Cause) error {
	from := w.regionOf(a)
	if _, err := ledgerRefusal(w.ledger.Relocate(a.account, int(to))); err != nil {
		return err
	}
	roles := len(w.cfg.Roles)
	w.live.inRole[int(from)*roles+int(a.role)]--
	w.live.inRole[int(to)*roles+int(a.role)]++
	w.state.hadAgent[int(to)*roles+int(a.role)] = true
	// Units cannot overflow here: a commodity holds at most MaxQuantity, and
	// there are at most MaxCommodities of them.
	if w.obs.supplied {
		v, err := w.ledger.Account(a.account)
		units, uerr := w.ledger.Units(a.account)
		if err != nil || uerr != nil {
			invariant("the account of a relocated agent is unreadable")
		}
		w.obs.deliver(AgentRelocated{Tick: w.Tick(), Agent: a.id, Role: a.role, From: from, To: to, Weight: a.weight, Money: Money(v.Balance), Units: units, Cause: cause})
	}
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "agent relocated",
			slog.Int64(logkeys.Agent, int64(a.id)), slog.Int64(logkeys.FromRegion, int64(from)), slog.Int64(logkeys.ToRegion, int64(to)),
			slog.String(logkeys.Cause, string(cause)), slog.Int64(logkeys.Households, a.weight))
	}
	return nil
}
