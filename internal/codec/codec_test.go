package codec_test

import (
	"bytes"
	"errors"
	"go/parser"
	"go/token"
	"math"
	"math/rand/v2"
	"slices"
	"strconv"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/codec"
)

// TestUvarint_Vectors verifies that U writes the minimal LEB128 form and
// reads it back, on the values spec-checkpoint.md 5.2 gives and on the
// largest value, whose encoding is nine bytes of 0xff and a final 0x01.
//
// Requirement: FR-CHK-010
func TestUvarint_Vectors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		v    uint64
		want []byte
	}{
		{"zero", 0, []byte{0x00}},
		{"one", 1, []byte{0x01}},
		{"one byte maximum", 127, []byte{0x7f}},
		{"128", 128, []byte{0x80, 0x01}},
		{"300", 300, []byte{0xac, 0x02}},
		{"two bytes maximum", 16383, []byte{0xff, 0x7f}},
		{"three bytes", 16384, []byte{0x80, 0x80, 0x01}},
		{"2^63", 1 << 63, []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01}},
		{"maximum", math.MaxUint64, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := codec.NewWriter(nil)
			w.U(tt.v)
			if got := w.Out(); !bytes.Equal(got, tt.want) {
				t.Errorf("U(%d) wrote % x, want % x", tt.v, got, tt.want)
			}
			if got := codec.USize(tt.v); got != len(tt.want) {
				t.Errorf("USize(%d) = %d, want %d", tt.v, got, len(tt.want))
			}
			r := codec.NewReader(tt.want)
			if got := r.U(); got != tt.v || r.Done() != nil {
				t.Errorf("U() of % x = %d, %v, want %d and no error", tt.want, got, r.Done(), tt.v)
			}
		})
	}
}

// TestUvarint_RejectsNonMinimalOverflowAndTruncated verifies that the reader
// refuses the spellings that the standard library's Uvarint accepts or
// reports differently: a last byte of 0 after a continuation (two spellings
// of one value), a tenth byte above 1 and an eleventh byte (both overflow),
// and input that ends inside a varint.
//
// Requirement: FR-CHK-010
func TestUvarint_RejectsNonMinimalOverflowAndTruncated(t *testing.T) {
	t.Parallel()
	ones := bytes.Repeat([]byte{0xff}, 9)
	tests := []struct {
		name string
		in   []byte
	}{
		{"zero in two bytes", []byte{0x80, 0x00}},
		{"one in three bytes", []byte{0x81, 0x80, 0x00}},
		{"non-minimal at the tenth byte", append(slices.Clone(ones), 0x00)},
		{"tenth byte above 1", append(slices.Clone(ones), 0x02)},
		{"eleven bytes", append(slices.Clone(ones), 0x81, 0x01)},
		{"empty", nil},
		{"cut after a continuation", []byte{0x80}},
		{"cut at nine bytes", slices.Clone(ones)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := codec.NewReader(tt.in)
			if got := r.U(); got != 0 {
				t.Errorf("U() of % x = %d, want 0 with the error", tt.in, got)
			}
			if err := r.Err(); !errors.Is(err, codec.ErrMalformed) {
				t.Errorf("U() of % x: Err() = %v, want ErrMalformed", tt.in, err)
			}
		})
	}
}

// TestZigzag_RoundTripAndVectors verifies the signed encoding on the values
// of spec-checkpoint.md 5.2 and on both ends of int64, including MinInt64,
// whose zigzag is the largest unsigned value.
//
// Requirement: FR-CHK-011
func TestZigzag_RoundTripAndVectors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		v    int64
		want []byte
	}{
		{"0", 0, []byte{0x00}},
		{"-1", -1, []byte{0x01}},
		{"1", 1, []byte{0x02}},
		{"-2", -2, []byte{0x03}},
		{"10000", 10000, []byte{0xa0, 0x9c, 0x01}},
		{"MaxInt64", math.MaxInt64, []byte{0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}},
		{"MinInt64", math.MinInt64, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := codec.NewWriter(nil)
			w.I(tt.v)
			if got := w.Out(); !bytes.Equal(got, tt.want) {
				t.Errorf("I(%d) wrote % x, want % x", tt.v, got, tt.want)
			}
			r := codec.NewReader(tt.want)
			if got := r.I(); got != tt.v || r.Done() != nil {
				t.Errorf("I() of % x = %d, %v, want %d and no error", tt.want, got, r.Done(), tt.v)
			}
		})
	}
}

// TestBool_RejectsOtherBytes verifies that B reads 0 and 1 and refuses every
// other byte and the end of input.
//
// Requirement: FR-CHK-012
func TestBool_RejectsOtherBytes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      []byte
		want    bool
		wantErr bool
	}{
		{"zero", []byte{0}, false, false},
		{"one", []byte{1}, true, false},
		{"two", []byte{2}, false, true},
		{"255", []byte{255}, false, true},
		{"empty", nil, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := codec.NewReader(tt.in)
			got := r.B()
			if got != tt.want || (r.Err() != nil) != tt.wantErr {
				t.Errorf("B() of % x = %v, %v, want %v and error %v", tt.in, got, r.Err(), tt.want, tt.wantErr)
			}
		})
	}
}

// TestIndex_BoundAndStrictOrder verifies that an index must be below its
// bound and that a key sequence must be strictly ascending, with the first
// key free.
//
// Requirement: FR-CHK-012, FR-CHK-014
func TestIndex_BoundAndStrictOrder(t *testing.T) {
	t.Parallel()
	enc := func(vs ...uint64) []byte {
		w := codec.NewWriter(nil)
		for _, v := range vs {
			w.U(v)
		}
		return w.Out()
	}
	t.Run("index below the bound", func(t *testing.T) {
		t.Parallel()
		r := codec.NewReader(enc(0, 4))
		if a, b := r.Index(5), r.Index(5); a != 0 || b != 4 || r.Err() != nil {
			t.Errorf("Index(5) twice = %d, %d, %v, want 0, 4 and no error", a, b, r.Err())
		}
	})
	t.Run("index at the bound", func(t *testing.T) {
		t.Parallel()
		r := codec.NewReader(enc(5))
		if got := r.Index(5); got != 0 || !errors.Is(r.Err(), codec.ErrMalformed) {
			t.Errorf("Index(5) of 5 = %d, %v, want 0 and ErrMalformed", got, r.Err())
		}
	})
	t.Run("index with a bound of zero", func(t *testing.T) {
		t.Parallel()
		r := codec.NewReader(enc(0))
		if got := r.Index(0); got != 0 || r.Err() == nil {
			t.Errorf("Index(0) of 0 = %d, %v, want 0 and an error: nothing is below 0", got, r.Err())
		}
	})
	tests := []struct {
		name    string
		keys    []uint64
		wantErr bool
	}{
		{"ascending", []uint64{0, 1, 7, 100}, false},
		{"first key is free", []uint64{0}, false},
		{"duplicate", []uint64{3, 3}, true},
		{"descent", []uint64{3, 9, 4}, true},
	}
	for _, tt := range tests {
		t.Run("strict "+tt.name, func(t *testing.T) {
			t.Parallel()
			r := codec.NewReader(enc(tt.keys...))
			var prev uint64
			for i := range tt.keys {
				prev = r.Strict(prev, i > 0)
			}
			if (r.Err() != nil) != tt.wantErr {
				t.Errorf("Strict over %v: Err() = %v, want error %v", tt.keys, r.Err(), tt.wantErr)
			}
		})
	}
}

// TestCount_AtMostBytesRemaining verifies R8: a stored count and an implied
// count are at most the bytes remaining divided by the least bytes of an
// element, so no count asks for more than the input holds.
//
// Requirement: FR-CHK-013
func TestCount_AtMostBytesRemaining(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		count   uint64
		tail    int
		minElem int
		want    int
	}{
		{"fits exactly", 3, 3, 1, 3},
		{"zero", 0, 0, 1, 0},
		{"one too many", 4, 3, 1, 0},
		{"two bytes an element, fits", 2, 4, 2, 2},
		{"two bytes an element, one too many", 3, 4, 2, 0},
		{"a minimum of zero is read as one", 3, 3, 0, 3},
		{"huge", 1 << 40, 8, 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := codec.NewWriter(nil)
			w.U(tt.count)
			in := append(w.Out(), make([]byte, tt.tail)...)
			r := codec.NewReader(in)
			if got := r.Count(tt.minElem); got != tt.want {
				t.Errorf("Count(%d) of count %d with %d bytes after = %d, want %d", tt.minElem, tt.count, tt.tail, got, tt.want)
			}
			if (r.Err() != nil) != (tt.want == 0 && tt.count != 0) {
				t.Errorf("Count(%d) of count %d: Err() = %v", tt.minElem, tt.count, r.Err())
			}
		})
	}
	t.Run("implied count", func(t *testing.T) {
		t.Parallel()
		r := codec.NewReader(make([]byte, 6))
		if got := r.Implied(3, 2); got != 3 || r.Err() != nil {
			t.Errorf("Implied(3, 2) with 6 bytes = %d, %v, want 3 and no error", got, r.Err())
		}
		if got := r.Implied(4, 2); got != 0 || !errors.Is(r.Err(), codec.ErrMalformed) {
			t.Errorf("Implied(4, 2) with 6 bytes = %d, %v, want 0 and ErrMalformed", got, r.Err())
		}
	})
	t.Run("implied count is negative", func(t *testing.T) {
		t.Parallel()
		r := codec.NewReader(make([]byte, 6))
		if got := r.Implied(-1, 1); got != 0 || r.Err() == nil {
			t.Errorf("Implied(-1, 1) = %d, %v, want 0 and an error", got, r.Err())
		}
	})
}

// TestBytes_LengthAtMostRemaining verifies R5 and R4: a length-prefixed array
// is at most the bytes remaining, a fixed array needs its length, and a
// slice read aliases the input but cannot be appended into it.
//
// Requirement: FR-CHK-013
func TestBytes_LengthAtMostRemaining(t *testing.T) {
	t.Parallel()
	t.Run("round trip", func(t *testing.T) {
		t.Parallel()
		w := codec.NewWriter(nil)
		w.Bytes([]byte("directed"))
		w.Bytes(nil)
		w.Raw([]byte{9, 8, 7})
		r := codec.NewReader(w.Out())
		a, b, c := r.Bytes(), r.Bytes(), r.Raw(3)
		if string(a) != "directed" || len(b) != 0 || !bytes.Equal(c, []byte{9, 8, 7}) || r.Done() != nil {
			t.Errorf("read %q, %q, % x, %v, want directed, empty, 09 08 07 and no error", a, b, c, r.Done())
		}
		if cap(a) != len(a) {
			t.Errorf("a slice read has capacity %d beyond its length %d", cap(a), len(a))
		}
	})
	tests := []struct {
		name string
		in   []byte
		read func(*codec.Reader) []byte
	}{
		{"length beyond the input", []byte{5, 'a', 'b'}, (*codec.Reader).Bytes},
		{"length is a bad varint", []byte{0x80, 0x00}, (*codec.Reader).Bytes},
		{"length of a huge value", []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f}, (*codec.Reader).Bytes},
		{"raw beyond the input", []byte{1, 2}, func(r *codec.Reader) []byte { return r.Raw(3) }},
		{"raw of a negative length", []byte{1, 2}, func(r *codec.Reader) []byte { return r.Raw(-1) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := codec.NewReader(tt.in)
			if got := tt.read(r); got != nil || !errors.Is(r.Err(), codec.ErrMalformed) {
				t.Errorf("read of % x = % x, %v, want nil and ErrMalformed", tt.in, got, r.Err())
			}
		})
	}
}

// TestReader_StickyErrorAndOffset verifies that the first error is kept with
// the offset at which the failing read began, that every later read returns
// a zero value without moving, and that Failf records a caller's finding at
// the current offset and does not replace an earlier error.
//
// Requirement: FR-CHK-013
func TestReader_StickyErrorAndOffset(t *testing.T) {
	t.Parallel()
	r := codec.NewReader([]byte{0x05, 0x02, 0x80, 0x00, 0x07})
	if got := r.U(); got != 5 || r.Offset() != 1 {
		t.Fatalf("U() = %d at offset %d, want 5 at 1", got, r.Offset())
	}
	if got := r.I(); got != 1 || r.Offset() != 2 {
		t.Fatalf("I() = %d at offset %d, want 1 at 2", got, r.Offset())
	}
	r.U() // the non-minimal varint 80 00 at offset 2
	first := r.Err()
	var ce *codec.Error
	if !errors.As(first, &ce) || ce.Offset != 2 {
		t.Fatalf("Err() = %v, want a *codec.Error at offset 2", first)
	}
	if r.Offset() != 2 {
		t.Errorf("a failed read moved the offset to %d, want 2", r.Offset())
	}
	if r.U() != 0 || r.I() != 0 || r.B() || r.Bytes() != nil || r.Raw(1) != nil || r.Count(1) != 0 || r.Index(9) != 0 || r.Strict(0, false) != 0 {
		t.Errorf("a read after the error returned a non-zero value")
	}
	r.Failf("later finding %d", 1)
	if r.Err() != first {
		t.Errorf("Err() = %v after Failf, want the first error %v", r.Err(), first)
	}
	if r.Done() != first {
		t.Errorf("Done() = %v, want the first error", r.Done())
	}

	t.Run("failf on a healthy reader", func(t *testing.T) {
		t.Parallel()
		r := codec.NewReader([]byte{1, 2})
		r.U()
		r.Failf("balance %d is above the bound", 9)
		var got *codec.Error
		if !errors.Is(r.Err(), codec.ErrMalformed) || !errors.As(r.Err(), &got) || got.Offset != 1 || got.Msg != "balance 9 is above the bound" {
			t.Errorf("Err() = %v, want malformed at offset 1 with the message", r.Err())
		}
	})
}

// TestReader_DoneRejectsTrailingBytes verifies R11: a reader that has read
// everything is done, and bytes left over are malformed.
//
// Requirement: FR-CHK-014
func TestReader_DoneRejectsTrailingBytes(t *testing.T) {
	t.Parallel()
	r := codec.NewReader([]byte{1, 0})
	r.U()
	if err := r.Done(); !errors.Is(err, codec.ErrMalformed) {
		t.Errorf("Done() with one byte left = %v, want ErrMalformed", err)
	}
	r = codec.NewReader([]byte{1})
	r.U()
	if err := r.Done(); err != nil {
		t.Errorf("Done() with nothing left = %v, want nil", err)
	}
	if r.Remaining() != 0 {
		t.Errorf("Remaining() = %d, want 0", r.Remaining())
	}
}

// TestWriterReader_RoundTrip verifies that 1000 values of every primitive
// written in a mixed order are read back equal and that the bytes written
// are exactly the bytes read, so an accepted input re-encodes to itself.
//
// Requirement: FR-CHK-015
func TestWriterReader_RoundTrip(t *testing.T) {
	t.Parallel()
	// The test's own source, with a fixed seed: the library draws nothing.
	rng := rand.New(rand.NewPCG(1, 2))
	type item struct {
		kind int
		u    uint64
		i    int64
		b    bool
		p    []byte
	}
	items := make([]item, 1000)
	w := codec.NewWriter([]byte{0xee}) // appends after existing bytes
	for n := range items {
		it := item{kind: rng.IntN(5)}
		switch it.kind {
		case 0:
			it.u = rng.Uint64() >> rng.UintN(64)
			w.U(it.u)
		case 1:
			it.i = int64(rng.Uint64()) >> rng.UintN(64)
			w.I(it.i)
		case 2:
			it.b = rng.IntN(2) == 1
			w.B(it.b)
		case 3:
			it.p = make([]byte, rng.IntN(20))
			for k := range it.p {
				it.p[k] = byte(rng.Uint32())
			}
			w.Bytes(it.p)
		case 4:
			it.u = uint64(rng.IntN(300))
			w.Count(int(it.u))
		}
		items[n] = it
	}
	out := w.Out()
	if out[0] != 0xee || w.Len() != len(out) {
		t.Fatalf("the writer did not append after dst: first byte %x, Len() %d of %d", out[0], w.Len(), len(out))
	}
	r := codec.NewReader(out[1:])
	for n, it := range items {
		switch it.kind {
		case 0:
			if got := r.U(); got != it.u {
				t.Fatalf("item %d: U() = %d, want %d", n, got, it.u)
			}
		case 1:
			if got := r.I(); got != it.i {
				t.Fatalf("item %d: I() = %d, want %d", n, got, it.i)
			}
		case 2:
			if got := r.B(); got != it.b {
				t.Fatalf("item %d: B() = %v, want %v", n, got, it.b)
			}
		case 3:
			if got := r.Bytes(); !bytes.Equal(got, it.p) {
				t.Fatalf("item %d: Bytes() = % x, want % x", n, got, it.p)
			}
		case 4:
			// The count is read with the fewest bytes per element, 0
			// stored bytes follow, so a count above the tail would fail.
			if got := r.U(); got != it.u {
				t.Fatalf("item %d: count read as U() = %d, want %d", n, got, it.u)
			}
		}
	}
	if err := r.Done(); err != nil {
		t.Errorf("Done() after reading every item = %v, want nil", err)
	}
}

// TestAlloc_CountBoundsTheSlice verifies that a count of 2^40 in a short
// input is refused before any slice is sized from it: the result is 0, so
// make([]T, 0, n) holds nothing, and the refusal allocates a constant.
//
// Requirement: FR-CHK-016
func TestAlloc_CountBoundsTheSlice(t *testing.T) {
	// Not parallel: AllocsPerRun measures the whole process.
	w := codec.NewWriter(nil)
	w.U(1 << 40)
	in := append(w.Out(), 1, 2, 3)
	got := 0
	allocs := testing.AllocsPerRun(100, func() {
		r := codec.NewReader(in)
		n := r.Count(1)
		s := make([]uint64, 0, n)
		got = cap(s)
	})
	if got != 0 {
		t.Errorf("Count of 2^40 in a 9-byte input sized a slice of capacity %d, want 0", got)
	}
	// The reader, the error and the zero-length slice: a constant, not a
	// function of the count.
	if allocs > 4 {
		t.Errorf("refusing a count of 2^40 allocated %v times, want a small constant", allocs)
	}
}

// TestImports_StdlibOnly verifies that the package imports errors and fmt and
// nothing else: no other package of the standard library, none of the
// module (no float, no time, no randomness, no goroutine support).
//
// Requirement: NFR-CHK-005
func TestImports_StdlibOnly(t *testing.T) {
	t.Parallel()
	allowed := []string{"errors", "fmt"}
	for _, file := range []string{"codec.go", "reader.go", "doc.go"} {
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, imp := range f.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("%s: import %s: %v", file, imp.Path.Value, err)
			}
			if !slices.Contains(allowed, path) {
				t.Errorf("%s imports %q, want only %v", file, path, allowed)
			}
		}
	}
}

// FuzzCodec drives the reader with a program of reads over arbitrary bytes,
// and asserts that nothing panics and that the bytes of every accepted prefix
// are re-encoded to the same bytes by the writer (the one-spelling property,
// FR-CHK-005).
//
// Requirement: NFR-CHK-003
func FuzzCodec(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3, 4, 5, 6}, []byte{0x00, 0x02, 0x01, 0x03, 'a', 'b', 'c', 0x02, 0x01, 0x05})
	f.Add([]byte{0, 0, 0}, []byte{0x80, 0x00})
	f.Add([]byte{3, 3}, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01})
	f.Fuzz(func(t *testing.T, program, data []byte) {
		r := codec.NewReader(data)
		w := codec.NewWriter(nil)
		var prev uint64
		for i, op := range program {
			switch op % 7 {
			case 0:
				w.U(r.U())
			case 1:
				w.I(r.I())
			case 2:
				w.B(r.B())
			case 3:
				w.Bytes(r.Bytes())
			case 4:
				w.Count(r.Count(1))
			case 5:
				w.U(r.Index(1 << 20))
			case 6:
				prev = r.Strict(prev, i > 0)
				w.U(prev)
			}
			if r.Err() != nil {
				return
			}
			if got, want := w.Out(), data[:r.Offset()]; !bytes.Equal(got, want) {
				t.Fatalf("program % x over % x: re-encoded % x, want % x", program, data, got, want)
			}
		}
	})
}

// TestError_MessageAndNegativeCount verifies the text of a reader error, that
// Implied does nothing after an error, and that a writer given a negative
// count stops with a message that says a programmer invariant was violated.
//
// Requirement: FR-CHK-013
func TestError_MessageAndNegativeCount(t *testing.T) {
	t.Parallel()
	r := codec.NewReader([]byte{0x80, 0x00})
	r.U()
	if got, want := r.Err().Error(), "codec: malformed input at byte 0: non-minimal varint"; got != want {
		t.Errorf("Err().Error() = %q, want %q", got, want)
	}
	if got := r.Implied(1, 1); got != 0 {
		t.Errorf("Implied after an error = %d, want 0", got)
	}
	defer func() {
		if recover() == nil {
			t.Errorf("Count(-1) did not panic")
		}
	}()
	codec.NewWriter(nil).Count(-1)
}

// TestReader_ReadsAfterAFindingReturnZero verifies that an error recorded by
// Failf, which leaves the offset on readable bytes, stops every later read: it
// returns the zero value and consumes nothing, so a decoder that checks Err
// once at the end of a record never acts on a value read after the finding.
//
// Requirement: FR-CHK-013
func TestReader_ReadsAfterAFindingReturnZero(t *testing.T) {
	t.Parallel()
	reads := []struct {
		name string
		read func(*codec.Reader) bool // reports whether a zero value came back
	}{
		{"U", func(r *codec.Reader) bool { return r.U() == 0 }},
		{"I", func(r *codec.Reader) bool { return r.I() == 0 }},
		{"B", func(r *codec.Reader) bool { return !r.B() }},
		{"Raw", func(r *codec.Reader) bool { return r.Raw(1) == nil }},
		{"Bytes", func(r *codec.Reader) bool { return r.Bytes() == nil }},
		{"Count", func(r *codec.Reader) bool { return r.Count(1) == 0 }},
		{"Implied", func(r *codec.Reader) bool { return r.Implied(1, 1) == 0 }},
		{"Index", func(r *codec.Reader) bool { return r.Index(9) == 0 }},
		{"Strict", func(r *codec.Reader) bool { return r.Strict(0, false) == 0 }},
	}
	for _, tt := range reads {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := codec.NewReader([]byte{1, 1, 1, 1})
			r.Failf("a finding")
			if !tt.read(r) || r.Offset() != 0 {
				t.Errorf("%s after a finding returned a value or moved to offset %d", tt.name, r.Offset())
			}
		})
	}
}

// TestReader_ExactFitsAreAccepted verifies the edges of the bounds: a length,
// a count, an implied count and an index that use exactly what the input or
// the bound allows are accepted, and a failed read returns zero.
//
// Requirement: FR-CHK-012, FR-CHK-013
func TestReader_ExactFitsAreAccepted(t *testing.T) {
	t.Parallel()
	t.Run("bytes that end the input", func(t *testing.T) {
		t.Parallel()
		r := codec.NewReader([]byte{2, 'o', 'k'})
		if got := r.Bytes(); string(got) != "ok" || r.Done() != nil {
			t.Errorf("Bytes() = %q, %v, want ok and no error", got, r.Done())
		}
	})
	t.Run("count below what fits", func(t *testing.T) {
		t.Parallel()
		r := codec.NewReader([]byte{2, 0, 0, 0, 0})
		if got := r.Count(1); got != 2 || r.Err() != nil {
			t.Errorf("Count(1) of 2 with 4 bytes after = %d, %v, want 2 and no error", got, r.Err())
		}
	})
	t.Run("implied counts of zero and of all", func(t *testing.T) {
		t.Parallel()
		r := codec.NewReader(make([]byte, 4))
		if a, b := r.Implied(0, 1), r.Implied(4, 1); a != 0 || b != 4 || r.Err() != nil {
			t.Errorf("Implied(0, 1), Implied(4, 1) = %d, %d, %v, want 0, 4 and no error", a, b, r.Err())
		}
	})
	t.Run("index above the bound", func(t *testing.T) {
		t.Parallel()
		r := codec.NewReader([]byte{9})
		if got := r.Index(5); got != 0 || r.Err() == nil {
			t.Errorf("Index(5) of 9 = %d, %v, want 0 and an error", got, r.Err())
		}
	})
	t.Run("strict that fails returns zero", func(t *testing.T) {
		t.Parallel()
		r := codec.NewReader([]byte{3})
		if got := r.Strict(7, true); got != 0 || r.Err() == nil {
			t.Errorf("Strict(7, true) of 3 = %d, %v, want 0 and an error", got, r.Err())
		}
	})
	t.Run("strict equal to the previous key", func(t *testing.T) {
		t.Parallel()
		r := codec.NewReader([]byte{7})
		if got := r.Strict(7, true); got != 0 || r.Err() == nil {
			t.Errorf("Strict(7, true) of 7 = %d, %v, want 0 and an error", got, r.Err())
		}
	})
	t.Run("strict that may start at zero", func(t *testing.T) {
		t.Parallel()
		r := codec.NewReader([]byte{0})
		if got := r.Strict(0, false); got != 0 || r.Err() != nil {
			t.Errorf("Strict(0, false) of 0 = %d, %v, want 0 and no error", got, r.Err())
		}
	})
}

// TestReader_BytesLengthBeyondInputFailsAtTheLength verifies that a length
// above the bytes remaining is reported by Bytes itself, at the offset where
// the length began, and not by the fixed-size read after it, which would
// report the offset after the length and, where int is 32 bits, would accept
// a length that wraps (2^32+1 reads as 1).
//
// Requirement: FR-CHK-012
func TestReader_BytesLengthBeyondInputFailsAtTheLength(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   []byte
		off  int
	}{
		{"one past the bytes remaining", []byte{4, 'a', 'b', 'c'}, 0},
		{"wraps a 32-bit int", []byte{0x81, 0x80, 0x80, 0x80, 0x10, 'x'}, 0},
		{"above the largest int", []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := codec.NewReader(tt.in)
			got := r.Bytes()
			var ce *codec.Error
			if got != nil || !errors.As(r.Err(), &ce) || ce.Offset != tt.off || ce.Msg != "length exceeds the bytes remaining" {
				t.Errorf("Bytes() = %v, %v, want nil and %q at offset %d", got, r.Err(), "length exceeds the bytes remaining", tt.off)
			}
		})
	}
}

// TestReader_BoundedEdges verifies that Bounded accepts a value equal to its
// maximum, refuses the next one naming what was read at the offset where the read
// began, and returns 0 for it.
//
// Requirement: FR-CHK-013
func TestReader_BoundedEdges(t *testing.T) {
	t.Parallel()
	r := codec.NewReader([]byte{5, 6})
	if got := r.Bounded("run", 5); got != 5 || r.Err() != nil {
		t.Fatalf("Bounded(5 of 5) = %d, %v, want 5 and no error", got, r.Err())
	}
	if got := r.Bounded("run", 5); got != 0 || r.Err() == nil || r.Err().Error() != "codec: malformed input at byte 1: run 6 is above 5" {
		t.Errorf("Bounded(6 of 5) = %d, %v, want 0 and the error naming run 6 above 5", got, r.Err())
	}
}

// TestReader_BoundedNegativeMaxRefusesEverything verifies that a negative maximum
// fails closed: it is not read as a huge unsigned limit that accepts any value.
//
// Requirement: FR-CHK-013
func TestReader_BoundedNegativeMaxRefusesEverything(t *testing.T) {
	t.Parallel()
	r := codec.NewReader([]byte{0})
	if got := r.Bounded("run", -1); got != 0 || r.Err() == nil || r.Err().Error() != "codec: malformed input at byte 0: run has a negative bound -1" {
		t.Errorf("Bounded(0 of -1) = %d, %v, want 0 and an error", got, r.Err())
	}
}
