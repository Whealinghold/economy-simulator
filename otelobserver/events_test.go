package otelobserver_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// allEvents is one zero value of every event type of the core's event.go
// (50), kept by hand because Go cannot list the types of a package. The root
// test of plan 2-22 Phase 9 compares its names with event.go. YieldShocked
// (plan 2-17) joins the list when that plan merges.
var allEvents = []economysim.Event{
	economysim.WorldCreated{},
	economysim.RunStarted{},
	economysim.RunEnded{},
	economysim.TickStarted{},
	economysim.TickEnded{},
	economysim.OrderPlaced{},
	economysim.OrderRejected{},
	economysim.TradeExecuted{},
	economysim.MarketCleared{},
	economysim.BeliefsUpdated{},
	economysim.RecipeExecuted{},
	economysim.RecipeRefused{},
	economysim.RecipeAbandoned{},
	economysim.ProductionHalted{},
	economysim.UpkeepShortfall{},
	economysim.WantShortfall{},
	economysim.AgentSpawned{},
	economysim.AgentBankrupted{},
	economysim.EstateOpened{},
	economysim.WeightChanged{},
	economysim.RoleChanged{},
	economysim.EntryLimited{},
	economysim.BoundaryFlow{},
	economysim.WearApplied{},
	economysim.WorldRestored{},
	economysim.AgentRelocated{},
	economysim.EdgeStatusChanged{},
	economysim.ReliefPaid{},
	economysim.MoneyFlowLimited{},
	economysim.MoneyFlowEnded{},
	economysim.MoneyFlowScaled{},
	economysim.EndowmentChanged{},
	economysim.EndowmentAdjusted{},
	economysim.OwnUseChanged{},
	economysim.PopulationChanged{},
	economysim.PopulationScaled{},
	economysim.ShipmentDispatched{},
	economysim.ShipmentRejected{},
	economysim.ShipmentArrived{},
	economysim.TransitFactorChanged{},
	economysim.RumoursServed{},
	economysim.MigrationStarted{},
	economysim.MigrationReviewed{},
	economysim.MerchantsPlanned{},
	economysim.ContractFormed{},
	economysim.ContractSettled{},
	economysim.ContractRound{},
	economysim.OutsideTraded{},
	economysim.OutsideEnded{},
	economysim.OutsideShocked{},
}

// TestObserve_EveryEventTypeNoPanic sends the zero value of every event type to
// a fresh adapter: none panics and none records a point, because a zero event
// names no ID of a names table and a step never started.
//
// Requirement: FR-OBS-052
func TestObserve_EveryEventTypeNoPanic(t *testing.T) {
	for _, e := range allEvents {
		t.Run(eventName(e), func(t *testing.T) {
			h := newHarness(t)
			h.send(e)
			if got := h.collect(); len(got) != 0 {
				t.Errorf("Observe(%T{}) recorded %v, want no point", e, got)
			}
		})
	}
}

func eventName(e economysim.Event) string { return fmt.Sprintf("%T", e) }
