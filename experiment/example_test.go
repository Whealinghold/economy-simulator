package experiment_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/Whealinghold/economy-sim/experiment"
)

// tinyPack is the smallest configuration the loader takes: one raw commodity that
// households consume, one role, one region and a group of ten households.
var tinyPack = []byte(`schema = 1
[world]
upkeep = [ { commodity = "grain", quantity = 1 } ]
[commodity.grain]
reference_price = 1000
raw = true
[role.farmer]
[region.home]
[region.home.group.farmer]
households = 10
`)

// tinySweep is a one-factor sweep of the belief step over the tiny pack: two
// points, two seeds each, three ticks, one measure.
func tinySweep() experiment.Spec {
	return experiment.Spec{
		Name:     "tiny",
		Base:     [][]byte{tinyPack},
		Factors:  []experiment.Factor{{Name: "step", Path: "world.belief_step", Min: 100, Max: 300, Steps: 2}},
		Replicas: 2,
		Seed0:    1,
		Ticks:    3,
		Measures: []experiment.Measure{{Name: "households", Fn: func(s experiment.Sample) (int64, bool, error) {
			return s.Close.Households, true, nil
		}}},
	}
}

// Plan lists the units of work of an experiment: one for each point and replica.
// A consumer that runs units on its own goroutines calls RunUnit on each.
func ExamplePlan() {
	units, err := experiment.Plan(tinySweep())
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, u := range units {
		fmt.Println("point", u.Point, "replica", u.Replica, "keys", len(u.Keys))
	}
	// Output:
	// point 0 replica 0 keys 1
	// point 0 replica 1 keys 1
	// point 1 replica 0 keys 1
	// point 1 replica 1 keys 1
}

// Run runs every unit on the caller's goroutine and assembles the result: for each
// point and variant the runs, and a reduction of each measure beside the counts of
// the runs used and failed.
func ExampleRun() {
	res, err := experiment.Run(context.Background(), tinySweep())
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, p := range res.Points {
		red := p.Variants[0].Reductions[0]
		fmt.Printf("point %d belief_step %d: %d of %d runs used, mean households %d\n",
			p.Index, p.Values[0], red.Used, red.Planned, red.Mean.Value)
	}
	// Output:
	// point 0 belief_step 100: 2 of 2 runs used, mean households 10
	// point 1 belief_step 300: 2 of 2 runs used, mean households 10
}

// exampleResult assembles two points of eight seed-matched replicas by hand, the
// readings of spec-experiments.md 19.7, as a consumer would from stored records.
func exampleResult() experiment.Result {
	e := experiment.Spec{
		Name:     "example",
		Base:     [][]byte{tinyPack},
		Factors:  []experiment.Factor{{Name: "step", Path: "world.belief_step", Min: 100, Max: 300, Steps: 2}},
		Replicas: 8,
		Seed0:    1,
		Ticks:    10,
		Measures: []experiment.Measure{{Name: "range_bp", Fn: func(experiment.Sample) (int64, bool, error) { return 0, false, nil }}},
	}
	var recs []experiment.Record
	for p, readings := range [][]int64{
		{100, 104, 98, 110, 101, 99, 107, 103},
		{103, 109, 97, 118, 107, 99, 113, 104},
	} {
		for r, v := range readings {
			recs = append(recs, experiment.Record{
				Key: experiment.Key{Point: p, Replica: r}, Seed: uint64(1 + r), Status: experiment.StatusOK,
				FailTick: -1, Final: []experiment.Reading{{Value: v, OK: true}},
			})
		}
	}
	res, err := experiment.Assemble(e, recs)
	if err != nil {
		panic(err) // the records are the example's own
	}
	return res
}

// Compare matches the runs of two groups by replica and gives the mean difference,
// the wins and losses, the exact sign test and the growth, all in integers.
func ExampleCompare() {
	res := exampleResult()
	c, err := experiment.Compare(res, experiment.Group{Point: 0}, experiment.Group{Point: 1}, "range_bp")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("pairs", c.Pairs, "mean difference", c.MeanDiff.Value)
	fmt.Println("wins", c.Wins, "losses", c.Losses, "ties", c.Ties)
	fmt.Println("sign test tail", c.SignTailPPM.Value, "ppm; growth", c.RelativeBP.Value, "bp")
	// Output:
	// pairs 8 mean difference 4
	// wins 6 losses 1 ties 1
	// sign test tail 125000 ppm; growth 341 bp
}

// Render is the canonical text of a result: the form two sweeps are diffed in and
// a golden test compares.
func ExampleRender() {
	for _, line := range strings.Split(experiment.Render(exampleResult()), "\n") {
		if strings.HasPrefix(line, "point") || strings.HasPrefix(line, "reduce") {
			fmt.Println(line)
		}
	}
	// Output:
	// point 0 hash - status ok same_as -1 levels 0 values 100
	// reduce range_bp planned 8 used 8 failed 0 undefined 0 missing 0 min 98 max 110 mean 103 std 4 q1000=98 q2500=99 q5000=101 q7500=104 q9000=110
	// point 1 hash - status ok same_as -1 levels 1 values 300
	// reduce range_bp planned 8 used 8 failed 0 undefined 0 missing 0 min 97 max 118 mean 106 std 7 q1000=97 q2500=99 q5000=104 q7500=109 q9000=118
}
