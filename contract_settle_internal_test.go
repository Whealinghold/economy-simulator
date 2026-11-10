package economysim

import (
	"errors"
	"log/slog"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/merchant"
)

// This file tests the settlement, the removal step, the open_contract refusal, the
// audit, the snapshot rows and the checkpoint section of the delivery contract (plan
// 2-15 Phase 7). The agents are those of trio (contract_round_internal_test.go): the
// deliverers 3, 4 and 5 in a_home and the payers 8, 9 and 10 in b_far.

// oneGrain is the plan of one contract of 10 grain from d32 to p61: an offer limit
// of 90 and a take limit of 110 give the price (90 + 110) / 2 = 100 and an escrow of
// 10 * 100 = 1 000. The coach takes 2 ticks (40 distance units at speed 30), so a
// contract formed in tick 0 arrives, and is settled, in tick 2.
func oneGrain() roundPlan {
	return roundPlan{
		offers: map[AgentID][]DeliveryOffer{d32: {{Mode: modeCoach, To: regFar, Commodity: cGrain, Quantity: 10, Limit: 90}}},
		takes:  map[AgentID][]DeliveryTake{p61: {{Commodity: cGrain, Quantity: 10, Limit: 110}}},
	}
}

// settled returns the ContractSettled events of the recorder.
func settled(rec *evLog) []ContractSettled { return of[ContractSettled](rec) }

// TestSettle_P14 verifies the table of spec-merchants.md 9.7 through the world: the
// contract of oneGrain is settled in the tick its shipment arrives, after
// ShipmentArrived, and the escrow is paid out and refunded by what reached.
//
// Arithmetic: everything arrived: 1 000 to the deliverer, 0 back. Short: 4 perish
// (3 born in tick 0, 1 in tick 1) dispatched in tick 1 arrive in tick 3, and a lot
// is delivered iff 1 + 2 <= born + 3 - 1, so the three older units spoil: price 100,
// paid 100 * 1 = 100, refunded 100 * 3 = 300 to the payer. Payer removed: the cargo
// is diverted to the estate and counts as reached, so 1 000 is paid and the cause is
// payer_removed.
//
// Requirement: FR-MER-040, FR-MER-056, FR-CAR-001
func TestSettle_P14(t *testing.T) {
	t.Parallel()
	t.Run("delivered", func(t *testing.T) {
		t.Parallel()
		w, rec, _ := roundWorld(t, trio, oneGrain(), d32)
		before := w.cash(&w.agents[d32])
		step(t, w, 3)
		got := settled(rec)
		want := ContractSettled{Tick: 2, Contract: 0, Payer: p61, Deliverer: d32, Region: regHome, To: regFar, Commodity: cGrain,
			Quantity: 10, Reached: 10, Paid: 1_000, Cause: CauseDelivered}
		if len(got) != 1 || got[0] != want {
			t.Fatalf("ContractSettled = %+v, want one: %+v", got, want)
		}
		if after := w.cash(&w.agents[d32]); after-before != 1_000 {
			t.Errorf("the deliverer's money rose by %d, want 1 000", after-before)
		}
		if len(w.mer.open) != 0 || w.mer.free[regFar].Len() != 1 {
			t.Errorf("open contracts %d, free Escrow accounts %d, want 0 and 1", len(w.mer.open), w.mer.free[regFar].Len())
		}
		arrived, ended := -1, -1
		for i, e := range rec.events {
			switch e.(type) {
			case ShipmentArrived:
				arrived = i
			case ContractSettled:
				ended = i
			}
		}
		if arrived < 0 || ended < arrived {
			t.Errorf("ShipmentArrived at %d, ContractSettled at %d, want the settlement after the arrival", arrived, ended)
		}
	})
	t.Run("short", func(t *testing.T) {
		t.Parallel()
		plan := roundPlan{at: 1,
			offers: map[AgentID][]DeliveryOffer{d32: {{Mode: modeCoach, To: regFar, Commodity: cPerish, Quantity: 4, Limit: 90}}},
			takes:  map[AgentID][]DeliveryTake{p61: {{Commodity: cPerish, Quantity: 4, Limit: 110}}},
		}
		trade := func(c *Config) {
			trio(c)
			c.Roles[len(c.Roles)-1].Trades = append(c.Roles[len(c.Roles)-1].Trades, "perish")
		}
		w, rec, _ := roundWorld(t, trade, plan, d32)
		give(t, w, d32, cPerish, 3)
		step(t, w, 1)
		give(t, w, d32, cPerish, 1)
		before := w.cash(&w.agents[p61])
		step(t, w, 3)
		got := settled(rec)
		want := ContractSettled{Tick: 3, Contract: 0, Payer: p61, Deliverer: d32, Region: regHome, To: regFar, Commodity: cPerish,
			Quantity: 4, Reached: 1, Spoiled: 3, Paid: 100, Refunded: 300, Cause: CauseShort}
		if len(got) != 1 || got[0] != want {
			t.Fatalf("ContractSettled = %+v, want one: %+v", got, want)
		}
		if after := w.cash(&w.agents[p61]); after-before != -100 {
			t.Errorf("the payer's money changed by %d across the contract, want -100 (400 escrowed, 300 back)", after-before)
		}
	})
	t.Run("payer removed", func(t *testing.T) {
		t.Parallel()
		w, rec, _ := roundWorld(t, trio, oneGrain(), d32)
		step(t, w, 1)
		if err := w.removeAgent(&w.agents[p61], CauseConsumer); err != nil {
			t.Fatalf("removeAgent() = %v, want nil", err)
		}
		step(t, w, 2)
		got := settled(rec)
		want := ContractSettled{Tick: 2, Contract: 0, Payer: p61, Deliverer: d32, Region: regHome, To: regFar, Commodity: cGrain,
			Quantity: 10, Reached: 10, Paid: 1_000, Cause: CausePayerRemoved}
		if len(got) != 1 || got[0] != want {
			t.Fatalf("ContractSettled = %+v, want one: %+v", got, want)
		}
		if arr := of[ShipmentArrived](rec); len(arr) != 1 || !arr[0].Diverted || arr[0].Receiver != AgentID(regFar) {
			t.Errorf("ShipmentArrived = %+v, want the cargo diverted to the estate of b_far", arr)
		}
	})
}

// TestSettle_ContractIDOrder verifies that contracts whose shipments arrive in one
// tick are settled in ascending contract ID (spec-merchants.md 9.7): two contracts of
// the same deliverer and payer formed in one round are settled 0 then 1.
//
// Requirement: FR-MER-040, FR-MER-056
func TestSettle_ContractIDOrder(t *testing.T) {
	t.Parallel()
	o := func(limit Price) DeliveryOffer {
		return DeliveryOffer{Mode: modeCoach, To: regFar, Commodity: cGrain, Quantity: 5, Limit: limit}
	}
	plan := roundPlan{
		offers: map[AgentID][]DeliveryOffer{d32: {o(10), o(11)}},
		takes:  map[AgentID][]DeliveryTake{p61: {{Commodity: cGrain, Quantity: 10, Limit: 100}}},
	}
	w, rec, _ := roundWorld(t, trio, plan, d32)
	step(t, w, 3)
	got := settled(rec)
	if len(got) != 2 || got[0].Contract != 0 || got[1].Contract != 1 || got[0].Tick != 2 || got[1].Tick != 2 {
		t.Errorf("ContractSettled = %+v, want contracts 0 then 1 in tick 2", got)
	}
}

// TestRemoval_Deliverer verifies spec-merchants.md 9.8.2 and FR-CAR-002: the removal
// of the deliverer of an open contract ends it at once with the whole escrow returned
// to the payer, ContractSettled comes before AgentBankrupted, the shipment is not
// touched, the payer receives the goods at the arrival and nothing settles then.
//
// Requirement: FR-MER-041, FR-CAR-002, FR-MER-056
func TestRemoval_Deliverer(t *testing.T) {
	t.Parallel()
	w, rec, _ := roundWorld(t, trio, oneGrain(), d32)
	cash := w.cash(&w.agents[p61])
	step(t, w, 1)
	if err := w.removeAgent(&w.agents[d32], CauseInsolvent); err != nil {
		t.Fatalf("removeAgent() = %v, want nil", err)
	}
	got := settled(rec)
	want := ContractSettled{Tick: 1, Contract: 0, Payer: p61, Deliverer: d32, Region: regHome, To: regFar, Commodity: cGrain,
		Quantity: 10, Refunded: 1_000, Cause: CauseDelivererRemoved}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("ContractSettled = %+v, want one: %+v", got, want)
	}
	if after := w.cash(&w.agents[p61]); after != cash {
		t.Errorf("the payer holds %d, want the %d it had before the contract", after, cash)
	}
	ended, bankrupt := -1, -1
	for i, e := range rec.events {
		switch e.(type) {
		case ContractSettled:
			ended = i
		case AgentBankrupted:
			bankrupt = i
		}
	}
	if ended < 0 || bankrupt < ended {
		t.Errorf("ContractSettled at %d, AgentBankrupted at %d, want the settlement first", ended, bankrupt)
	}
	if len(w.transport.ships) != 1 {
		t.Fatalf("shipments in flight = %d, want the one of the contract untouched", len(w.transport.ships))
	}
	step(t, w, 2)
	arr := of[ShipmentArrived](rec)
	if len(arr) != 1 || arr[0].Diverted || arr[0].Receiver != p61 || len(settled(rec)) != 1 {
		t.Errorf("ShipmentArrived = %+v and %d settlements, want the goods to the payer and no second settlement", arr, len(settled(rec)))
	}
}

// TestRemoval_Payer verifies spec-merchants.md 9.8.2: the removal of the payer leaves
// the contract standing (the escrow is a separate account), a spoiled part is refunded
// to the estate of the destination at the arrival, and the deliverer is paid for the
// units diverted as for the units delivered.
//
// Arithmetic: as the short case of TestSettle_P14, 4 perish at 100 with 3 spoiled: 100
// to the deliverer and 300 to the estate of b_far.
//
// Requirement: FR-MER-042
func TestRemoval_Payer(t *testing.T) {
	t.Parallel()
	plan := roundPlan{at: 1,
		offers: map[AgentID][]DeliveryOffer{d32: {{Mode: modeCoach, To: regFar, Commodity: cPerish, Quantity: 4, Limit: 90}}},
		takes:  map[AgentID][]DeliveryTake{p61: {{Commodity: cPerish, Quantity: 4, Limit: 110}}},
	}
	trade := func(c *Config) {
		trio(c)
		c.Roles[len(c.Roles)-1].Trades = append(c.Roles[len(c.Roles)-1].Trades, "perish")
	}
	w, rec, _ := roundWorld(t, trade, plan, d32)
	give(t, w, d32, cPerish, 3)
	step(t, w, 1)
	give(t, w, d32, cPerish, 1)
	step(t, w, 1)
	if err := w.removeAgent(&w.agents[p61], CauseStarved); err != nil {
		t.Fatalf("removeAgent() = %v, want nil", err)
	}
	if len(w.mer.open) != 1 || len(settled(rec)) != 0 {
		t.Fatalf("open contracts %d, settlements %d after the payer's removal, want the contract standing", len(w.mer.open), len(settled(rec)))
	}
	estate := w.estates.accounts[regFar]
	view, _ := w.ledger.Account(estate)
	step(t, w, 2)
	got := settled(rec)
	want := ContractSettled{Tick: 3, Contract: 0, Payer: p61, Deliverer: d32, Region: regHome, To: regFar, Commodity: cPerish,
		Quantity: 4, Reached: 1, Spoiled: 3, Paid: 100, Refunded: 300, Cause: CausePayerRemoved}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("ContractSettled = %+v, want one: %+v", got, want)
	}
	if after, _ := w.ledger.Account(estate); after.Balance < view.Balance+300 {
		t.Errorf("the estate of b_far holds %d, want at least the %d it held plus the 300 refunded", after.Balance, view.Balance)
	}
}

// TestRefusal_OpenContract verifies spec-merchants.md 9.8.1: the payer and the
// deliverer of an open contract are refused a relocation with ErrRegionBound naming
// open_contract (the open_shipment refusal of the same parties is skipped by name
// here), the instance keys are the contract IDs, and a deliverer with two contracts
// is still refused when one of them is skipped.
//
// Requirement: FR-MER-043
func TestRefusal_OpenContract(t *testing.T) {
	t.Parallel()
	o := func(limit Price) DeliveryOffer {
		return DeliveryOffer{Mode: modeCoach, To: regFar, Commodity: cGrain, Quantity: 5, Limit: limit}
	}
	plan := roundPlan{
		offers: map[AgentID][]DeliveryOffer{d32: {o(10), o(11)}},
		takes:  map[AgentID][]DeliveryTake{p61: {{Commodity: cGrain, Quantity: 10, Limit: 100}}},
	}
	w, _, _ := roundWorld(t, trio, plan, d32)
	step(t, w, 1)
	for _, party := range []struct {
		name string
		id   AgentID
		to   RegionID
	}{{"payer", p61, regHome}, {"deliverer", d32, regFar}} {
		_, err := w.RelocationRefusal(party.id, party.to, "open_shipment")
		var rb *RegionBoundError
		if !errors.Is(err, ErrRegionBound) || !errors.As(err, &rb) || rb.Obligation != "open_contract" {
			t.Errorf("RelocationRefusal(%s, skipping open_shipment) = %v, want ErrRegionBound naming open_contract", party.name, err)
		}
	}
	for _, tt := range []struct {
		name string
		skip []string
		held bool
	}{
		{"one of two contracts skipped", []string{"open_contract:0"}, true},
		{"the other one skipped", []string{"open_contract:1"}, true},
		{"both skipped", []string{"open_contract:0", "open_contract:1"}, false},
		{"skipped by name", []string{"open_contract"}, false},
		{"a contract that is not the agent's", []string{"open_contract:7"}, true},
	} {
		if got := w.mer.holds(&w.agents[d32], tt.skip); got != tt.held {
			t.Errorf("holds(deliverer, %v) (%s) = %t, want %t", tt.skip, tt.name, got, tt.held)
		}
	}
	if w.mer.holds(&w.agents[d35], nil) {
		t.Error("holds(agent 4, nil) = true, want false: it is a party to no contract")
	}
}

// TestArrival_SelfCarriedWithOpenContract verifies the arrival guard of
// spec-merchants.md 9.8.1 and ruling C16: a deliverer that carries its own lot arrives
// at P1.3 while its contract is open and the arrival is not held, because the
// contract formed on the arriving shipment is skipped by instance; it settles at P1.4
// in the same tick, with the deliverer's region at the event the destination.
//
// Arithmetic: the foot mode has speed 20, so 40 distance units take 2 ticks: dispatched
// in tick 0, arrives in tick 2. The trip carries the lot of 8 grain (mass 24, bulk 16)
// and the 2 grain of upkeep the traveller needs for 2 ticks (mass 6, bulk 4) in one
// carrying unit of 40 and 30; the escrow is 8 * 100 = 800.
//
// Requirement: FR-MER-043, FR-TRN-022
func TestArrival_SelfCarriedWithOpenContract(t *testing.T) {
	t.Parallel()
	plan := oneGrain()
	plan.offers[d32][0].Mode, plan.offers[d32][0].Quantity = modeFoot, 8
	foot := func(c *Config) {
		trio(c)
		c.Roles[len(c.Roles)-1].Merchant.Modes = []string{"coach", "foot"}
	}
	// The deliverer's holdings travel with it (spec-merchants.md 9.5 step 3), so it holds the lot alone.
	w, rec, _ := newMerchWorld(t, foot, plan.script())
	give(t, w, d32, cGrain, 10)
	step(t, w, 3)
	if g := w.regionOf(&w.agents[d32]); g != regFar {
		t.Errorf("the deliverer is in region %d after its arrival, want b_far (%d): the arrival was held", g, regFar)
	}
	got := settled(rec)
	if len(got) != 1 || got[0].Tick != 2 || got[0].Region != regFar || got[0].Paid != 800 || got[0].Cause != CauseDelivered {
		t.Errorf("ContractSettled = %+v, want one in tick 2 from b_far, 800 paid", got)
	}
}

// TestConservation_Escrow verifies spec-merchants.md 9.6.1 (FR-MER-044): over the
// whole life of a contract, formation, flight and settlement, with the audit on, the
// money of the accounts is conserved, the snapshot's total includes the escrow, and no
// BoundaryFlow names an escrow reason.
//
// Requirement: FR-MER-044, FR-CAR-001
func TestConservation_Escrow(t *testing.T) {
	t.Parallel()
	w, rec, _ := roundWorld(t, trio, oneGrain(), d32)
	total := func() (sum int64) {
		for id := range w.ledger.Len() {
			if v, err := w.ledger.Account(id); err == nil && v.Kind != ledger.KindBoundary {
				sum += v.Balance
			}
		}
		return sum
	}
	want := total()
	for tick := range 4 {
		step(t, w, 1)
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() = %v, want nil", err)
		}
		if got := total(); got != want || int64(s.Money) != want {
			t.Errorf("tick %d: money in the accounts %d, snapshot %d, want %d everywhere", tick, got, s.Money, want)
		}
	}
	for _, f := range of[BoundaryFlow](rec) {
		if f.Reason == ReasonEscrowIn || f.Reason == ReasonEscrowOut || f.Reason == ReasonEscrowRefund {
			t.Errorf("BoundaryFlow %+v names an escrow reason, want none", f)
		}
	}
}

// TestAudit_Contracts verifies FR-MER-045: a clean world passes, and each violation
// fails the audit with ErrAuditFailed: an Escrow account that holds more than price *
// quantity, an unreferenced Escrow account that is not empty, a contract ID that is
// not ascending, two contracts on one account, and a shipment that is not in flight.
//
// Requirement: FR-MER-045
func TestAudit_Contracts(t *testing.T) {
	t.Parallel()
	o := func(limit Price) DeliveryOffer {
		return DeliveryOffer{Mode: modeCoach, To: regFar, Commodity: cGrain, Quantity: 5, Limit: limit}
	}
	plan := roundPlan{
		offers: map[AgentID][]DeliveryOffer{d32: {o(10), o(11)}},
		takes:  map[AgentID][]DeliveryTake{p61: {{Commodity: cGrain, Quantity: 10, Limit: 100}}},
	}
	for _, tt := range []struct {
		name string
		edit func(t *testing.T, w *World)
	}{
		{"clean", nil},
		{"an escrow that holds too much", func(t *testing.T, w *World) {
			if err := w.ledger.Transfer(w.agents[p61].account, w.mer.open[0].escrow, 1, ledger.EscrowIn); err != nil {
				t.Fatalf("Transfer() = %v, want nil", err)
			}
		}},
		{"a free escrow that holds money", func(t *testing.T, w *World) {
			id, err := w.ledger.Open(ledger.KindEscrow, 0, int(regFar))
			if err == nil {
				err = w.ledger.Transfer(w.agents[p61].account, id, 1, ledger.EscrowIn)
			}
			if err != nil {
				t.Fatalf("filling a free Escrow account = %v, want nil", err)
			}
		}},
		{"IDs not ascending", func(_ *testing.T, w *World) { w.mer.open[0].id, w.mer.open[1].id = 1, 0 }},
		{"ID not below the next", func(_ *testing.T, w *World) { w.mer.next = 1 }},
		{"one account for two contracts", func(_ *testing.T, w *World) { w.mer.open[1].escrow = w.mer.open[0].escrow }},
		{"a shipment that is not in flight", func(_ *testing.T, w *World) { w.mer.open[0].shipment = 99 }},
		{"an account that is not an Escrow account", func(_ *testing.T, w *World) { w.mer.open[0].escrow = w.agents[p61].account }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _, _ := roundWorld(t, trio, plan, d32)
			step(t, w, 1)
			if len(w.mer.open) != 2 {
				t.Fatalf("open contracts = %d, want 2", len(w.mer.open))
			}
			if tt.edit != nil {
				tt.edit(t, w)
			}
			err := w.auditContracts()
			if bad := tt.edit != nil; bad != errors.Is(err, ErrAuditFailed) {
				t.Errorf("auditContracts(%s) = %v, want ErrAuditFailed: %t", tt.name, err, bad)
			}
		})
	}
}

// TestSnapshot_Contracts verifies the rows of spec-merchants.md 15.4 and
// spec-observability.md 5.3.8: the contracts in flight in ascending ID with their
// ticks left, the escrow in total and by region, a deep copy, and nothing with the
// module off.
//
// Arithmetic: formed in tick 0 and arriving in tick 2, so after the step of tick 0 the
// ticks left are 2 - 1 = 1; the escrow is 10 * 100 = 1 000.
//
// Requirement: FR-MER-044, FR-OBS-020
func TestSnapshot_Contracts(t *testing.T) {
	t.Parallel()
	w, _, _ := roundWorld(t, trio, oneGrain(), d32)
	s0, _ := w.Snapshot()
	if s0.Contracts != nil || s0.Escrow != 0 {
		t.Errorf("snapshot before any contract = %+v, %d, want none", s0.Contracts, s0.Escrow)
	}
	step(t, w, 1)
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() = %v, want nil", err)
	}
	want := []ContractRow{{Contract: 0, Payer: p61, Deliverer: d32, Region: regHome, To: regFar, Commodity: cGrain, Quantity: 10,
		Price: 100, Escrowed: 1_000, Shipment: w.mer.open[0].shipment, Formed: 0, Arrives: 2, TicksLeft: 1}}
	if !reflect.DeepEqual(s.Contracts, want) || s.Escrow != 1_000 || s.Regions[regFar].MoneyContract != 1_000 || s.Regions[regHome].MoneyContract != 0 {
		t.Errorf("Contracts %+v, Escrow %d, MoneyContract home %d far %d, want %+v, 1 000 and 1 000 in b_far", s.Contracts, s.Escrow, s.Regions[regHome].MoneyContract, s.Regions[regFar].MoneyContract, want)
	}
	s.Contracts[0].Quantity = 1
	if again, _ := w.Snapshot(); again.Contracts[0].Quantity != 10 {
		t.Error("a change of the returned rows reached the next snapshot, want a deep copy")
	}
	step(t, w, 2)
	if end, _ := w.Snapshot(); end.Contracts != nil || end.Escrow != 0 || end.Regions[regFar].MoneyContract != 0 {
		t.Errorf("snapshot after the settlement = %+v, %d, want none", end.Contracts, end.Escrow)
	}
}

// restoreMerch restores a checkpoint of a world built by roundWorld with the plan,
// whose roles name the strategy "script".
func restoreMerch(w *World, plan roundPlan, opts ...Option) (*World, error) {
	c, err := w.Checkpoint()
	if err != nil {
		return nil, err
	}
	opts = append([]Option{WithAudit(), WithStrategy("script", &merchScript{plan: plan.script()})}, opts...)
	return RestoreWorld(w.Config(), c, opts...)
}

// TestCheckpoint_Contracts verifies tag 21 (spec-merchants.md 16, FR-MER-060): the
// next contract ID and the contracts in flight round-trip in ID order, the free list
// of Escrow accounts is rebuilt, and the restored world's checkpoint equals the
// original's.
//
// Requirement: FR-MER-060, FR-CHK-016
func TestCheckpoint_Contracts(t *testing.T) {
	t.Parallel()
	w, _, _ := roundWorld(t, trio, oneGrain(), d32)
	step(t, w, 1)
	r, err := restoreMerch(w, oneGrain())
	if err != nil {
		t.Fatalf("RestoreWorld() = %v, want nil", err)
	}
	if !reflect.DeepEqual(r.mer.open, w.mer.open) || r.mer.next != w.mer.next || r.mer.next != 1 {
		t.Errorf("contracts %+v next %d after the restore, want %+v next 1", r.mer.open, r.mer.next, w.mer.open)
	}
	a, _ := w.Checkpoint()
	b, _ := r.Checkpoint()
	if !reflect.DeepEqual(a, b) {
		t.Error("the restored world's checkpoint differs from the one it was restored from")
	}
	step(t, r, 2) // the settlement frees the account the restore rebuilt as in use
	if r.mer.free[regFar].Len() != 1 || len(r.mer.open) != 0 {
		t.Errorf("free Escrow accounts %d, open contracts %d after the settlement, want 1 and 0", r.mer.free[regFar].Len(), len(r.mer.open))
	}
}

// TestRestore_ContractEscrowCrossCheck verifies that a restore refuses a section 21
// that no world could have written (FR-CHK-023, FR-MER-061): a contract out of order,
// beyond the next ID, with a shipment that is not in flight or is not its own, an
// account that is missing, is not an Escrow account or holds the wrong amount, or a
// party that is an estate. Each case edits the state of a world with one contract in
// flight before it is saved.
//
// Requirement: FR-CHK-023, FR-MER-061
func TestRestore_ContractEscrowCrossCheck(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		edit func(w *World)
	}{
		{"next contract not above the contracts", func(w *World) { w.mer.next = 0 }},
		{"shipment that is not in flight", func(w *World) { w.mer.open[0].shipment = 99 }},
		{"shipment of another deliverer", func(w *World) { w.mer.open[0].deliverer = d35 }},
		{"payer that is an estate", func(w *World) { w.mer.open[0].payer = 0 }},
		{"payer that is the deliverer", func(w *World) { w.mer.open[0].payer = w.mer.open[0].deliverer }},
		{"price that is not what the account holds", func(w *World) { w.mer.open[0].price++ }},
		{"quantity of 0", func(w *World) { w.mer.open[0].qty = 0 }},
		{"account beyond the ledger", func(w *World) { w.mer.open[0].escrow = w.ledger.Len() + 3 }},
		{"account that is not an Escrow account", func(w *World) { w.mer.open[0].escrow = w.agents[p61].account }},
		{"destination that is not the shipment's", func(w *World) { w.mer.open[0].to = regHome }},
		{"formation after the checkpoint", func(w *World) { w.mer.open[0].formed = w.Tick() }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, _, _ := roundWorld(t, trio, oneGrain(), d32)
			step(t, w, 1)
			tt.edit(w)
			_, err := restoreMerch(w, oneGrain())
			if !errors.Is(err, ErrCheckpointCorrupt) {
				t.Errorf("RestoreWorld(%s) error = %v, want ErrCheckpointCorrupt", tt.name, err)
			}
		})
	}
}

// TestMerchantsCross_RemovedDeliverer verifies that the cross-check of a restore
// refuses a contract whose deliverer is not a live agent (a contract ends at the
// removal of its deliverer).
//
// Requirement: FR-CHK-023, FR-MER-061
func TestMerchantsCross_RemovedDeliverer(t *testing.T) {
	t.Parallel()
	w, _, _ := roundWorld(t, trio, oneGrain(), d32)
	step(t, w, 1)
	if err := w.merchantsCross(); err != nil {
		t.Fatalf("merchantsCross() = %v, want nil on a live deliverer", err)
	}
	w.agents[w.mer.open[0].deliverer].weight = 0
	if err := w.merchantsCross(); !errors.Is(err, ErrCheckpointCorrupt) {
		t.Errorf("merchantsCross() = %v, want ErrCheckpointCorrupt", err)
	}
}

// TestFork_Merchants verifies FR-MER-061: a world restored between ticks with a
// contract in flight settles the same contract at the same tick and gives the same
// events and the same snapshot as the uninterrupted one.
//
// Requirement: FR-MER-061, FR-CHK-016
func TestFork_Merchants(t *testing.T) {
	t.Parallel()
	plan := oneGrain()
	w, rec, _ := roundWorld(t, trio, plan, d32)
	step(t, w, 1) // the contract is formed in tick 0 and settles in tick 2, after the cut
	cut := len(rec.events)
	rrec := &evLog{}
	r, err := restoreMerch(w, plan, WithObserver(rrec))
	if err != nil {
		t.Fatalf("RestoreWorld() = %v, want nil", err)
	}
	step(t, w, 3)
	step(t, r, 3)
	// The first event of a restore is WorldRestored; the rest are the ticks after it.
	if len(rrec.events) < 1 || !reflect.DeepEqual(rec.events[cut:], rrec.events[1:]) {
		t.Errorf("events after the restore: %d, want the %d of the uninterrupted world after WorldRestored", len(rrec.events), len(rec.events)-cut)
	}
	if n := len(settled(rec)); n != 1 || len(of[ContractSettled](rrec)) != 1 {
		t.Errorf("settlements: %d uninterrupted, %d after the restore, want 1 and 1", n, len(of[ContractSettled](rrec)))
	}
	sa, _ := w.Snapshot()
	sb, _ := r.Snapshot()
	if !reflect.DeepEqual(sa, sb) {
		t.Error("snapshot of the restored world differs from the uninterrupted one")
	}
}

// TestLogs_Debug verifies FR-MER-058: the contract writes "contract formed" and
// "contract settled" at Debug with the keys of spec-merchants.md 15.2, nothing above
// Debug during the ticks, and no Info line in the tick.
//
// Requirement: FR-MER-058
func TestLogs_Debug(t *testing.T) {
	t.Parallel()
	lines := newCapture()
	w, _, _ := newMerchWorld(t, trio, oneGrain().script(), WithLogger(slog.New(lines)))
	stock(t, w, d32)
	mark := len(*lines.lines)
	step(t, w, 3)
	for _, l := range (*lines.lines)[mark:] {
		if l.level == slog.LevelInfo || (l.level > slog.LevelDebug && l.msg != "group below floor") {
			t.Errorf("line %q at level %v in the tick, want Debug at most and no Info", l.msg, l.level)
		}
	}
	var formed, ended []logRec
	for _, l := range *lines.lines {
		switch l.msg {
		case "contract formed":
			formed = append(formed, l)
		case "contract settled":
			ended = append(ended, l)
		}
	}
	if len(formed) != 1 || formed[0].attrs["contract"] != 0 || formed[0].attrs["agent"] != int64(d32) || formed[0].attrs["price"] != 100 || formed[0].attrs["dest"] != int64(regFar) {
		t.Errorf("contract formed lines = %+v, want one for contract 0 by agent %d at price 100", formed, d32)
	}
	if len(ended) != 1 || ended[0].attrs["contract"] != 0 || ended[0].strs["reason"] != string(CauseDelivered) {
		t.Errorf("contract settled lines = %+v, want one for contract 0 with reason delivered", ended)
	}
}

// TestAlloc_NoObserver verifies NFR-MER-001 and NFR-MER-004: with the default logger
// and observer the admission of an offer and a take, the ordering of the lane and the
// settlement allocate nothing once their scratch has grown.
//
// Requirement: NFR-MER-001, NFR-MER-004
func TestAlloc_NoObserver(t *testing.T) {
	w, _, _ := newMerchWorld(t, trio, oneGrain().script())
	stock(t, w, d32)
	w.obs.supplied = false
	m := w.mer
	m.subOffers = append(m.subOffers[:0], submittedOffer{agent: d32, o: DeliveryOffer{Mode: modeCoach, To: regFar, Commodity: cGrain, Quantity: 10, Limit: 90}})
	m.subTakes = append(m.subTakes[:0], submittedTake{agent: p61, t: DeliveryTake{Commodity: cGrain, Quantity: 10, Limit: 110}})
	admit := func() {
		m.offers, m.takes, m.rOffers, m.rTakes = m.offers[:0], m.takes[:0], m.rOffers[:0], m.rTakes[:0]
		w.admitOffers()
		w.admitTakes()
		m.sortLanes()
		merchant.SortOffers(m.rOffers)
		merchant.SortBids(m.rTakes)
	}
	admit()
	if len(m.offers) != 1 || len(m.takes) != 1 {
		t.Fatalf("admitted %d offers and %d takes, want 1 and 1", len(m.offers), len(m.takes))
	}
	if got := testing.AllocsPerRun(100, admit); got != 0 {
		t.Errorf("allocations per admission and sort = %v, want 0", got)
	}
	m.offers, m.takes, m.rOffers, m.rTakes, m.subOffers, m.subTakes = m.offers[:0], m.takes[:0], m.rOffers[:0], m.rTakes[:0], m.subOffers[:0], m.subTakes[:0]
	step(t, w, 1) // forms the contract: its escrow holds 1 000
	ct := m.open[0]
	line := ArrivedLine{Commodity: cGrain, Shipped: 10, Delivered: 10}
	// One run settles the contract and puts it back as it was, so every run settles it.
	run := func() {
		_ = w.settleArrived(&ct, line)
		_ = w.ledger.Transfer(w.agents[d32].account, ct.escrow, 1_000, ledger.EscrowIn)
		m.free[regFar].Take()
	}
	run()
	if got := testing.AllocsPerRun(100, run); got != 0 {
		t.Errorf("allocations per settlement = %v, want 0", got)
	}
}
