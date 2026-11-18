package economysim_test

import (
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// TestValidateInstitutions_BodyListRepeat verifies that an entry listed twice in a
// body's Roles, Regions or Jurisdictions is ErrDuplicateID at the path of the second
// entry, and that the same body with the entry once is valid.
//
// Requirement: FR-ORG-047
//
// Guard: a repeat builds the same world as the single entry (the scope and the overlap
// test read distinct indexes) yet hashes differently, because canonical sorts a list and
// does not drop a repeat. Without the rule two spellings of one body would be two
// hashes, which spec-config.md 3.4 forbids for the lists of a body.
func TestValidateInstitutions_BodyListRepeat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		key  string
		set  func(b *economysim.Body)
	}{
		{"roles", "roles[1]", func(b *economysim.Body) { b.Roles = []string{"maker", "maker"} }},
		{"regions", "regions[1]", func(b *economysim.Body) { b.Regions = []string{"north", "north"} }},
		{"jurisdictions", "jurisdictions[1]", func(b *economysim.Body) { b.Jurisdictions = []string{"crown", "crown"} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := institutionsConfig()
			tt.set(&c.Modules.Institutions.Bodies[0])
			err := c.Validate()
			if !errors.Is(err, economysim.ErrDuplicateID) {
				t.Fatalf("Validate() = %v, want ErrDuplicateID", err)
			}
			var ve *economysim.ValidationError
			if !errors.As(err, &ve) || !slices.Equal(ve.Path, orgAt("bodies", 0, tt.key)) {
				t.Errorf("Validate() = %v, want an error at %v", err, orgAt("bodies", 0, tt.key))
			}
			// The single entry is the valid spelling of the same body.
			c = institutionsConfig()
			if err := c.Validate(); err != nil {
				t.Errorf("Validate() of the body without the repeat = %v, want nil", err)
			}
		})
	}
}
