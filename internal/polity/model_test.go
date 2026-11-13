package polity_test

import (
	"math/rand/v2"
	"slices"
)

// forest is a parent array and a member array, the input of polity.Build.
type forest struct {
	parent, member []int32
}

// randomForest builds a forest from seed with at most levels levels of
// jurisdictions: the labels are shuffled, so a parent may have a larger index
// than its child, which a build in file order would never see.
func randomForest(seed uint64, jurisdictions, regions, levels int) forest {
	rng := rand.New(rand.NewPCG(seed, 0x9e3779b97f4a7c15))
	depth := make([]int, jurisdictions)
	creation := make([]int32, jurisdictions) // creation order: parent index, -1 for a root
	for i := range creation {
		creation[i] = -1
		if i > 0 && rng.IntN(4) != 0 {
			p := rng.IntN(i)
			if depth[p]+1 < levels {
				creation[i] = int32(p)
				depth[i] = depth[p] + 1
			}
		}
	}
	label := rng.Perm(jurisdictions) // creation index -> final index
	f := forest{parent: make([]int32, jurisdictions), member: make([]int32, regions)}
	for i, p := range creation {
		if p < 0 {
			f.parent[label[i]] = -1
		} else {
			f.parent[label[i]] = int32(label[p])
		}
	}
	for r := range f.member {
		f.member[r] = -1
		if jurisdictions > 0 && rng.IntN(5) != 0 {
			f.member[r] = int32(rng.IntN(jurisdictions))
		}
	}
	return f
}

// modelChain is the chain of region r by climbing parents.
func (f forest) modelChain(r int) []int32 {
	var c []int32
	for cur := f.member[r]; cur >= 0; cur = f.parent[cur] {
		c = append(c, cur)
	}
	return c
}

// modelCrossing is the second formulation of spec-polity.md 14: climb from the
// direct jurisdiction of each region to the lowest ancestor they share, and
// reverse the entered side. It never filters one chain by the other.
func (f forest) modelCrossing(a, b int) (exit, enter []int32) {
	above := map[int32]bool{} // a lookup only; no result depends on its order
	for cur := f.member[b]; cur >= 0; cur = f.parent[cur] {
		above[cur] = true
	}
	cur := f.member[a]
	for cur >= 0 && !above[cur] {
		exit = append(exit, cur)
		cur = f.parent[cur]
	}
	shared := cur // the lowest jurisdiction covering both, or -1
	for c := f.member[b]; c >= 0 && c != shared; c = f.parent[c] {
		enter = append(enter, c)
	}
	slices.Reverse(enter)
	return exit, enter
}

// modelCommon is the lowest shared ancestor by the same climb.
func (f forest) modelCommon(a, b int) int32 {
	above := map[int32]bool{}
	for cur := f.member[b]; cur >= 0; cur = f.parent[cur] {
		above[cur] = true
	}
	for cur := f.member[a]; cur >= 0; cur = f.parent[cur] {
		if above[cur] {
			return cur
		}
	}
	return -1
}

// modelCovered lists the regions covered by j by testing each region's chain.
func (f forest) modelCovered(j int32) []int32 {
	var out []int32
	for r := range f.member {
		if slices.Contains(f.modelChain(r), j) {
			out = append(out, int32(r))
		}
	}
	return out
}
