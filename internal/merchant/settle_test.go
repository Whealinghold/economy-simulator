package merchant_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/merchant"
)

// TestSettle_Table verifies the four cases of spec-merchants.md 9.7 on the contract
// of example A (price 257, 150 units, an escrow of 38 550): everything arrived,
// 22 spoiled, none reached, and a cargo diverted to the estate of a removed payer
// (which counts as reached); escrow_out + escrow_refund is the escrow in each; and
// units that do not add up, a negative figure or an overflow are refused.
//
// Requirement: FR-MER-040, FR-MER-041, FR-MER-042, FR-CAR-001, FR-CAR-002
func TestSettle_Table(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                               string
		price, qty, reached, spoiled, lost int64
		out, refund                        int64
		ok                                 bool
	}{
		{"everything arrived", 257, 150, 150, 0, 0, 38_550, 0, true},
		// 257 * 128 = 32 896 and 257 * 22 = 5 654.
		{"22 spoiled", 257, 150, 128, 22, 0, 32_896, 5_654, true},
		{"none reached", 257, 150, 0, 150, 0, 0, 38_550, true},
		{"a price of 0", 0, 5, 5, 0, 0, 0, 0, true},
		{"nothing in the lot", 1, 0, 0, 0, 0, 0, 0, true},
		{"spoiled and lost", 257, 150, 100, 20, 30, 25_700, 12_850, true},
		{"a diverted cargo counts as reached", 257, 150, 150, 0, 0, 38_550, 0, true},
		{"the units do not add up", 257, 150, 100, 20, 20, 0, 0, false},
		{"more than the lot", 257, 150, 150, 1, 0, 0, 0, false},
		{"negative reached", 257, 150, -1, 151, 0, 0, 0, false},
		{"negative lost", 257, 150, 151, 0, -1, 0, 0, false},
		{"negative price", -1, 150, 150, 0, 0, 0, 0, false},
		{"overflow of the sum", 1, 1, 1<<63 - 1, 1, 0, 0, 0, false},
		{"overflow of the sum with the loss", 1, 1, 1<<63 - 1, 0, 1, 0, 0, false},
		{"overflow of the first sum with no units", 1, 0, 1<<63 - 1, 1, 0, 0, 0, false},
		{"overflow of the escrow", 1 << 62, 4, 2, 2, 0, 0, 0, false},
		{"overflow of the payment", 1 << 62, 4, 4, 0, 0, 0, 0, false},
		{"overflow of the refund", 1 << 62, 4, 0, 4, 0, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out, refund, ok := merchant.Settle(tt.price, tt.qty, tt.reached, tt.spoiled, tt.lost)
			if ok != tt.ok || out != tt.out || refund != tt.refund || (ok && out+refund != tt.price*tt.qty) {
				t.Errorf("Settle(%d, %d, %d, %d, %d) = %d, %d, %v, want %d, %d, %v (their sum is the escrow)", tt.price, tt.qty, tt.reached, tt.spoiled, tt.lost, out, refund, ok, tt.out, tt.refund, tt.ok)
			}
		})
	}
	if e, ok := merchant.Escrow(257, 150); !ok || e != 38_550 {
		t.Errorf("Escrow(257, 150) = %d, %v, want 38550, true: the refund of a removed deliverer", e, ok)
	}
	if e, ok := merchant.Escrow(1<<62, 4); ok {
		t.Errorf("Escrow(2^62, 4) = %d, true, want ok false", e)
	}
}

// TestWorked_SpecExamples verifies the chain of example A from the rumour to the
// contract with the functions together (spec-merchants.md 19.1): the cautious
// revenue 305, the net 289, the freight 4 200, the margin 13 350, the test, the
// haul ceiling 1 420, the break-even 245 and the offer limit 244 of agent 32, the
// contract price 257 and the escrow 38 550.
//
// Requirement: FR-MER-013, FR-MER-015, FR-MER-016, FR-MER-017, FR-MER-035
func TestWorked_SpecExamples(t *testing.T) {
	t.Parallel()
	rhat := merchant.Cautious(310, 305)
	rnet, _ := merchant.Net(rhat, 500, 0)
	cost, _ := merchant.Carriage{N: 1, K: 2, Hired: true, S: 6, HaulPrice: 700}.Cost()
	phi := merchant.FeeShare(5_000, 2_500, 32)
	r, ok := merchant.Margin(cost, phi, 150, rnet, 200)
	ceiling, _ := merchant.HaulCeiling(r.G, 0, 1, 6, phi)
	if !ok || rhat != 305 || rnet != 289 || cost != 4_200 || phi != 6_385 || r.G != 13_350 || !merchant.Pass(cost, phi, r.G) ||
		ceiling != 1_420 || r.BreakEven != 245 || r.AskLimit != 244 {
		t.Errorf("chain of example A = rhat %d, rnet %d, C %d, phi %d, %+v, ceiling %d, want 305, 289, 4200, 6385, G 13350, ceiling 1420, break-even 245, ask 244", rhat, rnet, cost, phi, r, ceiling)
	}
	price := merchant.Midpoint(r.AskLimit, 270)
	if e, _ := merchant.Escrow(price, 150); price != 257 || e != 38_550 {
		t.Errorf("contract price %d and escrow %d, want 257 and 38550", price, e)
	}
	// Example B: the trader of agent 18 on the 3-tick skiff trip.
	skiff := merchant.Carriage{N: 1, K: 3, WearPerTick: 5, VehiclePrice: 4_000, Durability: 400, Feed: []merchant.Feed{{Quantity: 1, Price: 30}}, UpkeepCost: 56, TimeCostBP: 10_000}
	costB, _ := skiff.Cost()
	phiB := merchant.FeeShare(5_000, 2_500, 18)
	rB, _ := merchant.Margin(costB, phiB, 100, 78, 56)
	if costB != 408 || phiB != 3_123 || !merchant.Pass(costB, phiB, rB.G) || rB.G != 2_200 || rB.BreakEven != 64 {
		t.Errorf("chain of example B = C %d, phi %d, %+v, want 408, 3123, G 2200, break-even 64", costB, phiB, rB)
	}
}

// TestImports_Guard verifies that the package imports only the standard library,
// internal/checked and internal/ordering, uses no floating-point type, no random
// source and no clock, and has no identifier that names a region, a commodity, a
// role, a mode or an agent, so it works over primitive numbers.
//
// Requirement: NFR-MER-003
//
// Guard: architecture.md 2.9, 3.5 and 9.1. An import of the root, a float in the
// arithmetic of a decision or an identifier that branches on a domain name would
// break the layering or the determinism, and no behavioural test would point at
// it.
func TestImports_Guard(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("glob: %v, %d files", err, len(files))
	}
	allowed := map[string]bool{
		"github.com/Whealinghold/economy-sim/internal/checked":  true,
		"github.com/Whealinghold/economy-sim/internal/ordering": true,
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			std := !strings.Contains(strings.SplitN(path, "/", 2)[0], ".")
			if !allowed[path] && (!std || path == "math/rand" || path == "math/rand/v2" || path == "time") {
				t.Errorf("%s imports %q, want the standard library (no rand, no time), internal/checked and internal/ordering only", file, path)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			low := strings.ToLower(id.Name)
			if low == "float32" || low == "float64" {
				t.Errorf("%s: identifier %q, want no floating point (architecture.md 3.5)", file, id.Name)
			}
			for _, bad := range []string{"region", "commodity", "role", "agent"} {
				if strings.Contains(low, bad) {
					t.Errorf("%s: identifier %q names a %s, want primitive names only", file, id.Name, bad)
				}
			}
			return true
		})
	}
}

// TestMatchSettle_NoAllocs verifies that the round and the settlement allocate
// nothing once the caller has sized the result: Match into a slice with room, the
// sorts, Midpoint, Settle, Take, Margin and Cost.
//
// Requirement: NFR-MER-001
func TestMatchSettle_NoAllocs(t *testing.T) {
	offers, takes := lane19()
	dst := make([]merchant.Pair, 0, 8)
	skiff := merchant.Carriage{N: 1, K: 3, WearPerTick: 5, VehiclePrice: 4_000, Durability: 400, Feed: []merchant.Feed{{Quantity: 1, Price: 30}}, UpkeepCost: 56, TimeCostBP: 10_000}
	allocs := testing.AllocsPerRun(100, func() {
		merchant.SortOffers(offers)
		merchant.SortBids(takes)
		dst = merchant.Match(dst[:0], offers, takes, nil)
		for i := range takes {
			takes[i].Qty = 200
		}
		_, _, _ = merchant.Settle(257, 150, 128, 22, 0)
		_, _, _ = merchant.Take(merchant.TakeInput{Price: 300, MarginBP: 1_000, Units: 150, Money: 1_000_000, MaxQty: 1_000})
		_, _ = merchant.Margin(408, 3_123, 100, 78, 56)
		_, _ = skiff.Cost()
		_ = merchant.FeeShare(5_000, 2_500, 32)
	})
	if allocs != 0 {
		t.Errorf("round, settlement, take, margin and cost allocated %v times per run, want 0", allocs)
	}
}
