package calibration

import (
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

// ownerInfo is what the parameter-owner table of specs/README.md says about one
// (scope token, parameter) pair: the owner spec file and its sections.
type ownerInfo struct {
	file string
	secs []string
}

type pair struct{ scope, field string }

var (
	reOwnerSpec = regexp.MustCompile("`(spec-[a-z\\-]+\\.md)` sections? ([0-9][0-9., and]*)")
	reSecNum    = regexp.MustCompile(`[0-9]+(?:\.[0-9]+)*`)
	reAlso      = regexp.MustCompile(`\(also ([A-Z][A-Za-z ]*?)[,)]`)
	reOwnerCell = regexp.MustCompile(`^([A-Z]{3}) ([0-9][0-9.]*)$`)
	reFirstTick = regexp.MustCompile("`([^`]+)`")
	reCodeFile  = regexp.MustCompile("(?m)^\\| `([A-Z]{3})` \\| `(spec-[a-z\\-]+\\.md)`")
)

// scopeAlias maps the owner table's spelling of a scope to the registry's: the
// owner table says WorldParams, the Go type of Config.World, and the registry
// says World, the key of [world] (spec-calibration.md 4.7, N-329).
func scopeAlias(scope string) string {
	if scope == "WorldParams" {
		return "World"
	}
	return scope
}

// ownerTable reads the parameter-owner table of specs/README.md as text. A row
// that names no owner section is returned in noSection.
func ownerTable(readme string) (pairs map[pair]ownerInfo, noSection []pair) {
	pairs = map[pair]ownerInfo{}
	sect := between(readme, "## Parameter owners", "## New domain terms")
	for _, line := range strings.Split(sect, "\n") {
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		cs := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
		if len(cs) < 3 {
			continue
		}
		pm := reFirstTick.FindStringSubmatch(cs[0])
		if pm == nil {
			continue
		}
		param, _, _ := strings.Cut(pm[1], " (")
		scopeCell := strings.TrimSpace(cs[1])
		scope := scopeCell
		if i := strings.IndexAny(scope, ";("); i >= 0 {
			scope = scope[:i]
		}
		scope = scopeAlias(strings.TrimSpace(scope))
		info := ownerInfo{}
		if m := reOwnerSpec.FindStringSubmatch(cs[2]); m != nil {
			info = ownerInfo{m[1], reSecNum.FindAllString(m[2], -1)}
		} else {
			noSection = append(noSection, pair{scope, param})
		}
		pairs[pair{scope, param}] = info
		if m := reAlso.FindStringSubmatch(scopeCell); m != nil {
			pairs[pair{scopeAlias(strings.TrimSpace(m[1])), param}] = info
		}
	}
	return pairs, noSection
}

// codeToFile maps an area code to its spec file, from the README index.
func codeToFile(readme string) map[string]string {
	out := map[string]string{"CUR": "spec-currency.md", "CAR": "spec-carriage.md"}
	for _, m := range reCodeFile.FindAllStringSubmatch(readme, -1) {
		out[m[1]] = m[2]
	}
	return out
}

// covered returns every (scope, field) that the registry covers: a knob row's
// key and each field of a group.
func (r *registry) covered() map[pair]bool {
	out := map[pair]bool{}
	for _, k := range r.knobs {
		s, f := keyParts(k[cKey])
		out[pair{s, f}] = true
	}
	for _, g := range r.groups {
		for _, f := range strings.Split(g[gFields], ",") {
			out[pair{g[gScope], strings.TrimSpace(f)}] = true
		}
	}
	return out
}

func sortedPairs(m map[pair]bool) []pair {
	out := make([]pair, 0, len(m))
	for p := range m {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].scope != out[j].scope {
			return out[i].scope < out[j].scope
		}
		return out[i].field < out[j].field
	})
	return out
}

// checkOwnerToRegistry is FR-CLB-002, first direction: every owner pair is a
// knob row's key or a field of a group of that scope.
func checkOwnerToRegistry(r *registry, owners map[pair]ownerInfo) []string {
	have := r.covered()
	var out []string
	for _, p := range sortedPairs(keysOf(owners)) {
		if !have[p] {
			out = append(out, "owner pair "+p.scope+"."+p.field+" is in no knob row or group")
		}
	}
	return out
}

func keysOf(owners map[pair]ownerInfo) map[pair]bool {
	out := map[pair]bool{}
	for p := range owners {
		out[p] = true
	}
	return out
}

// checkRegistryToOwner is FR-CLB-002, second direction: a key or a group field
// names an owner row, outside the scopes "Solve option" and "Presentation" and
// the catalogue-only fields of the demand module.
func checkRegistryToOwner(r *registry, owners map[pair]ownerInfo) []string {
	catalogueOnly := map[pair]bool{
		{"Taste", "Role"}: true, {"RegionTaste", "Region"}: true,
		{"RegionTaste", "Want"}: true, {"RegionTaste", "Scale"}: true,
	}
	var out []string
	for _, p := range sortedPairs(r.covered()) {
		_, ok := owners[p]
		if ok || p.scope == "Solve option" || p.scope == "Presentation" || catalogueOnly[p] {
			continue
		}
		out = append(out, "registry pair "+p.scope+"."+p.field+" names no owner row")
	}
	return out
}

// TestRegistry_CoverageOwnerToRegistry verifies that every row of the
// parameter-owner table, and every "(also <Scope>" remark of one, is covered by
// exactly the registry (4.7, 4.8), and that every row parses an owner section.
//
// Requirement: FR-CLB-002, FR-CLB-102
func TestRegistry_CoverageOwnerToRegistry(t *testing.T) {
	t.Parallel()
	s := loadSpecs(t)
	r := parseRegistry(s.clb)
	owners, noSection := ownerTable(s.readme)
	if len(owners) < 400 {
		t.Fatalf("parsed %d owner pairs, want at least 400", len(owners))
	}
	if len(noSection) > 0 {
		t.Errorf("owner rows without a section: %v", noSection)
	}
	if got := checkOwnerToRegistry(r, owners); len(got) > 0 {
		t.Errorf("checkOwnerToRegistry(spec) = %v, want none", got)
	}
	// An owner row with no registry cover: the registry loses the key of a row.
	c := r.clone()
	c.knobs = c.knobs[1:]
	if len(checkOwnerToRegistry(c, owners)) == 0 {
		t.Errorf("dropping a knob row did not fail the owner-to-registry check")
	}
	// A new owner pair the registry does not cover: a new Config field.
	owners[pair{"Group", "NewField"}] = ownerInfo{}
	if len(checkOwnerToRegistry(r, owners)) == 0 {
		t.Errorf("a new owner pair did not fail the owner-to-registry check")
	}
}

// TestRegistry_CoverageRegistryToOwner verifies that a registry key or group
// field names an owner row.
//
// Requirement: FR-CLB-002
func TestRegistry_CoverageRegistryToOwner(t *testing.T) {
	t.Parallel()
	s := loadSpecs(t)
	r := parseRegistry(s.clb)
	owners, _ := ownerTable(s.readme)
	if got := checkRegistryToOwner(r, owners); len(got) > 0 {
		t.Errorf("checkRegistryToOwner(spec) = %v, want none", got)
	}
	c := r.clone()
	c.knobs[0][cKey] = "Nowhere.Nothing"
	if len(checkRegistryToOwner(c, owners)) == 0 {
		t.Errorf("a key with no owner row did not fail the registry-to-owner check")
	}
	c = r.clone()
	c.groups[0][gScope] = "Recipe tool" // the scope token the owner table no longer spells
	if len(checkRegistryToOwner(c, owners)) == 0 {
		t.Errorf("a stale scope token did not fail the registry-to-owner check")
	}
}

// ownerSectionProblems checks one Owner cell ("PRO 3.2"): its code is an area,
// its section is a heading of that spec, and, for the pair, one of the README's
// sections for the same owner.
func ownerSectionProblems(id, cell string, p pair, files map[string]string, owners map[pair]ownerInfo, specText func(string) string) []string {
	m := reOwnerCell.FindStringSubmatch(cell)
	if m == nil || files[m[1]] == "" {
		return []string{id + ": owner cell " + cell + " does not resolve"}
	}
	num := regexp.QuoteMeta(m[2])
	text := specText(files[m[1]])
	if !regexp.MustCompile(`(?m)^(#+ )?` + num + `([. ]|$)`).MatchString(text) {
		return []string{id + ": section " + cell + " is not in " + files[m[1]]}
	}
	own, ok := owners[p]
	if ok && own.file != "" && (own.file != files[m[1]] || !secOK(m[2], own.secs)) {
		return []string{id + ": " + cell + " is not a README owner section of " + p.scope + "." + p.field}
	}
	return nil
}

func secOK(mine string, theirs []string) bool {
	return slices.ContainsFunc(theirs, func(t string) bool {
		return mine == t || strings.HasPrefix(mine, t+".") || strings.HasPrefix(t, mine+".")
	})
}

// checkOwnerSections is FR-CLB-102 for the knob rows.
func checkOwnerSections(r *registry, files map[string]string, owners map[pair]ownerInfo, specText func(string) string) []string {
	var out []string
	for _, k := range r.knobs {
		s, f := keyParts(k[cKey])
		out = append(out, ownerSectionProblems(k[cID], k[cOwner], pair{s, f}, files, owners, specText)...)
	}
	return out
}

// checkGroupOwners is FR-CLB-102 for the structure groups: each field's README
// section is one of the sections of the group's Owner cell.
func checkGroupOwners(r *registry, files map[string]string, owners map[pair]ownerInfo) []string {
	var out []string
	codeOf := map[string]string{}
	for c, f := range files {
		codeOf[f] = c
	}
	for _, g := range r.groups {
		var items [][2]string
		for _, part := range strings.Split(g[gOwner], ";") {
			t := strings.Fields(part)
			for _, n := range t[min(1, len(t)):] {
				items = append(items, [2]string{t[0], n})
			}
		}
		for _, f := range strings.Split(g[gFields], ",") {
			own, ok := owners[pair{g[gScope], strings.TrimSpace(f)}]
			if !ok || own.file == "" {
				continue
			}
			if !slices.ContainsFunc(items, func(it [2]string) bool {
				return it[0] == codeOf[own.file] && secOK(it[1], own.secs)
			}) {
				out = append(out, g[gID]+": "+strings.TrimSpace(f)+" is in no section of "+g[gOwner])
			}
		}
	}
	return out
}

// TestRegistry_OwnerSections verifies that each Owner cell of the registry
// resolves in the owner's spec and is a section of the README's owner column
// for the pair, and that a README row that parses no section fails.
//
// Requirement: FR-CLB-102
func TestRegistry_OwnerSections(t *testing.T) {
	t.Parallel()
	s := loadSpecs(t)
	r := parseRegistry(s.clb)
	owners, noSection := ownerTable(s.readme)
	files := codeToFile(s.readme)
	text := func(name string) string { return readDoc(t, "specs/"+name) }
	if len(noSection) > 0 {
		t.Errorf("README owner rows that parse no section: %v", noSection)
	}
	if got := checkOwnerSections(r, files, owners, text); len(got) > 0 {
		t.Errorf("checkOwnerSections(spec) = %v, want none", got)
	}
	if got := checkGroupOwners(r, files, owners); len(got) > 0 {
		t.Errorf("checkGroupOwners(spec) = %v, want none", got)
	}
	c := r.clone()
	c.knobs[0][cOwner] = "XYZ 1"
	if len(checkOwnerSections(c, files, owners, text)) == 0 {
		t.Errorf("an owner cell with an unknown area code did not fail")
	}
	c = r.clone()
	c.knobs[0][cOwner] = "BEL 99.9"
	if len(checkOwnerSections(c, files, owners, text)) == 0 {
		t.Errorf("an owner section that is not in the spec did not fail")
	}
	c = r.clone()
	c.groups[0][gOwner] = "XXX 1"
	if len(checkGroupOwners(c, files, owners)) == 0 {
		t.Errorf("a group owner cell that matches no README section did not fail")
	}
}
