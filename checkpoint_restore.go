package economysim

import (
	"bytes"
	"cmp"
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math"
	"strings"

	"github.com/Whealinghold/economy-sim/internal/codec"
	"github.com/Whealinghold/economy-sim/internal/ledger"
	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// This file is the restore: RestoreWorld in the order of validation of
// spec-checkpoint.md 6.2, the framing of the sections, the match of the
// extensions, the cross-checks of step 7 and the helpers the concept files' loadX
// share (plan 2-2 Phase 5; D251 to D256). The sections' readers are in the concept
// files, the table that names them is checkpoint_sections.go.

// restoreCtx is what the readers of the sections pass between them: the header's
// tick and the flag of tag 1, which the ledger needs when it reads tag 2, and the
// entries of tag 34 that step 11 loads.
type restoreCtx struct {
	tick         int64
	construction bool
	entries      []extRecord
	have         []extEntry // the registered Stateful extensions, the same set step 4 matched
}

// RestoreWorld builds a new world from cfg, the options and the checkpoint c, or
// returns an error and no world (spec-checkpoint.md 6). It applies the options and
// validates the configuration exactly as NewWorld does, requires the configuration
// to be the one the checkpoint was taken from, matches the extensions, reads every
// section into new structures checking the rules of the encoding and every
// reference, audits the ledger whether or not WithAudit was given, and only then
// loads the state of the Stateful extensions. A failure at any step before the last
// touches nothing the consumer owns; a failure in a LoadState leaves that instance
// as the extension left it, and the instances loaded before it (in kind, name order)
// already hold the checkpoint's state. The error is ErrCheckpointVersion, ErrCheckpointCorrupt,
// ErrCheckpointConfig, ErrCheckpointExtension, ErrOptionContext (WithSeed is refused:
// the checkpoint holds the seed, and WithReseed is the way to another continuation)
// or what NewWorld returns for the configuration.
//
// The world is at the tick of the checkpoint with the effective seed of its
// header, or of the reseed. It emits no WorldCreated and no AgentSpawned; an
// observer given to the restore receives WorldRestored first, and nothing at all
// when the restore fails.
func RestoreWorld(cfg Config, c Checkpoint, opts ...Option) (*World, error) {
	f, err := parseFrame(c.b)
	if err != nil {
		return nil, err
	}
	w, o, err := restoreShell(cfg, f, opts)
	if err != nil {
		return nil, err
	}
	rs := &restoreCtx{tick: f.info.Tick}
	if err := w.restoreState(rs, f.sections); err != nil {
		return nil, err
	}
	w.announceRestore(f.info, o)
	return w, nil
}

// restoreShell is steps 2 and 3: the options (WithSeed is refused, the effective
// seed is the header's or the reseeded one), the configuration through the one
// path construction uses, and the comparison of the hashes.
func restoreShell(cfg Config, f frame, opts []Option) (*World, options, error) {
	o, err := applyOptions(opts)
	if err != nil {
		return nil, o, err
	}
	if o.seedGiven {
		return nil, o, fmt.Errorf("WithSeed on RestoreWorld, the checkpoint holds the seed: %w", ErrOptionContext)
	}
	o.seed = f.info.Seed
	if o.reseedGiven {
		o.seed = reseed(o.seed, o.reseed)
	}
	w, err := buildShell(cfg, o)
	if err != nil {
		return nil, o, err
	}
	if w.hash != f.info.ConfigHash {
		return nil, o, fmt.Errorf("%w: checkpoint hash %x, configuration hash %x, encoding version %d",
			ErrCheckpointConfig, f.info.ConfigHash, w.hash, f.hashVersion)
	}
	return w, o, nil
}

// restoreState is steps 4 to 11 on a shell: the extensions are matched, the
// sections are read, the cross-checks and the audit run, and last the extensions
// load their state. A failure at any step before the last touches nothing the
// consumer owns.
func (w *World) restoreState(rs *restoreCtx, sections []byte) error {
	bodies, err := splitSections(&w.cfg, sections)
	if err != nil {
		return err
	}
	if rs.entries, err = w.matchExtensions(rs, bodies); err != nil {
		return err
	}
	w.allocate()
	if err := w.loadSections(rs, bodies); err != nil {
		return err
	}
	if err := w.checkpointCross(); err != nil {
		return err
	}
	w.finishScratch()
	w.seedWage = seedWages(&w.cfg)
	if err := auditFailure(w.Tick(), w.ledger.Audit()); err != nil {
		return fmt.Errorf("%w: %w", ErrCheckpointCorrupt, err)
	}
	return loadExtensionState(rs.have, rs.entries)
}

// sectionBody is the body of one section the configuration expects, with its row.
type sectionBody struct {
	row  *section
	body []byte
}

// splitSections cuts the bytes after the header into the bodies of the expected
// sections (spec-checkpoint.md 5.3.1): the next tag must be the next expected one,
// its length at most the bytes remaining, and nothing may follow the last. It
// reads no body.
func splitSections(cfg *Config, b []byte) ([]sectionBody, error) {
	r := codec.NewReader(b)
	var out []sectionBody
	for i := range checkpointSections {
		s := &checkpointSections[i]
		// Excluded from coverage (testing.md 9.5): every row of the table is present
		// with no condition until a module appends its section, and that change
		// tests the guard.
		if s.present != nil && !s.present(cfg) {
			continue
		}
		at := r.Offset()
		tag := r.U()
		body := r.Bytes()
		if err := r.Err(); err != nil {
			return nil, fmt.Errorf("%w: section %d (%s): %w", ErrCheckpointCorrupt, s.tag, s.name, err)
		}
		if tag != s.tag {
			return nil, fmt.Errorf("%w: byte %d: tag %d where section %d (%s) was expected", ErrCheckpointCorrupt, at, tag, s.tag, s.name)
		}
		out = append(out, sectionBody{row: s, body: body})
	}
	if err := r.Done(); err != nil {
		return nil, fmt.Errorf("%w: after the last section: %w", ErrCheckpointCorrupt, err)
	}
	return out, nil
}

// loadSections reads every section in the order of the table, each from exactly its
// bytes (step 6). A section that ends early, runs over or breaks a rule is
// ErrCheckpointCorrupt naming its tag.
func (w *World) loadSections(rs *restoreCtx, bodies []sectionBody) error {
	for _, sb := range bodies {
		r := codec.NewReader(sb.body)
		err := sb.row.load(w, rs, r)
		if err == nil {
			err = r.Done()
		}
		if err != nil {
			return fmt.Errorf("%w: section %d (%s): %w", ErrCheckpointCorrupt, sb.row.tag, sb.row.name, err)
		}
	}
	return nil
}

// extRecord is one entry of tag 34 as read: the kind, the name, the version and
// the state bytes, which alias the checkpoint.
type extRecord struct {
	kind    uint64
	name    string
	version uint32
	state   []byte
}

// kindsOfExtension is the number of kinds an entry may have: the strategy, the
// spawner, the clearing and the reserved policy (spec-checkpoint.md 4.2.4).
const kindsOfExtension = 4

// readExtensions reads tag 34: the entries in strictly ascending (kind, name).
func readExtensions(r *codec.Reader) []extRecord {
	n := r.Count(4)
	var out []extRecord
	for range n {
		if r.Err() != nil {
			break
		}
		e := extRecord{kind: r.Index(kindsOfExtension)}
		e.name = string(r.Bytes())
		v := r.U()
		if v > math.MaxUint32 {
			r.Failf("extension version %d is above %d", v, uint64(math.MaxUint32))
		}
		e.version, e.state = uint32(v), r.Bytes()
		if k := len(out) - 1; k >= 0 && r.Err() == nil && compareExt(out[k].kind, out[k].name, e.kind, e.name) >= 0 {
			r.Failf("extension entry %q (kind %d) is not above the one before it", e.name, e.kind)
		}
		out = append(out, e)
	}
	return out
}

// compareExt orders entries by kind and then by the name as a byte string, the
// order of spec-checkpoint.md 7.3.1.
func compareExt(ak uint64, an string, bk uint64, bn string) int {
	return cmp.Or(cmp.Compare(ak, bk), strings.Compare(an, bn))
}

// matchExtensions is step 4: the registered extensions that implement Stateful
// are the entries of tag 34 by (kind, name), and every strategy name in the table
// of tag 3 is registered. It returns the entries for step 11.
func (w *World) matchExtensions(rs *restoreCtx, bodies []sectionBody) ([]extRecord, error) {
	var recs []extRecord
	for _, sb := range bodies {
		switch sb.row.tag {
		case tagAgents:
			if err := w.checkStrategyNames(sb.body); err != nil {
				return nil, err
			}
		case tagExtensions:
			r := codec.NewReader(sb.body)
			recs = readExtensions(r)
			if err := r.Done(); err != nil {
				return nil, fmt.Errorf("%w: section %d (%s): %w", ErrCheckpointCorrupt, sb.row.tag, sb.row.name, err)
			}
		}
	}
	rs.have = w.statefulEntries()
	return recs, sameExtensions(rs.have, recs)
}

// sameExtensions requires the registered Stateful extensions and the entries, both
// in (kind, name) order, to be the same set.
func sameExtensions(have []extEntry, recs []extRecord) error {
	for i := 0; i < len(have) || i < len(recs); i++ {
		switch {
		case i >= len(recs) || (i < len(have) && compareExt(have[i].kind, have[i].name, recs[i].kind, recs[i].name) < 0):
			return fmt.Errorf("%w: %q (kind %d) is registered and Stateful but has no entry", ErrCheckpointExtension, have[i].name, have[i].kind)
		case i >= len(have) || compareExt(have[i].kind, have[i].name, recs[i].kind, recs[i].name) > 0:
			return fmt.Errorf("%w: entry %q (kind %d) is not registered or not Stateful", ErrCheckpointExtension, recs[i].name, recs[i].kind)
		}
	}
	return nil
}

// checkStrategyNames reads the table of strategy names at the head of tag 3 and
// requires each to be registered. A table it cannot read is corrupt.
func (w *World) checkStrategyNames(body []byte) error {
	r := codec.NewReader(body)
	names := readStrategyNames(r)
	if err := r.Err(); err != nil {
		return fmt.Errorf("%w: section %d (Agents): %w", ErrCheckpointCorrupt, tagAgents, err)
	}
	for _, name := range names {
		if !w.strategyRegistered(name) {
			return fmt.Errorf("%w: strategy %q of an agent: %w", ErrCheckpointExtension, name, ErrUnknownStrategy)
		}
	}
	return nil
}

// strategyRegistered reports whether a strategy name resolves: one of the two
// built-in names or one registered with WithStrategy.
func (w *World) strategyRegistered(name string) bool {
	if name == defaultStrategyName || name == directedStrategyName {
		return true
	}
	_, ok := w.reg.strategies.find(name)
	return ok
}

// loadExtensionState is step 11: LoadState of each Stateful extension in (kind,
// name) order, with the version and a copy of the bytes of its entry (the
// extension must not keep the slice, and a copy keeps a faulty one from altering
// the checkpoint).
func loadExtensionState(have []extEntry, recs []extRecord) error {
	for i, e := range have {
		if err := e.s.LoadState(recs[i].version, bytes.Clone(recs[i].state)); err != nil {
			return fmt.Errorf("%w: LoadState of %q (kind %d): %w", ErrCheckpointExtension, e.name, e.kind, err)
		}
	}
	return nil
}

// loadExtensions is the reader of tag 34 in the table. The entries were read and
// checked whole at step 4, before any section, and their state is loaded last at
// step 11 (spec-checkpoint.md 6.2.1); here the section is only consumed.
func (w *World) loadExtensions(_ *restoreCtx, r *codec.Reader) error {
	r.Raw(r.Remaining())
	return nil
}

// loadLedger is tag 2: the ledger reads its own section, then the estates are
// attached to the accounts it holds.
func (w *World) loadLedger(rs *restoreCtx, r *codec.Reader) error {
	if err := w.ledger.ReadState(r, rs.tick, rs.construction); err != nil {
		return err
	}
	e, err := attachEstates(w.ledger, len(w.cfg.Regions))
	if err != nil {
		return err
	}
	w.estates = e
	return nil
}

// loadSkills is tag 10, a zero-row skeleton: four counts that must be 0 until the
// skills module is built (plan 2-2 D244).
func (w *World) loadSkills(_ *restoreCtx, r *codec.Reader) error {
	for range 4 {
		if r.Count(1) != 0 {
			r.Failf("rows in the skills section, whose module is not built")
		}
	}
	return nil
}

// checkpointCross is step 7, the checks that need two sections. Modules append
// theirs here in the order of spec-checkpoint.md 6.2: every account after the
// estates is an Agent account, and an open one is the account of the live agent
// that is its owner and names it; a live agent's (region, role) has its
// had-an-agent bit set.
func (w *World) checkpointCross() error {
	regions, roles := len(w.cfg.Regions), len(w.cfg.Roles)
	first := w.estates.accounts[0]
	for id := first + regions; id < w.ledger.Len(); id++ {
		v, err := w.ledger.Account(id)
		if err != nil || v.Kind != ledger.KindAgent {
			return fmt.Errorf("%w: account %d is not an Agent account, the only kind after the estates in this configuration", ErrCheckpointCorrupt, id)
		}
		if v.Closed {
			continue
		}
		if v.Owner < regions || v.Owner >= len(w.agents) || w.agents[v.Owner].weight < 1 || w.agents[v.Owner].account != id {
			return fmt.Errorf("%w: open account %d is the account of no live agent", ErrCheckpointCorrupt, id)
		}
	}
	var missing error
	w.eachLive(func(a *agent) {
		// The account exists: step 7's loop above and the agent record check own it.
		v, _ := w.ledger.Account(a.account)
		if cell := v.Region*roles + int(a.role); !w.state.hadAgent[cell] && missing == nil {
			missing = fmt.Errorf("%w: agent %d lives in region %d, role %d, whose had-an-agent bit is not set", ErrCheckpointCorrupt, a.id, v.Region, a.role)
		}
	})
	return missing
}

// announceRestore is step 12: the configuration's warnings at Warn, once each, the
// Info line "world restored", and WorldRestored as the first event, with the
// snapshot at the restored boundary built only when an observer was supplied.
func (w *World) announceRestore(info CheckpointInfo, o options) {
	ctx := context.Background()
	w.logWarnings(ctx, w.cfg.Warnings())
	if w.log.Enabled(ctx, slog.LevelInfo) {
		w.log.LogAttrs(ctx, slog.LevelInfo, "world restored",
			slog.Int64(logkeys.Tick, w.Tick()), slog.Uint64(logkeys.Seed, w.state.seed),
			slog.String(logkeys.ConfigHash, hex.EncodeToString(w.hash[:8])),
			slog.Int64(logkeys.Agents, w.live.agents), slog.Int64(logkeys.Households, w.live.households),
			slog.Int64(logkeys.Format, int64(info.Format)), slog.Int64(logkeys.Bytes, int64(info.Bytes)))
	}
	if !w.obs.supplied {
		return
	}
	snap, err := w.Snapshot()
	// Excluded from coverage (testing.md 9.5): the snapshot of a world that passed
	// the audit cannot fail, and the branch only names the violated invariant.
	if err != nil {
		invariant("the snapshot of a restored world failed: " + err.Error())
	}
	w.obs.deliver(WorldRestored{
		Tick: w.Tick(), ConfigHash: w.hash, Format: info.Format,
		Reseeded: o.reseedGiven, Salt: o.reseed, Names: namesOf(&w.cfg), Snapshot: snap,
	})
}

// readSparse reads the rows writeSparse wrote into vals, which are all 0 and
// indexed by region * roles + role: a count, then (region, role, value) in strictly
// ascending key order, the value from 1 to limit (a row at the default is corrupt, R10;
// a run counts ticks, so the caller passes the tick as limit).
func readSparse(r *codec.Reader, vals []int64, regions, roles int, limit int64) {
	readSparseRows(r, vals, 3, limit, func() (uint64, string) {
		region, role := r.Index(uint64(regions)), r.Index(uint64(roles))
		return region*uint64(roles) + role, fmt.Sprintf("region %d, role %d", region, role)
	})
}

// readSparseByRegion is readSparse for a run kept per region: (region, value).
func readSparseByRegion(r *codec.Reader, vals []int64, limit int64) {
	readSparseRows(r, vals, 2, limit, func() (uint64, string) {
		region := r.Index(uint64(len(vals)))
		return region, fmt.Sprintf("region %d", region)
	})
}

// readSparseRows is the loop of both: a count of rows of at least minBytes, each a
// key from readKey (with its text for an error) and a value from 1 to limit, the
// keys strictly ascending.
func readSparseRows(r *codec.Reader, vals []int64, minBytes int, limit int64, readKey func() (uint64, string)) {
	n := r.Count(minBytes)
	prev := uint64(0)
	for i := range n {
		key, what := readKey()
		v := r.Bounded("run", limit)
		if r.Err() != nil {
			return
		}
		if v == 0 || (i > 0 && key <= prev) {
			r.Failf("sparse row (%s) is at its default or out of order", what)
			return
		}
		vals[key], prev = v, key
	}
}
