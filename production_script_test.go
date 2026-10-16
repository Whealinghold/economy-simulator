package economysim

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// scriptStats counts what a production script did, so a test can see the
// script was not vacuous.
type scriptStats struct {
	Starts, Refusals, Completions, Abandons, Reweighs, Shocks, Disposals int
}

// scriptAgents is the number of agents of a script.
const scriptAgents = 3

// productionScript runs ops random operations of the production steps against
// a real ledger over any canonical configuration, with no code specific to a
// recipe or commodity: seeds of stock (sometimes up to the quantity bound),
// starts of random recipes in random regions and seasons, advances with the
// completion at the end, abandons, weight falls and yield shocks. After every
// operation it checks the ledger's audit, the conservation of goods through
// the four reasons, and for every commodity that the units fixed at starts
// equal the units created plus those pending plus those dropped by abandons
// and weight falls. The error names the seed and the operation.
func productionScript(cfg *Config, seed uint64, ops int) (scriptStats, error) {
	var st scriptStats
	rt := newRecipeTable(cfg)
	yt := newYieldTable(cfg)
	var shocks yieldShocks
	s := newProductionScratch(cfg, &rt)
	l, err := newLedger(*cfg)
	if err != nil {
		return st, fmt.Errorf("seed %d: %w", seed, err)
	}
	if err := l.EndConstruction(); err != nil {
		return st, fmt.Errorf("seed %d: %w", seed, err)
	}
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	var all []RecipeID
	for i := range cfg.Recipes {
		all = append(all, RecipeID(i))
	}
	nCom, nReg, nSeason := len(cfg.Commodities), len(cfg.Regions), len(cfg.Calendar.Seasons)
	var accts [scriptAgents]int
	var jobs [scriptAgents]job
	var resid [scriptAgents][]int64
	for a := range accts {
		if accts[a], err = l.Open(ledger.KindAgent, a, 0); err != nil {
			return st, fmt.Errorf("seed %d: %w", seed, err)
		}
		resid[a] = make([]int64, residueLen(all, &rt))
	}
	fixed, created, dropped := make([]int64, nCom), make([]int64, nCom), make([]int64, nCom)
	pendingNow, jobCount := make([]int64, nCom), make([]int64, len(cfg.Recipes))
	sumPending := func(j *job, into []int64) {
		for i, o := range rt.row(j.recipe).outputs {
			into[o.commodity] += j.pending[i]
		}
	}

	check := func(op int, what string) error {
		fail := func(format string, args ...any) error {
			return fmt.Errorf("seed %d op %d (%s): %s", seed, op, what, fmt.Sprintf(format, args...))
		}
		if err := l.Audit(); err != nil {
			return fail("audit: %v", err)
		}
		// Goods held = Initial + recipe_output - recipe_input - disposal - wear.
		netOf := make([][4]int64, nCom)
		for _, fl := range l.AppendFlows(nil, false) {
			if fl.Commodity < 0 {
				continue
			}
			switch fl.Reason {
			case ledger.Initial, ledger.RecipeOutput:
				netOf[fl.Commodity][0] += fl.Total
			case ledger.RecipeInput, ledger.Disposal, ledger.Wear:
				netOf[fl.Commodity][1] += fl.Total
			}
		}
		clear(pendingNow)
		clear(jobCount)
		for a := range jobs {
			if !s.addPending(&jobs[a], pendingNow, jobCount) {
				return fail("pending aggregate overflows")
			}
		}
		for c := range nCom {
			var held int64
			for _, acct := range accts {
				h, _, err := l.Holding(acct, c)
				if err != nil {
					return fail("Holding: %v", err)
				}
				held += h
			}
			if want := netOf[c][0] - netOf[c][1]; held != want {
				return fail("commodity %d: held %d, want in %d - out %d = %d", c, held, netOf[c][0], netOf[c][1], want)
			}
			if fixed[c] != created[c]+pendingNow[c]+dropped[c] {
				return fail("commodity %d: fixed %d != created %d + pending %d + dropped %d", c, fixed[c], created[c], pendingNow[c], dropped[c])
			}
		}
		return nil
	}

	for op := range ops {
		a := rng.IntN(scriptAgents)
		j, acct := &jobs[a], accts[a]
		what := "noop"
		switch {
		case j.executions == 0 && rng.IntN(100) < 30:
			what = "seed"
			c := rng.IntN(nCom)
			n := int64(rng.IntN(40) + 1)
			held, _, err := l.Holding(acct, c)
			if err != nil {
				return st, fmt.Errorf("seed %d op %d: %w", seed, op, err)
			}
			if rng.IntN(8) == 0 {
				n = MaxQuantity - held - int64(rng.IntN(4))
			}
			if n > 0 && held+n <= MaxQuantity {
				from, ok := l.Boundary(ledger.Initial, c)
				if !ok {
					return st, fmt.Errorf("seed %d op %d: no Initial boundary", seed, op)
				}
				if err := l.Move(from, acct, c, n, ledger.Initial); err != nil {
					return st, fmt.Errorf("seed %d op %d: seed stock: %w", seed, op, err)
				}
			}
		case j.executions == 0:
			what = "start"
			rid := RecipeID(rng.IntN(len(cfg.Recipes)))
			region, season := RegionID(rng.IntN(nReg)), -1
			if nSeason > 0 {
				season = rng.IntN(nSeason)
			}
			e, has, ok := yt.effective(&shocks, region, rid, season, cfg.Recipes[rid].Sited)
			if !ok {
				return st, fmt.Errorf("seed %d op %d: effective factor overflows", seed, op)
			}
			q := &startReq{recipe: rid, account: acct, principal: AgentID(a), weight: int64(rng.IntN(6) + 1), maximum: int64(rng.IntN(4)), effective: e, hasProfile: has}
			off, _ := residueOffset(all, &rt, rid)
			res, err := s.start(l, j, resid[a][off:off+len(rt.row(rid).outputs)], q)
			if err != nil {
				return st, fmt.Errorf("seed %d op %d: start of recipe %d: %w", seed, op, rid, err)
			}
			if res.k == 0 {
				st.Refusals++
				break
			}
			st.Starts++
			sumPending(j, fixed)
		default:
			switch r := rng.IntN(100); {
			case r < 70:
				what = "advance"
				if done, _ := j.advance(); done {
					what = "complete"
					c := s.complete(l, j, acct, MaxQuantity)
					for i, o := range rt.row(c.recipe).outputs {
						created[o.commodity] += c.created[i]
						if c.disposed[i] > 0 {
							st.Disposals++
						}
					}
					st.Completions++
				}
			case r < 80:
				what = "abandon"
				sumPending(j, dropped)
				j.abandon()
				st.Abandons++
			case r < 95:
				what = "reweigh"
				before := make([]int64, nCom)
				sumPending(j, before)
				j.reweigh(int64(rng.IntN(int(j.executions) + 2)))
				after := make([]int64, nCom)
				if j.executions > 0 {
					sumPending(j, after)
				}
				for c := range nCom {
					dropped[c] += before[c] - after[c]
				}
				st.Reweighs++
			default:
				what = "shock"
				region, rid := RegionID(rng.IntN(nReg)), RecipeID(rng.IntN(len(cfg.Recipes)))
				bps := [...]int64{0, 3_000, 10_000, 15_000}
				if err := shocks.set(region, rid, bps[rng.IntN(len(bps))]); err != nil {
					return st, fmt.Errorf("seed %d op %d: %w", seed, op, err)
				}
				st.Shocks++
			}
		}
		if err := check(op, what); err != nil {
			return st, err
		}
	}
	return st, nil
}

// TestProductionInvariant runs the script over the small configuration of the
// steps tests for a range of seeds: a few thousand operations each, the audit
// and the conservation checked after every one, the failing seed in the
// message. Goods change only by recipe_input, recipe_output, disposal and wear
// (spec-ledger.md 8.2.2), and for every commodity the units fixed at starts
// equal the units created plus the units pending plus the units dropped by
// abandons and weight falls.
//
// Requirement: FR-PRO-044, FR-PRO-045, FR-PRO-029, FR-PRO-030
//
// Guard: testing.md 6.3, an invariant over a long random run finds the order
// dependent loss that a table of cases does not.
func TestProductionInvariant(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	cfg := stepsConfig()
	var total scriptStats
	for seed := uint64(1); seed <= 12; seed++ {
		st, err := productionScript(cfg, seed, 4_000)
		if err != nil {
			t.Fatal(err)
		}
		total.Starts += st.Starts
		total.Completions += st.Completions
		total.Abandons += st.Abandons
		total.Reweighs += st.Reweighs
		total.Disposals += st.Disposals
		total.Refusals += st.Refusals
	}
	t.Logf("script totals %+v", total)
	// The run must reach every kind of step, or it proves nothing.
	if total.Starts < 100 || total.Completions < 50 || total.Abandons < 10 || total.Reweighs < 10 || total.Disposals < 1 || total.Refusals < 10 {
		t.Errorf("script reached %+v, want starts, completions, abandons, reweighs, disposals and refusals", total)
	}
}
