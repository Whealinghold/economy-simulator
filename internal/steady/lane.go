package steady

// Lane is a directed route for one commodity: a column with +1 in the goods row of
// (From, Commodity), -(10 000-LossBP)/10 000 in that of (To, Commodity) and
// Labour/10 000 in the labour row of From, at cost 0 and with no capacity row.
// Labour is in household-ticks per 10 000 units, LossBP in basis points.
type Lane struct {
	From, To, Commodity int
	Labour, LossBP      int64
}

// checkLanes panics on a lane with a region or commodity out of range, a
// programmer invariant, and returns ErrBadProgramme for a self-lane, a negative
// labour or a loss outside 0 to 9 999.
func checkLanes(m Model) error {
	for _, ln := range m.Lanes {
		if !ln.inRange(len(m.Regions), m.Commodities) {
			panic("steady: invariant violated: lane region or commodity out of range")
		}
		if ln.From == ln.To || ln.Labour < 0 || ln.LossBP < 0 || ln.LossBP >= 10000 {
			return ErrBadProgramme
		}
	}
	return nil
}

// inRange reports whether the lane names regions below regions and a commodity
// below commodities.
func (ln Lane) inRange(regions, commodities int) bool {
	return ln.From >= 0 && ln.From < regions && ln.To >= 0 && ln.To < regions &&
		ln.Commodity >= 0 && ln.Commodity < commodities
}

// laneColumns builds the columns of the lanes, in order.
func laneColumns(lanes []Lane, l Layout) []Column {
	cols := make([]Column, len(lanes))
	for i, ln := range lanes {
		cols[i] = laneColumn(ln, l)
	}
	return cols
}

// laneColumn builds the column of a lane: it takes one unit from the goods row of
// its origin, adds (10 000-LossBP)/10 000 to that of its destination and charges
// Labour/10 000 household-ticks to the labour row of its origin.
func laneColumn(ln Lane, l Layout) Column {
	col := Column{Cost: Ratio{Num: 0, Den: 1}, Entries: []Entry{
		{Row: l.GoodsRow(ln.From, ln.Commodity), Val: Ratio{Num: 1, Den: 1}},
		{Row: l.GoodsRow(ln.To, ln.Commodity), Val: Ratio{Num: ln.LossBP - 10000, Den: 10000}},
	}}
	if ln.Labour > 0 {
		col.Entries = append(col.Entries, Entry{Row: l.LabourRow(ln.From), Val: Ratio{Num: ln.Labour, Den: 10000}})
	}
	return col
}
