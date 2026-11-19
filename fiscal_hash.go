package economysim

// This file is the configuration hash of the fiscal module (spec-config.md 8.2.2,
// spec-fiscal.md 14; a continuation file of fiscal.go, architecture.md 2.6). The ids
// are the declaration order of the spec's types and are pinned by
// TestFiscalHashEncodingPinned. A scale is state and is not hashed (FR-FIS-005), and
// Label of every flow is presentation only and has no id.

// pairsFiscal writes the pairs of a Fiscal: Taxes 1, Duties 2, Tolls 3, Payouts 4,
// Purchases 5 and Releases 6, each list in ID order (the canonical order).
func pairsFiscal(e *encoder, m Fiscal) {
	encodeList(e, 1, m.Taxes, func(e *encoder, t Tax) {
		e.text(1, t.ID)
		e.text(2, t.Authority)
		e.nested(3, func(e *encoder) { pairsFlowTarget(e, t.Target) })
		e.num(4, t.Amount)
		e.num(5, t.RateBP)
		e.num(6, t.Exempt)
		e.text(7, t.Per)
		e.nested(8, func(e *encoder) { pairsWindow(e, t.Window) })
	})
	encodeList(e, 2, m.Duties, func(e *encoder, d Duty) {
		e.text(1, d.ID)
		e.text(2, d.Authority)
		e.text(3, d.Region)
		e.text(4, d.Commodity)
		e.text(5, d.Payer)
		e.text(6, d.Scope)
		e.num(7, d.RateBP)
		e.nested(8, func(e *encoder) { pairsWindow(e, d.Window) })
	})
	encodeList(e, 3, m.Tolls, func(e *encoder, t Toll) {
		e.text(1, t.ID)
		e.text(2, t.Authority)
		e.text(3, t.On)
		e.text(4, t.Mode)
		e.num(5, t.Amount)
		e.nested(6, func(e *encoder) { pairsWindow(e, t.Window) })
	})
	encodeList(e, 4, m.Payouts, pairsPayout)
	encodeList(e, 5, m.Purchases, func(e *encoder, p Purchase) {
		e.text(1, p.ID)
		e.text(2, p.Authority)
		e.text(3, p.Commodity)
		e.num(4, p.Quantity)
		e.num(5, p.LimitBP)
		e.num(6, p.Hold)
		e.nested(7, func(e *encoder) { pairsWindow(e, p.Window) })
	})
	encodeList(e, 6, m.Releases, func(e *encoder, r Release) {
		e.text(1, r.ID)
		e.text(2, r.Authority)
		e.text(3, r.Commodity)
		e.num(4, r.Quantity)
		e.num(5, r.LimitBP)
		e.num(6, r.Reserve)
		e.nested(7, func(e *encoder) { pairsWindow(e, r.Window) })
	})
}

// pairsPayout writes the pairs of a Payout: ID 1, Authority 2, Target 3, Commodity 4,
// Amount 5, Level 6, Per 7, NotIncome 8 and Window 9.
func pairsPayout(e *encoder, p Payout) {
	e.text(1, p.ID)
	e.text(2, p.Authority)
	e.nested(3, func(e *encoder) { pairsFlowTarget(e, p.Target) })
	e.text(4, p.Commodity)
	e.num(5, p.Amount)
	e.num(6, p.Level)
	e.text(7, p.Per)
	e.flag(8, p.NotIncome)
	e.nested(9, func(e *encoder) { pairsWindow(e, p.Window) })
}

// pairsWindow writes the pairs of a Window: Start 1, Until 2 and Seasons 3, the
// seasons in the calendar's order (the canonical order), so two spellings of one
// window hash alike.
func pairsWindow(e *encoder, w Window) {
	e.num(1, w.Start)
	e.num(2, w.Until)
	e.texts(3, w.Seasons)
}
