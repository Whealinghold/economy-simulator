package economysim

import (
	"context"
	"log/slog"

	"github.com/Whealinghold/economy-sim/internal/logkeys"
)

// logProject is the Debug line of the project of a household in the Trade view
// (spec-saving-and-investment.md 12.2): "project chosen" with the goal and the buffer, or
// "project declined" with the bar, or for a standing refusal the world's reason of
// spec-infrastructure.md 4.2.2. The strategy has no logger, so the world logs it where it
// builds the view; the rule is the pure one, so the line cannot disagree with what the
// default does. It runs only when Debug is enabled, takes no draw and changes nothing.
func (s *saving) logProject(v *AgentView) {
	ctx := context.Background()
	if !s.log.Enabled(ctx, slog.LevelDebug) {
		return
	}
	reserve, upkeep := v.savingBasis()
	pk := pickProject(v, reserve, upkeep)
	switch {
	case pk.chosen:
		res := pk.target(0)
		s.log.LogAttrs(ctx, slog.LevelDebug, "project chosen", slog.Int64(logkeys.Tick, v.Tick.Tick),
			slog.Int64(logkeys.Agent, int64(v.Agent)), slog.Int64(logkeys.Recipe, int64(pk.choice.Recipe)),
			slog.Int64(logkeys.Goal, res.Goal), slog.Int64(logkeys.Buffer, res.Precaution))
	case pk.bar != "" && pk.first >= 0:
		reason := string(pk.bar)
		if pk.bar == BarStanding {
			reason = string(pk.standing)
		}
		s.log.LogAttrs(ctx, slog.LevelDebug, "project declined", slog.Int64(logkeys.Tick, v.Tick.Tick),
			slog.Int64(logkeys.Agent, int64(v.Agent)), slog.Int64(logkeys.Recipe, int64(pk.first)),
			slog.String(logkeys.Reason, reason))
	}
}
