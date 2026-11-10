//go:build race

package main

// raceEnabled is true when the test binary is built with -race. A 5 000-household
// world is about ten times slower under the detector, so the tests that step it use
// fewer ticks (raceTicks).
const raceEnabled = true
