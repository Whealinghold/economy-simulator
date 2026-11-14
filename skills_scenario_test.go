package economysim_test

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/occupation"
	"github.com/Whealinghold/economy-sim/internal/skill"
)

// packFiveSkills is pack five, warm, with the skill overlay: the second eligible recipe
// of the reference chain (C13) is on the crafters' list in this pack.
var packFiveSkills = []string{"base.toml", "lines.toml", "five.toml", "warm-five.toml", "overlay/skill.toml"}

// s06Ticks is the length of S06 (spec-reference-chain.md 7.4: 1 095 ticks).
const s06Ticks = 1095

// warnCounter is a slog.Handler that is enabled for Warn and above only, so a long
// run does not build the Debug lines of the world, and counts the Warns by message.
type warnCounter struct{ n map[string]int }

func (warnCounter) Enabled(_ context.Context, l slog.Level) bool { return l >= slog.LevelWarn }
func (w warnCounter) Handle(_ context.Context, r slog.Record) error {
	w.n[r.Message]++
	return nil
}
func (w warnCounter) WithAttrs([]slog.Attr) slog.Handler { return w }
func (w warnCounter) WithGroup(string) slog.Handler      { return w }

// s06Run is what a run of S06 measured.
type s06Run struct {
	started, completed, ended int64
	endShare, swing           int64 // the cooks' share of the households of the region, in basis points
	hasShare                  bool
	premium                   int64
	hasPremium                bool
	live, noPlaces            int64 // trainees at the end, Warns "no places open"
	predicted, tolerance      int64 // check D: the premium at break-even and its tolerance
	hasPredicted, dOK         bool
	final                     economysim.Snapshot
}

// s06Layers is the pack of one combination of S06: pack one, warm, with the
// imbalance of the cooks, and with the traineeship (overlay/skill) and without the
// tool (overlay/notool) as the combination says.
func s06Layers(skillOn, notool bool) []string {
	layers := append(slices.Clone(packOne), "overlay/imbalance-cook.toml")
	if skillOn {
		layers = append(layers, "overlay/skill.toml")
	}
	if notool {
		layers = append(layers, "overlay/notool.toml")
	}
	return layers
}

// roleIndex returns the index of the named role in the names table, or -1.
func skillRoleIndex(s economysim.Snapshot, name string) int { return slices.Index(s.Names.Roles, name) }

// runS06 runs one combination under seed with the audit, counting the events of the
// traineeships and the Warns, and measuring the share of the cooks of the first region
// at every fifth tick of the last 730 (spec-reference-chain.md S06).
func runS06(t *testing.T, layers []string, seed uint64) s06Run {
	t.Helper()
	cfg := loadValidPack(t, layers...)
	var r s06Run
	obs := economysim.ObserverFunc(func(e economysim.Event) {
		switch e.(type) {
		case economysim.TraineeshipStarted:
			r.started++
		case economysim.TraineeshipCompleted:
			r.completed++
		case economysim.TraineeshipEnded:
			r.ended++
		}
	})
	warns := warnCounter{n: map[string]int{}}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(seed), economysim.WithAudit(), economysim.WithObserver(obs), economysim.WithLogger(slog.New(warns)))
	if err != nil {
		t.Fatalf("seed %d: NewWorld() error = %v, want nil", seed, err)
	}
	var lo, hi int64 = 1 << 62, -1
	for i := range s06Ticks {
		if err := w.Step(); err != nil {
			t.Fatalf("seed %d: Step() at tick %d error = %v, want nil", seed, i, err)
		}
		if i >= s06Ticks-730 && i%5 == 0 {
			s, err := w.Snapshot()
			if err != nil {
				t.Fatalf("seed %d: Snapshot() at tick %d error = %v, want nil", seed, i, err)
			}
			if share, ok := cookShare(s); ok {
				lo, hi, r.hasShare = min(lo, share), max(hi, share), true
			}
		}
	}
	if r.final, err = w.Snapshot(); err != nil {
		t.Fatalf("seed %d: Snapshot() error = %v, want nil", seed, err)
	}
	r.endShare, _ = cookShare(r.final)
	if r.hasShare {
		r.swing = hi - lo
	}
	r.live, r.noPlaces = int64(len(r.final.Trainees)), int64(warns.n["no places open"])
	if c := skillRoleIndex(r.final, "cook"); c >= 0 {
		for _, row := range r.final.Roles {
			if int(row.Role) == c && row.Region == 0 {
				r.premium, r.hasPremium = row.PremiumBP, row.HasPremium
			}
		}
	}
	r.checkD(cfg, "cook")
	return r
}

// checkD reads check D on the run (FR-SKL-024): the premium that the break-even of the
// cook's tool predicts, 10 000 + P * CapitalReturn / (D_work * W_unskilled), with P the
// reference price of the tool its recipes require, D_work the ticks of a year and
// W_unskilled the mean observed wage of the unskilled roles of the region, against the
// observed premium within the tolerance. The tuition is not in it (C_train is 0 here), so a
// run with the traineeship is read against the tool alone. A run with no tool, no wage or
// no premium has none. It is reported by S06 and asserted by no pack (N-352).
func (r *s06Run) checkD(cfg economysim.Config, role string) {
	var price int64
	for _, ro := range cfg.Roles {
		if ro.ID != role {
			continue
		}
		for _, name := range ro.Recipes {
			for _, rc := range cfg.Recipes {
				for _, tool := range rc.Requires {
					if rc.ID == name {
						for _, c := range cfg.Commodities {
							if c.ID == tool.Commodity {
								price = max(price, int64(c.ReferencePrice))
							}
						}
					}
				}
			}
		}
	}
	var wages, n int64
	for _, row := range r.final.Roles {
		if row.Region == 0 && row.HasWage && cfg.Roles[row.Role].Unskilled {
			wages, n = wages+int64(row.WageObserved), n+1
		}
	}
	if n == 0 || wages <= 0 || !r.hasPremium {
		return
	}
	pred, ok := skill.BreakEven(price, 0, cfg.Regions[0].CapitalReturn, cfg.Calendar.TicksPerYear(), wages/n)
	if !ok {
		return
	}
	r.predicted, r.hasPredicted = pred, true
	r.dOK, r.tolerance = premiumCheckD(r.premium, pred, 0)
}

// cookShare returns the share, in basis points, of the households of the first
// region that are cooks, and false when the region has no household.
func cookShare(s economysim.Snapshot) (int64, bool) {
	c := skillRoleIndex(s, "cook")
	var cooks, all int64
	for _, row := range s.Roles {
		if row.Region != 0 {
			continue
		}
		all += row.Households
		if int(row.Role) == c {
			cooks += row.Households
		}
	}
	if all == 0 {
		return 0, false
	}
	return cooks * 10_000 / all, true
}

// TestScenarioS06_IncomeGapTraineeship runs S06 (spec-reference-chain.md 7.4) on pack
// one, warm, with the imbalance of the cooks, in its four combinations of the
// traineeship (overlay/skill) and of the tool (overlay/notool), 1 095 ticks, with the
// audit, for the seeds of the scenario. It asserts what is exact: the books balance every
// tick (the audit), a world without the overlay has no traineeship event, row and
// Warn, and a world with it ends with every start accounted for (started = completed +
// ended + the trainees still in training, FR-SKL-021) and with places filled. It records
// and never asserts what the bands of STY will judge once they have a value, and what the
// premium checks C, A and D read (the shares, the swing, the observed premium of the cook,
// the counters of the traineeships and the Warn): the pack collapses under its own rules
// (N-352), so the premium of its cooks is a finding and not a verdict. The outcomes are
// written to testdata/chains/expected/one.scenarios.txt with -update.
//
// Requirement: FR-SKL-005, FR-SKL-010, FR-SKL-011, FR-SKL-015, FR-SKL-016, FR-SKL-017, FR-SKL-021, FR-SKL-022, FR-SKL-023, FR-SKL-024, FR-REF-020
//
// Guard: a traineeship that lost a start (a place filled and never ended or completed), or
// a world with no overlay that grew a traineeship row, fails here.
func TestScenarioS06_IncomeGapTraineeship(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	const scenario = "S06"
	for _, c := range []struct {
		name            string
		skillOn, notool bool
	}{{"tool", false, false}, {"tool+skill", true, false}, {"notool", false, true}, {"notool+skill", true, true}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			layers := s06Layers(c.skillOn, c.notool)
			rep := newScenReport(fmt.Sprintf("S06 one+imbalance-cook %s seeds 1-%d, %d ticks", c.name, scenarioSeedCount(), s06Ticks))
			t.Run("seeds", func(t *testing.T) {
				for _, seed := range seedList(scenarioSeedCount()) {
					t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
						t.Parallel()
						r := runS06(t, layers, seed)
						reportS06(rep, seed, r)
						if !c.skillOn {
							checkExact(t, rep, scenario, c.name, seed, "no traineeship without the overlay", r.started+r.completed+r.ended+r.live+r.noPlaces == 0, "started %d, completed %d, ended %d, trainees %d, Warns %d", r.started, r.completed, r.ended, r.live, r.noPlaces)
							return
						}
						checkExact(t, rep, scenario, c.name, seed, "every start is completed, ended or still in training", r.started == r.completed+r.ended+r.live, "started %d, completed %d, ended %d, in training %d", r.started, r.completed, r.ended, r.live)
						checkExact(t, rep, scenario, c.name, seed, "places are filled", r.started > 0, "no traineeship started in %d ticks", s06Ticks)
					})
				}
			})
			rep.flush(t)
		})
	}
}

// reportS06 records the measures of one run of S06 under seed.
func reportS06(rep *scenReport, seed uint64, r s06Run) {
	if r.hasPredicted {
		rep.num(seed, "check D predicted premium bp", r.predicted)
		rep.check(seed, "check D reported: premium within the tolerance of the break-even", r.dOK, fmt.Sprintf("premium %d, predicted %d, tolerance %d", r.premium, r.predicted, r.tolerance))
	}
	rep.num(seed, "traineeships started", r.started)
	rep.num(seed, "traineeships completed", r.completed)
	rep.num(seed, "traineeships ended without completing", r.ended)
	rep.num(seed, "trainees at the end", r.live)
	rep.num(seed, "Warns no places open", r.noPlaces)
	rep.num(seed, "households at end", r.final.Households)
	if r.hasShare {
		rep.num(seed, "cook share at end (balanced 2267) bp", r.endShare)
		rep.num(seed, "cook share swing over the last 730 ticks bp", r.swing)
	} else {
		rep.absent(seed, "cook share at end (balanced 2267) bp")
	}
	if r.hasPremium {
		rep.num(seed, "observed premium of the cook bp (check A reports, the band 14100 to 19300)", r.premium)
		rep.check(seed, "check C reported: premium above 10 000 (a tool) or within 1 000 of it (no tool)", r.premium > 10_000 || (r.premium >= 9_000 && r.premium <= 11_000), fmt.Sprintf("premium %d", r.premium))
	} else {
		rep.absent(seed, "observed premium of the cook bp (check A reports, the band 14100 to 19300)")
	}
}

// TestSkillPremium_CheckC verifies, at the rule that makes it (spec-skill-barriers.md 3,
// FR-SKL-023), the ordinal properties of the premium that the entry rule produces: the
// premium at which entry breaks even is above 10 000 whenever the tool is worth at
// least a basis point of the annual wage, rises with the price of the tool and with the
// cost of the course, and is exactly 10 000 when both are free; and the application rule
// (FR-SKL-005) agrees: the least advantage over the own role that an agent applies for
// is the capital times the return over the annual upkeep, rounded up, and never falls
// as the price rises.
//
// Arithmetic: an unskilled wage of 2 500 for 365 ticks is 912 500 a year, and a return
// of 1 000 basis points on a tool of 41 155 is 41 155 * 1 000 / 912 500 = 45 basis points
// (truncated), so the premium is 10 045. For the rule, an upkeep of 1 000 for 365 ticks
// is 365 000, and a tool of 41 155 at 1 000 basis points needs an advantage of 41 155 000
// / 365 000 = 112.75, so 113 and not 112 (the rule is "at least").
//
// Requirement: FR-SKL-023, FR-SKL-005, FR-SKL-006
//
// Guard: the premium checks cannot be asserted on the packs, which collapse (N-352), so
// the property is held here, at the rule, where nothing else moves it.
func TestSkillPremium_CheckC(t *testing.T) {
	t.Parallel()
	const wage, work, ret = int64(2_500), int64(365), int64(1_000)
	var prevPremium, prevAdvantage int64 = -1, -1
	for _, price := range []int64{0, 10_000, 41_155, 400_000, 4_000_000} {
		got, ok := skill.BreakEven(price, 0, ret, work, wage)
		if !ok {
			t.Fatalf("BreakEven(price %d) not ok, want a premium", price)
		}
		if (price == 0) != (got == 10_000) && price*ret >= work*wage {
			t.Errorf("BreakEven(price %d) = %d, want exactly 10 000 for a free tool and above it for a tool worth a basis point", price, got)
		}
		if got < prevPremium {
			t.Errorf("BreakEven(price %d) = %d, below %d of a cheaper tool: the premium must rise with the price", price, got, prevPremium)
		}
		advantage := int64(0)
		for !occupation.ApplyOK(occupation.ApplyParams{ED: advantage, EO: 0, ToolPrice: price, CapitalReturn: ret, Upkeep: 1_000, Year: work}) {
			advantage++
		}
		if want := (price*ret + 365_000 - 1) / 365_000; advantage != want {
			t.Errorf("ApplyOK needs an advantage of %d for a tool of %d, want %d = the capital times the return over the annual upkeep, rounded up", advantage, price, want)
		}
		if advantage < prevAdvantage {
			t.Errorf("advantage needed for a tool of %d = %d, below %d of a cheaper tool", price, advantage, prevAdvantage)
		}
		prevPremium, prevAdvantage = got, advantage
	}
	if got, _ := skill.BreakEven(41_155, 0, ret, work, wage); got != 10_045 {
		t.Errorf("BreakEven(41 155) = %d, want 10 045", got)
	}
	tool, _ := skill.BreakEven(41_155, 0, ret, work, wage)
	toolAndCourse, _ := skill.BreakEven(41_155, 200_000, ret, work, wage)
	if toolAndCourse <= tool {
		t.Errorf("BreakEven with a course of 200 000 = %d, with the tool alone %d, want the course to raise it", toolAndCourse, tool)
	}
}

// premiumCheckD is check D of spec-skill-barriers.md 3 (FR-SKL-024): the observed premium
// is accepted when it is within the tolerance of the predicted one, the tolerance being
// max(25% of the predicted excess over 10 000, 200 basis points, cvx2), cvx2 being twice
// the coefficient of variation of the premium across seeds.
func premiumCheckD(observed, predicted, cvx2 int64) (ok bool, tolerance int64) {
	tolerance = skill.Tolerance(predicted-10_000, cvx2)
	return observed >= predicted-tolerance && observed <= predicted+tolerance, tolerance
}

// TestSkillPremium_CheckD_Tolerance verifies the acceptance of check D on the two worked
// figures of spec-skill-barriers.md 3: a tool-only chain predicts an excess of 400, 25%
// of which is 100, so the floor of 200 basis points holds, and a chain with training
// predicts an excess of 2 733, a tolerance of 683; an observed premium on the edge of the
// tolerance is accepted, one basis point beyond it is not, and twice the coefficient of
// variation raises the tolerance when it is larger.
//
// Requirement: FR-SKL-024, FR-SKL-025
//
// Guard: the packs collapse (N-352), so the verdict of check D is a report there; this is
// the one place that asserts the acceptance rule itself.
func TestSkillPremium_CheckD_Tolerance(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name                 string
		predicted, cv        int64
		wantTol, observedOK  int64 // observedOK is the farthest accepted premium
		beyond, wantAccepted int64 // beyond is one basis point past the edge
	}{
		{"tool only: the floor of 200 holds", 10_400, 0, 200, 10_600, 10_601, 1},
		{"with training: 25% of 2 733 is 683", 12_733, 0, 683, 13_416, 13_417, 1},
		{"twice the coefficient of variation is larger", 10_400, 450, 450, 10_850, 10_851, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ok, tol := premiumCheckD(tt.observedOK, tt.predicted, tt.cv)
			if tol != tt.wantTol || !ok {
				t.Errorf("premiumCheckD(%d, %d, %d) = %v, tolerance %d, want accepted with tolerance %d", tt.observedOK, tt.predicted, tt.cv, ok, tol, tt.wantTol)
			}
			if ok, _ := premiumCheckD(tt.beyond, tt.predicted, tt.cv); ok {
				t.Errorf("premiumCheckD(%d, %d, %d) accepted, want a premium one basis point beyond the tolerance rejected", tt.beyond, tt.predicted, tt.cv)
			}
			if ok, _ := premiumCheckD(tt.predicted, tt.predicted, tt.cv); !ok {
				t.Errorf("premiumCheckD(predicted = observed) rejected, want accepted")
			}
		})
	}
}

// TestSkillPremium_CheckA_ReportOnly reports the premium of the cooks of S06 against the
// whole-series band of check A (14 100 to 19 300 basis points, `evidence.md`) and never
// fails on it (FR-SKL-024): a value outside the band is a prompt to look at the recipe
// economics. It runs the four combinations of S06 for seed 1 on pack one over a year.
//
// Requirement: FR-SKL-024, FR-SKL-025
//
// Guard: the band is data of the reference chain and not an assertion; the test must
// log it and pass whatever the premium is.
func TestSkillPremium_CheckA_ReportOnly(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("run-level test, skipped with -short")
	}
	for _, c := range []struct {
		name            string
		skillOn, notool bool
	}{{"tool", false, false}, {"tool+skill", true, false}, {"notool", false, true}, {"notool+skill", true, true}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cfg := loadValidPack(t, s06Layers(c.skillOn, c.notool)...)
			w := newPackWorld(t, cfg, 1)
			if _, err := w.Run(context.Background(), 365); err != nil {
				t.Fatalf("Run(365) error = %v, want nil", err)
			}
			s, err := w.Snapshot()
			if err != nil {
				t.Fatalf("Snapshot() error = %v, want nil", err)
			}
			for _, row := range s.Roles {
				if row.HasPremium && s.Names.Roles[row.Role] == "cook" && row.Region == 0 {
					in := row.PremiumBP >= 14_100 && row.PremiumBP <= 19_300
					t.Logf("check A (%s): cook premium %d bp, whole-series band 14100 to 19300, inside: %v (report only, FR-SKL-024)", c.name, row.PremiumBP, in)
				}
			}
		})
	}
}

// TestScenarioS06_SecondRecipeHasNoPlaces runs the second leg of C13 (S06: "the second
// recipe is not on the cook's list in one, so that leg is exercised in five"): pack five,
// warm, with overlay/skill.toml, seed 1, 300 ticks, with the audit. The crafters' eligible
// recipe makes less than it costs even at a full yield, so a place never opens for it
// and the Warn "no places open" is logged for the crafters, while the cooks, whose
// eligible recipe pays, have trainees; the snapshot holds the counterpart (a queue of
// crafters and no trainee of the role).
//
// Requirement: FR-SKL-016, FR-SKL-022, FR-REF-020
//
// Guard: without a negative-gain role the Warn is never seen in a run of a pack, and the
// incentive cliff (STB-L37) it counts would be tested only on a planted world.
func TestScenarioS06_SecondRecipeHasNoPlaces(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	cfg := loadValidPack(t, packFiveSkills...)
	h := newCaptureHandler()
	w := newPackWorld(t, cfg, 1, economysim.WithAudit(), economysim.WithLogger(slog.New(h)))
	for i := range 300 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d error = %v, want nil", i, err)
		}
	}
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	crafter, cook := slices.Index(s.Names.Roles, "crafter_a"), slices.Index(s.Names.Roles, "cook")
	var crafterQueue, crafterTaken, cookTaken int64
	for _, row := range s.Roles {
		switch int(row.Role) {
		case crafter:
			crafterQueue, crafterTaken = crafterQueue+row.Queue, crafterTaken+row.PlacesTaken
		case cook:
			cookTaken += row.PlacesTaken
		}
	}
	if crafterTaken != 0 || crafterQueue == 0 {
		t.Errorf("crafter_a: %d trainees, %d queued, want none in training and a queue (the gain of finish_a_basic is negative)", crafterTaken, crafterQueue)
	}
	if cookTaken == 0 {
		t.Errorf("cook: no trainee in training at tick 300, want some (prepare_basic pays)")
	}
	warned := false
	for _, l := range h.sink.lines {
		if l.msg == "no places open" && l.level == slog.LevelWarn {
			warned = true
		}
	}
	if !warned {
		t.Errorf("no Warn \"no places open\" in 300 ticks, want one for the crafters whose gain is negative")
	}
}

// TestScenario_SkillThrottleVisible verifies the throttle of spec-skill-barriers.md 4.9
// as the snapshot shows it: on pack one, warm, with the overlay, seed 1, 300 ticks, a
// course of 1 000 ticks keeps every place of the cooks taken (Places 0 with trainees
// in training) and lets the queue grow longer than a course of 120 ticks does, where
// places open again as courses end.
//
// Arithmetic: with MaxTrainees 3 and about 85 cooks that can train, a course of 1 000
// ticks admits 255 trainees at most and none more until the first completes at tick
// 1 000, so from the moment they are all taken the queue only grows; at 120 ticks a place
// is freed every few ticks.
//
// Requirement: FR-SKL-017, FR-SKL-019
//
// Guard: the throttle is what the snapshot columns exist to show (FR-STB-013); a change
// that ignored MaxTrainees would fill places beyond it and this test would not see Places 0.
func TestScenario_SkillThrottleVisible(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	long := loadPackBytes(t, []byte("[role.cook]\ntraining_ticks = 1000\n[role.crafter_a]\ntraining_ticks = 1000\n"), packOneSkills...)
	short := loadValidPack(t, packOneSkills...)
	measure := func(cfg economysim.Config) (maxQueue, places, taken int64) {
		w := newPackWorld(t, cfg, 1, economysim.WithAudit())
		for i := range 300 {
			if err := w.Step(); err != nil {
				t.Fatalf("Step() at tick %d error = %v, want nil", i, err)
			}
			if i%5 == 0 {
				s, err := w.Snapshot()
				if err != nil {
					t.Fatalf("Snapshot() error = %v, want nil", err)
				}
				for _, row := range s.Roles {
					maxQueue = max(maxQueue, row.Queue)
				}
			}
		}
		s, err := w.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v, want nil", err)
		}
		for _, row := range s.Roles {
			places, taken = places+row.Places, taken+row.PlacesTaken
		}
		return maxQueue, places, taken
	}
	lq, lp, lt := measure(long)
	sq, sp, _ := measure(short)
	if lp != 0 || lt == 0 {
		t.Errorf("course of 1 000 ticks: places open %d, trainees %d, want 0 places open and trainees in training (every place taken)", lp, lt)
	}
	if lq <= sq {
		t.Errorf("longest queue with a course of 1 000 ticks = %d, with 120 = %d, want the longer course to queue more", lq, sq)
	}
	if sp == 0 {
		t.Errorf("course of 120 ticks: places open %d, want some (courses end and free places)", sp)
	}
}

// TestConfigOnly_SkillNeedsOnlyConfiguration verifies architecture.md 9.4 for the skill
// barriers: pack second, a setting that shares nothing with the reference chain, is
// given a role that trains and a recipe a trainee may run by a layer of data alone
// (overlay/second-skill.toml), and runs 400 ticks for seeds 1 to 3 with the audit; a
// course starts and completes in each. The test names no commodity, recipe or role.
//
// Requirement: FR-SKL-010, FR-SKL-028, FR-REF-031
//
// Guard: a branch on a commodity, recipe or role identifier in the traineeship would fail
// on a setting it never heard of.
func TestConfigOnly_SkillNeedsOnlyConfiguration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadValidPack(t, append(slices.Clone(packSecond), "overlay/second-skill.toml")...)
	for _, seed := range seedList(min(3, scenarioSeedCount())) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			var started, completed int
			obs := economysim.ObserverFunc(func(e economysim.Event) {
				switch e.(type) {
				case economysim.TraineeshipStarted:
					started++
				case economysim.TraineeshipCompleted:
					completed++
				}
			})
			w := newPackWorld(t, cfg, seed, economysim.WithAudit(), economysim.WithObserver(obs))
			for i := range 400 {
				if err := w.Step(); err != nil {
					t.Fatalf("seed %d: Step() at tick %d error = %v, want nil", seed, i, err)
				}
			}
			if started == 0 || completed == 0 {
				t.Errorf("seed %d: %d traineeships started and %d completed in 400 ticks, want both above 0", seed, started, completed)
			}
		})
	}
}

// TestConfig_NoToolPriceField verifies FR-SKL-003: no configuration struct has a
// field for the price of a tool, so a tool's price is whatever the market makes it. It
// reads every field name of the configuration types reachable from Config by reflection.
//
// Requirement: FR-SKL-003
//
// Guard: a "ToolPrice" configuration field would put a number that the market owns into
// the data and make the entry rule read it instead of the price the world publishes.
func TestConfig_NoToolPriceField(t *testing.T) {
	t.Parallel()
	seen := map[reflect.Type]bool{}
	var walk func(reflect.Type, string)
	walk = func(typ reflect.Type, path string) {
		for typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array {
			typ = typ.Elem()
		}
		if typ.Kind() != reflect.Struct || seen[typ] {
			return
		}
		seen[typ] = true
		for i := range typ.NumField() {
			f := typ.Field(i)
			if n := strings.ToLower(f.Name); strings.Contains(n, "toolprice") || (strings.Contains(n, "tool") && strings.Contains(n, "price")) {
				t.Errorf("configuration field %s.%s names a tool price, want none (FR-SKL-003)", path, f.Name)
			}
			walk(f.Type, path+"."+f.Name)
		}
	}
	walk(reflect.TypeFor[economysim.Config](), "Config")
	if len(seen) < 10 {
		t.Fatalf("walked %d configuration types, want the whole tree", len(seen))
	}
}

// toolsOf returns, per role name, the commodities its recipes require and the commodities
// its recipes make, from the configuration alone.
func toolsOf(cfg economysim.Config) (requires, makes map[string]map[string]bool) {
	rec := map[string]economysim.Recipe{}
	for _, r := range cfg.Recipes {
		rec[r.ID] = r
	}
	requires, makes = map[string]map[string]bool{}, map[string]map[string]bool{}
	for _, role := range cfg.Roles {
		requires[role.ID], makes[role.ID] = map[string]bool{}, map[string]bool{}
		for _, name := range role.Recipes {
			for _, tool := range rec[name].Requires {
				requires[role.ID][tool.Commodity] = true
			}
			for _, o := range rec[name].Outputs {
				makes[role.ID][o.Commodity] = true
			}
		}
	}
	return requires, makes
}

// TestStrategy_ToolHolderDoesNotResell verifies FR-SKL-007: over 200 ticks of pack one,
// warm, with the traineeship, no agent places an ask for a commodity that a recipe of its
// role requires unless its role makes it, so a tool is bound to its holder; the estate,
// which is no agent of a role, sells the tools of the dead.
//
// Requirement: FR-SKL-007
//
// Guard: the binding is the default strategy's, there is no other mechanism (spec-skill-
// barriers.md 11); a strategy that sold its tool for cash would be seen here.
func TestStrategy_ToolHolderDoesNotResell(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneSkills...)
	requires, makes := toolsOf(cfg)
	res := runPack(t, cfg, runSpec{Seed: 1, Ticks: 200, Drive: driveStep, Observe: true})
	names := res.Final.Names
	role := map[economysim.AgentID]string{}
	asks := 0
	for _, e := range res.Events {
		switch x := e.(type) {
		case economysim.AgentSpawned:
			role[x.Agent] = names.Roles[x.Role]
		case economysim.RoleChanged:
			role[x.Agent] = names.Roles[x.To]
		case economysim.OrderPlaced:
			if x.Side != economysim.Ask {
				continue
			}
			asks++
			r, c := role[x.Agent], names.Commodities[x.Commodity]
			if r == "" {
				continue // an estate: no role
			}
			if requires[r][c] && !makes[r][c] {
				t.Errorf("tick %d: agent %d of role %s asked for %s, a tool its role requires and does not make (FR-SKL-007)", x.Tick, x.Agent, r, c)
			}
		}
	}
	if asks == 0 {
		t.Fatalf("no ask in 200 ticks, want the strategy to have traded")
	}
}

// TestScenario_ToolHolderDiesEstateSells verifies FR-SKL-008 and FR-SKL-009 end to end on
// pack one, warm, with the audit: over 400 ticks agents holding a tool die and the
// estate lists it (EstateOpened with lots), and the estate sells a tool afterwards
// (a TradeExecuted of a commodity that some recipe requires, with an estate as the
// seller) while the audit holds the books of goods and money exact every tick.
//
// Requirement: FR-SKL-008, FR-SKL-009
//
// Guard: the estate's sale only moves the tool between accounts, so conservation holds
// (FR-SKL-009); a sale that created or lost a unit would fail the audit of this run.
func TestScenario_ToolHolderDiesEstateSells(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	cfg := loadValidPack(t, packOneSkills...)
	tools := map[string]bool{}
	for _, r := range cfg.Recipes {
		for _, tool := range r.Requires {
			tools[tool.Commodity] = true
		}
	}
	res := runPack(t, cfg, runSpec{Seed: 1, Ticks: 400, Drive: driveStep, Observe: true, Audit: true})
	names := res.Final.Names
	listed, sold := 0, 0
	for _, e := range res.Events {
		switch x := e.(type) {
		case economysim.EstateOpened:
			if x.GoodsListed > 0 {
				listed++
			}
		case economysim.TradeExecuted:
			if tools[names.Commodities[x.Commodity]] && int(x.Seller) < len(names.Regions) {
				sold++
			}
		}
	}
	if listed == 0 || sold == 0 {
		t.Errorf("estates opened with lots %d times and sold a tool %d times in 400 ticks, want both above 0", listed, sold)
	}
}

// TestDeterminism_Skills verifies the cases of TestDeterminism_SameBytes that detCases(true)
// keeps on pack one, warm, with the traineeship overlay, seeds 1 and 2, 400 ticks: a run
// split into Run calls, with a logger, an observer or the audit, ends in the snapshot,
// the state hash and the config hash of the bare Step run, and its events and rendering
// equal the reference's. The 400 ticks hold placements, completions and the removal of
// trainers, so the queue, the accumulator and the courses are under it.
//
// Requirement: FR-CFG-026, FR-OBS-023, FR-WLD-063, FR-SKL-027, FR-SKL-028
//
// Guard: the traineeships draw no random number and read no wall clock; a draw or a
// dependence on the observer would fail here.
func TestDeterminism_Skills(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packOneSkills...)
	report := solveCold(t)
	render := func(seed uint64, res result) []byte {
		return renderGolden("one", seed, detTicks, samplesOnly(res, detTicks), report, cfg.Warnings())
	}
	for _, seed := range seedList(min(2, scenarioSeedCount())) {
		t.Run(fmt.Sprintf("seed %d", seed), func(t *testing.T) {
			t.Parallel()
			base := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks}))
			ref := runPack(t, cfg, detStep(runSpec{Seed: seed, Ticks: detTicks, Observe: true}))
			refEvents := renderEvents(withoutRunBrackets(ref))
			refGolden := render(seed, ref)
			if d := firstTickDiff(base.Samples, ref.Samples); d != "" {
				t.Fatalf("seed %d: observed Step run differs from the bare one at %s (FR-OBS-023)", seed, d)
			}
			for _, tc := range detCases(true) {
				spec := tc.spec()
				spec.Seed, spec.Ticks = seed, detTicks
				got := runPack(t, cfg, spec)
				if !reflect.DeepEqual(got.Final, base.Final) {
					t.Errorf("seed %d, case %q: final snapshot differs from the bare run", seed, tc.name)
				}
				if economysim.HashWorld(got.World) != economysim.HashWorld(base.World) {
					t.Errorf("seed %d, case %q: state hash of the world differs from the bare run", seed, tc.name)
				}
				if spec.Observe {
					if d := firstLineDiff(refEvents, renderEvents(withoutRunBrackets(got))); d != "" {
						t.Errorf("seed %d, case %q: events differ from the reference at %s", seed, tc.name, d)
					}
				}
				if spec.Observe && len(spec.Drive.chunks) == 0 {
					if d := firstLineDiff(refGolden, render(seed, got)); d != "" {
						t.Errorf("seed %d, case %q: rendering differs from the reference at %s", seed, tc.name, d)
					}
				}
			}
		})
	}
}
