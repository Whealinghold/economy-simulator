package settlement

// Headroom returns max(0, limit - households), the room a role's ceiling leaves
// (spec-settlements.md 6.4.2): nobody is removed for a ceiling, so a role above
// it has none. A negative limit counts as 0 and negative households as 0, so the
// difference of two non-negative int64 cannot overflow.
func Headroom(limit, households int64) int64 {
	limit = max(limit, 0)
	households = max(households, 0)
	return max(limit-households, 0)
}
