package indicator_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/indicator"
)

// named is one preset's result with the preset's name.
type named struct {
	name string
	v    any
}

// presetsOf computes presets 1 to 21 of spec-indicators.md 9.1 for a window of a
// run, with base the base window of the price index. The inputs a preset needs
// that a world cannot give yet are the caller's: Inequality takes the household
// rows of AgentState (plan 2-17), so it is not here, and WelfareDistribution is
// computed over the role rows of the closing snapshot (one row per role, the
// households whose ratio exists at the role's mean).
func presetsOf(t testing.TB, cfg economysim.Config, g economysim.Graph, base indicator.Base, rw runWin) []named {
	t.Helper()
	w, open, closing := rw.w, rw.open, rw.closing
	var out []named
	add := func(name string, v any, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s() error = %v, want nil", name, err)
		}
		out = append(out, named{name, v})
	}
	p, err := indicator.PricesOfWindow(w)
	if err != nil {
		t.Fatalf("PricesOfWindow() error = %v, want nil", err)
	}
	sp, err := indicator.PricesOfSnapshot(closing)
	if err != nil {
		t.Fatalf("PricesOfSnapshot() error = %v, want nil", err)
	}
	gdp, err := indicator.GDP(w, open, closing, p)
	add("GDP", gdp, err)
	va, err := indicator.ValueAdded(w, cfg, open, closing, p, indicator.NewClasses(cfg))
	add("ValueAdded", va, err)
	pi, err := indicator.PriceIndex(w, base, indicator.WeightsTraded, p, indicator.CellSel{})
	add("PriceIndex", pi, err)
	bl, err := indicator.BasketLevel(cfg, closing)
	add("BasketLevel", bl, err)
	dl, err := indicator.DeliveredPrice(w, nil, p)
	add("DeliveredPrice", dl, err)
	pg, err := indicator.PriceGap(w, p, dl, g)
	add("PriceGap", pg, err)
	sec, err := indicator.Sectors(w, open, closing, p)
	add("Sectors", sec, err)
	tpy := cfg.Calendar.TicksPerYear()
	to, err := indicator.Turnover(w, closing, tpy, indicator.CellSel{})
	add("Turnover", to, err)
	hh, err := indicator.Households(w, open, closing, tpy)
	add("Households", hh, err)
	wd, err := indicator.WelfareDistribution(roleRatios(t, closing), indicator.RowSet{}, 10_000)
	add("WelfareDistribution", wd, err)
	uf, err := indicator.Unfed(w, closing)
	add("Unfed", uf, err)
	tb, err := indicator.TradeBalance(w, p, gdp)
	add("TradeBalance", tb, err)
	rwg, err := indicator.RealWage(cfg, closing, sp)
	add("RealWage", rwg, err)
	sc, err := indicator.StockCover(w, closing, indicator.CellSel{})
	add("StockCover", sc, err)
	rs, err := indicator.RoleShares(w)
	add("RoleShares", rs, err)
	pr, err := indicator.PriceRange(w, indicator.CellSel{})
	add("PriceRange", pr, err)
	pn, err := indicator.Pinned(w, indicator.CellSel{})
	add("Pinned", pn, err)
	ds, err := indicator.Dispersion(w, indicator.CellSel{})
	add("Dispersion", ds, err)
	of, err := indicator.Offset(w, p, indicator.PricesOfReference(cfg), indicator.CellSel{})
	add("Offset", of, err)
	hi, err := indicator.HouseholdIncome(w)
	add("HouseholdIncome", hi, err)
	return out
}

// roleRatios is one household row per role of the snapshot whose ratio exists:
// the households whose ratio exists, at the role's mean.
func roleRatios(t testing.TB, s economysim.Snapshot) indicator.RowSet {
	t.Helper()
	var st []indicator.AgentRow
	for _, r := range s.Roles {
		if r.HasMeanRatio && r.RatioHouseholds > 0 {
			st = append(st, indicator.AgentRow{Region: r.Region, Weight: r.RatioHouseholds, WelfareRatioBP: r.MeanRatioBP, HasWelfareRatio: true})
		}
	}
	set, err := indicator.RowsOf(st, indicator.ColWelfareRatio, indicator.RowOpt{})
	if err != nil {
		t.Fatalf("RowsOf(role ratios) error = %v, want nil", err)
	}
	return set
}

// dump writes the non-zero integers, booleans and strings of v, one per line, as
// path = value. A pointer, map, interface or float in a result is a defect: a
// result is plain data (D563).
func dump(t testing.TB, b *strings.Builder, path string, v reflect.Value) {
	t.Helper()
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			dump(t, b, path+"."+v.Type().Field(i).Name, v.Field(i))
		}
	case reflect.Slice, reflect.Array:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return // a configuration hash: it moves with every pack edit and says nothing
		}
		for i := 0; i < v.Len(); i++ {
			dump(t, b, fmt.Sprintf("%s[%d]", path, i), v.Index(i))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v.Int() != 0 {
			fmt.Fprintf(b, "%s = %d\n", path, v.Int())
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v.Uint() != 0 {
			fmt.Fprintf(b, "%s = %d\n", path, v.Uint())
		}
	case reflect.Bool:
		if v.Bool() {
			fmt.Fprintf(b, "%s = true\n", path)
		}
	case reflect.String:
		if v.String() != "" {
			fmt.Fprintf(b, "%s = %q\n", path, v.String())
		}
	default:
		t.Fatalf("%s: a result holds a %s, want plain integers, booleans, strings, structs and slices", path, v.Kind())
	}
}

// render writes the presets of a window as text.
func render(t testing.TB, results []named) []byte {
	t.Helper()
	var b strings.Builder
	for _, r := range results {
		fmt.Fprintf(&b, "## %s\n", r.name)
		dump(t, &b, r.name, reflect.ValueOf(r.v))
	}
	return []byte(b.String())
}
