package economysim_test

import (
	"encoding/binary"
	"errors"
	"math/rand/v2"
	"slices"
	"testing"

	economysim "github.com/Whealinghold/economy-sim"
)

// This file tests the checkpoint sections of the information module (plan 2-13
// Phase 6): tag 16, the price record, and tag 22, the empty integration section.

// Tags of the information module's sections.
const (
	informationSectionTag uint64 = 16
	rumoursSectionTag     uint64 = 22
)

// packTwoInformation is the cold pack two with its trade overlay, the edges.
var packTwoInformation = append(slices.Clone(packLayers["two"]), tradeOverlay("two"))

// rumourReader is the default strategy whose limit prices move with the rumours it
// is told, so that a wrong record, a wrong draw or a wrong vintage changes an order
// and shows in the events. It keeps no state (architecture.md 7.7).
type rumourReader struct{}

func (rumourReader) Work(v *economysim.AgentView, rng *rand.Rand) economysim.JobRequest {
	return economysim.DefaultStrategy{}.Work(v, rng)
}

func (rumourReader) Revise(o *economysim.OutcomeView, rng *rand.Rand) economysim.Belief {
	return economysim.DefaultStrategy{}.Revise(o, rng)
}

func (rumourReader) Trade(v *economysim.AgentView, rng *rand.Rand, p *economysim.TradePlan) {
	economysim.DefaultStrategy{}.Trade(v, rng, p)
	var shift int64
	for _, r := range v.Rumours {
		shift += int64(r.Price)%7 + r.Lag
	}
	for i := range p.Intents {
		p.Intents[i].Limit += economysim.Price(shift)
	}
}

// informationPack returns pack two with the information module on, every role
// informed and following rumourReader, an information speed of 50 over the edge of
// 100 (a lag of 3), the given error and horizon, and the option that registers the
// strategy.
func informationPack(t testing.TB, errorBP, maxAge int64) (economysim.Config, []economysim.Option) {
	t.Helper()
	cfg := loadValidPack(t, packTwoInformation...)
	cfg.Modules.Information = &economysim.Information{InfoSpeed: 50, RumourErrorBP: errorBP, RumourMaxAge: maxAge}
	for i := range cfg.Roles {
		cfg.Roles[i].Informed, cfg.Roles[i].Strategy = true, "reader"
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
	return cfg, []economysim.Option{economysim.WithStrategy("reader", rumourReader{})}
}

// informationCheckpoint returns the bytes of a checkpoint of the information pack
// taken after ticks ticks, with the configuration and options that restore it.
func informationCheckpoint(t *testing.T, ticks int, errorBP, maxAge int64) ([]byte, economysim.Config, []economysim.Option) {
	t.Helper()
	cfg, opts := informationPack(t, errorBP, maxAge)
	w, err := economysim.NewWorld(cfg, append([]economysim.Option{economysim.WithSeed(1)}, opts...)...)
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	step(t, w, ticks)
	return mustMarshal(t, mustCheckpoint(t, w)), cfg, opts
}

// tagsOf lists the tags of the sections of a checkpoint in order.
func tagsOf(t *testing.T, raw []byte) []uint64 {
	t.Helper()
	var tags []uint64
	for _, s := range decodeFrame(t, raw).sections {
		tags = append(tags, s.tag)
	}
	return tags
}

// TestCheckpoint_InformationTags verifies that a world with the module on writes
// tags 16 and 22 in tag order (after 15, and tag 22 before the extensions) and a
// world with it off writes neither, so the bytes of the module-free checkpoints do
// not move.
//
// Requirement: FR-INF-029, FR-CHK-004
func TestCheckpoint_InformationTags(t *testing.T) {
	t.Parallel()
	raw, _, _ := informationCheckpoint(t, 3, 500, 10)
	got := tagsOf(t, raw)
	for _, tag := range []uint64{informationSectionTag, rumoursSectionTag} {
		if !slices.Contains(got, tag) {
			t.Errorf("tags of a world with the module on = %v, want %d", got, tag)
		}
	}
	if !slices.IsSorted(got) {
		t.Errorf("tags = %v, want ascending", got)
	}
	cfg, opts := informationPack(t, 500, 10)
	cfg.Modules.Information = nil
	for i := range cfg.Roles {
		cfg.Roles[i].Informed = false
	}
	w, err := economysim.NewWorld(cfg, opts...)
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	for _, tag := range tagsOf(t, mustMarshal(t, mustCheckpoint(t, w))) {
		if tag == informationSectionTag || tag == rumoursSectionTag {
			t.Errorf("a world with the module off wrote tag %d, want neither 16 nor 22", tag)
		}
	}
}

// infoRows splits the body of tag 16 into, for each source, its entries as
// (Price, Mean, Age); ok is false when the body does not parse whole.
func infoRows(body []byte) (rows [][][3]uint64, ok bool) {
	next := func() (uint64, bool) {
		v, n := binary.Uvarint(body)
		if n <= 0 {
			return 0, false
		}
		body = body[n:]
		return v, true
	}
	for len(body) > 0 {
		n, ok := next()
		if !ok {
			return nil, false
		}
		var row [][3]uint64
		for range n {
			var e [3]uint64
			for k := range e {
				if e[k], ok = next(); !ok {
					return nil, false
				}
			}
			row = append(row, e)
		}
		rows = append(rows, row)
	}
	return rows, true
}

// infoBody encodes rows as tag 16 does.
func infoBody(rows [][][3]uint64) []byte {
	var b []byte
	for _, row := range rows {
		b = binary.AppendUvarint(b, uint64(len(row)))
		for _, e := range row {
			for _, v := range e {
				b = binary.AppendUvarint(b, v)
			}
		}
	}
	return b
}

// TestCheckpoint_InformationCorrupt verifies that a restore refuses, as corrupt,
// a section 16 with a count above the depth, unequal counts, a count that is not
// the ticks run, a price of 0, a price or a mean above MaxPrice, an age below 0
// (a value above the int64 range), trailing bytes and missing bytes, and a section
// 22 with a row.
//
// Arithmetic: a horizon of 4 and 6 ticks run give every source 4 entries; 5 is
// above the depth, 3 is not the ticks run (6 capped at 4). The age of -1 is
// written as 2^64 - 1.
//
// Requirement: FR-INF-029, FR-CHK-016
func TestCheckpoint_InformationCorrupt(t *testing.T) {
	t.Parallel()
	raw, cfg, opts := informationCheckpoint(t, 6, 500, 4)
	good, ok := infoRows(decodeFrame(t, raw).sectionOf(t, informationSectionTag))
	if !ok || len(good) == 0 || len(good[0]) != 4 {
		t.Fatalf("section 16 of a checkpoint at tick 6 with depth 4 = %d sources of %v entries, want sources of 4", len(good), len(good[0]))
	}
	mutate := func(f func(rows [][][3]uint64) [][][3]uint64) []byte {
		rows := make([][][3]uint64, len(good))
		for i := range good {
			rows[i] = slices.Clone(good[i])
		}
		return infoBody(f(rows))
	}
	cases := []struct {
		name string
		body []byte
	}{
		{"a count above the depth", mutate(func(r [][][3]uint64) [][][3]uint64 { r[0] = append(r[0], r[0][0]); return r })},
		{"unequal counts", mutate(func(r [][][3]uint64) [][][3]uint64 { r[1] = r[1][1:]; return r })},
		{"every count below the ticks run", mutate(func(r [][][3]uint64) [][][3]uint64 {
			for i := range r {
				r[i] = r[i][1:]
			}
			return r
		})},
		{"a price of zero", mutate(func(r [][][3]uint64) [][][3]uint64 { r[0][2][0] = 0; return r })},
		{"a price above the bound", mutate(func(r [][][3]uint64) [][][3]uint64 { r[0][2][0] = economysim.MaxPrice + 1; return r })},
		{"a mean above the bound", mutate(func(r [][][3]uint64) [][][3]uint64 { r[0][2][1] = economysim.MaxPrice + 1; return r })},
		{"a negative age", mutate(func(r [][][3]uint64) [][][3]uint64 { r[0][2][2] = 1<<64 - 1; return r })},
		{"trailing bytes", append(mutate(func(r [][][3]uint64) [][][3]uint64 { return r }), 0)},
		{"missing bytes", infoBody(good)[:len(infoBody(good))-1]},
		{"empty", nil},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := reframe(t, raw, func(secs []section) []section {
				return editSection(secs, informationSectionTag, func([]byte) []byte { return tt.body })
			})
			if _, err := economysim.RestoreWorld(cfg, c, opts...); !errors.Is(err, economysim.ErrCheckpointCorrupt) {
				t.Errorf("RestoreWorld() error = %v, want ErrCheckpointCorrupt", err)
			}
		})
	}
	t.Run("a row in the integration section", func(t *testing.T) {
		t.Parallel()
		c := reframe(t, raw, func(secs []section) []section {
			return editSection(secs, rumoursSectionTag, func([]byte) []byte { return []byte{1} })
		})
		if _, err := economysim.RestoreWorld(cfg, c, opts...); !errors.Is(err, economysim.ErrCheckpointCorrupt) {
			t.Errorf("RestoreWorld() error = %v, want ErrCheckpointCorrupt", err)
		}
	})
	t.Run("the good section restores", func(t *testing.T) {
		t.Parallel()
		c := reframe(t, raw, func(secs []section) []section { return secs })
		if _, err := economysim.RestoreWorld(cfg, c, opts...); err != nil {
			t.Errorf("RestoreWorld(unedited) error = %v, want nil", err)
		}
	})
}

// FuzzInformationSection fuzzes the decoder of tag 16: the body is the input, the
// rest of a valid checkpoint is kept and the trailer is recomputed. No input
// panics; a refusal is ErrCheckpointCorrupt; an accepted input is a world that
// passes its audit and saves the same section.
//
// Requirement: NFR-CHK-003, FR-CHK-016, FR-INF-029
func FuzzInformationSection(f *testing.F) {
	cfg, opts := informationPack(f, 500, 4)
	w, err := economysim.NewWorld(cfg, append([]economysim.Option{economysim.WithSeed(1)}, opts...)...)
	if err != nil {
		f.Fatalf("NewWorld() error = %v, want nil", err)
	}
	for range 6 {
		if err := w.Step(); err != nil {
			f.Fatalf("Step() error = %v, want nil", err)
		}
	}
	c, err := w.Checkpoint()
	if err != nil {
		f.Fatalf("Checkpoint() error = %v, want nil", err)
	}
	raw, err := c.MarshalBinary()
	if err != nil {
		f.Fatalf("MarshalBinary() error = %v, want nil", err)
	}
	for _, s := range decodeFrame(f, raw).sections {
		if s.tag == informationSectionTag {
			f.Add(s.body)
		}
	}
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Fuzz(func(t *testing.T, body []byte) {
		c := reframe(t, raw, func(secs []section) []section {
			return editSection(secs, informationSectionTag, func([]byte) []byte { return slices.Clone(body) })
		})
		w, err := economysim.RestoreWorld(cfg, c, opts...)
		if err != nil {
			if !errors.Is(err, economysim.ErrCheckpointCorrupt) {
				t.Fatalf("RestoreWorld(section %x) error = %v, want ErrCheckpointCorrupt", body, err)
			}
			return
		}
		if err := w.Audit(); err != nil {
			t.Fatalf("Audit() of the world restored from section %x error = %v, want nil", body, err)
		}
		for _, s := range decodeFrame(t, mustMarshal(t, mustCheckpoint(t, w))).sections {
			if s.tag == informationSectionTag && !slices.Equal(s.body, body) {
				t.Fatalf("restore then save of section %x gave %x, want the same bytes", body, s.body)
			}
		}
	})
}

// TestFork_Information verifies the fork-at-N of the cold pack two with the
// information module alone on, every role informed, an error of 500 and a strategy
// whose orders move with the rumours (so a wrong record or a wrong draw shows in the
// events): at ticks 0, 1, inside a job, across a season change, and after the
// horizon of 10 is full, the original and two restores give the same events,
// snapshot, audit and checkpoint bytes. The draws taken after the boundary are the
// candidates of the informed views, so the equality of the events carries them.
//
// Requirement: FR-INF-029, FR-INF-046, FR-CHK-050, FR-CHK-052, FR-CHK-053
func TestFork_Information(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg, opts := informationPack(t, 500, 10)
	for _, cell := range []struct {
		name string
		n, m int
	}{
		{"before the first step", 0, 30}, {"after one step", 1, 30}, {"inside the horizon", 5, 30},
		{"at the horizon", 10, 30}, {"inside a job", 100, 30}, {"across a season change", 150, 30}, {"late", 80, 120},
	} {
		t.Run(cell.name, func(t *testing.T) {
			t.Parallel()
			forkAtN(t, forkCase{
				Name: "information", Cfg: cfg, Seed: 1, N: cell.n, M: cell.m, Logs: true,
				Opts: func() []economysim.Option { return opts },
				Inspect: func(t *testing.T, restored, _ []economysim.Event) {
					var served int
					for _, e := range restored {
						if r, ok := e.(economysim.RumoursServed); ok && r.Offered > 0 {
							served++
						}
					}
					if cell.n+cell.m > 20 && served == 0 {
						t.Errorf("N %d: the restored world served no rumour, want the module to be at work", cell.n)
					}
				},
			})
		})
	}
}

// TestFork_InformationReseed verifies that a restore with a salt keeps the price
// record of the checkpoint (the section 16 of the restored world's own checkpoint is
// the original's) and changes the draws after it, so the rumoured errors, and with
// them the orders the strategy moves by the rumours, are not the original's; and
// that two restores with the same salt agree (forkAtN).
//
// Requirement: FR-INF-029, FR-INF-014, FR-CHK-056
func TestFork_InformationReseed(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("end-to-end test, skipped with -short")
	}
	cfg, opts := informationPack(t, 500, 10)
	const n, m = 30, 20
	salt := uint64(7)
	limits := func(rec *recorder) (out []economysim.Price) {
		for _, e := range rec.events {
			if o, ok := e.(economysim.OrderPlaced); ok {
				out = append(out, o.Limit)
			}
		}
		return out
	}
	plainRec, reseedRec := &recorder{}, &recorder{}
	plain, err := economysim.NewWorld(cfg, append([]economysim.Option{economysim.WithSeed(1)}, opts...)...)
	if err != nil {
		t.Fatalf("NewWorld() error = %v, want nil", err)
	}
	step(t, plain, n)
	ck := mustCheckpoint(t, plain)
	reseeded, err := economysim.RestoreWorld(cfg, ck, append([]economysim.Option{economysim.WithReseed(salt), economysim.WithObserver(reseedRec)}, opts...)...)
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	record := func(w *economysim.World) []byte {
		return decodeFrame(t, mustMarshal(t, mustCheckpoint(t, w))).sectionOf(t, informationSectionTag)
	}
	if !slices.Equal(record(plain), record(reseeded)) {
		t.Error("the restored world's price record differs from the checkpoint's, want a reseed to leave it")
	}
	// The original goes on, watched from here by a second restore without a salt.
	same, err := economysim.RestoreWorld(cfg, ck, append([]economysim.Option{economysim.WithObserver(plainRec)}, opts...)...)
	if err != nil {
		t.Fatalf("RestoreWorld() error = %v, want nil", err)
	}
	step(t, same, m)
	step(t, reseeded, m)
	if a, b := limits(plainRec), limits(reseedRec); len(a) == 0 || slices.Equal(a, b) {
		t.Errorf("the order limits after a reseed are the original's (%d orders), want the errors of the rumours to differ", len(a))
	}
	forkAtN(t, forkCase{
		Name: "information reseed", Cfg: cfg, Seed: 1, N: n, M: m, Reseed: &salt,
		Opts: func() []economysim.Option { return opts },
	})
}
