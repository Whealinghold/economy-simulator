package ledger_test

import (
	"errors"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// byName maps the name of a built-in reason in spec-ledger.md 7.2 to the
// ledger's constant. The ledger holds no string (plan 1-1 A3), so the test
// carries the names.
var byName = map[string]ledger.Reason{
	"trade": ledger.Trade, "initial": ledger.Initial, "recipe_input": ledger.RecipeInput,
	"recipe_output": ledger.RecipeOutput, "disposal": ledger.Disposal,
	"consumption": ledger.Consumption, "spoilage": ledger.Spoilage, "wear": ledger.Wear,
	"estate_money": ledger.EstateMoney, "estate_goods": ledger.EstateGoods,
	"start_money": ledger.StartMoney, "split": ledger.Split, "merge": ledger.Merge,
	"births": ledger.Births, "starvation": ledger.Starvation,
	"loan_out": ledger.LoanOut, "loan_repay": ledger.LoanRepay, "loan_interest": ledger.LoanInterest,
	"relief": ledger.Relief,
}

// notBuilt lists the reasons of the starting registry that a module owns and
// that the ledger does not have yet, in the staged form of architecture.md
// 9.7. A module's change deletes its entries.
var notBuilt = []string{
	"want_consumption", "outside_budget", "outside_unspent",
	"outside_receipts", "outside_offer", "outside_unsold", "outside_purchases",
	"escrow_in", "escrow_out", "escrow_refund", "wage_escrow", "wage_pay",
	"wage_refund", "rent",
	"title_sale", "premium", "payout", "transit_out", "transit_in", "haul_use",
	"transit_use", "loss", "deaths",
}

// specRow is one parsed row of the registry table of spec-ledger.md 7.2.
type specRow struct {
	names   []string
	of      []string
	class   string
	between string
}

var nameRe = regexp.MustCompile("`([a-z_]+)`")

// readSpecRows parses the starting registry of 7.2 by a plain text parse.
func readSpecRows(t *testing.T) []specRow {
	t.Helper()
	b, err := os.ReadFile("../../doc/specs/spec-ledger.md")
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	text := string(b)
	i := strings.Index(text, "| Reason | Of | Kind | Between | Owner |")
	j := strings.Index(text[i:], "\n\n")
	if i < 0 || j < 0 {
		t.Fatalf("registry table of 7.2 not found")
	}
	var rows []specRow
	for _, line := range strings.Split(text[i:i+j], "\n")[2:] {
		cells := strings.Split(line, "|")
		if len(cells) < 6 || !strings.HasPrefix(strings.TrimSpace(cells[1]), "`") {
			continue
		}
		r := specRow{class: strings.TrimSpace(cells[3]), between: strings.ToLower(strings.TrimSpace(cells[4]))}
		for _, m := range nameRe.FindAllStringSubmatch(cells[1], -1) {
			r.names = append(r.names, m[1])
		}
		for _, w := range strings.Split(cells[2], ",") {
			r.of = append(r.of, strings.TrimSpace(w))
		}
		rows = append(rows, r)
	}
	if len(rows) < 20 {
		t.Fatalf("parsed %d rows of the registry, want at least 20", len(rows))
	}
	return rows
}

// pairing parses a "Between" cell into the set of (from, to) pairs of the
// kinds the ledger builds (Agent, Estate and Boundary), and whether "any
// holder" was said.
func pairing(between string) (from, to []ledger.Kind, back bool) {
	if i := strings.IndexAny(between, ";:"); i >= 0 {
		between = between[:i]
	}
	back = strings.Contains(between, "or back")
	l, r, _ := strings.Cut(between, " to ")
	kinds := func(s string) []ledger.Kind {
		var out []ledger.Kind
		if strings.Contains(s, "any holder") {
			for k := ledger.KindAgent; k < ledger.KindBoundary; k++ {
				out = append(out, k)
			}
		}
		for w, k := range map[string]ledger.Kind{"agent": ledger.KindAgent, "estate": ledger.KindEstate, "boundary": ledger.KindBoundary} {
			if strings.Contains(s, w) {
				out = append(out, k)
			}
		}
		return out
	}
	return kinds(l), kinds(r), back
}

// TestRegistry_AgreesWithSpec verifies, by a plain text parse of the table of
// spec-ledger.md 7.2, that every built reason has the class and the pairing
// of its row for money and for goods, for every ordered pair of account kinds,
// and that a thing the row does not list is refused. births is the one row
// FR-LED-055 narrows to money from an estate to an agent.
//
// Requirement: FR-LED-004, FR-LED-037, FR-LED-038, FR-LED-055
func TestRegistry_AgreesWithSpec(t *testing.T) {
	t.Parallel()
	built := map[string]bool{}
	// listed records the things each built reason has a row for in the spec;
	// a reason such as initial has one row for money and one for goods.
	listed := map[string]map[string]bool{}
	rows := readSpecRows(t)
	for _, row := range rows {
		for _, name := range row.names {
			if listed[name] == nil {
				listed[name] = map[string]bool{}
			}
			for _, of := range row.of {
				listed[name][of] = true
			}
		}
	}
	for _, row := range rows {
		for _, name := range row.names {
			r, ok := byName[name]
			if !ok {
				if !slices.Contains(notBuilt, name) {
					t.Errorf("reason %q of 7.2 is neither built nor in notBuilt", name)
				}
				continue
			}
			built[name] = true
			if name == "births" {
				continue // checked against FR-LED-055 below
			}
			from, to, back := pairing(row.between)
			for _, goods := range []bool{false, true} {
				thing := "money"
				if goods {
					thing = "goods"
				}
				class, has := ledger.ClassOf(r, goods)
				if has != listed[name][thing] {
					t.Errorf("%s of %s: ClassOf has a row = %v, the spec lists it = %v", name, thing, has, listed[name][thing])
					continue
				}
				if !has || !slices.Contains(row.of, thing) {
					continue
				}
				if want := row.class; (class == ledger.Internal && want != "internal") ||
					(class == ledger.Faucet && want != "faucet") || (class == ledger.Sink && want != "sink") {
					t.Errorf("%s of %s: class %d, spec says %q", name, thing, class, want)
				}
				for a := ledger.KindAgent; a <= ledger.KindBoundary; a++ {
					for b := ledger.KindAgent; b <= ledger.KindBoundary; b++ {
						want := slices.Contains(from, a) && slices.Contains(to, b)
						if back {
							want = want || (slices.Contains(from, b) && slices.Contains(to, a))
						}
						if got := ledger.Allowed(r, goods, a, b); got != want {
							t.Errorf("Allowed(%s, %s, kind %d, kind %d) = %v, spec says %v", name, thing, a, b, got, want)
						}
					}
				}
			}
		}
	}
	for name := range byName {
		if !built[name] {
			t.Errorf("built reason %q is not a row of 7.2", name)
		}
	}
	for _, name := range notBuilt {
		if _, ok := byName[name]; ok {
			t.Errorf("notBuilt lists %q, which is built: delete the stale entry", name)
		}
	}

	// FR-LED-055: births is money from an estate to an agent, nothing else.
	for a := ledger.KindAgent; a <= ledger.KindBoundary; a++ {
		for b := ledger.KindAgent; b <= ledger.KindBoundary; b++ {
			want := a == ledger.KindEstate && b == ledger.KindAgent
			if got := ledger.Allowed(ledger.Births, false, a, b); got != want {
				t.Errorf("Allowed(births, money, %d, %d) = %v, want %v", a, b, got, want)
			}
			if ledger.Allowed(ledger.Births, true, a, b) {
				t.Errorf("Allowed(births, goods, %d, %d) = true, want a refusal", a, b)
			}
		}
	}
	if ledger.Allowed(ledger.Reason(200), false, ledger.KindAgent, ledger.KindEstate) {
		t.Errorf("Allowed accepted an unknown reason")
	}
	if ledger.Allowed(ledger.Trade, false, ledger.Kind(50), ledger.KindAgent) {
		t.Errorf("Allowed accepted an unknown kind")
	}
}

// TestTransfer_FollowsRegistry verifies that Transfer accepts a money
// transfer for exactly the pairs Allowed names, and refuses every other with
// ErrReasonMismatch, for every reason and every ordered pair of kinds.
//
// Requirement: FR-LED-004
func TestTransfer_FollowsRegistry(t *testing.T) {
	t.Parallel()
	for r := ledger.Reason(0); r < ledger.NumReasons; r++ {
		for a := ledger.KindAgent; a <= ledger.KindBoundary; a++ {
			for b := ledger.KindAgent; b <= ledger.KindBoundary; b++ {
				f := newFix(t, testMaxMoney)
				// One funded account of each non-boundary kind, in region 0.
				acct := map[ledger.Kind]int{ledger.KindBoundary: f.mint}
				for k := ledger.KindAgent; k < ledger.KindBoundary; k++ {
					id, err := f.l.Open(k, 0, 0)
					if err != nil {
						t.Fatalf("Open(%d): %v", k, err)
					}
					acct[k] = id
					// Only the kinds Initial reaches can be funded; the
					// others are refused before funds are looked at.
					if ledger.Allowed(ledger.Initial, false, ledger.KindBoundary, k) {
						if err := f.l.Transfer(f.mint, id, 10, ledger.Initial); err != nil {
							t.Fatalf("fund: %v", err)
						}
					}
				}
				// The boundary of Initial is the only money boundary; for any
				// other reason it does not fit, as the mismatch it should be.
				from, to := acct[a], acct[b]
				if from == to {
					continue
				}
				err := f.l.Transfer(from, to, 1, r)
				_, hasRow := ledger.ClassOf(r, false)
				wantOK := hasRow && ledger.Allowed(r, false, a, b)
				if a == ledger.KindBoundary || b == ledger.KindBoundary {
					wantOK = wantOK && r == ledger.Initial
				}
				switch {
				case wantOK && err != nil:
					t.Errorf("Transfer(reason %d, kind %d -> %d) = %v, want nil", r, a, b, err)
				case !wantOK && !errors.Is(err, ledger.ErrReasonMismatch):
					t.Errorf("Transfer(reason %d, kind %d -> %d) = %v, want ErrReasonMismatch", r, a, b, err)
				}
			}
		}
	}
}

// TestRegistry_RowsAndFlowSlots verifies the layout guarantee that a
// money-only reason has no goods slot: a ledger with no commodity still
// totals money, and a goods reason has no money slot.
//
// Requirement: FR-LED-023, FR-LED-052
func TestRegistry_RowsAndFlowSlots(t *testing.T) {
	t.Parallel()
	l, err := ledger.New(ledger.Config{Regions: 1, MaxMoney: 10, MaxQuantity: 10})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := l.OpenBoundary(ledger.Initial, 0); !errors.Is(err, ledger.ErrUnknownCommodity) {
		t.Errorf("OpenBoundary of commodity 0 with none configured = %v, want ErrUnknownCommodity", err)
	}
	if _, err := l.OpenBoundary(ledger.Initial, ledger.Money); err != nil {
		t.Errorf("OpenBoundary(Initial, Money) = %v, want nil", err)
	}
	if _, err := l.OpenBoundary(ledger.Spoilage, ledger.Money); !errors.Is(err, ledger.ErrReasonMismatch) {
		t.Errorf("OpenBoundary(Spoilage, Money) = %v, want ErrReasonMismatch (goods only)", err)
	}
}

// TestRegistry_Relief verifies the relief row of spec-ledger.md 7.2: a transfer
// by relief from an estate to an agent is accepted when the reason is declared,
// every other pairing is ErrReasonMismatch, an undeclared ledger refuses it, and
// declaring it opens no boundary account.
//
// Requirement: FR-LED-052, FR-EST-016
func TestRegistry_Relief(t *testing.T) {
	t.Parallel()
	// The estate is funded with 40 by the initial faucet, so a payment can clear.
	fund := func(f *fix) {
		t.Helper()
		if err := f.l.Transfer(f.mint, f.e0, 40, ledger.Initial); err != nil {
			t.Fatalf("fund the estate: %v", err)
		}
	}
	tests := []struct {
		name     string
		from, to func(f *fix) int
		want     error
	}{
		{"estate to agent", func(f *fix) int { return f.e0 }, func(f *fix) int { return f.a0 }, nil},
		{"agent to estate", func(f *fix) int { return f.a0 }, func(f *fix) int { return f.e0 }, ledger.ErrReasonMismatch},
		{"agent to agent", func(f *fix) int { return f.a0 }, func(f *fix) int { return f.a1 }, ledger.ErrReasonMismatch},
		{"boundary to agent", func(f *fix) int { return f.mint }, func(f *fix) int { return f.a1 }, ledger.ErrReasonMismatch},
		{"estate to boundary", func(f *fix) int { return f.e0 }, func(f *fix) int { return f.mint }, ledger.ErrReasonMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFix(t, testMaxMoney)
			fund(f)
			err := f.l.Transfer(tt.from(f), tt.to(f), 5, ledger.Relief)
			if !errors.Is(err, tt.want) || (tt.want == nil && err != nil) {
				t.Errorf("Transfer(relief, %s) = %v, want %v", tt.name, err, tt.want)
			}
		})
	}

	t.Run("an undeclared ledger refuses it", func(t *testing.T) {
		t.Parallel()
		l, err := ledger.New(ledger.Config{Regions: 1, MaxMoney: testMaxMoney, MaxQuantity: testMaxQuantity})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		mint, _ := l.OpenBoundary(ledger.Initial, ledger.Money)
		est, _ := l.Open(ledger.KindEstate, 0, 0)
		ag, _ := l.Open(ledger.KindAgent, 1, 0)
		if err := l.Transfer(mint, est, 10, ledger.Initial); err != nil {
			t.Fatalf("fund: %v", err)
		}
		if err := l.Transfer(est, ag, 5, ledger.Relief); !errors.Is(err, ledger.ErrReasonMismatch) {
			t.Errorf("Transfer(relief) on a ledger that did not declare it = %v, want ErrReasonMismatch", err)
		}
	})

	t.Run("declaring it opens no boundary account", func(t *testing.T) {
		t.Parallel()
		l, err := ledger.New(ledger.Config{Regions: 1, MaxMoney: testMaxMoney, MaxQuantity: testMaxQuantity, Declared: []ledger.Reason{ledger.Relief}})
		if err != nil {
			t.Fatalf("New(Declared: relief) = %v, want nil", err)
		}
		if _, err := l.OpenBoundary(ledger.Relief, ledger.Money); err == nil {
			t.Errorf("OpenBoundary(relief) = nil, want an error: an internal reason has no boundary account")
		}
	})
}
