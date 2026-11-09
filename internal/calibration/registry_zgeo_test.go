package calibration

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// ownerDefault is the Default and Bound cells of a field's row in the owner
// spec's parameter table, which the checks of z, geo and the off number read.
type ownerDefault struct{ def, bound string }

var reBound = regexp.MustCompile("^\\s*(-?[0-9][0-9 ]*|`?[A-Za-z]+`?)\\s+to\\s+(.+)$")

// ownerRow finds the row of field in the spec text. scope, when given, skips a
// row of a table whose first column is another scope (the owner spec has one
// table per scope, with the field in the second column).
func ownerRow(text, field, scope string) (ownerDefault, bool) {
	lines := strings.Split(text, "\n")
	scoped := firstCellsOf(lines, field)
	re := regexp.MustCompile("^\\|(?:[^|]*\\|?|\\s*)\\s*`" + regexp.QuoteMeta(field) + "`\\s*\\|")
	for n, ln := range lines {
		if c := cells(ln); scope != "" && len(c) > 1 && c[0] != scope && scoped[scope] {
			continue
		}
		if re.MatchString(ln) {
			if d, ok := headerOf(lines, n); ok {
				return d, true
			}
		}
	}
	return ownerDefault{}, false
}

// firstCellsOf is the set of first cells of the rows whose second cell is field.
func firstCellsOf(lines []string, field string) map[string]bool {
	out := map[string]bool{}
	for _, l := range lines {
		if !strings.HasPrefix(l, "|") {
			continue
		}
		if c := cells(l); len(c) > 1 && c[1] == field {
			out[c[0]] = true
		}
	}
	return out
}

// headerOf looks up to 40 lines above n for the table header with a Default
// column and reads the row at n against it.
func headerOf(lines []string, n int) (ownerDefault, bool) {
	for k := n; k > max(0, n-40); k-- {
		if !strings.HasPrefix(lines[k], "|") || !strings.Contains(lines[k], "Default") {
			continue
		}
		hdr, row := cells(lines[k]), cells(lines[n])
		di := indexOf(hdr, "Default")
		if di < 0 || len(row) != len(hdr) {
			return ownerDefault{}, false
		}
		d := ownerDefault{def: row[di]}
		if bi := indexOf(hdr, "Bound"); bi >= 0 {
			d.bound = row[bi]
		}
		return d, true
	}
	return ownerDefault{}, false
}

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

// boundRange reads "lo to hi" of a Bound cell: a number, or ok false for a
// named constant.
func boundRange(bound string) (lo, hi int, loOK, hiOK bool) {
	m := reBound.FindStringSubmatch(bound)
	if m == nil {
		return 0, 0, false, false
	}
	hiText := strings.SplitN(strings.SplitN(m[2], ",", 2)[0], ";", 2)[0]
	lo, loOK = atoi(m[1])
	hi, hiOK = atoi(hiText)
	return lo, hi, loOK, hiOK
}

var reZero = regexp.MustCompile(`(?i)^(0\b|none|nil|absent|empty|\(none\))`)

// zgeoProblems checks one knob row against the owner's default (FR-CLB-108):
// the flag z only where the default is 0, none or the world's; a geo row has a
// default of at least 1 or z; a required default has a lowest legal value of at
// least 1; an off number is not the owner's bound.
func zgeoProblems(k []string, d ownerDefault) []string {
	id, flags := k[cID], words(k[cFlags])
	z := indexOf(flags, "z") >= 0
	ds := strings.ToLower(strings.TrimSpace(d.def))
	zero := reZero.MatchString(ds)
	worlds := strings.HasPrefix(ds, "the world")
	var out []string
	if k[cGrid] == "geo" {
		out = append(out, geoProblems(id, ds, d.bound, z, zero || worlds)...)
	}
	if z && !(zero || worlds) {
		out = append(out, id+": z needs an owner default of 0, none or the world's")
	}
	return append(out, offBoundProblems(id, flags, d.bound)...)
}

// geoProblems is the geo part of zgeoProblems: a required default needs a lowest
// legal value of at least 1 and no z; any other default below 1 needs z.
func geoProblems(id, ds, bound string, z, zeroLike bool) []string {
	if strings.HasPrefix(ds, "required") {
		return requiredGeoProblems(id, bound, z)
	}
	if n, isNum := atoi(ds); (zeroLike || (isNum && n < 1)) && !z {
		return []string{id + ": a geo row with an owner default below 1 and no z"}
	}
	return nil
}

func requiredGeoProblems(id, bound string, z bool) []string {
	lo, _, loOK, _ := boundRange(bound)
	switch {
	case z:
		return []string{id + ": z on a required default"}
	case !loOK || lo < 1:
		return []string{id + ": a required default of a geo row has a lowest legal value below 1"}
	}
	return nil
}

// offBoundProblems checks an off number against the owner's bound: the highest
// value is written off=max and the lowest off=min (0 and 1 are exempt).
func offBoundProblems(id string, flags []string, bound string) []string {
	lo, hi, loOK, hiOK := boundRange(bound)
	if !loOK || !hiOK {
		return nil
	}
	var out []string
	for _, f := range flags {
		v, ok := strings.CutPrefix(f, "off=")
		num, err := strconv.Atoi(v)
		if !ok || err != nil {
			continue
		}
		if num == hi {
			out = append(out, id+": off equals the owner's highest value: write off=max")
		}
		if num == lo && num != 0 && num != 1 {
			out = append(out, id+": off equals the owner's lowest value: write off=min")
		}
	}
	return out
}

// checkZGeo is FR-CLB-108 for every knob row whose owner default is found.
func checkZGeo(r *registry, files map[string]string, specText func(string) string) []string {
	var out []string
	for _, k := range r.knobs {
		code, _, _ := strings.Cut(k[cOwner], " ")
		file := files[code]
		if file == "" {
			continue
		}
		scope, field := keyParts(k[cKey])
		scope = scope[strings.LastIndexByte(scope, '.')+1:]
		d, ok := ownerRow(specText(file), field, scope)
		if ok {
			out = append(out, zgeoProblems(k, d)...)
		}
	}
	return out
}

// TestRegistry_ZGeoAndOut verifies that the flag z is only on a row whose owner
// default is 0, none or the world's, that a geo grid needs a default of at least
// 1 or z, that a required default has a lowest legal value of at least 1, that
// an off number is not the owner's bound, and that `out` is only on the five
// fields the solver writes.
//
// Requirement: FR-CLB-108, FR-CLB-109
func TestRegistry_ZGeoAndOut(t *testing.T) {
	t.Parallel()
	s := loadSpecs(t)
	r := parseRegistry(s.clb)
	files := codeToFile(s.readme)
	text := func(name string) string { return readDoc(t, "specs/"+name) }
	if got := checkZGeo(r, files, text); len(got) > 0 {
		t.Errorf("checkZGeo(spec) = %v, want none", got)
	}
	geoRow := []string{"K-X", "World.X", "K", "BEL 11", "bp", "U", "geo", "C4", "-", "-", "-", "-"}
	cases := []struct {
		name string
		row  []string
		d    ownerDefault
	}{
		{"geo on a default of 0 with no z", geoRow, ownerDefault{def: "0", bound: "0 to 10 000"}},
		{"z on a default of 5", with(geoRow, cFlags, "z"), ownerDefault{def: "5", bound: "0 to 10 000"}},
		{"required default with a lowest value of 0", geoRow, ownerDefault{def: "required", bound: "0 to 10 000"}},
		{"off equals the owner's highest value", with(geoRow, cFlags, "off=10000"), ownerDefault{def: "5", bound: "1 to 10000"}},
		{"off equals the owner's lowest value", with(geoRow, cFlags, "off=3"), ownerDefault{def: "5", bound: "3 to 10000"}},
	}
	for _, tc := range cases {
		if got := zgeoProblems(tc.row, tc.d); len(got) == 0 {
			t.Errorf("zgeoProblems(%s) found nothing, want a problem", tc.name)
		}
	}
	if got := zgeoProblems(with(geoRow, cFlags, "z"), ownerDefault{def: "0", bound: "0 to 10 000"}); len(got) > 0 {
		t.Errorf("zgeoProblems(z on a default of 0) = %v, want none", got)
	}
	if got := checkOut(r); len(got) > 0 {
		t.Errorf("checkOut(spec) = %v, want none", got)
	}
	c := r.clone()
	c.groups[0][gFlags] = "out"
	if len(checkOut(c)) == 0 {
		t.Errorf("checkOut with out on a field the solver does not write found nothing")
	}
	c = r.clone()
	c.knobs[0][cFlags] = "out"
	if len(checkOut(c)) == 0 {
		t.Errorf("checkOut with out on a knob row found nothing")
	}
}

func with(row []string, cell int, v string) []string {
	c := append([]string(nil), row...)
	c[cell] = v
	return c
}

// outOK are the fields the solver writes (VIA 8, REF position 4); `Pools` is
// not one: the author may seed it (VIA 8.3.3).
func outOK() map[pair]bool {
	return map[pair]bool{
		{"Commodity", "ReferencePrice"}: true, {"RegionPrice", "Price"}: true,
		{"WarmStart", "Prices"}: true, {"WarmStart", "Wages"}: true, {"WarmStart", "Groups"}: true,
	}
}

// checkOut is FR-CLB-109: the flag `out` is only on a field of outOK.
func checkOut(r *registry) []string {
	var out []string
	ok := outOK()
	for _, k := range r.knobs {
		s, f := keyParts(k[cKey])
		if indexOf(words(k[cFlags]), "out") >= 0 && !ok[pair{s, f}] {
			out = append(out, k[cID]+": out on "+k[cKey])
		}
	}
	for _, g := range r.groups {
		if indexOf(words(g[gFlags]), "out") < 0 {
			continue
		}
		for _, f := range strings.Split(g[gFields], ",") {
			if !ok[pair{g[gScope], strings.TrimSpace(f)}] {
				out = append(out, g[gID]+": out on "+g[gScope]+"."+strings.TrimSpace(f))
			}
		}
	}
	return out
}
