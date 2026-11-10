package calibration

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// libraryMaxFactors is MaxFactors of EXP 13.2. internal/ never imports
// experiment/ (architecture.md 2.9), so the test names the number.
const libraryMaxFactors = 16

// libraryProblems reads the records of a library tree and returns what is wrong
// with them: every testdata/calibration/<pack>/*.txt is parsed, checked one by one
// and, per pack in file-name order, as a sequence; every
// testdata/golden/stylised/*.stability.txt is parsed, checked, and must hold only
// the lines of a stability record (8.4). A tree with none returns nothing.
func libraryProblems(root string, k Known) ([]string, error) {
	var out []string
	dirs, err := filepath.Glob(filepath.Join(root, "testdata", "calibration", "*"))
	if err != nil {
		return nil, err
	}
	for _, dir := range dirs {
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			continue
		}
		ps, err := packProblems(dir, k)
		if err != nil {
			return nil, err
		}
		out = append(out, ps...)
	}
	stab, _ := filepath.Glob(filepath.Join(root, "testdata", "golden", "stylised", "*.stability.txt"))
	slices.Sort(stab)
	for _, f := range stab {
		ps, err := stabilityProblems(f, k)
		if err != nil {
			return nil, err
		}
		out = append(out, ps...)
	}
	return out, nil
}

// packProblems checks the records of one pack directory, one by one and as a
// sequence in file-name order.
func packProblems(dir string, k Known) ([]string, error) {
	var out []string
	files, _ := filepath.Glob(filepath.Join(dir, "*.txt"))
	slices.Sort(files)
	var rs []Record
	var names []string
	for _, f := range files {
		r, ps, err := readRecord(f, k)
		if err != nil {
			return nil, err
		}
		out = append(out, ps...)
		if r.Pack == "" { // it did not parse: the problem is already listed
			continue
		}
		if r.Pack != filepath.Base(dir) {
			out = append(out, f+": pack "+r.Pack+" is not the directory "+filepath.Base(dir))
		}
		rs, names = append(rs, r), append(names, f)
	}
	for _, p := range CheckSequence(rs, k, libraryMaxFactors) {
		out = append(out, names[p.Record]+": "+p.Rule+": "+p.Detail)
	}
	return out, nil
}

// stabilityProblems checks one stability record: the record checks, only the lines
// of a stability record, and the pack its file name gives.
func stabilityProblems(f string, k Known) ([]string, error) {
	r, out, err := readRecord(f, k)
	if err != nil {
		return nil, err
	}
	if !stabilityOnly(r) {
		out = append(out, f+": a stability record holds only the lines calibration, pack, commit, config, document, stability, claim and decision")
	}
	if want := strings.TrimSuffix(filepath.Base(f), ".stability.txt"); r.Pack != want {
		out = append(out, f+": pack "+r.Pack+" is not the file's pack "+want)
	}
	return out, nil
}

// readRecord parses and checks one record file. A parse failure is a problem; an
// unreadable file is an error.
func readRecord(path string, k Known) (Record, []string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Record{}, nil, err
	}
	r, err := Parse(b)
	if err != nil {
		return Record{}, []string{path + ": " + err.Error()}, nil
	}
	var out []string
	for _, p := range CheckRecord(r, k) {
		out = append(out, path+": line "+itoa(p.Line)+": "+p.Rule+": "+p.Detail)
	}
	return r, out, nil
}

// stabilityOnly reports whether a record holds only the lines of a stability
// record: its header, documents, stability and claim lines and the decision.
func stabilityOnly(r Record) bool {
	keep := Record{Pack: r.Pack, Lineage: r.Lineage, Stage: r.Stage, Attempt: r.Attempt, Block: r.Block, Commit: r.Commit, Parent: r.Parent,
		Base: r.Base, Chosen: r.Chosen, Documents: r.Documents, Stability: r.Stability, Claims: r.Claims, Decision: r.Decision}
	return reflect.DeepEqual(r, keep)
}

// TestLibraryRecords_Checked verifies that every record of the library under
// testdata/calibration/<pack>/ and every stability record under
// testdata/golden/stylised/ parses and passes the record checks, and the sequence
// checks per pack. None exists yet, so that set passes; the check is proved able to
// fail on a temporary tree with a valid record, a pack in the wrong directory, a
// record that does not parse, a sequence out of order and a stability record with
// a line it may not hold.
//
// Requirement: FR-CLB-040, FR-CLB-041, FR-CLB-042, FR-CLB-043, FR-CLB-053, FR-CLB-054
func TestLibraryRecords_Checked(t *testing.T) {
	t.Parallel()
	s := loadSpecs(t)
	k := registryKnown(s, parseRegistry(s.clb))
	if ps, err := libraryProblems("../..", k); err != nil || len(ps) != 0 {
		t.Errorf("libraryProblems(the repository) = %v, %v, want none", ps, err)
	}

	root := t.TempDir()
	put := func(rel string, r Record) {
		t.Helper()
		b, err := Render(r)
		if err != nil {
			t.Fatal(err)
		}
		putFile(t, filepath.Join(root, rel), b)
	}
	c1 := fill(stageOnly("C1", 1, "stop"))
	c2 := fill(stageOnly("C2", 1, "stop"))
	put("testdata/calibration/one/c1-1.txt", c1)
	put("testdata/calibration/one/c2-1.txt", c2)
	stability := fill(stageOnly("C4a", 1, "review"))
	stability.Stability = []Stability{stab("default", "gate", 1, 10, true, "UNDETERMINED", "none")}
	stability.Claims = []Claim{{"K-BEL12", "one", "undetermined"}}
	stability.Stability[0].Knob = "K-BEL12"
	put("testdata/golden/stylised/one.stability.txt", stability)
	kk := checkKnown()
	if ps, err := libraryProblems(root, kk); err != nil || len(ps) != 0 {
		t.Fatalf("libraryProblems(valid tree) = %v, %v, want none", ps, err)
	}

	put("testdata/calibration/one/c2-2.txt", fill(stageOnly("C2", 2, "stop"))) // a stage run again after its stop
	put("testdata/calibration/three/c1-1.txt", c1)                             // pack one in the directory of pack three
	putFile(t, filepath.Join(root, "testdata/calibration/one/c3-1.txt"), []byte("flavour\n"))
	odd := stability
	odd.Facts = []Fact{{ID: "P02", Class: 'B', Target: "report", Result: "report"}}
	put("testdata/golden/stylised/two.stability.txt", odd)
	ps, err := libraryProblems(root, kk)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"c1-1.txt: pack one is not the directory three", "c3-1.txt: calibration: invalid record", "stage_order", "holds only the lines", "two.stability.txt: pack one is not the file's pack two"} {
		if !slices.ContainsFunc(ps, func(p string) bool { return strings.Contains(p, want) }) {
			t.Errorf("libraryProblems(bad tree) = %v, want a problem containing %q", ps, want)
		}
	}
}

// fill gives a stage record the header fields Render needs.
func fill(r Record) Record {
	r.Commit, r.Base, r.Chosen = "4f2a9c1", "91d3a7c2e05b4a18", "same"
	return r
}

func putFile(t *testing.T, path string, b []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}
