package experiment_test

import (
	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/experiment"
)

// valid returns a small experiment that passes Validate: the tests break one
// rule of it at a time.
func valid() experiment.Spec {
	return experiment.Spec{
		Name:     "t",
		Base:     [][]byte{[]byte("base")},
		Replicas: 2,
		Seed0:    1,
		Ticks:    10,
		Measures: []experiment.Measure{{Name: "m", Fn: func(experiment.Sample) (int64, bool, error) { return 1, true, nil }}},
	}
}

func step(tick int64) experiment.Step {
	return experiment.Step{Tick: tick, Do: func(*economysim.World) error { return nil }}
}

func level(paths ...string) experiment.Level {
	var l experiment.Level
	for _, p := range paths {
		l.Changes = append(l.Changes, experiment.Change{Path: p, Value: 1})
	}
	return l
}

// twoLevels is an explicit factor over one path.
func twoLevels(path string) experiment.Factor {
	return experiment.Factor{Levels: []experiment.Level{level(path), level(path)}}
}
