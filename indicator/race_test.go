//go:build race

package indicator_test

// raceEnabled is true when the test binary is built with -race. It mirrors the
// constant of the root test package, which an external test package cannot
// import (plan 2-18 D576); plan 2-19 reuses it.
const raceEnabled = true
