//go:build !race

package main

// raceEnabled is true when the test binary is built with -race.
const raceEnabled = false
