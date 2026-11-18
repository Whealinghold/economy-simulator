package economysim_test

import (
	"fmt"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// packTwoStorage is the module-alone pack of S94: the variant two with the trade and
// transport overlays and overlay/storage-two.toml (plan 3-sto D14, spec-reference-chain.md
// 7.12), cold like the transport pack.
var packTwoStorage = append(slices.Clone(packTwoTransport), "overlay/storage-two.toml")

// narrowConsignees returns cfg with the base room of the role of each region's consignee
// (the household the scripted shippers send to, the first that is not a carrier) cut to
// what it holds at the start plus one tick's upkeep plus slack units: room for at most
// slack more units of cargo, the shape of STO-S4, for the world of the seed. It returns the
// consignee of each region.
func narrowConsignees(t testing.TB, cfg economysim.Config, opts []economysim.Option, seed uint64, slack int64) (economysim.Config, []economysim.AgentID) {
	t.Helper()
	w := newPackWorld(t, cfg, seed, opts...)
	carrier := slices.IndexFunc(cfg.Roles, func(r economysim.Role) bool { return r.ID == "carrier" })
	own := *cfg.Modules.Storage
	own.Roles = slices.Clone(own.Roles)
	cfg.Modules.Storage = &own
	var first []economysim.AgentID
	for g := range cfg.Regions {
		ids, err := w.Agents(economysim.RegionID(g))
		if err != nil {
			t.Fatalf("Agents(%d) error = %v, want nil", g, err)
		}
		for _, id := range ids {
			st, err := w.Agent(id)
			if err != nil {
				t.Fatalf("Agent(%d) error = %v, want nil", id, err)
			}
			if st.Role == economysim.RoleID(carrier) {
				continue
			}
			var upkeep int64
			for _, u := range st.Upkeep {
				upkeep += u.Quantity * st.Weight
			}
			need := st.Room.Load.Mass + upkeep + slack
			per := (need + st.Weight - 1) / st.Weight
			m := cfg.Modules.Storage
			role := cfg.Roles[st.Role].ID
			if i := slices.IndexFunc(m.Roles, func(r economysim.RoleRoom) bool { return r.Role == role }); i >= 0 {
				m.Roles[i].Base = &economysim.Capacity{Mass: per, Bulk: per}
			} else {
				m.Roles = append(m.Roles, economysim.RoleRoom{Role: role, Base: &economysim.Capacity{Mass: per, Bulk: per}})
			}
			first = append(first, id)
			break
		}
	}
	return cfg, first
}

// TestScenarioS94_TwoShippersOneConsignee is STO-S4 (spec-storage.md 16): the scripted
// shippers of the transport pack on the variant two with the storage module, the base
// room of the consignees' role cut to room for a few units of cargo, seeds 1 to 10,
// 40 ticks, audit on. Exact: the audit holds every tick, the load of every role is within
// its room at every tick, and a shipment is refused consignee_full by at least one of
// the shippers while another is served (first come). Reported: the shipments
// dispatched and refused and the shippers that were refused twice (a retry).
//
// Requirement: FR-STO-014, FR-STO-015, FR-STO-040
//
// Guard: a consignee with no room that took the cargo anyway would lose goods at the
// arrival (the credit would meet ErrNoRoom), and one that refused every shipment would
// show no shipment served.
func TestScenarioS94_TwoShippersOneConsignee(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	base, opts := shipperPackOn(t, packTwoStorage)
	stoScenario(t, "two", "S94 two+transport+storage two shippers one consignee, 40 ticks", func(t *testing.T, rep *scenReport, seed uint64) {
		cfg, _ := narrowConsignees(t, base, opts, seed, 4)
		rec := &recorder{}
		w := newPackWorld(t, cfg, seed, append(slices.Clone(opts), economysim.WithAudit(), economysim.WithObserver(rec))...)
		broke := ""
		for i := range 40 {
			if err := w.Step(); err != nil {
				t.Fatalf("seed %d: Step() at tick %d error = %v, want nil", seed, i, err)
			}
			if b := roomBreak(t, w); b != "" && broke == "" {
				broke = b
			}
		}
		checkExact(t, rep, "S94", "two", seed, "the load of a role is within its room at every tick", broke == "", "%s", broke)
		var sent, refused int64
		refusedBy := map[economysim.AgentID]int{}
		for _, e := range rec.events {
			switch e := e.(type) {
			case economysim.ShipmentDispatched:
				sent++
			case economysim.ShipmentRejected:
				if e.Reason == economysim.ReasonConsigneeFull {
					refused++
					refusedBy[e.Agent]++
				}
			}
		}
		var retried int64
		for _, n := range refusedBy {
			if n > 1 {
				retried++
			}
		}
		rep.num(seed, "shipments dispatched", sent)
		rep.num(seed, "shipments refused consignee_full", refused)
		rep.num(seed, "shippers refused more than once", retried)
		checkExact(t, rep, "S94", "two", seed, "a shipment is served while another is refused for room", sent >= 1 && refused >= 1, "dispatched %d, refused %d", sent, refused)
	})
}

// stoScenario runs fn for the seeds of a storage scenario and records its block.
func stoScenario(t *testing.T, pack, title string, fn func(t *testing.T, rep *scenReport, seed uint64)) {
	t.Helper()
	techScenario(t, pack, title, fn)
}

// roomBreak returns what the snapshot of w shows of a load above a room: the first role
// row that has one, or "".
func roomBreak(t testing.TB, w *economysim.World) string {
	t.Helper()
	s, err := w.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}
	for _, r := range s.Roles {
		if r.LoadMass > r.RoomMass || r.LoadBulk > r.RoomBulk {
			return fmt.Sprintf("tick %d row %+v", s.Tick, r)
		}
	}
	return ""
}
