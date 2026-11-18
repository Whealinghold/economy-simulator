package facility

// RoundRobin splits n facilities of an initial row over agents agents, one each
// in turn from the first (spec-infrastructure.md 3.4.2): the first n mod agents
// agents get one more than the rest, so 3 over 5 gives 1, 1, 1, 0, 0 and 8 over
// 5 gives 2, 2, 2, 1, 1. It returns nil when agents is below 1 or n is below 0.
func RoundRobin(n int64, agents int) []int64 {
	if agents < 1 || n < 0 {
		return nil
	}
	out := make([]int64, agents)
	base, extra := n/int64(agents), n%int64(agents)
	for i := range out {
		out[i] = base
		if int64(i) < extra {
			out[i]++
		}
	}
	return out
}
