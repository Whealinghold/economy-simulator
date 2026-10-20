package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// FuzzLoad verifies, for any three documents, that the loader never panics,
// that a rejected set returns the zero Config and an error that wraps a
// loader sentinel, and that an accepted set loads again to an equal Config with
// an equal hash, and its Validate does not panic and returns nil or an error
// classified by a sentinel or a *ValidationError.
//
// The seeds are the layers of the packs one and second of testdata/chains, and
// documents at and beyond the nesting and breadth limits; the corpus files of
// testdata/fuzz/FuzzLoad hold the nested and the 10 000-key documents.
//
// Requirement: FR-CFG-024, NFR-CFG-002
func FuzzLoad(f *testing.F) {
	read := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join("..", "testdata", "chains", filepath.FromSlash(name)))
		if err != nil {
			f.Fatalf("read seed %s: %v", name, err)
		}
		return b
	}
	f.Add(read("base.toml"), read("one.toml"), read("warm-one.toml"))
	f.Add(read("second/second.toml"), read("second/warm-second.toml"), []byte(nil))
	f.Add(read("base.toml"), read("one.toml"), read("defects/d4_long_job.toml"))
	f.Add([]byte(minimal), []byte("[world]\nscale = 2\n"), []byte("[world]\nscale = 3\n"))
	f.Add([]byte(minimal+"[a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p.q]\n"), []byte("x = [[[[[[1]]]]]]\n"), []byte("y = { a = { b = 1 } }\n"))
	f.Add([]byte(minimal+"[modules.estates]\n"), []byte("[role.a]\ntarget_ratio = 1.5\n"), []byte("world = 1\n"))
	f.Fuzz(func(t *testing.T, a, b, c []byte) {
		cfg, err := config.Parse(a, b, c)
		if err != nil {
			known := false
			for _, s := range loaderSentinels {
				known = known || errors.Is(err, s)
			}
			if !known {
				t.Errorf("Parse(%q, %q, %q) error = %v, want a loader sentinel", a, b, c, err)
			}
			if !reflect.DeepEqual(cfg, economysim.Config{}) {
				t.Errorf("Parse(%q, %q, %q) rejected but returned a non-zero Config", a, b, c)
			}
			if strings.ContainsAny(err.Error(), "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x0b\x0c\x0e\x0f\x1b\x7f") {
				t.Errorf("Parse(%q, %q, %q) error text has a control character: %q", a, b, c, err.Error())
			}
			return
		}
		if verr := cfg.Validate(); verr != nil && !classified(verr) {
			t.Errorf("Validate() of an accepted set = %v, want nil, a sentinel or a *ValidationError", verr)
		}
		again, err := config.Parse(a, b, c)
		if err != nil {
			t.Fatalf("Parse(%q, %q, %q) accepted and then rejected: %v", a, b, c, err)
		}
		if !reflect.DeepEqual(cfg, again) {
			t.Errorf("Parse(%q, %q, %q) twice: %+v and %+v, want equal", a, b, c, cfg, again)
		}
		if economysim.HashConfig(cfg) != economysim.HashConfig(again) {
			t.Errorf("HashConfig of an accepted document differs between loads")
		}
	})
}

// classified reports whether a Validate error is made only of *ValidationError
// values (alone or in a join) whose rule is an exported sentinel, and
// *ViabilityError values of the five viability rules, or is the ErrTooManyErrors
// cut.
func classified(err error) bool {
	if errors.Is(err, economysim.ErrTooManyErrors) {
		return true
	}
	ves, vias := validationErrors(err), viabilityErrors(err)
	for _, ve := range ves {
		if ve.Rule == nil || !slices.ContainsFunc(validationSentinels, func(s error) bool { return errors.Is(ve.Rule, s) }) {
			return false
		}
	}
	for _, ve := range vias {
		if !slices.Contains(viabilityRules, ve.Rule) || !errors.Is(ve, economysim.ErrNotViable) {
			return false
		}
	}
	return len(ves)+len(vias) > 0
}
