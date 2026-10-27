package economysim_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// The checkpoint tests of the money supply module: its section, tag 13
// (spec-money-supply.md 14, spec-checkpoint.md 4.2), and the fork-at-N check of
// plan 2-2 D258 with the module on.

// moneyCheckpointFlows are the flows of the checkpoint tests: a rate grant per
// year, whose carries are started at the first visit and kept, and a fixed levy
// with a limit, which ends in the run.
func moneyCheckpointFlows() []economysim.MoneyFlow {
	aid := flowTo("aid", economysim.FlowGrant, "farmer")
	aid.Amount, aid.Per = 3_650, economysim.PerYear
	tax := flowTo("tax", economysim.FlowLevy, "baker")
	tax.Amount, tax.Limit = 3, 100
	return []economysim.MoneyFlow{aid, tax}
}

// moneyCheckpointConfig is moneyWorldConfig with those flows, bakers holding 50.
func moneyCheckpointConfig() economysim.Config {
	c := moneyWorldConfig(10, 10, moneyCheckpointFlows()...)
	c.Roles[0].StartMoney, c.Roles[1].StartMoney = 50, 50
	return c
}

// moneyCheckpointRaw steps the world of moneyCheckpointConfig n ticks and returns
// its checkpoint bytes.
func moneyCheckpointRaw(t *testing.T, n int) []byte {
	t.Helper()
	w := moneyWorld(t, moneyCheckpointConfig())
	step(t, w, n)
	return mustMarshal(t, mustCheckpoint(t, w))
}

// bodyValues reads a section body that is a sequence of unsigned varints, as tag
// 13 is, into its values.
func bodyValues(t *testing.T, body []byte) []uint64 {
	t.Helper()
	var out []uint64
	for off := 0; off < len(body); {
		var v uint64
		v, off = uv(t, body, off)
		out = append(out, v)
	}
	return out
}

// bodyBytes is the inverse of bodyValues.
func bodyBytes(vals []uint64) []byte {
	var b []byte
	for _, v := range vals {
		b = binary.AppendUvarint(b, v)
	}
	return b
}

// TestCheckpointTable_Tag13 verifies that tag 13 is in a checkpoint of a world
// with the module on, between the demand tag's place and the extensions, and in
// none of a world with it off.
//
// Requirement: FR-MON-040, FR-CHK-013
func TestCheckpointTable_Tag13(t *testing.T) {
	t.Parallel()
	tags := func(c economysim.Config) []uint64 {
		var out []uint64
		for _, s := range decodeFrame(t, mustMarshal(t, mustCheckpoint(t, moneyWorld(t, c)))).sections {
			out = append(out, s.tag)
		}
		return out
	}
	off := moneyCheckpointConfig()
	off.Modules.MoneySupply = nil
	if got, want := tags(moneyCheckpointConfig()), []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 13, 34}; !slices.Equal(got, want) {
		t.Errorf("tags with the module on = %v, want %v", got, want)
	}
	if got := tags(off); slices.Contains(got, 13) {
		t.Errorf("tags with the module off = %v, want no 13", got)
	}
}

// TestCheckpoint_MoneySupplyBytesCanonical verifies the body of tag 13: the
// totals, scales and ends of the flows in ID order, then the started carries of
// each flow as (account, carry) pairs in ascending account ID; no pair before the
// first step (a carry is started at the first visit), and equal bytes for equal
// states, also after a restore.
//
// Requirement: FR-MON-040
func TestCheckpoint_MoneySupplyBytesCanonical(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	if body := bodyValues(t, tagBody(t, moneyCheckpointRaw(t, 0), 13)); !slices.Equal(body, []uint64{0, 10_000, 0, 0, 10_000, 0, 0, 0}) {
		t.Errorf("tag 13 before the first step = %v, want two flows at total 0, scale 10 000, not ended, and two empty carry lists", body)
	}
	raw := moneyCheckpointRaw(t, 5)
	vals := bodyValues(t, tagBody(t, raw, 13))
	if len(vals) < 8 || vals[1] != 10_000 || vals[4] != 10_000 {
		t.Fatalf("tag 13 after 5 ticks = %v, want the two flows' totals, scales and ends first", vals)
	}
	pairs := vals[6:]
	n := int(pairs[0])
	if n != 10 || len(pairs) < 1+2*n {
		t.Fatalf("aid carries = %d in %v, want the 10 farmer accounts", n, pairs)
	}
	for i := 1; i < n; i++ {
		if pairs[1+2*i] <= pairs[1+2*(i-1)] {
			t.Errorf("accounts of the pairs are not ascending: %v", pairs[1:1+2*n])
		}
	}
	w, err := economysim.RestoreWorld(moneyCheckpointConfig(), reframe(t, raw, func(s []section) []section { return s }))
	if err != nil {
		t.Fatalf("RestoreWorld error = %v, want nil", err)
	}
	if again := mustMarshal(t, mustCheckpoint(t, w)); !bytes.Equal(again, raw) {
		t.Error("the restored world's checkpoint differs from the one it was restored from")
	}
}

// TestCheckpoint_MoneySupplyCorrupt verifies that tag 13 is refused when a total
// is above the flow's limit, a scale above MaxFlowScale, an end out of range or
// not matching the total, a count above the bytes, a pair naming no agent's
// account or out of order, or a carry at or above its denominator.
//
// Requirement: FR-MON-040, FR-CHK-013
func TestCheckpoint_MoneySupplyCorrupt(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	t.Parallel()
	// Two ticks: the levy has collected 60 of its limit of 100 and has not ended.
	raw := moneyCheckpointRaw(t, 2)
	base := bodyValues(t, tagBody(t, raw, 13))
	// aid: total, scale, ended at 0 to 2; tax: 3 to 5; then aid's count and pairs.
	// A rate grant per year has a denominator of 365 * 10^4.
	const den = 365 * 10_000
	tests := []struct {
		name string
		edit func(v []uint64)
	}{
		{"total above the limit", func(v []uint64) { v[3] = 101 }},
		{"scale above the bound", func(v []uint64) { v[1] = economysim.MaxFlowScale + 1 }},
		{"end out of range", func(v []uint64) { v[2] = 3 }},
		{"ended below its limit", func(v []uint64) { v[5] = 1 }},
		{"ended at the cap with a limit set", func(v []uint64) { v[3], v[5] = 100, 2 }},
		{"not ended at its limit", func(v []uint64) { v[3], v[5] = 100, 0 }},
		{"count above the bytes", func(v []uint64) { v[6] = 1 << 40 }},
		{"account of a boundary", func(v []uint64) { v[7] = 1 }},
		{"account beyond the ledger", func(v []uint64) { v[7] = 1 << 30 }},
		{"accounts descending", func(v []uint64) { v[7], v[9] = v[9], v[7] }},
		{"carry at its denominator", func(v []uint64) { v[8] = den }},
	}
	if _, err := economysim.RestoreWorld(moneyCheckpointConfig(), reframe(t, raw, func(s []section) []section { return s })); err != nil {
		t.Fatalf("the unedited checkpoint: RestoreWorld error = %v, want nil", err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ck := reframe(t, raw, func(s []section) []section {
				return editSection(s, 13, func([]byte) []byte {
					v := slices.Clone(base)
					tt.edit(v)
					return bodyBytes(v)
				})
			})
			if _, err := economysim.RestoreWorld(moneyCheckpointConfig(), ck); !errors.Is(err, economysim.ErrCheckpointCorrupt) {
				t.Errorf("RestoreWorld(%s) error = %v, want ErrCheckpointCorrupt", tt.name, err)
			}
		})
	}
}

// TestFork_MoneySupply verifies fork-at-N with the module on, at the points of the
// plan: at tick 0 (nothing started), with the carries of the levy mid-year, and in
// a second variant after a flow has ended at its limit. The reference chain with
// overlay/faucet-sunk.toml (pack one) runs the first two; a small world with a
// limited levy runs the third, because it ends in 60 ticks, which the pack's
// placeholder amounts do not. The original, the restore and the restore of the
// restore must agree on snapshots, events and the state hash for 30 ticks.
//
// Requirement: FR-MON-040
func TestFork_MoneySupply(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	pack := loadValidPack(t, append(slices.Clone(packOne), "overlay/faucet-sunk.toml")...)
	for _, n := range []int{0, 60} {
		forkAtN(t, forkCase{Name: "one+faucet-sunk", Cfg: pack, Seed: 1, N: n, M: forkM})
	}
	forkAtN(t, forkCase{Name: "small, a flow ended", Cfg: moneyCheckpointConfig(), Seed: 1, N: 40, M: forkM,
		Inspect: func(t *testing.T, restored, since []economysim.Event) {
			for _, e := range restored {
				if _, ok := e.(economysim.MoneyFlowEnded); ok {
					t.Error("the restored world emitted MoneyFlowEnded after the boundary, want none: the flow had ended before it")
				}
			}
		}})
}
