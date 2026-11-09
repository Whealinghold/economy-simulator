package transit

// HaulPerUnit returns s = ceil(distance / span), the haul units one carrying
// unit costs over a distance (spec-transport.md 3.3.3). ok is false when
// distance is negative or span is not positive.
func HaulPerUnit(distance, span int64) (s int64, ok bool) {
	if distance < 0 || span <= 0 {
		return 0, false
	}
	return ceilDiv(distance, span), true
}
