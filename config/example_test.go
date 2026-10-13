package config_test

import (
	"errors"
	"fmt"
	"testing/fstest"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

const exampleBase = `schema = 1

[commodity.grain]
reference_price = 1000
raw = true

[role.farmer]

[region.home]

[region.home.group.farmer]
households = 10
`

// Parse merges its documents in order, so a later document changes what an
// earlier one set. The result is a plain Config that can be validated.
func ExampleParse() {
	cfg, err := config.Parse(
		[]byte(exampleBase),
		[]byte("schema = 1\n[commodity.grain]\nreference_price = 2000\n"),
	)
	if err != nil {
		fmt.Println("parse:", err)
		return
	}
	fmt.Println("commodities:", len(cfg.Commodities))
	fmt.Println("grain price:", cfg.Commodities[0].ReferencePrice)
	// Output:
	// commodities: 1
	// grain price: 2000
}

// A key the loader does not know is rejected with its path, so a misspelt key
// is never silently ignored.
func ExampleParse_unknownKey() {
	_, err := config.Parse([]byte("schema = 1\n[world]\nscael = 2\n"))
	fmt.Println("unknown key:", errors.Is(err, economysim.ErrUnknownKey))
	// Output:
	// unknown key: true
}

// LoadFS reads named files from any fs.FS, in order, and never opens a path of
// its own: the caller chooses the root.
func ExampleLoadFS() {
	fsys := fstest.MapFS{
		"base.toml":    {Data: []byte(exampleBase)},
		"pricier.toml": {Data: []byte("schema = 1\n[commodity.grain]\nreference_price = 3000\n")},
	}
	cfg, err := config.LoadFS(fsys, "base.toml", "pricier.toml")
	if err != nil {
		fmt.Println("load:", err)
		return
	}
	fmt.Println("grain price:", cfg.Commodities[0].ReferencePrice)
	// Output:
	// grain price: 3000
}
