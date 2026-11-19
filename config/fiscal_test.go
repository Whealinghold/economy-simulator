package config_test

import (
	"reflect"
	"strconv"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// fiscalDoc is the file form of spec-fiscal.md 3, 14 and 18: one flow of each of the six
// kinds with every key, the target as an inline table as the spec writes it.
const fiscalDoc = minimal + `
[modules.fiscal.tax.tithe]
authority = "crown"
target = { kind = "households", region = "north", role = "cook" }
rate_bp = 100
exempt = 2000
per = "year"
window = { start = 4, until = 90, seasons = ["winter", "spring"] }
label = "Tithe"

[modules.fiscal.tax.remit]
authority = "crown"
target = { kind = "treasury", jurisdiction = "inland" }
amount = 7

[modules.fiscal.duty.sales]
authority = "crown"
region = "north"
commodity = "staple"
payer = "seller"
scope = "domestic"
rate_bp = 300
label = "Sales"

[modules.fiscal.toll.ferry]
authority = "inland"
on = "exit"
mode = "foot"
amount = 4

[modules.fiscal.payout.alms]
authority = "crown"
target = { kind = "households", region = "capital" }
commodity = "staple"
amount = 2
not_income = false
per = "tick"

[modules.fiscal.payout.top]
authority = "crown"
target = { kind = "pool" }
level = 500

[modules.fiscal.purchase.buy]
authority = "crown"
commodity = "staple"
quantity = 500
limit_bp = 8000
hold = 6000
window = { seasons = ["autumn"] }

[modules.fiscal.release.sell]
authority = "crown"
commodity = "staple"
quantity = 500
limit_bp = 12000
reserve = 100
`

// fiscalSame is fiscalDoc written the other way: the tables in another order and the
// season list reversed (the calendar's order is canonical).
const fiscalSame = minimal + `
[modules.fiscal.release.sell]
reserve = 100
limit_bp = 12000
quantity = 500
commodity = "staple"
authority = "crown"
[modules.fiscal.purchase.buy]
window = { seasons = ["autumn"] }
hold = 6000
limit_bp = 8000
quantity = 500
commodity = "staple"
authority = "crown"
[modules.fiscal.payout.top]
level = 500
target = { kind = "pool" }
authority = "crown"
[modules.fiscal.payout.alms]
per = "tick"
amount = 2
commodity = "staple"
target = { kind = "households", region = "capital" }
authority = "crown"
[modules.fiscal.toll.ferry]
amount = 4
mode = "foot"
on = "exit"
authority = "inland"
[modules.fiscal.duty.sales]
label = "Sales"
rate_bp = 300
scope = "domestic"
payer = "seller"
commodity = "staple"
region = "north"
authority = "crown"
[modules.fiscal.tax.remit]
amount = 7
target = { jurisdiction = "inland", kind = "treasury" }
authority = "crown"
[modules.fiscal.tax.tithe]
label = "Tithe"
window = { start = 4, until = 90, seasons = ["spring", "winter"] }
per = "year"
exempt = 2000
rate_bp = 100
target = { kind = "households", role = "cook", region = "north" }
authority = "crown"
`

// fiscalWant is the module that fiscalDoc loads to.
func fiscalWant() *economysim.Fiscal {
	return &economysim.Fiscal{
		Taxes: []economysim.Tax{
			{ID: "remit", Authority: "crown", Target: economysim.FlowTarget{Kind: "treasury", Jurisdiction: "inland"}, Amount: 7},
			{ID: "tithe", Authority: "crown", Target: economysim.FlowTarget{Kind: "households", Region: "north", Role: "cook"}, RateBP: 100,
				Exempt: 2000, Per: "year", Window: economysim.Window{Start: 4, Until: 90, Seasons: []string{"winter", "spring"}}, Label: "Tithe"},
		},
		Duties: []economysim.Duty{{ID: "sales", Authority: "crown", Region: "north", Commodity: "staple", Payer: "seller", Scope: "domestic", RateBP: 300, Label: "Sales"}},
		Tolls:  []economysim.Toll{{ID: "ferry", Authority: "inland", On: "exit", Mode: "foot", Amount: 4}},
		Payouts: []economysim.Payout{
			{ID: "alms", Authority: "crown", Target: economysim.FlowTarget{Kind: "households", Region: "capital"}, Commodity: "staple", Amount: 2, Per: "tick"},
			{ID: "top", Authority: "crown", Target: economysim.FlowTarget{Kind: "pool"}, Level: 500},
		},
		Purchases: []economysim.Purchase{{ID: "buy", Authority: "crown", Commodity: "staple", Quantity: 500, LimitBP: 8000, Hold: 6000,
			Window: economysim.Window{Seasons: []string{"autumn"}}}},
		Releases: []economysim.Release{{ID: "sell", Authority: "crown", Commodity: "staple", Quantity: 500, LimitBP: 12000, Reserve: 100}},
	}
}

// TestLoad_Fiscal verifies that the tables of spec-fiscal.md 3.1 load: the six kinds in
// ID order with every key, the target as an inline table, the window, a key-less
// [modules.fiscal] table switching the module on with no flow, no table leaving it off,
// two spellings of one module hashing alike, and the file and the Go value hashing alike.
//
// Requirement: FR-FIS-005, FR-CFG-093
func TestLoad_Fiscal(t *testing.T) {
	t.Parallel()
	got, err := parseStrings(fiscalDoc)
	if err != nil {
		t.Fatalf("Parse(fiscalDoc) error = %v, want nil", err)
	}
	want := fiscalWant()
	if !reflect.DeepEqual(got.Modules.Fiscal, want) {
		t.Errorf("Parse(fiscalDoc).Modules.Fiscal = %+v, want %+v", got.Modules.Fiscal, want)
	}
	if same, err := parseStrings(fiscalSame); err != nil || economysim.HashConfig(same) != economysim.HashConfig(got) {
		t.Errorf("Parse(fiscalSame) = %+v, %v, want the same hash as fiscalDoc: the lists and the seasons hash in canonical order", same.Modules.Fiscal, err)
	}
	goCfg := economysim.Config{World: economysim.DefaultWorldParams(), Calendar: economysim.DefaultCalendar()}
	goCfg.Modules.Fiscal = want
	if economysim.HashConfig(goCfg) != economysim.HashConfig(got) {
		t.Error("HashConfig of the file differs from that of the same Go value")
	}
	if on, err := parseStrings(minimal + "[modules.fiscal]\n"); err != nil || !reflect.DeepEqual(on.Modules.Fiscal, &economysim.Fiscal{}) {
		t.Errorf("Parse(empty table).Modules.Fiscal = %+v, %v, want the module on and empty", on.Modules.Fiscal, err)
	}
	if off, _ := parseStrings(minimal); off.Modules.Fiscal != nil {
		t.Errorf("Parse(minimal).Modules.Fiscal = %+v, want nil: the module is off", off.Modules.Fiscal)
	}
}

// TestLoad_FiscalRejections verifies the keys and types the loader rejects in the
// fiscal tables, each with the rule and the path of the key.
//
// Requirement: FR-CFG-093
func TestLoad_FiscalRejections(t *testing.T) {
	t.Parallel()
	at := func(more ...string) []string { return append([]string{"modules", "fiscal"}, more...) }
	const head = "authority = \"crown\"\n"
	tests := []struct {
		name string
		doc  string
		rule error
		path []string
	}{
		{"an unknown key of the module", minimal + "[modules.fiscal]\nlevel = 1\n", economysim.ErrUnknownKey, at("level")},
		{"an unknown key of a tax", minimal + "[modules.fiscal.tax.a]\n" + head + "target = { kind = \"pool\" }\nscale = 1\n", economysim.ErrUnknownKey, at("tax", "a", "scale")},
		{"a tax with no authority", minimal + "[modules.fiscal.tax.a]\ntarget = { kind = \"pool\" }\n", economysim.ErrMissingField, at("tax", "a", "authority")},
		{"a tax with no target", minimal + "[modules.fiscal.tax.a]\n" + head, economysim.ErrMissingField, at("tax", "a", "target")},
		{"a target with no kind", minimal + "[modules.fiscal.tax.a]\n" + head + "target = { role = \"cook\" }\n", economysim.ErrMissingField, at("tax", "a", "target", "kind")},
		{"a target with a store", minimal + "[modules.fiscal.tax.a]\n" + head + "target = { kind = \"pool\", store = \"bin\" }\n", economysim.ErrUnknownKey, at("tax", "a", "target", "store")},
		{"a float amount", minimal + "[modules.fiscal.tax.a]\n" + head + "target = { kind = \"pool\" }\namount = 1.5\n", economysim.ErrNotInteger, at("tax", "a", "amount")},
		{"a string window", minimal + "[modules.fiscal.tax.a]\n" + head + "target = { kind = \"pool\" }\nwindow = \"x\"\n", economysim.ErrWrongType, at("tax", "a", "window")},
		{"a window with a stray key", minimal + "[modules.fiscal.tax.a]\n" + head + "target = { kind = \"pool\" }\nwindow = { end = 1 }\n", economysim.ErrUnknownKey, at("tax", "a", "window", "end")},
		{"a number of seasons", minimal + "[modules.fiscal.tax.a]\n" + head + "target = { kind = \"pool\" }\nwindow = { seasons = [1] }\n", economysim.ErrWrongType, at("tax", "a", "window", "seasons", "[0]")},
		{"a duty with no payer", minimal + "[modules.fiscal.duty.a]\n" + head + "rate_bp = 1\n", economysim.ErrMissingField, at("duty", "a", "payer")},
		{"a duty with no rate", minimal + "[modules.fiscal.duty.a]\n" + head + "payer = \"buyer\"\n", economysim.ErrMissingField, at("duty", "a", "rate_bp")},
		{"a toll with no on", minimal + "[modules.fiscal.toll.a]\n" + head + "amount = 1\n", economysim.ErrMissingField, at("toll", "a", "on")},
		{"a toll with no amount", minimal + "[modules.fiscal.toll.a]\n" + head + "on = \"exit\"\n", economysim.ErrMissingField, at("toll", "a", "amount")},
		{"a payout with a boolean not_income of 1", minimal + "[modules.fiscal.payout.a]\n" + head + "target = { kind = \"pool\" }\nnot_income = 1\n", economysim.ErrWrongType, at("payout", "a", "not_income")},
		{"a purchase with no quantity", minimal + "[modules.fiscal.purchase.a]\n" + head + "commodity = \"x\"\nlimit_bp = 1\n", economysim.ErrMissingField, at("purchase", "a", "quantity")},
		{"a release with no limit", minimal + "[modules.fiscal.release.a]\n" + head + "commodity = \"x\"\nquantity = 1\n", economysim.ErrMissingField, at("release", "a", "limit_bp")},
		{"a release with a hold", minimal + "[modules.fiscal.release.a]\n" + head + "commodity = \"x\"\nquantity = 1\nlimit_bp = 1\nhold = 1\n", economysim.ErrUnknownKey, at("release", "a", "hold")},
		{"a scalar where a tax table", minimal + "[modules.fiscal.tax]\na = 1\n", economysim.ErrWrongType, at("tax", "a")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseStrings(tt.doc)
			if err == nil || !hasError(err, tt.rule, tt.path) {
				t.Errorf("Parse(%q) errors = %s, want %v at %q", tt.doc, describe(err), tt.rule, tt.path)
			}
		})
	}
}

// FuzzLoadFiscal verifies, for random tables of a tax, a duty and a purchase, that the
// loader never panics, that a rejected document returns an error, that an accepted one
// loads again to an equal Config with an equal hash, and that Validate does not panic.
//
// Requirement: NFR-FIS-003, FR-CFG-024
func FuzzLoadFiscal(f *testing.F) {
	f.Add("tithe", "crown", "households", int64(100), int64(300), "seller", int64(8000))
	f.Add("", "", "", int64(-1), int64(-1), "", int64(-1))
	f.Add("x", "nowhere", "treasury", int64(1)<<62, int64(1)<<62, "buyer", int64(1)<<62)
	f.Fuzz(func(t *testing.T, id, authority, kind string, rate, dutyRate int64, payer string, limit int64) {
		doc := minimal + "[modules.fiscal.tax." + strconv.Quote(id) + "]\nauthority = " + strconv.Quote(authority) +
			"\ntarget = { kind = " + strconv.Quote(kind) + " }\nrate_bp = " + strconv.FormatInt(rate, 10) +
			"\n[modules.fiscal.duty.d]\nauthority = " + strconv.Quote(authority) + "\npayer = " + strconv.Quote(payer) +
			"\nrate_bp = " + strconv.FormatInt(dutyRate, 10) +
			"\n[modules.fiscal.purchase.p]\nauthority = " + strconv.Quote(authority) + "\ncommodity = \"staple\"\nquantity = 1\nlimit_bp = " +
			strconv.FormatInt(limit, 10) + "\n"
		got, err := parseStrings(doc)
		if err != nil {
			return
		}
		again, err := parseStrings(doc)
		if err != nil || !reflect.DeepEqual(got, again) || economysim.HashConfig(got) != economysim.HashConfig(again) {
			t.Fatalf("Parse(%q) is not repeatable: %v", doc, err)
		}
		if got.Validate() != nil {
			return // Validate must not panic; the classification of its errors is the property of FuzzLoad
		}
		if got.Modules.Fiscal == nil || len(got.Modules.Fiscal.Duties) != 1 {
			t.Fatalf("Validate accepted %q with %+v, want the module with its duty", doc, got.Modules.Fiscal)
		}
	})
}
