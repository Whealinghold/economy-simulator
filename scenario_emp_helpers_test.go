package economysim_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file is the helper layer of the scenarios of the employment module, S97 to S101
// (spec-employment.md 17, plan 3-emp Phase 8): a tape that keeps the labour and hire events
// of a run as series by tick, and the checks every one of them shares: the hires conform to
// the bill and the order of the events, no hire has no bill, the escrow is whole, and the
// money of the world moves only by the boundary flows.

// labourTape is the observer of an employment scenario. It keeps the LabourRound sums and
// the hires as series by tick, summed over regions, and the net money of the boundary flows
// by tick; a hireChecker (employment_hire_e2e_test.go) checks the order and the bill.
type labourTape struct {
	check *hireChecker
	seen  int
	// net is the money the faucets put in less the sinks took out, by tick; the
	// construction bucket is tick -1.
	net map[int64]economysim.Money
	// offered, hired, noVacancy, posted, filled and unfilled are the LabourRound sums by
	// tick; formedHH the households of the HireFormed of the tick and wageHH their wage
	// times households.
	offered, hired, noVacancy, posted, filled, unfilled, formedHH, wageHH []int64
	// formed and ended count the hire events; noBill the formations that escrowed nothing;
	// paid the wages of the LabourRound; employerFunds and noInput the positions those two
	// checks closed; lastHire the tick of the last formation, -1 for none.
	formed, ended, noBill, employerFunds, noInput, lastHire int64
	paid                                                    economysim.Money
	// byCause counts the hires ended by cause. anyCause says that a hire may end by any
	// cause of the removals and role changes of the pack's spawner, not only by completion,
	// which the checker of the one-tick jobs of S97 to S100 requires.
	byCause  map[economysim.Cause]int64
	anyCause bool
}

func newLabourTape() *labourTape {
	return &labourTape{check: newHireChecker(), net: map[int64]economysim.Money{}, byCause: map[economysim.Cause]int64{}, lastHire: -1}
}

// bump adds v to the series s at tick, growing it with zeros.
func bump(s *[]int64, tick, v int64) {
	for int64(len(*s)) <= tick {
		*s = append(*s, 0)
	}
	(*s)[tick] += v
}

// Observe implements economysim.Observer.
func (l *labourTape) Observe(e economysim.Event) {
	hireStep(l.check, l.seen, e)
	l.seen++
	switch e := e.(type) {
	case economysim.BoundaryFlow:
		l.flow(e)
	case economysim.HireFormed:
		l.formed++
		l.lastHire = e.Tick
		if e.Escrowed <= 0 {
			l.noBill++
		}
		bump(&l.formedHH, e.Tick, e.Households)
		bump(&l.wageHH, e.Tick, int64(e.Wage)*e.Households)
	case economysim.HireEnded:
		l.ended++
		l.byCause[e.Cause]++
	case economysim.LabourRound:
		l.round(e)
	}
}

// flow adds the money a boundary flow put in or took out.
func (l *labourTape) flow(e economysim.BoundaryFlow) {
	switch {
	case e.Commodity != economysim.NoCommodity:
	case e.Kind == economysim.KindFaucet:
		l.net[e.Tick] += economysim.Money(e.Total)
	default:
		l.net[e.Tick] -= economysim.Money(e.Total)
	}
}

// round adds a LabourRound to the series of its tick.
func (l *labourTape) round(e economysim.LabourRound) {
	bump(&l.offered, e.Tick, e.Offered)
	bump(&l.hired, e.Tick, e.Hired)
	bump(&l.noVacancy, e.Tick, e.NoVacancy)
	bump(&l.posted, e.Tick, e.Posted)
	bump(&l.filled, e.Tick, e.Filled)
	bump(&l.unfilled, e.Tick, e.Unfilled)
	l.paid += e.Paid
	l.employerFunds += e.EmployerFunds
	l.noInput += e.NoInput
}

// sum is the sum of a series.
func sum(s []int64) int64 {
	var n int64
	for _, v := range s {
		n += v
	}
	return n
}

// totalAt is the money of the world after the step of a tick.
type totalAt struct{ tick, money int64 }

// empRun is what an employment scenario keeps of a run.
type empRun struct {
	tape   *labourTape
	run    scenarioRun
	totals []totalAt
}

// moneyOf is the money of every sector of a snapshot.
func moneyOf(s economysim.Snapshot) int64 {
	var n economysim.Money
	for _, r := range s.Regions {
		n += r.MoneyHouseholds + r.MoneyPool + r.MoneyContract + r.MoneyTreasury
	}
	return int64(n)
}

// runEmp runs cfg for ticks with the audit on and the tape as observer, taking the money
// of the world after the first step and after every 50th. It keeps no per-tick snapshot, so
// ten parallel seeds hold none of them (the lesson of S113).
func runEmp(t testing.TB, cfg economysim.Config, seed uint64, ticks int64, script []scriptStep, opts ...economysim.Option) empRun {
	t.Helper()
	return runEmpTape(t, newLabourTape(), cfg, seed, ticks, script, opts...)
}

// runEmpTape is runEmp with the tape the caller built.
func runEmpTape(t testing.TB, tape *labourTape, cfg economysim.Config, seed uint64, ticks int64, script []scriptStep, opts ...economysim.Option) empRun {
	t.Helper()
	r := empRun{tape: tape}
	r.run = runScenario(t, cfg, scenarioSpec{
		Seed: seed, Ticks: ticks, Script: script,
		Snap:    func(i int64) bool { return i == 0 || (i+1)%50 == 0 },
		OnSnap:  func(i int64, s economysim.Snapshot) { r.totals = append(r.totals, totalAt{i, moneyOf(s)}) },
		Options: append([]economysim.Option{economysim.WithObserver(r.tape), economysim.WithAudit()}, opts...),
	})
	return r
}

// moneyBreak returns the first pair of snapshots whose money moved by other than the net of
// the boundary flows of the ticks between them, or "": the wage escrow, pay and refund
// create and destroy nothing.
func (r empRun) moneyBreak() string {
	for i := 1; i < len(r.totals); i++ {
		a, b := r.totals[i-1], r.totals[i]
		var want int64
		for k := a.tick + 1; k <= b.tick; k++ {
			want += int64(r.tape.net[k])
		}
		if got := b.money - a.money; got != want {
			return fmt.Sprintf("money moved %d between the steps %d and %d, the boundary flows net %d", got, a.tick, b.tick, want)
		}
	}
	return ""
}

// escrowBreak returns what the final snapshot shows of an escrow that is not the sum of the
// accounts of the hires in progress, or "".
func (r empRun) escrowBreak() string {
	f := r.run.Final
	var held, contract economysim.Money
	for _, h := range f.Hires {
		held += h.Held
	}
	for _, g := range f.Regions {
		contract += g.MoneyContract
	}
	if f.Escrow != held || contract != held {
		return fmt.Sprintf("escrow %d, the sector contract %d, the accounts of the %d open hires %d", f.Escrow, contract, len(f.Hires), held)
	}
	return ""
}

// firstProblem is the first thing the hire checker found, or "".
func (l *labourTape) firstProblem() string {
	for _, p := range l.check.problems {
		if l.anyCause && strings.Contains(p, " Cause = ") {
			continue
		}
		return p
	}
	return ""
}

// invariants asserts, for the seed, what every employment scenario holds: the hires conform
// (the bill, the order of the events, the bill back whole), no hire has no bill, the escrow
// is the money of the open hires, and the money moves only by the boundary flows. With
// closed true every hire has ended (one-tick jobs), so none is open and no escrow is left.
func (r empRun) invariants(t testing.TB, rep *scenReport, scen, pack string, seed uint64, closed bool) {
	t.Helper()
	l := r.tape
	checkExact(t, rep, scen, pack, seed, "the hires conform to the bill, the order and the escrow identity", l.firstProblem() == "", "%s", l.firstProblem())
	checkExact(t, rep, scen, pack, seed, "no hire is formed without a bill", l.noBill == 0, "%d formations escrowed nothing", l.noBill)
	checkExact(t, rep, scen, pack, seed, "the escrow is the money of the open hires", r.escrowBreak() == "", "%s", r.escrowBreak())
	checkExact(t, rep, scen, pack, seed, "the money moves only by the boundary flows", r.moneyBreak() == "", "%s", r.moneyBreak())
	open := int64(len(r.run.Final.Hires))
	checkExact(t, rep, scen, pack, seed, "every hire formed has ended or is open", l.formed-l.ended == open, "%d formed, %d ended, %d open", l.formed, l.ended, open)
	if closed {
		checkExact(t, rep, scen, pack, seed, "no hire is open and no escrow is left", open == 0 && r.run.Final.Escrow == 0, "%d open, escrow %d", open, r.run.Final.Escrow)
	}
}

// report records what a run shows of the labour market, never asserts it: the pack collapses
// under its own rules (N-352), so a figure is a measure and not a band.
func (r empRun) report(rep *scenReport, seed uint64) {
	l := r.tape
	rep.num(seed, "households hired", sum(l.formedHH))
	rep.num(seed, "households offered", sum(l.offered))
	rep.num(seed, "households with no vacancy", sum(l.noVacancy))
	rep.num(seed, "positions posted", sum(l.posted))
	rep.num(seed, "positions unfilled", sum(l.unfilled))
	rep.num(seed, "positions closed for employer funds", l.employerFunds)
	rep.num(seed, "positions closed for no input", l.noInput)
	rep.num(seed, "wages paid", int64(l.paid))
	rep.num(seed, "tick of the last hire", l.lastHire)
	rep.num(seed, "households at end", r.run.Final.Households)
}

// hireScenario is techScenario for the employment scenarios.
func hireScenario(t *testing.T, pack, title string, fn func(t *testing.T, rep *scenReport, seed uint64)) {
	t.Helper()
	techScenario(t, pack, title, fn)
}

// hirePackWantsOff is the pack one-demand with the hire overlay and no taste row for the
// cooks, the employers: with no want in their ladder their profit has no outlet but upkeep
// (spec-employment.md 11.3.1).
func hirePackWantsOff(t testing.TB) economysim.Config {
	t.Helper()
	cfg := loadValidPack(t, packOneDemandHire...)
	cfg.Modules.Demand.Tastes = slices.DeleteFunc(slices.Clone(cfg.Modules.Demand.Tastes), func(x economysim.Taste) bool { return x.Role == "cook" })
	return cfg
}
