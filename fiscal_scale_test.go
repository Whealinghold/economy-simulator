package economysim_test

import (
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// fiscalAndMoneyConfig is fiscalConfig with one money flow of the money supply module, so
// the flow scale has two owners in one namespace of IDs.
func fiscalAndMoneyConfig() economysim.Config {
	c := fiscalConfig()
	c.Modules.MoneySupply = &economysim.MoneySupply{Flows: []economysim.MoneyFlow{{
		ID: "stipend", Kind: economysim.FlowGrant, Amount: 1,
		Target: economysim.FlowTarget{Kind: economysim.TargetHouseholds, Region: "north", Role: "cook"},
	}}}
	return c
}

// TestSetFlowScales_Fiscal verifies the flow scale of the fiscal module through the
// integration call: a fiscal flow's ID is accepted beside a money flow's, in one call, the
// events MoneyFlowScaled come in one flow-ID order whichever module owns the flow, a scale
// equal to the one in force is silent, Slots lists the non-default scales of both owners in
// that order, and the errors are those of the money flows (ErrUnknownFlow, ErrOutOfBounds,
// ErrDuplicateID) with nothing changed by a refused call.
//
// Requirement: FR-FIS-010, FR-FIS-043
func TestSetFlowScales_Fiscal(t *testing.T) {
	t.Parallel()
	rec := &recorder{}
	w := mustWorld(t, fiscalAndMoneyConfig(), economysim.WithObserver(rec))
	rec.forget()
	changes := []economysim.FlowScale{{Flow: "stipend", BP: 5_000}, {Flow: "poll", BP: 0}, {Flow: "alms", BP: 7_000}, {Flow: "tithe", BP: 10_000}}
	if err := w.SetFlowScales(changes); err != nil {
		t.Fatalf("SetFlowScales(%v) error = %v, want nil", changes, err)
	}
	var got []string
	for _, e := range rec.events {
		if s, ok := e.(economysim.MoneyFlowScaled); ok {
			got = append(got, s.Flow)
		}
	}
	if want := []string{"alms", "poll", "stipend"}; !slices.Equal(got, want) {
		t.Errorf("MoneyFlowScaled flows = %v, want %v: one flow-ID order, and tithe is silent at 10000", got, want)
	}
	slots, err := w.Slots()
	want := []economysim.FlowScale{{Flow: "alms", BP: 7_000}, {Flow: "poll", BP: 0}, {Flow: "stipend", BP: 5_000}}
	if err != nil || !slices.Equal(slots.FlowScales, want) {
		t.Errorf("Slots().FlowScales = %v, %v, want %v", slots.FlowScales, err, want)
	}
	wantScaleRefusals(t, w, rec)
	if after, _ := w.Slots(); !slices.Equal(after.FlowScales, want) {
		t.Errorf("Slots().FlowScales after the refusals = %v, want %v unchanged", after.FlowScales, want)
	}
}

// wantScaleRefusals fails the test unless SetFlowScales refuses each bad call with its
// sentinel and emits nothing.
func wantScaleRefusals(t *testing.T, w *economysim.World, rec *recorder) {
	t.Helper()
	for _, tt := range []struct {
		name    string
		changes []economysim.FlowScale
		want    error
	}{
		{"an unknown flow", []economysim.FlowScale{{Flow: "nope", BP: 1}}, economysim.ErrUnknownFlow},
		{"a scale above the bound", []economysim.FlowScale{{Flow: "sales", BP: economysim.MaxFlowScale + 1}}, economysim.ErrOutOfBounds},
		{"a flow of both owners twice", []economysim.FlowScale{{Flow: "stipend", BP: 1}, {Flow: "stipend", BP: 2}}, economysim.ErrDuplicateID},
	} {
		rec.forget()
		if err := w.SetFlowScales(tt.changes); !errors.Is(err, tt.want) {
			t.Errorf("SetFlowScales(%s) error = %v, want %v", tt.name, err, tt.want)
		}
		if len(rec.events) != 0 {
			t.Errorf("SetFlowScales(%s) emitted %d events, want none from a refused call", tt.name, len(rec.events))
		}
	}
}

// TestSetFlowScales_NeitherModuleOn verifies that the call is ErrModuleOff when neither
// the money supply nor the fiscal module is on, and that the fiscal module alone is enough.
//
// Requirement: FR-FIS-010
func TestSetFlowScales_NeitherModuleOn(t *testing.T) {
	t.Parallel()
	off := mustWorld(t, referenceConfig())
	if err := off.SetFlowScales([]economysim.FlowScale{{Flow: "poll", BP: 1}}); !errors.Is(err, economysim.ErrModuleOff) {
		t.Errorf("SetFlowScales with neither module on error = %v, want ErrModuleOff", err)
	}
	fiscalOnly := mustWorld(t, fiscalConfig())
	if err := fiscalOnly.SetFlowScales([]economysim.FlowScale{{Flow: "poll", BP: 1}}); err != nil {
		t.Errorf("SetFlowScales(poll) with the fiscal module alone error = %v, want nil", err)
	}
}
