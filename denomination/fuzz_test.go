package denomination_test

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/economy-sim/denomination"
)

// FuzzFormat verifies, for any valid ladder, precision and int64 amount, that
// Format does not panic, is never empty, writes a sign only when a count or
// fraction is non-zero, and that the printed counts add up to the magnitude
// within the truncated remainder.
//
// Requirement: NFR-CUR-001, FR-CUR-012, FR-CUR-016
func FuzzFormat(f *testing.F) {
	f.Add(int64(0), int64(10_000), int64(1_000), int64(100), 0)
	f.Add(int64(1), int64(1), int64(1), int64(1), 2)
	f.Add(int64(-1), int64(240), int64(12), int64(1), 6)
	f.Add(int64(1_460), int64(10_000), int64(1_000), int64(100), 2)
	f.Add(int64(math.MinInt64), int64(denomination.MaxSize), int64(1), int64(1), 6)
	f.Add(int64(math.MaxInt64), int64(10_000), int64(1_000), int64(100), 0)
	f.Fuzz(func(t *testing.T, amount, s1, s2, s3 int64, p int) {
		var sizes []int64
		for _, s := range []int64{s1, s2, s3} {
			if s >= 1 && s <= denomination.MaxSize {
				sizes = append(sizes, s)
			}
		}
		slices.SortFunc(sizes, func(a, b int64) int { return int(b - a) })
		sizes = slices.Compact(sizes)
		if len(sizes) == 0 || p < 0 || p > denomination.MaxPrecision {
			t.Skip("not a valid ladder or precision")
		}
		var l denomination.Ladder
		for i, s := range sizes {
			l = append(l, denomination.Unit{Name: "n" + strconv.Itoa(i), Symbol: "u" + strconv.Itoa(i), Size: s})
		}
		fm, err := denomination.NewFormatter(l, denomination.WithPrecision(p))
		if err != nil {
			t.Fatalf("NewFormatter(%v, %d) error = %v", l, p, err)
		}
		out := fm.Format(amount)
		if out == "" {
			t.Fatalf("Format(%d) on %v = empty", amount, l)
		}
		checkFormatted(t, out, amount, l, p)
	})
}

// checkFormatted parses out back and checks the sign and the sum: the printed
// whole counts and fraction add up to at most the magnitude, and what is left
// is below one step of the last digit printed (below the smallest size at
// precision 0; below size/10^p plus the floor of the fraction at p > 0, so
// left*10^p < size + 10^p).
func checkFormatted(t *testing.T, out string, amount int64, l denomination.Ladder, p int) {
	t.Helper()
	body, neg := strings.CutPrefix(out, "-")
	mag := uint64(amount)
	if amount < 0 {
		mag = -uint64(amount)
	}
	scale := uint64(1)
	for range p {
		scale *= 10
	}
	var sum uint64
	nonZero := false
	fields := strings.Fields(body)
	for i := 0; i+1 < len(fields); i += 2 {
		var size uint64
		for _, u := range l {
			if u.Symbol == fields[i+1] {
				size = uint64(u.Size)
			}
		}
		whole, frac, _ := strings.Cut(fields[i], ".")
		n, err := strconv.ParseUint(whole, 10, 64)
		if err != nil || size == 0 {
			t.Fatalf("Format(%d) = %q: cannot parse %q", amount, out, fields[i])
		}
		sum += n * size
		if frac != "" {
			fv, err := strconv.ParseUint(frac, 10, 64)
			if err != nil {
				t.Fatalf("Format(%d) = %q: cannot parse fraction %q", amount, out, frac)
			}
			sum += fv * size / scale
			nonZero = nonZero || fv != 0
		}
		nonZero = nonZero || n != 0
	}
	if neg && !nonZero {
		t.Errorf("Format(%d) = %q: sign with nothing non-zero", amount, out)
	}
	if sum > mag {
		t.Errorf("Format(%d) = %q: printed %d exceeds magnitude %d", amount, out, sum, mag)
	}
	small := uint64(l[len(l)-1].Size)
	if left := mag - sum; (p == 0 && left >= small) || (p > 0 && left*scale >= small+scale) {
		t.Errorf("Format(%d) = %q: unprinted %d is not below %d", amount, out, left, small)
	}
}
