package economysim

import (
	"context"
	"log/slog"

	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
	tec "github.com/Whealinghold/economy-sim/internal/technology"
)

// This file is the reporting of the technology module (spec-technology.md 11; the
// continuation file of technology.go, architecture.md 2.6): the events, the
// snapshot rows, the names table and the Debug lines. Nothing here changes state.

// techNames is the names table of the fields: the string ID of each TechFieldID, in
// ascending order, nil when the module is off (spec-technology.md 11.1.3).
func techNames(c *Config) []string {
	m := c.Modules.Technology
	if m == nil {
		return nil
	}
	out := make([]string, len(m.Fields))
	for i, f := range m.Fields {
		out[i] = f.ID
	}
	return out
}

// reportTechnology delivers TechnologyAdvanced for every cell in which anything
// accrued, in ascending (region, field) order, and writes the Debug line of each
// (spec-technology.md 11.1.1, 11.5). It reads the counters before the step clears
// them. With no observer and no Debug logger it does nothing.
func (w *World) reportTechnology() {
	ctx := context.Background()
	debug := w.log.Enabled(ctx, slog.LevelDebug)
	if !w.obs.supplied && !debug {
		return
	}
	t := w.tec
	for i := range t.results {
		r := &t.results[i]
		if !r.Accrued() && t.counter[i] == 0 {
			continue
		}
		region, field := t.regions[i/t.nf], TechFieldID(i%t.nf)
		if w.obs.supplied {
			w.obs.deliver(TechnologyAdvanced{
				Tick: w.Tick(), Region: region, Field: field, Executions: t.counter[i],
				Experience: r.Experience, Discoveries: r.Discoveries, BeforeBP: r.Before, AfterBP: r.After,
				DoingAddedBP: r.Doing, TrendAddedBP: r.Trend, DiscoveryAddedBP: r.Discovery,
				CatchUpAddedBP: r.CatchUp, ClampedBP: r.Clamped, AtCap: r.After >= t.fields[field].CapBP,
			})
		}
		if debug {
			w.log.LogAttrs(ctx, slog.LevelDebug, "technology advanced", slog.Int64(logkeys.Tick, w.Tick()),
				slog.Int64(logkeys.Region, int64(region)), slog.String(logkeys.Field, w.cfg.Modules.Technology.Fields[field].ID),
				slog.Int64(logkeys.Gain, r.After))
		}
	}
}

// reportAdjusted delivers the TechnologyAdjusted of the cell a call changed and
// writes its Debug line. before is the gain before the call and lost the part of
// the delta the bounds cut.
func (w *World) reportAdjusted(cell int, delta, before, lost int64) {
	t := w.tec
	region, field := t.regions[cell/t.nf], TechFieldID(cell%t.nf)
	if w.obs.supplied {
		w.obs.deliver(TechnologyAdjusted{Tick: w.Tick(), Region: region, Field: field, DeltaBP: delta,
			BeforeBP: before, AfterBP: t.gain[cell], LostBP: lost})
	}
	if ctx := context.Background(); w.log.Enabled(ctx, slog.LevelDebug) {
		w.log.LogAttrs(ctx, slog.LevelDebug, "technology adjusted", slog.Int64(logkeys.Tick, w.Tick()),
			slog.Int64(logkeys.Region, int64(region)), slog.String(logkeys.Field, w.cfg.Modules.Technology.Fields[field].ID),
			slog.Int64(logkeys.Gain, t.gain[cell]))
	}
}

// snapshotTechnology fills the technology rows: one for each cell with experience
// or advance above 0, in ascending (region, field) order. It is a function of the
// saved state (spec-technology.md 11.4) and returns at once with the module off.
func (w *World) snapshotTechnology(s *Snapshot) {
	t := w.tec
	if t == nil {
		return
	}
	for i, c := range t.cells {
		if c.Experience > 0 || c.Advance > 0 {
			s.Tech = append(s.Tech, TechRow{Region: t.regions[i/t.nf], Field: TechFieldID(i % t.nf),
				Experience: c.Experience, AdvanceBP: c.Advance, GainBP: t.gain[i]})
		}
	}
}

// saveTechnology writes tag 33 (spec-checkpoint.md 4.2, spec-technology.md 12.1): for
// each cell in ascending (RegionID, TechFieldID) order its Experience, Advance and
// CatchUp as unsigned varints (the count is R * F from the configuration and is not
// written), then for each field in ascending order its TrendCarry. The key is derived
// from the effective seed, the gains from the cells and the counters are empty at a
// boundary, so none of them is written.
func (w *World) saveTechnology(cw *codec.Writer) error {
	t := w.tec
	for _, c := range t.cells {
		cw.U(uint64(c.Experience))
		cw.U(uint64(c.Advance))
		cw.U(uint64(c.CatchUp))
	}
	for _, c := range t.trendCarry {
		cw.U(uint64(c))
	}
	return nil
}

// loadTechnology reads tag 33 into the state built for the configuration, in the order
// of spec-checkpoint.md 6.2: Experience in [0, MaxExperience], Advance in [0, CapBP] of
// its field, CatchUp in [0, 10 000 * TicksPerYear), TrendCarry in [0, TicksPerYear);
// each violation is ErrCheckpointCorrupt and the caller refuses a byte after the last
// field. It then derives the gain of every cell, which is never saved (FR-TEC-028); a gain that
// does not fit is a corrupt section, not an invariant, and leaves the gains untouched.
func (w *World) loadTechnology(_ *restoreCtx, r *codec.Reader) error {
	t := w.tec
	for i := range t.cells {
		f := &t.fields[i%t.nf]
		t.cells[i] = tec.Cell{
			Experience: r.Bounded("technology experience", MaxExperience),
			Advance:    r.Bounded("technology advance", f.CapBP),
			CatchUp:    r.Bounded("technology catch-up carry", 10_000*t.tpy-1),
		}
	}
	for i := range t.trendCarry {
		t.trendCarry[i] = r.Bounded("technology trend carry", t.tpy-1)
	}
	if r.Err() != nil {
		return nil
	}
	gains := make([]int64, len(t.cells))
	for i, c := range t.cells {
		g, ok := tec.Gain(t.fields[i%t.nf], c.Experience, c.Advance)
		if !ok {
			r.Failf("technology cell %d: a gain does not fit", i)
			return nil
		}
		gains[i] = g
	}
	copy(t.gain, gains)
	return nil
}
