package economysim

import "testing"

// tenureHire adds the tenure module to the hire world: the ore of region r0 is let at rent
// per unit drawn, closed to non-holders when closed, and the role holder holds the whole of
// it (per household).
func tenureHire(holder string, rent int64, closed bool) func(c *Config) {
	return func(c *Config) {
		c.Modules.Estates = &Estates{}
		c.Modules.Tenure = &TenureConfig{
			Terms:    []TenureTerm{{Region: "r0", Resource: "ore", HasRent: true, Rent: rent, HasClosed: true, Closed: closed}},
			Holdings: []Holding{{Region: "r0", Resource: "ore", Holder: HolderRef{Kind: HolderKindRole, Role: holder}, ShareBP: 10_000}},
		}
	}
}

// TestFormation_NoTitleRentFundsBounds verifies check 8 of the labour round
// (spec-employment.md 5.5, spec-land.md 4.7.2) with a titled endowment: a recipe that draws
// 3 ore an execution, the whole ore held by one role.
//
// A closed endowment held by the hands leaves the farm no unit open by rule: no_title. A let
// endowment held by the hands leaves the farm the whole draw to rent, 3 * 1 000 = 3 000
// for one household, above its 1 000 money: rent_funds. When the farm holds the title its
// entitlement covers the draw, the least rent is 0 and the hire forms.
//
// Requirement: FR-EMP-023
func TestFormation_NoTitleRentFundsBounds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		holder string
		closed bool
		field  func(r LabourRound) int64
		want   int64
		hired  int64
	}{
		{"closed to the farm: no_title", "hand", true, func(r LabourRound) int64 { return r.NoTitle }, 1, 0},
		{"let to the farm at a rent it cannot pay: rent_funds", "hand", false, func(r LabourRound) int64 { return r.RentFunds }, 1, 0},
		{"the farm holds the title: no refusal", "farm", true, func(r LabourRound) int64 { return r.NoTitle + r.RentFunds }, 0, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			log := &labourLog{}
			o := defaultHire(hireOpts{ore: 1_000, farmMoney: 1_000, edit: tenureHire(tt.holder, 1_000, tt.closed)})
			w := hireWorld(t, o, farmWants(1, 100, 0), WithObserver(log))
			stepN(t, w, 1)
			if len(log.rounds) != 1 {
				t.Fatalf("LabourRound events = %d, want 1", len(log.rounds))
			}
			if r := log.rounds[0]; tt.field(r) != tt.want || r.Matched != tt.hired || r.NoEndowment != 0 {
				t.Errorf("LabourRound = %+v, want the reason's field %d, Matched %d and no no_endowment", r, tt.want, tt.hired)
			}
		})
	}
}

// TestFormation_NoRoomBound verifies check 11 of the labour round: with the storage module
// on and a room of 3 mass and 3 bulk for a unit of 1 and 1, an execution makes 2 grain, so the farm's store
// holds the output of one execution and not two. The first hand is hired (committed 0 + 1 =
// 1 execution, 2 grain), the second would make 2 executions, 4 grain above the room: no_room.
//
// Requirement: FR-EMP-023
func TestFormation_NoRoomBound(t *testing.T) {
	t.Parallel()
	log := &labourLog{}
	edit := func(c *Config) {
		c.Commodities[0].Mass = 1
		c.Modules.Storage = &Storage{Base: &Capacity{Mass: 3, Bulk: 3}}
	}
	w := hireWorld(t, defaultHire(hireOpts{hands: 2, edit: edit}), farmWants(2, 100, 0), WithObserver(log))
	stepN(t, w, 1)
	if len(log.rounds) != 1 {
		t.Fatalf("LabourRound events = %d, want 1", len(log.rounds))
	}
	if r := log.rounds[0]; r.Matched != 1 || r.NoRoom != 1 || r.Unfilled != 1 {
		t.Errorf("LabourRound = %+v, want Matched 1, NoRoom 1 and Unfilled 1", r)
	}
}
