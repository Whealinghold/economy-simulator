package otelobserver

import (
	"slices"

	economysim "github.com/Whealinghold/economy-sim"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// snapNames is the names of a snapshot as attributes. A snapshot carries its own
// names, so the method needs nothing of the world or of the adapter's table.
type snapNames struct {
	commodity, region, role, recipe, mode []string
	// population and migration say the module is on in the snapshot (Modules): a
	// gauge of a module that is off is absent, not zero.
	population, migration bool
}

// at is the attribute of place i of a name list, or the zero attribute (which
// gauge drops) outside it.
func at(key string, ss []string, i int) attribute.KeyValue {
	if i < 0 || i >= len(ss) {
		return attribute.KeyValue{}
	}
	return attribute.String(key, ss[i])
}

// gauge records v for an instrument at the attributes of a row; a nil attribute
// (an ID outside the names) drops the sample (spec-observability.md 6.6.4).
func (o *Observer) gauge(inst int, v int64, kv ...attribute.KeyValue) {
	for _, a := range kv {
		if !a.Valid() {
			return
		}
	}
	o.inst[inst].gauge.Record(bg, v, metric.WithAttributes(kv...))
}

// RecordSnapshot records the instruments that are read from a snapshot
// (spec-observability.md 6.2.2): each once per series, from the rows named. The
// consumer calls it between two steps; the world never does and the adapter never
// calls the world. An instrument is absent until the first call, and keeps its
// last value between calls. An ID outside the snapshot's names, a negative count
// where it makes no sense and a nil slice drop or skip the row; it never panics.
func (o *Observer) RecordSnapshot(s economysim.Snapshot) {
	n := snapNames{commodity: s.Names.Commodities, region: s.Names.Regions, role: s.Names.Roles, recipe: s.Names.Recipes,
		mode: s.Names.Modes, population: slices.Contains(s.Modules, "population"), migration: slices.Contains(s.Modules, "migration")}
	for _, m := range s.Markets {
		o.recordMarket(n, m)
	}
	o.recordJobs(n, s.Jobs)
	for _, r := range s.Regions {
		o.recordRegion(n, r)
	}
	for _, r := range s.Roles {
		o.recordRole(n, r)
	}
	if n.population && s.Population.HasReplacement {
		o.gauge(iPopulationReplacementRatio, s.Population.ReplacementRatioBP)
	}
	if slices.Contains(s.Modules, "transport") {
		o.recordTransport(n, s)
	}
}

// recordJobs records the executions in progress by recipe and region, and 0 for
// a pair that had a row in an earlier snapshot and has none now, so that an idle
// producer does not keep its last count (FR-OBS-097). The pairs seen are bounded
// by recipes and regions.
func (o *Observer) recordJobs(n snapNames, jobs []economysim.JobRow) {
	now := make(map[[2]int]int64, len(jobs))
	for _, j := range jobs {
		now[[2]int{int(j.Recipe), int(j.Region)}] += j.Executions
	}
	for k := range o.jobPairs {
		if _, ok := now[k]; !ok {
			now[k] = 0
		}
	}
	for k, v := range now {
		o.gauge(iRecipeInProgress, v, o.pair(keyRecipe, n.recipe, k[0], n.region, k[1])...)
	}
	o.jobPairs = now
}

// recordTransport records the shipments overdue by mode, 0 for a mode with none so
// that a cleared guard reads 0 and not a stale count, and the agents away.
func (o *Observer) recordTransport(n snapNames, s economysim.Snapshot) {
	overdue := make([]int64, len(n.mode))
	for _, sh := range s.Shipments {
		if m := int(sh.Mode); sh.TicksLeft <= 0 && m >= 0 && m < len(overdue) {
			overdue[m]++
		}
	}
	for m, v := range overdue {
		o.gauge(iTransportOverdue, v, at(keyMode, n.mode, m))
	}
	o.gauge(iTransportAway, int64(len(s.Away)))
}

// pair is the attributes of a name of one list and a region.
func (o *Observer) pair(key string, names []string, id int, regions []string, region int) []attribute.KeyValue {
	return []attribute.KeyValue{at(key, names, id), at(keyRegion, regions, region)}
}

func (o *Observer) recordMarket(n snapNames, m economysim.MarketRow) {
	kv := o.pair(keyCommodity, n.commodity, int(m.State.Commodity), n.region, int(m.State.Region))
	o.gauge(iProductionPending, int64(m.Pending), kv...)
	o.gauge(iEstateAtFloor, int64(m.EstateAtFloor), kv...)
	o.gauge(iLedgerLots, m.Lots, kv...)
	for i, held := range []int64{int64(m.HeldByAgents), int64(m.HeldByEstate)} {
		o.gauge(iMarketStock, held, append(kv[:2:2], attribute.String(keyHolder, closed[keyHolder][i]))...)
	}
}

func (o *Observer) recordRegion(n snapNames, r economysim.RegionRow) {
	region := at(keyRegion, n.region, int(r.Region))
	sector := func(i int, v economysim.Money) {
		o.gauge(iMoneySupply, int64(v), region, attribute.String(keySector, closed[keySector][i]))
	}
	sector(0, r.MoneyHouseholds)
	sector(1, r.MoneyPool)
	sector(2, r.MoneyContract)
	if n.population && r.HouseholdCap > 0 {
		o.gauge(iPopulationCap, r.HouseholdCap, region)
	}
	if n.migration {
		o.gauge(iMigrationPipeline, r.OnJourneyFrom, region, attribute.String(keyDirection, closed[keyDirection][0]))
		o.gauge(iMigrationPipeline, r.OnJourneyTo, region, attribute.String(keyDirection, closed[keyDirection][1]))
	}
}

func (o *Observer) recordRole(n snapNames, r economysim.RoleRow) {
	kv := o.pair(keyRole, n.role, int(r.Role), n.region, int(r.Region))
	o.gauge(iRoleAgents, r.Agents, kv...)
	o.gauge(iWelfareClamped, r.ClampedHouseholds, kv...)
	o.gauge(iHouseholdPrivationMax, r.MaxPrivation, kv...)
	if r.HasMeanRatio {
		o.gauge(iWelfareRatio, r.MeanRatioBP, kv...)
		o.gauge(iWelfareRatioHouseholds, r.RatioHouseholds, kv...)
	}
	if r.HasWage {
		o.gauge(iWageObserved, int64(r.WageObserved), kv...)
	}
	if r.HasExpected {
		o.gauge(iRoleExpectedRatio, r.ExpectedRatioBP, kv...)
		viable := int64(0)
		if r.Viable {
			viable = 1
		}
		o.gauge(iRoleViable, viable, kv...)
	}
	if n.population {
		o.gauge(iPopulationNaturalIncrease, r.NaturalIncreaseBP, kv...)
	}
	if r.Households > 0 && r.HeldByFloor >= 0 {
		o.gauge(iWelfareFloorHeld, ratioBP(r.HeldByFloor, r.Households), kv...)
	}
}
