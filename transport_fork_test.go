package economysim_test

import (
	"bytes"
	"context"
	"errors"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// packTwoTransport is the module-alone pack of the transport module: the variant two
// with the trade overlay (the edges) and the transport overlay, which carries its own
// carrier role (spec-transport.md 17, plan 2-12 D456). Like the pack one of the
// other modules it is cold: the warm start is not layered.
var packTwoTransport = append(slices.Clone(packLayers["two"]), tradeOverlay("two"), "overlay/transport-two.toml")

// shipper is the scripted shipper of the transport goldens and fork tests: the default
// strategy, and every shipEvery ticks each household of the carrier role that holds the upkeep of its whole trip sends one unit of staple on foot to the
// first household of the other region. It keeps no state (architecture.md 7.7), so a
// restore needs nothing of it. first is the first household of each region.
type shipper struct {
	first     []economysim.AgentID
	shipEvery int64
	role      economysim.RoleID
	staple    economysim.CommodityID
}

const modeFootID economysim.ModeID = 1 // "foot" follows "coach" in the canonical order of the overlay

func (shipper) Work(v *economysim.AgentView, rng *rand.Rand) economysim.JobRequest {
	return economysim.DefaultStrategy{}.Work(v, rng)
}

func (shipper) Revise(o *economysim.OutcomeView, rng *rand.Rand) economysim.Belief {
	return economysim.DefaultStrategy{}.Revise(o, rng)
}

func (s shipper) Trade(v *economysim.AgentView, rng *rand.Rand, p *economysim.TradePlan) {
	economysim.DefaultStrategy{}.Trade(v, rng, p)
	if v.Role != s.role || v.Tick.Tick%s.shipEvery != 1 || int(v.Region) >= len(s.first) {
		return
	}
	to := economysim.RegionID(1 - int(v.Region))
	if !s.provisioned(v, to) {
		return
	}
	p.Shipments = append(p.Shipments, economysim.ShipmentRequest{
		Mode: modeFootID, To: to, Consignee: s.first[to], Lines: []economysim.Line{{Commodity: s.staple, Quantity: 1}},
	})
}

// provisioned reports whether the household holds the upkeep of its weight for the
// whole trip to region to, less the unit it ships: it cannot buy while away.
func (s shipper) provisioned(v *economysim.AgentView, to economysim.RegionID) bool {
	var ticks int64
	for _, r := range v.Routes {
		if r.To == to && r.Mode == modeFootID && r.Has {
			ticks = r.Ticks
		}
	}
	if ticks == 0 {
		return false
	}
	for _, u := range v.Upkeep {
		var stock int64
		for _, c := range v.Commodities {
			if c.Commodity == u.Commodity {
				stock = c.Stock
			}
		}
		if u.Commodity == s.staple {
			stock--
		}
		if stock < u.Quantity*v.Weight*ticks {
			return false
		}
	}
	return true
}

// shipperPack returns the module-alone configuration with its foot mode sped up to 50,
// so that a trip over an edge of 100 takes 2 ticks and a household can hold the upkeep
// of the whole trip (at the overlay's 20 it takes 5, more than any household holds),
// the roles on the shipper, and the option that registers it.
func shipperPack(t testing.TB) (economysim.Config, []economysim.Option) {
	t.Helper()
	cfg := loadValidPack(t, packTwoTransport...)
	rec := &recorder{}
	if _, err := economysim.NewWorld(cfg, economysim.WithObserver(rec)); err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	carrier := slices.IndexFunc(cfg.Roles, func(r economysim.Role) bool { return r.ID == "carrier" })
	staple := slices.IndexFunc(cfg.Commodities, func(c economysim.Commodity) bool { return c.ID == "staple" })
	if staple < 0 || carrier < 0 || cfg.Modules.Transport.Modes[modeFootID].ID != "foot" {
		t.Fatalf("pack two has no staple, no carrier or its mode %d is not foot", modeFootID)
	}
	// The consignee of a region is a household that never ships: a traveller cannot
	// move while it is the consignee of another shipment, so two carriers that
	// consigned to each other would hold each other's arrival for ever (5.7).
	first := make([]economysim.AgentID, len(cfg.Regions))
	seen := make([]bool, len(cfg.Regions))
	for _, e := range rec.events {
		if a, ok := e.(economysim.AgentSpawned); ok && !seen[a.Region] && a.Role != economysim.RoleID(carrier) {
			seen[a.Region], first[a.Region] = true, a.Agent
		}
	}
	cfg.Modules.Transport.Modes[modeFootID].Speed = 50
	for i := range cfg.Roles {
		cfg.Roles[i].Strategy = "ship"
	}
	s := shipper{first: first, shipEvery: 7, role: economysim.RoleID(carrier), staple: economysim.CommodityID(staple)}
	return cfg, []economysim.Option{economysim.WithStrategy("ship", s)}
}

// TestDeterminism_Transport verifies, for the module-alone pack with and without the
// scripted shipper, that the final state does not depend on the way the run is driven
// or watched: the cases of detCases(true), seed 1, end in the snapshot and state hash
// of a bare Step run.
//
// Requirement: FR-REF-025, FR-TRN-044
//
// Guard: shipments keep state that no other determinism test reaches; a run that
// differed by observation or chunking would show first here.
func TestDeterminism_Transport(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	for _, variant := range []string{"idle", "shipper"} {
		t.Run(variant, func(t *testing.T) {
			t.Parallel()
			cfg, opts := loadValidPack(t, packTwoTransport...), []economysim.Option(nil)
			if variant == "shipper" {
				cfg, opts = shipperPack(t)
			}
			base := runPack(t, cfg, detStep(runSpec{Seed: 1, Ticks: detTicks, Extra: opts}))
			for _, tc := range detCases(true) {
				spec := tc.spec()
				spec.Seed, spec.Ticks = 1, detTicks
				spec.Extra = append(slices.Clone(spec.Extra), opts...)
				got := runPack(t, cfg, spec)
				if !reflect.DeepEqual(got.Final, base.Final) {
					t.Errorf("%s case %q: final snapshot differs from the bare run", variant, tc.name)
				}
				if economysim.HashWorld(got.World) != economysim.HashWorld(base.World) {
					t.Errorf("%s case %q: state hash differs from the bare run", variant, tc.name)
				}
			}
		})
	}
}

// TestFork_Transport verifies the fork-at-N of the module-alone pack with the scripted
// shipper (FR-TRN-036): at ticks before the first dispatch, inside a journey (the carriers dispatch at tick 1
// and the shipments arrive at tick 3, so N 2 and 3 have a cargo in flight and a
// traveller away, N 3 on the tick of arrival) and after the arrivals, the original and two
// restores give the same events, snapshot, audit and checkpoint bytes for 20 ticks;
// the away flags and the free lists are recomputed, not saved. The cell inside a
// journey also checks that the shipment in flight at the boundary arrived after it.
//
// Requirement: FR-TRN-036, FR-CHK-050, FR-CHK-052, FR-CHK-053
func TestFork_Transport(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg, opts := shipperPack(t)
	for _, tc := range []struct {
		name     string
		n        int
		inFlight bool
	}{{"before the first step", 0, false}, {"after a dispatch tick", 2, true}, {"after the arrivals", 5, false}, {"on the tick of arrival", 3, true}, {"much later", 30, false}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var inFlight int
			forkAtN(t, forkCase{
				Name: "transport shipper", Cfg: cfg, Seed: 1, N: tc.n, M: 20, Logs: true,
				Opts: func() []economysim.Option { return opts },
				Before: func(w *economysim.World) {
					s, err := w.Snapshot()
					if err != nil {
						t.Fatalf("Snapshot() error = %v, want nil", err)
					}
					inFlight = len(s.Shipments)
					if len(s.Away) == 0 && inFlight > 0 {
						t.Errorf("tick %d: %d shipments in flight and nobody away, want the travellers", w.Tick(), inFlight)
					}
				},
				Inspect: func(t *testing.T, restored, since []economysim.Event) {
					if tc.inFlight && inFlight == 0 {
						t.Errorf("N %d: no shipment in flight at the boundary, want the cell to be inside a journey", tc.n)
					}
					if tc.inFlight && !slices.ContainsFunc(restored, func(e economysim.Event) bool { _, ok := e.(economysim.ShipmentArrived); return ok }) {
						t.Errorf("N %d: the restored world delivered no ShipmentArrived, want the shipment in flight to arrive", tc.n)
					}
				},
			})
		})
	}
}

// transportSectionTag is the tag of the transport section of the checkpoint.
const transportSectionTag = 17

// transportCheckpointBytes is a checkpoint of the shipper world with shipments in flight,
// its bytes, and the configuration and options that restore it.
func transportCheckpointBytes(t testing.TB) ([]byte, economysim.Config, []economysim.Option) {
	t.Helper()
	cfg, opts := shipperPack(t)
	w, err := economysim.NewWorld(cfg, append([]economysim.Option{economysim.WithSeed(1)}, opts...)...)
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	if _, err := w.Run(context.Background(), 2); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if s, err := w.Snapshot(); err != nil || len(s.Shipments) == 0 {
		t.Fatalf("Snapshot() = %d shipments in flight, %v, want some", len(s.Shipments), err)
	}
	c, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	b, err := c.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v, want nil", err)
	}
	return b, cfg, opts
}

// FuzzTransportSection fuzzes the decoder of tag 17: the body of the section is the
// input, the rest of a valid checkpoint with shipments in flight is kept, and the
// trailer is recomputed. No input panics; a refusal is ErrCheckpointCorrupt; and an
// accepted input is a world that passes its audit and saves the same section.
//
// Requirement: NFR-CHK-003, FR-CHK-016, FR-TRN-036
func FuzzTransportSection(f *testing.F) {
	raw, cfg, opts := transportCheckpointBytes(f)
	for _, s := range decodeFrame(f, raw).sections {
		if s.tag == transportSectionTag {
			f.Add(s.body)
		}
	}
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Fuzz(func(t *testing.T, body []byte) {
		c := reframe(t, raw, func(secs []section) []section {
			return editSection(secs, transportSectionTag, func([]byte) []byte { return slices.Clone(body) })
		})
		w, err := economysim.RestoreWorld(cfg, c, opts...)
		if err != nil {
			if !errors.Is(err, economysim.ErrCheckpointCorrupt) {
				t.Fatalf("RestoreWorld(section %x) error = %v, want ErrCheckpointCorrupt", body, err)
			}
			return
		}
		if err := w.Audit(); err != nil {
			t.Fatalf("Audit() of the world restored from section %x error = %v, want nil", body, err)
		}
		again := mustMarshal(t, mustCheckpoint(t, w))
		for _, s := range decodeFrame(t, again).sections {
			if s.tag == transportSectionTag && !bytes.Equal(s.body, body) {
				t.Fatalf("restore then save of section %x gave %x, want the same bytes", body, s.body)
			}
		}
	})
}
