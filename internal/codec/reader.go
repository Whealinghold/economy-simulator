package codec

import "fmt"

// Reader decodes the canonical encodings of Writer from a byte slice. It
// keeps the first error and returns zero values after it, so a decoder reads
// straight down a record and checks Err once at the end. A Reader never
// panics on any input and allocates nothing in proportion to a number it reads
// (R8): the caller sizes a slice only from the result of Count or Implied,
// which are at most the bytes that remain.
type Reader struct {
	b   []byte
	off int
	err error
}

// NewReader returns a Reader over b. It does not copy b, and the slices Raw
// and Bytes return alias it.
func NewReader(b []byte) *Reader { return &Reader{b: b} }

// Err returns the first error, or nil.
func (r *Reader) Err() error { return r.err }

// Offset returns the number of bytes consumed.
func (r *Reader) Offset() int { return r.off }

// Remaining returns the number of bytes not yet consumed.
func (r *Reader) Remaining() int { return len(r.b) - r.off }

// fail records the first error, at the offset where the failing read began.
func (r *Reader) fail(at int, msg string) {
	if r.err == nil {
		r.err = &Error{Offset: at, Msg: msg}
	}
}

// Failf records a violation the caller found in a value it read, such as a
// value out of its range (R13), as a malformed input at the current offset.
// The first error wins.
func (r *Reader) Failf(format string, args ...any) {
	r.fail(r.off, fmt.Sprintf(format, args...))
}

// U reads an unsigned integer as a minimal LEB128 varint of at most ten bytes,
// the tenth at most 1 (R1). A non-minimal, overflowing or truncated varint is
// malformed. The standard library's Uvarint accepts a non-minimal encoding,
// so it is not used (spec-checkpoint.md 5.2.1).
func (r *Reader) U() uint64 {
	if r.err != nil {
		return 0
	}
	var v uint64
	for i := 0; ; i++ {
		if r.off+i >= len(r.b) {
			r.fail(r.off, "truncated varint")
			return 0
		}
		c := r.b[r.off+i]
		if i == 9 && c > 1 {
			r.fail(r.off, "varint overflows 64 bits")
			return 0
		}
		v |= uint64(c&0x7f) << (7 * i)
		if c < 0x80 {
			if i > 0 && c == 0 {
				r.fail(r.off, "non-minimal varint")
				return 0
			}
			r.off += i + 1
			return v
		}
	}
}

// I reads a signed integer written as zigzag (R2).
func (r *Reader) I() int64 {
	u := r.U()
	return int64(u>>1) ^ -int64(u&1)
}

// B reads a boolean: one byte, 0 or 1 (R3).
func (r *Reader) B() bool {
	if r.err != nil {
		return false
	}
	if r.off >= len(r.b) {
		r.fail(r.off, "truncated boolean")
		return false
	}
	c := r.b[r.off]
	if c > 1 {
		r.fail(r.off, "boolean is neither 0 nor 1")
		return false
	}
	r.off++
	return c == 1
}

// Raw reads n bytes whose length the format fixes (R4). The result aliases
// the input. A negative n, or an n beyond the bytes remaining, is malformed.
func (r *Reader) Raw(n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || n > r.Remaining() {
		r.fail(r.off, "fewer bytes remain than the format fixes")
		return nil
	}
	p := r.b[r.off : r.off+n : r.off+n]
	r.off += n
	return p
}

// Bytes reads a length and then that many bytes (R5). The length is at most
// the bytes remaining. The result aliases the input.
func (r *Reader) Bytes() []byte {
	at := r.off
	n := r.U()
	if n > uint64(r.Remaining()) {
		r.fail(at, "length exceeds the bytes remaining")
		return nil
	}
	return r.Raw(int(n))
}

// Count reads the count of a sequence (R8). Every element takes at least
// minPerElement bytes (at least 1 is assumed), so a count above what the bytes
// remaining can hold is malformed and no count asks for more than the input.
func (r *Reader) Count(minPerElement int) int {
	at := r.off
	n := r.U()
	if !r.fits(n, minPerElement) {
		r.fail(at, "count exceeds the bytes remaining")
		return 0
	}
	return int(n)
}

// Implied checks a count that the configuration implies (R8) in the same way
// as Count does a stored one: n elements of at least minPerElement bytes must
// fit in the bytes remaining. It returns n, or 0 after recording the error.
// Nothing is read.
func (r *Reader) Implied(n, minPerElement int) int {
	if r.err != nil {
		return 0
	}
	if !r.fits(uint64(n), minPerElement) {
		r.fail(r.off, "an implied count exceeds the bytes remaining")
		return 0
	}
	return n
}

// fits reports whether n elements of at least minPerElement bytes (1 if less)
// fit in the bytes remaining.
func (r *Reader) fits(n uint64, minPerElement int) bool {
	return n <= uint64(r.Remaining()/max(minPerElement, 1))
}

// Index reads an index or an enumeration, which must be below n (R6).
func (r *Reader) Index(n uint64) uint64 {
	at := r.off
	v := r.U()
	if v >= n {
		r.fail(at, "index or enumeration out of its bound")
		return 0
	}
	return v
}

// Strict reads the next key of a sequence that is strictly ascending (R9):
// it must exceed prev, unless started is false, which is the first key.
func (r *Reader) Strict(prev uint64, started bool) uint64 {
	at := r.off
	v := r.U()
	if started && v <= prev {
		r.fail(at, "key is not above the previous key")
		return 0
	}
	return v
}

// Done returns the first error, or an error if bytes remain (R11), or nil.
func (r *Reader) Done() error {
	if r.Remaining() != 0 {
		r.fail(r.off, "trailing bytes after the last field")
	}
	return r.err
}

// Bounded reads a U that is at most max, or records a violation of R13 naming
// what was read and returns 0.
func (r *Reader) Bounded(what string, max int64) int64 {
	v := r.U()
	if max < 0 || v > uint64(max) {
		r.Failf("%s %d is above %d", what, v, max)
		return 0
	}
	return int64(v)
}
