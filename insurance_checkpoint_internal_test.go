package economysim

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
)

// This file tests tag 29 of the checkpoint, the audit and the snapshot of the insurance module
// (plan 3-ins Phase 7, spec-insurance.md 9.4, 15.4, 16): the round trip of open route policies
// and a payout claim, the checks of the decoder, a fuzz of the section, the audit of the
// identities, the snapshot rows and the group of AgentState.
//
// The world is settleWorld's: a coach shipment of 8 grain for insHome at a sum of 8 000, cover
// share 8 000 bp, so a maximum payout of 6 400 and a premium of 1 600.

// insCkConfig is insConfig with a payout claim due three ticks after its formation, so that
// the claim of the tests is still open when the checkpoint is taken.
func insCkConfig() Config {
	c := insConfig(defaultInsure(), defaultUnderwrite())
	c.Modules.Insurance.PayoutTicks = 3
	return c
}

// insCkWorld returns a world at the boundary after tick 3 with one payout claim open (policy 0
// ended with a shortfall of 1 100 at tick 2, due at tick 5) and one open route policy (policy
// 1, formed before tick 3 for insHome2's shipment, which is in flight).
func insCkWorld(t *testing.T) (*World, *evLog) {
	t.Helper()
	w, rec := settleWorld(t, insCkConfig(), 8)
	step(t, w, 2)
	moneyOf(t, w, insBacker, 1_500)
	step(t, w, 1)
	moneyOf(t, w, insBacker, 10_000)
	insDispatch(t, w, insHome2, 8)
	planCover(w, insHome2, coachLane)
	planOffer(w, insBacker, bigOffer)
	w.p33cCoverRound()
	if got := len(w.insurance.open); got != 1 || w.insurance.open[0].id != 1 {
		t.Fatalf("open policies = %+v, want policy 1 alone", w.insurance.open)
	}
	step(t, w, 1)
	return w, rec
}

// restoreIns restores a checkpoint of w into a new world with the strategy of the tests.
func restoreIns(t *testing.T, w *World, opts ...Option) *World {
	t.Helper()
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint error = %v, want nil", err)
	}
	opts = append([]Option{WithStrategy("script", &insScript{}), WithAudit()}, opts...)
	r, err := RestoreWorld(insCkConfig(), ck, opts...)
	if err != nil {
		t.Fatalf("RestoreWorld error = %v, want nil", err)
	}
	return r
}

// TestCheckpoint_InsuranceRoundTrip verifies that a world with an open route policy and an open
// payout claim restores to the same state: the policy list, the six totals, the indexes, the
// claim with its origin, the snapshot and the same bytes when saved again, and that the
// restored world settles the policy at the arrival with the figures and the events of the
// uninterrupted one (FR-INS-087).
//
// Requirement: FR-INS-080, FR-INS-087, FR-INS-006
func TestCheckpoint_InsuranceRoundTrip(t *testing.T) {
	t.Parallel()
	w, rec := insCkWorld(t)
	r := restoreIns(t, w, WithObserver(&evLog{}))
	requireSameInsurance(t, w, r)
	moneyOf(t, w, insBacker, 5_000)
	moneyOf(t, r, insBacker, 5_000)
	step(t, w, 3)
	step(t, r, 3)
	if !reflect.DeepEqual(w.insurance.tot, r.insurance.tot) || len(r.insurance.open) != 0 {
		t.Errorf("after the arrival the restored totals %+v and %d open, want %+v and none", r.insurance.tot, len(r.insurance.open), w.insurance.tot)
	}
	if got := of[CoverEnded](rec); len(got) != 2 || got[1].Policy != 1 {
		t.Errorf("the uninterrupted world ended %+v, want policies 0 and 1", got)
	}
}

// requireSameInsurance fails the test unless the restored world r has the policies, the totals,
// the indexes, the payout claim, the snapshot rows and the bytes of w.
func requireSameInsurance(t *testing.T, w, r *World) {
	t.Helper()
	if !reflect.DeepEqual(w.insurance.open, r.insurance.open) || w.insurance.tot != r.insurance.tot || w.insurance.next != r.insurance.next {
		t.Errorf("restored policies %+v totals %+v next %d, want %+v %+v %d", r.insurance.open, r.insurance.tot, r.insurance.next, w.insurance.open, w.insurance.tot, w.insurance.next)
	}
	if !reflect.DeepEqual(w.insurance.liability, r.insurance.liability) || !r.insurance.covered(w.insurance.open[0].shipment) {
		t.Errorf("restored liability %v, want %v and the shipment covered", r.insurance.liability, w.insurance.liability)
	}
	if c := openClaim(t, r, 0); c.Origin != openClaim(t, w, 0).Origin || c.PrincipalOwed != 1_100 {
		t.Errorf("restored claim = %+v, want the payout claim of 1100", c)
	}
	requireSameBytesAndRows(t, w, r)
}

// requireSameBytesAndRows fails the test unless r writes the bytes of w and has its snapshot rows.
func requireSameBytesAndRows(t *testing.T, w, r *World) {
	t.Helper()
	if a, b := errVal(w.Checkpoint()), errVal(r.Checkpoint()); !bytes.Equal(errVal(a.MarshalBinary()), errVal(b.MarshalBinary())) {
		t.Error("the restored world writes other bytes")
	}
	if a, b := errVal(w.Snapshot()), errVal(r.Snapshot()); !reflect.DeepEqual(a.Policies, b.Policies) || a.Insurance != b.Insurance {
		t.Errorf("restored snapshot policies %+v totals %+v, want %+v %+v", b.Policies, b.Insurance, a.Policies, a.Insurance)
	}
}

// TestCheckpoint_InsuranceOffIsAbsent verifies that a world with the module off writes no
// section 29 and that the shipment record of tag 17 is the same bytes with the module on and
// off for the same shipment: the module adds nothing to it (spec-insurance.md 16.1, FR-INS-080).
//
// Requirement: FR-INS-080
func TestCheckpoint_InsuranceOffIsAbsent(t *testing.T) {
	t.Parallel()
	off := insConfig(nil, nil)
	off.Modules.Insurance = nil
	on, _ := newInsWorld(t, insConfig(nil, nil), nil)
	no, _ := newInsWorld(t, off, nil)
	insDispatch(t, on, insHome, 8)
	insDispatch(t, no, insHome, 8)
	var a, b codec.Writer
	if err := on.saveTransport(&a); err != nil {
		t.Fatalf("saveTransport error = %v, want nil", err)
	}
	if err := no.saveTransport(&b); err != nil {
		t.Fatalf("saveTransport error = %v, want nil", err)
	}
	if !bytes.Equal(a.Out(), b.Out()) || len(a.Out()) == 0 {
		t.Errorf("shipment record with the module on = %x, off = %x, want the same non-empty bytes", a.Out(), b.Out())
	}
	for _, s := range checkpointSections {
		if s.tag == 29 && s.present(&off) {
			t.Error("section 29 is present with the module off")
		}
	}
}

// insRec is one policy of a body of tag 29.
type insRec struct{ id, insured, underwriter, shipment, sum, maxPayout, coverBP, premium, formed uint64 }

// goodRec is the open policy of insCkWorld's first world: policy 0 for insHome's shipment 0.
var goodRec = insRec{0, uint64(insHome), uint64(insBacker), 0, 8_000, 6_400, 8_000, 1_600, 0}

// insBody builds a body of tag 29.
func insBody(next uint64, recs []insRec, totals [6]uint64) []byte {
	cw := codec.NewWriter(nil)
	cw.U(next)
	cw.Count(len(recs))
	for _, r := range recs {
		for _, v := range [...]uint64{r.id, r.insured, r.underwriter, r.shipment, r.sum, r.maxPayout, r.coverBP, r.premium, r.formed} {
			cw.U(v)
		}
	}
	for _, v := range totals {
		cw.U(v)
	}
	return cw.Out()
}

// loadInsBody loads a body into the insurance state of w, which a fresh state replaces, with
// maxPolicies as the limit, and returns the error of the section and of the end of the body.
func loadInsBody(w *World, body []byte, maxPolicies int64) error {
	cfg := *w.insurance.cfg
	cfg.MaxPolicies = maxPolicies
	w.insurance = newInsurance(&w.cfg)
	w.insurance.cfg = &cfg
	r := codec.NewReader(body)
	err := w.loadInsurance(&restoreCtx{tick: 3}, r)
	if err == nil {
		err = r.Done()
	}
	return err
}

// hugeCount is a body of next ID 0 that claims 2^60 policies and holds none.
func hugeCount() []byte {
	cw := codec.NewWriter(nil)
	cw.U(0)
	cw.U(1 << 60)
	return cw.Out()
}

// with returns r with the field set by f.
func (r insRec) with(f func(*insRec)) insRec { f(&r); return r }

// TestCheckpoint_InsuranceRefusesCorrupt verifies one failing body for each check of
// spec-insurance.md 16.5: a policy whose accounts are equal or are no live agent's, whose
// shipment is not in flight or has another shipper, whose maximum payout is not
// halfEven(CoverBP * Sum / 10 000), IDs that are not ascending or not below the next ID, a
// shipment covered twice, a count above MaxPolicies, a field out of its range, totals that are
// not the next ID or are below the open policies, and a body that is cut or has a tail; a valid
// body loads.
//
// Requirement: FR-INS-082
func TestCheckpoint_InsuranceRefusesCorrupt(t *testing.T) {
	t.Parallel()
	w, _ := settleWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), 8, 8)
	good, second := []insRec{goodRec}, insRec{1, uint64(insHome2), uint64(insBacker), 1, 8_000, 6_400, 8_000, 1_600, 0}
	totals := [6]uint64{2, 3_200, 0, 0, 0, 0}
	for _, tt := range []struct {
		name string
		body []byte
		max  int64
		ok   bool
	}{
		{"valid", insBody(2, []insRec{goodRec, second}, totals), 50, true},
		{"empty", insBody(0, nil, [6]uint64{}), 50, true},
		{"the same account twice", insBody(2, []insRec{goodRec.with(func(r *insRec) { r.underwriter = r.insured })}, totals), 50, false},
		{"an agent never issued", insBody(2, []insRec{goodRec.with(func(r *insRec) { r.underwriter = 99 })}, totals), 50, false},
		{"an estate as the underwriter", insBody(2, []insRec{goodRec.with(func(r *insRec) { r.underwriter = 0 })}, totals), 50, false},
		{"a shipment never issued", insBody(2, []insRec{goodRec.with(func(r *insRec) { r.shipment = 9 })}, totals), 50, false},
		{"another shipper", insBody(2, []insRec{goodRec.with(func(r *insRec) { r.insured = uint64(insHome2) })}, totals), 50, false},
		{"a wrong maximum payout", insBody(2, []insRec{goodRec.with(func(r *insRec) { r.maxPayout = 6_401 })}, totals), 50, false},
		{"a cover share of 0", insBody(2, []insRec{goodRec.with(func(r *insRec) { r.coverBP, r.maxPayout = 0, 0 })}, totals), 50, false},
		{"a cover share above 10 000", insBody(2, []insRec{goodRec.with(func(r *insRec) { r.coverBP = 10_001 })}, totals), 50, false},
		{"a sum of 0", insBody(2, []insRec{goodRec.with(func(r *insRec) { r.sum, r.maxPayout = 0, 0 })}, totals), 50, false},
		{"a sum above MaxMoney", insBody(2, []insRec{goodRec.with(func(r *insRec) { r.sum = MaxMoney + 1 })}, totals), 50, false},
		{"formed after the tick", insBody(2, []insRec{goodRec.with(func(r *insRec) { r.formed = 4 })}, totals), 50, false},
		{"IDs descending", insBody(2, []insRec{second, goodRec}, totals), 50, false},
		{"an ID twice", insBody(2, []insRec{goodRec, goodRec}, totals), 50, false},
		{"an ID not below the next", insBody(1, []insRec{goodRec.with(func(r *insRec) { r.id = 1 })}, [6]uint64{1, 1_600}), 50, false},
		{"a shipment covered twice", insBody(2, []insRec{goodRec, second.with(func(r *insRec) { r.shipment = 0; r.insured = uint64(insHome) })}, totals), 50, false},
		{"more policies than MaxPolicies", insBody(2, []insRec{goodRec, second}, totals), 1, false},
		{"formed is not the next ID", insBody(2, good, [6]uint64{1, 1_600}), 50, false},
		{"premiums below the open policies", insBody(2, []insRec{goodRec, second}, [6]uint64{2, 1_600}), 50, false},
		{"a total above MaxMoney", insBody(2, good, [6]uint64{2, 3_200, MaxMoney + 1}), 50, false},
		{"liability of one underwriter above MaxMoney", insBody(2, []insRec{
			goodRec.with(func(r *insRec) { r.sum, r.maxPayout, r.coverBP = MaxMoney, MaxMoney, 10_000; r.premium = 0 }),
			second.with(func(r *insRec) { r.sum, r.maxPayout, r.coverBP = MaxMoney, MaxMoney, 10_000; r.premium = 0 })}, [6]uint64{2}), 50, false},
		{"a count of 2^60", hugeCount(), 50, false},
		{"a body with a tail", append(insBody(0, nil, [6]uint64{}), 1), 50, false},
		{"a body cut short", insBody(0, nil, [6]uint64{})[:4], 50, false},
	} {
		if err := loadInsBody(w, tt.body, tt.max); (err == nil) != tt.ok {
			t.Errorf("%s: loadInsurance error = %v, want success %v", tt.name, err, tt.ok)
		}
	}
}

// TestCheckpoint_InsuranceRefusesCorruptRestore verifies the audit that follows a restore
// refuses a checkpoint whose due is not paid + the ledger's Awarded + dropped + cut: the
// identity of 9.4 reads the ledger's section, which the decoder of the module does not.
//
// Requirement: FR-INS-082, FR-INS-081
func TestCheckpoint_InsuranceRefusesCorruptRestore(t *testing.T) {
	t.Parallel()
	w, _ := insCkWorld(t)
	w.insurance.tot.due += 7
	ck, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint error = %v, want nil", err)
	}
	_, err = RestoreWorld(insCkConfig(), ck, WithStrategy("script", &insScript{}))
	if !errors.Is(err, ErrCheckpointCorrupt) {
		t.Errorf("RestoreWorld of a checkpoint with a due of 7 over its parts error = %v, want ErrCheckpointCorrupt", err)
	}
}

// FuzzDecodeInsuranceSection feeds arbitrary bytes to the decoder of tag 29: it returns an
// error or a state whose policies are sound, ascending and at most MaxPolicies, and never
// panics.
//
// Requirement: FR-INS-082
func FuzzDecodeInsuranceSection(f *testing.F) {
	f.Add(insBody(0, nil, [6]uint64{}))
	f.Add(insBody(1, []insRec{goodRec}, [6]uint64{1, 1_600}))
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01})
	f.Fuzz(func(t *testing.T, body []byte) {
		w, _ := settleWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), 8)
		if err := loadInsBody(w, body, 50); err != nil {
			return
		}
		in := w.insurance
		if err := errors.Join(in.auditPolicies(w), in.auditIndexes(), in.checkOpenTotals()); err != nil || int64(len(in.open)) > 50 || in.tot.formed != in.next {
			t.Fatalf("decoded state is unsound: %v (open %d, formed %d, next %d)", err, len(in.open), in.tot.formed, in.next)
		}
	})
}

// TestAudit_InsuranceIdentities verifies that World.Audit checks the identities of
// spec-insurance.md 9.4 and passes on a sound world: the due is paid + the ledger's Awarded +
// dropped + cut, the policies formed are the next ID, the premiums cover the open policies, an
// open policy has a shipment in flight, a live insured and a live underwriter that differ, and
// the indexes are the lists'.
//
// Requirement: FR-INS-081, FR-INS-006
func TestAudit_InsuranceIdentities(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		bend func(w *World)
	}{
		{"the due is more than its parts", func(w *World) { w.insurance.tot.due++ }},
		{"the due is less than its parts", func(w *World) { w.insurance.tot.paid++ }},
		{"the policies formed are not the next ID", func(w *World) { w.insurance.next++ }},
		{"the premiums are below the open policies", func(w *World) { w.insurance.tot.premiums = 0 }},
		{"the shipment left", func(w *World) { w.transport.ships = w.transport.ships[:0] }},
		{"the two parties are one", func(w *World) { w.insurance.open[0].underwriter = w.insurance.open[0].insured }},
		{"the maximum payout is not the share of the sum", func(w *World) { w.insurance.open[0].maxPayout++ }},
		{"a liability is off", func(w *World) { w.insurance.liability[insBacker]++ }},
		{"the insured count is off", func(w *World) { w.insurance.insured[insHome2]++ }},
		{"the by-shipment index lost the policy", func(w *World) { clear(w.insurance.byShipment) }},
	} {
		w, _ := insCkWorld(t)
		auditNow(t, w)
		tt.bend(w)
		if err := w.Audit(); !errors.Is(err, ErrAuditFailed) {
			t.Errorf("%s: Audit error = %v, want ErrAuditFailed", tt.name, err)
		}
	}
}

// TestSnapshot_PoliciesAndTotals verifies the rows of spec-insurance.md 15.4: Policies has a
// row for each open route policy in ascending ID with the fields of the record and the region
// of the insured, Insurance has the six totals and the shortfall read from the ledger's
// Awarded, the rows are deep copies, and both are empty with the module off.
//
// Requirement: FR-INS-076
func TestSnapshot_PoliciesAndTotals(t *testing.T) {
	t.Parallel()
	w, _ := insCkWorld(t)
	s := errVal(w.Snapshot())
	want := []CoverPolicyRow{{Policy: 1, Insured: insHome2, Underwriter: insBacker, Region: regHome, Shipment: 1,
		Sum: 8_000, MaxPayout: 6_400, CoverBP: 8_000, Premium: 1_600, Formed: 3}}
	if !reflect.DeepEqual(s.Policies, want) {
		t.Errorf("Policies = %+v, want %+v", s.Policies, want)
	}
	wantTotals := InsuranceTotals{Formed: 2, Premiums: 3_200, Due: 1_600, Paid: 500, Short: 1_100}
	if s.Insurance != wantTotals || s.Insurance.Short != s.Credit.Awarded {
		t.Errorf("Insurance = %+v, want %+v with Short = Credit.Awarded %d", s.Insurance, wantTotals, s.Credit.Awarded)
	}
	s.Policies[0].Sum = 1
	if w.insurance.open[0].sum != 8_000 {
		t.Error("changing a row changed the state: the rows are not deep copies")
	}
	off := insConfig(nil, nil)
	off.Modules.Insurance = nil
	o, _ := newInsWorld(t, off, nil)
	if os := errVal(o.Snapshot()); len(os.Policies) != 0 || os.Insurance != (InsuranceTotals{}) {
		t.Errorf("snapshot with the module off has policies %+v and totals %+v, want none", os.Policies, os.Insurance)
	}
	if !contains(s.Modules, "insurance") {
		t.Errorf("Modules = %v, want insurance in it", s.Modules)
	}
}

// contains reports whether the list has the name.
func contains(list []string, name string) bool {
	for _, v := range list {
		if v == name {
			return true
		}
	}
	return false
}

// TestWorldRestored_SeedsLiabilityAndClaims verifies that the snapshot an observer is given at a
// restore carries the open policies, whose maximum payouts sum to the open liability of the
// underwriter, and the payout claim with its origin, so an observer can seed
// economysim.insurance.liability and its claims from it (FR-INS-077).
//
// Requirement: FR-INS-077
func TestWorldRestored_SeedsLiabilityAndClaims(t *testing.T) {
	t.Parallel()
	w, _ := insCkWorld(t)
	rec := &evLog{}
	restoreIns(t, w, WithObserver(rec))
	got := only(t, "WorldRestored", of[WorldRestored](rec)).Snapshot
	var liability Money
	for _, p := range got.Policies {
		if p.Underwriter == insBacker {
			liability += p.MaxPayout
		}
	}
	if liability != Money(w.insurance.liabilityOf(insBacker)) || liability != 6_400 {
		t.Errorf("liability of the underwriter from the rows = %d, want 6400 (the state's %d)", liability, w.insurance.liabilityOf(insBacker))
	}
	if len(got.Claims) != 1 || got.Claims[0].Origin != OriginPayout || got.Claims[0].PrincipalOwed != 1_100 || got.Insurance.Short != 1_100 {
		t.Errorf("claims = %+v, short = %d, want one payout claim of 1100", got.Claims, got.Insurance.Short)
	}
}

// TestAgentState_CoverGroup verifies the insurance group of AgentState (spec-insurance.md
// 16.3.1; the field is Insurance, since Cover is the own-use group): the insured holds one open
// policy and no liability, the underwriter one policy and a liability of the maximum payout, an
// agent with none has the zero value, and the group is the zero value with the module off.
//
// Requirement: FR-INS-080
func TestAgentState_CoverGroup(t *testing.T) {
	t.Parallel()
	w, _ := insCkWorld(t)
	for id, want := range map[AgentID]CoverState{insHome2: {Policies: 1}, insBacker: {Policies: 1, Open: 6_400}, insHome: {}, insFar: {}} {
		if got := errVal(w.Agent(id)).Insurance; got != want {
			t.Errorf("agent %d Insurance = %+v, want %+v", id, got, want)
		}
	}
	off := insConfig(nil, nil)
	off.Modules.Insurance = nil
	o, _ := newInsWorld(t, off, nil)
	if got := errVal(o.Agent(insBacker)).Insurance; got != (CoverState{}) {
		t.Errorf("Insurance with the module off = %+v, want the zero value", got)
	}
}
