package economysim_test

import (
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// presentationOnly is the allow-list of the hash test (spec-config.md 8.3): the
// fields whose doc comment says presentation only and that no result and no
// hash reads. At Stage 0 they are the six labels of the core. A module's change
// adds its own labels here and in the doc comment of the field.
var presentationOnly = map[string]bool{
	"Commodity.Label":           true,
	"Commodity.UnitLabel":       true,
	"Recipe.Label":              true,
	"Role.Label":                true,
	"Region.Label":              true,
	"WorldParams.DistanceLabel": true,
}

// pinnedReferenceHash is the hash of referenceConfig, the all-off reference
// configuration (spec-config.md 8.2.5).
const pinnedReferenceHash = "9abf7b8fa64bc36b9f223f7d195b81e7934fa9d88c297085ca4128d0a98275a0"

// pinnedReferenceHashAnchorHoldOff is the pin that held before the field
// WorldParams.AnchorHold existed: the hash of referenceConfig with that field at
// its off value. It must not move, since a field at its off value is not written
// (FR-CFG-104, architecture.md 2.10.5).
const pinnedReferenceHashAnchorHoldOff = "9e3c1357520b0e052fee1e1d773d45e49ce992a8d1cfbd7029a6a24a91038285"

// hexHash renders the hash of cfg as lower-case hex, the form of the spec.
func hexHash(cfg economysim.Config) string {
	h := economysim.HashConfig(cfg)
	return hex.EncodeToString(h[:])
}

// populate sets every field of v to a distinct non-zero value: integers by a
// counter, strings as zero-padded counters (so that string order is counter
// order), bools true, every slice of two elements and every pointer allocated.
// Calling it twice with a counter at 0 gives two equal values.
func populate(v reflect.Value, n *int64) {
	switch v.Kind() {
	case reflect.Int64:
		*n++
		v.SetInt(*n)
	case reflect.Bool:
		v.SetBool(true)
	case reflect.String:
		*n++
		v.SetString(fmt.Sprintf("s%05d", *n))
	case reflect.Struct:
		for i := range v.NumField() {
			populate(v.Field(i), n)
		}
	case reflect.Slice:
		s := reflect.MakeSlice(v.Type(), 2, 2)
		for i := range 2 {
			populate(s.Index(i), n)
		}
		v.Set(s)
	case reflect.Pointer:
		p := reflect.New(v.Type().Elem())
		populate(p.Elem(), n)
		v.Set(p)
	default:
		panic("populate: unsupported kind " + v.Kind().String())
	}
}

// populatedConfig returns a Config with every field set and two elements in
// every list, so that a test can change each field in turn.
func populatedConfig() economysim.Config {
	var c economysim.Config
	var n int64
	populate(reflect.ValueOf(&c).Elem(), &n)
	return c
}

// reverseLists reverses, in place and at every depth, each slice of v except
// the ones named in keep ("Type.Field"): the lists whose order is their meaning.
func reverseLists(v reflect.Value, keep map[string]bool) {
	switch v.Kind() {
	case reflect.Pointer:
		reverseLists(v.Elem(), keep)
	case reflect.Struct:
		for i := range v.NumField() {
			f := v.Field(i)
			if f.Kind() == reflect.Slice && !keep[v.Type().Name()+"."+v.Type().Field(i).Name] {
				swap := reflect.Swapper(f.Interface())
				for lo, hi := 0, f.Len()-1; lo < hi; lo, hi = lo+1, hi-1 {
					swap(lo, hi)
				}
			}
			reverseLists(f, keep)
		}
	case reflect.Slice:
		for i := range v.Len() {
			reverseLists(v.Index(i), keep)
		}
	}
}

// fieldTarget is one field of the configuration that a test changes in turn: a
// scalar, a list (changed by appending an element) or a pointer (changed by
// setting it nil), reached from the root by path. A path step of -1 is element
// 0 of a list or the target of a pointer.
type fieldTarget struct {
	name string // "Type.Field"
	path []int
}

// collectTargets lists every field reachable from t, descending into structs,
// into the first element of lists of structs and through pointers.
func collectTargets(t reflect.Type, path []int, out *[]fieldTarget) {
	for i := range t.NumField() {
		f := t.Field(i)
		p := append(slices.Clone(path), i)
		switch f.Type.Kind() {
		case reflect.Struct:
			collectTargets(f.Type, p, out)
		case reflect.Slice:
			*out = append(*out, fieldTarget{t.Name() + "." + f.Name, p})
			if f.Type.Elem().Kind() == reflect.Struct {
				collectTargets(f.Type.Elem(), append(slices.Clone(p), -1), out)
			}
		case reflect.Pointer:
			*out = append(*out, fieldTarget{t.Name() + "." + f.Name, p})
			collectTargets(f.Type.Elem(), append(slices.Clone(p), -1), out)
		default:
			*out = append(*out, fieldTarget{t.Name() + "." + f.Name, p})
		}
	}
}

// at returns the value that path reaches from root.
func at(root reflect.Value, path []int) reflect.Value {
	v := root
	for _, step := range path {
		switch {
		case step >= 0:
			v = v.Field(step)
		case v.Kind() == reflect.Slice:
			v = v.Index(0)
		default:
			v = v.Elem()
		}
	}
	return v
}

// change makes v a different value of its type: an integer plus one, a bool
// flipped, a string with a suffix, a list one zero element longer, a pointer
// nil.
func change(v reflect.Value) {
	switch v.Kind() {
	case reflect.Int64:
		v.SetInt(v.Int() + 1)
	case reflect.Bool:
		v.SetBool(!v.Bool())
	case reflect.String:
		v.SetString(v.String() + "x")
	case reflect.Slice:
		v.Set(reflect.Append(v, reflect.Zero(v.Type().Elem())))
	case reflect.Pointer:
		v.Set(reflect.Zero(v.Type()))
	default:
		panic("change: unsupported kind " + v.Kind().String())
	}
}

// TestHashConfig_SpecVectors verifies the encoder against the worked examples of
// spec-config.md 8.2.4: the empty Config, one region, the same with an empty
// WarmStart, and the five-entity configuration whose 350 bytes the spec lists.
// The Estates row of that table needs the Estates module, which is built later,
// and is added with it.
//
// Requirement: FR-CFG-025, FR-CFG-104, FR-CFG-105
func TestHashConfig_SpecVectors(t *testing.T) {
	t.Parallel()
	fiveEntities := economysim.Config{
		Commodities: []economysim.Commodity{{ID: "grain", Mass: 1, Bulk: 1, ReferencePrice: 100}},
		Recipes: []economysim.Recipe{{
			ID: "grow", Duration: 1,
			Outputs: []economysim.Output{{Commodity: "grain", Quantity: 1, CostShare: 10_000}},
		}},
		Roles: []economysim.Role{{
			ID: "farmer", Recipes: []string{"grow"}, StartMoney: 10_000, Strategy: "default",
			TargetRatio: 15_000, DefaultRatio: 10_000, ExpectedWorkTicks: 250, MinWorkShare: 4_000,
		}},
		Regions: []economysim.Region{{ID: "north", ReferenceBasket: 100}},
		Groups:  []economysim.Group{{Region: "north", Role: "farmer", Households: 10}},
	}
	oneRegion := economysim.Config{Regions: []economysim.Region{{ID: "north"}}}
	oneRegionWarm := oneRegion
	oneRegionWarm.WarmStart = &economysim.WarmStart{}
	tests := []struct {
		name string
		cfg  economysim.Config
		want string
	}{
		{"empty config", economysim.Config{}, "fb50dc0717ff266cf9baf82b1ce7a1c2ef6d9247859680b11a19fb7077f5f222"},
		{"one region", oneRegion, "ed652dfd86904c18fd76feb17175e7efeef6d4afe94bec1e3b4f5a2e2409f013"},
		{"one region with empty warm start", oneRegionWarm, "a2f375f381fced5c69544bf3dfb14446d513d6e0eaceb9574964bbfe8ea8a812"},
		{"five entities", fiveEntities, "87a8a42335e50ea33777b9290b1adc14c80a8b8a1451e31b72dd664f72c2f0f5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := hexHash(tt.cfg); got != tt.want {
				t.Errorf("HashConfig(%s) = %s, want %s", tt.name, got, tt.want)
			}
		})
	}
}

// TestHashConfig_WarmStartPresence verifies that a nil WarmStart and an empty
// non-nil one are two configurations with two hashes, because a non-nil pointer
// is always written.
//
// Requirement: FR-CFG-104
func TestHashConfig_WarmStartPresence(t *testing.T) {
	t.Parallel()
	nilWarm := referenceConfig()
	nilWarm.WarmStart = nil
	emptyWarm := referenceConfig()
	emptyWarm.WarmStart = &economysim.WarmStart{}
	if hexHash(nilWarm) == hexHash(emptyWarm) {
		t.Errorf("HashConfig(WarmStart nil) = HashConfig(WarmStart &WarmStart{}) = %s, want different hashes", hexHash(nilWarm))
	}
	if hexHash(emptyWarm) == hexHash(referenceConfig()) {
		t.Errorf("HashConfig(WarmStart &WarmStart{}) equals the hash with a warm start table, want different hashes")
	}
}

// TestHashConfig_CanonicalOrder verifies that writing every entity list in the
// reverse order gives the same hash, so the order of a file or a slice cannot
// move it. The calendar's months and seasons are ordered lists and are kept.
//
// Requirement: FR-CFG-005
func TestHashConfig_CanonicalOrder(t *testing.T) {
	t.Parallel()
	keep := map[string]bool{"Calendar.Months": true, "Calendar.Seasons": true}
	reversed := populatedConfig()
	reverseLists(reflect.ValueOf(&reversed).Elem(), keep)
	base := populatedConfig()
	if reflect.DeepEqual(base, reversed) {
		t.Fatal("reversing the lists left the configuration unchanged, so the test would prove nothing")
	}
	if got, want := hexHash(reversed), hexHash(base); got != want {
		t.Errorf("HashConfig(lists reversed) = %s, want %s (equal to the order as written)", got, want)
	}
}

// TestHashConfig_OrderedListsKeepOrder verifies that the months and seasons of
// the calendar are ordered lists: reversing them is a different calendar and
// moves the hash, and loading may not sort them.
//
// Requirement: FR-CFG-034
func TestHashConfig_OrderedListsKeepOrder(t *testing.T) {
	t.Parallel()
	base := populatedConfig()
	for _, list := range []string{"Months", "Seasons"} {
		t.Run(strings.ToLower(list), func(t *testing.T) {
			t.Parallel()
			c := populatedConfig()
			f := reflect.ValueOf(&c.Calendar).Elem().FieldByName(list)
			reflect.Swapper(f.Interface())(0, 1)
			if hexHash(c) == hexHash(base) {
				t.Errorf("HashConfig(Calendar.%s reversed) = HashConfig(as written), want a different hash", list)
			}
		})
	}
}

// TestHashConfig_EveryFieldMovesHash verifies, by reflection over a fully
// populated Config, that changing any one field changes the hash unless the
// field is on the presentation-only allow-list, in which case it must not. A
// field that the encoder forgets, or a label that gets hashed, fails here. The
// seed, the logger and the observers are not fields, so the other half of the
// requirement holds by the signature of HashConfig; its world half is Stage 1.
//
// Requirement: FR-CFG-026, FR-CFG-071
func TestHashConfig_EveryFieldMovesHash(t *testing.T) {
	t.Parallel()
	var targets []fieldTarget
	collectTargets(reflect.TypeFor[economysim.Config](), nil, &targets)
	if len(targets) < 100 {
		t.Fatalf("collected %d fields, want the whole configuration (more than 100)", len(targets))
	}
	base := hexHash(populatedConfig())
	seen := map[string]bool{}
	for _, tg := range targets {
		seen[tg.name] = true
		t.Run(tg.name+fmt.Sprint(tg.path), func(t *testing.T) {
			t.Parallel()
			c := populatedConfig()
			change(at(reflect.ValueOf(&c).Elem(), tg.path))
			moved := hexHash(c) != base
			if allowed := presentationOnly[tg.name]; moved == allowed {
				t.Errorf("HashConfig after changing %s moved = %v, want %v (presentation only = %v)",
					tg.name, moved, !allowed, allowed)
			}
		})
	}
	for name := range presentationOnly {
		if !seen[name] {
			t.Errorf("allow-listed field %s is not a field of Config", name)
		}
	}
}

// TestHashConfig_AllowListMatchesDocComments verifies that the allow-list of the
// hash test equals the set of fields whose doc comment in config.go says
// "presentation only": a field whose comment says so and that is not listed
// fails, and so does a listed field whose comment does not say so.
//
// Requirement: FR-CFG-026
func TestHashConfig_AllowListMatchesDocComments(t *testing.T) {
	t.Parallel()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "config.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse config.go: %v", err)
	}
	documented := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		spec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		st, ok := spec.Type.(*ast.StructType)
		if !ok {
			return true
		}
		for _, f := range st.Fields.List {
			if f.Doc == nil || !strings.Contains(strings.ToLower(f.Doc.Text()), "presentation only") {
				continue
			}
			for _, name := range f.Names {
				documented[spec.Name.Name+"."+name.Name] = true
			}
		}
		return true
	})
	if !maps.Equal(documented, presentationOnly) {
		t.Errorf("fields whose doc comment says presentation only = %v, want the allow-list %v",
			slices.Sorted(maps.Keys(documented)), slices.Sorted(maps.Keys(presentationOnly)))
	}
}

// TestHashConfig_ReferencePinned verifies that the hash of the all-off
// reference configuration equals a constant, so an append that was not off is
// seen before any golden file exists.
//
// Requirement: FR-CFG-105
//
// Guard: this constant is the earliest signal that an append left a trace. A
// pull request that adds a field or a module and moves it has broken the rule of
// spec-config.md 8.2.3 (the default of an appended field is zero, a module that
// is off is not written) or has changed the encoding on purpose, and must say
// which. Without the test a module could change every hash and be noticed only
// when the first golden file fails.
func TestHashConfig_ReferencePinned(t *testing.T) {
	t.Parallel()
	if got := hexHash(referenceConfig()); got != pinnedReferenceHash {
		t.Errorf("HashConfig(referenceConfig()) = %s, want %s", got, pinnedReferenceHash)
	}
}

// TestHashConfig_AnchorHoldOffLeavesNoTrace verifies that the reference
// configuration with AnchorHold at 0 hashes to the value the pin had before the
// field was appended, and that 5 000 changes it.
//
// Requirement: FR-CFG-104, FR-CFG-107
//
// Guard: the pack opts into the anchor hold and so moved the pin of
// referenceConfig on purpose (a pack edit, testing.md 7.2). This test keeps the
// rule of architecture.md 2.10.5 under test: an appended field at its off value
// leaves no trace, so a library user who never sets it keeps every hash.
func TestHashConfig_AnchorHoldOffLeavesNoTrace(t *testing.T) {
	t.Parallel()
	cfg := referenceConfig()
	cfg.World.AnchorHold = 0
	if got := hexHash(cfg); got != pinnedReferenceHashAnchorHoldOff {
		t.Errorf("HashConfig(referenceConfig() with AnchorHold 0) = %s, want %s", got, pinnedReferenceHashAnchorHoldOff)
	}
	if hexHash(cfg) == hexHash(referenceConfig()) {
		t.Errorf("HashConfig with AnchorHold 0 equals HashConfig with 5 000, want them to differ")
	}
}

// TestHashConfig_DoesNotChangeInput verifies that HashConfig leaves its argument
// as it was, in particular in the order the caller wrote it, because the sort of
// canonical order runs on a copy.
//
// Guard: a world owns its configuration by deep copy (spec-config.md 3.5); a
// hash that sorted the caller's slices in place would change a Config the caller
// still holds, and the change would show only in a later, unrelated comparison.
func TestHashConfig_DoesNotChangeInput(t *testing.T) {
	t.Parallel()
	cfg := referenceConfig()
	// The reference lists upkeep as staple, prepared, fuel: not in canonical order.
	economysim.HashConfig(cfg)
	if want := referenceConfig(); !reflect.DeepEqual(cfg, want) {
		t.Errorf("Config after HashConfig differs from the Config before it")
	}
}

// TestHashConfig_Signature verifies that HashConfig is a pure function of a
// Config that returns the 32 bytes of the digest.
//
// Requirement: FR-CFG-106
func TestHashConfig_Signature(t *testing.T) {
	t.Parallel()
	var f func(economysim.Config) [32]byte = economysim.HashConfig
	if f(economysim.Config{}) != economysim.HashConfig(economysim.Config{}) {
		t.Error("HashConfig(Config{}) differs between two calls")
	}
}
