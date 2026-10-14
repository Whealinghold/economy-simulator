package ledger

import "testing"

// costRig is a funded ledger out of construction, with a money bound large
// enough that a long loop never reaches it, and a long-running claim.
type costRig struct {
	*rig
	a, b  int
	claim int64
}

func newCostRig(tb testing.TB) *costRig {
	tb.Helper()
	g := newRig(tb, 1<<40)
	g.fund(tb)
	l := g.l
	for _, h := range g.holders[:2] {
		if err := l.Transfer(g.mint, h, 1<<20, Initial); err != nil {
			tb.Fatalf("setup: %v", err)
		}
	}
	cl, err := l.Lend(g.holders[0], g.holders[1], 1<<20, 1<<20, 500, 1<<30)
	if err != nil {
		tb.Fatalf("setup: Lend: %v", err)
	}
	if err := l.EndConstruction(); err != nil {
		tb.Fatalf("setup: %v", err)
	}
	// Both agents hold a lot of every commodity, so a move between them
	// merges into an existing lot and a steady loop adds no lot.
	return &costRig{rig: g, a: g.holders[0], b: g.holders[1], claim: cl}
}

// TestCost_OperationsAllocateNothing verifies that the operations of a steady
// state allocate nothing: a transfer, a move between existing lots, a reserve
// and release of money and of goods, a transform, a Settle that does not end
// its claim, an ageing pass with nothing due, Expiring and a read of the flow
// totals into a slice with room.
//
// Requirement: NFR-LED-001, NFR-LED-004, FR-LED-006
func TestCost_OperationsAllocateNothing(t *testing.T) {
	c := newCostRig(t)
	l := c.l
	flows := make([]Flow, 0, 1024)
	lines := []Line{{Commodity: 2, Quantity: 1}}
	tests := []struct {
		name string
		f    func()
	}{
		{"transfer", func() {
			c.must(l.Transfer(c.a, c.b, 1, Trade))
			c.must(l.Transfer(c.b, c.a, 1, Trade))
		}},
		{"move between existing lots", func() {
			c.must(l.Move(c.a, c.b, 2, 1, Trade))
			c.must(l.Move(c.b, c.a, 2, 1, Trade))
		}},
		{"reserve and release money", func() {
			c.must(l.ReserveMoney(c.a, 5))
			c.must(l.ReleaseMoney(c.a, 5))
		}},
		{"reserve and release goods", func() {
			c.must(l.ReserveGoods(c.a, 2, 1))
			c.must(l.ReleaseGoods(c.a, 2, 1))
		}},
		{"transform in a steady state", func() {
			c.must(l.Transform(c.a, lines, lines))
		}},
		{"settle that keeps the claim", func() {
			c.must(l.Settle(c.claim, 1, 1))
		}},
		{"ageing with nothing due", func() {
			c.must(l.Age())
		}},
		{"expiring", func() {
			if _, err := l.Expiring(c.a, 0, l.Tick()); err != nil {
				panic(err)
			}
		}},
		{"audit-free reads", func() {
			flows = l.AppendFlows(flows[:0], false)
			_ = l.HasReservation()
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.f() // warm up: a first call may grow a slice once.
			if got := testing.AllocsPerRun(100, tt.f); got != 0 {
				t.Errorf("%s allocates %v times per run, want 0", tt.name, got)
			}
		})
	}
}

func (c *costRig) must(err error) {
	if err != nil {
		panic(err)
	}
}

// BenchmarkTransfer reports the cost and allocations of a transfer.
//
// Requirement: NFR-LED-003
func BenchmarkTransfer(b *testing.B) {
	c := newCostRig(b)
	b.ReportAllocs()
	for b.Loop() {
		c.must(c.l.Transfer(c.a, c.b, 1, Trade))
		c.a, c.b = c.b, c.a
	}
}

// BenchmarkMove reports the cost and allocations of a move between lots.
//
// Requirement: NFR-LED-003
func BenchmarkMove(b *testing.B) {
	c := newCostRig(b)
	b.ReportAllocs()
	for b.Loop() {
		c.must(c.l.Move(c.a, c.b, 0, 1, Trade))
		c.a, c.b = c.b, c.a
	}
}

// BenchmarkAge reports the cost of an ageing pass with nothing due, over all
// the accounts.
//
// Requirement: NFR-LED-003
func BenchmarkAge(b *testing.B) {
	c := newCostRig(b)
	b.ReportAllocs()
	for b.Loop() {
		c.must(c.l.Age())
	}
}

// BenchmarkTransform reports the cost and allocations of a transform.
//
// Requirement: NFR-LED-003
func BenchmarkTransform(b *testing.B) {
	c := newCostRig(b)
	in, out := []Line{{Commodity: 2, Quantity: 2}}, []Line{{Commodity: 0, Quantity: 1}, {Commodity: 2, Quantity: 2}}
	b.ReportAllocs()
	for b.Loop() {
		c.must(c.l.Transform(c.a, in, out))
		// Take the output back so the account does not grow.
		c.must(c.l.Move(c.a, c.bnd[Disposal][0], 0, 1, Disposal))
	}
}

// BenchmarkWearItems reports the cost and allocations of wearing items down.
//
// Requirement: NFR-LED-004
func BenchmarkWearItems(b *testing.B) {
	c := newCostRig(b)
	b.ReportAllocs()
	for b.Loop() {
		// Three items worn out and three created, so the holding is steady.
		c.must(c.l.WearItems(c.a, 1, 3, 5))
		c.must(c.l.Move(c.bnd[Initial][1], c.a, 1, 3, Initial))
	}
}

// BenchmarkExpiring reports the cost and allocations of Expiring.
//
// Requirement: NFR-LED-004
func BenchmarkExpiring(b *testing.B) {
	c := newCostRig(b)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := c.l.Expiring(c.a, 0, c.l.Tick()+2); err != nil {
			b.Fatal(err)
		}
	}
}

// TestCost_WearItemsAllocatesNothing verifies that a wear of one item, which
// moves no lot, allocates nothing. Wear consumes durability, so the run count
// stays under the stock (12 items of 5 units).
//
// Requirement: NFR-LED-001
func TestCost_WearItemsAllocatesNothing(t *testing.T) {
	c := newCostRig(t)
	f := func() { c.must(c.l.WearItems(c.a, 1, 1, 1)) }
	f() // warm up
	if got := testing.AllocsPerRun(20, f); got != 0 {
		t.Errorf("WearItems allocates %v times per run, want 0", got)
	}
}
