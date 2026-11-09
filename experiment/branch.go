package experiment

import (
	"context"
	"fmt"
	"slices"

	economysim "github.com/Whealinghold/economy-sim"
)

func (p progress) clone() progress {
	p.ticks = slices.Clone(p.ticks)
	p.series = slices.Clone(p.series)
	p.final = slices.Clone(p.final)
	return p
}

// runBranched runs a unit with a branch (spec-experiments.md 6): the burn-in
// once, a checkpoint at the branch point, and for each variant and continuation
// a restore that runs the rest. Worlds are held one at a time: the burn-in is
// dropped before the first restore and each branch before the next. A burn-in
// that fails is every key's failure (FR-EXP-033); a cancelled context discards
// the whole unit (5.4.2).
func (e Spec) runBranched(ctx context.Context, cfg economysim.Config, u Unit) ([]Record, error) {
	at := e.Branch.At
	burnKey := u.Keys[0] // variant 0, continuation 0: the key of the burn-in (5.5.2)
	seed := e.baseRecord(burnKey).Seed
	_, values, _ := e.point(u.Point)
	w, err := economysim.NewWorld(cfg, e.buildOptions(burnKey, seed)...)
	if err != nil {
		st, tick := classify(stageBuild, err, 0)
		return e.inherit(u, [32]byte{}, &failure{status: st, tick: tick, detail: err.Error()}, progress{}), nil
	}
	hash := w.ConfigHash()
	g := newEngine(ctx, &e, cfg, w, burnKey, e.steps(values, -1), progress{})
	cp, f := g.burnIn(at)
	if f != nil && f.cancelled {
		return nil, fmt.Errorf("experiment: run unit: %w", ctx.Err())
	}
	if f != nil {
		return e.inherit(u, hash, f, g.progress), nil
	}
	burn := g.progress
	out := make([]Record, 0, len(u.Keys))
	for _, k := range u.Keys {
		rec, err := e.branch(ctx, cfg, cp, burnKey, k, values, burn, hash)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, nil
}

// burnIn runs the world to the branch point and takes the checkpoint: the
// boundaries below at in full, then the ticks to at and the close of the window
// that ends there. The steps at at, and the window that opens there, belong to
// the branches (spec-experiments.md 6.1.2).
func (g *engine) burnIn(at int64) (economysim.Checkpoint, *failure) {
	for _, b := range g.bounds {
		if b >= at {
			break
		}
		if f := g.boundary(b); f != nil {
			return economysim.Checkpoint{}, f
		}
	}
	if f := g.advance(at); f != nil {
		return economysim.Checkpoint{}, f
	}
	if f := g.closeAt(at); f != nil {
		return economysim.Checkpoint{}, f
	}
	cp, err := g.w.Checkpoint()
	if err != nil {
		st, tick := classify(stageCheckpoint, err, at)
		return cp, &failure{status: st, tick: tick, detail: err.Error()}
	}
	return cp, nil
}

// runBranch runs a restored world from the branch point to Ticks: the steps at
// at, the window that opens there, then every later boundary in full.
func (g *engine) runBranch(at int64) *failure {
	if f := g.stepsAt(at); f != nil {
		return f
	}
	if f := g.openAt(at); f != nil {
		return f
	}
	for _, b := range g.bounds {
		if b <= at {
			continue
		}
		if f := g.boundary(b); f != nil {
			return f
		}
	}
	return nil
}

// branch restores the checkpoint for the key k and runs the rest. The options are
// Options(k) and the measures' observers, with WithReseed(Salt0 + c) only when
// there are two or more continuations and never WithSeed (FR-EXP-032); after the
// restore each measure with a Fork copies what the burn-in's observer built.
func (e Spec) branch(ctx context.Context, cfg economysim.Config, cp economysim.Checkpoint, burnKey, k Key, values []int64, burn progress, hash [32]byte) (Record, error) {
	rec := e.baseRecord(k)
	rec.Hash = hash
	opts := e.restoreOptions(k, rec)
	w, err := economysim.RestoreWorld(cfg, cp, opts...)
	if err != nil {
		st, tick := classify(stageRestore, err, e.Branch.At)
		return e.fail(rec, &failure{status: st, tick: tick, detail: err.Error()}, burn), nil
	}
	for _, m := range e.Measures {
		if m.Fork != nil {
			m.Fork(burnKey, k)
		}
	}
	rec.Hash = w.ConfigHash()
	g := newEngine(ctx, &e, cfg, w, k, e.steps(values, k.Variant), burn.clone())
	return e.finishRecord(ctx, g, func() *failure { return g.runBranch(e.Branch.At) }, rec)
}

// restoreOptions are the options of a restored world: Options(k), the measures'
// observers, and the reseed of the continuation when there are two or more.
func (e Spec) restoreOptions(k Key, rec Record) []economysim.Option {
	var opts []economysim.Option
	if e.Options != nil {
		opts = e.Options(k)
	}
	opts = append(slices.Clone(opts), e.observers(k)...)
	if rec.HasSalt {
		opts = append(opts, economysim.WithReseed(rec.Salt))
	}
	return opts
}

// inherit is the record of every key of a unit whose burn-in failed: the
// burn-in's status and tick, the windows that closed before it with
// Keep.Series, and no branch run (spec-experiments.md 6, FR-EXP-033).
func (e Spec) inherit(u Unit, hash [32]byte, f *failure, p progress) []Record {
	out := make([]Record, len(u.Keys))
	for i, k := range u.Keys {
		out[i] = e.baseRecord(k)
		out[i].Hash = hash
		out[i] = e.fail(out[i], f, p)
	}
	return out
}
