// Command inspector runs the economy simulator from a pack of configuration
// layers and prints what happened: the roles, the markets, the money by role
// and the events of each window. It is the headless half of an inspector: the
// same engine a terminal UI would sit on, with output a script or an agent can
// read.
//
// Example:
//
//	inspector -chains testdata/chains -pack one -seed 1 -ticks 600 -every 100
package main

import "os"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
