package economysim

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// oneRegion is the configuration of the one-region row of the worked examples of
// spec-config.md 8.2.4.
func oneRegion() Config { return Config{Regions: []Region{{ID: "north"}}} }

// TestHash_DemandOnExample verifies the worked example of spec-config.md 8.2.4
// for a module that is on and empty: the one-region row with `000c 0001 0000
// 0000` inserted before the last terminator (Modules; Demand, id 1, and its
// terminator; the terminator of Modules), and the SHA-256 of the whole byte
// string. The digest was also computed by an independent encoder (Python) from
// the bytes of the row and of the Estates row of the same table, which agree with
// the spec.
//
// Requirement: FR-CFG-025, FR-CFG-060, FR-CFG-104
func TestHash_DemandOnExample(t *testing.T) {
	t.Parallel()
	const (
		base     = "0100070000000000000001000100000000000000056e6f7274680000" + "0000"
		insert   = "000c" + "0001" + "0000" + "0000"
		wantHash = "d4a84ffadc64ea332b99fe14423ca67eb052928dd5a16d91d054b27eeb1493d0"
	)
	cfg := oneRegion()
	cfg.Modules.Demand = &Demand{}
	want := base[:len(base)-4] + insert + base[len(base)-4:]
	got := encodeConfig(cfg.canonical())
	if hex.EncodeToString(got) != want {
		t.Errorf("encodeConfig(one region, Demand on) = %x, want %s", got, want)
	}
	sum := sha256.Sum256(got)
	if hex.EncodeToString(sum[:]) != wantHash || HashConfig(cfg) != sum {
		t.Errorf("HashConfig(one region, Demand on) = %x, want %s", HashConfig(cfg), wantHash)
	}
}

// TestHash_DemandOnEmptyDiffersFromNil verifies that a demand module that is on
// with no field hashes differently from one that is off, and that a module with
// one want differs from both, because a non-nil pointer is always written
// (spec-config.md 8.2.1).
//
// Requirement: FR-CFG-060, FR-CFG-104
func TestHash_DemandOnEmptyDiffersFromNil(t *testing.T) {
	t.Parallel()
	off, empty, one := oneRegion(), oneRegion(), oneRegion()
	empty.Modules.Demand = &Demand{}
	one.Modules.Demand = &Demand{Wants: []Want{{ID: "w"}}}
	tests := []struct {
		name string
		a, b Config
	}{
		{"off and on empty", off, empty},
		{"on empty and one want", empty, one},
		{"off and one want", off, one},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if HashConfig(tt.a) == HashConfig(tt.b) {
				t.Errorf("HashConfig(%s) are equal, want different hashes", tt.name)
			}
		})
	}
}

// moduleIDs are the field ids of the modules that are built, by field name: the
// position of the module in the Modules row of spec-config.md 3.1 (8.2.2). Each
// module's change appends its row.
var moduleIDs = map[string]uint16{"Demand": 1, "MoneySupply": 2, "Resources": 5, "Subsistence": 8, "Estates": 25}

// TestHashConfig_ModulePairsAscending verifies, from the bytes, that the pairs of
// Modules are written in ascending id order and under the id of the module's
// position, for a configuration with every built module on and fully populated:
// the ids of every struct in the encoding are read back with a decoder driven by
// the types, and each must be strictly ascending.
//
// Guard: the pairs of Modules are written by one function that a later module
// inserts into at its id, not at the end. A module appended at the end of the
// function with a smaller id would write its pair out of order and move no test
// until the hash of an all-modules-on configuration was pinned; the check of
// the ids against moduleIDs is extended by each module's change.
func TestHashConfig_ModulePairsAscending(t *testing.T) {
	t.Parallel()
	var c Config
	var n int64
	fillModules(reflect.ValueOf(&c.Modules).Elem(), &n)
	bytes := encodeConfig(c.canonical())
	r := &idReader{b: bytes[1:]} // the version byte
	if err := r.structOf(reflect.TypeFor[Config](), "Config"); err != nil {
		t.Fatalf("decode of the encoding: %v", err)
	}
	if len(r.b) != 0 {
		t.Errorf("decode left %d bytes unread, want 0", len(r.b))
	}
	modules := r.modules
	if want := len(moduleIDs); len(modules) != want {
		t.Fatalf("Modules wrote %d pairs, want %d (one per built module)", len(modules), want)
	}
	ft := reflect.TypeFor[Modules]()
	for i := range ft.NumField() {
		if id := moduleIDs[ft.Field(i).Name]; !slices.Contains(modules, id) {
			t.Errorf("module %s: pair id %d not written, ids written %v", ft.Field(i).Name, id, modules)
		}
	}
}

// fillModules sets every pointer of Modules to a populated value: integers by a
// counter, strings as padded counters, two elements in every list.
func fillModules(v reflect.Value, n *int64) {
	switch v.Kind() {
	case reflect.Int64:
		*n++
		v.SetInt(*n)
	case reflect.String:
		*n++
		v.SetString(fmt.Sprintf("s%05d", *n))
	case reflect.Struct:
		for i := range v.NumField() {
			fillModules(v.Field(i), n)
		}
	case reflect.Slice:
		s := reflect.MakeSlice(v.Type(), 2, 2)
		for i := range 2 {
			fillModules(s.Index(i), n)
		}
		v.Set(s)
	case reflect.Pointer:
		p := reflect.New(v.Type().Elem())
		fillModules(p.Elem(), n)
		v.Set(p)
	}
}

// idReader walks an encoding with the types of the configuration, fails on a
// struct whose pair ids are not strictly ascending, and records the ids of the
// pairs of Modules.
type idReader struct {
	b       []byte
	modules []uint16
}

// presentationLabels are the fields that have no id (spec-config.md 8.3).
var presentationLabels = map[string]bool{"Want.Label": true}

// structOf reads the pairs of one struct and its terminator.
func (r *idReader) structOf(t reflect.Type, name string) error {
	var fields []reflect.StructField
	for i := range t.NumField() {
		if f := t.Field(i); !presentationLabels[name+"."+f.Name] && !isDistanceLabel(name, f.Name) && !isLabel(f.Name) {
			fields = append(fields, f)
		}
	}
	var last uint16
	for {
		if len(r.b) < 2 {
			return fmt.Errorf("%s: truncated", name)
		}
		id := binary.BigEndian.Uint16(r.b)
		r.b = r.b[2:]
		if id == 0 {
			return nil
		}
		field := reflect.StructField{}
		if name == "Modules" {
			// A module's id is its position in the Modules row, not its index
			// among the fields that are built.
			for _, f := range fields {
				if moduleIDs[f.Name] == id {
					field = f
				}
			}
			if field.Name == "" {
				return fmt.Errorf("%s: id %d is the id of no built module", name, id)
			}
		} else {
			if int(id) > len(fields) {
				return fmt.Errorf("%s: id %d beyond its %d fields", name, id, len(fields))
			}
			field = fields[id-1]
		}
		if id <= last {
			return fmt.Errorf("%s: pair id %d after %d, want strictly ascending", name, id, last)
		}
		last = id
		if name == "Modules" {
			r.modules = append(r.modules, id)
		}
		if err := r.value(field.Type, name+"."+field.Name); err != nil {
			return err
		}
	}
}

func isLabel(field string) bool { return field == "Label" || field == "UnitLabel" }

func isDistanceLabel(typ, field string) bool { return typ == "WorldParams" && field == "DistanceLabel" }

// value reads one value of type t.
func (r *idReader) value(t reflect.Type, name string) error {
	switch t.Kind() {
	case reflect.Int64:
		r.b = r.b[8:]
	case reflect.Bool:
		r.b = r.b[1:]
	case reflect.String:
		n := binary.BigEndian.Uint64(r.b)
		r.b = r.b[8+n:]
	case reflect.Pointer:
		return r.structOf(t.Elem(), t.Elem().Name())
	case reflect.Struct:
		return r.structOf(t, t.Name())
	case reflect.Slice:
		count := binary.BigEndian.Uint64(r.b)
		r.b = r.b[8:]
		for range count {
			if err := r.value(t.Elem(), strings.TrimSuffix(name, "s")); err != nil {
				return err
			}
		}
	}
	return nil
}
