package economysim

import (
	"reflect"
	"testing"
)

// TestTenureConstruct_ModuleOnWithoutTitleEqualsOff verifies that the module on with no
// title and no term changes no result of the run: the snapshot after 20 ticks equals
// the module off's but for the fields of the tenure module (spec-land.md 1, FR-LND-006
// first half; the rows of the snapshot carry the scale and the commons share).
//
// Requirement: FR-LND-006, FR-LND-001
func TestTenureConstruct_ModuleOnWithoutTitleEqualsOff(t *testing.T) {
	t.Parallel()
	off := tenureWorldConfig(nil)
	off.Modules.Tenure = nil
	on := tenureWorldConfig(func(m *TenureConfig) { m.Terms, m.Holdings = nil, nil })
	a, b := newMill(t, off, WithSeed(exitSeed)), newMill(t, on, WithSeed(exitSeed))
	for range 20 {
		if err := a.Step(); err != nil {
			t.Fatalf("Step of the world with the module off = %v, want nil", err)
		}
		if err := b.Step(); err != nil {
			t.Fatalf("Step of the world with the module on = %v, want nil", err)
		}
	}
	sa, _ := a.Snapshot()
	sb, _ := b.Snapshot()
	sb.Modules, sb.ConfigHash = sa.Modules, sa.ConfigHash
	for i := range sb.Endowments {
		sb.Endowments[i].ScaleBP, sb.Endowments[i].CommonsBP = 0, 0
	}
	if !reflect.DeepEqual(sa, sb) {
		t.Errorf("the snapshot with the module on and no title differs from the module off's after 20 ticks")
	}
}
