package config_test

import (
	"errors"
	"io/fs"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
	"unicode/utf8"

	economysim "github.com/Whealinghold/economy-sim"
	"github.com/Whealinghold/economy-sim/config"
)

// notTooLarge reports whether err is not a limit rejection: a document at a
// limit must pass the limits and go on to be mapped, where its keys are
// unknown.
func notTooLarge(err error) bool { return !errors.Is(err, economysim.ErrTooLarge) }

// TestParse_Limits verifies, for each limit of spec-config.md 6.3.1, that a
// document at the limit passes it and a document one over is ErrTooLarge naming
// the constant.
//
// Requirement: FR-CFG-022
func TestParse_Limits(t *testing.T) {
	t.Parallel()
	line := func(n int) []byte { return []byte(minimal + "#" + strings.Repeat("x", n-1) + "\n") }
	docs := func(n int) [][]byte {
		out := make([][]byte, n)
		for i := range out {
			out[i] = []byte(minimal)
		}
		return out
	}
	tests := []struct {
		name  string
		docs  [][]byte
		over  bool
		limit string
	}{
		{"documents at the limit", docs(economysim.MaxDocuments), false, ""},
		{"documents over the limit", docs(economysim.MaxDocuments + 1), true, "MaxDocuments"},
		{"one document at the size limit", [][]byte{padded(economysim.MaxDocumentBytes)}, false, ""},
		{"one document over the size limit", [][]byte{padded(economysim.MaxDocumentBytes + 1)}, true, "MaxDocumentBytes"},
		{"merged size at the limit", [][]byte{
			padded(economysim.MaxDocumentBytes), padded(economysim.MaxDocumentBytes),
			padded(economysim.MaxDocumentBytes), padded(economysim.MaxDocumentBytes),
		}, false, ""},
		{"merged size over the limit", [][]byte{
			padded(economysim.MaxDocumentBytes), padded(economysim.MaxDocumentBytes),
			padded(economysim.MaxDocumentBytes), padded(economysim.MaxDocumentBytes), []byte("\n"),
		}, true, "MaxMergedBytes"},
		{"line at the limit", [][]byte{line(economysim.MaxLineBytes)}, false, ""},
		{"line over the limit", [][]byte{line(economysim.MaxLineBytes + 1)}, true, "MaxLineBytes"},
		{"header of the deepest allowed key", [][]byte{[]byte(minimal + "[" + dotted(economysim.MaxNestingDepth) + "]\n")}, false, ""},
		{"header one segment too deep", [][]byte{[]byte(minimal + "[" + dotted(economysim.MaxNestingDepth+1) + "]\n")}, true, "MaxNestingDepth"},
		{"array header one segment too deep", [][]byte{[]byte(minimal + "[[" + dotted(economysim.MaxNestingDepth+1) + "]]\n")}, true, "MaxNestingDepth"},
		{"dotted key at the limit", [][]byte{[]byte(minimal + dotted(economysim.MaxNestingDepth) + " = 1\n")}, false, ""},
		{"dotted key one segment too deep", [][]byte{[]byte(minimal + dotted(economysim.MaxNestingDepth+1) + " = 1\n")}, true, "MaxNestingDepth"},
		// Header and key count together: 3 + 13 = 16, then 3 + 14 = 17.
		{"header and key at the limit", [][]byte{[]byte(minimal + "[h.i.j]\n" + dotted(13) + " = 1\n")}, false, ""},
		{"header and key together too deep", [][]byte{[]byte(minimal + "[h.i.j]\n" + dotted(14) + " = 1\n")}, true, "MaxNestingDepth"},
		// The key is one level and each array adds one: 1 + 15 = 16, then 1 + 16.
		{"arrays at the limit", [][]byte{[]byte(minimal + "x = " + strings.Repeat("[", 15) + "1" + strings.Repeat("]", 15) + "\n")}, false, ""},
		{"arrays one level too deep", [][]byte{[]byte(minimal + "x = " + strings.Repeat("[", 16) + "1" + strings.Repeat("]", 16) + "\n")}, true, "MaxNestingDepth"},
		{"inline tables at the limit", [][]byte{[]byte(minimal + "a = " + strings.Repeat("{ a = ", 15) + "1" + strings.Repeat(" }", 15) + "\n")}, false, ""},
		{"inline tables one level too deep", [][]byte{[]byte(minimal + "a = " + strings.Repeat("{ a = ", 16) + "1" + strings.Repeat(" }", 16) + "\n")}, true, "MaxNestingDepth"},
		{"arrays far over the library bound", [][]byte{[]byte(minimal + "x = " + strings.Repeat("[", 20_000) + strings.Repeat("]", 20_000) + "\n")}, true, "MaxNestingDepth"},
		{"keys of one table at the limit", [][]byte{[]byte(minimal + flatKeys("", economysim.MaxTableKeys-1))}, false, ""},
		{"keys of one table over the limit", [][]byte{[]byte(minimal + flatKeys("", economysim.MaxTableKeys))}, true, "MaxTableKeys"},
		{"keys of a dotted table over the limit", [][]byte{[]byte(minimal + flatKeys("x.", economysim.MaxTableKeys+1))}, true, "MaxTableKeys"},
		{"elements of an array at the limit", [][]byte{[]byte(minimal + "x = [" + strings.Repeat("0,", economysim.MaxTableKeys-1) + "0]\n")}, false, ""},
		{"elements of an array over the limit", [][]byte{[]byte(minimal + "x = [" + strings.Repeat("0,", economysim.MaxTableKeys) + "0]\n")}, true, "MaxTableKeys"},
		{"keys of an inline table over the limit", [][]byte{[]byte(minimal + "x = { " + strings.TrimSuffix(strings.ReplaceAll(flatKeys("", economysim.MaxTableKeys+1), "\n", ", "), ", ") + " }\n")}, true, "MaxTableKeys"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := config.Parse(tt.docs...)
			if !tt.over {
				if !notTooLarge(err) {
					t.Errorf("Parse(%s) error = %v, want no ErrTooLarge at the limit", tt.name, err)
				}
				return
			}
			if !errors.Is(err, economysim.ErrTooLarge) {
				t.Fatalf("Parse(%s) error = %v, want errors.Is ErrTooLarge", tt.name, err)
			}
			if !strings.Contains(err.Error(), tt.limit) {
				t.Errorf("Parse(%s) error = %q, want it to name %s", tt.name, err, tt.limit)
			}
		})
	}
}

// TestParse_KeysOfSeveralTablesAreCountedPerTable verifies that the key limit
// is per table: two tables of 6 000 keys each (12 000 in all, over the limit)
// pass, and a table whose children arrive by many headers is counted as one.
//
// Requirement: FR-CFG-022
func TestParse_KeysOfSeveralTablesAreCountedPerTable(t *testing.T) {
	t.Parallel()
	const each = economysim.MaxTableKeys * 6 / 10
	two := minimal + "[a]\n" + flatKeys("", each) + "[b]\n" + flatKeys("", each)
	if _, err := parseStrings(two); !notTooLarge(err) {
		t.Errorf("Parse(two tables of 6 000 keys) error = %v, want no ErrTooLarge", err)
	}
	var b strings.Builder
	b.WriteString(minimal)
	for i := range economysim.MaxTableKeys + 1 {
		b.WriteString("[t")
		b.WriteString(strings.Repeat("9", 1+i%3))
		b.WriteString(itoa(i))
		b.WriteString("]\n")
	}
	if _, err := parseStrings(b.String()); !errors.Is(err, economysim.ErrTooLarge) {
		t.Errorf("Parse(%d headers under the root) error = %v, want ErrTooLarge", economysim.MaxTableKeys+1, err)
	}
}

// itoa writes n in decimal.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for ; n > 0; n /= 10 {
		d = append([]byte{byte('0' + n%10)}, d...)
	}
	return string(d)
}

// TestParse_ErrorTextIsBoundedAndHasNoControlCharacters verifies that a path
// segment of an error is cut and quoted, so a 100 KB key and a key of control
// characters give a short single-line message.
//
// Requirement: FR-CFG-100
func TestParse_ErrorTextIsBoundedAndHasNoControlCharacters(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		key  string
		// tail is the text the error must hold: the cut segment, quoted, and
		// the cut marker; empty when the key is not cut.
		tail string
	}{
		{"oversized key", strings.Repeat("k", 100_000), strings.Repeat("k", 64) + `"...`},
		{"control characters", `\u0001\u0007\u001b[31m\u0000\n`, ""},
		// 3-byte runes: the 64-byte cut falls inside the 22nd rune (bytes 63 to
		// 65), so the cut backs up to byte 63, which is 21 whole runes.
		{"multibyte at the cut", strings.Repeat("€", 100), strings.Repeat("€", 21) + `"...`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseStrings(minimal + "[role.\"" + tt.key + "\"]\nbogus = 1\n")
			if !errors.Is(err, economysim.ErrUnknownKey) {
				t.Fatalf("Parse(%s) error = %v, want ErrUnknownKey", tt.name, err)
			}
			text := err.Error()
			if len(text) > 200 {
				t.Errorf("Parse(%s) error text is %d bytes, want at most 200", tt.name, len(text))
			}
			if !utf8.ValidString(text) {
				t.Errorf("Parse(%s) error text is not valid UTF-8: %q", tt.name, text)
			}
			if !strings.Contains(text, tt.tail) {
				t.Errorf("Parse(%s) error text = %q, want it to hold %q", tt.name, text, tt.tail)
			}
			for _, r := range text {
				if r < 0x20 || r == 0x7f {
					t.Errorf("Parse(%s) error text has control character %U: %q", tt.name, r, text)
					break
				}
			}
			ves := validationErrors(err)
			if len(ves) != 1 || len(ves[0].Path[1]) == 0 {
				t.Fatalf("Parse(%s) = %d errors, want 1 holding the key as given", tt.name, len(ves))
			}
		})
	}
}

// TestLoadFS_ReadsNamedFilesInOrder verifies LoadFS over an in-memory file
// system: the files are merged in the order of the names.
//
// Requirement: FR-CFG-023
func TestLoadFS_ReadsNamedFilesInOrder(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"base.toml":    {Data: []byte(minimal + "[world]\nscale = 2\n")},
		"overlay.toml": {Data: []byte("[world]\nscale = 5\n")},
	}
	tests := []struct {
		names []string
		want  int64
	}{
		{[]string{"base.toml", "overlay.toml"}, 5},
		{[]string{"overlay.toml", "base.toml"}, 2},
	}
	for _, tt := range tests {
		cfg, err := config.LoadFS(fsys, tt.names...)
		if err != nil {
			t.Fatalf("LoadFS(%q) error = %v, want nil", tt.names, err)
		}
		if cfg.World.Scale != tt.want {
			t.Errorf("LoadFS(%q).World.Scale = %d, want %d", tt.names, cfg.World.Scale, tt.want)
		}
	}
}

// lyingFS serves files of unlimited length whose FileInfo says they are empty,
// as an fs.FS may.
type lyingFS struct{}

// Open returns an endless file.
func (lyingFS) Open(name string) (fs.File, error) { return endless{}, nil }

// endless is a file that never ends and reports size 0.
type endless struct{}

func (endless) Stat() (fs.FileInfo, error) { return nil, errors.New("no stat") }
func (endless) Close() error               { return nil }
func (endless) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = '#'
	}
	return len(p), nil
}

// TestLoadFS_Limits verifies that the count of names is checked before any file
// is opened, that each file is read through a bound whatever the file system
// says of its size, and that the merged budget is shared.
//
// Requirement: FR-CFG-022
func TestLoadFS_Limits(t *testing.T) {
	t.Parallel()
	t.Run("too many names are refused before any is opened", func(t *testing.T) {
		t.Parallel()
		names := make([]string, economysim.MaxDocuments+1)
		for i := range names {
			names[i] = "missing.toml"
		}
		_, err := config.LoadFS(fstest.MapFS{}, names...)
		if !errors.Is(err, economysim.ErrTooLarge) || errors.Is(err, fs.ErrNotExist) {
			t.Errorf("LoadFS(%d missing names) error = %v, want ErrTooLarge and not ErrNotExist", len(names), err)
		}
	})
	t.Run("an endless file is cut at the size limit", func(t *testing.T) {
		t.Parallel()
		_, err := config.LoadFS(lyingFS{}, "x.toml")
		if !errors.Is(err, economysim.ErrTooLarge) || !strings.Contains(err.Error(), "MaxDocumentBytes") {
			t.Errorf("LoadFS(endless file) error = %v, want ErrTooLarge naming MaxDocumentBytes", err)
		}
	})
	t.Run("a file one byte over the size limit", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{"big.toml": {Data: padded(economysim.MaxDocumentBytes + 1)}}
		_, err := config.LoadFS(fsys, "big.toml")
		if !errors.Is(err, economysim.ErrTooLarge) {
			t.Errorf("LoadFS(file of MaxDocumentBytes+1) error = %v, want ErrTooLarge", err)
		}
	})
	t.Run("merged budget is shared", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{"a.toml": {Data: padded(economysim.MaxDocumentBytes)}, "b.toml": {Data: []byte("\n")}}
		names := []string{"a.toml", "a.toml", "a.toml", "a.toml"}
		if _, err := config.LoadFS(fsys, names...); !notTooLarge(err) {
			t.Errorf("LoadFS(four files of MaxDocumentBytes) error = %v, want no ErrTooLarge", err)
		}
		_, err := config.LoadFS(fsys, append(names, "b.toml")...)
		if !errors.Is(err, economysim.ErrTooLarge) || !strings.Contains(err.Error(), "MaxMergedBytes") {
			t.Errorf("LoadFS(over the merged budget) error = %v, want ErrTooLarge naming MaxMergedBytes", err)
		}
	})
	t.Run("an unreadable name wraps the fs error and quotes the name", func(t *testing.T) {
		t.Parallel()
		name := "missing-\x01-" + strings.Repeat("n", 100) + ".toml"
		_, err := config.LoadFS(fstest.MapFS{}, name)
		if !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("LoadFS(missing) error = %v, want errors.Is fs.ErrNotExist", err)
		}
		if strings.Contains(err.Error(), "\x01") {
			t.Errorf("LoadFS(missing) error text holds a control character: %q", err.Error())
		}
		if want := strings.Repeat("n", 100)[:64-len("missing-\x01-")]; !strings.Contains(err.Error(), `"...`) || strings.Contains(err.Error(), want+"n") {
			// 64 bytes kept: "missing-\x01-" and 52 "n"; the cut marker follows the quote.
			t.Errorf("LoadFS(long name) error = %q, want the name cut at 64 bytes with a trailing ...", err.Error())
		}
	})
	t.Run("a multibyte name is cut at a rune boundary", func(t *testing.T) {
		t.Parallel()
		// 3-byte runes: the 64-byte cut falls inside the 22nd rune, so it backs
		// up to 21 whole runes (63 bytes).
		_, err := config.LoadFS(fstest.MapFS{}, strings.Repeat("€", 100)+".toml")
		if !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("LoadFS(missing multibyte name) error = %v, want errors.Is fs.ErrNotExist", err)
		}
		if want := strings.Repeat("€", 21) + `"...`; !strings.Contains(err.Error(), want) || !utf8.ValidString(err.Error()) {
			t.Errorf("LoadFS(multibyte name) error = %q, want valid UTF-8 holding %q", err.Error(), want)
		}
	})
	t.Run("a document error names the document", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{"a.toml": {Data: []byte(minimal)}, "b.toml": {Data: []byte("[x\n")}}
		_, err := config.LoadFS(fsys, "a.toml", "b.toml")
		if !errors.Is(err, economysim.ErrSyntax) || !strings.Contains(err.Error(), "document 2") {
			t.Errorf("LoadFS(second file malformed) error = %v, want ErrSyntax naming document 2", err)
		}
	})
}

// countingFS serves the files of its MapFS, and for the name "endless.toml" a
// file that never ends and counts the bytes it hands out.
type countingFS struct {
	fstest.MapFS
	read *int
}

// Open returns the counting endless file for "endless.toml".
func (c countingFS) Open(name string) (fs.File, error) {
	if name == "endless.toml" {
		return countingFile{read: c.read}, nil
	}
	return c.MapFS.Open(name)
}

// countingFile is an endless file that adds each read to *read.
type countingFile struct {
	endless
	read *int
}

func (f countingFile) Read(p []byte) (int, error) {
	n, err := f.endless.Read(p)
	*f.read += n
	return n, err
}

// TestLoadFS_ReadIsBoundedByTheTighterLimit verifies that LoadFS stops reading
// an endless file at the smaller of MaxDocumentBytes and what is left of
// MaxMergedBytes, plus the one byte that proves the limit was crossed.
//
// Guard: replacing min(MaxDocumentBytes, budget) with either operand makes it
// read up to MaxMergedBytes, or past the merged budget, from an untrusted file.
func TestLoadFS_ReadIsBoundedByTheTighterLimit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		files fstest.MapFS
		names []string
		bound int // the most bytes the endless file may hand out
		limit string
	}{
		{"document limit", fstest.MapFS{}, []string{"endless.toml"},
			economysim.MaxDocumentBytes + 1, "MaxDocumentBytes"},
		{"merged budget", fstest.MapFS{
			"full.toml":  {Data: padded(economysim.MaxDocumentBytes)},
			"small.toml": {Data: padded(1 << 20)},
		}, []string{"full.toml", "full.toml", "full.toml", "small.toml", "endless.toml"},
			economysim.MaxMergedBytes - 3*economysim.MaxDocumentBytes - (1 << 20) + 1, "MaxMergedBytes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			read := new(int)
			_, err := config.LoadFS(countingFS{MapFS: tt.files, read: read}, tt.names...)
			if !errors.Is(err, economysim.ErrTooLarge) || !strings.Contains(err.Error(), tt.limit) {
				t.Fatalf("LoadFS(endless file) error = %v, want ErrTooLarge naming %s", err, tt.limit)
			}
			if *read != tt.bound {
				t.Errorf("LoadFS(endless file) read %d bytes, want exactly %d", *read, tt.bound)
			}
		})
	}
}

// TestLoadFS_ReadErrorIsReturned verifies that an error of the file's Read is
// returned wrapped, not swallowed.
//
// Requirement: FR-CFG-022
func TestLoadFS_ReadErrorIsReturned(t *testing.T) {
	t.Parallel()
	_, err := config.LoadFS(failingFS{}, "x.toml")
	if !errors.Is(err, errRead) {
		t.Errorf("LoadFS(failing file) error = %v, want errors.Is errRead", err)
	}
}

// errRead is the error of failingFS.
var errRead = errors.New("disk on fire")

// failingFS serves a file whose Read fails.
type failingFS struct{}

// Open returns a file whose Read fails.
func (failingFS) Open(name string) (fs.File, error) { return failing{}, nil }

// failing is a file whose Read fails.
type failing struct{ endless }

func (failing) Read([]byte) (int, error) { return 0, errRead }

// TestParse_LongHeaderWithManyKeysIsLinear verifies that the guard pass does
// not walk a table header again for each key of the table: a 100 KB header
// followed by 2 000 key-value lines allocates a small multiple of the document,
// where a pass that rebuilt the header path per line would allocate hundreds of
// megabytes. It counts bytes allocated, not time, and runs serially so that no
// other test adds to the count.
//
// Requirement: NFR-CFG-002
func TestParse_LongHeaderWithManyKeysIsLinear(t *testing.T) {
	doc := minimal + "[role.\"" + strings.Repeat("h", 100_000) + "\"]\n" + strings.Repeat("a=1\n", 2000)
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	_, err := parseStrings(doc)
	runtime.ReadMemStats(&after)
	if err == nil {
		t.Fatalf("Parse(long header, 2000 repeated keys) error = nil, want a rejection")
	}
	// A rebuilt header path is at least 100 KB a line: 2 000 lines is over
	// 200 MB. A linear pass is a few copies of the 110 KB document.
	got, limit := after.TotalAlloc-before.TotalAlloc, uint64(32*len(doc))
	if got > limit {
		t.Errorf("Parse(%d-byte document) allocated %d bytes, want at most %d (linear in the document)", len(doc), got, limit)
	}
}
