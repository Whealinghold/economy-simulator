package economysim

// position is one entry of the table of spec-world.md 6 that has a call: its
// identifier, the area code of its owner and where it runs (architecture.md
// 2.10.3, testing.md 6.8).
type position struct {
	id    string
	owner string
	where placement
}

// placement says where a position runs: in the straight line of positions
// (line) or inside beginStep or endStep (bracket).
type placement int

const (
	// line is a position with its own call in the straight line.
	line placement = iota
	// bracket is a position that runs inside beginStep or endStep.
	bracket
)

// positionList is the list of the positions of spec-world.md 6 that have a call
// or run in a bracket, in the order of the table. It is never written. A
// module's change deletes its entries from notBuiltPositions and appends its
// own here, in the order of the table.
var positionList = [...]position{
	{"P0.1", "WLD", bracket}, // the guards, in beginStep
	{"P0.2", "LED", line},
	{"P0.2a", "OBS", bracket}, // the construction bucket, in beginStep
	{"P0.3", "OBS", bracket},  // TickStarted, in beginStep
	{"P1.1", "CAL", line},
	{"P1.2", "PRO", line},
	{"P1.6", "EST", line},
	{"P1.7", "MON", line},
	{"P2.1", "PRO", line},
	{"P2.3", "RES", line},
	{"P2.5", "AGT", line},
	{"P2.6", "RES", line},
	{"P2.7", "PRO", line},
	{"P2.8a", "OBS", line},
	{"P3.2", "AGT", line},
	{"P3.4", "EST", line},
	{"P3.6", "INF", line},
	{"P4.1", "CLR", line},
	{"P5.1", "CLR", line},
	{"P5.3", "EST", line},
	{"P5.4", "CLR", line},
	{"P6.1", "AGT", line},
	{"P6.2", "DEM", line},
	{"P6.3", "WEL", line},
	{"P6.4", "AGT", line},
	{"P7.1", "SPN", line},
	{"P8.1", "LED", line},
	{"P8.2", "LED", line},
	{"P8.3", "RES", line},
	{"P8.4", "LED", line},
	{"P8.5", "OBS", bracket}, // TickEnded, in endStep
	{"P8.6", "WLD", bracket}, // the tick advance, in endStep
}

// notBuiltPositions are the positions of spec-world.md 6 that have no call
// yet, in the order of the table. The position test fails on a listed position
// that has a call and on a position of the table that is neither here nor in
// positionList.
var notBuiltPositions = [...]string{
	"P0.3a",
	"P1.3", "P1.3a", "P1.4", "P1.5", "P1.7a", "P1.7b", "P1.8", "P1.9", "P1.9a",
	"P2.2", "P2.4", "P2.5a", "P2.8", "P2.8b",
	"P3.1", "P3.1a", "P3.3", "P3.3a", "P3.3b", "P3.3c", "P3.4a", "P3.5", "P3.6a", "P3.6b", "P3.6c",
	"P4.1a", "P4.2",
	"P5.2",
	"P7.2", "P7.2a", "P7.2b",
	"P8.1a", "P8.1b", "P8.3a", "P8.3b", "P8.3c",
}

// positions is the straight line of a step: one call for each position of
// positionList whose where is line, in the order of the table of spec-world.md
// 6, and nothing else. It returns the first error of a position and the
// positions after it do not run. The positions that run in beginStep and
// endStep are not here.
//
// Excluded from coverage (testing.md 9.5): the `return err` after a position
// other than p41Clear and p84Audit. Such a position returns an error only for a
// ledger or overflow refusal of the world's own accounts, which no test can
// inject from the root (D190); the clearing and audit failures, which can be
// injected, are tested.
func (w *World) positions() error {
	w.p02()
	if err := w.p11Moment(); err != nil {
		return err
	}
	w.p12Anchors()
	if err := w.p16Relief(); err != nil {
		return err
	}
	w.p17Flows()
	w.p21Complete()
	w.p23Snapshot()
	if err := w.p25Work(); err != nil {
		return err
	}
	w.p26Ration()
	if err := w.p27Start(); err != nil {
		return err
	}
	w.p28aHalted()
	if err := w.p32Trade(); err != nil {
		return err
	}
	if err := w.p34Estates(); err != nil {
		return err
	}
	w.p36Aggregates()
	if err := w.p41Clear(); err != nil {
		return err
	}
	return w.positionsTail()
}

// positionsTail is the second half of the straight line, split from positions
// to stay within the Lizard complexity limit (N-243). The position test reads
// it as part of the straight line, after positions and in source order.
func (w *World) positionsTail() error {
	w.p51Window()
	if err := w.p53Markdown(); err != nil {
		return err
	}
	w.p54Outcomes()
	if err := w.p61Upkeep(); err != nil {
		return err
	}
	if err := w.p62Wants(); err != nil {
		return err
	}
	w.p63Welfare()
	w.p64Shortfall()
	if err := w.p71Entry(); err != nil {
		return err
	}
	if err := w.p81Age(); err != nil {
		return err
	}
	w.p82Flows()
	w.p83Regen()
	return w.p84Audit()
}

// p02 is P0.2: nothing is cleared at entry. The flow totals and the per-tick
// counters are cleared at P8.6, so the position is empty.
func (w *World) p02() {}
