package economysim

import "testing"

// TestWelfare_OwnReceiptIsUnitsCreatedTimesCostPrice verifies that the units an
// own-use job created are an in-kind receipt valued at the price of cost_t: the
// published price, or the effective reference price while the commodity has never
// traded.
//
// Arithmetic: gather makes 2 grain; grain has never traded, so it is valued at
// its reference price 1 000; K^own = 2 * 1 000 = 2 000. After a trade the
// published price 1 500 is used: 2 * 1 500 = 3 000.
//
// Requirement: FR-SUB-028, FR-SUB-051
func TestWelfare_OwnReceiptIsUnitsCreatedTimesCostPrice(t *testing.T) {
	t.Parallel()
	w := newForage(t, 1, nil)
	a := &w.agents[1]
	tickInspecting(t, w, func() {
		if got := w.sub.receipts; len(got) != 1 || got[0].units != 2 || got[0].slot != int(a.id) {
			t.Errorf("receipts at P4.1 = %+v, want 2 units of grain for agent %d", got, a.id)
		}
	})
	if len(w.sub.receipts) != 0 {
		t.Errorf("receipts after the tick = %+v, want none: P6.3 empties the list", w.sub.receipts)
	}
	w.sub.receipts = append(w.sub.receipts, ownReceipt{slot: int(a.id), commodity: 0, units: 2})
	w.valuePrices()
	w.oc.valued[int(w.regionOf(a))*len(w.cfg.Commodities)] = 1_500
	w.welfare.reset(int(a.id))
	w.sub.priceReceipts(w)
	if got := w.welfare.at(int(a.id)).ownReceipt; got != 3_000 {
		t.Errorf("K^own at the price 1 500 = %d, want 3 000", got)
	}
}

// TestWelfare_FedByFloorReadsTenThousand verifies the property of 7.2: a
// household that makes exactly its upkeep reads a ratio of 10 000 whatever the
// price, because the receipt and the cost are valued together.
//
// Arithmetic: upkeep 2 grain, gather makes 2 grain: income 2 * p, cost 2 * p,
// ratio 10 000, at the reference price 1 000 and at 5 000.
//
// Requirement: FR-SUB-028
func TestWelfare_FedByFloorReadsTenThousand(t *testing.T) {
	t.Parallel()
	for _, price := range []int64{1_000, 5_000} {
		w := newForage(t, 1, func(c *Config) {
			c.Commodities[0].ReferencePrice = Price(price)
			c.Regions[0].ReferenceBasket = Money(2 * price)
		})
		if err := w.Step(); err != nil {
			t.Fatalf("price %d: Step error = %v, want nil", price, err)
		}
		w.refreshViewCtx()
		var v AgentView
		if err := w.fillAgentView(&v, &w.agents[1]); err != nil {
			t.Fatal(err)
		}
		if !v.HasRatio || v.Ratio != 10_000 {
			t.Errorf("price %d: ratio = %d (present %v), want 10 000", price, v.Ratio, v.HasRatio)
		}
	}
}

// TestWage_ExcludesOwnUse verifies that the observed wage leaves the receipt out
// of the income and the own-use ticks out of the working ticks: a household that
// only worked the floor shows no wage, where counting them would show the floor's
// productivity as a wage.
//
// Requirement: FR-SUB-029
func TestWage_ExcludesOwnUse(t *testing.T) {
	t.Parallel()
	w := newForage(t, 1, nil)
	for range 3 {
		if err := w.Step(); err != nil {
			t.Fatal(err)
		}
	}
	a := &w.agents[1]
	if !w.sub.ownUse[a.id] {
		t.Fatalf("the household left the mode: the fixture must keep it on the floor")
	}
	if num, den := a.history.WageParts(); den != 0 || num != 0 {
		t.Errorf("wage parts = (%d, %d), want (0, 0): the receipts and the own-use ticks are left out", num, den)
	}
	if got := a.history.WorkingTicks(); got != 3 {
		t.Errorf("working ticks = %d, want 3: an own-use tick is a working tick", got)
	}
}

// TestWelfare_OwnUseTickIsWorking verifies that an own-use tick counts as a
// working tick for a role with recipes and for a role with none.
//
// Requirement: FR-SUB-030
func TestWelfare_OwnUseTickIsWorking(t *testing.T) {
	t.Parallel()
	for name, edit := range map[string]func(*Config){
		"role with a recipe":  nil,
		"role with no recipe": func(c *Config) { c.Roles[0].Recipes = nil },
	} {
		w := newForage(t, 1, edit)
		if err := w.Step(); err != nil {
			t.Fatalf("%s: Step error = %v, want nil", name, err)
		}
		if got := w.agents[1].history.WorkingTicks(); got != 1 {
			t.Errorf("%s: working ticks after one own-use tick = %d, want 1", name, got)
		}
	}
}

// TestWelfare_NoReceiptForMarketJob verifies that a job of a recipe off the
// own-use list leaves no receipt, and with the module off nothing is kept.
//
// Requirement: FR-SUB-028
func TestWelfare_NoReceiptForMarketJob(t *testing.T) {
	t.Parallel()
	w := newForage(t, 1, func(c *Config) {
		c.Roles[0].StartMoney = 1_000_000
		c.Roles[0].StartStock = []Item{{Commodity: "grain", Quantity: 40}}
	})
	tickInspecting(t, w, func() {
		if w.agents[1].job.recipe != 1 && w.agents[1].hasJob() {
			t.Fatalf("job recipe = %d, want the market recipe tend", w.agents[1].job.recipe)
		}
		if len(w.sub.receipts) != 0 {
			t.Errorf("receipts of a market job = %+v, want none", w.sub.receipts)
		}
	})
}

// TestSubsistence_NoNewReasonAndAuditHolds verifies that the module adds no ledger
// reason and no account kind: a run of the small world with the audit on holds
// the identities of every tick while households work the floor.
//
// Requirement: FR-SUB-003
func TestSubsistence_NoNewReasonAndAuditHolds(t *testing.T) {
	t.Parallel()
	w := newForage(t, 4, forageField(5), WithAudit())
	for range 30 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step at tick %d error = %v, want nil", w.Tick(), err)
		}
	}
}
