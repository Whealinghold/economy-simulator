package economysim

import (
	"math"
	"testing"
)

// This file tests the cover surface against hostile figures (plan 3-ins Phases 4 and 5):
// a view or a plan a wrapper strategy built by hand, and a request against a shipment of
// another agent. None may panic, wrap or change the world.

// TestDefaultOffer_HostileFiguresOfAHandBuiltView verifies that a view whose figures are
// negative or at the int64 edges gives no offer, or one inside the money bound, never a
// panic and never a wrapped figure.
//
// Worked: bids of MinInt64 against a balance of 1 overflow the subtraction, so nothing is
// offered; a balance of MaxInt64 with a share of 10 000 bp and a cap of 10 000 bp offers at
// most MaxMoney; a negative open liability has no meaning, so nothing is offered.
//
// Requirement: FR-INS-062, FR-INS-063
func TestDefaultOffer_HostileFiguresOfAHandBuiltView(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cv   CoverView
		bids int64
		ok   bool
	}{
		{"bids that overflow the subtraction", CoverView{Money: 1, MinSum: 1}, math.MinInt64, false},
		{"negative open liability", CoverView{Money: 1_000, MinSum: 1, Open: -1}, 0, false},
		{"negative money", CoverView{Money: -5, MinSum: 1}, 0, false},
		{"money at the edge is capped", CoverView{Money: math.MaxInt64, MinSum: 1}, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.cv.Underwrite = &UnderwriteTerms{LoadingBP: 1, CoverBP: 10_000, CapBP: 10_000, ShareBP: 10_000}
			o, ok := DefaultOffer(&AgentView{Insurance: &tt.cv}, tt.bids)
			if ok != tt.ok {
				t.Fatalf("DefaultOffer ok = %v, want %v (offer %+v)", ok, tt.ok, o)
			}
			if ok && (o.MaxLiability < tt.cv.MinSum || o.MaxLiability > MaxMoney) {
				t.Errorf("DefaultOffer MaxLiability = %d, want from MinSum to MaxMoney", o.MaxLiability)
			}
		})
	}
}

// TestDefaultCover_HostilePlanFigures verifies that the default insured reads a plan whose
// quantities are negative or at the int64 edges, or whose commodity the view holds no value
// of, without a panic: a product that overflows reads as the largest value, so the lane is
// asked for; a negative or unvalued cargo is worth nothing, so it is not.
//
// Requirement: FR-INS-061
func TestDefaultCover_HostilePlanFigures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		line Line
		want int
	}{
		{"quantity at the edge overflows the product", Line{Commodity: grain, Quantity: math.MaxInt64}, 1},
		{"negative quantity is worth nothing", Line{Commodity: grain, Quantity: -5}, 0},
		{"a commodity the view has no value of", Line{Commodity: 99, Quantity: 5}, 0},
		{"a negative commodity", Line{Commodity: -1, Quantity: 5}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := coverView(InsureTerms{MinLossBP: 1, RuinShareBP: 1, MinCoverBP: 1, MaxLoadingBP: 7}, 500)
			p := &TradePlan{Shipments: []ShipmentRequest{{Mode: modeCoach, To: regFar, Consignee: 6, Lines: []Line{tt.line}}}}
			DefaultCover(v, p)
			if len(p.Cover) != tt.want {
				t.Errorf("DefaultCover asked %d lane(s), want %d", len(p.Cover), tt.want)
			}
		})
	}
}

// TestAsked_AnotherAgentsShipmentIsNotAsked verifies that a request names only its own
// agent's shipments: another agent's shipment on the same lane from the same region is not
// a candidate, nor is one from another region or already covered.
//
// Requirement: FR-INS-012, FR-INS-013
func TestAsked_AnotherAgentsShipmentIsNotAsked(t *testing.T) {
	t.Parallel()
	w, _ := newInsWorld(t, insConfig(defaultInsure(), defaultUnderwrite()), nil)
	req := &coverRequest{agent: insHome, lane: coachLane}
	own := shipment{id: 1, mode: modeCoach, from: regHome, to: regFar, shipper: insHome}
	other := own
	other.shipper = insHome2
	elsewhere := own
	elsewhere.from = regFar
	covered := own
	covered.id = 2
	w.insurance.byShipment[covered.id] = 1
	tests := []struct {
		name string
		s    *shipment
		want bool
	}{
		{"its own", &own, true},
		{"another agent's", &other, false},
		{"from another region", &elsewhere, false},
		{"already covered", &covered, false},
	}
	for _, tt := range tests {
		if got := w.asked(tt.s, regHome, req); got != tt.want {
			t.Errorf("asked(%s) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
