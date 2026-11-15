package economysim

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/clearing"
	"github.com/Whealinghold/economy-sim/internal/room"
)

// The benchmarks of the storage module (NFR-STO-003), each reporting allocations. The
// cost of the room rules is a few integer operations per commodity; the figures are
// reported, not asserted.

// BenchmarkAdmitWithRoom is the admission of a bid of a limited account: every row of
// the table, the no_room row last.
//
// Requirement: NFR-STO-003
func BenchmarkAdmitWithRoom(b *testing.B) {
	f := clearing.Facts{
		AgentKnown: true, SideAllowed: true, CommodityKnown: true, Tradable: true, Bid: true,
		Price: 100, Qty: 10, MaxPrice: MaxPrice, MaxQuantity: MaxQuantity, MaxMoney: MaxMoney,
		Available: 1_000_000, Limited: true, FreeMass: 100, FreeBulk: 100, SizeMass: 3, SizeBulk: 2,
	}
	b.ReportAllocs()
	for b.Loop() {
		if clearing.Admit(f) != clearing.Admitted {
			b.Fatal("the bid was not admitted")
		}
	}
}

// BenchmarkStartSizing is the room bound of a start: the largest k whose pending output
// fits the free room less the upkeep room (spec-storage.md 5.3.2).
//
// Requirement: NFR-STO-003
func BenchmarkStartSizing(b *testing.B) {
	in := []room.Item{{Qty: 2, Size: room.Pair{Mass: 3, Bulk: 2}}}
	out := []room.Output{{Item: room.Item{Qty: 3, Size: room.Pair{Mass: 3, Bulk: 2}}}}
	b.ReportAllocs()
	for b.Loop() {
		room.KRoom(in, out, 10_000, room.Pair{Mass: 300, Bulk: 200}, room.Pair{Mass: 3, Bulk: 2}, 1000)
	}
}

// BenchmarkTargetScaling is the scaling f of the targets of 16 commodities to the room
// (spec-storage.md 6.2.2).
//
// Requirement: NFR-STO-003
func BenchmarkTargetScaling(b *testing.B) {
	items := make([]room.Item, 16)
	for i := range items {
		items[i] = room.Item{Qty: int64(3 + i), Size: room.Pair{Mass: int64(1 + i%4), Bulk: int64(1 + i%3)}}
	}
	b.ReportAllocs()
	for b.Loop() {
		room.Scale(items, room.Pair{Mass: 200, Bulk: 150}, room.Pair{Mass: 10, Bulk: 10}, room.Pair{Mass: 20, Bulk: 20})
	}
}

// BenchmarkDiscardOrder is the order of the discard that makes room for tier 1 over 16
// candidates, each a surplus of one lot (spec-storage.md 6.3.2).
//
// Requirement: NFR-STO-003
func BenchmarkDiscardOrder(b *testing.B) {
	cands := make([]room.Candidate, 16)
	lots := make([]int64, 16)
	for i := range cands {
		lots[i] = int64(2 + i%5)
		cands[i] = room.Candidate{Commodity: i, Size: room.Pair{Mass: int64(1 + i%4), Bulk: int64(1 + i%3)}, Lots: lots[i : i+1]}
	}
	b.ReportAllocs()
	for b.Loop() {
		room.DiscardOrder(room.Pair{Mass: 40, Bulk: 30}, room.Pair{Mass: 200, Bulk: 150}, cands)
	}
}
