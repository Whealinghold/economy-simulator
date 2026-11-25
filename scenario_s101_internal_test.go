package economysim

import (
	"bytes"
	"fmt"
	"testing"
)

// TestScenarioS101_ModuleTakesNoWordWithOrWithoutOffers is the harness test of EMP-S5
// (spec-employment.md 17, FR-EMP-061): over seeds 1 to 10, in a world of farms and hands with
// the audit on, the state of the world's PCG after the labour round (P2.5a) and after the
// labour report (P2.8b) is the state it was in after the position before, so neither takes a
// word, whether the households offer and the employers post (hires form every tick) or nothing
// is offered. The other draw sites key their own streams from the seed, the site, the tick and
// the entity (ADR 0008), so a position that leaves the PCG where it found it cannot move
// another site's draws. A whole world with the module on is not compared with one with it off:
// its cost anchors differ by design (spec-viability.md 4.7), so only the draws are the same.
//
// Setup: 2 farms, 2 hands, one farm vacancy of 2 positions at a bid of 100 against a reserve of
// 10 when offers are made; the script returns no job at all otherwise.
//
// Requirement: FR-EMP-018, FR-EMP-061, NFR-EMP-005
//
// Guard: a draw taken by the round or the report would move the words of every later position
// only in worlds with the module on, which no off golden can see.
func TestScenarioS101_ModuleTakesNoWordWithOrWithoutOffers(t *testing.T) {
	t.Parallel()
	for _, offers := range []bool{true, false} {
		for seed := uint64(1); seed <= 10; seed++ {
			t.Run(fmt.Sprintf("offers %v seed %d", offers, seed), func(t *testing.T) {
				t.Parallel()
				var plan func(v *AgentView) JobRequest
				if offers {
					plan = farmWants(2, 100, 10)
				}
				log := &hireLog{}
				w := hireWorld(t, defaultHire(hireOpts{farms: 2, hands: 2}), plan, WithSeed(seed), WithObserver(log))
				s101TakesNoWord(t, w, seed, 4)
				if offers && len(log.formed) == 0 {
					t.Errorf("seed %d: no hire formed in 4 ticks with offers, want some: the round did not run, the test would prove nothing", seed)
				}
				if !offers && (len(log.formed) != 0 || len(log.ended) != 0) {
					t.Errorf("seed %d: %d hires formed with nothing offered, want 0", seed, len(log.formed))
				}
			})
		}
	}
}

// s101TakesNoWord replays ticks steps of w and, at P2.5a and P2.8b, compares the state of the
// PCG with the one the position before left.
func s101TakesNoWord(t *testing.T, w *World, seed uint64, ticks int) {
	t.Helper()
	var prev []byte
	checked := 0
	for range ticks {
		err := replayStep(w, func(position string) {
			state, err := w.rnd.pcg.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary() error = %v, want nil", err)
			}
			if (position == "P2.5a" || position == "P2.8b") && !bytes.Equal(prev, state) {
				t.Errorf("seed %d tick %d: PCG state after %s = %x, want %x: the position took a word", seed, w.Tick(), position, state, prev)
			}
			if position == "P2.5a" || position == "P2.8b" {
				checked++
			}
			prev = state
		})
		if err != nil {
			t.Fatalf("seed %d: replayStep() error = %v, want nil", seed, err)
		}
	}
	if checked != 2*ticks {
		t.Errorf("seed %d: compared %d positions, want %d (P2.5a and P2.8b of each tick)", seed, checked, 2*ticks)
	}
}
