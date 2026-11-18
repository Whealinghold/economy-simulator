package institution

// Headroom returns the room a count of places leaves to the households that are in
// a scope (spec-institutions.md 3.4.2, FR-ORG-007): max(0, places - households), and
// noLimit true when places is 0 (the body is then a privilege of its roles and has no
// figure). Nobody is removed for being above the places, so a count above them has
// no headroom. A negative places counts as no limit and negative households as 0.
func Headroom(places, households int64) (room int64, noLimit bool) {
	if places <= 0 {
		return 0, true
	}
	return max(places-max(households, 0), 0), false
}

// Least returns the least of two limits (spec-institutions.md 3.4.3): the role's
// headroom is the least of its class limit's and its body's. A side with noLimit
// true has no figure and does not bind; with neither, the result has no limit.
func Least(a int64, aNoLimit bool, b int64, bNoLimit bool) (room int64, noLimit bool) {
	switch {
	case aNoLimit && bNoLimit:
		return 0, true
	case aNoLimit:
		return b, false
	case bNoLimit:
		return a, false
	}
	return min(a, b), false
}
