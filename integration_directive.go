package economysim

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the directive call of spec-integration.md 4.3, split from integration.go.

// Direct leaves a directive in the mailbox of a directed external agent
// (spec-integration.md 4.3). It replaces what the mailbox held, an empty directive
// empties it, and it applies to the next step only unless Sticky. It returns
// ErrInStep, ErrWorldFailed, then, for the agent, ErrUnknownAgent, ErrAgentRemoved,
// ErrNotExternal and ErrNotDirected, then for the directive ErrUnknownRecipe,
// ErrOutOfBounds, ErrNotOnRole, ErrUnknownCommodity, ErrDuplicateID and
// ErrNotTradable, in the order of 4.3.2; a refused call changes nothing. It
// emits no event and checks nothing a tick can change (whether an order is
// affordable): the step refuses that.
func (w *World) Direct(id AgentID, d Directive) error {
	if err := w.enter("Direct", "", false); err != nil {
		return err
	}
	a, err := w.externalAgent(id)
	if err == nil && !isDirected(w.strategyOf(a)) {
		err = ErrNotDirected
	}
	if err != nil {
		return fmt.Errorf("Direct: agent %d: %w", id, err)
	}
	if err := w.checkDirective(a, &d); err != nil {
		return fmt.Errorf("Direct: agent %d: %w", id, err)
	}
	a.directive = d.mailbox()
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "directive set", slog.Int64(logkeys.Tick, w.Tick()), slog.Int64(logkeys.Agent, int64(id)))
	}
	return nil
}

// mailbox is the directive as the agent keeps it: a deep copy, so the caller's
// slices can change after the call, or nil for a directive with nothing in it.
func (d *Directive) mailbox() *Directive {
	if d == nil || d.Job == nil && len(d.Intents) == 0 && len(d.Discards) == 0 && len(d.Hold) == 0 {
		return nil
	}
	c := Directive{Sticky: d.Sticky}
	if d.Job != nil {
		j := *d.Job
		c.Job = &j
	}
	c.Intents = slices.Clone(d.Intents)
	c.Discards = slices.Clone(d.Discards)
	c.Hold = slices.Clone(d.Hold)
	return &c
}

// checkDirective is rows 2 to 6 of the table of spec-integration.md 4.3.2, in
// order. Row 1 (the module of Offer, Hire, Borrow and Lend) and row 7 wait for the
// modules that own those fields (plan 2-17 Scope).
func (w *World) checkDirective(a *agent, d *Directive) error {
	if err := w.checkDirectedJob(a, d.Job); err != nil {
		return err
	}
	if err := w.checkDirectedIntents(d.Intents); err != nil {
		return err
	}
	row := w.roles.row(a.role)
	for i, in := range d.Intents {
		if _, ok := slices.BinarySearch(row.tradable, in.Commodity); !ok {
			return argError(ErrNotTradable, itoa(int(in.Commodity)), "", "d", entry("intents", i), "commodity")
		}
	}
	if err := w.checkDirectedDiscards(d.Discards); err != nil {
		return err
	}
	return w.checkDirectedHold(d.Hold)
}

// checkDirectedJob is rows 2 and 3: a recipe that exists and is on the role's list
// (or its own-use list), and Executions from 0 to the agent's weight. A request for
// no job has nothing to check.
func (w *World) checkDirectedJob(a *agent, j *JobRequest) error {
	if j == nil || j.NoJob {
		return nil
	}
	if j.Recipe < 0 || int(j.Recipe) >= len(w.cfg.Recipes) {
		return argError(ErrUnknownRecipe, itoa(int(j.Recipe)), "", "d", "job", "recipe")
	}
	if j.Executions < 0 || j.Executions > a.weight {
		return argError(ErrOutOfBounds, itoa(int(j.Executions)), "weight", "d", "job", "executions")
	}
	if reason := admit(w.roles.row(a.role).jobs, j.Recipe); reason != "" {
		return argError(ErrNotOnRole, itoa(int(j.Recipe)), "", "d", "job", "recipe")
	}
	return nil
}

// known reports whether c names a commodity of the configuration.
func (w *World) known(c CommodityID) bool { return c >= 0 && int(c) < len(w.cfg.Commodities) }

// checkDirectedIntents is row 4: for each intent, in order, a known commodity, a
// valid side and purpose, and each amount in its bound; then the commodity not
// seen in an earlier intent.
func (w *World) checkDirectedIntents(intents []Intent) error {
	for i, in := range intents {
		seg := entry("intents", i)
		if !w.known(in.Commodity) {
			return argError(ErrUnknownCommodity, itoa(int(in.Commodity)), "", "d", seg, "commodity")
		}
		for _, f := range [...]struct {
			name, bound string
			v, lo, hi   int64
		}{
			{"side", "", int64(in.Side), int64(Bid), int64(Ask)},
			{"purpose", "", int64(in.Purpose), int64(PurposeOther), int64(PurposeTool)},
			{"quantity", "MaxQuantity", int64(in.Quantity), 1, MaxQuantity},
			{"limit", "MaxPrice", int64(in.Limit), 1, MaxPrice},
			{"target", "MaxQuantity", int64(in.Target), 0, MaxQuantity},
			{"reserve", "MaxQuantity", int64(in.Reserve), 0, MaxQuantity},
			{"unit_cost", "MaxPrice", int64(in.UnitCost), 0, MaxPrice},
		} {
			if f.v < f.lo || f.v > f.hi {
				return argError(ErrOutOfBounds, itoa(int(f.v)), f.bound, "d", seg, f.name)
			}
		}
		for _, earlier := range intents[:i] {
			if earlier.Commodity == in.Commodity {
				return argError(ErrDuplicateID, itoa(int(in.Commodity)), "", "d", seg, "commodity")
			}
		}
	}
	return nil
}

// checkDirectedDiscards is the discards of row 6: a known commodity, a quantity in
// [1, MaxQuantity], each commodity once.
func (w *World) checkDirectedDiscards(discards []Discard) error {
	for i, ds := range discards {
		seg := entry("discards", i)
		switch {
		case !w.known(ds.Commodity):
			return argError(ErrUnknownCommodity, itoa(int(ds.Commodity)), "", "d", seg, "commodity")
		case ds.Quantity < 1 || ds.Quantity > MaxQuantity:
			return argError(ErrOutOfBounds, itoa(int(ds.Quantity)), "MaxQuantity", "d", seg, "quantity")
		}
		for _, earlier := range discards[:i] {
			if earlier.Commodity == ds.Commodity {
				return argError(ErrDuplicateID, itoa(int(ds.Commodity)), "", "d", seg, "commodity")
			}
		}
	}
	return nil
}

// checkDirectedHold is the hold list of row 6: each a known commodity, once.
func (w *World) checkDirectedHold(hold []CommodityID) error {
	for i, c := range hold {
		if !w.known(c) {
			return argError(ErrUnknownCommodity, itoa(int(c)), "", "d", entry("hold", i))
		}
		if slices.Contains(hold[:i], c) {
			return argError(ErrDuplicateID, itoa(int(c)), "", "d", entry("hold", i))
		}
	}
	return nil
}
