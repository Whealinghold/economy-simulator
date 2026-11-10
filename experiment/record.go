package experiment

import (
	economysim "github.com/Whealinghold/economy-sim"
)

// Status is the outcome of one run: a closed set (spec-experiments.md 9.2,
// 13.4). Adding a member raises the result Format.
type Status string

// The statuses.
const (
	// StatusOK is a run that completed.
	StatusOK Status = "ok"
	// StatusUnresolved is a document the loader refused: no world was built.
	StatusUnresolved Status = "unresolved"
	// StatusInvalidConfig is a Prepare error or a configuration that fails
	// validation: no world was built.
	StatusInvalidConfig Status = "invalid_config"
	// StatusWorld is any other error of NewWorld or RestoreWorld.
	StatusWorld Status = "world"
	// StatusScript is an error out of a Step.Do; the world is healthy.
	StatusScript Status = "script"
	// StatusInvalidTrade is a tick whose clearing returned an invalid trade list.
	StatusInvalidTrade Status = "invalid_trade"
	// StatusAudit is an audit that found a violated identity.
	StatusAudit Status = "audit"
	// StatusCheckpoint is an error of Checkpoint on a healthy world.
	StatusCheckpoint Status = "checkpoint"
	// StatusStep is any other error of a tick.
	StatusStep Status = "step"
	// StatusMissing is a record absent from an assembly.
	StatusMissing Status = "missing"
)

// A Reading is one integer, or undefined.
type Reading struct {
	// Value is the integer; 0 when the reading is undefined.
	Value int64
	// OK is false for an undefined reading (a zero base, an empty set, an error).
	OK bool
}

// A Record is one run (spec-experiments.md 8.1.1).
type Record struct {
	// Key names the run.
	Key Key
	// Seed is the seed the run's first world was built with.
	Seed uint64
	// Salt is the reseed of a continuation; HasSalt says whether there was one.
	Salt uint64
	// HasSalt is true when the run was reseeded.
	HasSalt bool
	// Hash is World.ConfigHash of the world built; zero if none was.
	Hash [32]byte
	// Status is the outcome.
	Status Status
	// FailTick is the tick of the failure; -1 if none.
	FailTick int64
	// Detail is the error text of a failure, or the first measure error.
	Detail string
	// Ticks is the end tick of each window that closed; nil with Every 0 or
	// without Keep.Series.
	Ticks []int64
	// Final is one reading per measure over [Transient, Ticks).
	Final []Reading
	// Pass is the verdict of each criterion on this run, in criterion order.
	Pass []bool
	// Series is [window][measure] with Keep.Series; nil with Every 0.
	Series [][]Reading
	// Digest is the last 32 bytes of the final checkpoint; zero if the run failed.
	Digest [32]byte
	// Snapshot is the final snapshot, with Keep.Snapshot.
	Snapshot economysim.Snapshot
	// HasSnapshot is true when Snapshot is set.
	HasSnapshot bool
}
