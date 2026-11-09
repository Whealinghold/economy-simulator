package experiment

import (
	"context"
	"fmt"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
	"github.com/Whealinghold/economy-sim/internal/design"
)

// A Warning is a rule name and a path, as in spec-config.md 7.2.
type Warning struct {
	// Rule is the name of the rule that applied.
	Rule string
	// Path is the Go path of the factor, or the point index of a result rule.
	Path string
}

// PointCheck is what Check says of one point (spec-experiments.md 14.3).
type PointCheck struct {
	// Point is the index of the point.
	Point int
	// Status is ok, unresolved or invalid_config.
	Status Status
	// Detail is the error text, or empty.
	Detail string
	// Warnings are the rules of 14.4 that applied.
	Warnings []Warning
}

// overlays returns the documents of point p after the base (spec-experiments.md
// 3.3.2): for each factor in order, a range factor's one change, or its level's
// Docs and then one document for each Change.
func (e Spec) overlays(values []int64) ([][]byte, error) {
	var docs [][]byte
	for f, fac := range e.Factors {
		if fac.Path != "" {
			d, err := design.Overlay(fac.Path, values[f])
			if err != nil {
				return nil, err
			}
			docs = append(docs, d)
			continue
		}
		lv := fac.Levels[values[f]]
		docs = append(docs, lv.Docs...)
		for _, ch := range lv.Changes {
			d, err := design.Overlay(ch.Path, ch.Value)
			if err != nil {
				return nil, err
			}
			docs = append(docs, d)
		}
	}
	return docs, nil
}

// resolve turns point p into a validated Config: the loader on the base and the
// overlays, then Prepare, then Config.Validate (spec-experiments.md 5.1.3 step
// 1). A failure is the point's: StatusUnresolved when the loader refused a
// document, StatusInvalidConfig when Prepare or Validate did.
func (e Spec) resolve(ctx context.Context, p int) (economysim.Config, Status, error) {
	_, values, ok := e.point(p)
	if !ok {
		return economysim.Config{}, StatusUnresolved, fmt.Errorf("point %d: %w", p, ErrUnknownRun)
	}
	over, err := e.overlays(values)
	if err != nil {
		return economysim.Config{}, StatusUnresolved, err
	}
	docs := make([][]byte, 0, len(e.Base)+len(over))
	docs = append(append(docs, e.Base...), over...)
	cfg, err := config.Parse(docs...)
	if err != nil {
		return economysim.Config{}, StatusUnresolved, err
	}
	if e.Prepare != nil {
		if cfg, err = e.Prepare(ctx, cfg); err != nil {
			return economysim.Config{}, StatusInvalidConfig, err
		}
	}
	if err := cfg.Validate(); err != nil {
		return economysim.Config{}, StatusInvalidConfig, err
	}
	return cfg, StatusOK, nil
}

// Check resolves every point (the loader, the overlays, Prepare and the
// validation of the resolved Config) and reports, for each, its status, the
// error text and its warnings. It runs no tick, builds no world, calls no
// Options and computes no hash (spec-experiments.md 14.3).
func Check(ctx context.Context, e Spec) ([]PointCheck, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	n, _ := e.pointCount() // Validate has checked it
	out := make([]PointCheck, 0, n)
	for p := range n {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("check: %w", err)
		}
		pc := PointCheck{Point: p, Status: StatusOK, Warnings: e.designWarnings()}
		if _, status, err := e.resolve(ctx, p); err != nil {
			pc.Status, pc.Detail = status, err.Error()
		}
		out = append(out, pc)
	}
	return out, nil
}

// designWarnings returns the warnings of the design alone: a spread over a
// range factor with fewer values than points (spec-experiments.md 4.4.3, 14.4).
func (e Spec) designWarnings() []Warning {
	if e.Design != Spread {
		return nil
	}
	var out []Warning
	for f, fac := range e.Factors {
		// Max - Min + 1 values, in 128 bits' worth of range: compare as unsigned.
		if fac.Path != "" && uint64(fac.Max)-uint64(fac.Min) < uint64(e.DesignN)-1 {
			out = append(out, Warning{Rule: "ranged_narrower_than_design", Path: idx("Factors", f)})
		}
	}
	return out
}
