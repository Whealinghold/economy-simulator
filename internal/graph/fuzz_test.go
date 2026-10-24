package graph

import (
	"math"
	"slices"
	"testing"
)

// decode turns fuzz bytes into a graph and a few queries: the first byte is
// the region count (1 to 8), then each triple of bytes is an edge (from, to,
// length). A fourth byte per triple is not read; a length byte of 255 means
// 2^55, so sums near int64 are reachable. Duplicates and self edges are
// dropped, as the root's validation would reject them, and the list is sorted.
func decode(data []byte) (int, []Edge) {
	if len(data) == 0 {
		return 1, nil
	}
	n := int(data[0])%8 + 1
	seen := map[[2]int]bool{}
	var open []Edge
	for i := 1; i+2 < len(data); i += 3 {
		from, to := int(data[i])%n, int(data[i+1])%n
		length := int64(data[i+2]) + 1
		if data[i+2] == 255 {
			length = 1 << 55
		}
		if from == to || seen[[2]int{from, to}] {
			continue
		}
		seen[[2]int{from, to}] = true
		open = append(open, Edge{From: from, To: to, Length: length})
	}
	slices.SortFunc(open, func(a, b Edge) int {
		if a.From != b.From {
			return a.From - b.From
		}
		return a.To - b.To
	})
	return n, open
}

// encode is the inverse of decode for a small graph, to seed the fuzzer.
func encode(n int, open []Edge) []byte {
	out := []byte{byte(n - 1)}
	for _, e := range open {
		out = append(out, byte(e.From), byte(e.To), byte(e.Length-1))
	}
	return out
}

// FuzzBuild verifies that Build and the queries never panic for any edge list
// and any region number, and that when Build succeeds the triangle inequality
// holds, a path has the length Dist, and every consecutive pair of a path is
// an open edge.
//
// Requirement: NFR-REG-003
func FuzzBuild(f *testing.F) {
	// The three worked examples of spec-regions.md 12 and a graph with
	// lengths near the overflow (255 means 2^55).
	a := matrix{{0, 55, 120}, {30, 0, 70}, {90, 40, 0}}
	b := matrix{{0, 3000, 0}, {3000, 0, 2000}, {0, 9400, 0}}
	c := matrix{{0, 160, 80}, {160, 0, 80}, {80, 80, 0}}
	for _, m := range []matrix{a, b, c} {
		f.Add(encode(len(m), clampEdges(m.edges())), int64(100), 1)
	}
	f.Add([]byte{3, 0, 1, 255, 1, 2, 255, 2, 0, 255}, int64(math.MaxInt64), -1)
	f.Fuzz(func(t *testing.T, data []byte, limit int64, region int) {
		n, open := decode(data)
		tb, err := Build(n, open)
		if err != nil {
			return // an overflow is a reported error, not a panic
		}
		lens := map[[2]int]int64{}
		for _, e := range open {
			lens[[2]int{e.From, e.To}] = e.Length
		}
		tb.AppendWithin(nil, region, limit)
		tb.Comp(region)
		tb.Next(region, 0)
		tb.AppendPath(nil, region, 0)
		for a := 0; a < n; a++ {
			for b := 0; b < n; b++ {
				checkQuery(t, tb, lens, a, b)
			}
		}
	})
}

// clampEdges caps the lengths at 254 so encode keeps them in one byte below 255.
func clampEdges(open []Edge) []Edge {
	out := slices.Clone(open)
	for i := range out {
		out[i].Length = min(out[i].Length, 254)
	}
	return out
}

// checkQuery asserts the properties of one (a, b) pair of a built graph.
func checkQuery(t *testing.T, tb *Tables, lens map[[2]int]int64, a, b int) {
	t.Helper()
	d, ok := tb.Dist(a, b)
	if !ok {
		if path, pok := tb.AppendPath(nil, a, b); pok || path != nil {
			t.Fatalf("AppendPath(%d, %d) = %v, true for an unreachable pair", a, b, path)
		}
		return
	}
	for c := 0; c < tb.n; c++ {
		dbc, ok1 := tb.Dist(b, c)
		dac, ok2 := tb.Dist(a, c)
		if ok1 && d <= math.MaxInt64-dbc && (!ok2 || dac > d+dbc) {
			t.Fatalf("triangle inequality: Dist(%d, %d) = %d, %v but Dist(%d, %d) + Dist(%d, %d) = %d", a, c, dac, ok2, a, b, b, c, d+dbc)
		}
	}
	path, pok := tb.AppendPath(nil, a, b)
	if !pok || path[0] != a || path[len(path)-1] != b {
		t.Fatalf("AppendPath(%d, %d) = %v, %v, want a path from %d to %d", a, b, path, pok, a, b)
	}
	var total int64
	for i := 0; i+1 < len(path); i++ {
		w, open := lens[[2]int{path[i], path[i+1]}]
		if !open {
			t.Fatalf("path %v from %d to %d uses %d to %d, which is not an open edge", path, a, b, path[i], path[i+1])
		}
		total += w
	}
	if total != d {
		t.Fatalf("path %v from %d to %d has length %d, want Dist %d", path, a, b, total, d)
	}
}

// BenchmarkBuild reports the cost of a construction for 100 regions and 400
// edges (a larger realm) and for the bound, 1 000 regions and 20 000 edges
// (spec-regions.md 4.4.3). It reports; it never fails.
//
// Requirement: NFR-REG-002
func BenchmarkBuild(b *testing.B) {
	for _, c := range []struct {
		name         string
		n, perRegion int
	}{{"100 regions 400 edges", 100, 4}, {"1000 regions 20000 edges", 1000, 20}} {
		rng := splitmix(3)
		var open []Edge
		for from := 0; from < c.n; from++ {
			seen := map[int]bool{}
			for len(seen) < c.perRegion {
				if to := rng.below(c.n); to != from {
					seen[to] = true
				}
			}
			tos := make([]int, 0, len(seen))
			for to := range seen {
				tos = append(tos, to)
			}
			slices.Sort(tos)
			for _, to := range tos {
				open = append(open, Edge{From: from, To: to, Length: int64(10 + rng.below(990))})
			}
		}
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := Build(c.n, open); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
