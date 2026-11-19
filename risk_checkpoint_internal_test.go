package economysim

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
)

// This file tests tag 28 of the checkpoint and the loss share the shipment record of
// tag 17 carries when the risk module is on (plan 3-rsk Phase 5, spec-risk.md 15,
// FR-RSK-035, 036): the round trip, the checks of the decoder, a fuzz of the section
// and the audit of the loss share.

// riskStoreShip is riskShip with two store rows over the storage module, so that the
// section has both lists.
func riskStoreShip(rows ...RouteRisk) func(c *Config) {
	return func(c *Config) {
		riskShip(rows...)(c)
		c.Modules.Storage = &Storage{Base: &Capacity{Mass: 100_000, Bulk: 100_000}, Classes: []StoreClass{{ID: "bin", Mass: 4, Bulk: 3}, {ID: "shed", Mass: 5}}}
		c.Modules.Risk.Stores = []StoreRisk{{Region: "a_home", Class: "bin", StrikeBP: 500, SeverityBP: 4_000}, {Region: "b_far", Class: "shed", StrikeBP: 200, SeverityBP: 10_000, Common: true}}
	}
}

// riskCkWorld builds the world of the tests of this file: the rows of 18.1 and two
// store rows.
func riskCkWorld(t *testing.T) (*World, *evLog) {
	t.Helper()
	return newShipWorld(t, riskStoreShip(riskCoach("a_home", "b_far", 10_000, 2_500, false), riskCoach("b_far", "c_isle", 1_000, 6_000, true)), nil)
}

// TestCheckpoint_RiskRoundTrip verifies that a world with route scales, store scales
// and a shipment in flight that carries a loss share restores to the same state: the
// slots, the share of the shipment, the probabilities and the table of expected
// losses, the same bytes when it is saved again, and the same arrival and loss as the
// uninterrupted world (the guard of FR-RSK-033 for a checkpoint).
//
// Requirement: FR-RSK-035
func TestCheckpoint_RiskRoundTrip(t *testing.T) {
	t.Parallel()
	w, rec := riskCkWorld(t)
	haulFor(t, w, 1)
	if err := w.SetRouteRiskScales([]RouteRiskScale{{From: regFar, To: regIsle, Mode: modeCoach, BP: 30_000}}); err != nil {
		t.Fatalf("SetRouteRiskScales error = %v, want nil", err)
	}
	if err := w.SetStoreRiskScales([]StoreRiskScale{{Region: regHome, Class: 0, BP: 2_500}, {Region: regFar, Class: 1, BP: 0}}); err != nil {
		t.Fatalf("SetStoreRiskScales error = %v, want nil", err)
	}
	grain := commodityOf(t, w, "grain")
	give(t, w, shipHome, grain, 8)
	id, reason := ask(w, shipHome, 0, ShipmentRequest{Mode: modeCoach, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: grain, Quantity: 8}}})
	if reason != "" {
		t.Fatalf("dispatch = (%d, %q), want dispatched", id, reason)
	}
	step(t, w, 1)
	if w.transport.ships[0].loss != 2_500 {
		t.Fatalf("shipment %d has loss %d, want 2500 from a certain row", id, w.transport.ships[0].loss)
	}
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint error = %v, want nil", err)
	}
	r, err := RestoreWorld(riskCkConfig(), ck, WithStrategy("script", &shipScript{}), WithAudit())
	if err != nil {
		t.Fatalf("RestoreWorld error = %v, want nil", err)
	}
	requireSameRiskState(t, w, r)
	step(t, w, 3)
	step(t, r, 3)
	got := of[ShipmentArrived](rec)
	if len(got) != 1 || got[0].Lines[0].Lost != 2 {
		t.Fatalf("the uninterrupted world's arrival = %+v, want 2 of 8 lost", got)
	}
	if b := lossBoundary(t, r, grain); b != 2 {
		t.Errorf("the restored world lost %d grain at the arrival, want the same 2", b)
	}
}

// requireSameRiskState fails the test unless the restored world r has the slots, the
// loss share, the store scales, the probability of the scaled row and the bytes of w.
func requireSameRiskState(t *testing.T, w, r *World) {
	t.Helper()
	if a, b := errVal(w.Slots()), errVal(r.Slots()); !reflect.DeepEqual(a, b) {
		t.Errorf("Slots after the restore = %+v, want %+v", b, a)
	}
	if r.transport.ships[0].loss != 2_500 || !reflect.DeepEqual(r.risk.storeScales, w.risk.storeScales) {
		t.Errorf("restored loss share %d and store scales %+v, want 2500 and %+v", r.transport.ships[0].loss, r.risk.storeScales, w.risk.storeScales)
	}
	if a, b := w.risk.find(regFar, regIsle, modeCoach).p, r.risk.find(regFar, regIsle, modeCoach).p; a != b || a != 3_000 {
		t.Errorf("probability of the scaled row = %d restored, %d original, want 3000 (1 000 * 30 000)", b, a)
	}
	if a, b := r.risk.findStore(regHome, 0).p, r.risk.findStore(regFar, 1).p; a != 125 || b != 0 {
		t.Errorf("restored store probabilities = %d and %d, want 125 (500 * 2 500) and 0", a, b)
	}
	if a, b := errVal(w.Checkpoint()), errVal(r.Checkpoint()); !bytes.Equal(errVal(a.MarshalBinary()), errVal(b.MarshalBinary())) {
		t.Error("the restored world writes other bytes")
	}
}

// riskCkConfig is the configuration of riskCkWorld.
func riskCkConfig() Config {
	c := shipConfig()
	riskStoreShip(riskCoach("a_home", "b_far", 10_000, 2_500, false), riskCoach("b_far", "c_isle", 1_000, 6_000, true))(&c)
	return c
}

// riskBody builds a body of tag 28 for the tests of the checks.
func riskBody(fill func(cw *codec.Writer)) []byte {
	cw := codec.NewWriter(nil)
	fill(cw)
	return cw.Out()
}

// loadRiskBody loads a body into a fresh world and returns the error of the section
// and of the end of the body.
func loadRiskBody(t testing.TB, body []byte) error {
	t.Helper()
	w := newMill(t, riskCkConfig(), WithStrategy("script", &shipScript{}))
	r := codec.NewReader(body)
	err := w.loadRisk(&restoreCtx{tick: 10}, r)
	if err == nil {
		err = r.Done()
	}
	return err
}

// routeScale writes one route scale of a body and storeScale one store scale.
func routeScale(cw *codec.Writer, from, to, mode, bp uint64) {
	cw.U(from)
	cw.U(to)
	cw.U(mode)
	cw.U(bp)
}

func storeScale(cw *codec.Writer, region, class, bp uint64) {
	cw.U(region)
	cw.U(class)
	cw.U(bp)
}

// TestCheckpoint_RiskRefusesCorrupt verifies one failing body for each check of
// spec-risk.md 15.3: a scale of 10 000, one above MaxShockBP, keys that are not
// ascending or repeat, a key that names no row or no region, mode or class, and a
// body that is cut; a valid body loads.
//
// Requirement: FR-RSK-036
func TestCheckpoint_RiskRefusesCorrupt(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		body func(cw *codec.Writer)
		ok   bool
	}{
		{"empty", func(cw *codec.Writer) { cw.Count(0); cw.Count(0) }, true},
		{"valid", func(cw *codec.Writer) {
			cw.Count(2)
			routeScale(cw, 0, 1, 0, 5_000)
			routeScale(cw, 1, 2, 0, 0)
			cw.Count(1)
			storeScale(cw, 0, 0, 3_000)
		}, true},
		{"a route scale of 10 000", func(cw *codec.Writer) { cw.Count(1); routeScale(cw, 0, 1, 0, 10_000); cw.Count(0) }, false},
		{"a route scale above MaxShockBP", func(cw *codec.Writer) { cw.Count(1); routeScale(cw, 0, 1, 0, MaxShockBP+1); cw.Count(0) }, false},
		{"route keys descending", func(cw *codec.Writer) {
			cw.Count(2)
			routeScale(cw, 1, 2, 0, 5_000)
			routeScale(cw, 0, 1, 0, 5_000)
			cw.Count(0)
		}, false},
		{"a route key twice", func(cw *codec.Writer) {
			cw.Count(2)
			routeScale(cw, 0, 1, 0, 5_000)
			routeScale(cw, 0, 1, 0, 6_000)
			cw.Count(0)
		}, false},
		{"a key with no row", func(cw *codec.Writer) { cw.Count(1); routeScale(cw, 0, 1, 1, 5_000); cw.Count(0) }, false},
		{"a region out of range", func(cw *codec.Writer) { cw.Count(1); routeScale(cw, 0, 9, 0, 5_000); cw.Count(0) }, false},
		{"a mode out of range", func(cw *codec.Writer) { cw.Count(1); routeScale(cw, 0, 1, 9, 5_000); cw.Count(0) }, false},
		{"a store scale of 10 000", func(cw *codec.Writer) { cw.Count(0); cw.Count(1); storeScale(cw, 0, 0, 10_000) }, false},
		{"a store scale above MaxShockBP", func(cw *codec.Writer) { cw.Count(0); cw.Count(1); storeScale(cw, 0, 0, MaxShockBP+1) }, false},
		{"store keys descending", func(cw *codec.Writer) {
			cw.Count(0)
			cw.Count(2)
			storeScale(cw, 1, 1, 5_000)
			storeScale(cw, 0, 0, 5_000)
		}, false},
		{"a store key with no row", func(cw *codec.Writer) { cw.Count(0); cw.Count(1); storeScale(cw, 0, 1, 5_000) }, false},
		{"a class out of range", func(cw *codec.Writer) { cw.Count(0); cw.Count(1); storeScale(cw, 0, 5, 5_000) }, false},
		{"the store list missing", func(cw *codec.Writer) { cw.Count(0) }, false},
		{"a body with a tail", func(cw *codec.Writer) { cw.Count(0); cw.Count(0); cw.U(1) }, false},
	} {
		err := loadRiskBody(t, riskBody(tt.body))
		if (err == nil) != tt.ok {
			t.Errorf("%s: loadRisk error = %v, want success %v", tt.name, err, tt.ok)
		}
	}
}

// TestCheckpoint_RiskShipmentLossRange verifies that the loss share of a shipment
// record is read only with the module on and is refused above 10 000.
//
// Requirement: FR-RSK-036
func TestCheckpoint_RiskShipmentLossRange(t *testing.T) {
	t.Parallel()
	var off *risk
	s := shipment{}
	if off.readLoss(codec.NewReader(nil), &s); s.loss != 0 {
		t.Errorf("a nil module read a loss share %d, want it to read nothing", s.loss)
	}
	on := &risk{}
	for _, tt := range []struct {
		bp   uint64
		want bool
	}{{0, true}, {10_000, true}, {10_001, false}} {
		r := codec.NewReader(riskBody(func(cw *codec.Writer) { cw.U(tt.bp) }))
		on.readLoss(r, &s)
		if (r.Err() == nil) != tt.want {
			t.Errorf("loss share %d: error %v, want valid %v", tt.bp, r.Err(), tt.want)
		}
	}
	if w := codec.NewWriter(nil); true {
		off.saveLoss(w, &shipment{loss: 7})
		if len(w.Out()) != 0 {
			t.Errorf("a nil module wrote %x, want nothing", w.Out())
		}
	}
}

// TestAudit_RiskLossShareInRange verifies that World.Audit checks that the loss share
// of every shipment in flight is within 0 to 10 000 with the module on (spec-risk.md
// 7.5.1) and passes on a share inside.
//
// Requirement: FR-RSK-001
func TestAudit_RiskLossShareInRange(t *testing.T) {
	t.Parallel()
	w, _ := riskCkWorld(t)
	haulFor(t, w, 1)
	askCoach(t, w, regFar)
	if err := w.Audit(); err != nil {
		t.Fatalf("Audit error = %v, want nil", err)
	}
	for _, bad := range []int64{-1, 10_001} {
		w.transport.ships[0].loss = bad
		if err := w.Audit(); !errors.Is(err, ErrAuditFailed) {
			t.Errorf("Audit with a loss share of %d error = %v, want ErrAuditFailed", bad, err)
		}
	}
}

// FuzzDecodeRiskSection feeds arbitrary bytes to the decoder of tag 28: it returns an
// error or a state whose scales are in range, ascending and of configured rows, and
// never panics.
//
// Requirement: FR-RSK-036
func FuzzDecodeRiskSection(f *testing.F) {
	f.Add(riskBody(func(cw *codec.Writer) { cw.Count(0); cw.Count(0) }))
	f.Add(riskBody(func(cw *codec.Writer) {
		cw.Count(1)
		routeScale(cw, 0, 1, 0, 5_000)
		cw.Count(1)
		storeScale(cw, 0, 0, 3_000)
	}))
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01})
	f.Fuzz(func(t *testing.T, body []byte) {
		w := newMill(t, riskCkConfig(), WithStrategy("script", &shipScript{}))
		r := codec.NewReader(body)
		err := w.loadRisk(&restoreCtx{tick: 10}, r)
		if err == nil {
			err = r.Done()
		}
		if err != nil {
			return
		}
		requireDecodedScales(t, w.risk)
	})
}

// requireDecodedScales fails the test unless every decoded scale is in range and every
// route scale names a row.
func requireDecodedScales(t *testing.T, r *risk) {
	t.Helper()
	for _, s := range r.scales {
		if s.bp < 0 || s.bp > MaxShockBP || s.bp == wholeBP || r.find(s.from, s.to, s.mode) == nil {
			t.Fatalf("decoded route scale %+v is out of range or names no row", s)
		}
	}
	for _, s := range r.storeScales {
		if s.bp < 0 || s.bp > MaxShockBP || s.bp == wholeBP {
			t.Fatalf("decoded store scale %+v is out of range", s)
		}
	}
}
