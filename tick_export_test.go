package economysim

// This file is the test-only seam of the tick (architecture.md 10.2): it gives
// the external tests the one derived value of the entry and phase 1 that no
// event or snapshot row carries. Nothing here is exported from production code.

// UnitCostOf returns the cost anchor of a commodity in a region as the last P1.2
// computed it, 0 when unknown.
func UnitCostOf(w *World, region RegionID, c CommodityID) int64 { return w.prod.unitCost(region, c) }

// HasReservation reports whether the ledger of the world holds any open
// reservation, for the window test of FR-WLD-045.
func HasReservation(w *World) bool { return w.ledger.HasReservation() }
