"""Run counts and core-hours of the stages of spec-calibration.md 7.3.

Every figure is a COUNT multiplied by the PLACEHOLDERS of spec-scale.md 6, STY 4.5.1,
EXP 10 and CHK 10.3 (N-015). Nothing is measured. One core, no parallelism.
Integer milliseconds. Run in Docker only (see grid.py).
"""

TICK_US = {            # microseconds a tick (placeholders): agents x 6.5 us, five = 53 ms (SCL 6.3)
    "one": 3900, "second": 6500, "two": 6500, "three": 11700, "five": 53000,
    "famine": 3900, "one-demand": 3900,   # one-demand: census of 600 households x 6.5 us (REF 8.2)
}
OBSERVER_PCT = 6       # EXP 5.6 / OBS 8.3: a placeholder 6% of a tick
SNAPSHOT_MS = 1.1      # OBS 5.5
CHECKPOINT_MS = 13     # CHK 10.3
RESTORE_MS = 25

TICKS_REF = 1095       # the REF 8.2 length of a steady scenario (three years); C5 uses 1 800
TICKS4Y = TICKS_REF
TICKS100Y = 36500


def run_ms(pack, ticks, windows=3, branch=False):
    t = TICK_US[pack] * ticks // 1000
    t = t + t * OBSERVER_PCT // 100
    t += int(SNAPSHOT_MS * (windows + 1)) + CHECKPOINT_MS
    if branch:
        t += RESTORE_MS
    return t


def hours(ms):
    return ms / 3_600_000


def explore_n(d):
    return max(16, 10 * d)   # a placeholder rule of 7.2: ten points a screened-in knob, at least 16


def stage(name, pack, k, d, ticks, R=10, confirm_seeds=50, confirm_packs=("one",), windows=3,
          branch=None, conf_seeds=None):
    screen_pts = 1 + 2 * k
    explore_pts = explore_n(d)
    n_search = (screen_pts + explore_pts) * R
    per = run_ms(pack, ticks, windows)
    if branch:
        # a unit: burn-in once, then v variants (burn_in + v * (ticks - burn_in)), one restore each
        burn, v = branch
        per = run_ms(pack, burn, 1) + v * (run_ms(pack, ticks - burn, windows, branch=True))
    n_conf = 0
    t_conf = 0
    for cp in confirm_packs:
        c = (conf_seeds or {}).get(cp, confirm_seeds) * 2                      # the chosen point and the base as a control
        n_conf += c
        t_conf += c * (run_ms(cp, ticks, windows) if not branch else per)
    total = n_search * per + t_conf
    return dict(name=name, pack=pack, k=k, d=d, screen=screen_pts, explore=explore_pts, R=R,
                search_runs=n_search, confirm_runs=n_conf, ms=total)


STAGES = [
    dict(name="C3 scale", pack="one", k=2, d=0, screen=0, explore=0, R=0, search_runs=30, confirm_runs=0,
         ms=30 * run_ms("one", TICKS4Y)),     # 3 scales x 10 seeds, no search (7.3.4)
    stage("C4a beliefs", "one", 16, 6, TICKS4Y, confirm_packs=("one", "second")),
    stage("C4b agents and welfare", "one", 13, 5, TICKS4Y, confirm_packs=("one", "second", "one", "one")),   # M06 and H03 legs: variants of one
    stage("C4c occupation", "one", 5, 3, TICKS4Y, confirm_packs=("one",)),
    stage("C5 shocks and ladder", "famine", 12, 4, 1800, branch=(400, 2), confirm_packs=("famine", "one", "one")),   # W05 on one and on one + nofloor
    stage("C6 demand and barriers", "one-demand", 9, 4, TICKS4Y, confirm_packs=("one-demand", "three", "five"), conf_seeds={"three": 4, "five": 2}),
    stage("C7a trade", "three", 12, 4, TICKS4Y, confirm_packs=("two", "three")),
    stage("C7b movement", "three", 10, 4, TICKS4Y, confirm_packs=("three",)),
    stage("C8 open layer", "one", 12, 3, TICKS4Y, confirm_packs=("one",)),
    stage("C9 long facts", "one-demand", 8, 4, TICKS100Y, confirm_seeds=10, confirm_packs=("one-demand",), windows=98),
]


def main():
    print("placeholders: tick us per pack =", TICK_US)
    print(f"observer +{OBSERVER_PCT}%, snapshot {SNAPSHOT_MS} ms, checkpoint {CHECKPOINT_MS} ms, restore {RESTORE_MS} ms")
    print()
    print(f"{'stage':26s} {'pack':11s} {'knobs':>5s} {'in':>3s} {'screen':>6s} {'explore':>7s} {'search runs':>11s} {'confirm runs':>12s} {'core-hours':>10s}")
    tot_runs = tot_ms = 0
    for s in STAGES:
        print(f"{s['name']:26s} {s['pack']:11s} {s['k']:5d} {s['d']:3d} {s['screen']:6d} {s['explore']:7d} "
              f"{s['search_runs']:11d} {s['confirm_runs']:12d} {hours(s['ms']):10.1f}")
        tot_runs += s["search_runs"] + s["confirm_runs"]
        tot_ms += s["ms"]
    print(f"{'all stages, one pass':26s} {'':11s} {'':5s} {'':3s} {'':6s} {'':7s} {'':11s} {tot_runs:12d} {hours(tot_ms):10.1f}")
    print()
    # the stability record of a claimed set of defaults: gate seeds and hold-out seeds, every shipped pack
    claimed = 49
    pts = 1 + 2 * claimed
    packs = ("one", "second", "two", "three", "five", "famine", "one-demand")   # the seven packs of REF 4.4
    per_seed = sum(run_ms(p, TICKS4Y) for p in packs)
    ms = pts * 20 * per_seed
    print(f"stability record of {claimed} claimed defaults: {pts} points x {len(packs)} packs x 20 gate and hold-out seeds = "
          f"{pts * len(packs) * 20} runs of {TICKS4Y} ticks = {hours(ms):.1f} core-hours")
    print()
    print("a single run, for scale:")
    for p in ("one", "three", "five"):
        print(f"  {p:6s} {TICKS4Y} ticks: {run_ms(p, TICKS4Y) / 1000:.1f} s     {TICKS100Y} ticks: {run_ms(p, TICKS100Y, 98) / 1000:.0f} s")


if __name__ == "__main__":
    main()
