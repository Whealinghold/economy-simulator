package economysim_test

import (
	"fmt"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// savingS143Shares are the precaution shares of the grower in S143, in basis points of the drawdown.
var savingS143Shares = []int64{0, 10_000, 20_000}

// TestScenarioS143_ThePrecautionBufferOfAProducerWithALongJob reports SAV-S2 of
// spec-saving-and-investment.md 14.3 on pack one with the saving overlay, 200 ticks, audit on: the
// grower (a 60-tick job, so months between its sales) at PrecautionShareBP 0, 10 000 and 20 000. The
// identities are asserted (the audit holds at every step, and the buffer of the event is exactly 0
// at share 0 and above 0 at a share above 0); the effect is reported: the sampled unfed household
// count and the buffer held, by share.
//
// Arithmetic: the buffer is min(cap * upkeep, share * drawdown / 10 000) over the reserve (spec
// 4.2), so a share of 0 gives the flat reserve and no excess. Whether more share means fewer unfed
// household-ticks, and whether the aggregate trough is no worse beyond one window, is what the
// report shows for STY; the placeholders of the overlay assert none (N-015, N-027). The cut of the
// partners' income of the spec's variant needs a second consumer of the grower's money and is not
// run on this pack.
//
// Requirement: FR-SAV-004, FR-SAV-006, FR-SAV-007, FR-SAV-033
//
// Guard: a buffer that exists with no share, or that never exists with one, fails here.
func TestScenarioS143_ThePrecautionBufferOfAProducerWithALongJob(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end scenario, skipped with -short")
	}
	savingScenario(t, "S143 one+saving the grower's buffer by share, 200 ticks", scenarioSeedCount(), func(t *testing.T, rep *scenReport, seed uint64) {
		for _, share := range savingS143Shares {
			cfg := savingPack(t, func(c *economysim.Config) { roleOf(c, "grower").Saver.PrecautionShareBP = share })
			st := savingRun(t, cfg, seed, 200)
			name := fmt.Sprintf("share %d", share)
			rep.num(seed, name+": unfed household samples", st.UnfedSamples)
			rep.num(seed, name+": buffer, summed over the events", int64(st.Buffer))
			ok := (share == 0) == (st.Buffer == 0)
			checkExact(t, rep, "S143", "one", seed, name+": the buffer exists exactly when the share is above 0", ok, "buffer %d at share %d", st.Buffer, share)
		}
	})
}
