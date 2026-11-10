package economysim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestHash_MerchantsFieldIds verifies the bytes of the merchants module and of
// the merchant block under the ids of spec-config.md 8.2.2 (MaxOpen 1; Modes 1 to
// TakeUnits 9; Role.MerchantFeeShare 21 and Role.Merchant 22), that the module is
// the pair of id 10 after Migration (9) and before Estates (25), that each field
// changes the hash, that a block of all zeros is still written, that the share
// and the block are pairs only with the block, and that the order of the modes
// does not matter.
//
// Requirement: FR-MER-053, FR-CFG-079, FR-CFG-105
//
// Guard: a pair written out of id order, a field left out of the hash or an off
// module that leaves a trace would hash a configuration differently from the one
// the spec defines.
func TestHash_MerchantsFieldIds(t *testing.T) {
	t.Parallel()
	id := func(n uint16) []byte { return binary.BigEndian.AppendUint16(nil, n) }
	u64 := func(n uint64) []byte { return binary.BigEndian.AppendUint64(nil, n) }
	num := func(n uint16, v uint64) []byte { return bytes.Join([][]byte{id(n), u64(v)}, nil) }

	e := &encoder{}
	pairsModules(e, Modules{Estates: &Estates{}, Merchants: &Merchants{MaxOpen: 1_000}, Migration: &Migration{}})
	want := bytes.Join([][]byte{id(10), num(1, 1_000), {0, 0}, id(25), {0, 0}}, nil)
	if got := e.buf; !bytes.Contains(got, want) || bytes.Index(got, id(9)) > bytes.Index(got, id(10)) {
		t.Errorf("pairsModules(Migration, Merchants, Estates) = % x, want the pair of id 10 % x after id 9 and before id 25", got, want)
	}

	e = &encoder{}
	pairsMerchant(e, Merchant{Modes: []string{"m"}, LotUnits: 1, FeeShareSpreadBP: 2_500, MaxSourceAge: 2, TimeCostBP: 10_000,
		ConsigneeCutBP: 500, Consigns: true, TakeMarginBP: 1_000, TakeUnits: 150})
	modes := bytes.Join([][]byte{id(1), u64(1), u64(1), []byte("m")}, nil)
	rest := bytes.Join([][]byte{num(2, 1), num(3, 2_500), num(4, 2), num(5, 10_000), num(6, 500), append(id(7), 1), num(8, 1_000), num(9, 150)}, nil)
	if got := e.buf; !bytes.Equal(got, append(modes, rest...)) {
		t.Errorf("pairsMerchant(defaults, coach) = % x, want % x", got, append(modes, rest...))
	}

	// The role: a pair of id 21 and a block of id 22, written even when the block is all zero.
	tail := func(r Role) []byte {
		e := &encoder{}
		pairsRole(e, r, false, false)
		return e.buf
	}
	bare := tail(Role{ID: "a", MerchantFeeShare: 5_000})
	if !bytes.Equal(bare, tail(Role{ID: "a"})) {
		t.Errorf("pairsRole(share 5000, no block) = % x, want the bytes of a role with no share: the share is a pair only with the block", bare)
	}
	zero := tail(Role{ID: "a", Merchant: &Merchant{}})
	if !bytes.HasSuffix(zero, append(id(22), 0, 0)) {
		t.Errorf("pairsRole(zero block) = % x, want it to end with the id 22 and the terminator", zero)
	}
	if full := tail(Role{ID: "a", MerchantFeeShare: 5_000, Merchant: &Merchant{}}); !bytes.Contains(full, num(21, 5_000)) || bytes.Index(full, id(21)) > bytes.LastIndex(full, id(22)) {
		t.Errorf("pairsRole(share and block) = % x, want the pair 21 before the block 22", full)
	}

	base := Config{Roles: []Role{{ID: "a"}}}
	with := func(mutate func(r *Role, m *Merchants)) Config {
		c := base
		c.Roles = []Role{{ID: "a", MerchantFeeShare: 5_000, Merchant: &Merchant{Modes: []string{"x", "y"}, LotUnits: 1, FeeShareSpreadBP: 2_500,
			MaxSourceAge: 2, TimeCostBP: 10_000, ConsigneeCutBP: 500, TakeMarginBP: 1_000}}}
		m := Merchants{MaxOpen: 1_000}
		c.Modules.Merchants = &m
		if mutate != nil {
			mutate(&c.Roles[0], c.Modules.Merchants)
		}
		return c
	}
	on := with(nil)
	if HashConfig(on) == HashConfig(base) || HashConfig(with(func(r *Role, m *Merchants) { r.Merchant = &Merchant{} })) == HashConfig(base) {
		t.Errorf("HashConfig of the block, on or empty, equals that of no block, want it hashed")
	}
	for name, mutate := range map[string]func(r *Role, m *Merchants){
		"MaxOpen":          func(r *Role, m *Merchants) { m.MaxOpen++ },
		"MerchantFeeShare": func(r *Role, m *Merchants) { r.MerchantFeeShare++ },
		"Modes":            func(r *Role, m *Merchants) { r.Merchant.Modes = []string{"x"} },
		"LotUnits":         func(r *Role, m *Merchants) { r.Merchant.LotUnits++ },
		"FeeShareSpreadBP": func(r *Role, m *Merchants) { r.Merchant.FeeShareSpreadBP++ },
		"MaxSourceAge":     func(r *Role, m *Merchants) { r.Merchant.MaxSourceAge++ },
		"TimeCostBP":       func(r *Role, m *Merchants) { r.Merchant.TimeCostBP++ },
		"ConsigneeCutBP":   func(r *Role, m *Merchants) { r.Merchant.ConsigneeCutBP++ },
		"Consigns":         func(r *Role, m *Merchants) { r.Merchant.Consigns = true },
		"TakeMarginBP":     func(r *Role, m *Merchants) { r.Merchant.TakeMarginBP++ },
		"TakeUnits":        func(r *Role, m *Merchants) { r.Merchant.TakeUnits++ },
		"Strategy":         func(r *Role, m *Merchants) { r.Strategy = merchantStrategy },
		"the module":       func(r *Role, m *Merchants) { *m = Merchants{} },
	} {
		if HashConfig(with(mutate)) == HashConfig(on) {
			t.Errorf("HashConfig did not change with %s, want it hashed", name)
		}
	}
	swapped := with(func(r *Role, m *Merchants) { r.Merchant.Modes = []string{"y", "x"} })
	if HashConfig(swapped) != HashConfig(on) {
		t.Errorf("HashConfig changed with the order of the modes, want the modes hashed by ID")
	}
}
