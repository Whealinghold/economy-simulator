//go:build race

package experiment_test

// raceEnabled is true when the test binary is built with -race. It mirrors the
// constant of the root test package, which an external test package cannot import.
const raceEnabled = true
