"""POP model 1b: where the oscillation starts. Same rules as fixed_point.py; a finer sweep of the gain."""
from fixed_point import *
print(f"{'m':>4} {'T':>5} {'tau_e':>6} {'k*tau':>7} {'crossings':>9} {'tail amp':>9}")
for T, tau_e in ((365, 365), (1095, 365)):
    for m in (8, 12, 16, 24, 32, 40, 48, 64, 96):
        Ns, s = run_full(N_start=300_000, m=m, T=T, tau_e=tau_e, years=400)
        k = k_per_year(m, 0.5); tau = (T/2 + tau_e)/TPY
        cr, amp = classify(s, Ns)
        print(f"{m:>4} {T:>5} {tau_e:>6} {k*tau:7.2f} {cr:>9} {amp:9.4f}")
