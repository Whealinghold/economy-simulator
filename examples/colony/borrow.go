package main

import (
	"context"
	"fmt"
	"io"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// borrowLayers are the layers of the borrowing colony: the pack and the credit layer.
var borrowLayers = append(append([]string{}, packLayers...), "pack/credit.toml")

// borrowTerm and borrowAsk are what the player asks: a loan of borrowAsk money, to be repaid in
// borrowTerm ticks. Both are placeholders of the example.
const (
	borrowTerm = 12
	borrowAsk  = 2_000
)

// Borrower runs the colony with the credit layer and one player, an algae tender added through
// the integration API with the directed strategy. At the first tick the player asks for a loan
// through Direct, the lenders (the drillers) fill it at the round, and the claim falls due borrowTerm
// ticks later, when the settlement takes the repayment out of the player's money and it asks
// again; the command prints one line a tick for the player: its money, the principal it
// owes and the tick its claim falls due. It uses the public API only.
func Borrower(out io.Writer, seed uint64, ticks int64) error {
	cfg, err := config.LoadFS(packFS, borrowLayers...)
	if err != nil {
		return fmt.Errorf("load the borrowing colony: %w", err)
	}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithAudit())
	if err != nil {
		return fmt.Errorf("build the world: %w", err)
	}
	role, ok := roleByID(cfg, "algae_tender")
	if !ok {
		return fmt.Errorf("the pack has no role algae_tender")
	}
	player, err := w.AddAgent(economysim.NewAgent{Region: 0, Role: role, Strategy: "directed", Money: 500})
	if err != nil {
		return fmt.Errorf("add the player: %w", err)
	}
	for range ticks {
		if err := askWhenFree(w, player); err != nil {
			return err
		}
		if _, err := w.Run(context.Background(), 1); err != nil {
			return fmt.Errorf("run: %w", err)
		}
		if err := playerLine(out, w, player); err != nil {
			return err
		}
	}
	return nil
}

// askWhenFree leaves the player's request in its mailbox. A directive applies to the next step only
// and replaces the request the role's default rule would make, so the player directs every tick: a
// request for borrowAsk when it owes nothing, and, while it owes, one for 1, below the module's
// min_principal, which the round refuses at admission and which keeps the default rule from borrowing
// in its place.
func askWhenFree(w *economysim.World, player economysim.AgentID) error {
	st, err := w.Agent(player)
	if err != nil {
		return fmt.Errorf("agent %d: %w", player, err)
	}
	principal := int64(borrowAsk)
	if st.Credit.Owed > 0 {
		principal = 1
	}
	ask := economysim.Directive{Borrow: &economysim.CreditRequest{Principal: principal, Term: borrowTerm, MaxRateBP: 2_000}}
	if err := w.Direct(player, ask); err != nil {
		return fmt.Errorf("direct the player: %w", err)
	}
	return nil
}

// roleByID returns the ID of the role named id.
func roleByID(cfg economysim.Config, id string) (economysim.RoleID, bool) {
	for i, r := range cfg.Roles {
		if r.ID == id {
			return economysim.RoleID(i), true
		}
	}
	return 0, false
}

// playerLine prints the player's money and what it owes after a tick.
func playerLine(out io.Writer, w *economysim.World, player economysim.AgentID) error {
	st, err := w.Agent(player)
	if err != nil {
		return fmt.Errorf("agent %d: %w", player, err)
	}
	_, err = fmt.Fprintf(out, "tick %d player money %d debt %d owed %d next due %d\n", w.Tick(), st.Money, st.Credit.Debt, st.Credit.Owed, st.Credit.NextDue)
	return err
}
