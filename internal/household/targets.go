package household

import "github.com/Whealinghold/economy-sim/internal/checked"

// Need is what one commodity needs this tick, as totals for the agent
// (spec-agents.md 7.3). The caller forms Job from the planned recipe, which
// this package does not know: InputJobs * k * q for an input of the recipe plus
// k for a tool, and 0 unless the agent starts a job next tick. OutTgt is the
// output target already formed, one ceiling over the whole product (OutputTarget).
type Need struct {
	// Upkeep is U_c, the units to eat at the end of this tick (the weight times
	// the per-household quantity).
	Upkeep int64
	// Job is Job_c, the inputs and tools of the next job.
	Job int64
	// OutTgt is OutTgt_c, the output target.
	OutTgt int64
	// Want is Want_c, the units funded for a want (0 with no demand module).
	Want int64
}

// Targets is the result of ComputeTargets for one commodity.
type Targets struct {
	// Reserve is the units to be eaten at the end of this tick.
	Reserve int64
	// UseTgt is the units to hold at the start of next tick for use.
	UseTgt int64
	// Target is the units to hold after this tick's upkeep: UseTgt + Want if
	// either is above 0, else OutTgt.
	Target int64
	// Keep is the units the agent wants to hold now: Reserve + UseTgt + Want.
	Keep int64
}

// ComputeTargets forms the targets of one commodity (spec-agents.md 7.3):
// Reserve = U, UseTgt = U * CoverTicks + Job, Target = UseTgt + Want if either
// is above 0 else OutTgt, and Keep = Reserve + UseTgt + Want. A negative field of
// n counts as 0. ok is false on overflow. Every term is linear in the totals, so
// the targets of a cohort of weight w are w times those of a household of
// weight 1 for the same Job and OutTgt per household.
func ComputeTargets(n Need, p Params) (t Targets, ok bool) {
	u, job, want := max(n.Upkeep, 0), max(n.Job, 0), max(n.Want, 0)
	use, ok := checked.Mul(u, max(p.CoverTicks, 0))
	if !ok {
		return Targets{}, false
	}
	if use, ok = checked.Add(use, job); !ok {
		return Targets{}, false
	}
	wanted, ok := checked.Add(use, want)
	if !ok {
		return Targets{}, false
	}
	keep, ok := checked.Add(u, wanted)
	if !ok {
		return Targets{}, false
	}
	target := max(n.OutTgt, 0)
	if use > 0 || want > 0 {
		target = wanted
	}
	return Targets{Reserve: u, UseTgt: use, Target: target, Keep: keep}, true
}

// Tiers splits the position of one commodity into its gap, its surplus and the
// three tiers of a bid (spec-agents.md 7.3.3, 7.3.4):
//
//	gap     = max(0, keep - stock)       a bid, if above 0
//	surplus = max(0, stock - keep)       an ask, if above 0 and no bid
//	t1      = min(gap, max(0, reserve - stock))
//	t2      = min(gap - t1, max(0, jobwant - max(0, stock - reserve)))
//	t3      = gap - t1 - t2
//
// The three tiers add to the gap by construction. The function takes no price,
// so the tiers are the same at every price (FR-AGT-069). A negative argument
// counts as 0, which keeps every subtraction in range. With keep at least
// reserve, as ComputeTargets gives, the min in t1 never acts.
func Tiers(keep, stock, reserve, jobwant int64) (gap, surplus, t1, t2, t3 int64) {
	keep, stock, reserve, jobwant = max(keep, 0), max(stock, 0), max(reserve, 0), max(jobwant, 0)
	gap = max(0, keep-stock)
	surplus = max(0, stock-keep)
	t1 = min(gap, max(0, reserve-stock))
	left := max(0, stock-reserve)
	t2 = min(gap-t1, max(0, jobwant-left))
	t3 = gap - t1 - t2
	return gap, surplus, t1, t2, t3
}
