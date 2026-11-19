package economysim

import (
	"bytes"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
)

// The tests of the state of the fiscal module: its section of the checkpoint, tag 30
// (spec-fiscal.md 17, FR-FIS-060, FR-FIS-061), and its audit.

// fiscalStateWorld is the world of fiscalMillConfig with a head tax per year on the home
// millers, a rate tax on the yonder millers, a remittance, a flow of the module paused
// and one at a scale of 5 000, stepped for three ticks so every flow has pairs. The
// year-long period leaves a carry that is not zero.
func fiscalStateWorld(t *testing.T) *World {
	t.Helper()
	yonder := FlowTarget{Kind: TargetHouseholds, Region: "yonder", Role: "miller"}
	m := &Fiscal{Taxes: []Tax{
		{ID: "poll", Authority: "crown", Target: homeMillers, Amount: 1_000, Per: PerYear},
		{ID: "remit", Authority: "crown", Target: FlowTarget{Kind: TargetTreasury}, RateBP: 100},
		{ID: "tithe", Authority: "shire", Target: yonder, RateBP: 50},
	}}
	w, _, _ := fiscalMill(t, m, nil)
	step(t, w, 3)
	if err := w.SetFlowScales([]FlowScale{{Flow: "tithe", BP: 0}, {Flow: "poll", BP: 5_000}}); err != nil {
		t.Fatalf("SetFlowScales error = %v, want nil", err)
	}
	return w
}

// TestCheckpoint_Fiscal verifies tag 30 over a world with the carries of every shape (a
// fixed tax per year, a rate tax and a remittance), a flow paused and one at half scale:
// the restore gives the bytes of the original, the same pairs and scales, and the two
// pay the same in the ticks that follow, with the paused flow still paused; the state is
// listed by Slots in one flow-ID order; and a restore under another fiscal configuration
// is ErrCheckpointConfig.
//
// Requirement: FR-FIS-060, FR-FIS-010
func TestCheckpoint_Fiscal(t *testing.T) {
	t.Parallel()
	w := fiscalStateWorld(t)
	cp, err := w.Checkpoint()
	if err != nil {
		t.Fatalf("Checkpoint error = %v, want nil", err)
	}
	restored, err := RestoreWorld(w.Config(), cp, WithAudit())
	if err != nil {
		t.Fatalf("RestoreWorld error = %v, want nil", err)
	}
	if again, _ := restored.Checkpoint(); !bytes.Equal(again.b, cp.b) {
		t.Errorf("checkpoint of the restored world has %d bytes and differs from the original's %d", len(again.b), len(cp.b))
	}
	wantSameFiscal(t, restored, w)
	step(t, w, 3)
	step(t, restored, 3)
	a, _ := w.Checkpoint()
	b, _ := restored.Checkpoint()
	if !bytes.Equal(a.b, b.b) {
		t.Error("checkpoints of the original and the restored world differ after three more ticks, want the same bytes")
	}
	other := w.Config()
	other.Modules.Fiscal.Taxes[0].Amount++
	if _, err := RestoreWorld(other, cp); !errors.Is(err, ErrCheckpointConfig) {
		t.Errorf("RestoreWorld with another tax = %v, want ErrCheckpointConfig", err)
	}
}

// TestCheckpoint_FiscalOffWritesNoSection verifies that a world with the module off
// writes no section 30 and one with it on does, so the bytes of a module-less world are
// those of format 1 (the pinned fixtures are tested by TestFixture_Format1).
//
// Requirement: FR-FIS-060
func TestCheckpoint_FiscalOffWritesNoSection(t *testing.T) {
	t.Parallel()
	for _, row := range checkpointSections {
		if row.tag == 30 && (row.present == nil || row.present(&Config{})) {
			t.Errorf("section 30 is present for a configuration with no fiscal module, want absent")
		}
	}
	on := fiscalStateWorld(t)
	off := newMill(t, fiscalMillConfigOff(), WithSeed(exitSeed))
	step(t, off, 3)
	a, _ := on.Checkpoint()
	b, _ := off.Checkpoint()
	if bytes.Equal(a.b, b.b) || len(a.b) <= len(b.b) {
		t.Errorf("checkpoint with the module on has %d bytes and off %d, want a longer one with the section", len(a.b), len(b.b))
	}
}

// wantSameFiscal fails the test unless the pairs and the scale of every flow of got are
// those of want, and Slots lists the scales of got in one flow-ID order.
func wantSameFiscal(t *testing.T, got, want *World) {
	t.Helper()
	for i := range want.fiscal.flows {
		g, h := &got.fiscal.flows[i], &want.fiscal.flows[i]
		if !slices.Equal(g.carry, h.carry) || g.scale != h.scale {
			t.Errorf("restored flow %q has pairs %v and scale %d, want %v and %d", h.id, g.carry, g.scale, h.carry, h.scale)
		}
	}
	slots, err := got.Slots()
	if err != nil || !slices.Equal(slots.FlowScales, []FlowScale{{Flow: "poll", BP: 5_000}, {Flow: "tithe", BP: 0}}) {
		t.Errorf("Slots().FlowScales of the restored world = %v, %v, want poll 5000 and tithe 0 in ID order", slots.FlowScales, err)
	}
}

// fiscalMillConfigOff is fiscalMillConfig with the module off.
func fiscalMillConfigOff() Config { return fiscalMillConfig(nil) }

// fiscalBody writes a section body: the pairs of each flow that keeps a carry in the
// order of fiscalStateWorld's flows (poll, remit, tithe), then the scale list.
func fiscalBody(poll, remit, tithe [][2]uint64, scales [][2]uint64) []byte {
	cw := codec.NewWriter(nil)
	for _, pairs := range [][][2]uint64{poll, remit, tithe, scales} {
		cw.Count(len(pairs))
		for _, p := range pairs {
			cw.U(p[0])
			cw.U(p[1])
		}
	}
	return cw.Out()
}

// TestFiscal_LoadRefusals verifies that the loader of tag 30 rejects, naming the fault, a
// carry on an account that is not open or of the wrong kind for its flow, pairs that are
// not in ascending account ID, a carry outside its denominator, a scale list that names a
// flow that does not exist, repeats one or holds a scale of 10 000, and trailing bytes;
// and that the section a world writes is accepted.
//
// Arithmetic: the carry of the poll flow is below 365 * 10^4 = 3 650 000; of the rate
// taxes below 10^8. Account 6 is the first home miller's, account 2 a treasury's (the
// estates are 0 and 1), account 0 an estate's.
//
// Requirement: FR-FIS-061
func TestFiscal_LoadRefusals(t *testing.T) {
	t.Parallel()
	w := fiscalStateWorld(t)
	acct := func(id AgentID) uint64 { return uint64(w.agents[id].account) }
	miller, other := acct(4), acct(5)
	treasury := uint64(w.polity.tr[0].account)
	for _, tt := range []struct {
		name string
		body []byte
		ok   bool
	}{
		{"an empty section", fiscalBody(nil, nil, nil, nil), true},
		{"a pair of an agent", fiscalBody([][2]uint64{{miller, 5}}, nil, nil, nil), true},
		{"a remittance pair on a treasury", fiscalBody(nil, [][2]uint64{{treasury, 5}}, nil, nil), true},
		{"a scale", fiscalBody(nil, nil, nil, [][2]uint64{{0, 5_000}, {2, 0}}), true},
		{"an estate account for a household flow", fiscalBody([][2]uint64{{0, 5}}, nil, nil, nil), false},
		{"an agent account for a remittance", fiscalBody(nil, [][2]uint64{{miller, 5}}, nil, nil), false},
		{"an account that does not exist", fiscalBody([][2]uint64{{1 << 40, 5}}, nil, nil, nil), false},
		{"accounts not ascending", fiscalBody([][2]uint64{{other, 1}, {miller, 2}}, nil, nil, nil), false},
		{"an account twice", fiscalBody([][2]uint64{{miller, 1}, {miller, 2}}, nil, nil, nil), false},
		{"a carry at its denominator", fiscalBody([][2]uint64{{miller, 3_650_000}}, nil, nil, nil), false},
		{"a rate carry at its denominator", fiscalBody(nil, nil, [][2]uint64{{other, 100_000_000}}, nil), false},
		{"a scale of a flow that does not exist", fiscalBody(nil, nil, nil, [][2]uint64{{3, 5_000}}), false},
		{"a scale repeated", fiscalBody(nil, nil, nil, [][2]uint64{{1, 5_000}, {1, 6_000}}), false},
		{"a scale of 10 000", fiscalBody(nil, nil, nil, [][2]uint64{{1, wholeBP}}), false},
		{"a scale above the bound", fiscalBody(nil, nil, nil, [][2]uint64{{1, MaxFlowScale + 1}}), false},
		{"trailing bytes", append(fiscalBody(nil, nil, nil, nil), 0), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			target := fiscalStateWorld(t)
			r := codec.NewReader(tt.body)
			err := target.loadFiscal(nil, r)
			if err == nil {
				err = r.Done()
			}
			if (err == nil) != tt.ok {
				t.Errorf("loadFiscal(%s) error = %v, want an error: %v", tt.name, err, !tt.ok)
			}
		})
	}
}

// TestFiscal_AuditNamesACarryOnAWrongAccount verifies that the audit passes for a world
// that has stepped and names the flow whose carry sits on an account of the wrong kind,
// is out of its order or out of its denominator.
//
// Requirement: FR-FIS-060
func TestFiscal_AuditNamesACarryOnAWrongAccount(t *testing.T) {
	t.Parallel()
	w := fiscalStateWorld(t)
	if err := w.Audit(); err != nil {
		t.Fatalf("Audit() = %v, want nil", err)
	}
	f := &w.fiscal.flows[0]
	good := slices.Clone(f.carry)
	for _, tt := range []struct {
		name   string
		break_ func()
	}{
		{"a wrong kind", func() { f.carry[0].account = 0 }},
		{"an order", func() { f.carry[0], f.carry[1] = f.carry[1], f.carry[0] }},
		{"a denominator", func() { f.carry[0].carry = f.carryDen() }},
	} {
		tt.break_()
		if err := w.Audit(); err == nil {
			t.Errorf("Audit() with %s broken = nil, want an error naming the flow", tt.name)
		}
		f.carry = slices.Clone(good)
	}
}
