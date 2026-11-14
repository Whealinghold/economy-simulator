package economysim_test

import (
	"bytes"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// storagePackDoc is the storage tables of a file, layered over the pack one: a base
// room far above anything a household of the pack holds, a row for the cooks that
// leaves them unlimited, and two classes written in the order opposite to their IDs.
const storagePackDoc = `
[modules.storage.base]
mass = 100000
bulk = 100000

[modules.storage.role.cook]

[modules.storage.class.shed]
mass = 50
bulk = 50

[modules.storage.class.bin]
mass = 400
bulk = 300
label = "a grain bin"
`

// TestStorage_PackFileToWorld verifies the path of a consumer from files to a world:
// the pack one with the storage tables layered over it loads and validates, a world
// is built from it with no warning of the module, the world's configuration is the
// canonical form of the loaded one (classes by ID), and its hash is the loaded
// configuration's and differs from the pack's without the module.
//
// Requirement: FR-STO-002, FR-STO-003, FR-CFG-085
func TestStorage_PackFileToWorld(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadPackBytes(t, []byte(storagePackDoc), packOne...)
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
	if err != nil {
		t.Fatalf("NewWorld(pack one + storage) error = %v, want nil", err)
	}
	got := w.Config().Modules.Storage
	want := &economysim.Storage{
		Base:  &economysim.Capacity{Mass: 100_000, Bulk: 100_000},
		Roles: []economysim.RoleRoom{{Role: "cook"}},
		Classes: []economysim.StoreClass{
			{ID: "bin", Mass: 400, Bulk: 300, Label: "a grain bin"},
			{ID: "shed", Mass: 50, Bulk: 50},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("World.Config().Modules.Storage = %+v, want %+v (canonical: classes by ID)", got, want)
	}
	if w.ConfigHash() != economysim.HashConfig(cfg) {
		t.Errorf("World.ConfigHash() = %x, want HashConfig of the loaded configuration %x", w.ConfigHash(), economysim.HashConfig(cfg))
	}
	if off := loadValidPack(t, packOne...); economysim.HashConfig(off) == w.ConfigHash() {
		t.Errorf("hash of the pack with the module = that of the pack without, want different")
	}
	// The base room is roomy, so the only findings are the two classes no row names.
	var rules []string
	for _, wn := range w.Warnings() {
		if strings.HasPrefix(strings.Join(wn.Path, "."), "modules.storage") {
			rules = append(rules, wn.Rule+"@"+strings.Join(wn.Path, "."))
		}
	}
	if wantW := []string{"store_class_unused@modules.storage.classes[0]", "store_class_unused@modules.storage.classes[1]"}; !slices.Equal(rules, wantW) {
		t.Errorf("World.Warnings() of the module = %v, want %v", rules, wantW)
	}
}

// sameButHash returns res with the ConfigHash of WorldCreated and of the final
// snapshot set to hash: the one field of a run that a module's presence must change
// (FR-CFG-026).
func sameButHash(res result, hash [32]byte) result {
	out := res
	out.Final.ConfigHash = hash
	out.Events = slices.Clone(res.Events)
	for i, e := range out.Events {
		if wc, ok := e.(economysim.WorldCreated); ok {
			wc.ConfigHash = hash
			out.Events[i] = wc
		}
	}
	return out
}

// TestStorage_NewWorldRejectsASmallRoom verifies that construction refuses a pack
// whose storage tables give a base room below one tick's upkeep, with the sentinel
// of the rule and no world, so a file that breaks the module never reaches a run.
//
// Requirement: FR-STO-038
func TestStorage_NewWorldRejectsASmallRoom(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test, skipped with -short")
	}
	cfg := loadValidPack(t, packOne...)
	cfg.Modules.Storage = &economysim.Storage{Base: &economysim.Capacity{Mass: 1, Bulk: 1}}
	w, err := economysim.NewWorld(cfg, economysim.WithSeed(1))
	if !errors.Is(err, economysim.ErrRoomShape) || w != nil {
		t.Errorf("NewWorld(base room 1) = %v, %v, want a nil world and ErrRoomShape", w, err)
	}
}

// TestStorage_InertOverARun verifies, over a run of the pack one with the audit on
// and a recording observer, that a world built with the storage module on behaves
// exactly as one built with it off: the same events in the same order (but for the
// configuration hash of WorldCreated, which the module moves) and the same final
// snapshot, for two seeds. The module has no state, no position and no event
// yet, so its presence, with or without a limited role, moves nothing.
//
// Requirement: FR-STO-001
//
// Guard: the module is inert until its commitments are built (plan 3-sto Phase 5).
// The empty module and the module with no limited role must stay identical to off
// for good; the case with a base room is the one that Phase 4 and 5 will change and
// that their PR replaces with the invariant of spec-storage.md 4.4.
func TestStorage_InertOverARun(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	off := loadValidPack(t, packOne...)
	variants := []struct {
		name string
		on   *economysim.Storage
	}{
		{"the empty module", &economysim.Storage{}},
		{"a base room and a class", &economysim.Storage{
			Base:    &economysim.Capacity{Mass: 100_000, Bulk: 100_000},
			Classes: []economysim.StoreClass{{ID: "bin", Mass: 400, Bulk: 300}},
		}},
	}
	for seed := uint64(1); seed <= uint64(seedsFor(raceEnabled, 2)); seed++ {
		base := runPack(t, off, runSpec{Seed: seed, Ticks: 30, Observe: true, Audit: true})
		for _, v := range variants {
			t.Run(v.name, func(t *testing.T) {
				cfg := off
				cfg.Modules.Storage = v.on
				raw := runPack(t, cfg, runSpec{Seed: seed, Ticks: 30, Observe: true, Audit: true})
				if raw.World.ConfigHash() == base.World.ConfigHash() {
					t.Errorf("seed %d: the configuration hash with %s = that of the module off, want different", seed, v.name)
				}
				res := sameButHash(raw, base.World.ConfigHash())
				if !bytes.Equal(renderEvents(res), renderEvents(base)) {
					t.Errorf("seed %d: the run with %s renders differently from the run with the module off, want identical", seed, v.name)
				}
				if !reflect.DeepEqual(res.Events, base.Events) {
					t.Errorf("seed %d: events with %s differ from the module off (%d and %d events), want identical", seed, v.name, len(res.Events), len(base.Events))
				}
			})
		}
	}
}
