package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	economysim "github.com/Whealinghold/economy-sim"
)

// reporter prints snapshots and event windows, as text or as JSON lines.
type reporter struct {
	out      io.Writer
	o        options
	tr       *tracker
	show     map[string]bool
	prefixes []string
	trace    map[string]bool
	err      error
}

func newReporter(out io.Writer, o options, tr *tracker) *reporter {
	r := &reporter{out: out, o: o, tr: tr, show: map[string]bool{}, trace: map[string]bool{}}
	for _, s := range strings.Split(o.show, ",") {
		r.show[strings.TrimSpace(s)] = true
	}
	for _, s := range strings.Split(o.events, ",") {
		if s = strings.TrimSpace(s); s != "" {
			r.prefixes = append(r.prefixes, s)
		}
	}
	for _, s := range strings.Split(o.trace, ",") {
		if s = strings.TrimSpace(s); s != "" {
			r.trace[s] = true
		}
	}
	return r
}

func (r *reporter) printf(format string, a ...any) {
	if _, err := fmt.Fprintf(r.out, format, a...); err != nil && r.err == nil {
		r.err = err
	}
}

func (r *reporter) json(v any) {
	b, err := json.Marshal(v)
	if err != nil && r.err == nil {
		r.err = err
		return
	}
	r.printf("%s\n", b)
}

// roleOut, marketOut and snapOut are the JSON shapes of a snapshot.
type roleOut struct {
	Region, Role      string
	Households, Unfed int64
	MeanRatioBP       int64 `json:",omitempty"`
	ExpectedRatioBP   int64 `json:",omitempty"`
	Viable            bool
	WageObserved      int64
	Money             int64
	MoneyPerHousehold int64
}

type marketOut struct {
	Region, Commodity            string
	Price, Volume, HeldByAgents  int64
	HeldByEstate                 int64
	UnmatchedBid, UnmatchedAsk   int64
	BestBid, BestAsk, CostAnchor int64
}

type snapOut struct {
	Kind                                 string
	Tick, Households, Unfed, Money, Pool int64
	Roles                                []roleOut   `json:",omitempty"`
	Markets                              []marketOut `json:",omitempty"`
}

// buildSnap turns a snapshot and the tracker's derived money into the report
// shape, with every role and market.
func buildSnap(s economysim.Snapshot, tr *tracker) snapOut {
	n := s.Names
	money := tr.roleMoney()
	so := snapOut{Kind: "snapshot", Tick: s.Tick, Households: s.Households, Money: int64(s.Money)}
	var byRole = map[economysim.RoleID]int64{}
	for _, row := range s.Roles {
		so.Unfed += row.Unfed
		byRole[row.Role] += row.Households
	}
	var tracked int64
	for _, m := range money {
		tracked += m
	}
	so.Pool = int64(s.Money) - tracked // the estates' share, and the tracking error
	for _, row := range s.Roles {
		ro := roleOut{
			Region: n.Regions[row.Region], Role: n.Roles[row.Role],
			Households: row.Households, Unfed: row.Unfed, Viable: row.Viable,
			WageObserved: int64(row.WageObserved), Money: money[row.Role],
		}
		if row.HasMeanRatio {
			ro.MeanRatioBP = row.MeanRatioBP
		}
		if row.HasExpected {
			ro.ExpectedRatioBP = row.ExpectedRatioBP
		}
		if h := byRole[row.Role]; h > 0 {
			ro.MoneyPerHousehold = ro.Money / h
		}
		so.Roles = append(so.Roles, ro)
	}
	for _, m := range s.Markets {
		c := m.State.Commodity
		mo := marketOut{
			Region: n.Regions[m.State.Region], Commodity: n.Commodities[c],
			Price: int64(m.State.Price), Volume: int64(m.State.Volume),
			HeldByAgents: int64(m.HeldByAgents), HeldByEstate: int64(m.HeldByEstate),
			UnmatchedBid: int64(m.State.UnmatchedBid), UnmatchedAsk: int64(m.State.UnmatchedAsk),
			BestBid: int64(m.State.BestBid), BestAsk: int64(m.State.BestAsk),
			CostAnchor: int64(tr.cleared[marketKey{m.State.Region, c}].CostAnchor),
		}
		so.Markets = append(so.Markets, mo)
	}
	return so
}

// snapshot prints the boundary of tick s.Tick.
func (r *reporter) snapshot(s economysim.Snapshot) {
	n := s.Names
	so := buildSnap(s, r.tr)
	if !r.show["roles"] && !r.show["money"] {
		so.Roles = nil
	}
	if !r.show["markets"] {
		so.Markets = nil
	}
	if r.o.asJSON {
		r.json(so)
		return
	}
	r.printf("== tick %d  households %d  unfed %d  money %d (estates and tracking error %d)\n",
		so.Tick, so.Households, so.Unfed, so.Money, so.Pool)
	if len(so.Roles) > 0 {
		r.printf("  %-10s %-6s %6s %6s %8s %8s %6s %8s %12s %10s\n",
			"role", "region", "hh", "unfed", "ratioBP", "expectBP", "viable", "wage", "money", "money/hh")
		for _, ro := range so.Roles {
			r.printf("  %-10s %-6s %6d %6d %8d %8d %6v %8d %12d %10d\n",
				ro.Role, ro.Region, ro.Households, ro.Unfed, ro.MeanRatioBP, ro.ExpectedRatioBP,
				ro.Viable, ro.WageObserved, ro.Money, ro.MoneyPerHousehold)
		}
	}
	if len(so.Markets) > 0 {
		r.printf("  %-10s %-6s %7s %7s %9s %8s %9s %9s %7s %7s %7s\n",
			"commodity", "region", "price", "volume", "stock", "estate", "unmetBid", "unmetAsk", "bestBid", "bestAsk", "anchor")
		for _, mo := range so.Markets {
			r.printf("  %-10s %-6s %7d %7d %9d %8d %9d %9d %7d %7d %7d\n",
				mo.Commodity, mo.Region, mo.Price, mo.Volume, mo.HeldByAgents, mo.HeldByEstate,
				mo.UnmatchedBid, mo.UnmatchedAsk, mo.BestBid, mo.BestAsk, mo.CostAnchor)
		}
	}
	for _, m := range s.Markets {
		c := n.Commodities[m.State.Commodity]
		if mc, ok := r.tr.cleared[marketKey{m.State.Region, m.State.Commodity}]; ok && r.trace[c] {
			r.printf("  trace %s tick %d price %d age %d volume %d unmet bid %d ask %d best bid %v/%d ask %v/%d anchor %d\n",
				c, mc.Tick, mc.Price, mc.PriceAge, mc.Volume, mc.UnmatchedBid, mc.UnmatchedAsk,
				mc.HasBestBid, mc.BestBid, mc.HasBestAsk, mc.BestAsk, mc.CostAnchor)
		}
	}
}

// window prints the counters of the ticks since the last window, ending at
// tick (exclusive), and starts a new window.
func (r *reporter) window(tick int64) {
	if !r.show["events"] {
		r.tr.flush(nil)
		return
	}
	kv := r.tr.flush(r.prefixes)
	if r.o.asJSON {
		m := map[string]int64{}
		for _, s := range kv {
			i := strings.LastIndexByte(s, '=')
			var v int64
			fmt.Sscan(s[i+1:], &v)
			m[s[:i]] = v
		}
		r.json(struct {
			Kind   string
			Until  int64
			Counts map[string]int64
		}{"events", tick, m})
		return
	}
	r.printf("  events before tick %d: %s\n", tick, strings.Join(kv, " "))
}
