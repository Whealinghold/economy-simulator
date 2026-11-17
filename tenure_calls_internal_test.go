package economysim

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// tr is a TitleTransfer of ore in a region.
func tr(region RegionID, from, to AgentID, share int64, price Money) TitleTransfer {
	return TitleTransfer{Region: region, From: from, To: to, ShareBP: share, Price: price}
}

// TestTransferTitles_RefusalsLeaveEverythingUnchanged verifies, one case for each row of
// the checks of spec-land.md 6.3.4, that the call returns the sentinel of the row and
// changes no title, no money and emits no event.
//
// Requirement: FR-LND-025
func TestTransferTitles_RefusalsLeaveEverythingUnchanged(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		mut     func(m *TenureConfig)
		changes []TitleTransfer
		want    error
	}{
		{"unknown region", nil, []TitleTransfer{tr(9, 2, 3, 10, 0)}, ErrUnknownRegion},
		{"unknown resource", nil, []TitleTransfer{{Region: 0, Resource: 5, From: 2, To: 3, ShareBP: 10}}, ErrUnknownResource},
		{"unknown agent", nil, []TitleTransfer{tr(0, 2, 99, 10, 0)}, ErrUnknownAgent},
		{"estate of another region", nil, []TitleTransfer{tr(0, 1, 3, 10, 0)}, ErrNotAHolder},
		{"zero share", nil, []TitleTransfer{tr(0, 2, 3, 0, 0)}, ErrOutOfBounds},
		{"share above 10000", nil, []TitleTransfer{tr(0, NoAgent, 3, 10_001, 0)}, ErrOutOfBounds},
		{"negative price", nil, []TitleTransfer{tr(0, 2, 3, 10, -1)}, ErrOutOfBounds},
		{"from equals to", nil, []TitleTransfer{tr(0, 2, 2, 10, 0)}, ErrOutOfBounds},
		{"both the commons", nil, []TitleTransfer{tr(0, NoAgent, NoAgent, 10, 0)}, ErrOutOfBounds},
		{"sender holds less", nil, []TitleTransfer{tr(0, 2, 3, 101, 0)}, ErrNoTitle},
		{"sender holds none", nil, []TitleTransfer{tr(0, 5, 3, 1, 0)}, ErrNoTitle},
		{"commons hold less", nil, []TitleTransfer{tr(0, NoAgent, 5, 8_701, 0)}, ErrNoTitle},
		{"receiver above the cap", func(m *TenureConfig) { m.MaxHolderShareBP = 150 }, []TitleTransfer{tr(0, 2, 3, 51, 0)}, ErrTitleCap},
		{"enclosure above the cap", func(m *TenureConfig) { m.MaxHolderShareBP = 150 }, []TitleTransfer{tr(0, NoAgent, 5, 151, 0)}, ErrTitleCap},
		{"payer cannot pay", nil, []TitleTransfer{tr(0, 2, 3, 10, MaxMoney)}, ledger.ErrInsufficientFunds},
		{"duplicate key", nil, []TitleTransfer{tr(0, 2, 3, 10, 0), tr(0, 2, 3, 20, 0)}, ErrDuplicateID},
		{"second entry fails", nil, []TitleTransfer{tr(0, 2, 3, 10, 0), tr(0, 2, 4, 101, 0)}, ErrNoTitle},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			obs := &evLog{}
			w := tenureWorld(t, tc.mut, WithObserver(obs))
			before, money := tnHolders(t, w, 0), tnTotalMoney(t, w)
			err := w.TransferTitles(tc.changes)
			if !errors.Is(err, tc.want) {
				t.Fatalf("TransferTitles error = %v, want %v", err, tc.want)
			}
			if after := tnHolders(t, w, 0); !reflect.DeepEqual(before, after) || tnTotalMoney(t, w) != money {
				t.Errorf("a refused call changed the titles or the money")
			}
			if len(of[TitleTransferred](obs)) != 0 {
				t.Errorf("a refused call emitted TitleTransferred")
			}
		})
	}
}

// TestTransferTitles_PayeeAtMaxMoney_ErrOverflow verifies that a price that would take
// the payee's balance past MaxMoney is refused with ErrOverflow and moves nothing.
//
// Requirement: FR-LND-025
func TestTransferTitles_PayeeAtMaxMoney_ErrOverflow(t *testing.T) {
	t.Parallel()
	w := tenureWorld(t, nil)
	payee, err := w.AddAgent(NewAgent{Region: 0, Money: MaxMoney - 10})
	if err != nil {
		t.Fatalf("AddAgent error = %v, want nil", err)
	}
	payer, err := w.AddAgent(NewAgent{Region: 0, Money: 1_000})
	if err != nil {
		t.Fatalf("AddAgent error = %v, want nil", err)
	}
	if err := w.TransferTitles([]TitleTransfer{tr(0, 0, payee, 50, 0)}); err != nil {
		t.Fatalf("the gift to the payee error = %v, want nil", err)
	}
	if err := w.TransferTitles([]TitleTransfer{tr(0, payee, payer, 10, 11)}); !errors.Is(err, ErrOverflow) {
		t.Errorf("a price past MaxMoney = %v, want ErrOverflow", err)
	}
	if err := w.TransferTitles([]TitleTransfer{tr(0, payee, payer, 10, 10)}); err != nil {
		t.Errorf("a price to exactly MaxMoney = %v, want nil", err)
	}
	if got := tnMoney(t, w, payee); got != MaxMoney {
		t.Errorf("payee holds %d, want MaxMoney", got)
	}
}

// TestTransferTitles_PriceMovesByTitleSale_NoMoneyCreated verifies that a share sold for
// a price moves the share and the money together, by the reason title_sale, creates no
// money, and emits one TitleTransferred of cause consumer with the shares after.
//
// Requirement: FR-LND-025, FR-LND-029
func TestTransferTitles_PriceMovesByTitleSale_NoMoneyCreated(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := tenureWorld(t, nil, WithObserver(obs))
	total := tnTotalMoney(t, w)
	buyer, seller := tnMoney(t, w, 3), tnMoney(t, w, 2)
	if err := w.TransferTitles([]TitleTransfer{tr(0, 2, 3, 40, 25)}); err != nil {
		t.Fatalf("TransferTitles error = %v, want nil", err)
	}
	if got, want := tnPairs(tnHolders(t, w, 0)), [][2]int64{{0, 1_000}, {2, 60}, {3, 140}, {4, 100}}; !reflect.DeepEqual(got, want) {
		t.Errorf("titles = %v, want %v", got, want)
	}
	if tnMoney(t, w, 3) != buyer-25 || tnMoney(t, w, 2) != seller+25 || tnTotalMoney(t, w) != total {
		t.Errorf("money: buyer %d, seller %d, total %d, want the price moved and the total %d unchanged", tnMoney(t, w, 3), tnMoney(t, w, 2), tnTotalMoney(t, w), total)
	}
	got := of[TitleTransferred](obs)
	want := TitleTransferred{Region: 0, Resource: 0, From: 2, To: 3, ShareBP: 40, Price: 25, Cause: CauseConsumer, FromAfter: 60, ToAfter: 140}
	if len(got) != 1 || got[0] != want {
		t.Errorf("events = %+v, want [%+v]", got, want)
	}
}

// TestTransferTitles_EnclosureAndRelease verifies that NoAgent on the sending side takes
// from the commons and the buyer pays the pool, and on the receiving side gives back to
// the commons and the pool pays the seller, and that the event names the pool's side.
//
// Requirement: FR-LND-025, FR-LND-029
func TestTransferTitles_EnclosureAndRelease(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := tenureWorld(t, nil, WithObserver(obs))
	pool := tnMoney(t, w, 0)
	if err := w.TransferTitles([]TitleTransfer{tr(0, NoAgent, 5, 200, 30)}); err != nil {
		t.Fatalf("enclosure error = %v, want nil", err)
	}
	tnEq(t, "commons and pool money after the enclosure", [2]int64{tnHolders(t, w, 0).CommonsBP, tnMoney(t, w, 0)}, [2]int64{8_500, pool + 30})
	if err := w.TransferTitles([]TitleTransfer{tr(0, 5, NoAgent, 50, 12)}); err != nil {
		t.Fatalf("release error = %v, want nil", err)
	}
	tnEq(t, "commons and pool money after the release", [2]int64{tnHolders(t, w, 0).CommonsBP, tnMoney(t, w, 0)}, [2]int64{8_550, pool + 18})
	evs := of[TitleTransferred](obs)
	tnEq(t, "the sides of the events", [][3]any{{evs[0].FromPool, evs[0].ToPool, evs[0].FromAfter}, {evs[1].FromPool, evs[1].ToPool, evs[1].ToAfter}},
		[][3]any{{true, false, int64(8_500)}, {false, true, int64(8_550)}})
}

// TestTransferTitles_PoolAsHolder verifies that the pool gives and receives a title
// like any holder, with the cap not applying to it.
//
// Requirement: FR-LND-025, FR-LND-041
func TestTransferTitles_PoolAsHolder(t *testing.T) {
	t.Parallel()
	w := tenureWorld(t, func(m *TenureConfig) { m.MaxHolderShareBP = 100 })
	if err := w.TransferTitles([]TitleTransfer{tr(0, 2, 0, 100, 0), tr(0, 0, 5, 100, 0)}); err != nil {
		t.Fatalf("TransferTitles error = %v, want nil", err)
	}
	if got, want := tnPairs(tnHolders(t, w, 0)), [][2]int64{{0, 1_000}, {3, 100}, {4, 100}, {5, 100}}; !reflect.DeepEqual(got, want) {
		t.Errorf("titles = %v, want %v", got, want)
	}
	if err := w.TransferTitles([]TitleTransfer{tr(0, 3, 0, 100, 0)}); err != nil {
		t.Errorf("a gift to the pool above the cap = %v, want nil: the pool is exempt", err)
	}
}

// TestTransferTitles_EventPerEntryInCanonicalOrder verifies that the events are one per
// entry in ascending (region, resource, From, To) whatever the order of the slice, and
// that the two orders give the same titles.
//
// Requirement: FR-LND-025, FR-LND-029
func TestTransferTitles_EventPerEntryInCanonicalOrder(t *testing.T) {
	t.Parallel()
	entries := []TitleTransfer{tr(1, 6, 7, 5, 0), tr(0, 3, 4, 7, 0), tr(0, 2, 5, 9, 0)}
	run := func(order []int) ([]TitleTransferred, [][2]int64) {
		obs := &evLog{}
		w := tenureWorld(t, nil, WithObserver(obs))
		var in []TitleTransfer
		for _, i := range order {
			in = append(in, entries[i])
		}
		if err := w.TransferTitles(in); err != nil {
			t.Fatalf("TransferTitles error = %v, want nil", err)
		}
		return of[TitleTransferred](obs), append(tnPairs(tnHolders(t, w, 0)), tnPairs(tnHolders(t, w, 1))...)
	}
	evA, titlesA := run([]int{0, 1, 2})
	evB, titlesB := run([]int{2, 1, 0})
	if !reflect.DeepEqual(evA, evB) || !reflect.DeepEqual(titlesA, titlesB) {
		t.Errorf("the two orders of the slice give different events or titles")
	}
	if len(evA) != 3 || evA[0].From != 2 || evA[1].From != 3 || evA[2].Region != 1 {
		t.Errorf("events = %+v, want (home, 2), (home, 3), (yonder, 6)", evA)
	}
}

// TestTransferTitles_ModuleOffAndTooMany verifies ErrModuleOff with the module off and
// ErrOutOfBounds for more than MaxTitleTransfers entries.
//
// Requirement: FR-LND-025
func TestTransferTitles_ModuleOffAndTooMany(t *testing.T) {
	t.Parallel()
	off := newMill(t, oreMill(5, 10, 100))
	if err := off.TransferTitles(nil); !errors.Is(err, ErrModuleOff) {
		t.Errorf("TransferTitles with the module off = %v, want ErrModuleOff", err)
	}
	w := tenureWorld(t, nil)
	if err := w.TransferTitles(make([]TitleTransfer, MaxTitleTransfers+1)); !errors.Is(err, ErrOutOfBounds) {
		t.Errorf("TransferTitles with too many entries = %v, want ErrOutOfBounds", err)
	}
	if err := w.TransferTitles(nil); err != nil {
		t.Errorf("TransferTitles with no entry = %v, want nil", err)
	}
}

// TestSetRentScales_AtomicDefaultClearsEventPerChange verifies the slot of the rent
// scale: the rent in force is the posted rent times the scale, a scale of 10 000 clears
// the key and is not listed in Slots, one RentScaled is emitted for each change in
// ascending key order and none for an unchanged key, and a refused call changes nothing.
//
// Requirement: FR-LND-026, FR-LND-015
func TestSetRentScales_AtomicDefaultClearsEventPerChange(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := tenureWorld(t, func(m *TenureConfig) { m.Terms[0].Rent = 10 }, WithObserver(obs))
	tnEq(t, "Rent", errVal(w.Rent(0, 0)), int64(10))
	if err := w.SetRentScales([]EndowmentScale{{Region: 1, BP: 3_000}, {Region: 0, BP: 5_000}, {Region: 0, Resource: 0, BP: 5_000}}); !errors.Is(err, ErrDuplicateID) {
		t.Fatalf("SetRentScales with a repeated key = %v, want ErrDuplicateID", err)
	}
	if err := w.SetRentScales([]EndowmentScale{{Region: 1, BP: 3_000}, {Region: 0, BP: 5_000}}); err != nil {
		t.Fatalf("SetRentScales error = %v, want nil", err)
	}
	tnEq(t, "Rent after a scale of 5000", errVal(w.Rent(0, 0)), int64(5))
	tnEq(t, "events", of[RentScaled](obs), []RentScaled{{Region: 0, From: 10_000, To: 5_000}, {Region: 1, From: 10_000, To: 3_000}})
	tnEq(t, "Slots().RentScales", errVal(w.Slots()).RentScales, []EndowmentScale{{Region: 0, BP: 5_000}, {Region: 1, BP: 3_000}})
	if err := w.SetRentScales([]EndowmentScale{{Region: 0, BP: 10_000}, {Region: 1, BP: 3_000}}); err != nil {
		t.Fatalf("SetRentScales clearing error = %v, want nil", err)
	}
	tnEq(t, "Slots().RentScales after clearing", errVal(w.Slots()).RentScales, []EndowmentScale{{Region: 1, BP: 3_000}})
	tnEq(t, "RentScaled events", len(of[RentScaled](obs)), 3)
}

// errVal returns the value of a call that the test expects to succeed.
func errVal[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// TestSetRentScales_Refusals verifies the errors of the slot and that a refusal changes
// nothing and emits nothing.
//
// Requirement: FR-LND-026
func TestSetRentScales_Refusals(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := tenureWorld(t, nil, WithObserver(obs))
	cases := []struct {
		name    string
		changes []EndowmentScale
		want    error
	}{
		{"unknown region", []EndowmentScale{{Region: 5, BP: 1}}, ErrUnknownRegion},
		{"unknown resource", []EndowmentScale{{Region: 0, Resource: 3, BP: 1}}, ErrUnknownResource},
		{"above MaxShockBP", []EndowmentScale{{Region: 0, BP: MaxShockBP + 1}}, ErrOutOfBounds},
		{"negative", []EndowmentScale{{Region: 0, BP: -1}}, ErrOutOfBounds},
		{"second entry fails", []EndowmentScale{{Region: 0, BP: 100}, {Region: 1, BP: -5}}, ErrOutOfBounds},
	}
	for _, tc := range cases {
		if err := w.SetRentScales(tc.changes); !errors.Is(err, tc.want) {
			t.Errorf("%s: SetRentScales error = %v, want %v", tc.name, err, tc.want)
		}
	}
	if s, _ := w.Slots(); len(s.RentScales) != 0 || len(of[RentScaled](obs)) != 0 {
		t.Errorf("a refused call left a scale or an event")
	}
	off := newMill(t, oreMill(5, 10, 100))
	if err := off.SetRentScales(nil); !errors.Is(err, ErrModuleOff) {
		t.Errorf("SetRentScales with the module off = %v, want ErrModuleOff", err)
	}
	if err := w.SetRentScales([]EndowmentScale{{Region: 0, BP: 0}}); err != nil {
		t.Errorf("a scale of 0 (free rent) = %v, want nil", err)
	}
	if r, _ := w.Rent(0, 0); r != 0 {
		t.Errorf("Rent with a scale of 0 = %d, want 0", r)
	}
}

// TestTransferTitles_PriceBetweenOneAccount_EventKeepsPriceMovesNoMoney pins spec-land.md
// 6.3.2: a release by the pool is paid by the pool to the pool, the same account, so
// the ledger moves nothing and the event records the agreed price.
func TestTransferTitles_PriceBetweenOneAccount_EventKeepsPriceMovesNoMoney(t *testing.T) {
	t.Parallel()
	obs := &evLog{}
	w := tenureWorld(t, nil, WithObserver(obs))
	total, pool := tnTotalMoney(t, w), tnMoney(t, w, 0)
	if err := w.TransferTitles([]TitleTransfer{tr(0, 0, NoAgent, 40, 25)}); err != nil {
		t.Fatalf("TransferTitles error = %v, want nil", err)
	}
	if tnMoney(t, w, 0) != pool || tnTotalMoney(t, w) != total {
		t.Errorf("pool %d, total %d, want %d and %d unchanged", tnMoney(t, w, 0), tnTotalMoney(t, w), pool, total)
	}
	got := of[TitleTransferred](obs)
	if len(got) != 1 || got[0].Price != 25 || got[0].From != 0 || got[0].To != NoAgent {
		t.Errorf("events = %+v, want one entry with Price 25", got)
	}
}
