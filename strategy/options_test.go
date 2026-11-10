package strategy_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
	"github.com/Whealinghold/economy-sim/strategy"
)

// merchantDoc adds the merchants module and a merchant role "broker" to the pack
// two with its trade, transport and information overlays: one household in each
// region, a hired mode and a take. It is the Go-side of a file that Phase 8 of
// plan 2-15 will ship as overlay/merchants.toml.
const merchantDoc = `
[modules.merchants]
max_open = 1000

[role.broker]
strategy = "merchant"
informed = true
trades = [ "fuel", "haul" ]
merchant_fee_share = 5000

[role.broker.merchant]
modes = [ "coach" ]
lot_units = 1
consigns = true
take_units = 150

[region.north.group.broker]
households = 1

[region.south.group.broker]
households = 1
`

// merchantPack parses the layers of the pack two and merchantDoc and validates the
// result.
func merchantPack(t testing.TB) economysim.Config {
	t.Helper()
	var docs [][]byte
	for _, l := range []string{"base.toml", "two.toml", "overlay/trade-two.toml", "overlay/transport-two.toml", "overlay/information-two.toml"} {
		b, err := os.ReadFile(filepath.Join("..", "testdata", "chains", l))
		if err != nil {
			t.Fatalf("read layer %s: %v", l, err)
		}
		docs = append(docs, b)
	}
	cfg, err := config.Parse(append(docs, []byte(merchantDoc))...)
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
	return cfg
}

// TestOptions_RegistersMerchant verifies that Options registers the merchant under
// the name "merchant", so a configuration whose role names it builds a world, and
// that the world steps with merchants in it.
//
// Requirement: FR-MER-001
func TestOptions_RegistersMerchant(t *testing.T) {
	t.Parallel()
	opts := strategy.Options()
	if len(opts) != 1 || strategy.MerchantName != "merchant" {
		t.Fatalf("Options() = %d options, MerchantName = %q, want 1 and %q", len(opts), strategy.MerchantName, "merchant")
	}
	w, err := economysim.NewWorld(merchantPack(t), append(opts, economysim.WithAudit())...)
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	for range 5 {
		if err := w.Step(); err != nil {
			t.Fatalf("Step() at tick %d error = %v, want nil", w.Tick(), err)
		}
	}
}

// TestNewWorld_UnregisteredMerchantFails verifies that a role that names the
// merchant strategy needs the registration: without Options the world is refused
// with ErrUnknownStrategy (spec-merchants.md 3.1.1).
//
// Requirement: FR-MER-001
func TestNewWorld_UnregisteredMerchantFails(t *testing.T) {
	t.Parallel()
	_, err := economysim.NewWorld(merchantPack(t))
	if !errors.Is(err, economysim.ErrUnknownStrategy) {
		t.Errorf("NewWorld() without Options error = %v, want ErrUnknownStrategy", err)
	}
}
