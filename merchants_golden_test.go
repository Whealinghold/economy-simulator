package economysim_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/strategy"
)

// merchantsPosLines is the number of positions that ran per tick with the merchants
// module on over the transport and information modules: the 22 of the off set, P1.3
// and P3.1 of transport, P3.6 and P5.2 of information, and P1.4, P3.3 and P3.3a of
// the merchants.
const merchantsPosLines = 29

// packMerchants is the module-alone pack of the variant: the cold pack, its trade
// overlay (the edges), its transport overlay (the modes and the carrier), its
// information overlay (the rumours) and the merchants overlay (spec-reference-chain.md
// 4.5, 7.9; plan 2-15 D514). The module needs the two modules, so "alone" is the module
// and what it depends on (ErrModuleDependency).
func packMerchants(variant string) []string {
	overlay := "overlay/merchants.toml"
	if variant == "three" {
		overlay = "overlay/merchants-three.toml"
	}
	return append(slices.Clone(packLayers[variant]), tradeOverlay(variant),
		"overlay/transport-"+variant+".toml", "overlay/information-"+variant+".toml", overlay)
}

// TestPacks_MerchantsOverlayLoads verifies that the merchants overlay loads over the
// variants two and three with their trade, transport and information overlays,
// validates, puts the module and the role on, and raises none of the module's route
// or share warnings (lane_without_consignee, cargo_never_delivered, consigns_without_cargo,
// merchant_no_consignee, merchants_one_agent and the three share warnings).
//
// Requirement: FR-MER-052, FR-MER-054, FR-REF-025
//
// Guard: the golden and the scenarios read these files; a pack that stopped being valid
// or lost its role would change them without a message.
func TestPacks_MerchantsOverlayLoads(t *testing.T) {
	t.Parallel()
	for _, variant := range []string{"two", "three"} {
		t.Run(variant, func(t *testing.T) {
			t.Parallel()
			cfg := loadValidPack(t, packMerchants(variant)...)
			i := slices.IndexFunc(cfg.Roles, func(r economysim.Role) bool { return r.ID == "broker" })
			if cfg.Modules.Merchants == nil || i < 0 || cfg.Roles[i].Merchant == nil || cfg.Roles[i].Strategy != "merchant" {
				t.Fatalf("pack %s: module %+v, broker role at %d, want the module on and a broker with a merchant block", variant, cfg.Modules.Merchants, i)
			}
			for _, w := range cfg.Warnings() {
				if strings.HasPrefix(w.Rule, "merchant") || w.Rule == "lane_without_consignee" || w.Rule == "cargo_never_delivered" || w.Rule == "consigns_without_cargo" {
					t.Errorf("pack %s: warning %+v, want none of the module's", variant, w)
				}
			}
		})
	}
}

// TestOverlay_MerchantsValidatesAndSolves verifies D516: the pack with a role that has
// no recipe validates, and SolveSteadyState either accepts it or reports why with an
// error that is not a panic, so the golden can say which.
//
// Requirement: FR-MER-052
func TestOverlay_MerchantsValidatesAndSolves(t *testing.T) {
	t.Parallel()
	cfg := loadValidPack(t, packMerchants("two")...)
	ss, err := economysim.SolveSteadyState(context.Background(), cfg)
	switch {
	case err != nil:
		t.Logf("SolveSteadyState reports why: %v", err)
	case len(ss.Programmes) == 0:
		t.Error("SolveSteadyState accepted the pack with no programme, want at least one")
	}
}

// solveMerchants solves the steady state of the module-alone pack, the programme the
// golden reports; a pack the solver refuses reports nothing (D516).
func solveMerchants(t testing.TB) solveReport {
	t.Helper()
	ss, err := economysim.SolveSteadyState(context.Background(), loadValidPack(t, packMerchants("two")...))
	if err != nil {
		return solveReport{}
	}
	return newSolveReport(ss)
}

// contractScript is the scripted merchant of the contracts golden and the scenarios: the
// default strategy after its first ticks, and in them every broker of the region from sends an
// offer of a lot of fuel to the region to, and every broker of to posts a take for
// the same lot. It keeps no state (architecture.md 7.7), so a restore needs nothing of
// it. It is registered under the name "merchant", which the broker role names, in place
// of strategy.Options().
type contractScript struct {
	from, to economysim.RegionID
	fuel     economysim.CommodityID
	coach    economysim.ModeID
	until    int64
	lot      economysim.Quantity
}

func (contractScript) Work(v *economysim.AgentView, rng *rand.Rand) economysim.JobRequest {
	return economysim.DefaultStrategy{}.Work(v, rng)
}

func (contractScript) Revise(o *economysim.OutcomeView, rng *rand.Rand) economysim.Belief {
	return economysim.DefaultStrategy{}.Revise(o, rng)
}

func (s contractScript) Trade(v *economysim.AgentView, rng *rand.Rand, p *economysim.TradePlan) {
	// In its scripted ticks a broker places no order: the default's asks for its surplus
	// would reserve the fuel the offer needs.
	if !v.HasMerchant || v.Tick.Tick > s.until {
		economysim.DefaultStrategy{}.Trade(v, rng, p)
		return
	}
	switch v.Region {
	case s.from:
		p.Offers = append(p.Offers, economysim.DeliveryOffer{Mode: s.coach, To: s.to, Commodity: s.fuel, Quantity: s.lot, Limit: 100})
	case s.to:
		p.Takes = append(p.Takes, economysim.DeliveryTake{Commodity: s.fuel, Quantity: s.lot, Limit: 120})
	}
}

// scriptedMerchants returns the pack two with the merchants overlay and the option that
// registers the scripted merchant: brokers of the north offer 10 fuel to the south at
// 100 in ticks 0 and 1, and brokers of the south take 10 at 120, so the price of a
// contract is (100 + 120) / 2 = 110 and the coach (100 distance units at speed 30)
// takes 4 ticks: the contracts formed in ticks 0 and 1 settle in ticks 4 and 5.
func scriptedMerchants(t testing.TB) (economysim.Config, []economysim.Option) {
	t.Helper()
	cfg := loadValidPack(t, packMerchants("two")...)
	fuel := slices.IndexFunc(cfg.Commodities, func(c economysim.Commodity) bool { return c.ID == "fuel" })
	coach := slices.IndexFunc(cfg.Modules.Transport.Modes, func(m economysim.Mode) bool { return m.ID == "coach" })
	north := slices.IndexFunc(cfg.Regions, func(r economysim.Region) bool { return r.ID == "north" })
	south := slices.IndexFunc(cfg.Regions, func(r economysim.Region) bool { return r.ID == "south" })
	if fuel < 0 || coach < 0 || north < 0 || south < 0 {
		t.Fatalf("pack two has fuel %d, coach %d, north %d, south %d, want all", fuel, coach, north, south)
	}
	s := contractScript{from: economysim.RegionID(north), to: economysim.RegionID(south), fuel: economysim.CommodityID(fuel),
		coach: economysim.ModeID(coach), until: 1, lot: 10}
	return cfg, []economysim.Option{economysim.WithStrategy("merchant", s)}
}

// TestGolden_MerchantsAlone verifies the module-alone golden of the merchants module:
// pack two with the trade, transport, information and merchants overlays (testing.md
// 7.1.1), the merchant strategy, seed 1, 400 ticks, as a Step loop with the recorder and
// WithAudit, against testdata/golden/merchants/two.txt; the events of a Run against
// events-two.txt, which holds MerchantsPlanned and ContractRound; and the per-position
// state hash of ticks 0 to 29 against two.positions.txt, with P1.4, P3.3 and P3.3a in it.
// The cold pack's households die in the first ticks, so the merchants find no trip
// that pays and the golden holds the module's counts and no contract: the contracts are
// the next golden's. A run without the observer ends in the same snapshot.
//
// Requirement: FR-MER-062, FR-REF-025, FR-REF-026, FR-OBS-023, FR-WLD-004
//
// Guard: a behaviour change of the module moves the files; they are regenerated with
// -update only after the diff is understood (testing.md 7.2). DELIBERATE NEW GOLDEN:
// the files are new, no existing golden moves.
func TestGolden_MerchantsAlone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg := loadValidPack(t, packMerchants("two")...)
	checkMerchantsGolden(t, "two", cfg, strategy.Options(), goldenTicks, "MerchantsPlanned", "ContractRound")
}

// TestGolden_MerchantsContracts verifies the same goldens with the scripted merchant
// of scriptedMerchants, 40 ticks, so the files hold the contracts: the offers and takes
// of the round, ContractFormed with its ShipmentDispatched, the arrival and the
// ContractSettled of each contract, in testdata/golden/merchants/two-contracts.txt,
// events-two-contracts.txt and two-contracts.positions.txt.
//
// Requirement: FR-MER-062, FR-MER-056, FR-WLD-004
//
// Guard: without a contract the golden would not see the formation and the settlement;
// the events file must hold ContractFormed and ContractSettled. DELIBERATE NEW GOLDEN:
// the files are new, no existing golden moves.
func TestGolden_MerchantsContracts(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg, opts := scriptedMerchants(t)
	checkMerchantsGolden(t, "two-contracts", cfg, opts, 40, "ContractFormed", "ContractSettled", "ContractRound")
}

// checkMerchantsGolden runs the three goldens of one variant: the rendering of the
// run, the events of a Run and the per-position hash, against
// testdata/golden/merchants/<name>.txt, events-<name>.txt and <name>.positions.txt.
// wantEvents are event types the events file must hold, so that the golden sees the
// module.
func checkMerchantsGolden(t *testing.T, name string, cfg economysim.Config, extra []economysim.Option, ticks int64, wantEvents ...string) {
	t.Helper()
	spec := goldenSpec(extra)
	spec.Ticks = ticks
	res := runPack(t, cfg, spec)
	got := renderGolden(name, goldenSeed, ticks, samplesOnly(res, ticks), solveMerchants(t), cfg.Warnings())
	if want, err := os.ReadFile(moduleGoldenPath("merchants", name+".txt")); err == nil && !bytes.Equal(want, got) && !*updateFlag {
		t.Errorf("%s.txt differs from the golden: %s (testing.md 7.1.2)", name, merchantsPositionsNote(t, name, cfg, extra))
	}
	checkModuleGolden(t, "merchants", name+".txt", got)

	ev := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: ticks, Drive: driveRun(ticks), Observe: true, Extra: extra})
	step := runPack(t, cfg, runSpec{Seed: goldenSeed, Ticks: ticks, Drive: driveStep, Extra: extra})
	if !reflect.DeepEqual(ev.Final, step.Final) {
		t.Errorf("%s: final snapshot of Run(%d) observed differs from the bare Step loop's (seed %d)", name, ticks, goldenSeed)
	}
	events := renderEvents(ev)
	for _, w := range wantEvents {
		if !bytes.Contains(events, []byte(w)) {
			t.Errorf("%s: the events hold no %s, want the module's event in the golden", name, w)
		}
	}
	checkModuleGolden(t, "merchants", "events-"+name+".txt", events)

	lines := positionLines(t, cfg, goldenSeed, posTicks, extra...)
	if want := posTicks * merchantsPosLines; len(lines) != want {
		t.Errorf("%s: recorded %d position lines, want %d (%d positions per tick)", name, len(lines), want, merchantsPosLines)
	}
	for _, p := range []string{" P1.4 ", " P3.3 ", " P3.3a "} {
		if !slices.ContainsFunc(lines, func(l string) bool { return strings.Contains(l, p) }) {
			t.Errorf("%s: the positions file has no line for %q, want the module's three positions", name, strings.TrimSpace(p))
		}
	}
	checkModuleGolden(t, "merchants", name+".positions.txt", []byte(strings.Join(lines, "\n")+"\n"))
}

// merchantsPositionsNote replays the scripted run against the recorded positions file,
// so a failing golden says where the state first moved.
func merchantsPositionsNote(t testing.TB, name string, cfg economysim.Config, extra []economysim.Option) string {
	t.Helper()
	b, err := os.ReadFile(moduleGoldenPath("merchants", name+".positions.txt"))
	if err != nil {
		return "positions golden unreadable: " + err.Error()
	}
	want := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if note := divergenceNote(want, positionLines(t, cfg, goldenSeed, posTicks, extra...)); note != "" {
		return note
	}
	return fmt.Sprintf("positions of ticks 0 to %d agree with %s.positions.txt: the divergence is later", posTicks-1, name)
}

// TestGolden_OffUnchangedByMerchants verifies that the three files of the off set are
// byte for byte what they were before the merchants module (the sums of offGoldenSums),
// and that the pinned hash of the reference configuration did not move: a module that is
// off leaves no trace (architecture.md 2.10.5).
//
// Requirement: FR-MER-063, FR-REF-025
//
// Guard: a regenerated off golden would hide a non-interference failure of the module
// (testing.md 7.1.1).
func TestGolden_OffUnchangedByMerchants(t *testing.T) {
	t.Parallel()
	for name, want := range offGoldenSums {
		b, err := os.ReadFile(goldenPath(name))
		if err != nil {
			t.Fatalf("read off golden %s: %v", name, err)
		}
		sum := sha256.Sum256(b)
		if got := hex.EncodeToString(sum[:]); got != want {
			t.Errorf("sha256(testdata/golden/off/%s) = %s, want %s: the off set moved (testing.md 7.1.1)", name, got, want)
		}
	}
	if got := hexHash(referenceConfig()); got != pinnedReferenceHash {
		t.Errorf("HashConfig(referenceConfig()) = %s, want the pin %s", got, pinnedReferenceHash)
	}
}

// TestMerchants_DefaultStrategiesEqualOff verifies the second half of FR-MER-063: with
// the module on and every role on the default strategy (no merchant strategy, so no
// offer and no take), the run equals the run of the same pack with the module off and
// the merchant blocks removed, apart from the configuration hash and the module list.
//
// Requirement: FR-MER-063
//
// Guard: a module that is on and idle must not move a price, a count or a balance; the
// round, P1.4 and the filler run every tick and would show here first.
func TestMerchants_DefaultStrategiesEqualOff(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	on := loadValidPack(t, packMerchants("two")...)
	off := loadValidPack(t, packMerchants("two")...)
	for _, c := range []*economysim.Config{&on, &off} {
		for i := range c.Roles {
			c.Roles[i].Strategy = "default"
		}
	}
	off.Modules.Merchants = nil
	for i := range off.Roles {
		off.Roles[i].Merchant, off.Roles[i].MerchantFeeShare = nil, 0
	}
	spec := runSpec{Seed: goldenSeed, Ticks: 60, Drive: driveStep, Observe: true, Audit: true, Snapshots: snapEvery}
	a, b := runPack(t, on, spec), runPack(t, off, spec)
	for tick, sa := range a.Samples {
		sb := b.Samples[tick]
		sa.ConfigHash, sb.ConfigHash, sa.Modules, sb.Modules = [32]byte{}, [32]byte{}, nil, nil
		if !reflect.DeepEqual(sa, sb) {
			t.Fatalf("snapshot after tick %d (seed %d) differs between the module on with default strategies and off", tick, goldenSeed)
		}
	}
}
