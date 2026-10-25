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
	encodeList(e, 4, c.Recipes, pairsRecipe)
	encodeList(e, 5, c.Yields, pairsYield)
	encodeList(e, 6, c.Roles, pairsRole)
	encodeList(e, 7, c.Regions, pairsRegion)
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

// pairsRecipe writes the pairs of a Recipe: ids 1 to 8. Label is presentation
// only and has no id.
func pairsRecipe(e *encoder, r Recipe) {
	e.text(1, r.ID)
	e.num(2, r.Duration)
	encodeList(e, 3, r.Inputs, pairsItem)
	encodeList(e, 4, r.Outputs, pairsOutput)
	encodeList(e, 5, r.Requires, pairsTool)
	e.flag(6, r.Sited)
	e.flag(7, r.TraineeEligible)
	e.num(8, r.TraineeYield)
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

// pairsRole writes the pairs of a Role: ids 1 to 18. Label is presentation only
// and has no id.
func pairsRole(e *encoder, r Role) {
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
}

// pairsRegion writes the pairs of a Region: ids 1 to 5. Label is presentation
// only and has no id.
func pairsRegion(e *encoder, r Region) {
	e.text(1, r.ID)
	e.num(2, int64(r.ReferenceBasket))
	encodeList(e, 3, r.ExtraUpkeep, pairsItem)
	e.num(4, r.CapitalReturn)
	e.num(5, r.EstateMarkdown)
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

// pairsModules writes the pairs of Modules. No module is built at Stage 0, so
// there is none. A module's change adds its pointer here under the id of its
// position in the Modules row of spec-config.md 3.1 (Demand 1 to Estates 25),
// writing a non-nil pointer as its id, its pairs and a terminator, and never a
// nil one.
func pairsModules(*encoder, Modules) {}
