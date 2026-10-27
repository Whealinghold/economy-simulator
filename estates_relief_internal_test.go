package economysim

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/welfare"
)

// TestReliefRelease_Table verifies the release of a pool, pool * PoolRelease /
// 10 000 truncated toward zero, at the edges of the pool and of the share.
//
// Arithmetic: 9 999 * 1 / 10 000 = 0 (truncated); 9 999 * 100 / 10 000 = 99;
// 10 000 * 1 / 10 000 = 1; MaxMoney * 10 000 / 10 000 = MaxMoney, and
// MaxMoney * 100 / 10 000 is the 128-bit quotient, which does not overflow.
//
// Requirement: FR-EST-017
func TestReliefRelease_Table(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pool    Money
		release int64
		want    int64
	}{
		{"empty pool", 0, 100, 0},
		{"pool of 1", 1, 100, 0},
		{"pool of 1 whole share", 1, 10_000, 1},
		{"truncated below 1", 9_999, 1, 0},
		{"truncated", 9_999, 100, 99},
		{"exact", 10_000, 1, 1},
		{"zero share", 1_000_000, 0, 0},
		{"whole pool", 1_000_000, 10_000, 1_000_000},
		{"largest pool, whole", MaxMoney, 10_000, MaxMoney},
		{"largest pool, one percent", MaxMoney, 100, MaxMoney / 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := reliefRelease(tt.pool, tt.release)
			if !ok || got != tt.want {
				t.Errorf("reliefRelease(%d, %d) = %d, %v, want %d, true", tt.pool, tt.release, got, ok, tt.want)
			}
		})
	}
}

// TestReliefNeed_Table verifies the need of an agent, upkeep * (line - ratio) /
// 10 000 truncated for a ratio present and below the line, and 0 otherwise.
//
// Arithmetic with upkeep 1 000 and line 10 000: ratio 3 000 gives 1 000 * 7 000 /
// 10 000 = 700; ratio 0 gives 1 000; ratio -2 000 gives 1 000 * 12 000 / 10 000 =
// 1 200; a gap of 1 with upkeep 1 000 gives 1 000 / 10 000 = 0 (truncated).
//
// Requirement: FR-EST-018
func TestReliefNeed_Table(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		upkeep    int64
		ratio     int64
		present   bool
		want      int64
		wantFits  bool
		lineExtra int64 // added to the line of 10 000
	}{
		{"ratio below the line", 1_000, 3_000, true, 700, true, 0},
		{"ratio at the line", 1_000, 10_000, true, 0, true, 0},
		{"ratio above the line", 1_000, 12_000, true, 0, true, 0},
		{"ratio of zero", 1_000, 0, true, 1_000, true, 0},
		{"negative ratio", 1_000, -2_000, true, 1_200, true, 0},
		{"absent ratio", 1_000, 3_000, false, 0, true, 0},
		{"product truncated", 1_000, 9_999, true, 0, true, 0},
		{"the lowest ratio overflows the gap", 1_000, math.MinInt64, true, 0, false, 0},
		{"the product overflows", math.MaxInt64, 0, true, 0, true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := reliefNeed(tt.upkeep, 10_000, tt.ratio, tt.present)
			if tt.name == "the product overflows" {
				// MaxInt64 * 10 000 / 10 000 fits in 128 bits and in int64.
				tt.want, tt.wantFits = math.MaxInt64, true
			}
			if ok != tt.wantFits || (ok && got != tt.want) {
				t.Errorf("reliefNeed(%d, 10000, %d, %v) = %d, %v, want %d, %v",
					tt.upkeep, tt.ratio, tt.present, got, ok, tt.want, tt.wantFits)
			}
		})
	}
	// A quotient above int64 is not ok: 4 * MaxInt64 * 10 000 / 10 000 with a line
	// of 40 000 and ratio 0.
	if got, ok := reliefNeed(math.MaxInt64, 40_000, 0, true); ok {
		t.Errorf("reliefNeed(MaxInt64, 40000, 0, true) = %d, true, want not ok: the quotient is above int64", got)
	}
}

// TestReliefShare_Table verifies the payment of an agent among agents whose
// needs sum to a total, for a release below, at and above the total.
//
// Arithmetic: needs 700 and 200 (sum 900). Release 450: 450 * 700 / 900 = 350 and
// 450 * 200 / 900 = 100 (both exact). Release 100: 100 * 700 / 900 = 77 and 100 *
// 200 / 900 = 22 (truncated; 1 stays in the pool). Release 10 000, above the sum:
// 10 000 * 700 / 900 = 7 777, capped at the need 700, and 200 for the other.
//
// Requirement: FR-EST-018
func TestReliefShare_Table(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		release, need, sum int64
		want               int64
	}{
		{"release below the total, first", 450, 700, 900, 350},
		{"release below the total, second", 450, 200, 900, 100},
		{"truncated, first", 100, 700, 900, 77},
		{"truncated, second", 100, 200, 900, 22},
		{"release above the total pays the need", 10_000, 700, 900, 700},
		{"release above the total, second", 10_000, 200, 900, 200},
		{"release equal to the total", 900, 700, 900, 700},
		{"a sum of zero pays nothing", 450, 0, 0, 0},
		{"a need of zero pays nothing", 450, 0, 900, 0},
		{"a release of zero pays nothing", 0, 700, 900, 0},
		{"the only agent is paid its need", 5_000, 700, 700, 700},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := reliefShare(tt.release, tt.need, tt.sum)
			if !ok || got != tt.want {
				t.Errorf("reliefShare(%d, %d, %d) = %d, %v, want %d, true", tt.release, tt.need, tt.sum, got, ok, tt.want)
			}
		})
	}
}

// sharesOf pays the needs from the release the way the pass does: the sum of the
// needs, then each share.
func sharesOf(release int64, needs []int64) (pay []int64, ok bool) {
	var sum int64
	for _, n := range needs {
		sum += n
	}
	for _, n := range needs {
		p, fits := reliefShare(release, n, sum)
		if !fits {
			return nil, false
		}
		pay = append(pay, p)
	}
	return pay, true
}

// TestReliefShare_Properties verifies, over a grid of needs and releases, that
// no payment exceeds its need, the payments never sum to more than the release,
// a release above the total need pays every need in full, and the remainder is
// what the payments do not take.
//
// Requirement: FR-EST-018
func TestReliefShare_Properties(t *testing.T) {
	t.Parallel()
	needSets := [][]int64{{700, 200}, {1}, {1, 1, 1}, {5, 0, 9}, {1_000_000, 3}, {0, 0}, {123, 456, 789}}
	for _, needs := range needSets {
		var total int64
		for _, n := range needs {
			total += n
		}
		for _, release := range []int64{0, 1, 2, 7, 100, total - 1, total, total + 1, 10 * total, math.MaxInt64 / 2} {
			if release < 0 {
				continue
			}
			t.Run(fmt.Sprintf("needs %v release %d", needs, release), func(t *testing.T) {
				t.Parallel()
				pay, ok := sharesOf(release, needs)
				if !ok {
					t.Fatalf("sharesOf(%d, %v) overflowed, want it to fit", release, needs)
				}
				var paid int64
				for i, p := range pay {
					if p < 0 || p > needs[i] {
						t.Errorf("sharesOf(%d, %v) pays %d to need %d, want within [0, need]", release, needs, p, needs[i])
					}
					paid += p
				}
				if paid > release {
					t.Errorf("sharesOf(%d, %v) pays %d in all, want at most the release", release, needs, paid)
				}
				if release >= total && !slices.Equal(pay, needs) {
					t.Errorf("sharesOf(%d, %v) = %v, want every need in full: the release covers the total %d", release, needs, pay, total)
				}
			})
		}
	}
}

// TestReliefShare_OrderFree verifies that the payment of each agent is the same
// in every permutation of up to six agents: it depends on the agent's need and
// the totals only.
//
// Requirement: FR-EST-019
func TestReliefShare_OrderFree(t *testing.T) {
	t.Parallel()
	needs := []int64{700, 200, 3, 0, 55, 1_001}
	want := map[int64]int64{}
	var sum int64
	for _, n := range needs {
		sum += n
	}
	for _, n := range needs {
		want[n], _ = reliefShare(1_234, n, sum)
	}
	var permute func(prefix, rest []int64)
	count := 0
	permute = func(prefix, rest []int64) {
		if len(rest) == 0 {
			count++
			pay, ok := sharesOf(1_234, prefix)
			if !ok {
				t.Fatalf("sharesOf(%v) overflowed", prefix)
			}
			for i, n := range prefix {
				if pay[i] != want[n] {
					t.Errorf("sharesOf(1234, %v)[%d] = %d for need %d, want %d as in the first order", prefix, i, pay[i], n, want[n])
				}
			}
			return
		}
		for i := range rest {
			next := slices.Concat(rest[:i:i], rest[i+1:])
			permute(append(slices.Clone(prefix), rest[i]), next)
		}
	}
	permute(nil, needs)
	if count != 720 {
		t.Fatalf("visited %d permutations, want 720", count)
	}
}

// FuzzReliefShares fuzzes the release of a pool and its shares among three needs: no panic,
// the payments never exceed the needs or the release, the release is within the
// pool, and a permutation of the needs pays each need the same.
//
// Requirement: FR-EST-018, FR-EST-019
func FuzzReliefShares(f *testing.F) {
	f.Add(int64(700), int64(200), int64(0), int64(1_000_000), int64(100))
	f.Add(int64(1), int64(1), int64(1), int64(9_999), int64(1))
	f.Add(int64(MaxMoney), int64(MaxMoney/3), int64(5), int64(MaxMoney), int64(10_000))
	f.Fuzz(func(t *testing.T, a, b, c, pool, bp int64) {
		if pool < 0 || pool > MaxMoney || bp < 0 || bp > 10_000 || a < 0 || b < 0 || c < 0 || a > MaxMoney || b > MaxMoney || c > MaxMoney {
			t.Skip("outside the bounds of the configuration")
		}
		rel, ok := reliefRelease(Money(pool), bp)
		if !ok || rel > pool {
			t.Fatalf("reliefRelease(%d, %d) = %d, %v, want a release within the pool", pool, bp, rel, ok)
		}
		pay, ok := sharesOf(rel, []int64{a, b, c})
		if !ok {
			t.Fatalf("sharesOf(%d, [%d %d %d]) overflowed", rel, a, b, c)
		}
		needs := []int64{a, b, c}
		var paid int64
		for i := range pay {
			if pay[i] > needs[i] || pay[i] < 0 {
				t.Errorf("pay %d for need %d, want within [0, need]", pay[i], needs[i])
			}
			paid += pay[i]
		}
		if paid > rel {
			t.Errorf("paid %d of a release of %d, want at most the release", paid, rel)
		}
		rev, _ := sharesOf(rel, []int64{c, b, a})
		if rev[0] != pay[2] || rev[1] != pay[1] || rev[2] != pay[0] {
			t.Errorf("payments %v in order and %v reversed, want each need paid the same", pay, rev)
		}
	})
}

// reliefWorld builds the world of a shell with the estates module on, a pool of
// pool in the one region and households households of weight 1, each with a
// ring that holds a full window at the ratio the script gives. The farmer's
// upkeep is 1 grain, valued at the reference price 1 000 before the first trade,
// so U = 1 000 for a household of weight 1.
//
// Arithmetic: a record of cost 1 000 and income 1 000 * r / 10 000, pushed for the
// whole window, gives the ratio r exactly.
func reliefWorld(t *testing.T, pool Money, release int64, ratios ...int64) *World {
	t.Helper()
	return reliefWorldWith(t, nil, pool, release, ratios...)
}

// reliefWorldWith is reliefWorld with options given to NewWorld.
func reliefWorldWith(t *testing.T, opts []Option, pool Money, release int64, ratios ...int64) *World {
	t.Helper()
	c := shellConfig()
	c.Modules.Estates = &Estates{}
	c.Regions[0].PoolRelease = release
	c.Groups[0].Households = int64(len(ratios))
	c.WarmStart = &WarmStart{Pools: []WarmPool{{Region: "home", Money: pool}}}
	w := newMill(t, c, opts...)
	for i, r := range ratios {
		scriptRatio(w, AgentID(1+i), r)
	}
	return w
}

// scriptRatio replaces the ring of an agent with one whose whole window is at the
// ratio r when the upkeep is 1 000.
func scriptRatio(w *World, id AgentID, r int64) {
	a := &w.agents[id]
	a.history = newHistory(w.roles.row(a.role))
	for range a.history.Window() {
		a.history.Push(welfare.Record{Income: 1_000 * r / 10_000, Cost: 1_000})
	}
}

// reliefOf returns what the relief pass added to the welfare scratch of an agent.
func reliefOf(w *World, id AgentID) int64 { return w.welfare.at(int(id)).relief }

// TestRelief_Pass verifies the pass on a built world with a scripted pool and
// scripted rings: of three households at ratios 3 000, 8 000 and 12 000 only the
// first two are paid, in proportion to their needs, by relief transfers that are
// income of the tick, and the pool falls by the total paid.
//
// Arithmetic: the pool 45 000 releases 45 000 * 100 / 10 000 = 450. The needs are
// 1 000 * (10 000 - 3 000) / 10 000 = 700 and 1 000 * (10 000 - 8 000) / 10 000 =
// 200, the third household is above the line; the sum is 900. The shares are
// 450 * 700 / 900 = 350 and 450 * 200 / 900 = 100, so the pool falls to 44 550.
//
// Requirement: FR-EST-017, FR-EST-018, FR-WEL-040
func TestRelief_Pass(t *testing.T) {
	t.Parallel()
	w := reliefWorld(t, 45_000, 100, 3_000, 8_000, 12_000)
	var held [4]int64
	for id := AgentID(1); id <= 3; id++ {
		held[id], _ = balance(t, w, id)
	}
	if err := w.p16Relief(); err != nil {
		t.Fatalf("p16Relief() error = %v, want nil", err)
	}
	want := map[AgentID]int64{1: 350, 2: 100, 3: 0}
	for id, pay := range want {
		now, _ := balance(t, w, id)
		if now-held[id] != pay {
			t.Errorf("agent %d balance rose by %d, want the relief %d", id, now-held[id], pay)
		}
		if got := reliefOf(w, id); got != pay {
			t.Errorf("welfare scratch relief of agent %d = %d, want %d", id, got, pay)
		}
	}
	if got := w.estates.pool(w.ledger, 0); got != 44_550 {
		t.Errorf("pool after the pass = %d, want 44 550 (45 000 less 450)", got)
	}
	rp := w.relief
	if rp.paid[0] != 450 || rp.recipients[0] != 2 || rp.households[0] != 2 {
		t.Errorf("pass counts paid %d, recipients %d, households %d, want 450, 2, 2", rp.paid[0], rp.recipients[0], rp.households[0])
	}
}

// TestRelief_NoPoolNoPass verifies that a region with an empty pool, or with
// PoolRelease 0, pays nothing and counts nothing.
//
// Requirement: FR-EST-017
func TestRelief_NoPoolNoPass(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pool    Money
		release int64
	}{
		{"empty pool", 0, 100},
		{"release of zero", 45_000, 0},
		{"release truncated to zero", 99, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := reliefWorld(t, tt.pool, tt.release, 0, 0)
			before, _ := balance(t, w, 1)
			if err := w.p16Relief(); err != nil {
				t.Fatalf("p16Relief() error = %v, want nil", err)
			}
			if now, _ := balance(t, w, 1); now != before || reliefOf(w, 1) != 0 {
				t.Errorf("agent 1 balance %d -> %d with relief %d, want nothing paid", before, now, reliefOf(w, 1))
			}
			if got := w.estates.pool(w.ledger, 0); got != tt.pool {
				t.Errorf("pool = %d, want %d: nothing leaves it", got, tt.pool)
			}
			if rp := w.relief; rp.paid[0] != 0 || rp.recipients[0] != 0 {
				t.Errorf("pass counts paid %d, recipients %d, want 0 and 0", rp.paid[0], rp.recipients[0])
			}
		})
	}
}

// TestRelief_EstateNeverPaid verifies that no estate account is a recipient even
// when the relief line is at its bound, so every household has a need: the
// pseudo-agent rows keep no need and the estate's own money only falls by what
// the households are paid.
//
// Requirement: FR-EST-011
func TestRelief_EstateNeverPaid(t *testing.T) {
	t.Parallel()
	w := reliefWorld(t, 1_000_000, 10_000, 3_000, 0)
	w.cfg.Regions[0].ReliefBelow = MaxBasisPoints
	if err := w.p16Relief(); err != nil {
		t.Fatalf("p16Relief() error = %v, want nil", err)
	}
	if need := w.relief.rows[0].need; need != 0 {
		t.Errorf("need of the estate pseudo-agent = %d, want 0", need)
	}
	paid := reliefOf(w, 1) + reliefOf(w, 2)
	if paid == 0 {
		t.Fatalf("relief paid = 0, want the households paid: the setup proves nothing")
	}
	if got, want := w.estates.pool(w.ledger, 0), Money(1_000_000-paid); got != want {
		t.Errorf("pool after the pass = %d, want %d: it falls only by the households' relief", got, want)
	}
	if w.relief.rows[0].need != 0 || w.relief.recipients[0] != 2 {
		t.Errorf("recipients = %d, want the 2 households and no estate", w.relief.recipients[0])
	}
}

// TestRelief_ExternalAndCohort verifies that an external agent and an agent of
// weight 5 are paid, and that a cohort's need is the cohort's total.
//
// Arithmetic: both rings are at 5 000 and the line is 10 000. The household of
// weight 1 has U = 1 000 and a need of 1 000 * 5 000 / 10 000 = 500; the cohort of
// weight 5 has U = 5 000 and a need of 2 500. The release 10 000 covers the total
// 3 000, so each is paid its need in full.
//
// Requirement: FR-EST-018
func TestRelief_ExternalAndCohort(t *testing.T) {
	t.Parallel()
	c := shellConfig()
	c.Modules.Estates = &Estates{}
	c.Regions[0].PoolRelease = 10_000
	c.World.MinAgents = 1 // the group of 5 households is one agent of weight 5
	c.Roles = append(c.Roles, DefaultRole())
	c.Roles[1].ID = "reeve"
	c.Groups = []Group{{Region: "home", Role: "farmer", Households: 1}, {Region: "home", Role: "reeve", Households: 5, Scale: 5}}
	c.WarmStart = &WarmStart{Pools: []WarmPool{{Region: "home", Money: 10_000}}}
	w := newMill(t, c)
	if w.agents[2].weight != 5 {
		t.Fatalf("agent 2 weight = %d, want the cohort of 5", w.agents[2].weight)
	}
	w.agents[1].external = true
	scriptRatio(w, 1, 5_000)
	// The ring of a cohort is at the per-household ratio; its cost is the cohort's.
	a := &w.agents[2]
	a.history = newHistory(w.roles.row(a.role))
	for range a.history.Window() {
		a.history.Push(welfare.Record{Income: 2_500, Cost: 5_000})
	}
	if err := w.p16Relief(); err != nil {
		t.Fatalf("p16Relief() error = %v, want nil", err)
	}
	if got := reliefOf(w, 1); got != 500 {
		t.Errorf("relief of the external agent = %d, want 500", got)
	}
	if got := reliefOf(w, 2); got != 2_500 {
		t.Errorf("relief of the cohort of weight 5 = %d, want 2 500, its total need", got)
	}
	if w.relief.households[0] != 6 || w.relief.recipients[0] != 2 {
		t.Errorf("households %d, recipients %d, want 6 and 2", w.relief.households[0], w.relief.recipients[0])
	}
}

// TestRelief_NoAllocation verifies that a warmed pass allocates nothing.
//
// Requirement: NFR-EST-001
func TestRelief_NoAllocation(t *testing.T) {
	// Not parallel: testing.AllocsPerRun panics in a parallel test.
	w := reliefWorld(t, 1_000_000, 100, 3_000, 8_000, 12_000)
	if err := w.p16Relief(); err != nil {
		t.Fatalf("p16Relief() warm-up error = %v, want nil", err)
	}
	if got := testing.AllocsPerRun(50, func() {
		if err := w.p16Relief(); err != nil {
			t.Fatalf("p16Relief() error = %v, want nil", err)
		}
	}); got != 0 {
		t.Errorf("allocations per p16Relief = %v, want 0", got)
	}
}

// TestRelief_OverflowFailsStep is a Guard: a need that does not fit makes the
// pass return ErrOverflow naming the region and the agent, and nothing is paid
// before the refusal is known because every need is computed before the first
// transfer.
//
// Guard: the order of the passes of D350; paying while summing would leave the
// first households paid when the step fails.
//
// Arithmetic: with the line at MaxInt64 and rings at 10 000 each need is
// 1 000 * (MaxInt64 - 10 000) / 10 000 = 922 337 203 685 476 580, and the
// eleventh need takes the sum past MaxInt64.
//
// Requirement: FR-EST-018
func TestRelief_OverflowFailsStep(t *testing.T) {
	t.Parallel()
	ratios := make([]int64, 11)
	for i := range ratios {
		ratios[i] = 10_000
	}
	w := reliefWorld(t, 1_000_000, 10_000, ratios...)
	w.cfg.Regions[0].ReliefBelow = math.MaxInt64
	before, _ := balance(t, w, 1)
	err := w.p16Relief()
	if !errors.Is(err, ErrOverflow) {
		t.Fatalf("p16Relief() error = %v, want ErrOverflow", err)
	}
	if msg := err.Error(); !strings.Contains(msg, "agent 11") || !strings.Contains(msg, "region 0") {
		t.Errorf("p16Relief() error = %q, want it to name agent 11 and region 0", msg)
	}
	if now, _ := balance(t, w, 1); now != before || w.estates.pool(w.ledger, 0) != 1_000_000 {
		t.Errorf("after the refusal agent 1 holds %d (was %d) and the pool %d, want nothing paid", now, before, w.estates.pool(w.ledger, 0))
	}
}

// TestRelief_NotWage verifies that relief is income of the ratio and not earned:
// after the pass and P6.3 the newest record of a household on relief holds the
// relief, its total ratio is above its earned ratio, and the earned ratio is the
// one the household had without relief.
//
// Arithmetic: the ring is at 3 000 over a window of cost 1 000 per tick. The
// pass pays this lone household min(450, its need of 700) = 450, so the new
// record has income 450 (it earned nothing) and the record it replaces had 300.
// The total ratio rises; the earned one falls, as the new record earned 0.
//
// Requirement: FR-WEL-037, FR-WEL-040
func TestRelief_NotWage(t *testing.T) {
	t.Parallel()
	w := reliefWorld(t, 45_000, 100, 3_000)
	if err := w.p16Relief(); err != nil {
		t.Fatalf("p16Relief() error = %v, want nil", err)
	}
	w.p63Welfare()
	h := &w.agents[1].history
	recs := h.AppendRecords(nil)
	if last := recs[len(recs)-1]; last.Relief != 450 || last.Income < 450 {
		t.Errorf("newest record = %+v, want Relief 450 inside its Income", last)
	}
	total, ok1 := h.Ratio(1_000, 10_000)
	earned, ok2 := h.EarnedRatio(1_000, 10_000)
	if !ok1 || !ok2 || earned >= total {
		t.Errorf("ratio %d (%v), earned ratio %d (%v), want the earned ratio below the total by the relief", total, ok1, earned, ok2)
	}
}

// TestRelief_TimingAfterEntry verifies the timing of relief across ticks: the
// pool that relief reads at tick t+1 is the pool tick t left (after its entry
// step), and the relief of a tick is in that tick's welfare record, not the
// previous one's.
//
// Arithmetic: one household at ratio 0 has a need of its whole upkeep, 1 000. The
// pool 50 000 releases 500, below the need, so tick 0 pays exactly 500. Later
// ticks pay at most the release of the pool the previous tick left, which falls
// by 500 a tick and is refilled by nothing; relief raises the household's ratio, so
// its need falls and a later tick may pay less than the release.
//
// Requirement: FR-EST-017, FR-WEL-002
func TestRelief_TimingAfterEntry(t *testing.T) {
	t.Parallel()
	var paid []ReliefPaid
	obs := ObserverFunc(func(e Event) {
		if r, ok := e.(ReliefPaid); ok {
			paid = append(paid, r)
		}
	})
	w := reliefWorldWith(t, []Option{WithObserver(obs)}, 50_000, 100, 0)
	for range 3 {
		before := w.estates.pool(w.ledger, 0)
		if err := w.Step(); err != nil {
			t.Fatalf("Step() error = %v, want nil", err)
		}
		tick := w.Tick() - 1
		last := paid[len(paid)-1]
		release := int64(before) * 100 / 10_000
		if last.Tick != tick || int64(last.Paid) > release || (tick == 0 && int64(last.Paid) != release) {
			t.Errorf("tick %d: ReliefPaid = %+v, want Paid within the release %d of the pool %d the previous tick left (equal at tick 0)", tick, last, release, before)
		}
		recs := w.agents[1].history.AppendRecords(nil)
		if got := recs[len(recs)-1].Relief; got != int64(last.Paid) {
			t.Errorf("tick %d: newest welfare record holds relief %d, want %d: the relief of its own tick", tick, got, last.Paid)
		}
	}
}
