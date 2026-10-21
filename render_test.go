package economysim_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file is the renderer of the golden files (REF 8.5, OBS 9.5; plan 1-12
// D176): test code that renders a run by reflection over exported fields. It
// reads typed IDs through the names table and the zero rule of OBS 9.5.2 keeps a
// field a module appends at its off value out of every line.

// obsEventOrder is the order of the table of OBS 4.1: the order of the counts in
// events-one.txt. TestRender_EventOrderMatchesCatalogue parses the table and
// fails if this list drifts from it.
var obsEventOrder = []string{
	"WorldCreated", "WorldRestored", "RunStarted", "RunEnded", "TickStarted", "TickEnded",
	"OrderPlaced", "OrderRejected", "TradeExecuted", "MarketCleared", "BeliefsUpdated",
	"RecipeExecuted", "RecipeRefused", "RecipeAbandoned", "ProductionHalted",
	"OwnUseChanged", "UpkeepShortfall", "WantShortfall", "RoomLimited", "AgentSpawned",
	"AgentBankrupted", "EstateOpened", "WeightChanged", "RoleChanged", "EntryLimited",
	"PopulationChanged", "TraineeshipStarted", "TraineeshipCompleted", "TraineeshipEnded",
	"StoreChanged", "BoundaryFlow", "WearApplied", "ReliefPaid", "MoneyFlowLimited",
	"MoneyFlowEnded", "MoneyFlowScaled", "StoreLoss", "AgentRelocated", "EdgeStatusChanged",
	"ShipmentDispatched", "ShipmentArrived", "ShipmentRejected", "TransitFactorChanged",
	"ContractFormed", "ContractSettled", "ContractRound", "MerchantsPlanned",
	"MigrationStarted", "MigrationReviewed", "EndowmentChanged", "EndowmentAdjusted",
	"RumoursServed", "OutsideTraded", "OutsideEnded", "OutsideShocked", "YieldShocked",
	"PopulationScaled", "RouteRiskScaled", "StoreRiskScaled", "HireFormed", "HireEnded",
	"LabourRound", "LoanFormed", "LoanEnded", "LoanPassed", "CreditRound", "CreditSettled",
	"CreditRateScaled", "TechnologyAdvanced", "TechnologyAdjusted",
	"SettlementClassChanged", "SettlementHeld", "RentPaid", "TitleTransferred",
	"RentScaled", "FacilityChanged", "FacilityLimited", "FiscalFlowApplied",
	"TreasuryOrdered", "MembershipChanged", "RightRefused", "PriceControlled",
	"LegalLevelSet", "RightSuspensionSet", "SavingPlanned", "PolicyDecided", "CoverFormed",
	"CoverEnded", "CoverRound", "CoverSettled",
}

// corePkg is the import path of the root package, to tell its typed IDs.
var corePkg = reflect.TypeOf(economysim.RegionID(0)).PkgPath()

// namer renders values by the names table.
type namer struct{ names economysim.Names }

// idName returns the string of a typed ID value and true, or false if v is not
// a typed ID of the root package.
func (n namer) idName(v reflect.Value) (string, bool) {
	t := v.Type()
	if t.PkgPath() != corePkg || !strings.HasSuffix(t.Name(), "ID") || v.Kind() < reflect.Int || v.Kind() > reflect.Int64 {
		return "", false
	}
	i := v.Int()
	var table []string
	switch t.Name() {
	case "RegionID":
		table = n.names.Regions
	case "CommodityID":
		table = n.names.Commodities
	case "RoleID":
		table = n.names.Roles
	case "RecipeID":
		table = n.names.Recipes
	default: // AgentID and any ID with no names table is its integer
		return strconv.FormatInt(i, 10), true
	}
	switch {
	case v.Interface() == economysim.NoCommodity:
		return "-", true
	case i < 0 || i >= int64(len(table)):
		return "#" + strconv.FormatInt(i, 10), true
	}
	return table[i], true
}

// isSide reports whether t is the root package's Side.
func isSide(t reflect.Type) bool { return t.PkgPath() == corePkg && t.Name() == "Side" }

// keepZero reports whether a field of this type is rendered at its zero value:
// every typed ID and Side (OBS 9.5.2).
func (n namer) keepZero(v reflect.Value) bool {
	_, id := n.idName(v)
	return id || isSide(v.Type())
}

// value renders one value with no spaces outside a slice or struct.
func (n namer) value(v reflect.Value) string {
	if s, ok := n.idName(v); ok {
		return s
	}
	if isSide(v.Type()) {
		return v.Interface().(economysim.Side).String()
	}
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.String:
		return v.String()
	case reflect.Array:
		if v.Type().Elem().Kind() == reflect.Uint8 && v.Len() == 32 {
			b := make([]byte, 8)
			for i := range b {
				b[i] = byte(v.Index(i).Uint())
			}
			return hex.EncodeToString(b)
		}
	case reflect.Slice:
		parts := make([]string, v.Len())
		for i := range parts {
			parts[i] = n.value(v.Index(i))
		}
		return "[" + strings.Join(parts, " ") + "]"
	case reflect.Struct:
		return "{" + strings.Join(n.fields(v, nil), " ") + "}"
	}
	return "?" + v.Type().String() // a float, map or pointer: not allowed in an event
}

// fields renders the exported fields of a struct as Name=value, in declaration
// order, omitting a field at its zero value unless keepZero, and the fields skip
// names.
func (n namer) fields(v reflect.Value, skip func(reflect.StructField) bool) []string {
	var out []string
	for i := 0; i < v.NumField(); i++ {
		f := v.Type().Field(i)
		if !f.IsExported() || (skip != nil && skip(f)) {
			continue
		}
		fv := v.Field(i)
		if fv.IsZero() && !n.keepZero(fv) {
			continue
		}
		out = append(out, f.Name+"="+n.value(fv))
	}
	return out
}

// snapshotRows is the number of rows of the sections of a snapshot.
func snapshotRows(s economysim.Snapshot) int {
	return len(s.Regions) + len(s.Boundary) + len(s.Pending) + len(s.Construction) + len(s.Roles) + len(s.Markets) + len(s.Jobs)
}

// renderLine renders one event, or any struct with a Tick field, as one line of
// OBS 9.5.2. Names and Snapshot fields are omitted; WorldRestored shows
// SnapshotRows instead.
func renderLine(names economysim.Names, e any) string {
	v := reflect.ValueOf(e)
	n := namer{names}
	var rows int
	skip := func(f reflect.StructField) bool {
		if f.Name == "Snapshot" && f.Type == reflect.TypeOf(economysim.Snapshot{}) {
			rows = snapshotRows(v.FieldByName("Snapshot").Interface().(economysim.Snapshot))
		}
		return f.Name == "Tick" || f.Type == reflect.TypeOf(economysim.Names{}) || f.Type == reflect.TypeOf(economysim.Snapshot{})
	}
	parts := n.fields(v, skip)
	if rows > 0 {
		parts = append(parts, "SnapshotRows="+strconv.Itoa(rows))
	}
	line := "t=" + strconv.FormatInt(v.FieldByName("Tick").Int(), 10) + " " + v.Type().Name()
	if len(parts) > 0 {
		line += " " + strings.Join(parts, " ")
	}
	return line
}

// eventTick returns the Tick of an event.
func eventTick(e economysim.Event) int64 { return reflect.ValueOf(e).FieldByName("Tick").Int() }

// eventType returns the type name of an event.
func eventType(e economysim.Event) string { return reflect.TypeOf(e).Name() }

// digestOf is the first 8 bytes of the SHA-256 of the lines, each ended by a
// newline, in lower-case hex.
func digestOf(lines []string) string {
	h := sha256.New()
	for _, l := range lines {
		h.Write([]byte(l + "\n"))
	}
	return hex.EncodeToString(h.Sum(nil)[:8])
}

// sortedTypes orders event type names by the table of OBS 4.1, with a type the
// list does not know last, by name, so it is never dropped.
func sortedTypes(counts map[string]int) []string {
	var out []string
	for _, t := range obsEventOrder {
		if counts[t] > 0 {
			out = append(out, t)
		}
	}
	var rest []string
	for t := range counts {
		if !slices.Contains(obsEventOrder, t) {
			rest = append(rest, t)
		}
	}
	sort.Strings(rest)
	return append(out, rest...)
}

// renderEvents renders events-one.txt (OBS 9.5.3): the construction block, the
// full rendering of ticks 0 to 4, one line per tick with a digest, the count of
// each event type over the run in the order of OBS 4.1, and the final snapshot.
func renderEvents(res result) []byte {
	names := res.Final.Names
	var construction, body []economysim.Event
	started := false
	for _, e := range res.Events {
		switch {
		case eventType(e) == "TickStarted":
			started = true
		case eventType(e) == "RunStarted":
			body = append(body, e)
			continue
		}
		if started {
			body = append(body, e)
		} else {
			construction = append(construction, e)
		}
	}
	var b strings.Builder
	b.WriteString("construction\n")
	for _, e := range construction {
		b.WriteString(renderLine(names, e) + "\n")
	}
	b.WriteString("ticks 0 to 4\n")
	byTick := map[int64][]string{}
	for _, e := range body {
		line := renderLine(names, e)
		byTick[eventTick(e)] = append(byTick[eventTick(e)], line)
		if t := eventTick(e); t >= 0 && t <= 4 {
			b.WriteString(line + "\n")
		}
	}
	b.WriteString("per tick\n")
	ticks := make([]int64, 0, len(byTick))
	for t := range byTick {
		ticks = append(ticks, t)
	}
	slices.Sort(ticks)
	for _, t := range ticks {
		fmt.Fprintf(&b, "t=%d events=%d digest=%s\n", t, len(byTick[t]), digestOf(byTick[t]))
	}
	counts := map[string]int{}
	for _, e := range res.Events {
		counts[eventType(e)]++
	}
	b.WriteString("counts\n")
	for _, t := range sortedTypes(counts) {
		fmt.Fprintf(&b, "%s %d\n", t, counts[t])
	}
	b.WriteString("snapshot\n")
	b.WriteString(renderSnapshot(res.Final))
	return []byte(b.String())
}

// renderSnapshot renders a snapshot in the style of the events: a header line of
// the scalar fields, then one section per slice of rows with at least one row.
func renderSnapshot(s economysim.Snapshot) string {
	n := namer{s.Names}
	v := reflect.ValueOf(s)
	isRows := func(f reflect.StructField) bool {
		return f.Type.Kind() == reflect.Slice && f.Type.Elem().Kind() == reflect.Struct
	}
	skip := func(f reflect.StructField) bool { return isRows(f) || f.Type == reflect.TypeOf(economysim.Names{}) }
	head := "Snapshot"
	if parts := n.fields(v, skip); len(parts) > 0 {
		head += " " + strings.Join(parts, " ")
	}
	out := head + "\n"
	for i := 0; i < v.NumField(); i++ {
		f := v.Type().Field(i)
		if !f.IsExported() || !isRows(f) || v.Field(i).Len() == 0 {
			continue
		}
		out += f.Name + "\n"
		for j := 0; j < v.Field(i).Len(); j++ {
			row := v.Field(i).Index(j)
			line := row.Type().Name()
			if parts := n.fields(row, nil); len(parts) > 0 {
				line += " " + strings.Join(parts, " ")
			}
			out += line + "\n"
		}
	}
	return out
}

// solveReport is the part of the solver report the golden header holds (D172),
// copied from the SteadyState of SolveSteadyState.
type solveReport struct {
	SurplusBP, LabourShareBP, Pivots int64
	HasLabourShare                   bool
	Unpriced                         []string // "region/commodity"
	UnpricedWages                    []string
}

// newSolveReport reads the report of the first programme.
func newSolveReport(ss economysim.SteadyState) solveReport {
	var r solveReport
	if len(ss.Programmes) > 0 {
		p := ss.Programmes[0]
		r.SurplusBP, r.LabourShareBP, r.HasLabourShare, r.Pivots = p.SurplusBP, p.LabourShareBP, p.HasLabourShare, p.Pivots
	}
	for _, rc := range ss.Report.UnpricedCommodities {
		r.Unpriced = append(r.Unpriced, rc.Region+"/"+rc.Commodity)
	}
	r.UnpricedWages = append(r.UnpricedWages, ss.Report.UnpricedWages...)
	return r
}

// kv appends " name=value" for a non-zero value.
func addKV(b *strings.Builder, name string, v int64) {
	if v != 0 {
		fmt.Fprintf(b, " %s=%d", name, v)
	}
}

// renderGolden renders one.txt (REF 8.5 items 1 to 5): the header, the samples,
// the event counts, the flows and the final state. A count, an amount or a flag
// at zero is not written; a row or section with nothing in it is not written; a
// warning is listed only when it fired (the caller passes the warnings that
// fired). RunStarted and RunEnded are not counted, so a Step loop and a Run give
// the same file.
func renderGolden(cfgName string, seed uint64, ticks int64, res result, report solveReport, warnings []economysim.Warning) []byte {
	var b strings.Builder
	names := res.Final.Names
	fmt.Fprintf(&b, "format 1\nconfig %s\nhash %s\nseed %d ticks %d\n", cfgName, hex.EncodeToString(res.Final.ConfigHash[:]), seed, ticks)
	var ws []string
	for _, w := range warnings {
		ws = append(ws, "warning "+w.Rule+" "+strings.Join(w.Path, "/"))
	}
	sort.Strings(ws)
	for _, w := range ws {
		b.WriteString(w + "\n")
	}
	var sr strings.Builder
	addKV(&sr, "surplus_bp", report.SurplusBP)
	if report.HasLabourShare {
		addKV(&sr, "labour_share_bp", report.LabourShareBP)
	}
	addKV(&sr, "pivots", report.Pivots)
	if len(report.Unpriced) > 0 {
		sr.WriteString(" unpriced=" + strings.Join(report.Unpriced, ","))
	}
	if len(report.UnpricedWages) > 0 {
		sr.WriteString(" unpriced_wages=" + strings.Join(report.UnpricedWages, ","))
	}
	if sr.Len() > 0 {
		b.WriteString("solver" + sr.String() + "\n")
	}

	sampleAt := make([]int64, 0, len(res.Samples))
	for t := range res.Samples {
		sampleAt = append(sampleAt, t)
	}
	slices.Sort(sampleAt)
	anchors := costAnchors(res.Events)
	for _, t := range sampleAt {
		b.WriteString(renderSample(t, res.Samples[t], anchors[t]))
	}

	writeCounts(&b, res.Events)
	writeFlows(&b, names, res.Events)
	b.WriteString("final\n")
	writeFinal(&b, res.Final)
	return []byte(b.String())
}

// goldenMarketKey names a market in the cost-anchor table.
type goldenMarketKey struct{ region, commodity int }

// costAnchors returns, per tick, the cost anchor of each MarketCleared.
func costAnchors(events []economysim.Event) map[int64]map[goldenMarketKey]int64 {
	out := map[int64]map[goldenMarketKey]int64{}
	for _, e := range events {
		if mc, ok := e.(economysim.MarketCleared); ok && mc.CostAnchor != 0 {
			if out[mc.Tick] == nil {
				out[mc.Tick] = map[goldenMarketKey]int64{}
			}
			out[mc.Tick][goldenMarketKey{int(mc.Region), int(mc.Commodity)}] = int64(mc.CostAnchor)
		}
	}
	return out
}

// renderSample renders the sample after step t.
func renderSample(t int64, s economysim.Snapshot, anchors map[goldenMarketKey]int64) string {
	var b strings.Builder
	fmt.Fprintf(&b, "sample t=%d\n", t)
	for _, m := range s.Markets {
		st := m.State
		var r strings.Builder
		addKV(&r, "price", int64(st.Price))
		addKV(&r, "age", st.Age)
		addKV(&r, "volume", int64(st.Volume))
		addKV(&r, "unmatched_bid", int64(st.UnmatchedBid))
		addKV(&r, "unmatched_ask", int64(st.UnmatchedAsk))
		if st.HasBid {
			fmt.Fprintf(&r, " best_bid=%d", st.BestBid)
		}
		if st.HasAsk {
			fmt.Fprintf(&r, " best_ask=%d", st.BestAsk)
		}
		addKV(&r, "held_agents", int64(m.HeldByAgents))
		addKV(&r, "held_estate", int64(m.HeldByEstate))
		addKV(&r, "pending", int64(m.Pending))
		addKV(&r, "cost_anchor", anchors[goldenMarketKey{int(st.Region), int(st.Commodity)}])
		if r.Len() > 0 {
			fmt.Fprintf(&b, "market %s %s%s\n", tableName(s.Names.Regions, int(st.Region)), tableName(s.Names.Commodities, int(st.Commodity)), r.String())
		}
	}
	for _, r := range s.Roles {
		var l strings.Builder
		addKV(&l, "agents", r.Agents)
		addKV(&l, "households", r.Households)
		addKV(&l, "unfed", r.Unfed)
		if l.Len() > 0 {
			fmt.Fprintf(&b, "role %s %s%s\n", tableName(s.Names.Regions, int(r.Region)), tableName(s.Names.Roles, int(r.Role)), l.String())
		}
	}
	for _, r := range s.Regions {
		var l strings.Builder
		if r.Households > 0 {
			addKV(&l, "mean_money", int64(r.MoneyHouseholds)/r.Households)
		}
		addKV(&l, "pool", int64(r.MoneyPool))
		if l.Len() > 0 {
			fmt.Fprintf(&b, "region %s%s\n", tableName(s.Names.Regions, int(r.Region)), l.String())
		}
	}
	return b.String()
}

// name returns table[i], or "#i" outside the table.
func tableName(table []string, i int) string {
	if i < 0 || i >= len(table) {
		return "#" + strconv.Itoa(i)
	}
	return table[i]
}

// writeCounts writes the event counts: by type, by reason or cause, and per
// window of 100 ticks. RunStarted and RunEnded are not counted.
func writeCounts(b *strings.Builder, events []economysim.Event) {
	counts := map[string]int{}
	by := map[string]int{}
	windows := map[int64]map[string]int{}
	windowed := map[string]bool{"TradeExecuted": true, "RecipeExecuted": true, "OrderPlaced": true, "UpkeepShortfall": true}
	for _, e := range events {
		tn := eventType(e)
		if tn == "RunStarted" || tn == "RunEnded" {
			continue
		}
		counts[tn]++
		switch tn {
		case "OrderRejected", "RecipeAbandoned":
			by[tn+" reason="+string(reflect.ValueOf(e).FieldByName("Reason").String())]++
		case "AgentBankrupted":
			by[tn+" cause="+reflect.ValueOf(e).FieldByName("Cause").String()]++
		}
		if windowed[tn] {
			w := eventTick(e) / 100 * 100
			if windows[w] == nil {
				windows[w] = map[string]int{}
			}
			windows[w][tn]++
		}
	}
	b.WriteString("events\n")
	for _, t := range sortedTypes(counts) {
		fmt.Fprintf(b, "%s %d\n", t, counts[t])
	}
	keys := make([]string, 0, len(by))
	for k := range by {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(b, "%s %d\n", k, by[k])
	}
	starts := make([]int64, 0, len(windows))
	for w := range windows {
		starts = append(starts, w)
	}
	slices.Sort(starts)
	for _, w := range starts {
		fmt.Fprintf(b, "window %d", w)
		for _, tn := range []string{"TradeExecuted", "RecipeExecuted", "OrderPlaced", "UpkeepShortfall"} {
			if c := windows[w][tn]; c > 0 {
				fmt.Fprintf(b, " %s=%d", tn, c)
			}
		}
		b.WriteString("\n")
	}
}

// writeFlows writes the BoundaryFlow totals by region, reason, kind and
// commodity; the construction bucket (Tick -1) is its own section.
func writeFlows(b *strings.Builder, names economysim.Names, events []economysim.Event) {
	type key struct {
		construction bool
		region       economysim.RegionID
		reason       economysim.Reason
		kind         economysim.Kind
		commodity    economysim.CommodityID
	}
	totals := map[key]int64{}
	for _, e := range events {
		if f, ok := e.(economysim.BoundaryFlow); ok {
			totals[key{f.Tick < 0, f.Region, f.Reason, f.Kind, f.Commodity}] += f.Total
		}
	}
	var lines []string
	for k, v := range totals {
		if v == 0 {
			continue
		}
		sec := "flow"
		if k.construction {
			sec = "construction_flow"
		}
		c := "money"
		if k.commodity != economysim.NoCommodity {
			c = tableName(names.Commodities, int(k.commodity))
		}
		lines = append(lines, fmt.Sprintf("%s %s %s %s %s %d", sec, tableName(names.Regions, int(k.region)), k.reason, k.kind, c, v))
	}
	sort.Strings(lines)
	for _, l := range lines {
		b.WriteString(l + "\n")
	}
}

// writeFinal writes the final state: agents and households by region and role,
// the money and goods totals and the jobs in progress by recipe.
func writeFinal(b *strings.Builder, s economysim.Snapshot) {
	fmt.Fprintf(b, "agents %d households %d money %d\n", s.Agents, s.Households, s.Money)
	for _, r := range s.Roles {
		if r.Agents > 0 || r.Households > 0 {
			fmt.Fprintf(b, "role %s %s agents=%d households=%d\n", tableName(s.Names.Regions, int(r.Region)), tableName(s.Names.Roles, int(r.Role)), r.Agents, r.Households)
		}
	}
	goods := map[string]int64{}
	for _, m := range s.Markets {
		if u := int64(m.HeldByAgents) + int64(m.HeldByEstate); u != 0 {
			goods[tableName(s.Names.Commodities, int(m.State.Commodity))] += u
		}
	}
	keys := make([]string, 0, len(goods))
	for k := range goods {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(b, "goods %s %d\n", k, goods[k])
	}
	for _, j := range s.Jobs {
		fmt.Fprintf(b, "jobs %s %s agents=%d executions=%d households=%d\n", tableName(s.Names.Regions, int(j.Region)), tableName(s.Names.Recipes, int(j.Recipe)), j.Agents, j.Executions, j.Households)
	}
}

// testNames is the names table of the unit tests.
var testNames = economysim.Names{
	Regions:     []string{"alpha", "beta"},
	Commodities: []string{"grain", "flour"},
	Roles:       []string{"baker", "farmer"},
	Recipes:     []string{"bake"},
}

// TestRender_EventOrderMatchesCatalogue verifies obsEventOrder is the table of
// OBS 4.1 in the order of the file, by parsing it as the catalogue test does.
//
// Requirement: FR-OBS-080
//
// Guard: a catalogue row added or moved without the list would reorder or lose
// a count line of events-one.txt silently.
func TestRender_EventOrderMatchesCatalogue(t *testing.T) {
	t.Parallel()
	_, order := specRows(t)
	if !slices.Equal(order, obsEventOrder) {
		t.Errorf("obsEventOrder = %v, want the order of the table of spec-observability.md 4.1: %v", obsEventOrder, order)
	}
}

// TestRender_Line verifies each rule of OBS 9.5.2 on hand-built values.
//
// Requirement: FR-OBS-080
func TestRender_Line(t *testing.T) {
	t.Parallel()
	type Amount struct {
		Commodity economysim.CommodityID
		Quantity  economysim.Quantity
	}
	type Probe struct {
		Tick      int64
		Region    economysim.RegionID
		Commodity economysim.CommodityID
		Agent     economysim.AgentID
		Side      economysim.Side
		Count     int64
		Flag      bool
		Reason    economysim.Reason
		Hash      [32]byte
		Rows      []Amount
		Empty     []Amount
		Names     economysim.Names
	}
	h := [32]byte{0xab, 0xcd, 0, 0, 0, 0, 0, 0x01, 0xff}
	tests := []struct {
		name string
		in   Probe
		want string
	}{
		{"zero omitted, zero IDs and side shown", Probe{}, "t=0 Probe Region=alpha Commodity=grain Agent=0 Side=bid"},
		{"values", Probe{Tick: 7, Region: 1, Commodity: 1, Agent: 12, Side: economysim.Ask, Count: 3, Flag: true, Reason: "no_stock"},
			"t=7 Probe Region=beta Commodity=flour Agent=12 Side=ask Count=3 Flag=true Reason=no_stock"},
		{"out of range", Probe{Region: 9, Commodity: 5}, "t=0 Probe Region=#9 Commodity=#5 Agent=0 Side=bid"},
		{"no commodity", Probe{Commodity: economysim.NoCommodity}, "t=0 Probe Region=alpha Commodity=- Agent=0 Side=bid"},
		{"hash first 8 bytes", Probe{Hash: h}, "t=0 Probe Region=alpha Commodity=grain Agent=0 Side=bid Hash=abcd000000000001"},
		{"rows with zero rule", Probe{Rows: []Amount{{Commodity: 1, Quantity: 3}, {Quantity: 2}}},
			"t=0 Probe Region=alpha Commodity=grain Agent=0 Side=bid Rows=[{Commodity=flour Quantity=3} {Commodity=grain Quantity=2}]"},
		{"names omitted", Probe{Names: testNames}, "t=0 Probe Region=alpha Commodity=grain Agent=0 Side=bid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := renderLine(testNames, tt.in); got != tt.want {
				t.Errorf("renderLine = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRender_AppendedFieldWorkedExample is the worked example of OBS 9.5.2: a
// field a module appends to AgentSpawned renders no trace at its off value, and
// shows StoresLocked=2 with it.
//
// Requirement: FR-OBS-080, FR-OBS-091
//
// Guard: the off golden must not move when a module appends a field.
func TestRender_AppendedFieldWorkedExample(t *testing.T) {
	t.Parallel()
	const off = "t=0 AgentSpawned Agent=1 Region=alpha Role=baker Weight=1 StartMoney=100"
	real := economysim.AgentSpawned{Tick: 0, Agent: 1, Region: 0, Role: 0, Weight: 1, StartMoney: 100}
	if got := renderLine(testNames, real); got != off {
		t.Errorf("AgentSpawned line = %q, want %q", got, off)
	}
	type AgentSpawned struct {
		Tick         int64
		Agent        economysim.AgentID
		Region       economysim.RegionID
		Role         economysim.RoleID
		Weight       int64
		StartMoney   economysim.Money
		External     bool
		StoresLocked int64
	}
	appended := AgentSpawned{Agent: 1, Weight: 1, StartMoney: 100}
	if got := renderLine(testNames, appended); got != off {
		t.Errorf("line with StoresLocked at 0 = %q, want byte-equal %q", got, off)
	}
	appended.StoresLocked = 2
	if got, want := renderLine(testNames, appended), off+" StoresLocked=2"; got != want {
		t.Errorf("line with StoresLocked=2 = %q, want %q", got, want)
	}
}

// TestRender_Events verifies the layout of events-one.txt on a hand-built run:
// the construction block without RunStarted, the per-tick lines with the count
// and the digest of a known SHA-256 prefix, the counts in catalogue order.
//
// Requirement: FR-OBS-080, FR-REF-029
func TestRender_Events(t *testing.T) {
	t.Parallel()
	res := result{
		Final: economysim.Snapshot{Names: testNames, Tick: 1, Agents: 2,
			Regions: []economysim.RegionRow{{Region: 1, Agents: 2}}},
		Events: []economysim.Event{
			economysim.RunStarted{Tick: 0, Ticks: 1},
			economysim.WorldCreated{Tick: 0},
			economysim.AgentSpawned{Tick: 0, Agent: 1, Weight: 1},
			economysim.TickStarted{Tick: 0},
			economysim.TickEnded{Tick: 0},
			economysim.RunEnded{Tick: 1, Completed: 1},
		},
	}
	got := string(renderEvents(res))
	want := "construction\n" +
		"t=0 WorldCreated\n" +
		"t=0 AgentSpawned Agent=1 Region=alpha Role=baker Weight=1\n" +
		"ticks 0 to 4\n" +
		"t=0 RunStarted Ticks=1\n" +
		"t=0 TickStarted\n" +
		"t=0 TickEnded\n" +
		"t=1 RunEnded Completed=1\n" +
		"per tick\n" +
		"t=0 events=3 digest=" + digestOf([]string{"t=0 RunStarted Ticks=1", "t=0 TickStarted", "t=0 TickEnded"}) + "\n" +
		"t=1 events=1 digest=" + digestOf([]string{"t=1 RunEnded Completed=1"}) + "\n" +
		"counts\n" +
		"WorldCreated 1\nRunStarted 1\nRunEnded 1\nTickStarted 1\nTickEnded 1\nAgentSpawned 1\n" +
		"snapshot\n" +
		"Snapshot Tick=1 Agents=2\n" +
		"Regions\n" +
		"RegionRow Region=beta Agents=2\n"
	if got != want {
		t.Errorf("renderEvents =\n%s\nwant\n%s", got, want)
	}
	if d := digestOf([]string{"t=0 TickStarted", "t=0 TickEnded"}); d != "a9ef22e14c39f5ba" {
		t.Errorf("digestOf = %s, want a9ef22e14c39f5ba (the SHA-256 prefix of the two lines)", d)
	}
}

// TestRender_GoldenZeroRule verifies renderGolden writes only what is non-zero,
// lists a warning only when it was passed, leaves RunStarted and RunEnded out of
// the counts, and writes the solver line from the report.
//
// Requirement: FR-REF-029, FR-OBS-080
func TestRender_GoldenZeroRule(t *testing.T) {
	t.Parallel()
	res := result{
		Final: economysim.Snapshot{Names: testNames, Agents: 3, Households: 4, Money: 50,
			Roles: []economysim.RoleRow{{Region: 0, Role: 1, Agents: 3, Households: 4}, {Region: 0, Role: 0}}},
		Samples: map[int64]economysim.Snapshot{0: {Names: testNames,
			Markets: []economysim.MarketRow{
				{State: economysim.MarketState{Region: 0, Commodity: 0, Price: 10, Volume: 2}},
				{State: economysim.MarketState{Region: 0, Commodity: 1}},
			},
			Regions: []economysim.RegionRow{{Region: 0, Households: 4, MoneyHouseholds: 203}}}},
		Events: []economysim.Event{
			economysim.RunStarted{Ticks: 1},
			economysim.OrderRejected{Tick: 0, Reason: "no_stock"},
			economysim.RunEnded{Tick: 1},
		},
	}
	rep := solveReport{SurplusBP: 250, Pivots: 4, Unpriced: []string{"alpha/grain"}}
	with := string(renderGolden("one", 1, 1, res, rep, []economysim.Warning{{Rule: "no_upkeep", Path: []string{"World", "Upkeep"}}}))
	without := string(renderGolden("one", 1, 1, res, rep, nil))
	for _, w := range []string{
		"format 1\nconfig one\n", "seed 1 ticks 1\n", "warning no_upkeep World/Upkeep\n",
		"solver surplus_bp=250 pivots=4 unpriced=alpha/grain\n",
		"market alpha grain price=10 volume=2\n", "region alpha mean_money=50\n",
		"OrderRejected 1\n", "OrderRejected reason=no_stock 1\n",
		"role alpha farmer agents=3 households=4\n",
	} {
		if !strings.Contains(with, w) {
			t.Errorf("golden lacks %q:\n%s", w, with)
		}
	}
	for _, bad := range []string{"RunStarted", "RunEnded", "market alpha flour", "role alpha baker", "unpriced_wages", "labour_share", "window"} {
		if strings.Contains(with, bad) {
			t.Errorf("golden contains %q, want it absent by the zero rule:\n%s", bad, with)
		}
	}
	if strings.Contains(without, "warning") {
		t.Errorf("golden without warnings lists one:\n%s", without)
	}
}

// TestRender_SolveReport verifies the report copied for the golden header.
//
// Requirement: FR-REF-029
func TestRender_SolveReport(t *testing.T) {
	t.Parallel()
	ss := economysim.SteadyState{
		Programmes: []economysim.ProgrammeResult{{SurplusBP: 250, LabourShareBP: 4000, HasLabourShare: true, Pivots: 7}},
	}
	ss.Report.UnpricedCommodities = []economysim.RegionCommodity{{Region: "north", Commodity: "seed"}}
	ss.Report.UnpricedWages = []string{"north"}
	got := newSolveReport(ss)
	want := solveReport{SurplusBP: 250, LabourShareBP: 4000, HasLabourShare: true, Pivots: 7, Unpriced: []string{"north/seed"}, UnpricedWages: []string{"north"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("newSolveReport = %+v, want %+v", got, want)
	}
	if empty := newSolveReport(economysim.SteadyState{}); !reflect.DeepEqual(empty, solveReport{}) {
		t.Errorf("newSolveReport(empty) = %+v, want zero", empty)
	}
}
