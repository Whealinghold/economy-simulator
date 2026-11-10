//go:build !race

package experiment_test

// raceEnabled is true when the test binary is built with -race (see race_test.go).
const raceEnabled = false
