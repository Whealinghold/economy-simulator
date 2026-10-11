"""Spec section 11: the budget of a sweep, from the placeholder costs of spec-scale.md 6.

Nothing here is measured (note N-015): the per-agent costs c_0 = 2 us and c_t = 1 us,
the reference tradable set of 8, and 53 ms a tick for the first consumer's world are
the placeholders of spec-scale.md 6.2 and 6.3; 4 ms a tick for the `one` pack is
spec-reference-chain.md 8.2. The additions of the harness (an observer 6%, a snapshot
1.1 ms, a checkpoint 13 ms, a restore 25 ms) are the placeholders of spec-observability.md
8.3, 5.5 and spec-checkpoint.md 10.3.
"""
TICK_FIVE_MS = 53        # five regions, 5 000 agents (spec-scale.md 6.3)
TICK_ONE_MS = 4          # `one`, 600 agents (spec-reference-chain.md 8.2)
OBS_PCT = 6              # an observer that ignores order and trade events
SNAP_MS = 1.1
CKPT_MS = 13
REST_MS = 25


def run_seconds(ticks, tick_ms, observer=True, samples=0, pairing=False):
    ms = ticks * tick_ms * (100 + (OBS_PCT if observer else 0)) / 100
    ms += samples * SNAP_MS + CKPT_MS   # the end checkpoint: the digest of every record
    if pairing:
        ms += samples * CKPT_MS
    return ms / 1000


def fmt(sec):
    if sec < 120:
        return "%.0f s" % sec
    if sec < 7200:
        return "%.0f min" % (sec / 60)
    return "%.1f h" % (sec / 3600)


def main():
    print("== one run, no observer, no samples (the bare cost of the ticks)")
    for name, tick in (("one", TICK_ONE_MS), ("five", TICK_FIVE_MS)):
        for ticks in (400, 1095, 3650):
            print("  %-4s %5d ticks  %s   (%.1f runs an hour)" % (
                name, ticks, fmt(run_seconds(ticks, tick, False)), 3600 / run_seconds(ticks, tick, False)))

    print()
    print("== a sweep: points x replicas runs of T ticks, one core, observer on, a sample every 30 ticks")
    print("   pack  points  replicas  ticks   runs   total (one core)")
    for name, tick, points, reps, ticks in (
        ("one", TICK_ONE_MS, 3, 10, 400),
        ("one", TICK_ONE_MS, 24, 10, 1095),
        ("one", TICK_ONE_MS, 100, 10, 1095),
        ("five", TICK_FIVE_MS, 12, 10, 1095),
        ("five", TICK_FIVE_MS, 100, 10, 1095),
    ):
        runs = points * reps
        per = run_seconds(ticks, tick, True, samples=ticks // 30)
        total = runs * per
        print("   %-4s %6d %8d %6d %6d   %-18s" % (name, points, reps, ticks, runs, fmt(total)))

    print()
    print("== the saving of a branch: a burn-in of B ticks run once, then K branches of T - B ticks each")
    print("   pack  T     B    K   without branching   with branching   saved")
    for name, tick, T, B, K in (("one", TICK_ONE_MS, 1095, 365, 10), ("five", TICK_FIVE_MS, 1095, 365, 10), ("five", TICK_FIVE_MS, 1095, 730, 20)):
        without = K * T * tick / 1000
        with_b = (B * tick + K * ((T - B) * tick + REST_MS)) / 1000
        print("   %-4s %5d %4d %3d   %-17s %-15s %d%%" % (name, T, B, K, fmt(without), fmt(with_b), round(100 * (without - with_b) / without)))

    print()
    print("== the size of a Result: runs x measures x (1 + windows) readings; 9 bytes packed (an int64 and a flag), 16 as a Go struct")
    print("   points  replicas  measures  samples  final only (16 B)  with the series (16 B)  with the series (9 B)")
    for points, reps, measures, samples in ((24, 10, 20, 36), (100, 10, 20, 36), (1000, 10, 20, 36), (1000, 50, 40, 120)):
        cells = points * reps * measures
        def fm(b):
            return "%d KB" % (b // 1000) if b < 10**6 else "%.1f MB" % (b / 1e6)
        print("   %6d %8d %9d %8d  %-17s  %-22s  %s" % (points, reps, measures, samples, fm(cells * 16), fm(cells * (1 + samples) * 16), fm(cells * (1 + samples) * 9)))


if __name__ == "__main__":
    main()
