package economysim

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file tests the route side of the risk module at dispatch (plan 3-rsk
// Phase 4): the keyed draws (D13, D14), P3.1a, step 4a, the expected loss and its
// reads, over the world of shipConfig with a third region isle joined to far, so
// that a route from home to isle has two edges (the example of spec-risk.md 18.1).

// riskShip adds the edges far to isle and back, so that the path from home to isle
// is home, far, isle, and the route rows.
func riskShip(rows ...RouteRisk) func(c *Config) {
	return func(c *Config) {
		c.Edges = append(c.Edges, Edge{From: "b_far", To: "c_isle", Distance: 40}, Edge{From: "c_isle", To: "b_far", Distance: 40})
		c.Modules.Risk = &Risk{Routes: rows}
	}
}

// riskCoach is a route row of the hired mode coach.
func riskCoach(from, to string, strike, severity int64, common bool) RouteRisk {
	return RouteRisk{From: from, To: to, Mode: "coach", StrikeBP: strike, SeverityBP: severity, Common: common}
}

// riskFoot is a route row of the self-carried mode foot.
func riskFoot(from, to string, strike, severity int64, common bool) RouteRisk {
	return RouteRisk{From: from, To: to, Mode: "foot", StrikeBP: strike, SeverityBP: severity, Common: common}
}

// askFoot dispatches a trip of shipHome on foot to a region, to itself.
func askFoot(t *testing.T, w *World, to RegionID) ShipmentID {
	t.Helper()
	id, reason := ask(w, shipHome, 0, ShipmentRequest{Mode: modeFoot, To: to, ToSelf: true})
	if reason != "" {
		t.Fatalf("trip to %d = (%d, %q), want dispatched", to, id, reason)
	}
	return id
}

// stateOf returns the bytes of the state of the world's PCG: equal bytes before and
// after a call mean the call took no word and keyed no stream.
func stateOf(t *testing.T, w *World) []byte {
	t.Helper()
	b, err := w.rnd.pcg.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v, want nil", err)
	}
	return b
}

// haulFor gives shipHome the haul units and the grain for n coach shipments of one
// unit of grain to far (4 haul units a carrying unit; 8 gives room for a longer route).
func haulFor(t *testing.T, w *World, n int) {
	t.Helper()
	give(t, w, shipHome, commodityOf(t, w, "haul"), int64(8*n))
	give(t, w, shipHome, commodityOf(t, w, "grain"), int64(n))
}

// askCoach dispatches one coach shipment of one grain from home to a region.
func askCoach(t *testing.T, w *World, to RegionID) ShipmentID {
	t.Helper()
	id, reason := ask(w, shipHome, 0, ShipmentRequest{Mode: modeCoach, To: to, Consignee: shipFar,
		Lines: []Line{{Commodity: commodityOf(t, w, "grain"), Quantity: 1}}})
	if reason != "" {
		t.Fatalf("dispatch to %d = (%d, %q), want dispatched", to, id, reason)
	}
	return id
}

// TestDraws_KeyedNotPositional verifies the keys of the two route draws against the
// worked example of spec-risk.md 18.1 (models/rsk/loss.py): under seed 42 the stream
// (siteRiskDispatch, tick 12, shipment 1) reduces to 1 817 and the stream
// (siteRiskCommon, tick 12, From 1, sub 2 * 64) to 2 566. A draw depends on its key
// and on no other draw.
//
// Guard: spec-risk.md 3.4 and 11.2 were written for one positional stream and ADR
// 0008 replaced it. If a site changed its key or took its word from a shared stream,
// every strike of the module would move and these two values with it.
//
// Requirement: FR-RSK-007
func TestDraws_KeyedNotPositional(t *testing.T) {
	t.Parallel()
	s := newStream(42)
	s.keyed(siteRiskDispatch, 12, 1)
	if got := s.draw(wholeBP); got != 1_817 {
		t.Errorf("first draw of (siteRiskDispatch, 12, 1) = %d, want 1817", got)
	}
	s.keyedPair(siteRiskCommon, 12, 1, 2*MaxModes+0)
	if got := s.draw(wholeBP); got != 2_566 {
		t.Errorf("first draw of (siteRiskCommon, 12, 1, 128) = %d, want 2566", got)
	}
	s.keyed(siteRiskDispatch, 12, 1)
	s.draw(wholeBP)
	s.keyed(siteRiskDispatch, 12, 1)
	if got := s.draw(wholeBP); got != 1_817 {
		t.Errorf("draw after a rekey = %d, want 1817: the stream of a key does not depend on the draws before it", got)
	}
}

// pcgBytes returns the state of the PCG of a stream.
func pcgBytes(s *stream) []byte {
	b, err := s.pcg.MarshalBinary()
	if err != nil {
		return nil
	}
	return b
}

// TestDispatch_TwoRowsFoldInPathOrder verifies that a shipment from home to isle
// over two unit rows of probability 10 000 (both always strike) takes one word per
// row, keeps rhe(10 000 * 7 500, 10 000) = 7 500 and then rhe(7 500 * 6 000, 10 000)
// = 4 500, so LossBP is 5 500 (spec-risk.md 4.3.1), in the record and in
// ShipmentDispatched; and that a severity of 10 000 is a total loss.
//
// Requirement: FR-RSK-009, FR-RSK-011, FR-RSK-007
func TestDispatch_TwoRowsFoldInPathOrder(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name       string
		first, sec int64
		want       int64
	}{
		{"2 500 then 4 000", 2_500, 4_000, 5_500},
		{"4 000 then 2 500", 4_000, 2_500, 5_500},
		{"a total loss on the first row", 10_000, 4_000, 10_000},
		{"a total loss on the second row", 2_500, 10_000, 10_000},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w, rec := newShipWorld(t, riskShip(riskFoot("a_home", "b_far", 10_000, tt.first, false), riskFoot("b_far", "c_isle", 10_000, tt.sec, false)), nil)
			give(t, w, shipHome, commodityOf(t, w, "perish"), 4)
			id := askFoot(t, w, regIsle)
			if got := w.transport.ships[0].loss; got != tt.want {
				t.Errorf("shipment %d loss = %d, want %d", id, got, tt.want)
			}
			if d := of[ShipmentDispatched](rec); len(d) != 1 || d[0].LossBP != tt.want {
				t.Errorf("ShipmentDispatched = %+v, want one with LossBP %d", d, tt.want)
			}
			// Two words of the keyed stream of the shipment were taken, and no other.
			want := newStream(w.rnd.seed)
			want.keyed(siteRiskDispatch, w.Tick(), int64(id))
			want.draw(wholeBP)
			want.draw(wholeBP)
			if !bytes.Equal(stateOf(t, w), pcgBytes(want)) {
				t.Error("the world's stream is not the keyed stream of the shipment after two words")
			}
		})
	}
}

// TestDispatch_StrikeIsTheKeyedWordBelowP verifies that a unit row of probability
// 5 000 strikes a shipment exactly when the first word of the stream (siteRiskDispatch,
// tick, shipment) reduces to a value below 5 000, over 24 shipments, and that both
// outcomes occur.
//
// Requirement: FR-RSK-006, FR-RSK-009
func TestDispatch_StrikeIsTheKeyedWordBelowP(t *testing.T) {
	t.Parallel()
	const n = 24
	w, _ := newShipWorld(t, riskShip(riskCoach("a_home", "b_far", 5_000, 2_500, false)), nil)
	haulFor(t, w, n)
	struck := 0
	for range n {
		id := askCoach(t, w, regFar)
		s := newStream(w.rnd.seed)
		s.keyed(siteRiskDispatch, w.Tick(), int64(id))
		want := int64(0)
		if s.draw(wholeBP) < 5_000 {
			want = 2_500
			struck++
		}
		if got := w.transport.ships[id].loss; got != want {
			t.Errorf("shipment %d loss = %d, want %d", id, got, want)
		}
	}
	if struck == 0 || struck == n {
		t.Errorf("%d of %d shipments struck at 5 000, want some and not all", struck, n)
	}
}

// TestDraws_NoWordAtZero verifies that a row whose probability is 0 takes no word and
// keys no stream (spec-risk.md 3.4.3, FR-RSK-007), at dispatch and at P3.1a, and that
// a unit row at P3.1a and a refused request take none either.
//
// Requirement: FR-RSK-007, FR-RSK-010, FR-RSK-034
func TestDraws_NoWordAtZero(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, riskShip(riskCoach("a_home", "b_far", 5_000, 2_500, false), riskCoach("b_far", "c_isle", 5_000, 2_500, true)), nil)
	haulFor(t, w, 3)
	for i := range w.risk.routes {
		w.risk.routes[i].scale, w.risk.routes[i].p = 0, 0
	}
	before := stateOf(t, w)
	w.p31aCommon()
	id := askCoach(t, w, regFar)
	if !bytes.Equal(before, stateOf(t, w)) {
		t.Error("rows with probability 0 took a word or keyed a stream")
	}
	if got := w.transport.ships[id].loss; got != 0 {
		t.Errorf("loss = %d, want 0 with every probability 0", got)
	}
	// A unit row only: P3.1a takes no word. A refused request takes none.
	w.risk.routes[0].scale, w.risk.routes[0].p = wholeBP, 5_000
	w.p31aCommon()
	if _, reason := ask(w, shipHome, 0, ShipmentRequest{Mode: modeCoach, To: regFar, Consignee: shipFar, Lines: []Line{{Commodity: commodityOf(t, w, "grain"), Quantity: 1_000}}}); reason == "" {
		t.Fatal("the oversized request was dispatched, want a refusal")
	}
	if !bytes.Equal(before, stateOf(t, w)) {
		t.Error("P3.1a over unit rows, or a refused request, took a word")
	}
}

// TestP31a_CommonRowsEveryTickTickOnly verifies that a common row takes one word of
// the stream (siteRiskCommon, tick, From, To * MaxModes + Mode) at P3.1a every tick,
// with nothing on the road, and that its outcome is the word below its probability.
//
// Requirement: FR-RSK-008, FR-RSK-034
func TestP31a_CommonRowsEveryTickTickOnly(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, riskShip(riskCoach("a_home", "b_far", 5_000, 2_500, true), riskCoach("c_isle", "b_far", 1, 100, false)), nil)
	row := w.risk.find(regHome, regFar, modeCoach)
	if row == nil || !row.common {
		t.Fatal("no common row from home to far for the coach")
	}
	seen := map[bool]int{}
	for tick := range int64(20) {
		if w.Tick() != tick {
			t.Fatalf("Tick() = %d, want %d", w.Tick(), tick)
		}
		step(t, w, 1)
		s := newStream(w.rnd.seed)
		s.keyedPair(siteRiskCommon, tick, int64(regHome), int64(regFar)*MaxModes+int64(modeCoach))
		want := s.draw(wholeBP) < 5_000
		if row.struck != want {
			t.Errorf("tick %d: outcome = %v, want %v", tick, row.struck, want)
		}
		seen[want]++
	}
	if seen[true] == 0 || seen[false] == 0 {
		t.Errorf("outcomes over 20 ticks = %v, want both", seen)
	}
}

// TestDispatch_CommonRowOutcomeIsTheTicks verifies that two shipments on a common row
// in one tick share its outcome and take no word of their own for it: a struck row
// takes its share from both, a spared row from neither, and no stream is keyed.
//
// Requirement: FR-RSK-008, FR-RSK-009
func TestDispatch_CommonRowOutcomeIsTheTicks(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, riskShip(riskCoach("a_home", "b_far", 5_000, 2_500, true)), nil)
	haulFor(t, w, 4)
	for _, struck := range []bool{true, false} {
		w.risk.routes[0].struck = struck
		before := stateOf(t, w)
		want := int64(0)
		if struck {
			want = 2_500
		}
		for range 2 {
			id := askCoach(t, w, regFar)
			if got := w.transport.ships[id].loss; got != want {
				t.Errorf("struck %v: shipment %d loss = %d, want %d", struck, id, got, want)
			}
		}
		if !bytes.Equal(before, stateOf(t, w)) {
			t.Errorf("struck %v: a shipment on a common row took a word", struck)
		}
	}
}

// TestExpected_Example18_1 verifies the expected loss of the route of spec-risk.md
// 18.1: rows of 3 000 at 2 500 and 1 000 at 6 000 give f = 9 250 and 9 400, K = 9 250
// and 8 695, a loss of 1 305; the first row alone gives 750. The value is the one
// RouteLoss returns and the one a RouteRow of the origin carries, and it follows an
// edge status: with the second edge closed there is no path, and the table is
// rebuilt at the call that reopens it.
//
// Requirement: FR-RSK-024, FR-RSK-025
func TestExpected_Example18_1(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, riskShip(riskCoach("a_home", "b_far", 3_000, 2_500, false), riskCoach("b_far", "c_isle", 1_000, 6_000, true)), nil)
	t.Run("RouteLoss", func(t *testing.T) {
		for _, tt := range []struct {
			to   RegionID
			mode ModeID
			want int64
		}{{regIsle, modeCoach, 1_305}, {regFar, modeCoach, 750}, {regIsle, modeFoot, 0}} {
			if got, ok := w.RouteLoss(regHome, tt.to, tt.mode); !ok || got != tt.want {
				t.Errorf("RouteLoss(home, %d, %d) = (%d, %v), want (%d, true)", tt.to, tt.mode, got, ok, tt.want)
			}
		}
	})
	t.Run("RouteRow", func(t *testing.T) {
		w.p31Fix()
		got := map[[2]int]RouteRow{}
		for _, r := range w.transport.tables.routes[regHome] {
			got[[2]int{int(r.To), int(r.Mode)}] = r
		}
		if r := got[[2]int{int(regIsle), int(modeCoach)}]; !r.Has || r.LossBP != 1_305 {
			t.Errorf("RouteRow(home to isle, coach) = %+v, want Has with LossBP 1305", r)
		}
		if r := got[[2]int{int(regIsle), int(modeFoot)}]; r.LossBP != 0 {
			t.Errorf("RouteRow(home to isle, foot).LossBP = %d, want 0: no row for the mode", r.LossBP)
		}
	})
	t.Run("edge status", func(t *testing.T) {
		closeFarIsle := []EdgeStatus{{From: regFar, To: regIsle, Closed: true, FactorBP: wholeBP}}
		if err := w.setEdgeStatuses(closeFarIsle, CauseConsumer); err != nil {
			t.Fatalf("setEdgeStatuses(close) error = %v, want nil", err)
		}
		if got, ok := w.RouteLoss(regHome, regIsle, modeCoach); ok || got != 0 {
			t.Errorf("RouteLoss with the edge closed = (%d, %v), want (0, false): no path", got, ok)
		}
		closeFarIsle[0].Closed = false
		if err := w.setEdgeStatuses(closeFarIsle, CauseConsumer); err != nil {
			t.Fatalf("setEdgeStatuses(open) error = %v, want nil", err)
		}
		if got, ok := w.RouteLoss(regHome, regIsle, modeCoach); !ok || got != 1_305 {
			t.Errorf("RouteLoss after the reopening = (%d, %v), want (1305, true)", got, ok)
		}
	})
}

// TestRouteLoss_ZeroOffAndNoPath verifies that RouteLoss is 0 with ok true for a
// route of a world with the module off, and ok false for no path and for an unknown
// key.
//
// Requirement: FR-RSK-024
func TestRouteLoss_ZeroOffAndNoPath(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, nil, nil)
	if got, ok := w.RouteLoss(regHome, regFar, modeCoach); !ok || got != 0 {
		t.Errorf("RouteLoss with the module off = (%d, %v), want (0, true)", got, ok)
	}
	if got, ok := w.RouteLoss(regHome, regIsle, modeCoach); ok || got != 0 {
		t.Errorf("RouteLoss(no path) = (%d, %v), want (0, false)", got, ok)
	}
	if got, ok := w.RouteLoss(regHome, regFar, ModeID(9)); ok || got != 0 {
		t.Errorf("RouteLoss(unknown mode) = (%d, %v), want (0, false)", got, ok)
	}
}

// TestTransportLanes_LossFromRowsAtTenThousand verifies that the lane of a hired
// mode takes the expected loss of the configured rows at the scale of 10 000 over
// the path of the configured graph (spec-risk.md 8.3), 1 305 for the route of 18.1
// and 0 without the module.
//
// Requirement: FR-RSK-026
func TestTransportLanes_LossFromRowsAtTenThousand(t *testing.T) {
	t.Parallel()
	on, off := shipConfig(), shipConfig()
	riskShip(riskCoach("a_home", "b_far", 3_000, 2_500, false), riskCoach("b_far", "c_isle", 1_000, 6_000, true))(&on)
	riskShip()(&off)
	off.Modules.Risk = nil
	loss := func(c Config) map[[2]string]int64 {
		lanes, err := c.TransportLanes()
		if err != nil {
			t.Fatalf("TransportLanes() error = %v, want nil", err)
		}
		got := map[[2]string]int64{}
		for _, l := range lanes {
			got[[2]string{l.From, l.To}] = l.LossBP
		}
		return got
	}
	if got := loss(on); got[[2]string{"a_home", "c_isle"}] != 1_305 || got[[2]string{"a_home", "b_far"}] != 750 || got[[2]string{"c_isle", "a_home"}] != 0 {
		t.Errorf("lane losses with the rows = %v, want home to isle 1305, home to far 750, isle to home 0", got)
	}
	for k, v := range loss(off) {
		if v != 0 {
			t.Errorf("lane %v loss = %d with the module off, want 0", k, v)
		}
	}
}

// TestCost_RiskSitesZeroAllocs verifies that the read of the expected loss, P3.1a and
// step 4a allocate nothing (NFR-RSK-001).
//
// Requirement: FR-RSK-025, NFR-RSK-001
func TestCost_RiskSitesZeroAllocs(t *testing.T) {
	w, _ := newShipWorld(t, riskShip(riskCoach("a_home", "b_far", 3_000, 2_500, false), riskCoach("b_far", "c_isle", 1_000, 6_000, true)), nil)
	w.obs = observers{}
	s := &shipment{id: 1, mode: modeCoach, from: regHome, to: regIsle}
	for name, f := range map[string]func(){
		"RouteLoss": func() { w.RouteLoss(regHome, regIsle, modeCoach) },
		"P3.1a":     w.p31aCommon,
		"step 4a":   func() { s.loss = w.riskStrike(s) },
	} {
		if got := testing.AllocsPerRun(50, f); got != 0 {
			t.Errorf("AllocsPerRun(%s) = %v, want 0", name, got)
		}
	}
}

// TestRoutes_NeutralWithTheModuleOff verifies that RouteRow.LossBP is 0 for every
// route of a world with the module off.
//
// Requirement: FR-RSK-024
func TestRoutes_NeutralWithTheModuleOff(t *testing.T) {
	t.Parallel()
	w, _ := newShipWorld(t, nil, nil)
	w.p31Fix()
	for _, row := range w.transport.tables.routes[regHome] {
		if row.LossBP != 0 {
			t.Errorf("RouteRow %+v has LossBP %d with the module off, want 0", row, row.LossBP)
		}
	}
}

// TestRisk_NeutralAtZero verifies, for the route side, that a world with the module
// on and every probability 0 equals the world with it off: the same events, in the
// same order, over 12 ticks of a script that ships every tick.
//
// Requirement: FR-RSK-005
func TestRisk_NeutralAtZero(t *testing.T) {
	t.Parallel()
	ship := func(v *AgentView) []ShipmentRequest {
		if v.Agent != shipHome {
			return nil
		}
		return []ShipmentRequest{{Mode: modeFoot, To: regFar, ToSelf: true}}
	}
	run := func(c func(*Config)) []Event {
		w, rec := newShipWorld(t, c, ship)
		if w.risk != nil {
			for i := range w.risk.routes {
				w.risk.routes[i].scale, w.risk.routes[i].p = 0, 0
			}
		}
		step(t, w, 12)
		return slices.DeleteFunc(rec.events, func(e Event) bool { _, created := e.(WorldCreated); return created })
	}
	on := run(riskShip(RouteRisk{From: "a_home", To: "b_far", Mode: "foot", StrikeBP: 9_000, SeverityBP: 9_000},
		RouteRisk{From: "b_far", To: "a_home", Mode: "foot", StrikeBP: 9_000, SeverityBP: 9_000, Common: true}))
	off := run(nil)
	if len(off) == 0 || len(on) != len(off) {
		t.Fatalf("%d events with the module on at 0, %d with it off, want the same non-empty run", len(on), len(off))
	}
	for i := range on {
		if !reflect.DeepEqual(on[i], off[i]) {
			t.Fatalf("event %d differs: %+v with the module on, %+v off", i, on[i], off[i])
		}
	}
}

// TestLogs_RouteStruck verifies the Debug line of a shipment dispatched with a loss
// share above 0, with the keys of spec-risk.md 14.3.
//
// Requirement: FR-OBS-021
func TestLogs_RouteStruck(t *testing.T) {
	t.Parallel()
	cap := newCapture()
	w, _ := newShipWorld(t, riskShip(riskCoach("a_home", "b_far", 10_000, 2_500, false)), nil, WithLogger(slog.New(cap)))
	haulFor(t, w, 2)
	id := askCoach(t, w, regFar)
	var got *logRec
	for i := range *cap.lines {
		if l := &(*cap.lines)[i]; l.msg == "route struck" {
			got = l
		}
	}
	if got == nil || got.level != slog.LevelDebug || got.attrs[logkeys.Shipment] != int64(id) || got.attrs[logkeys.Share] != 2_500 ||
		got.attrs[logkeys.Mode] != int64(modeCoach) || got.attrs[logkeys.Region] != int64(regHome) || got.attrs[logkeys.Dest] != int64(regFar) {
		t.Errorf("route struck line = %+v, want Debug with shipment %d, share 2500, mode coach, region home, dest far", got, id)
	}
}

// riskSources returns the text of the non-test files of the module.
func riskSources(t *testing.T) map[string]string {
	t.Helper()
	files, err := filepath.Glob("risk*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("Glob(risk*.go) = %v, %v, want the module's files", files, err)
	}
	out := map[string]string{}
	for _, f := range files {
		if matched, _ := filepath.Match("*_test.go", f); matched {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v, want nil", f, err)
		}
		out[f] = string(b)
	}
	return out
}

// TestRisk_NoBranchOnIdentifier verifies, by a text guard over the module's files,
// that no code compares anything with a string literal (an identifier of a region,
// mode, commodity or class would be one).
//
// Guard: the module is data. A branch on an identifier in risk*.go would make a pack
// that lacks the identifier behave differently (architecture.md 9.1).
//
// Requirement: FR-RSK-039
func TestRisk_NoBranchOnIdentifier(t *testing.T) {
	t.Parallel()
	cmpLit := regexp.MustCompile(`(==|!=)\s*"|"\s*(==|!=)|case\s+"`)
	for name, text := range riskSources(t) {
		if cmpLit.MatchString(text) {
			t.Errorf("%s compares with a string literal, want no branch on an identifier", name)
		}
	}
}

// TestRisk_SetsNoEdgeStatus verifies, by a text guard, that the module never sets
// the status of an edge, and that a world with the module registers no more
// region_bound obligation than one without (FR-RSK-031, 032, N-090): a closed road
// is the consumer's call, and a cargo is bound by open_shipment already.
//
// Requirement: FR-RSK-031, FR-RSK-032
func TestRisk_SetsNoEdgeStatus(t *testing.T) {
	t.Parallel()
	forbidden := regexp.MustCompile(`setEdgeStatuses|EdgeStatus\{|EdgeStatusChanged|registerBound`)
	for name, text := range riskSources(t) {
		if forbidden.MatchString(text) {
			t.Errorf("%s names an edge status or a region_bound registration, want neither", name)
		}
	}
	on, _ := newShipWorld(t, riskShip(riskCoach("a_home", "b_far", 3_000, 2_500, false)), nil)
	off, _ := newShipWorld(t, nil, nil)
	if len(on.bounds.rows) != len(off.bounds.rows) {
		t.Errorf("%d region-bound obligations with the module, %d without, want equal", len(on.bounds.rows), len(off.bounds.rows))
	}
}
