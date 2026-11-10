package verdict

import (
	"slices"
	"strconv"
	"strings"
)

// FactRow is one row of a verdict record (STY 8.5): what a fact came to over
// the seeds of one scenario, optionally for one region and role.
type FactRow struct {
	ID    string
	Class FactClass
	// Statistic names the measure: a band key ("s01.recover_ticks") or a word
	// ("sign").
	Statistic string
	// Region and Role are empty when the row is not per region or role.
	Region, Role string
	// NotApplicable marks a fact whose module is absent (STY 4.6, FR-STY-009);
	// Reason says which.
	NotApplicable bool
	Reason        string
	// N is the number of seeds planned; Outside lists the seeds outside the
	// band or that failed, ascending.
	N       int
	Outside []int64
	// BandSet and Band: the band value, when the key has one.
	BandSet bool
	Band    int64
	// HasPrior and Prior: the prior of the spec, shown beside an unset band.
	HasPrior bool
	Prior    int64
	// HasValues and Min, Median, Max: the observed values over the seeds.
	HasValues        bool
	Min, Median, Max int64
}

// ScenarioRecord is the rows of one scenario of a pack.
type ScenarioRecord struct {
	Scenario string
	Facts    []FactRow
	// Regimes holds the verdict of each detector that ran.
	Regimes []Verdict
}

// Record is the verdict record of one pack (STY 8.5): plain text of integers,
// the golden file testdata/golden/stylised/<pack>.txt.
type Record struct {
	Pack string
	// Seeds is the seed range as written, "1..10".
	Seeds          string
	Commit, Config string
	Scenarios      []ScenarioRecord
}

// RenderVerdict renders a verdict record. The text is a function of the record
// alone: the scenarios, the rows and the regimes are put in a fixed order (pack,
// scenario, fact ID, region, role; detectors in the order of STY 6.2), and the
// text holds no float, no wall-clock time and no map order (FR-STY-085,
// NFR-STY-002).
func RenderVerdict(r Record) string {
	r = flatRecord(r)
	var b strings.Builder
	scs := slices.Clone(r.Scenarios)
	slices.SortStableFunc(scs, func(x, y ScenarioRecord) int { return strings.Compare(x.Scenario, y.Scenario) })
	for _, sc := range scs {
		b.WriteString("pack " + r.Pack + "  scenario " + sc.Scenario + "  seeds " + r.Seeds + "  commit " + r.Commit + "  config " + r.Config + "\n")
		facts := slices.Clone(sc.Facts)
		slices.SortStableFunc(facts, compareRows)
		for _, f := range facts {
			renderRow(&b, f)
		}
		renderRegimes(&b, sc.Regimes)
	}
	return b.String()
}

func compareRows(x, y FactRow) int {
	for _, c := range []int{
		strings.Compare(x.ID, y.ID), strings.Compare(x.Region, y.Region), strings.Compare(x.Role, y.Role),
		strings.Compare(x.Statistic, y.Statistic), strings.Compare(string(x.Class), string(y.Class)),
	} {
		if c != 0 {
			return c
		}
	}
	return 0
}

// valuesIndent aligns the second line of a row under the statistic.
const valuesIndent = "                       "

func renderRow(b *strings.Builder, f FactRow) {
	b.WriteString("fact " + f.ID + "  class " + string(f.Class) + "  " + f.Statistic)
	if f.Region != "" {
		b.WriteString("  region " + f.Region)
	}
	if f.Role != "" {
		b.WriteString("  role " + f.Role)
	}
	switch {
	case f.NotApplicable:
		b.WriteString("  not applicable (" + f.Reason + ")\n")
		return
	case f.Class == ClassR:
		b.WriteString("  report\n")
	case !f.BandSet && (f.Class == ClassB || f.Class == ClassP):
		b.WriteString("  band unset  prior " + priorWord(f) + "\n")
	default:
		if f.BandSet {
			b.WriteString("  band " + strconv.FormatInt(f.Band, 10))
		}
		ok := "invalid" // more seeds outside than seeds: a caller bug the record must show
		if len(f.Outside) <= f.N {
			ok = strconv.Itoa(f.N - len(f.Outside))
		}
		b.WriteString("  ok " + ok + "/" + strconv.Itoa(f.N) + "\n")
	}
	if f.HasValues {
		b.WriteString(valuesIndent + "values min " + i64(f.Min) + " median " + i64(f.Median) + " max " + i64(f.Max) + outsideWord(f) + "\n")
	}
}

func i64(v int64) string { return strconv.FormatInt(v, 10) }

func priorWord(f FactRow) string {
	if f.HasPrior {
		return i64(f.Prior)
	}
	return "none"
}

// outsideWord counts the seeds outside the band, or outside the prior while the
// band is unset, and lists them. A row with neither a band nor a prior has
// nothing to be outside of.
func outsideWord(f FactRow) string {
	word := "outside"
	switch {
	case f.BandSet:
	case f.HasPrior:
		word = "outside-prior"
	default:
		return ""
	}
	s := "  " + word + " " + strconv.Itoa(len(f.Outside))
	if len(f.Outside) == 0 {
		return s
	}
	parts := make([]string, len(f.Outside))
	for i, o := range f.Outside {
		parts[i] = i64(o)
	}
	label := " (seed "
	if len(parts) > 1 {
		label = " (seeds "
	}
	return s + label + strings.Join(parts, ", ") + ")"
}

func renderRegimes(b *strings.Builder, vs []Verdict) {
	if len(vs) == 0 {
		return
	}
	order := Detectors()
	vs = slices.Clone(vs)
	rank := func(d Detector) int { // an unknown detector sorts last, not first
		if i := slices.Index(order, d); i >= 0 {
			return i
		}
		return len(order)
	}
	slices.SortStableFunc(vs, func(x, y Verdict) int { return rank(x.Detector) - rank(y.Detector) })
	parts := make([]string, len(vs))
	for i, v := range vs {
		parts[i] = v.String()
	}
	b.WriteString("regime " + strings.Join(parts, "   ") + "\n")
}

// The claim words of a stability record. A default is called stable only by
// this procedure (STY 7.3.1); an unexamined neighbour is UNDETERMINED (7.2).
const (
	ClaimStable       = "stable"
	ClaimNotStable    = "not_stable"
	ClaimOneNeighbour = "one_neighbour"
	ClaimUndetermined = "undetermined"
)

// StabilityPoint is the three checks at one point of a knob: the default or a
// neighbour.
type StabilityPoint struct {
	// Name is "default", "lower" or "upper".
	Name   string
	Checks [3]Result
}

// Stability is the stability record of a default (STY 7.3.3).
type Stability struct {
	Pack           string
	Seeds          string
	Commit, Config string
	Points         []StabilityPoint
}

// ClaimOf derives the claim line from the points (STY 7.3.1, 7.2): any check
// NOT STABLE gives not_stable; any other check not STABLE gives undetermined;
// with all checks STABLE, the default and both neighbours give stable, the
// default and one neighbour (a default at the edge of its grid) give
// one_neighbour, and the default alone gives undetermined, since the
// neighbour is unexamined. A record with no default point is undetermined.
func ClaimOf(points []StabilityPoint) string {
	hasDefault := false
	open := false
	others := map[string]bool{}
	for _, p := range points {
		hasDefault = hasDefault || p.Name == "default"
		if p.Name == "lower" || p.Name == "upper" { // any other name is not a neighbour
			others[p.Name] = true
		}
		for _, c := range p.Checks {
			if c == NotStable {
				return ClaimNotStable
			}
			open = open || c != Stable
		}
	}
	switch {
	case !hasDefault || open:
		return ClaimUndetermined
	case len(others) >= 2:
		return ClaimStable
	case len(others) == 1:
		return ClaimOneNeighbour
	}
	return ClaimUndetermined
}

// RenderStability renders a stability record in the form of the verdict record
// (STY 8.5, 7.3.3): the header, one `point` line per point with the three
// checks, and the `claim` line. The same input gives the same bytes. The
// points are written in the order given.
func RenderStability(s Stability) string {
	var b strings.Builder
	b.WriteString("stability pack " + s.Pack + "  seeds " + s.Seeds + "  commit " + s.Commit + "  config " + s.Config + "\n")
	for _, p := range s.Points {
		b.WriteString("point " + p.Name)
		for i, name := range Checks() {
			b.WriteString("  " + name + " " + p.Checks[i].String())
		}
		b.WriteString("\n")
	}
	b.WriteString("claim " + ClaimOf(s.Points) + "\n")
	return b.String()
}

// oneLine collapses every run of white space to one space, so a caller's string
// cannot add a line or the double-space field separator to the record.
func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// flatRecord returns r with every free-text field made one line.
func flatRecord(r Record) Record {
	r.Pack, r.Seeds, r.Commit, r.Config = oneLine(r.Pack), oneLine(r.Seeds), oneLine(r.Commit), oneLine(r.Config)
	r.Scenarios = slices.Clone(r.Scenarios)
	for i, sc := range r.Scenarios {
		sc.Scenario = oneLine(sc.Scenario)
		sc.Facts = slices.Clone(sc.Facts)
		for j, f := range sc.Facts {
			f.ID, f.Statistic, f.Region, f.Role, f.Reason = oneLine(f.ID), oneLine(f.Statistic), oneLine(f.Region), oneLine(f.Role), oneLine(f.Reason)
			sc.Facts[j] = f
		}
		sc.Regimes = slices.Clone(sc.Regimes)
		for j, v := range sc.Regimes {
			v.Reason, v.Detector = oneLine(v.Reason), Detector(oneLine(string(v.Detector)))
			sc.Regimes[j] = v
		}
		r.Scenarios[i] = sc
	}
	return r
}
