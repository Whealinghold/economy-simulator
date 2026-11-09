package experiment

import (
	"context"
	"fmt"
	"slices"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/internal/design"
)

// RunOption changes how a sweep runs.
type RunOption func(*runConfig)

type runConfig struct {
	record func(Record)
	prior  []Record
}

// WithRecordFunc sets f to be called synchronously, in plan order, after each
// record, on the caller's goroutine. f must not call the runner
// (spec-experiments.md 5.2.2).
func WithRecordFunc(f func(Record)) RunOption { return func(c *runConfig) { c.record = f } }

// WithPrior gives records already produced by an earlier, cancelled or crashed
// sweep: a unit whose keys are all present is not run again. The records are
// trusted as Assemble trusts them (spec-experiments.md 5.3.4).
func WithPrior(recs []Record) RunOption { return func(c *runConfig) { c.prior = recs } }

// RunUnit runs one unit of Plan and returns one record for each of its keys. It
// is a pure function of the experiment and the unit: it shares nothing with any
// other call, so a consumer's own runner may call it from many goroutines
// (spec-experiments.md 5.1.2, 5.3). A cancelled context discards the run in
// flight and returns the context's error wrapped with %w and no records
// (5.4.2). The errors it returns are those of Validate, ErrUnknownRun for a unit
// outside the plan, and the context's; a failed run is a record.
func RunUnit(ctx context.Context, e Spec, u Unit) ([]Record, error) {
	if err := e.checkUnit(ctx, u); err != nil {
		return nil, err
	}
	return e.resolvedUnit(ctx, u)
}

// resolvedUnit is RunUnit for a unit of a validated experiment's Plan.
func (e Spec) resolvedUnit(ctx context.Context, u Unit) ([]Record, error) {
	cfg, status, err := e.resolve(ctx, u.Point)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("experiment: run unit: %w", ctx.Err())
		}
		return e.failedUnit(u, status, err.Error()), nil
	}
	return e.runUnit(ctx, cfg, u)
}

// ExecuteUnit is RunUnit with a Config the consumer built (spec-experiments.md
// 3.3.5): it skips the base, the factors and Prepare, and everything after the
// resolution is identical. The unit's point only names the records.
func ExecuteUnit(ctx context.Context, cfg economysim.Config, e Spec, u Unit) ([]Record, error) {
	if err := e.checkUnit(ctx, u); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return e.failedUnit(u, StatusInvalidConfig, err.Error()), nil
	}
	return e.runUnit(ctx, cfg, u)
}

// checkUnit validates the experiment and the unit: the unit must be exactly the
// one Plan lists for its point and replica.
func (e Spec) checkUnit(ctx context.Context, u Unit) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("experiment: run unit: %w", err)
	}
	if err := e.Validate(); err != nil {
		return err
	}
	points, _ := e.pointCount() // Validate has checked it
	if u.Point < 0 || u.Point >= points || u.Replica < 0 || u.Replica >= e.Replicas || !slices.Equal(u.Keys, e.unitKeys(u.Point, u.Replica)) {
		return fmt.Errorf("%w: unit of point %d replica %d", ErrUnknownRun, u.Point, u.Replica)
	}
	return nil
}

// unitKeys returns the keys of the unit of a point and replica, by variant then
// continuation.
func (e Spec) unitKeys(p, r int) []Key {
	variants, conts := e.branchShape()
	keys := make([]Key, 0, variants*conts)
	for v := range variants {
		for c := range conts {
			keys = append(keys, Key{Point: p, Replica: r, Variant: v, Cont: c})
		}
	}
	return keys
}

// baseRecord is the record of a key before the run: its seed and salt, no
// failure, every reading undefined and every criterion failed.
func (e Spec) baseRecord(k Key) Record {
	r := Record{
		Key:      k,
		Seed:     design.Seed(e.Seed0, e.Seeding == PerPointSeeds, k.Point, k.Replica, e.Replicas),
		FailTick: -1,
		Status:   StatusOK,
		Final:    make([]Reading, len(e.Measures)),
		Pass:     make([]bool, len(e.Criteria)),
	}
	if b := e.Branch; b != nil && b.Continuations >= 2 {
		r.Salt, r.HasSalt = b.Salt0+uint64(k.Cont), true
	}
	return r
}

// failedUnit is the record of every key of a unit that never built a world.
func (e Spec) failedUnit(u Unit, status Status, detail string) []Record {
	out := make([]Record, len(u.Keys))
	for i, k := range u.Keys {
		out[i] = e.baseRecord(k)
		out[i].Status, out[i].FailTick, out[i].Detail = status, 0, detail
	}
	return out
}

// runUnit runs the unit of a resolved Config.
func (e Spec) runUnit(ctx context.Context, cfg economysim.Config, u Unit) ([]Record, error) {
	if e.Branch != nil {
		return e.runBranched(ctx, cfg, u)
	}
	k := u.Keys[0]
	rec := e.baseRecord(k)
	w, err := economysim.NewWorld(cfg, e.buildOptions(k, rec.Seed)...)
	if err != nil {
		st, tick := classify(stageBuild, err, 0)
		return []Record{e.fail(rec, &failure{status: st, tick: tick, detail: err.Error()}, progress{})}, nil
	}
	rec.Hash = w.ConfigHash()
	_, values, _ := e.point(k.Point)
	g := newEngine(ctx, &e, cfg, w, k, e.steps(values, -1), progress{})
	return e.conclude(ctx, g, rec)
}

// conclude finishes a run that has run its boundaries and returns its record
// as the only record of the unit.
func (e Spec) conclude(ctx context.Context, g *engine, rec Record) ([]Record, error) {
	r, err := e.finishRecord(ctx, g, g.runAll, rec)
	if err != nil {
		return nil, err
	}
	return []Record{r}, nil
}

// finishRecord runs the engine with run, then takes the digest and builds the
// record; a cancellation is the returned error and no record.
func (e Spec) finishRecord(ctx context.Context, g *engine, run func() *failure, rec Record) (Record, error) {
	f := run()
	var digest [32]byte
	var snap economysim.Snapshot
	if f == nil {
		digest, snap, f = g.finish()
	}
	if f != nil && f.cancelled {
		return Record{}, fmt.Errorf("experiment: run unit: %w", ctx.Err())
	}
	if f != nil {
		return e.fail(rec, f, g.progress), nil
	}
	rec.Digest, rec.Detail, rec.Final = digest, g.detail, g.final
	rec.Ticks, rec.Series = g.ticks, g.series
	if e.Keep.Snapshot {
		rec.Snapshot, rec.HasSnapshot = snap, true
	}
	for i, c := range e.Criteria {
		rec.Pass[i] = e.passes(c, rec.Final)
	}
	return rec, nil
}

// fail is the record of a run that stopped: its readings undefined, every
// criterion failed, no digest, and with Keep.Series the windows that closed
// before the failure (spec-experiments.md 9.2).
func (e Spec) fail(rec Record, f *failure, p progress) Record {
	rec.Status, rec.FailTick, rec.Detail = f.status, f.tick, f.detail
	if e.Keep.Series {
		rec.Ticks, rec.Series = slices.Clone(p.ticks), slices.Clone(p.series)
	}
	return rec
}

// passes reports whether the final readings pass the criterion: the measure's
// reading is defined and inside the band.
func (e Spec) passes(c Criterion, final []Reading) bool {
	i := slices.IndexFunc(e.Measures, func(m Measure) bool { return m.Name == c.Measure })
	if i < 0 || i >= len(final) || !final[i].OK {
		return false
	}
	v := final[i].Value
	switch c.Op {
	case AtMost:
		return v <= c.A
	case AtLeast:
		return v >= c.A
	case Between:
		return c.A <= v && v <= c.B
	}
	return false
}

// observers returns the observer options of the measures that have one, in
// measure order; a measure without an Observer registers none, so a
// snapshot-only sweep builds a world with no observer (FR-EXP-017).
func (e Spec) observers(k Key) []economysim.Option {
	var out []economysim.Option
	for _, m := range e.Measures {
		if m.Observer != nil {
			out = append(out, economysim.WithObserver(m.Observer(k)))
		}
	}
	return out
}

// buildOptions are the options of a world built from tick 0: the consumer's
// Options(key), then the measures' observers, then WithSeed last, so the seed is
// the harness's whatever the options say (spec-experiments.md 5.1.3 step 2).
func (e Spec) buildOptions(k Key, seed uint64) []economysim.Option {
	return append(e.worldOptions(k), economysim.WithSeed(seed))
}

// worldOptions are the consumer's Options(k) and then the measures' observers, in a
// slice of their own.
func (e Spec) worldOptions(k Key) []economysim.Option {
	var opts []economysim.Option
	if e.Options != nil {
		opts = e.Options(k)
	}
	return append(slices.Clone(opts), e.observers(k)...)
}

// runAll runs the sweep sequentially on the caller's goroutine and returns the
// records of the complete units, prior records included, in plan order. On a
// cancelled context it returns the records so far and the wrapped context
// error (spec-experiments.md 5.2.1, 5.2.3).
func (e Spec) runAll(ctx context.Context, opts ...RunOption) ([]Record, error) {
	var rc runConfig
	for _, o := range opts {
		if o != nil {
			o(&rc)
		}
	}
	units, err := Plan(e)
	if err != nil {
		return nil, err
	}
	// Prior records are trusted as Assemble trusts them: a key outside the plan or
	// twice with different contents is refused (spec-experiments.md 5.3.4).
	prior, err := e.indexRecords(rc.prior)
	if err != nil {
		return nil, err
	}
	var out []Record
	for _, u := range units {
		if recs, ok := priorOf(prior, u); len(prior) > 0 && ok {
			out = append(out, recs...)
			continue
		}
		if err := ctx.Err(); err != nil {
			return append(out, e.priorAfter(prior, units, u)...), fmt.Errorf("experiment: run: %w", err)
		}
		recs, err := e.resolvedUnit(ctx, u) // Plan has validated the experiment and listed the unit
		if err != nil {
			return append(out, e.priorAfter(prior, units, u)...), err
		}
		for _, r := range recs {
			if rc.record != nil {
				rc.record(r)
			}
		}
		out = append(out, recs...)
	}
	return out, nil
}

// priorOf returns the prior records of the keys of u, in key order, and whether
// all of them are present.
func priorOf(prior map[Key]Record, u Unit) ([]Record, bool) {
	out := make([]Record, 0, len(u.Keys))
	for _, k := range u.Keys {
		r, ok := prior[k]
		if !ok {
			return nil, false
		}
		out = append(out, r)
	}
	return out, true
}

// priorAfter returns the prior records of the complete units after u, so that a
// cancelled sweep does not drop work the caller already had.
func (e Spec) priorAfter(prior map[Key]Record, units []Unit, u Unit) []Record {
	var out []Record
	for _, v := range units[u.Point*e.Replicas+u.Replica+1:] {
		if recs, ok := priorOf(prior, v); ok {
			out = append(out, recs...)
		}
	}
	return out
}
