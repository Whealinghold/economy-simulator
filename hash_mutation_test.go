package economysim

import (
	"bytes"
	"testing"
)

// Equivalent mutants of hash.go that no test can kill (gomu, one-line why each):
//   - text: `s != ""` -> `s > ""`: "" sorts below every string, so the two tests are the same.
//   - texts: `len(s) == 0` -> `len(s) <= 0`: a length is never negative.
//   - nested: `len(e.buf) == body` -> `len(e.buf) <= body`: pairs only appends, so the length is never below body.
//   - encodeList: `len(s) == 0` -> `len(s) <= 0`: a length is never negative.

// TestEncoder_NumWritesNegative pins that a negative integer is a non-zero
// value and is written, as its two's-complement 8 bytes, not skipped as if it
// were zero.
//
// Requirement: FR-CFG-104
func TestEncoder_NumWritesNegative(t *testing.T) {
	t.Parallel()
	got := encodeConfig(Config{World: WorldParams{Scale: -1}})
	want := []byte{
		hashVersion,
		0x00, 0x01, // World, field 1
		0x00, 0x01, // Scale, field 1
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // -1
		0x00, 0x00, // end of World
		0x00, 0x00, // end of Config
	}
	if !bytes.Equal(got, want) {
		t.Errorf("encodeConfig = % x, want % x", got, want)
	}
}
