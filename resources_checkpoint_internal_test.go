package economysim

import (
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
)

// scarceMill is oreMill of one-tick jobs with a constant source of 8 + 1/365 units a
// tick: the stock stays below what ten agents ask, so units are short every tick
// (the run of ticks grows), and the odd 1/365 leaves a carry between whole units.
func scarceMill(stock int64) Config {
	c := oreMill(1, 10, stock)
	c.Modules.Resources.Resources[0].RatePerYear = 8*365 + 1
	c.Roles[0].StartStock = []Item{{Commodity: "grain", Quantity: 100}}
	return c
}

// TestCheckpoint_ResourcesRoundTrip verifies that section 15 saves and restores the
// stocks, the carries, the runs of ticks short and the shock scales, that a stock
// at 0 whose capacity is above 0 is restored as 0 and not as the starting stock,
// and that the identity of the next tick holds from the restored base (the world is
// audited).
//
// Requirement: FR-RES-042
func TestCheckpoint_ResourcesRoundTrip(t *testing.T) {
	t.Parallel()
	c := scarceMill(20)
	w := newMill(t, c, WithSeed(3), WithAudit())
	step(t, w, 7)
	w.setRegenScales(0, 0, 2_500)
	w.setCapacityScales(0, 0, 12_000)
	r := w.res
	if r.carry[0] == 0 || r.stock[0] == 0 || r.exhaustedRun[0] == 0 {
		t.Fatalf("carry, stock, run = %d, %d, %d, want all non-zero so the round trip carries something", r.carry[0], r.stock[0], r.exhaustedRun[0])
	}
	got := restoreWith(t, w, c)
	g := got.res
	if !slices.Equal(g.stock, r.stock) || !slices.Equal(g.carry, r.carry) || !slices.Equal(g.exhaustedRun, r.exhaustedRun) ||
		!slices.Equal(g.capScale, r.capScale) || !slices.Equal(g.regenScale, r.regenScale) || !slices.Equal(g.base, r.stock) {
		t.Errorf("restored state = stock %v carry %v run %v scales %v %v base %v, want stock %v carry %v run %v scales %v %v and base = stock",
			g.stock, g.carry, g.exhaustedRun, g.capScale, g.regenScale, g.base, r.stock, r.carry, r.exhaustedRun, r.capScale, r.regenScale)
	}
	step(t, got, 3)
	// A depleted stock is saved as 0, not restored as the starting stock.
	w.adjustEndowments(0, 0, -MaxEndowment)
	if empty := restoreWith(t, w, c); empty.Endowment(0, 0) != 0 {
		t.Errorf("Endowment after restoring a depleted stock = %d, want 0 (not the starting stock 20)", empty.Endowment(0, 0))
	}
}

// restoreWith checkpoints w and restores the checkpoint with c.
func restoreWith(t *testing.T, w *World, c Config) *World {
	t.Helper()
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	got, err := RestoreWorld(c, ck, WithAudit())
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	return got
}

// TestCheckpoint_SectionAbsentWhenOff verifies that tag 15 is written when the
// resources module is on and only then, so a configuration without the module has the
// checkpoint it had before the module existed.
//
// Requirement: FR-RES-042
func TestCheckpoint_SectionAbsentWhenOff(t *testing.T) {
	t.Parallel()
	var row *section
	for i := range checkpointSections {
		if checkpointSections[i].tag == 15 {
			row = &checkpointSections[i]
		}
	}
	off, on := millConfig(3, 4), oreMill(3, 4, 20)
	if row == nil || row.present == nil || row.present(&off) || !row.present(&on) {
		t.Fatalf("section 15 = %+v, want a row present exactly when Modules.Resources is set", row)
	}
	a, b := newMill(t, off), newMill(t, on)
	ca, cb := saveSections(t, a), saveSections(t, b)
	if slices.Contains(ca, 15) || !slices.Contains(cb, 15) {
		t.Errorf("tags written: off %v, on %v, want 15 only with the module on", ca, cb)
	}
}

// saveSections returns the tags a checkpoint of w writes, in order.
func saveSections(t *testing.T, w *World) []uint64 {
	t.Helper()
	var tags []uint64
	for i := range checkpointSections {
		if s := &checkpointSections[i]; s.present == nil || s.present(&w.cfg) {
			tags = append(tags, s.tag)
		}
	}
	return tags
}

// FuzzDecodeResources verifies that corrupt bytes of tag 15 give an error and never
// a panic, and that bytes that decode leave every stock inside its effective
// capacity once the cross-check has run.
//
// Requirement: FR-RES-042, NFR-RES-003
func FuzzDecodeResources(f *testing.F) {
	w, err := NewWorld(scarceMill(20), WithSeed(3))
	if err != nil {
		f.Fatal(err)
	}
	for range 5 {
		if err := w.Step(); err != nil {
			f.Fatal(err)
		}
	}
	w.setRegenScales(0, 0, 2_500)
	for i := range checkpointSections {
		if checkpointSections[i].tag == 15 {
			cw := codec.NewWriter(nil)
			if err := checkpointSections[i].save(w, cw); err != nil {
				f.Fatal(err)
			}
			f.Add(cw.Out())
			f.Add(cw.Out()[:len(cw.Out())/2])
		}
	}
	f.Add([]byte{})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01})
	f.Fuzz(func(t *testing.T, body []byte) {
		fresh, err := NewWorld(scarceMill(20))
		if err != nil {
			t.Fatal(err)
		}
		r := codec.NewReader(body)
		err = fresh.loadResources(nil, r)
		if err == nil {
			err = r.Done()
		}
		if err == nil {
			err = fresh.resourcesCross()
		}
		if err != nil {
			return
		}
		for cell, st := range fresh.res.stock {
			if st < 0 || st > fresh.res.effectiveCapacity(cell) {
				t.Fatalf("decoded stock %d outside [0, %d]", st, fresh.res.effectiveCapacity(cell))
			}
		}
	})
}

// TestCheckpoint_ResourcesRejects verifies that a body of tag 15 that breaks a rule
// is refused: a stock above the effective capacity, a run above the ticks elapsed, a
// scale above MaxShockBP, a row naming an endowment that does not exist, rows out of
// order or repeated, and a row at the default scales, which is never saved.
//
// Requirement: FR-RES-042
func TestCheckpoint_ResourcesRejects(t *testing.T) {
	t.Parallel()
	// One endowment (capacity 1 000): the body is its stock, carry and run, then the rows.
	body := func(stock, carry, run uint64, rows ...[4]uint64) []byte {
		cw := codec.NewWriter(nil)
		cw.U(stock)
		cw.U(carry)
		cw.U(run)
		cw.Count(len(rows))
		for _, r := range rows {
			for _, v := range r {
				cw.U(v)
			}
		}
		return cw.Out()
	}
	for _, tt := range []struct {
		name string
		body []byte
	}{
		{"stock above the capacity", body(1_001, 0, 0)},
		{"stock above the capacity under the scale", body(600, 0, 0, [4]uint64{0, 0, 5_000, 10_000})},
		{"run above the ticks", body(10, 0, 1)},
		{"scale above the bound", body(10, 0, 0, [4]uint64{0, 0, MaxShockBP + 1, 10_000})},
		{"unknown region", body(10, 0, 0, [4]uint64{1, 0, 5_000, 10_000})},
		{"unknown resource", body(10, 0, 0, [4]uint64{0, 1, 5_000, 10_000})},
		{"repeated row", body(10, 0, 0, [4]uint64{0, 0, 5_000, 10_000}, [4]uint64{0, 0, 6_000, 10_000})},
		{"row at the default", body(10, 0, 0, [4]uint64{0, 0, 10_000, 10_000})},
		{"carry at D", body(10, 1_000_000_000_000*365, 0)},
		{"trailing byte", append(body(10, 0, 0), 0)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := newMill(t, oreMill(3, 4, 20))
			r := codec.NewReader(tt.body)
			err := w.loadResources(nil, r)
			if err == nil {
				err = r.Done()
			}
			if err == nil {
				err = w.resourcesCross()
			}
			if err == nil {
				t.Errorf("loadResources(%q) = nil, want an error", tt.name)
			}
		})
	}
	w := newMill(t, oreMill(3, 4, 20))
	if r := codec.NewReader(body(10, 5, 0, [4]uint64{0, 0, 5_000, 10_000})); w.loadResources(nil, r) != nil || r.Done() != nil || w.resourcesCross() != nil {
		t.Errorf("a valid body was refused")
	}
}
