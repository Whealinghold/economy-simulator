package economysim

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"testing"
)

// seqOp is one call or act of the seeded sequences.
type seqOp struct {
	name string
	run  func(r *seqRun)
}

// seqOps are the calls and acts of the seeded sequences, in the order a draw picks them.
// The ops of the first three are the calls that need no household, so index 2 is the
// one a sequence falls back to when none is left.
var seqOps = [...]seqOp{
	{"add_store", (*seqRun).addStore}, {"remove_store", (*seqRun).removeStore}, {"add_agent", (*seqRun).addAgent},
	{"remove_agent", (*seqRun).removeAgent}, {"relocate", (*seqRun).relocate}, {"reduce", (*seqRun).reduce},
	{"grow", (*seqRun).grow}, {"switch", (*seqRun).switchRole}, {"step", (*seqRun).step1}, {"step3", (*seqRun).step3},
}

// seqAddAgent is the index of add_agent in seqOps.
const seqAddAgent = 2

// seqNeedsAgent says whether an op acts on a household that exists.
func seqNeedsAgent(k int) bool {
	return seqOps[k].name != "step" && seqOps[k].name != "step3" && k != seqAddAgent
}

// seqSeeds is the number of seeds of the sequences: each is 60 calls on a world of four
// households, so a run is a few milliseconds and the same count serves under -race.
const seqSeeds = 40

// seqAllowed are the errors a call of a sequence may return: every refusal of the
// module's calls and of the calls around it. Any other error, and any panic, is a
// defect of the pledge or the room.
var seqAllowed = []error{
	ErrNoRoom, ErrOutOfBounds, ErrStoreInUse, ErrAgentAway, ErrNotLimited, ErrStoreLocked,
	ErrUnknownAgent, ErrAgentRemoved, ErrNotExternal, ErrRegionBound, ErrSameRegion, ErrOverflow,
}

// seqRun drives one seeded sequence over a world of the mill recipe with the storage
// module: the households start jobs of 3 ticks that pledge their output, and between
// the steps the sequence adds and removes stores and agents, relocates, grows and
// reduces weights and changes roles, the calls and acts that move the room and the
// pledge. After each of them the audit and the room of every live agent are checked.
type seqRun struct {
	t     *testing.T
	w     *World
	rng   *rand.Rand
	seed  uint64
	log   []string
	done  map[string]int
	roles [3]RoleID // miller, monk (no base room), tiny (a small room)
}

// stoSeqWorld is stoMillWorld with the roles monk and tiny added and the strategy
// script on the millers, so a role change can end the stores or cut the load.
func stoSeqWorld(t *testing.T) (*World, [3]RoleID) {
	t.Helper()
	var ids [3]RoleID
	w, _ := stoMillWorld(t, 60, 40, stoMiller{max: 6}, func(c *Config) {
		ids = [3]RoleID{0, RoleID(len(c.Roles)), RoleID(len(c.Roles) + 1)}
		monk, tiny := DefaultRole(), DefaultRole()
		monk.ID, monk.Trades = "monk", []string{"grain", "flour"}
		tiny.ID, tiny.Trades = "tiny", []string{"grain", "flour"}
		c.Roles = append(c.Roles, monk, tiny)
		c.Modules.Storage.Roles = []RoleRoom{{Role: "monk"}, {Role: "tiny", Base: &Capacity{Mass: 20, Bulk: 10}}}
		c.Modules.Storage.Classes = append(c.Modules.Storage.Classes, StoreClass{ID: "shed", Mass: 10, Bulk: 10})
		c.Modules.Storage = c.Modules.Storage.canonical()
	})
	return w, ids
}

// live returns the IDs of the households that exist, ascending.
func (r *seqRun) live() []AgentID {
	var out []AgentID
	for i := r.w.firstHousehold; i < len(r.w.agents); i++ {
		if r.w.agents[i].weight >= 1 {
			out = append(out, r.w.agents[i].id)
		}
	}
	return out
}

// pick returns a live household drawn from the sequence's own stream.
func (r *seqRun) pick() AgentID {
	l := r.live()
	return l[r.rng.IntN(len(l))]
}

// fail stops the sequence naming the seed and the calls so far, so one seed replays it.
func (r *seqRun) fail(format string, args ...any) {
	r.t.Helper()
	r.t.Fatalf("seed %d after %d calls %v: %s", r.seed, len(r.log), r.log, fmt.Sprintf(format, args...))
}

// call records one op and checks its error against seqAllowed.
func (r *seqRun) call(name string, err error) {
	r.t.Helper()
	r.log = append(r.log, name)
	if err == nil {
		r.done[name]++
		return
	}
	for _, ok := range seqAllowed {
		if errors.Is(err, ok) {
			r.done[name+"_refused"]++
			return
		}
	}
	r.fail("%s = %v, want nil or a refusal of the module", name, err)
}

// check is the invariant after every call: the audit passes (the load of each account
// is its lots, the books and the pledge are what the jobs and the cargo imply, the
// reserved room is 0), and no live limited agent has a load above its room.
func (r *seqRun) check() {
	r.t.Helper()
	if err := r.w.Audit(); err != nil {
		r.fail("Audit() = %v, want nil", err)
	}
	for _, id := range r.live() {
		st, err := r.w.Agent(id)
		if err != nil {
			r.fail("Agent(%d) = %v, want nil", id, err)
		}
		rm := st.Room
		if rm.Load.Mass > rm.Room.Mass || rm.Load.Bulk > rm.Room.Bulk {
			r.fail("agent %d: load %+v above room %+v", id, rm.Load, rm.Room)
		}
		if rm.Free.Mass < 0 || rm.Free.Bulk < 0 {
			r.fail("agent %d: free room %+v is negative", id, rm.Free)
		}
	}
}

// op runs the draw k; a draw that needs a household when none is left adds one.
func (r *seqRun) op(k int) {
	r.t.Helper()
	if seqNeedsAgent(k) && len(r.live()) == 0 {
		k = seqAddAgent
	}
	seqOps[k].run(r)
}

func (r *seqRun) addStore() {
	r.t.Helper()
	r.call("add_store", r.w.AddStore(r.pick(), StoreClassID(r.rng.IntN(2))))
}

func (r *seqRun) removeStore() {
	r.t.Helper()
	r.call("remove_store", r.w.RemoveStore(r.pick(), StoreClassID(r.rng.IntN(2))))
}

func (r *seqRun) addAgent() {
	r.t.Helper()
	grain := commodityOf(r.t, r.w, "grain")
	_, err := r.w.AddAgent(NewAgent{Region: RegionID(r.rng.IntN(2)), Role: 0, Strategy: "script", Money: 5,
		Stock: []StockItem{{Commodity: grain, Quantity: Quantity(r.rng.IntN(30))}}})
	r.call("add_agent", err)
}

func (r *seqRun) removeAgent() {
	r.t.Helper()
	r.call("remove_agent", r.w.RemoveAgent(r.pick()))
}

func (r *seqRun) relocate() {
	r.t.Helper()
	r.call("relocate", r.w.Relocate(r.pick(), RegionID(r.rng.IntN(2))))
}

// reduce takes one household from a household of weight 2 or more; a reduction is never
// refused for room.
func (r *seqRun) reduce() {
	r.t.Helper()
	a := &r.w.agents[r.pick()]
	if a.weight < 2 {
		return
	}
	r.log = append(r.log, "reduce")
	r.done["reduce"]++
	if a.hasJob() {
		r.done["reduce_job"]++ // a reduction that re-weighs a running job
	}
	if err := r.w.execReduce(r.w.regionOf(a), a, Decision{Agent: a.id, Households: 1, Cause: CauseStarved}); err != nil {
		r.fail("execReduce() = %v, want nil: a reduction is never refused for room", err)
	}
}

// grow adds one household to an agent, as a birth does.
func (r *seqRun) grow() {
	r.t.Helper()
	a := &r.w.agents[r.pick()]
	r.log = append(r.log, "grow")
	r.done["grow"]++
	a.weight++
	r.w.weightChangedStorage(a)
}

// switchRole moves a household to another of the three roles; a role change is never
// refused for room.
func (r *seqRun) switchRole() {
	r.t.Helper()
	a := &r.w.agents[r.pick()]
	to := r.roles[r.rng.IntN(3)]
	if a.role == to {
		return
	}
	r.log = append(r.log, "switch")
	r.done["switch"]++
	if err := r.w.execSwitch(r.w.regionOf(a), a, Decision{Agent: a.id, Role: to, Cause: CauseReview}); err != nil {
		r.fail("execSwitch() = %v, want nil: a role change is never refused for room", err)
	}
}

func (r *seqRun) steps(n int) {
	r.t.Helper()
	for range n {
		r.log = append(r.log, "step")
		r.done["step"]++
		if err := r.w.Step(); err != nil {
			r.fail("Step() = %v, want nil (WithAudit)", err)
		}
	}
}

func (r *seqRun) step1() { r.t.Helper(); r.steps(1) }
func (r *seqRun) step3() { r.t.Helper(); r.steps(3) }

// TestStorage_SeededCallSequencesKeepThePledge verifies the pledge lifecycle at the
// level of the world: over seeded sequences of 60 calls and steps (add and remove a
// store, add and remove an agent, relocate, grow and reduce a weight, change role, step
// one or three ticks) every refusal is a sentinel of the module and never a panic, the
// audit passes after every call (no pledge leaks, none is released twice, the load is
// its lots, the reserved room is 0), and no account has a load above its room or a
// negative free room. The sequences meet every way a pledge ends: a completion, an
// abandon by a relocation or a role change, a removal of the agent, a weight change.
//
// Requirement: FR-STO-005, FR-STO-013, FR-STO-017, FR-STO-018, FR-STO-020, FR-STO-032, FR-STO-033, FR-STO-040
//
// Guard: each ledger entry point of the room has its own unit test, but a pledge is
// derived from the job, the cargo and the weight together, and only a mix of calls
// (a store removed under a running job, a role change over a pledge, a relocation of a
// principal) can show a book that drifts.
func TestStorage_SeededCallSequencesKeepThePledge(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	total := map[string]int{}
	for seed := uint64(1); seed <= seqSeeds; seed++ {
		w, ids := stoSeqWorld(t)
		r := &seqRun{t: t, w: w, rng: rand.New(rand.NewPCG(seed, 77)), seed: seed, done: total, roles: ids}
		func() {
			defer func() {
				if p := recover(); p != nil {
					r.fail("panic: %v", p)
				}
			}()
			for range 60 {
				r.op(r.rng.IntN(len(seqOps)))
				r.check()
			}
		}()
	}
	for _, name := range []string{"add_store", "remove_store", "add_agent", "remove_agent", "relocate", "reduce", "grow", "switch", "step", "reduce_job"} {
		if total[name] == 0 {
			t.Errorf("no %s call of the sequences succeeded, want at least one over the seeds (the test would prove nothing for it): %v", name, total)
		}
	}
}
