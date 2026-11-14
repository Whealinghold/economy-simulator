package tenure_test

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/tenure"
)

// payeeLimit is the most money a lessor may hold in these runs, low enough that
// some rent is forfeited.
const payeeLimit = 8

// endowmentRun is the result of one endowment passed through every rule the way
// the step will use them: apportionment, tiers, lessor supply, attribution and
// the rent transfers.
type endowmentRun struct {
	stock, entitled int64
	res             tenure.Result
	reqs            []tenure.Request // by ID after the run
	supply          int64
	triples         []tenure.Triple
	debit           map[int64]int64 // rent paid by each tenant
	credit          map[int64]int64 // rent received by each lessor
	forfeited       int64
}

// runEndowment passes one random endowment through the rules. Holders have IDs 1
// to n; tenants have IDs 1 to m, so the low IDs are both (a self transfer).
// order shuffles the listing of every input with the given generator.
func runEndowment(t *testing.T, rng *rand.Rand, order *rand.Rand) endowmentRun {
	t.Helper()
	stock := int64(rng.IntN(300))
	n := 1 + rng.IntN(5)
	parts := make([]tenure.Part, 0, n+1)
	left := int64(tenure.BasisPoints)
	for i := range n {
		bp := int64(rng.IntN(int(left)/2 + 1))
		left -= bp
		parts = append(parts, tenure.Part{Ref: i, ID: int64(i + 1), Share: bp, Key: uint64(rng.IntN(3))})
	}
	parts = append(parts, tenure.Part{Ref: n, Share: left, Commons: true})
	out := make([]int64, n+1)
	order.Shuffle(len(parts), func(i, j int) { parts[i], parts[j] = parts[j], parts[i] })
	if !tenure.Apportion(stock, parts, out) {
		t.Fatalf("Apportion(%d, %+v) not ok", stock, parts)
	}
	holders := make([]tenure.Holder, n)
	for i := range n {
		holders[i] = tenure.Holder{ID: int64(i + 1), Entitled: out[i]}
	}
	rent := int64(rng.IntN(6))
	p := tenure.Params{Stock: stock, Commons: out[n], Rent: rent}
	m := 1 + rng.IntN(8)
	reqs := make([]tenure.Request, m)
	for i := range reqs {
		id := int64(i + 1)
		reqs[i] = tenure.Request{Ref: i, ID: id, Principal: id, Executions: int64(rng.IntN(40)), Draw: int64(1 + rng.IntN(3)),
			Key: uint64(rng.IntN(3)), Balance: int64(rng.IntN(200)), Exempt: rng.IntN(6) == 0}
	}
	order.Shuffle(len(holders), func(i, j int) { holders[i], holders[j] = holders[j], holders[i] })
	order.Shuffle(len(reqs), func(i, j int) { reqs[i], reqs[j] = reqs[j], reqs[i] })
	var s tenure.Scratch
	res, ok := s.Tiers(p, holders, reqs)
	if !ok {
		t.Fatalf("Tiers(%+v, %+v, %+v) not ok", p, holders, reqs)
	}
	run := endowmentRun{stock: stock, res: res, debit: map[int64]int64{}, credit: map[int64]int64{}}
	for _, v := range out {
		run.entitled += v
	}
	slices.SortFunc(reqs, func(a, b tenure.Request) int { return int(a.ID - b.ID) })
	run.reqs = reqs
	run.attribute(t, &s, rent, holders)
	return run
}

// attribute splits the let units among the lessors and moves the rent.
func (run *endowmentRun) attribute(t *testing.T, s *tenure.Scratch, rent int64, holders []tenure.Holder) {
	t.Helper()
	slices.SortFunc(holders, func(a, b tenure.Holder) int { return int(a.ID - b.ID) })
	lessorParts := make([]tenure.Part, len(holders))
	for i, h := range holders {
		lessorParts[i] = tenure.Part{Ref: i, ID: h.ID, Share: h.Unused}
	}
	supply := make([]int64, len(holders))
	if !tenure.Supply(run.res.LetUnits, lessorParts, supply) {
		t.Fatalf("Supply(%d, %+v) not ok", run.res.LetUnits, lessorParts)
	}
	var tenants, lessors []tenure.Taker
	for _, r := range run.reqs {
		if r.Let > 0 {
			tenants = append(tenants, tenure.Taker{ID: r.ID, Units: r.Let * r.Draw})
		}
	}
	for i, h := range holders {
		run.supply += supply[i]
		if supply[i] > 0 {
			lessors = append(lessors, tenure.Taker{ID: h.ID, Units: supply[i]})
		}
	}
	var ok bool
	if run.triples, ok = s.Attribute(nil, tenants, lessors); !ok {
		t.Fatalf("Attribute(%+v, %+v) not ok", tenants, lessors)
	}
	for _, tr := range run.triples {
		payer, payee := tenants[tr.Tenant].ID, lessors[tr.Lessor].ID
		if tr.Self {
			continue
		}
		due := tr.Units * rent
		if run.reqs[payer-1].Exempt {
			due = 0
		}
		paid, lost, _ := tenure.Credit(run.credit[payee], due, payeeLimit)
		run.debit[payer] += due
		run.credit[payee] += paid
		run.forfeited += lost
	}
}

// TestRules_UsedTogetherConserveUnitsAndMoney passes random endowments through
// apportionment, tiers, lessor supply, attribution and the rent transfer as the
// step will, and checks what only the rules together guarantee: the
// entitlements sum to the stock, no more than the stock is drawn, the let units
// taken equal the lessors' supply and the attributed units, a tenant pays
// no more than its balance, and the rent debited equals the rent credited plus
// the rent forfeited.
//
// Requirement: FR-LND-007, FR-LND-008, FR-LND-009, FR-LND-010, FR-LND-011, FR-LND-013, FR-LND-016, FR-LND-017
//
// Guard: each rule is exact alone; a unit or a coin could still be lost where
// one rule's output feeds the next, and nothing else tests that join.
func TestRules_UsedTogetherConserveUnitsAndMoney(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	var let, transfers, forfeits int
	for seed := uint64(1); seed <= 40; seed++ {
		rng := rand.New(rand.NewPCG(seed, 5))
		for i := range 25 {
			run := runEndowment(t, rng, rand.New(rand.NewPCG(seed, uint64(i))))
			checkRun(t, seed, i, run)
			let += min(1, int(run.res.LetUnits))
			transfers += min(1, len(run.debit))
			forfeits += min(1, int(run.forfeited))
		}
	}
	// the scenarios must reach the rules this test is about, or it proves nothing
	if let < 50 || transfers < 50 || forfeits < 5 {
		t.Errorf("only %d runs let units, %d paid rent, %d forfeited: the random endowments miss the rent rules", let, transfers, forfeits)
	}
}

// checkRun asserts the conservation properties of one endowment.
func checkRun(t *testing.T, seed uint64, i int, run endowmentRun) {
	t.Helper()
	at := func(msg string, args ...any) { t.Fatalf("seed %d case %d: "+msg, append([]any{seed, i}, args...)...) }
	if run.entitled != run.stock {
		at("entitlements sum to %d, want the stock %d", run.entitled, run.stock)
	}
	if got := run.res.OwnUnits + run.res.CommonsUnits + run.res.LetUnits; got > run.stock {
		at("drew %d units of a stock of %d", got, run.stock)
	}
	var attributed, debit, credit int64
	for _, tr := range run.triples {
		attributed += tr.Units
	}
	for _, v := range run.debit {
		debit += v
	}
	for _, v := range run.credit {
		credit += v
	}
	if run.supply != run.res.LetUnits || attributed != run.res.LetUnits {
		at("let units %d, lessor supply %d, attributed %d, want all equal", run.res.LetUnits, run.supply, attributed)
	}
	if debit != credit+run.forfeited {
		at("rent debited %d, credited %d, forfeited %d: money created or destroyed", debit, credit, run.forfeited)
	}
	for _, r := range run.reqs {
		if run.debit[r.ID] > r.Balance {
			at("tenant %d paid %d with a balance of %d", r.ID, run.debit[r.ID], r.Balance)
		}
	}
}

// TestRules_ListingOrderDoesNotChangeTheOutcome verifies that the whole chain of
// rules gives the same grants, supplies and transfers however the holders,
// requests and parts are listed: the same endowment is passed through twice with
// differently shuffled inputs and the per-ID results are compared.
//
// Requirement: FR-LND-011, NFR-LND-003
func TestRules_ListingOrderDoesNotChangeTheOutcome(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	for seed := uint64(1); seed <= 40; seed++ {
		first := runEndowment(t, rand.New(rand.NewPCG(seed, 5)), rand.New(rand.NewPCG(seed, 1)))
		second := runEndowment(t, rand.New(rand.NewPCG(seed, 5)), rand.New(rand.NewPCG(seed, 2)))
		if first.res != second.res || !slices.Equal(first.triples, second.triples) || first.forfeited != second.forfeited {
			t.Fatalf("seed %d: two listings gave %+v / %v and %+v / %v", seed, first.res, first.triples, second.res, second.triples)
		}
		for j := range first.reqs {
			if a, b := first.reqs[j], second.reqs[j]; a.Own != b.Own || a.Commons != b.Commons || a.Let != b.Let || a.PricedOut != b.PricedOut {
				t.Fatalf("seed %d: request %d got %+v and %+v", seed, a.ID, a, b)
			}
		}
	}
}
