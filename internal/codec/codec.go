package codec

import (
	"errors"
	"fmt"
)

// ErrMalformed is what every error of a Reader matches with errors.Is: the
// bytes are not a canonical encoding of the thing being read (R1 to R13 of
// spec-checkpoint.md 5.2). The root maps it to its own sentinel.
var ErrMalformed = errors.New("codec: malformed input")

// Error is the error of a Reader. It says where the input went wrong and
// matches ErrMalformed.
type Error struct {
	// Offset is the byte offset in the input of the Reader at which the read
	// that failed began.
	Offset int
	// Msg says what was found.
	Msg string
}

// Error implements error.
func (e *Error) Error() string {
	return fmt.Sprintf("codec: malformed input at byte %d: %s", e.Offset, e.Msg)
}

// Unwrap lets errors.Is match ErrMalformed.
func (e *Error) Unwrap() error { return ErrMalformed }

// Writer appends canonical encodings to a byte slice. It cannot fail. The zero
// value is ready to use.
type Writer struct {
	b []byte
}

// NewWriter returns a Writer that appends to dst, which may be nil.
func NewWriter(dst []byte) *Writer { return &Writer{b: dst} }

// Out returns the bytes written so far, including those of dst. The Writer
// keeps writing after the same array, so a caller that keeps the slice must
// stop writing.
func (w *Writer) Out() []byte { return w.b }

// Len returns the number of bytes in Out.
func (w *Writer) Len() int { return len(w.b) }

// USize returns the number of bytes U writes for v: 1 for 0, then one more
// for every seven bits.
func USize(v uint64) int {
	n := 1
	for ; v >= 0x80; v >>= 7 {
		n++
	}
	return n
}

// U appends an unsigned integer as a minimal LEB128 varint, least significant
// seven bits first (R1).
func (w *Writer) U(v uint64) {
	for v >= 0x80 {
		w.b = append(w.b, byte(v)|0x80)
		v >>= 7
	}
	w.b = append(w.b, byte(v))
}

// I appends a signed integer as the zigzag of itself, then U (R2).
func (w *Writer) I(v int64) {
	if v < 0 {
		w.U(uint64(^v)<<1 + 1)
		return
	}
	w.U(uint64(v) << 1)
}

// B appends a boolean as one byte, 0 or 1 (R3).
func (w *Writer) B(v bool) {
	if v {
		w.b = append(w.b, 1)
		return
	}
	w.b = append(w.b, 0)
}

// Raw appends bytes whose length the format fixes (R4).
func (w *Writer) Raw(p []byte) { w.b = append(w.b, p...) }

// Bytes appends a length and then that many bytes (R5).
func (w *Writer) Bytes(p []byte) {
	w.U(uint64(len(p)))
	w.b = append(w.b, p...)
}

// Count appends the count of a sequence (R8) as U. A negative count is a
// programmer error.
func (w *Writer) Count(n int) {
	if n < 0 {
		panic("codec: programmer error: violated invariant: negative count")
	}
	w.U(uint64(n))
}
