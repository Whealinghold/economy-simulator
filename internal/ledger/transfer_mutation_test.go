package ledger_test

import (
	"testing"

	"github.com/Whealinghold/economy-sim/internal/ledger"
)

// TestTransferReserved_ExactReservation verifies that a marked transfer of
// exactly the reserved amount is accepted and consumes the whole reservation,
// while one unit more is refused with ErrReservationExceeds.
//
// Requirement: FR-LED-017
func TestTransferReserved_ExactReservation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		amount       int64
		wantErr      error
		wantBalance  int64
		wantReserved int64
	}{
		// 100 held, 10 reserved: 10 drawn leaves 90 held and 0 reserved.
		{name: "amount equals reservation", amount: 10, wantBalance: 90, wantReserved: 0},
		// 11 asked against 10 reserved changes nothing.
		{name: "one above reservation", amount: 11, wantErr: ledger.ErrReservationExceeds, wantBalance: 100, wantReserved: 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFix(t, testMaxMoney)
			if err := f.l.ReserveMoney(f.a0, 10); err != nil {
				t.Fatalf("ReserveMoney: %v", err)
			}
			if err := f.l.TransferReserved(f.a0, f.a1, tt.amount, ledger.Trade); err != tt.wantErr {
				t.Fatalf("TransferReserved: got %v, want %v", err, tt.wantErr)
			}
			v, _ := f.l.Account(f.a0)
			if v.Balance != tt.wantBalance || v.Reserved != tt.wantReserved {
				t.Errorf("balance, reserved = %d, %d, want %d, %d", v.Balance, v.Reserved, tt.wantBalance, tt.wantReserved)
			}
		})
	}
}
