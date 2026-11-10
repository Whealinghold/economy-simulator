package economysim

import (
	"reflect"
	"slices"
	"testing"
)

// TestExternalHash_IDSevenBetweenSixAndEight verifies, from the bytes, that the
// pair of the external module is written under id 7 between Information (6) and
// Subsistence (8) and before Migration (9): a configuration with exactly the
// modules of ids 6, 7, 8 and 9 on, fully populated, writes the ids 6, 7, 8, 9 in
// that order.
//
// Requirement: FR-EXT-043
// Guard: the module is built after ids 8 to 10 and is declared last in Modules;
// writing it at the end of pairsModules would put id 7 after id 25 and break the
// ascending order the hash depends on (spec-config.md 8.2.2).
func TestExternalHash_IDSevenBetweenSixAndEight(t *testing.T) {
	t.Parallel()
	var c Config
	var n int64
	mods := reflect.ValueOf(&c.Modules).Elem()
	for _, name := range []string{"Information", "External", "Subsistence", "Migration"} {
		fillModules(mods.FieldByName(name), &n)
	}
	r := &idReader{b: encodeConfig(c.canonical())[1:]} // after the version byte
	if err := r.structOf(reflect.TypeFor[Config](), "Config"); err != nil {
		t.Fatalf("decode of the encoding: %v", err)
	}
	if want := []uint16{6, 7, 8, 9}; !slices.Equal(r.modules, want) {
		t.Errorf("module ids written = %v, want %v", r.modules, want)
	}
}

// TestReservedReasons_CoversOutside verifies that the six reasons of the external
// module are in reservedReasons, so a money flow with one of those IDs is
// ErrReasonCollision.
//
// Requirement: FR-MON-036, FR-EXT-017
func TestReservedReasons_CoversOutside(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"outside_budget", "outside_unspent", "outside_receipts", "outside_offer", "outside_unsold", "outside_purchases"} {
		if !slices.Contains(reservedReasons, name) {
			t.Errorf("reservedReasons lacks %q, want the six outside reasons", name)
		}
	}
}
