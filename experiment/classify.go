package experiment

import (
	"errors"

	economysim "github.com/Whealinghold/economy-sim"
)

// stage says where an error came from; with the error it decides the Status
// (spec-experiments.md 9.1). One function, so two code paths cannot give one
// error two codes.
type stage int

const (
	// stageBuild is NewWorld.
	stageBuild stage = iota
	// stageRestore is RestoreWorld.
	stageRestore
	// stageTick is a call of World.Run.
	stageTick
	// stageScript is a Step.Do.
	stageScript
	// stageCheckpoint is World.Checkpoint on a healthy world.
	stageCheckpoint
)

// classify returns the Status of err raised at stage s, and the tick it is
// recorded at: 0 for NewWorld, else the tick the caller passes.
func classify(s stage, err error, tick int64) (Status, int64) {
	switch s {
	case stageBuild:
		var ve *economysim.ValidationError
		switch {
		case errors.Is(err, economysim.ErrAuditFailed):
			return StatusAudit, 0
		case errors.As(err, &ve):
			return StatusInvalidConfig, 0
		}
		return StatusWorld, 0
	case stageRestore:
		return StatusWorld, tick
	case stageTick:
		switch {
		case errors.Is(err, economysim.ErrInvalidTrade):
			return StatusInvalidTrade, tick
		case errors.Is(err, economysim.ErrAuditFailed):
			return StatusAudit, tick
		}
		return StatusStep, tick
	case stageScript:
		return StatusScript, tick
	}
	return StatusCheckpoint, tick
}
