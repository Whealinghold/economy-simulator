package calibration

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var (
	reSTYFactClass = regexp.MustCompile(`\*Class:\* ([XOBRP])`)
	reSTYDetector  = regexp.MustCompile(`(?m)^\| \*\*(R\d+[a-z]?)\*\* \|`)
	reSnakeCap     = regexp.MustCompile(`[A-Z][a-z0-9]*`)
)

// registryKnown fills Known from the parsed registry and from STY, the data the
// record checks need (plan 2-21 D626): the package reads no file, this test helper
// does. An `off=min` or `off=max` is not resolved (the bound is the owner's), so
// that row has no off value here; a numeric `off=` is.
func registryKnown(s *specs, r *registry) Known {
	k := Known{Knobs: map[string]KnobInfo{}, Facts: map[string]byte{}, Detectors: map[string]bool{}, Paths: map[string]string{}}
	for _, row := range r.knobs {
		info := knobInfo(row)
		k.Knobs[row[cID]] = info
		if p, ok := pathPattern(info.Key); ok {
			k.Paths[p] = info.Key
		}
	}
	for _, blk := range strings.Split(s.sty, "**STY-")[1:] {
		if m := reSTYFactClass.FindStringSubmatch(blk); m != nil {
			k.Facts[blk[:3]] = m[1][0]
		}
	}
	for _, m := range reSTYDetector.FindAllStringSubmatch(s.sty, -1) {
		k.Detectors[m[1]] = true
	}
	return k
}

// knobInfo reads one knob row of the registry.
func knobInfo(row []string) KnobInfo {
	info := KnobInfo{Class: row[cClass][0], Key: row[cKey], Stage: row[cStage], Owner: strings.Fields(row[cOwner])[0], Moves: map[string]MoveSign{}}
	for _, f := range words(row[cFlags]) {
		info.V1x = info.V1x || f == "v1x"
		info.Z = info.Z || f == "z"
		if v, ok := strings.CutPrefix(f, "off="); ok {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				info.Off = Off{Has: true, Value: n}
			}
		}
	}
	for _, tok := range strings.Split(row[cMoves], ";") {
		if m := reMove.FindStringSubmatch(strings.TrimSpace(tok)); m != nil {
			info.Moves[m[1]+"/"+m[2]] = MoveSign{Sign: m[3][0], Source: sourceOf(m[3])}
		}
	}
	return info
}

// sourceOf is the source letter of a move sign: S, M, E or H, 0 for `?`.
func sourceOf(sign string) byte {
	if len(sign) == 2 {
		return sign[1]
	}
	return 0
}

// snake respells a Go field name as the key of a configuration file, the inverse of
// pascal: BP and ID keep to one word each, and InProgressBP is in_progress.
func snake(field string) string {
	if field == "InProgressBP" {
		return "in_progress"
	}
	f := strings.NewReplacer("BP", "Bp", "ID", "Id").Replace(field)
	parts := reSnakeCap.FindAllString(f, -1)
	for i := range parts {
		parts[i] = strings.ToLower(parts[i])
	}
	return strings.Join(parts, "_")
}

// pathPattern is the configuration path of a field of a table keyed by an ID, with
// `*` for the instance (EXP 3.2.1), or false for a scope that has no such table.
func pathPattern(key string) (string, bool) {
	scope, field := keyParts(key)
	switch scope {
	case "Role":
		return "role.*." + snake(field), true
	case "Region":
		return "region.*." + snake(field), true
	case "Group":
		return "region.*.group.*." + snake(field), true
	}
	return "", false
}

// TestRegistry_Known verifies the Known built from the real registry: a row with
// its class, stage, owner and moves, the facts and regimes of STY with their
// classes, and the path patterns of the tables keyed by an ID.
//
// Requirement: FR-CLB-007, FR-CLB-009
func TestRegistry_Known(t *testing.T) {
	t.Parallel()
	s := loadSpecs(t)
	k := registryKnown(s, parseRegistry(s.clb))
	bel := k.Knobs["K-BEL12"]
	checks := []struct {
		name      string
		got, want any
	}{
		{"K-BEL12 class", bel.Class, byte('D')}, {"K-BEL12 key", bel.Key, "World.CostFloor"},
		{"K-BEL12 stage", bel.Stage, "C4"}, {"K-BEL12 owner", bel.Owner, "BEL"},
		{"K-BEL12 move P02/Pinned.Share", bel.Moves["P02/Pinned.Share"], MoveSign{'+', 'S'}},
		{"fact P01 class", k.Facts["P01"], byte('X')}, {"fact P02 class", k.Facts["P02"], byte('B')},
		{"detector R5a", k.Detectors["R5a"], true},
		{"path role.*.target_ratio", k.Paths["role.*.target_ratio"], "Role.TargetRatio"},
		{"at least 100 rows", len(k.Knobs) >= 100, true},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("Known %s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

// TestRegistry_FlagsOffAndPaths verifies the flags, the numeric off value, the
// snake form of a field and the paths of the tables keyed by an ID.
//
// Requirement: FR-CLB-009, FR-CLB-103
func TestRegistry_FlagsOffAndPaths(t *testing.T) {
	t.Parallel()
	row := func(flags string) KnobInfo {
		return knobInfo([]string{"K-X", "World.A", "D", "BEL 1", "bp", "U", "geo", "C4", flags, "-", "-", "-"})
	}
	z, off := row("z v1x off=0"), row("off=min")
	checks := []struct {
		name      string
		got, want any
	}{
		{"z", z.Z, true}, {"v1x", z.V1x, true}, {"numeric off", z.Off, Off{Has: true, Value: 0}},
		{"off=min is not resolved", off.Off, Off{}},
		{"snake ExpectedWorkTicks", snake("ExpectedWorkTicks"), "expected_work_ticks"}, {"snake RateBP", snake("RateBP"), "rate_bp"},
		{"snake InProgressBP", snake("InProgressBP"), "in_progress"}, {"snake ID", snake("ID"), "id"}, {"snake CostFloor", snake("CostFloor"), "cost_floor"},
	}
	for _, key := range [][2]string{{"Role.A", "role.*.a"}, {"Region.A", "region.*.a"}, {"Group.A", "region.*.group.*.a"}, {"Edge.Distance", ""}} {
		got, _ := pathPattern(key[0])
		checks = append(checks, struct {
			name      string
			got, want any
		}{"pathPattern " + key[0], got, key[1]})
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}
