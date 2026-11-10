//go:build race

package main

// raceEnabled is true when the test binary is built with -race; the runs then use
// one seed and fewer ticks (colonySeeds, colonyTicks).
const raceEnabled = true
