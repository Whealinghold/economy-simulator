package economysim

import (
	"crypto/sha256"
	"encoding/binary"
)

// hashVersion is the version byte of the canonical encoding (spec-config.md
// 8.2.1). A change to the encoding itself (a type, the terminator, an id) is a
// new version, never an edit of this one.
const hashVersion = 0x01

// HashConfig returns the configuration hash of cfg: the SHA-256 of the
// canonical binary encoding of spec-config.md 8.2, version 1. Equal
// configurations give equal hashes on every platform, whatever the order in
// which entities were written (spec-config.md 3.4).
//
// The encoding is a struct as a run of (field id, value) pairs and a
// terminator. A field at its zero value is not written, a module that is off
// is not written, and a presentation-only field never is, so a field or a
// module added later at its off value leaves the hash of every existing
// configuration as it was. A non-nil pointer is always written, so a nil
// WarmStart and an empty one differ.
//
// HashConfig hashes cfg as given: it applies no option and validates nothing,
// so an invalid Config has a hash.
func HashConfig(cfg Config) [32]byte {
	return sha256.Sum256(encodeConfig(cfg.canonical()))
}

// encodeConfig returns the bytes of the encoding of a canonical Config. The
// field ids are the positions of the fields in spec-config.md 8.2.2, counted
// from 1 and not counting presentation-only fields; ids 8 (Edges) and 9
// (RegionPrices) sit between Regions and Groups. An id is never reused or moved.
func encodeConfig(c Config) []byte {
	e := &encoder{buf: []byte{hashVersion}}
	e.nested(1, func(e *encoder) { pairsWorld(e, c.World) })
	e.nested(2, func(e *encoder) { pairsCalendar(e, c.Calendar) })
	encodeList(e, 3, c.Commodities, pairsCommodity)
	encodeList(e, 4, c.Recipes, func(e *encoder, r Recipe) {
		pairsRecipe(e, r, c.Modules.Resources != nil)
	})
	encodeList(e, 5, c.Yields, pairsYield)
	encodeList(e, 6, c.Roles, func(e *encoder, r Role) {
		pairsRole(e, r, c.Modules.Information != nil, c.Modules.Migration != nil)
	})
	encodeList(e, 7, c.Regions, func(e *encoder, r Region) {
		pairsRegion(e, r, c.Modules.Estates != nil)
	})
	encodeList(e, 8, c.Edges, pairsEdge)
	encodeList(e, 9, c.RegionPrices, pairsRegionPrice)
	encodeList(e, 10, c.Groups, pairsGroup)
	if c.WarmStart != nil {
		e.id(11)
		pairsWarmStart(e, *c.WarmStart)
		e.end()
	}
	e.nested(12, func(e *encoder) { pairsModules(e, c.Modules) })
	e.end()
	return e.buf
}

// encoder appends the canonical encoding to buf. It holds no state beyond it.
type encoder struct{ buf []byte }

// id appends a field id, or the terminator when it is 0.
func (e *encoder) id(id uint16) { e.buf = binary.BigEndian.AppendUint16(e.buf, id) }

// end appends the terminator of a struct.
func (e *encoder) end() { e.id(0) }

// u64 appends 8 bytes, big-endian.
func (e *encoder) u64(v uint64) { e.buf = binary.BigEndian.AppendUint64(e.buf, v) }

// str appends a string: its length as 8 bytes, then its bytes.
func (e *encoder) str(s string) {
	e.u64(uint64(len(s)))
	e.buf = append(e.buf, s...)
}

// num writes an integer field unless it is 0.
func (e *encoder) num(id uint16, v int64) {
	if v != 0 {
		e.id(id)
		e.u64(uint64(v))
	}
}

// flag writes a bool field unless it is false.
func (e *encoder) flag(id uint16, v bool) {
	if v {
		e.id(id)
		e.buf = append(e.buf, 1)
	}
}

// text writes a string field unless it is empty.
func (e *encoder) text(id uint16, s string) {
	if s != "" {
		e.id(id)
		e.str(s)
	}
}

// texts writes a slice of strings: its count as 8 bytes, then each string.
// Nothing is written for an empty slice.
func (e *encoder) texts(id uint16, s []string) {
	if len(s) == 0 {
		return
	}
	e.id(id)
	e.u64(uint64(len(s)))
	for _, v := range s {
		e.str(v)
	}
}

// nested writes a struct-valued field: its id, the pairs and the terminator,
// unless every field of the struct is zero, in which case nothing is written.
// A struct is all zero exactly when it writes no pair.
func (e *encoder) nested(id uint16, pairs func(*encoder)) {
	start := len(e.buf)
	e.id(id)
	body := len(e.buf)
	pairs(e)
	if len(e.buf) == body {
		e.buf = e.buf[:start]
		return
	}
	e.end()
}

// encodeList writes a slice of structs: its count as 8 bytes, then each
// element as its pairs and a terminator, in the order of the slice. Nothing is
// written for an empty slice. An element that is all zero is still written, as
// the terminator alone.
func encodeList[T any](e *encoder, id uint16, s []T, pairs func(*encoder, T)) {
	if len(s) == 0 {
		return
	}
	e.id(id)
	e.u64(uint64(len(s)))
	for _, v := range s {
		pairs(e, v)
		e.end()
	}
}

// pairsWorld writes the pairs of a World: ids 1 to 45 in declaration order.
// DistanceLabel is presentation only and has no id.
func pairsWorld(e *encoder, w WorldParams) {
	e.num(1, w.Scale)
	e.num(2, w.MinAgents)
	e.num(3, w.BulkFactor)
	e.num(4, w.WelfareWindow)
	e.num(5, w.WindowCycles)
	encodeList(e, 6, w.Upkeep, pairsItem)
	e.num(7, w.CoverTicks)
	e.num(8, w.InputJobs)
	e.num(9, w.OutputCoverTicks)
	e.num(10, w.FavourFloor)
	e.num(11, w.ReserveTicks)
	e.num(12, w.SpendMaxShare)
	e.num(13, w.SpendHalfTicks)
	e.num(14, w.RestBand)
	e.num(15, w.ProduceHaltRatio)
	e.num(16, w.BasketTolerance)
	e.num(17, w.BeliefStep)
	e.num(18, w.BeliefAdjustProb)
	e.num(19, w.BeliefSignificance)
	e.num(20, w.BeliefStockLow)
	e.num(21, w.BeliefStockHigh)
	e.num(22, w.BeliefMaxAge)
	e.num(23, w.BeliefWidthInit)
	e.num(24, w.BeliefWidthMin)
	e.num(25, w.BeliefWidthMax)
	e.num(26, w.BeliefWidthStep)
	e.num(27, w.BeliefEvidenceShare)
	e.num(28, w.CostFloor)
	e.num(29, w.CostCeiling)
	e.num(30, w.RefBandLow)
	e.num(31, w.RefBandHigh)
	e.num(32, w.PriceWindow)
	e.text(33, w.Spawner)
	e.num(34, w.ReviewTicks)
	e.num(35, w.EntryShare)
	e.num(36, w.RoleHysteresis)
	e.num(37, w.RoleCooldown)
	e.num(38, w.EntryRatio)
	e.num(39, w.ChangePrivation)
	e.num(40, w.RemovePrivation)
	e.num(41, w.DeathStep)
	e.num(42, w.ExitShare)
	e.num(43, w.Renewal)
	e.num(44, w.EntryPoolShare)
	e.text(45, w.Clearing)
	e.num(46, w.AnchorHold)
}

// pairsCalendar writes the pairs of a Calendar: ids 1 to 8. Months and Seasons
// are ordered lists and are written as given.
func pairsCalendar(e *encoder, c Calendar) {
	e.num(1, c.HoursPerDay)
	e.num(2, c.DaysPerWeek)
	encodeList(e, 3, c.Months, pairsMonth)
	e.num(4, c.TicksPerDay)
	e.num(5, c.DaysPerTick)
	e.num(6, c.StartDay)
	encodeList(e, 7, c.Seasons, pairsSeason)
	e.num(8, c.SeasonStart)
}

// pairsMonth writes the pairs of a Month: ID 1, Days 2.
func pairsMonth(e *encoder, m Month) {
	e.text(1, m.ID)
	e.num(2, m.Days)
}

// pairsSeason writes the pairs of a Season: ID 1, Days 2.
func pairsSeason(e *encoder, s Season) {
	e.text(1, s.ID)
	e.num(2, s.Days)
}

// pairsCommodity writes the pairs of a Commodity: ids 1 to 8. Label and
// UnitLabel are presentation only and have no id.
func pairsCommodity(e *encoder, c Commodity) {
	e.text(1, c.ID)
	e.num(2, c.Mass)
	e.num(3, c.Bulk)
	e.num(4, c.ShelfLife)
	e.num(5, c.Durability)
	e.texts(6, c.Classes)
	e.flag(7, c.Raw)
	e.num(8, int64(c.ReferencePrice))
}

// pairsItem writes the pairs of an Item: Commodity 1, Quantity 2.
func pairsItem(e *encoder, i Item) {
	e.text(1, i.Commodity)
	e.num(2, int64(i.Quantity))
}

// pairsOutput writes the pairs of an Output: ids 1 to 4.
func pairsOutput(e *encoder, o Output) {
	e.text(1, o.Commodity)
	e.num(2, int64(o.Quantity))
	e.num(3, o.CostShare)
	e.flag(4, o.Discard)
}

// pairsTool writes the pairs of a Tool: Commodity 1, WearPerUse 2.
func pairsTool(e *encoder, t Tool) {
	e.text(1, t.Commodity)
	e.num(2, t.WearPerUse)
}

// pairsRecipe writes the pairs of a Recipe: ids 1 to 9. Label is presentation
// only and has no id. Draws (9) is written only while the resources module is on
// (spec-config.md 8.2.3), so a draw in a configuration that fails validation for
// want of the module changes no hash.
func pairsRecipe(e *encoder, r Recipe, resources bool) {
	e.text(1, r.ID)
	e.num(2, r.Duration)
	encodeList(e, 3, r.Inputs, pairsItem)
	encodeList(e, 4, r.Outputs, pairsOutput)
	encodeList(e, 5, r.Requires, pairsTool)
	e.flag(6, r.Sited)
	e.flag(7, r.TraineeEligible)
	e.num(8, r.TraineeYield)
	if resources {
		encodeList(e, 9, r.Draws, pairsDraw)
	}
}

// pairsDraw writes the pairs of a Draw: Resource 1, Quantity 2.
func pairsDraw(e *encoder, d Draw) {
	e.text(1, d.Resource)
	e.num(2, int64(d.Quantity))
}

// pairsYield writes the pairs of a YieldProfile: ids 1 to 5.
func pairsYield(e *encoder, y YieldProfile) {
	e.text(1, y.Region)
	e.text(2, y.Recipe)
	e.flag(3, y.HasAll)
	e.num(4, y.All)
	encodeList(e, 5, y.Seasons, pairsSeasonYield)
}

// pairsSeasonYield writes the pairs of a SeasonYield: Season 1, BP 2.
func pairsSeasonYield(e *encoder, s SeasonYield) {
	e.text(1, s.Season)
	e.num(2, s.BP)
}

// pairsRole writes the pairs of a Role: ids 1 to 25. Label is presentation only
// and has no id. Informed 19 is the information module's and is a pair only
// while it is on (informationOn) and the flag is true; Fixed 20 is the migration
// module's, on the same terms (migrationOn); MerchantFeeShare 21 and Merchant 22
// are the merchants', pairs only when the block is non-nil; Hireable 23 is a pair
// only when true and Employer 24 only when non-nil (the employment module's);
// Credit 25 only when non-nil (the credit module's).
func pairsRole(e *encoder, r Role, informationOn, migrationOn bool) {
	e.text(1, r.ID)
	e.texts(2, r.Recipes)
	encodeList(e, 3, r.Upkeep, pairsItem)
	e.texts(4, r.Trades)
	e.num(5, int64(r.StartMoney))
	encodeList(e, 6, r.StartStock, pairsItem)
	e.text(7, r.Strategy)
	e.flag(8, r.Unskilled)
	e.num(9, r.TargetRatio)
	e.num(10, r.TargetSpread)
	e.num(11, r.DefaultRatio)
	e.num(12, r.InKind)
	e.num(13, r.ExpectedWorkTicks)
	e.num(14, r.MinWorkShare)
	e.num(15, r.TrainingTicks)
	e.num(16, r.MaxTrainees)
	e.num(17, r.TeachingLoad)
	e.flag(18, r.Closed)
	e.flag(19, informationOn && r.Informed)
	e.flag(20, migrationOn && r.Fixed)
	if r.Merchant != nil {
		e.num(21, r.MerchantFeeShare)
		e.id(22) // written even when every field is zero: a non-nil block is a merchant role
		pairsMerchant(e, *r.Merchant)
		e.end()
	}
	e.flag(23, r.Hireable)
	if r.Employer != nil {
		e.id(24) // written even when PayrollBP is zero: a non-nil block is an employer role
		e.num(1, r.Employer.PayrollBP)
		e.end()
	}
	if r.Credit != nil {
		e.id(25) // written even when empty: a non-nil block is a credit role
		pairsCreditTerms(e, *r.Credit)
		e.end()
	}
	e.text(26, r.Needs)
	e.text(27, r.Ceiling)
}

// pairsRegion writes the pairs of a Region: ids 1 to 7. Label is presentation
// only and has no id. PoolRelease 6 and ReliefBelow 7 are the estates module's
// and are pairs only while it is on (estatesOn) and not zero.
func pairsRegion(e *encoder, r Region, estatesOn bool) {
	e.text(1, r.ID)
	e.num(2, int64(r.ReferenceBasket))
	encodeList(e, 3, r.ExtraUpkeep, pairsItem)
	e.num(4, r.CapitalReturn)
	e.num(5, r.EstateMarkdown)
	if estatesOn {
		e.num(6, r.PoolRelease)
		e.num(7, r.ReliefBelow)
	}
}

// pairsEdge writes the pairs of an Edge: From 1, To 2, Distance 3.
func pairsEdge(e *encoder, g Edge) {
	e.text(1, g.From)
	e.text(2, g.To)
	e.num(3, g.Distance)
}

// pairsRegionPrice writes the pairs of a RegionPrice: Region 1, Commodity 2,
// Price 3.
func pairsRegionPrice(e *encoder, p RegionPrice) {
	e.text(1, p.Region)
	e.text(2, p.Commodity)
	e.num(3, int64(p.Price))
}

// pairsGroup writes the pairs of a Group: ids 1 to 4.
func pairsGroup(e *encoder, g Group) {
	e.text(1, g.Region)
	e.text(2, g.Role)
	e.num(3, g.Households)
	e.num(4, g.Scale)
}

// pairsWarmStart writes the pairs of a WarmStart: Prices 1, Wages 2, Groups 3,
// Pools 4.
func pairsWarmStart(e *encoder, w WarmStart) {
	encodeList(e, 1, w.Prices, pairsSeedPrice)
	encodeList(e, 2, w.Wages, pairsSeedWage)
	encodeList(e, 3, w.Groups, pairsWarmGroup)
	encodeList(e, 4, w.Pools, pairsWarmPool)
}

// pairsSeedPrice writes the pairs of a SeedPrice: Region 1, Commodity 2,
// Price 3.
func pairsSeedPrice(e *encoder, s SeedPrice) {
	e.text(1, s.Region)
	e.text(2, s.Commodity)
	e.num(3, int64(s.Price))
}

// pairsSeedWage writes the pairs of a SeedWage: Region 1, Wage 2.
func pairsSeedWage(e *encoder, s SeedWage) {
	e.text(1, s.Region)
	e.num(2, int64(s.Wage))
}

// pairsWarmGroup writes the pairs of a WarmGroup: Region 1, Role 2, Money 3,
// Stock 4, Jobs 5.
func pairsWarmGroup(e *encoder, g WarmGroup) {
	e.text(1, g.Region)
	e.text(2, g.Role)
	e.num(3, int64(g.Money))
	encodeList(e, 4, g.Stock, pairsItem)
	encodeList(e, 5, g.Jobs, pairsJobSpec)
}

// pairsJobSpec writes the pairs of a JobSpec: Recipe 1, InProgressBP 2.
func pairsJobSpec(e *encoder, j JobSpec) {
	e.text(1, j.Recipe)
	e.num(2, j.InProgressBP)
}

// pairsWarmPool writes the pairs of a WarmPool: Region 1, Money 2.
func pairsWarmPool(e *encoder, p WarmPool) {
	e.text(1, p.Region)
	e.num(2, int64(p.Money))
}

// pairsModules writes the pairs of Modules in ascending id order. A module's
// change adds its pointer here at the place of its id in the Modules row of
// spec-config.md 3.1 (Demand 1 to Estates 25), not at the end, so a module with a
// smaller id than one already built moves no id. A non-nil pointer is written
// explicitly as its id, its pairs and a terminator, and not through nested, which
// writes nothing for an empty body: a module that is on and empty differs from
// one that is off (spec-config.md 8.2.1, 8.2.2).
func pairsModules(e *encoder, m Modules) {
	if m.Demand != nil {
		e.id(1)
		pairsDemand(e, *m.Demand)
		e.end()
	}
	if m.MoneySupply != nil {
		e.id(2)
		encodeList(e, 1, m.MoneySupply.Flows, pairsMoneyFlow)
		e.end()
	}
	if m.Population != nil {
		e.id(3)
		pairsPopulation(e, *m.Population)
		e.end()
	}
	if m.Transport != nil {
		e.id(4)
		pairsTransport(e, *m.Transport)
		e.end()
	}
	if m.Resources != nil {
		e.id(5)
		pairsResources(e, *m.Resources)
		e.end()
	}
	if m.Information != nil {
		e.id(6)
		pairsInformation(e, *m.Information)
		e.end()
	}
	if m.External != nil {
		e.id(7)
		pairsExternal(e, *m.External)
		e.end()
	}
	if m.Subsistence != nil {
		e.id(8)
		pairsSubsistence(e, *m.Subsistence)
		e.end()
	}
	pairsLaterModules(e, m)
}

// pairsLaterModules writes the pairs of the modules of ids 9, 10, 11, 12, 13 and 14 of
// Modules, after those pairsModules writes (a continuation for the Lizard limit).
func pairsLaterModules(e *encoder, m Modules) {
	if m.Migration != nil {
		e.id(9)
		pairsMigration(e, *m.Migration)
		e.end()
	}
	if m.Merchants != nil {
		e.id(10)
		e.num(1, m.Merchants.MaxOpen)
		e.end()
	}
	if m.Settlements != nil {
		e.id(11)
		pairsSettlements(e, *m.Settlements)
		e.end()
	}
	if m.Storage != nil {
		e.id(12)
		pairsStorage(e, *m.Storage)
		e.end()
	}
	if m.Employment != nil {
		e.id(13)
		e.end()
	}
	if m.Tenure != nil {
		e.id(14)
		pairsTenure(e, *m.Tenure)
		e.end()
	}
	pairsLastModules(e, m)
}

// pairsLastModules writes the pairs of the modules of ids 15, 16, 19, 21, 24 and 25 of Modules,
// after those pairsLaterModules writes (a continuation for the Lizard limit).
func pairsLastModules(e *encoder, m Modules) {
	if m.Credit != nil {
		e.id(15)
		pairsCredit(e, *m.Credit)
		e.end()
	}
	if m.Risk != nil {
		e.id(16)
		pairsRisk(e, *m.Risk)
		e.end()
	}
	if m.Polity != nil {
		e.id(19)
		pairsPolity(e, *m.Polity)
		e.end()
	}
	if m.Institutions != nil {
		e.id(21)
		pairsInstitutions(e, *m.Institutions)
		e.end()
	}
	if m.Technology != nil {
		e.id(24)
		pairsTechnology(e, *m.Technology)
		e.end()
	}
	if m.Estates != nil {
		e.id(25)
		e.end()
	}
}

// pairsInstitutions writes the pairs of an Institutions: Bodies 1, Rights 2, Controls
// 3, RateLimits 4, OwnUseBound 5. A Body has ID 1, Roles 2, Regions 3, Jurisdictions 4
// (each the sorted IDs), Places 5, MaxTrainees 6 and Needs 7; a Right has ID 1, Body 2,
// Activity 3, Subject 4, Region 5, Jurisdiction 6 and Who 7; a Control has ID 1,
// Commodity 2, Region 3, Jurisdiction 4, CapBP 5 and FloorBP 6; a RateLimit has ID 1,
// Region 2, Jurisdiction 3 and CapBP 4 (Label is presentation only and has no id). The
// four lists are in ID order. The legal levels and the right suspensions of the slots
// are state and are not hashed (spec-institutions.md 7.1, FR-ORG-038).
func pairsInstitutions(e *encoder, m Institutions) {
	encodeList(e, 1, m.Bodies, func(e *encoder, b Body) {
		e.text(1, b.ID)
		e.texts(2, b.Roles)
		e.texts(3, b.Regions)
		e.texts(4, b.Jurisdictions)
		e.num(5, b.Places)
		e.num(6, b.MaxTrainees)
		e.text(7, b.Needs)
	})
	encodeList(e, 2, m.Rights, func(e *encoder, r Right) {
		e.text(1, r.ID)
		e.text(2, r.Body)
		e.text(3, r.Activity)
		e.text(4, r.Subject)
		e.text(5, r.Region)
		e.text(6, r.Jurisdiction)
		e.text(7, r.Who)
	})
	encodeList(e, 3, m.Controls, func(e *encoder, c Control) {
		e.text(1, c.ID)
		e.text(2, c.Commodity)
		e.text(3, c.Region)
		e.text(4, c.Jurisdiction)
		e.num(5, c.CapBP)
		e.num(6, c.FloorBP)
	})
	encodeList(e, 4, m.RateLimits, func(e *encoder, l RateLimit) {
		e.text(1, l.ID)
		e.text(2, l.Region)
		e.text(3, l.Jurisdiction)
		e.num(4, l.CapBP)
	})
	e.flag(5, m.OwnUseBound)
}

// pairsTechnology writes the pairs of a Technology: Fields 1, in ID order. A
// TechField has ID 1, Recipes 2 (the recipe IDs, sorted), CapBP 3, Base 4,
// DoublingBP 5, TrendBP 6, DiscoveryBP 7, StepBP 8, CatchUpBP 9 and Reach 10
// (Label is presentation only and has no id, spec-technology.md 12.3).
func pairsTechnology(e *encoder, t Technology) {
	encodeList(e, 1, t.Fields, func(e *encoder, f TechField) {
		e.text(1, f.ID)
		e.texts(2, f.Recipes)
		e.num(3, f.CapBP)
		e.num(4, f.Base)
		e.num(5, f.DoublingBP)
		e.num(6, f.TrendBP)
		e.num(7, f.DiscoveryBP)
		e.num(8, f.StepBP)
		e.num(9, f.CatchUpBP)
		e.num(10, f.Reach)
	})
}

// pairsStorage writes the pairs of a Storage: Base 1, Roles 2, Classes 3. A
// Capacity has Mass 1 and Bulk 2; a RoleRoom has Role 1, Base 2 and Stores 3, the
// class IDs in class order; a StoreClass has ID 1, Mass 2, Bulk 3 and Needs 4
// (Label is presentation only and has no id). A Base is a pointer: it is written
// when it is non-nil even if both fields are zero, so an unlimited role and one with
// a zero base room differ. The lists are in canonical order, the rows by role and
// the classes by ID.
func pairsStorage(e *encoder, s Storage) {
	pairsBase(e, 1, s.Base)
	encodeList(e, 2, s.Roles, func(e *encoder, r RoleRoom) {
		e.text(1, r.Role)
		pairsBase(e, 2, r.Base)
		e.texts(3, r.Stores)
	})
	encodeList(e, 3, s.Classes, func(e *encoder, c StoreClass) {
		e.text(1, c.ID)
		e.num(2, c.Mass)
		e.num(3, c.Bulk)
		e.text(4, c.Needs)
	})
}

// pairsBase writes a Capacity pointer under id: nothing for nil, else the id, the
// pairs Mass 1 and Bulk 2 and the terminator, even when both are zero.
func pairsBase(e *encoder, id uint16, c *Capacity) {
	if c == nil {
		return
	}
	e.id(id)
	e.num(1, c.Mass)
	e.num(2, c.Bulk)
	e.end()
}

// pairsSettlements writes the pairs of a Settlements: Classes 1, BandBP 2, Dwell 3,
// Regions 4. A SettlementClass has ID 1, Enter 2, Flags 3 and Limits 4 (Label is
// presentation only and has no id); a ClassLimit has Name 1 and Value 2; a
// RegionSettlement has Region 1, ThresholdBP 2 and Floor 3. The classes are written
// in canonical order, by Enter, each class's flags and limits by name and the rows
// by region, so two configurations of equal content hash alike.
func pairsSettlements(e *encoder, s Settlements) {
	encodeList(e, 1, s.Classes, func(e *encoder, c SettlementClass) {
		e.text(1, c.ID)
		e.num(2, c.Enter)
		e.texts(3, c.Flags)
		encodeList(e, 4, c.Limits, func(e *encoder, l ClassLimit) {
			e.text(1, l.Name)
			e.num(2, l.Value)
		})
	})
	e.num(2, s.BandBP)
	e.num(3, s.Dwell)
	encodeList(e, 4, s.Regions, func(e *encoder, r RegionSettlement) {
		e.text(1, r.Region)
		e.num(2, r.ThresholdBP)
		e.text(3, r.Floor)
	})
}

// pairsCredit writes the pairs of a Credit: MinPrincipal 1, ShieldTicks 2,
// WriteOffTicks 3, MaxClaims 4 (spec-config.md 8.2.2). The rate scales of
// SetRateScales are state and are not hashed.
func pairsCredit(e *encoder, c Credit) {
	e.num(1, c.MinPrincipal)
	e.num(2, c.ShieldTicks)
	e.num(3, c.WriteOffTicks)
	e.num(4, c.MaxClaims)
}

// pairsCreditTerms writes the pairs of a CreditTerms: Lend 1 and Borrow 2, each
// written when non-nil even if every field is zero, so a lender of rate 0 and a
// role with no block differ, and so do a lender and a borrower. A LendTerms has
// RateBP 1, MaxTermTicks 2, KeepTicks 3, ShareBP 4; a BorrowTerms has MaxRateBP 1,
// MaxDebtTicks 2, SlackTicks 3.
func pairsCreditTerms(e *encoder, t CreditTerms) {
	if l := t.Lend; l != nil {
		e.id(1)
		e.num(1, l.RateBP)
		e.num(2, l.MaxTermTicks)
		e.num(3, l.KeepTicks)
		e.num(4, l.ShareBP)
		e.end()
	}
	if b := t.Borrow; b != nil {
		e.id(2)
		e.num(1, b.MaxRateBP)
		e.num(2, b.MaxDebtTicks)
		e.num(3, b.SlackTicks)
		e.end()
	}
}

// pairsTenure writes the pairs of a TenureConfig: Terms 1, Holdings 2, RentStepBP 3,
// RentFloorBP 4, RentCeilingBP 5, OwnUseExempt 6, MaxHolderShareBP 7 (spec-config.md
// 8.2.2). A TenureTerm has Region 1, Resource 2, HasRent 3, Rent 4, HasClosed 5 and
// Closed 6; a Holding has Region 1, Resource 2, Holder 3 and ShareBP 4; a HolderRef
// has Kind 1 and Role 2. The lists are written in canonical order, and the pointer
// is written even when the module is empty, so a module that is on differs from one
// that is off.
func pairsTenure(e *encoder, t TenureConfig) {
	encodeList(e, 1, t.Terms, func(e *encoder, r TenureTerm) {
		e.text(1, r.Region)
		e.text(2, r.Resource)
		e.flag(3, r.HasRent)
		e.num(4, r.Rent)
		e.flag(5, r.HasClosed)
		e.flag(6, r.Closed)
	})
	encodeList(e, 2, t.Holdings, func(e *encoder, h Holding) {
		e.text(1, h.Region)
		e.text(2, h.Resource)
		e.nested(3, func(e *encoder) {
			e.text(1, h.Holder.Kind)
			e.text(2, h.Holder.Role)
		})
		e.num(4, h.ShareBP)
	})
	e.num(3, t.RentStepBP)
	e.num(4, t.RentFloorBP)
	e.num(5, t.RentCeilingBP)
	e.flag(6, t.OwnUseExempt)
	e.num(7, t.MaxHolderShareBP)
}

// pairsRisk writes the pairs of a Risk: Routes 1, Stores 2 (spec-config.md 8.2.2). A
// RouteRisk has From 1, To 2, Mode 3, StrikeBP 4, SeverityBP 5 and Common 6; a
// StoreRisk has Region 1, Class 2, StrikeBP 3, SeverityBP 4 and Common 5. The lists
// are written in canonical order, and the pointer is written even when the module is
// empty, so a module that is on differs from one that is off.
func pairsRisk(e *encoder, r Risk) {
	encodeList(e, 1, r.Routes, func(e *encoder, row RouteRisk) {
		e.text(1, row.From)
		e.text(2, row.To)
		e.text(3, row.Mode)
		e.num(4, row.StrikeBP)
		e.num(5, row.SeverityBP)
		e.flag(6, row.Common)
	})
	encodeList(e, 2, r.Stores, func(e *encoder, row StoreRisk) {
		e.text(1, row.Region)
		e.text(2, row.Class)
		e.num(3, row.StrikeBP)
		e.num(4, row.SeverityBP)
		e.flag(5, row.Common)
	})
}

// pairsPolity writes the pairs of a PolityConfig: Jurisdictions 1, Members 2.
// Jurisdiction has ID 1, Parent 2, Seat 3, StartMoney 4 (Label is presentation
// only and has no id); Member has Region 1, Jurisdiction 2. The lists are written
// in canonical order, the jurisdictions by ID and the members by region. The
// pointer is written even when empty, so a module that is on and empty differs
// from one that is off.
func pairsPolity(e *encoder, p PolityConfig) {
	encodeList(e, 1, p.Jurisdictions, func(e *encoder, j Jurisdiction) {
		e.text(1, j.ID)
		e.text(2, j.Parent)
		e.text(3, j.Seat)
		e.num(4, int64(j.StartMoney))
	})
	encodeList(e, 2, p.Members, func(e *encoder, m Member) {
		e.text(1, m.Region)
		e.text(2, m.Jurisdiction)
	})
}

// pairsMoneyFlow writes the pairs of a MoneyFlow: ID 1, Kind 2, Target 3, Amount
// 4, Level 5, RateBP 6, Exempt 7, Per 8, Start 9, Until 10, Limit 11, NotIncome 12.
// Label is presentation only and has no id.
func pairsMoneyFlow(e *encoder, f MoneyFlow) {
	e.text(1, f.ID)
	e.text(2, f.Kind)
	e.nested(3, func(e *encoder) {
		e.text(1, f.Target.Kind)
		e.text(2, f.Target.Region)
		e.text(3, f.Target.Role)
		e.text(4, f.Target.Store)
	})
	e.num(4, f.Amount)
	e.num(5, f.Level)
	e.num(6, f.RateBP)
	e.num(7, f.Exempt)
	e.text(8, f.Per)
	e.num(9, f.Start)
	e.num(10, f.Until)
	e.num(11, f.Limit)
	e.flag(12, f.NotIncome)
}

// pairsDemand writes the pairs of a Demand: Wants 1, Tastes 2, RegionTastes 3.
func pairsDemand(e *encoder, d Demand) {
	encodeList(e, 1, d.Wants, pairsWant)
	encodeList(e, 2, d.Tastes, pairsTaste)
	encodeList(e, 3, d.RegionTastes, pairsRegionTaste)
}

// pairsWant writes the pairs of a Want: ids 1 to 8. Label is presentation only
// and has no id.
func pairsWant(e *encoder, w Want) {
	e.text(1, w.ID)
	encodeList(e, 2, w.Items, pairsWantItem)
	e.num(3, w.Quantity)
	e.num(4, w.Period)
	e.num(5, w.Rank)
	e.num(6, w.Elasticity)
	e.num(7, w.Ceiling)
	e.num(8, w.Substitution)
}

// pairsWantItem writes the pairs of a WantItem: Commodity 1, Weight 2.
func pairsWantItem(e *encoder, i WantItem) {
	e.text(1, i.Commodity)
	e.num(2, i.Weight)
}

// pairsTaste writes the pairs of a Taste: Role 1, Want 2, Scale 3.
func pairsTaste(e *encoder, t Taste) {
	e.text(1, t.Role)
	e.text(2, t.Want)
	e.num(3, t.Scale)
}

// pairsRegionTaste writes the pairs of a RegionTaste: Region 1, Want 2, Scale 3.
func pairsRegionTaste(e *encoder, t RegionTaste) {
	e.text(1, t.Region)
	e.text(2, t.Want)
	e.num(3, t.Scale)
}

// pairsPopulation writes the pairs of a Population: BirthSchedule 1,
// DeathSchedule 2 (Knot: Ratio 1, Rate 2), MinHouseholds 3, BirthStake 4, Regions
// 5 (RegionPopulation: Region 1, BirthScale 2, DeathScale 3, HouseholdCap 4). The
// schedules are in knot order and the rows by region.
func pairsPopulation(e *encoder, p Population) {
	encodeList(e, 1, p.BirthSchedule, pairsKnot)
	encodeList(e, 2, p.DeathSchedule, pairsKnot)
	e.num(3, p.MinHouseholds)
	e.num(4, p.BirthStake)
	encodeList(e, 5, p.Regions, func(e *encoder, r RegionPopulation) {
		e.text(1, r.Region)
		e.num(2, r.BirthScale)
		e.num(3, r.DeathScale)
		e.num(4, r.HouseholdCap)
	})
}

// pairsKnot writes the pairs of a Knot: Ratio 1, Rate 2.
func pairsKnot(e *encoder, k Knot) {
	e.num(1, k.Ratio)
	e.num(2, k.Rate)
}

// pairsSubsistence writes the pairs of a SubsistenceConfig: Recipes 1, Roles 2
// (Role 1, Recipes 2), EnterTicks 3, ExitTicks 4, RetryJobs 5.
func pairsSubsistence(e *encoder, s SubsistenceConfig) {
	e.texts(1, s.Recipes)
	encodeList(e, 2, s.Roles, func(e *encoder, r RoleOwnUse) {
		e.text(1, r.Role)
		e.texts(2, r.Recipes)
	})
	e.num(3, s.EnterTicks)
	e.num(4, s.ExitTicks)
	e.num(5, s.RetryJobs)
}

// pairsTransport writes the pairs of a Transport: Modes 1 (Mode: ID 1, Speed 2,
// UnitMass 3, UnitBulk 4, Crew 5, Vehicle 6, WearPerTick 7, Feed 8 (an Item: the
// ids of pairsItem), Haul 9, HaulSpan 10). Label is presentation only and has no
// id. The modes are in ID order and the feed in commodity order.
func pairsTransport(e *encoder, t Transport) {
	encodeList(e, 1, t.Modes, func(e *encoder, m Mode) {
		e.text(1, m.ID)
		e.num(2, m.Speed)
		e.num(3, m.UnitMass)
		e.num(4, m.UnitBulk)
		e.num(5, m.Crew)
		e.text(6, m.Vehicle)
		e.num(7, m.WearPerTick)
		encodeList(e, 8, m.Feed, pairsItem)
		e.text(9, m.Haul)
		e.num(10, m.HaulSpan)
	})
}

// pairsInformation writes the pairs of an Information: InfoSpeed 1,
// RumourErrorBP 2, RumourMaxAge 3.
func pairsInformation(e *encoder, i Information) {
	e.num(1, i.InfoSpeed)
	e.num(2, i.RumourErrorBP)
	e.num(3, i.RumourMaxAge)
}

// pairsMigration writes the pairs of a Migration: MoveSpeed 1, MoveReviewTicks
// 2, MoveBelow 3, MoveGain 4, MoveWidth 5, MoveCooldown 6, MoveHorizon 7,
// MoveOutShare 8, MoveInShare 9, Regions 10 (RegionMove: Region 1, OutShare 2,
// InShare 3).
func pairsMigration(e *encoder, m Migration) {
	e.num(1, m.MoveSpeed)
	e.num(2, m.MoveReviewTicks)
	e.num(3, m.MoveBelow)
	e.num(4, m.MoveGain)
	e.num(5, m.MoveWidth)
	e.num(6, m.MoveCooldown)
	e.num(7, m.MoveHorizon)
	e.num(8, m.MoveOutShare)
	e.num(9, m.MoveInShare)
	encodeList(e, 10, m.Regions, func(e *encoder, r RegionMove) {
		e.text(1, r.Region)
		e.num(2, r.OutShare)
		e.num(3, r.InShare)
	})
}

// pairsMerchant writes the pairs of a Merchant: Modes 1 (by ID), LotUnits 2,
// FeeShareSpreadBP 3, MaxSourceAge 4, TimeCostBP 5, ConsigneeCutBP 6, Consigns 7,
// TakeMarginBP 8, TakeUnits 9.
func pairsMerchant(e *encoder, m Merchant) {
	e.texts(1, m.Modes)
	e.num(2, m.LotUnits)
	e.num(3, m.FeeShareSpreadBP)
	e.num(4, m.MaxSourceAge)
	e.num(5, m.TimeCostBP)
	e.num(6, m.ConsigneeCutBP)
	e.flag(7, m.Consigns)
	e.num(8, m.TakeMarginBP)
	e.num(9, int64(m.TakeUnits))
}

// pairsResources writes the pairs of a Resources: Resources 1, Endowments 2,
// HorizonYears 3.
func pairsResources(e *encoder, r Resources) {
	encodeList(e, 1, r.Resources, pairsResource)
	encodeList(e, 2, r.Endowments, pairsEndowment)
	e.num(3, r.HorizonYears)
}

// pairsResource writes the pairs of a Resource: ID 1, Shape 2, RatePerYear 3,
// RateBP 4, Refuge 5, Capacity 6, InitialBP 7, PlanShareBP 8, Abundance 9,
// Seasons 10. Label is presentation only and has no id.
func pairsResource(e *encoder, r Resource) {
	e.text(1, r.ID)
	e.text(2, r.Shape)
	e.num(3, r.RatePerYear)
	e.num(4, r.RateBP)
	e.num(5, r.Refuge)
	e.num(6, r.Capacity)
	e.num(7, r.InitialBP)
	e.num(8, r.PlanShareBP)
	encodeList(e, 9, r.Abundance, func(e *encoder, k AbundanceKnot) {
		e.num(1, k.Share)
		e.num(2, k.Factor)
	})
	encodeList(e, 10, r.Seasons, func(e *encoder, s SeasonRegen) {
		e.text(1, s.Season)
		e.num(2, s.BP)
	})
}

// pairsEndowment writes the pairs of an Endowment: Region 1, Resource 2,
// HasCapacity 3, Capacity 4, HasInitial 5, Initial 6, HasRegenScale 7,
// RegenScaleBP 8.
func pairsEndowment(e *encoder, r Endowment) {
	e.text(1, r.Region)
	e.text(2, r.Resource)
	e.flag(3, r.HasCapacity)
	e.num(4, r.Capacity)
	e.flag(5, r.HasInitial)
	e.num(6, r.Initial)
	e.flag(7, r.HasRegenScale)
	e.num(8, r.RegenScaleBP)
}

// pairsExternal writes the pairs of an External: Nodes 1. A node's Label is
// presentation only and has no id; the runtime state (drift, shock, carries,
// totals) is not configuration and is not written (spec-config.md 8.2.1).
func pairsExternal(e *encoder, x External) {
	encodeList(e, 1, x.Nodes, pairsOutsideNode)
}

// pairsOutsideNode writes the pairs of an OutsideNode: Region 1, Lines 2, Gates 3.
func pairsOutsideNode(e *encoder, n OutsideNode) {
	e.text(1, n.Region)
	encodeList(e, 2, n.Lines, pairsOutsideLine)
	encodeList(e, 3, n.Gates, pairsOutsideGate)
}

// pairsOutsideLine writes the pairs of an OutsideLine: Commodity 1, Base 2,
// PriceBySeason 3, TrendBP 4, FloorBP 5, CeilingBP 6, WalkStepBP 7, WalkRevertBP
// 8, WalkBoundBP 9, Sell 10, Buy 11.
func pairsOutsideLine(e *encoder, l OutsideLine) {
	e.text(1, l.Commodity)
	e.num(2, int64(l.Base))
	encodeList(e, 3, l.PriceBySeason, pairsSeasonFactor)
	e.num(4, l.TrendBP)
	e.num(5, l.FloorBP)
	e.num(6, l.CeilingBP)
	e.num(7, l.WalkStepBP)
	e.num(8, l.WalkRevertBP)
	e.num(9, l.WalkBoundBP)
	e.nested(10, func(e *encoder) { pairsOutsideSide(e, l.Sell) })
	e.nested(11, func(e *encoder) { pairsOutsideSide(e, l.Buy) })
}

// pairsOutsideSide writes the pairs of an OutsideSide: MarginBP 1, Capacity 2,
// Window 3, TotalUnits 4, TotalValue 5.
func pairsOutsideSide(e *encoder, s OutsideSide) {
	e.num(1, s.MarginBP)
	e.num(2, s.Capacity)
	encodeList(e, 3, s.Window, pairsSeasonFactor)
	e.num(4, s.TotalUnits)
	e.num(5, s.TotalValue)
}

// pairsSeasonFactor writes the pairs of a SeasonFactor: Season 1, BP 2.
func pairsSeasonFactor(e *encoder, f SeasonFactor) {
	e.text(1, f.Season)
	e.num(2, f.BP)
}

// pairsOutsideGate writes the pairs of an OutsideGate: Region 1, ScaleBP 2.
func pairsOutsideGate(e *encoder, g OutsideGate) {
	e.text(1, g.Region)
	e.num(2, g.ScaleBP)
}
