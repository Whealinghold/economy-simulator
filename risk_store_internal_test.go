package economysim

import (
	"bytes"
	"log/slog"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/logkeys"
	"github.com/Whealinghold/economy-sim/internal/room"
)

// This file tests the store side of the risk module at P8.1a (plan 3-rsk Phase 6):
// the stored fraction and the share of a class, decide then apply, the keyed draws
// D15, the event StoreLoss and the log line, over the world of the room tests (the
// millers limited to a base room of (60, 40), the classes barn (100, 120) and shed
// (10, 10), the commodity grain of size (3, 2)).

// The store classes of stoWorld, by ID.
const (
	classBarn StoreClassID = iota
	classShed
)

// storeRisk is a store row of the region home.
func storeRisk(class string, strike, severity int64, common bool) StoreRisk {
	return StoreRisk{Region: "a_home", Class: class, StrikeBP: strike, SeverityBP: severity, Common: common}
}

// riskStoreWorld is stoWorld with the store rows.
func riskStoreWorld(t *testing.T, rows ...StoreRisk) (*World, *evLog) {
	t.Helper()
	return stoWorld(t, func(c *Config) { c.Modules.Risk = &Risk{Stores: rows} })
}

// stockAgent gives an agent the stores of the classes and grain up to total units.
func stockAgent(t *testing.T, w *World, id AgentID, total int64, classes ...StoreClassID) {
	t.Helper()
	a := &w.agents[id]
	for _, k := range classes {
		if err := w.addStore(a, k, CauseConsumer); err != nil {
			t.Fatalf("addStore(%d, %d) error = %v, want nil", id, k, err)
		}
	}
	if err := w.sto.apply(w.ledger, a); err != nil {
		t.Fatalf("apply(%d) error = %v, want nil", id, err)
	}
	grain := commodityOf(t, w, "grain")
	held, _, err := w.ledger.Holding(a.account, int(grain))
	if err != nil {
		t.Fatalf("Holding(%d) error = %v, want nil", id, err)
	}
	if total > held {
		give(t, w, id, grain, total-held)
	}
}

// grainOf returns the units of grain an agent holds.
func grainOf(t *testing.T, w *World, id AgentID) int64 {
	t.Helper()
	held, _, err := w.ledger.Holding(w.agents[id].account, int(commodityOf(t, w, "grain")))
	if err != nil {
		t.Fatalf("Holding(%d) error = %v, want nil", id, err)
	}
	return held
}

// storeLosses returns the StoreLoss events recorded.
func storeLosses(rec *evLog) []StoreLoss { return of[StoreLoss](rec) }

// TestShare_Example18_2 verifies the stored fraction, the share of a class and the
// loss of a limited owner with two classes, the worked example of spec-risk.md 18.2
// on the room-test world: two StoreLoss events, barn then shed, each with its share and
// its units, and the goods gone from the account.
//
// Arithmetic: the agent holds 40 grain, a load of (120, 80), over the base (60, 40),
// with a barn (100, 120) and a shed (10, 10): SumCap (110, 130), stored (60, 40). Barn:
// x = floor(60 * 100 / 110) = 54 and floor(40 * 120 / 130) = 36, phi = floor(54 *
// 10 000 / 120) = 4 500 and floor(36 * 10 000 / 80) = 4 500; share rhe(4 500 * 4 000 /
// 10 000) = 1 800; loss rhe(40 * 1 800 / 10 000 = 7.2) = 7. Shed: x = 5 and 3, phi =
// 416 and 375, so 416; share 416; loss rhe(40 * 416 / 10 000 = 1.664) = 2. The 9 units
// are taken from the quantity at the start, 40, and 31 are left.
//
// Requirement: FR-RSK-017, FR-RSK-018, FR-RSK-019, FR-RSK-021
func TestShare_Example18_2(t *testing.T) {
	t.Parallel()
	w, rec := riskStoreWorld(t, storeRisk("barn", 10_000, 4_000, false), storeRisk("shed", 10_000, 10_000, true))
	stockAgent(t, w, shipHome, 40, classBarn, classShed)
	w.p81aStores()
	grain := commodityOf(t, w, "grain")
	want := []StoreLoss{
		{Agent: shipHome, Region: regHome, Class: classBarn, Stores: 1, ShareBP: 1_800, Lost: []Amount{{Commodity: grain, Quantity: 7}}},
		{Agent: shipHome, Region: regHome, Class: classShed, Stores: 1, Common: true, ShareBP: 416, Lost: []Amount{{Commodity: grain, Quantity: 2}}},
	}
	if got := storeLosses(rec); !reflect.DeepEqual(got, want) {
		t.Errorf("StoreLoss events = %+v, want %+v", got, want)
	}
	if got := grainOf(t, w, shipHome); got != 31 {
		t.Errorf("grain held after the strike = %d, want 31", got)
	}
}

// TestShare_NStoresStruckMultiplyTheShare verifies that a class with several stores
// struck takes n times the share of one: three barns on a unit row.
//
// Arithmetic: 40 grain is a load of (120, 80); three barns give SumCap (300, 360) and
// stored (60, 40); a barn holds x = floor(60 * 100 / 300) = 20 and floor(40 * 120 /
// 360) = 13, phi = floor(20 * 10 000 / 120) = 1 666 and floor(13 * 10 000 / 80) =
// 1 625, so 1 666; share min(10 000, 3 * rhe(1 666 * 10 000 / 10 000)) = 4 998; loss
// rhe(40 * 4 998 / 10 000 = 19.992) = 20. The cap of 10 000 is a rule of internal/risk
// (Share), which a stored load below the load keeps out of reach of a real world.
//
// Requirement: FR-RSK-019
func TestShare_NStoresStruckMultiplyTheShare(t *testing.T) {
	t.Parallel()
	w, rec := riskStoreWorld(t, storeRisk("barn", 10_000, 10_000, false))
	stockAgent(t, w, shipHome, 40, classBarn, classBarn, classBarn)
	w.p81aStores()
	got := storeLosses(rec)
	if len(got) != 1 || got[0].Stores != 3 || got[0].ShareBP != 4_998 || len(got[0].Lost) != 1 || got[0].Lost[0].Quantity != 20 {
		t.Errorf("StoreLoss events = %+v, want one with 3 stores, share 4998 and 20 grain lost", got)
	}
	if held := grainOf(t, w, shipHome); held != 20 {
		t.Errorf("grain held after the strike = %d, want 20", held)
	}
}

// TestApply_CappedAtStartQuantity verifies that the loss of a second class is capped
// by what the first left: two classes at 9 000 each take rhe(40 * 9 000 / 10 000) = 36
// and then min(36, 40 - 36) = 4 of 40 units, so the account is empty and the second
// event carries the 4 and never a total above the quantity at the start.
//
// Requirement: FR-RSK-019
func TestApply_CappedAtStartQuantity(t *testing.T) {
	t.Parallel()
	w, _ := riskStoreWorld(t, storeRisk("barn", 10_000, 10_000, false))
	stockAgent(t, w, shipHome, 40, classBarn)
	r := w.risk
	r.strikes, r.amounts = r.strikes[:0], r.amounts[:0]
	r.strikes = append(r.strikes,
		storeStrike{agent: shipHome, class: classBarn, stores: 1, share: 9_000},
		storeStrike{agent: shipHome, class: classShed, stores: 1, share: 9_000})
	w.applyStrikes()
	grain := commodityOf(t, w, "grain")
	if want := []Amount{{Commodity: grain, Quantity: 36}, {Commodity: grain, Quantity: 4}}; !reflect.DeepEqual(r.amounts, want) {
		t.Errorf("amounts lost = %+v, want %+v", r.amounts, want)
	}
	if held := grainOf(t, w, shipHome); held != 0 {
		t.Errorf("grain held = %d, want 0", held)
	}
}

// TestP81a_OneAgentsLossMovesNoLaterAgentsDraws verifies that the strikes are decided
// before any loss is applied and keyed by agent: the second agent's StoreLoss is the
// same whether or not the first agent holds anything to lose.
//
// Requirement: FR-RSK-018
func TestP81a_OneAgentsLossMovesNoLaterAgentsDraws(t *testing.T) {
	t.Parallel()
	run := func(first int64) []StoreLoss { // 10 grain is under the base room: nothing stored
		w, rec := riskStoreWorld(t, storeRisk("barn", 10_000, 5_000, false))
		stockAgent(t, w, shipHome, first, classBarn)
		stockAgent(t, w, shipHome2, 40, classBarn)
		w.p81aStores()
		return slicesOfAgent(storeLosses(rec), shipHome2)
	}
	if with, without := run(40), run(10); len(with) != 1 || !reflect.DeepEqual(with, without) {
		t.Errorf("second agent's StoreLoss = %+v with the first loaded, %+v with it holding nothing stored, want one and equal", with, without)
	}
}

// slicesOfAgent returns the events of one agent.
func slicesOfAgent(events []StoreLoss, id AgentID) (out []StoreLoss) {
	for _, e := range events {
		if e.Agent == id {
			out = append(out, e)
		}
	}
	return out
}

// TestDraws_StoreKeyedStreams verifies the keys of D15 (spec-risk.md 11.2): a unit row
// draws from the stream (siteRiskStore, tick, agent, class), one word a store, and a
// common row from (siteRiskStoreCommon, tick, region, class), whatever the other
// agents and rows did. The expected outcome is the first word of that stream below the
// probability, computed here from the key.
//
// Requirement: FR-RSK-018, FR-RSK-033
func TestDraws_StoreKeyedStreams(t *testing.T) {
	t.Parallel()
	const p = 5_000
	for _, tt := range []struct {
		name   string
		common bool
		key    func(w *World)
	}{
		{"unit row", false, func(w *World) { w.rnd.keyedPair(siteRiskStore, 0, int64(shipHome), int64(classBarn)) }},
		{"common row", true, func(w *World) { w.rnd.keyedPair(siteRiskStoreCommon, 0, int64(regHome), int64(classBarn)) }},
	} {
		w, rec := riskStoreWorld(t, storeRisk("barn", p, 10_000, tt.common))
		stockAgent(t, w, shipHome, 40, classBarn)
		tt.key(w)
		want := w.rnd.draw(wholeBP) < p
		w.p81aStores()
		if got := len(storeLosses(rec)) == 1; got != want {
			t.Errorf("%s: struck = %v, want %v from the first word of the keyed stream", tt.name, got, want)
		}
	}
}

// TestDraws_StoreNoWordAtZeroOrNoExposure verifies that a row with a probability of 0
// takes no word, that a unit row takes none for an agent with no stored load, and that
// a common row with a probability above 0 draws every tick whatever the stores.
//
// Requirement: FR-RSK-007, FR-RSK-034
func TestDraws_StoreNoWordAtZeroOrNoExposure(t *testing.T) {
	t.Parallel()
	zero, _ := riskStoreWorld(t, storeRisk("barn", 5_000, 10_000, false))
	stockAgent(t, zero, shipHome, 40, classBarn)
	if err := zero.SetStoreRiskScales([]StoreRiskScale{{Region: regHome, Class: classBarn, BP: 0}}); err != nil {
		t.Fatalf("SetStoreRiskScales error = %v, want nil", err)
	}
	before := stateOf(t, zero)
	zero.p81aStores()
	if !bytes.Equal(before, stateOf(t, zero)) {
		t.Error("a unit row at a probability of 0 took a word")
	}
	empty, _ := riskStoreWorld(t, storeRisk("barn", 5_000, 10_000, false))
	stockAgent(t, empty, shipHome, 10, classBarn) // a load of (30, 20) is under the base room
	before = stateOf(t, empty)
	empty.p81aStores()
	if !bytes.Equal(before, stateOf(t, empty)) {
		t.Error("an agent with no stored load took a word")
	}
	common, _ := riskStoreWorld(t, storeRisk("barn", 5_000, 10_000, true))
	before = stateOf(t, common)
	common.p81aStores()
	if bytes.Equal(before, stateOf(t, common)) {
		t.Error("a common row with a probability above 0 and no store took no word, want its word every tick")
	}
}

// TestStrike_StoreCapacityLevyAndHasStoreUnchanged verifies that a strike takes goods
// and nothing else: the stores of the agent, its room and its stored classes are the
// same after it (spec-risk.md 6.3.3).
//
// Requirement: FR-RSK-020
func TestStrike_StoreCapacityLevyAndHasStoreUnchanged(t *testing.T) {
	t.Parallel()
	w, rec := riskStoreWorld(t, storeRisk("barn", 10_000, 10_000, false))
	stockAgent(t, w, shipHome, 40, classBarn)
	owned, rm := w.sto.owned(shipHome), roomOfAgent(t, w, shipHome).Room
	w.p81aStores()
	if len(storeLosses(rec)) != 1 {
		t.Fatalf("StoreLoss events = %d, want 1", len(storeLosses(rec)))
	}
	if got := w.sto.owned(shipHome); got != owned {
		t.Errorf("stores owned after the strike = %d, want %d", got, owned)
	}
	if got := roomOfAgent(t, w, shipHome).Room; got != rm {
		t.Errorf("room after the strike = %+v, want %+v", got, rm)
	}
}

// TestStrike_CohortSameShareWholeAccount verifies that a cohort loses the same share
// of its whole account and its store capacity is not multiplied by the weight
// (spec-risk.md 6.3.4).
//
// Arithmetic: weight 3 over the base (60, 40) is (180, 120); 90 grain is a load of
// (270, 180); one barn (100, 120): stored (min(100, 270 - 180), min(120, 180 - 120)) =
// (90, 60), x = 90 and 60, phi = floor(90 * 10 000 / 270) = 3 333 and 60 * 10 000 /
// 180 = 3 333; share 3 333; loss rhe(90 * 3 333 / 10 000 = 29.997) = 30.
//
// Requirement: FR-RSK-038
func TestStrike_CohortSameShareWholeAccount(t *testing.T) {
	t.Parallel()
	w, rec := riskStoreWorld(t, storeRisk("barn", 10_000, 10_000, false))
	w.agents[shipHome].weight = 3
	stockAgent(t, w, shipHome, 90, classBarn)
	w.p81aStores()
	got := storeLosses(rec)
	if len(got) != 1 || got[0].ShareBP != 3_333 || len(got[0].Lost) != 1 || got[0].Lost[0].Quantity != 30 {
		t.Errorf("StoreLoss events = %+v, want one with share 3333 and 30 grain", got)
	}
}

// TestEvent_StoreLossEmptyLostIsAStrikeWithNoLoss verifies that a strike whose every
// product rounds to 0 is still reported, with no units (spec-risk.md 6.4.1).
//
// Arithmetic: 23 grain with a shed alone is a load of (69, 46) in a room of (70, 50):
// stored (min(10, 69 - 60), min(10, 46 - 40)) = (9, 6), x = 9 and 6, phi = floor(9 *
// 10 000 / 69) = 1 304 and floor(6 * 10 000 / 46) = 1 304; a severity of 1 gives share
// rhe(1 304 * 1 / 10 000) = 0, so no unit is lost.
//
// Requirement: FR-RSK-021
func TestEvent_StoreLossEmptyLostIsAStrikeWithNoLoss(t *testing.T) {
	t.Parallel()
	w, rec := riskStoreWorld(t, storeRisk("shed", 10_000, 1, false))
	stockAgent(t, w, shipHome, 23, classShed)
	w.p81aStores()
	want := []StoreLoss{{Agent: shipHome, Region: regHome, Class: classShed, Stores: 1}}
	if got := storeLosses(rec); !reflect.DeepEqual(got, want) {
		t.Errorf("StoreLoss events = %+v, want %+v (a strike with no units)", got, want)
	}
	if held := grainOf(t, w, shipHome); held != 23 {
		t.Errorf("grain held = %d, want 23", held)
	}
}

// TestP81a_AscendingAgentThenClass verifies the order of the events: ascending agent
// ID, then class, whatever the order the stores were given in.
//
// Requirement: FR-RSK-021
func TestP81a_AscendingAgentThenClass(t *testing.T) {
	t.Parallel()
	w, rec := riskStoreWorld(t, storeRisk("barn", 10_000, 1_000, false), storeRisk("shed", 10_000, 1_000, false))
	stockAgent(t, w, shipHome2, 40, classShed, classBarn)
	stockAgent(t, w, shipHome, 40, classShed, classBarn)
	w.p81aStores()
	var order [][2]int
	for _, e := range storeLosses(rec) {
		order = append(order, [2]int{int(e.Agent), int(e.Class)})
	}
	if want := [][2]int{{3, 0}, {3, 1}, {4, 0}, {4, 1}}; !reflect.DeepEqual(order, want) {
		t.Errorf("(agent, class) of the events = %v, want %v", order, want)
	}
}

// TestStoreRead_MatchesStorageState verifies that what the risk module reads of the
// storage module is what the storage module defines: the stored load of an account is
// the smaller of the capacity of its stores and the load above the base room, per
// dimension, and the class capacities are those of the configuration.
//
// Guard: the module reads storage state in the package (plan 3-rsk D8, N-375) and no
// edit of storage.go is planned. If the representation of the stores or of the stored
// load changed, the risk module would draw and take shares from the wrong state and
// no other test would notice.
func TestStoreRead_MatchesStorageState(t *testing.T) {
	t.Parallel()
	w, _ := riskStoreWorld(t, storeRisk("barn", 10_000, 1_000, false))
	stockAgent(t, w, shipHome, 40, classBarn, classShed)
	a := &w.agents[shipHome]
	st, err := w.Agent(shipHome)
	if err != nil {
		t.Fatalf("Agent() error = %v, want nil", err)
	}
	// Load (120, 80) less the base (60, 40) is (60, 40), under the capacity (110, 130).
	want := room.Pair{Mass: min(110, st.Room.Load.Mass-60), Bulk: min(130, st.Room.Load.Bulk-40)}
	if got := w.storedLoad(a); got != want {
		t.Errorf("storedLoad() = %+v, want %+v", got, want)
	}
	stored, load, sumCap := w.storeExposure(a)
	if stored != want || load != (room.Pair{Mass: st.Room.Load.Mass, Bulk: st.Room.Load.Bulk}) || sumCap != (room.Pair{Mass: 110, Bulk: 130}) {
		t.Errorf("storeExposure() = %+v, %+v, %+v, want %+v, the load of the room query and (110, 130)", stored, load, sumCap, want)
	}
	if got := w.sto.classes; got[classBarn] != (room.Pair{Mass: stoBarnMass, Bulk: stoBarnBulk}) || w.sto.count(shipHome, int(classBarn)) != 1 {
		t.Errorf("classes = %+v, barns owned = %d, want the configured barn and 1", got, w.sto.count(shipHome, int(classBarn)))
	}
}

// TestLogs_StoreStruck verifies the Debug line of a class struck, with the keys of
// spec-risk.md 14.3. A barn alone holds the stored load (60, 40) of 40 grain, phi =
// 60 * 10 000 / 120 = 5 000, so the share at a severity of 4 000 is 2 000.
//
// Requirement: FR-OBS-021
func TestLogs_StoreStruck(t *testing.T) {
	t.Parallel()
	cap := newCapture()
	w, _ := stoWorld(t, func(c *Config) {
		c.Modules.Risk = &Risk{Stores: []StoreRisk{storeRisk("barn", 10_000, 4_000, false)}}
	}, WithLogger(slog.New(cap)))
	stockAgent(t, w, shipHome, 40, classBarn)
	w.p81aStores()
	var got *logRec
	for i := range *cap.lines {
		if l := &(*cap.lines)[i]; l.msg == "store struck" {
			got = l
		}
	}
	if got == nil || got.level != slog.LevelDebug || got.attrs[logkeys.Agent] != int64(shipHome) || got.attrs[logkeys.Share] != 2_000 ||
		got.strs[logkeys.StoreClass] != "barn" {
		t.Errorf("store struck line = %+v, want Debug with agent %d, share 2000 and class barn", got, shipHome)
	}
}

// TestCost_RiskStoreSiteZeroAllocs verifies that P8.1a allocates nothing with no
// observer and no logger, over an owner struck by a unit row and a common row
// (NFR-RSK-001).
//
// Requirement: NFR-RSK-001
func TestCost_RiskStoreSiteZeroAllocs(t *testing.T) {
	w, _ := riskStoreWorld(t, storeRisk("barn", 10_000, 1_000, false), storeRisk("shed", 10_000, 1_000, true))
	stockAgent(t, w, shipHome, 40, classBarn, classShed)
	w.obs = observers{}
	w.p81aStores() // warms the scratch
	if got := testing.AllocsPerRun(20, w.p81aStores); got != 0 {
		t.Errorf("AllocsPerRun(P8.1a) = %v, want 0", got)
	}
}

// TestRisk_NeutralAtZeroStores verifies, for the store side, that a world with the
// module on and every store probability 0 equals the world with it off: the same
// events over 12 ticks with an owner of two classes.
//
// Requirement: FR-RSK-005
func TestRisk_NeutralAtZeroStores(t *testing.T) {
	t.Parallel()
	run := func(rows []StoreRisk) []Event {
		mut := func(c *Config) {
			if rows != nil {
				c.Modules.Risk = &Risk{Stores: rows}
			}
		}
		w, rec := stoWorld(t, mut)
		stockAgent(t, w, shipHome, 40, classBarn, classShed)
		if w.risk != nil {
			for i := range w.risk.stores {
				w.risk.stores[i].scale, w.risk.stores[i].p = 0, 0
			}
		}
		step(t, w, 12)
		return slices.DeleteFunc(rec.events, func(e Event) bool { _, created := e.(WorldCreated); return created })
	}
	on := run([]StoreRisk{storeRisk("barn", 9_000, 9_000, false), storeRisk("shed", 9_000, 9_000, true)})
	off := run(nil)
	if len(off) == 0 || !reflect.DeepEqual(on, off) {
		t.Errorf("%d events with the module on at 0, %d with no row, want the same non-empty run", len(on), len(off))
	}
}

// TestInvariants_Risk_StoreBoundaryBalanceEqualsEvents verifies, over 12 ticks of a
// world with the audit on, that the units the loss boundary holds are exactly the
// units the StoreLoss events say were lost, each tick, and that something was lost
// (spec-risk.md 7.3 items 2 and 3, store side).
//
// Requirement: FR-RSK-022, FR-RSK-023
func TestInvariants_Risk_StoreBoundaryBalanceEqualsEvents(t *testing.T) {
	t.Parallel()
	w, rec := riskStoreWorld(t, storeRisk("barn", 6_000, 5_000, false), storeRisk("shed", 5_000, 8_000, true))
	stockAgent(t, w, shipHome, 40, classBarn, classShed)
	stockAgent(t, w, shipHome2, 40, classBarn)
	grain := commodityOf(t, w, "grain")
	var lost int64
	for range 12 {
		step(t, w, 1)
		lost = 0
		for _, e := range storeLosses(rec) {
			for _, a := range e.Lost {
				lost += int64(a.Quantity)
			}
		}
		if got := lossBoundary(t, w, grain); got != lost {
			t.Fatalf("tick %d: the loss boundary holds %d grain, the events say %d", w.Tick(), got, lost)
		}
	}
	if lost == 0 {
		t.Error("no unit was lost in 12 ticks at probabilities of 6 000 and 5 000, want some")
	}
}

// TestRisk_WorkIsCounted verifies the store half of NFR-RSK-002: a tick with no store
// row above 0 reads no agent at P8.1a. A store record of an agent that does not exist
// stands in for the read: reading it indexes the agents and panics, so it is read when
// a row is above 0 (the control) and never when every row is at 0.
//
// Requirement: NFR-RSK-002
// Guard: P8.1a visiting the agents of a world that has no store risk in force.
func TestRisk_WorkIsCounted(t *testing.T) {
	t.Parallel()
	w, _ := riskStoreWorld(t, storeRisk("barn", 5_000, 10_000, false))
	stockAgent(t, w, shipHome, 40, classBarn)
	ghost := len(w.agents)
	w.sto.stores = append(w.sto.stores, make([][]uint8, ghost+1-len(w.sto.stores))...)
	w.sto.stores[ghost] = []uint8{1}
	reads := func() (read bool) {
		defer func() { read = recover() != nil }()
		w.p81aStores()
		return false
	}
	if !reads() {
		t.Fatal("P8.1a with a row above 0 did not read the store record of the stand-in, want the control to read it")
	}
	if err := w.SetStoreRiskScales([]StoreRiskScale{{Region: regHome, Class: classBarn, BP: 0}}); err != nil {
		t.Fatalf("SetStoreRiskScales error = %v, want nil", err)
	}
	if reads() {
		t.Error("P8.1a with every row at 0 read an agent, want none (NFR-RSK-002)")
	}
}
