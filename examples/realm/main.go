package main

import (
	"log/slog"
	"os"
)

// main loads the pack and reports the time of the start-up solve. The server and
// its flags come with the next change (plan 2-24 Phase 4).
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg, took, err := LoadTimed()
	if err != nil {
		logger.Error("load the realm pack", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("realm pack loaded", slog.Int("regions", len(cfg.Regions)), slog.Duration("solve", took))
}
