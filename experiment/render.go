package experiment

import (
	"encoding/hex"
	"strconv"
	"strings"
)

// Render returns the canonical text of the readings and verdicts of a Result
// (spec-experiments.md 8.2.2): UTF-8, LF, integers in decimal, a hash and a digest
// as the first 8 bytes in lower-case hex, the order of the tree, and a dash where a
// value is undefined. It is a pure function of the Result with no float, map order
// or time, and the same on every platform; two sweeps are diffed in this form and
// a golden test compares it. It omits the Snapshot of a record and keeps the error
// text of a run to one detail line, with line breaks replaced by a space. An empty
// list of levels or values prints as a dash.
func Render(r Result) string {
	var b strings.Builder
	line := func(parts ...string) { b.WriteString(strings.Join(parts, " ")); b.WriteByte('\n') }
	itoa := strconv.Itoa
	line("experiment", oneLine(r.Name), "format", itoa(r.Format), "points", itoa(len(r.Points)),
		"measures", itoa(len(r.Measures)), "criteria", itoa(len(r.Criteria)))
	for k, m := range r.Measures {
		line("measure", itoa(k), oneLine(m))
	}
	for _, p := range r.Points {
		renderPoint(line, r, p)
	}
	return b.String()
}

// renderPoint writes a point, its warnings and its variants.
func renderPoint(line func(...string), r Result, p PointResult) {
	line("point", strconv.Itoa(p.Index), "hash", hexOf(p.Hash), "status", string(p.Status),
		"same_as", strconv.Itoa(p.SameAs), "levels", list(p.Levels, strconv.Itoa),
		"values", list(p.Values, func(v int64) string { return strconv.FormatInt(v, 10) }))
	for _, w := range p.Warnings {
		line("warning", w.Rule, w.Path)
	}
	for _, v := range p.Variants {
		line("variant", strconv.Itoa(v.Variant))
		for _, run := range v.Runs {
			renderRun(line, r, run)
		}
		for i, red := range v.Reductions {
			if i < len(r.Measures) { // a hand-built Result may carry more rows than names
				renderReduction(line, r.Measures[i], red)
			}
		}
		for i, ps := range v.Pass {
			if i >= len(r.Criteria) {
				break
			}
			line("pass", oneLine(r.Criteria[i]), "passes", strconv.Itoa(ps.Passes), "planned", strconv.Itoa(ps.Planned),
				"share", strconv.Itoa(ps.ShareBP))
		}
	}
}

// renderRun writes one record: its header, the detail, the final readings, the
// verdicts and the windows.
func renderRun(line func(...string), r Result, run Record) {
	salt := "-"
	if run.HasSalt {
		salt = strconv.FormatUint(run.Salt, 10)
	}
	line("run", "replica", strconv.Itoa(run.Key.Replica), "cont", strconv.Itoa(run.Key.Cont),
		"seed", strconv.FormatUint(run.Seed, 10), "salt", salt, "status", string(run.Status),
		"fail_tick", strconv.FormatInt(run.FailTick, 10), "digest", hexOf(run.Digest))
	if run.Detail != "" {
		line("detail", oneLine(run.Detail))
	}
	if len(r.Measures) > 0 {
		line(append([]string{"final"}, named(r.Measures, run.Final)...)...)
	}
	if len(r.Criteria) > 0 {
		verdicts := make([]string, len(r.Criteria))
		for i, c := range r.Criteria {
			c = oneLine(c)
			verdicts[i] = c + "=fail"
			if i < len(run.Pass) && run.Pass[i] {
				verdicts[i] = c + "=pass"
			}
		}
		line(append([]string{"verdict"}, verdicts...)...)
	}
	for w, row := range run.Series {
		tick := "-"
		if w < len(run.Ticks) {
			tick = strconv.FormatInt(run.Ticks[w], 10)
		}
		line(append([]string{"window", tick}, named(r.Measures, row)...)...)
	}
}

// renderReduction writes the reduce line of one measure.
func renderReduction(line func(...string), name string, red Reduction) {
	parts := []string{"reduce", oneLine(name), "planned", strconv.Itoa(red.Planned), "used", strconv.Itoa(red.Used),
		"failed", strconv.Itoa(red.Failed), "undefined", strconv.Itoa(red.Undefined),
		"missing", strconv.Itoa(red.Missing), "min", dash(red.Min), "max", dash(red.Max),
		"mean", dash(red.Mean), "std", dash(red.Std)}
	for i, q := range red.Q {
		parts = append(parts, "q"+strconv.Itoa(Quantile(i))+"="+dash(q))
	}
	line(parts...)
}

// named returns "name=value" for each measure; a missing reading is a dash.
func named(names []string, row []Reading) []string {
	out := make([]string, len(names))
	for i, n := range names {
		var rd Reading
		if i < len(row) {
			rd = row[i]
		}
		out[i] = oneLine(n) + "=" + dash(rd)
	}
	return out
}

// dash is a reading in decimal, or a dash when it is undefined.
func dash(r Reading) string {
	if !r.OK {
		return "-"
	}
	return strconv.FormatInt(r.Value, 10)
}

// hexOf is the first 8 bytes of a hash in lower-case hex, or a dash for zero.
func hexOf(h [32]byte) string {
	if h == ([32]byte{}) {
		return "-"
	}
	return hex.EncodeToString(h[:8])
}

// list joins the elements with commas, or is a dash for none.
func list[T any](xs []T, f func(T) string) string {
	if len(xs) == 0 {
		return "-"
	}
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = f(x)
	}
	return strings.Join(parts, ",")
}

// oneLine replaces line breaks with a space, so that a name or an error text
// cannot start a line of the rendering.
func oneLine(s string) string {
	return strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(s)
}
