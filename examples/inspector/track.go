package main

import (
	"fmt"
	"sort"
	"strings"

	economysim "github.com/Whealinghold/economy-sim"
)

// marketKey names a (region, commodity) market.
type marketKey struct {
	region    economysim.RegionID
	commodity economysim.CommodityID
}

// tracker is the observer of a run. It keeps the counters of the current
// window, the last clearing of every market and, per agent, the money and
// the role, which the snapshot does not give by role.
type tracker struct {
	names   economysim.Names
	counts  map[string]int64
	cleared map[marketKey]economysim.MarketCleared
	// prices is the published price of each market, one entry per tick.
	prices map[marketKey][]int64

	role  map[economysim.AgentID]economysim.RoleID
	money map[economysim.AgentID]int64

	// regions is the number of regions: a trade seller below it is an estate.
	regions int
}

func newTracker() *tracker {
	return &tracker{
		counts:  map[string]int64{},
		cleared: map[marketKey]economysim.MarketCleared{},
		prices:  map[marketKey][]int64{},
		role:    map[economysim.AgentID]economysim.RoleID{},
		money:   map[economysim.AgentID]int64{},
	}
}

// Observe implements economysim.Observer.
func (t *tracker) Observe(e economysim.Event) {
	n := t.names
	switch v := e.(type) {
	case economysim.WorldCreated:
		t.names = v.Names
		t.regions = len(v.Names.Regions)
	case economysim.AgentSpawned:
		t.role[v.Agent] = v.Role
		t.money[v.Agent] += int64(v.StartMoney)
	case economysim.RoleChanged:
		t.role[v.Agent] = v.To
		t.counts[fmt.Sprintf("move.%s>%s", n.Roles[v.From], n.Roles[v.To])] += v.Weight
	case economysim.WeightChanged:
		if v.To < v.From {
			t.money[v.Agent] -= int64(v.Money)
			t.counts[fmt.Sprintf("cut.%s.%s", n.Roles[v.Role], v.Cause)] += v.From - v.To
		} else {
			t.money[v.Agent] += int64(v.Money)
		}
	case economysim.AgentBankrupted:
		delete(t.money, v.Agent)
		delete(t.role, v.Agent)
		t.counts[fmt.Sprintf("dead.%s.%s", n.Roles[v.Role], v.Cause)] += v.Weight
	case economysim.TradeExecuted:
		t.trade(v)
	case economysim.MarketCleared:
		k := marketKey{v.Region, v.Commodity}
		t.cleared[k] = v
		t.prices[k] = append(t.prices[k], int64(v.Price))
	case economysim.RecipeExecuted:
		t.counts["exec."+n.Recipes[v.Recipe]] += v.Executions
	case economysim.RecipeAbandoned:
		t.counts[fmt.Sprintf("aband.%s.%s", n.Recipes[v.Recipe], v.Reason)] += v.Executions
	case economysim.RecipeRefused:
		r := n.Recipes[v.Recipe]
		t.counts["refuse."+r+".no_tool"] += v.NoTool
		t.counts["refuse."+r+".no_input"] += v.NoInput
		t.counts["refuse."+r+".no_site"] += v.NoSite
		t.counts["refuse."+r+".zero_yield"] += v.ZeroYield
	case economysim.UpkeepShortfall:
		for _, r := range v.Short {
			t.counts["short."+n.Roles[v.Role]+"."+n.Commodities[r.Commodity]] += r.Units
		}
	case economysim.EntryLimited:
		t.counts["entry.entered"] += v.Entered
		t.counts["entry.removed"] += v.Removed
		t.counts["entry.unfunded"] += v.Unfunded
		t.counts["entry.no_viable_role"] += v.NoViableRole
		t.counts["entry.stranded"] += v.Stranded
	}
}

// trade books one trade: the money moves between the two agents, and the
// window counts the volume and the value by commodity and the revenue and
// spend by role.
func (t *tracker) trade(v economysim.TradeExecuted) {
	val := int64(v.Price) * int64(v.Quantity)
	c := t.names.Commodities[v.Commodity]
	t.counts["vol."+c] += int64(v.Quantity)
	t.counts["val."+c] += val
	if r, ok := t.role[v.Buyer]; ok {
		t.money[v.Buyer] -= val
		t.counts["spend."+t.names.Roles[r]] += val
	}
	if r, ok := t.role[v.Seller]; ok { // an estate is not in the role table
		t.money[v.Seller] += val
		t.counts["rev."+t.names.Roles[r]] += val
	}
}

// roleMoney returns the money of the live agents by role, derived from the
// events (the snapshot has the total only).
func (t *tracker) roleMoney() map[economysim.RoleID]int64 {
	out := map[economysim.RoleID]int64{}
	for a, m := range t.money {
		out[t.role[a]] += m
	}
	return out
}

// flush returns the window's counters whose key starts with one of prefixes
// (all of them when prefixes is empty), as sorted "key=value" strings, and
// starts a new window.
func (t *tracker) flush(prefixes []string) []string {
	var out []string
	for k, v := range t.counts {
		if keep(k, prefixes) {
			out = append(out, fmt.Sprintf("%s=%d", k, v))
		}
	}
	sort.Strings(out)
	clear(t.counts)
	return out
}

func keep(k string, prefixes []string) bool {
	if len(prefixes) == 0 {
		return true
	}
	for _, p := range prefixes {
		if strings.HasPrefix(k, p) {
			return true
		}
	}
	return false
}
