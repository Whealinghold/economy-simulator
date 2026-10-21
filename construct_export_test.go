package economysim

import "github.com/Whealinghold/economy-sim/internal/ledger"

// This file is the test-only seam of the construction (architecture.md 10.2):
// it flattens what NewWorld built to plain values, so the external tests can
// check the ID space, the accounts and the markets of a world loaded from a pack
// (the internal tests cannot import config/, which imports this package).
// Nothing here is exported from production code.

// AgentRow is one household agent as the construction left it.
type AgentRow struct {
	ID      AgentID
	Account int
	Region  RegionID
	Role    RoleID
	Weight  int64
	Money   int64
	// Stock is the units held of each commodity, by CommodityID.
	Stock    []int64
	Beliefs  []Belief
	Target   int64
	Residues []int64
	Strategy int
}

// ConstructionRows returns the household agents of w in ascending ID, which is
// the order of the agent table.
func ConstructionRows(w *World) []AgentRow {
	var out []AgentRow
	for i := len(w.cfg.Regions); i < len(w.agents); i++ {
		a := &w.agents[i]
		v, err := w.ledger.Account(a.account)
		if err != nil {
			panic("test seam: " + err.Error())
		}
		row := AgentRow{
			ID: a.id, Account: a.account, Region: RegionID(v.Region), Role: a.role, Weight: a.weight,
			Money: v.Balance, Target: a.target, Residues: append([]int64(nil), a.residues...), Strategy: a.strategy,
		}
		for c := range w.cfg.Commodities {
			held, _, err := w.ledger.Holding(a.account, c)
			if err != nil {
				panic("test seam: " + err.Error())
			}
			row.Stock = append(row.Stock, held)
		}
		for _, b := range a.beliefs {
			row.Beliefs = append(row.Beliefs, Belief{Centre: Price(b.Centre), Width: b.Width})
		}
		out = append(out, row)
	}
	return out
}

// AccountPlan is the layout of the ledger's accounts: the number of boundary
// accounts (they come first), the account of the estate of each region, and the
// number of accounts altogether.
type AccountPlan struct {
	Boundary int
	Estates  []int
	Total    int
}

// ConstructionAccounts returns the account layout of w and checks the kind of
// each account on the way: the boundary accounts, then the estates, then the
// agents.
func ConstructionAccounts(w *World) AccountPlan {
	p := AccountPlan{Total: w.ledger.Len(), Estates: append([]int(nil), w.estates.accounts...)}
	for i := range p.Total {
		v, err := w.ledger.Account(i)
		if err != nil {
			panic("test seam: " + err.Error())
		}
		if v.Kind == ledger.KindBoundary {
			p.Boundary++
		}
	}
	return p
}

// MarketRecord is one market of the table.
type MarketRecord struct {
	Region    RegionID
	Commodity CommodityID
	Price     Price
	Age       int64
	Volume    Quantity
	WindowLen int
	BoundRun  int64
}

// MarketRecords returns the market table of w in table order.
func MarketRecords(w *World) []MarketRecord {
	out := make([]MarketRecord, len(w.markets.reports))
	for i := range out {
		r := &w.markets.reports[i]
		out[i] = MarketRecord{
			Region: RegionID(i / w.markets.commodities), Commodity: CommodityID(i % w.markets.commodities),
			Price: r.Price, Age: r.Age, Volume: r.Volume, WindowLen: w.markets.windows[i].Len(), BoundRun: w.markets.boundRun[i],
		}
	}
	return out
}

// WorldCounts returns the live counters of w: agents and households.
func WorldCounts(w *World) (agents, households int64) { return w.live.agents, w.live.households }

// LotRow is one lot of an agent's account.
type LotRow struct {
	Commodity  int
	Born       int64
	Durability int64
	Quantity   int64
}

// AgentLots returns the lots of agent id in the ledger's canonical order, which
// is by commodity, then Born, then durability.
func AgentLots(w *World, id AgentID) []LotRow {
	lots, err := w.ledger.AppendLots(nil, w.agents[id].account)
	if err != nil {
		panic("test seam: " + err.Error())
	}
	out := make([]LotRow, len(lots))
	for i, l := range lots {
		out[i] = LotRow{Commodity: l.Commodity, Born: l.Born, Durability: l.Durability, Quantity: l.Quantity}
	}
	return out
}

// AgentJobView is the job of an agent as construction left it.
type AgentJobView struct {
	Has        bool
	Recipe     RecipeID
	Remaining  int64
	Executions int64
	Principal  AgentID
	Yield      int64
	Pending    []int64
}

// AgentJob returns the job in progress of agent id.
func AgentJob(w *World, id AgentID) AgentJobView {
	a := &w.agents[id]
	return AgentJobView{
		Has: a.hasJob(), Recipe: a.job.recipe, Remaining: a.job.remaining, Executions: a.job.executions,
		Principal: a.job.principal, Yield: a.job.yield, Pending: append([]int64(nil), a.job.pending...),
	}
}

// PoolMoney returns the money of the estate account of a region, the regional
// pool.
func PoolMoney(w *World, r RegionID) Money { return w.estates.pool(w.ledger, r) }

// StartOffsets returns the virtual start offsets of a recipe in a region, the
// set the construction draws the offsets of the jobs from.
func StartOffsets(w *World, region RegionID, recipe RecipeID) []int64 {
	return virtualStartOffsets(&w.cfg, &w.yields, region, recipe)
}

// SeedWages returns the starting wage the world keeps for each region.
func SeedWages(w *World) []int64 { return append([]int64(nil), w.seedWage...) }

// ConstructionFlows returns the totals of the construction bucket and of the
// current tick, in the order the ledger gives them.
func ConstructionFlows(w *World) (bucket, tick []ledger.Flow) {
	return w.ledger.AppendFlows(nil, true), w.ledger.AppendFlows(nil, false)
}

// PCGState returns the marshalled state of the world's random source.
func PCGState(w *World) []byte {
	b, err := w.rnd.pcg.MarshalBinary()
	if err != nil {
		panic("test seam: " + err.Error())
	}
	return b
}
