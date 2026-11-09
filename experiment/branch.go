package experiment

import (
	"context"
	"errors"

	economysim "github.com/Whealinghold/economy-sim"
)

// runBranched is built in Phase 6.
func (e Spec) runBranched(ctx context.Context, cfg economysim.Config, u Unit) ([]Record, error) {
	return nil, errors.New("experiment: branching is not built")
}
