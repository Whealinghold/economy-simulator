"""Acceptance arithmetic for the stop rules of spec-calibration.md 7.4.

Exact binomial arithmetic with fractions, no floats in the verdicts. Reported in
basis points. Run in Docker only (see grid.py).
"""
from fractions import Fraction as F
from math import comb


def sr2_need(n):
    """ceil(9n / 10): the seeds that must be inside a band (STY 8.3)."""
    return -(-9 * n // 10)


def p_at_least(n, k, p):
    """P(X >= k), X ~ Binomial(n, p), exact."""
    return sum(F(comb(n, j)) * p ** j * (1 - p) ** (n - j) for j in range(k, n + 1))


def bp(x):
    return int(round(x * 10000))


def table():
    print("A. one conjunct: chance a point whose per-seed 'inside' probability is p passes SR2")
    print("   p      n=10 (need 9)   n=50 (need 45)   n=10 SR1 (all)   n=50 SR1 (all)")
    for p in (F(80, 100), F(85, 100), F(90, 100), F(95, 100), F(97, 100), F(99, 100), F(995, 1000)):
        a = p_at_least(10, sr2_need(10), p)
        b = p_at_least(50, sr2_need(50), p)
        c = p ** 10
        d = p ** 50
        print(f"   {p!s:6s} {bp(a):10d}      {bp(b):10d}       {bp(c):10d}      {bp(d):10d}")
    print()
    print("B. many conjuncts: the stability record of one default needs 3 checks x P packs x 3 points")
    print("   (default and both neighbours) at the gate (n=10, SR2); independent, an upper bound on joint pass")
    print("   p      one conjunct   9 conj (1 pack)   27 conj (3 packs)   45 conj (5 packs)   63 conj (7 packs)")
    for p in (F(95, 100), F(97, 100), F(98, 100), F(99, 100), F(995, 1000)):
        one = p_at_least(10, sr2_need(10), p)
        print(f"   {p!s:6s} {bp(one):10d}   {bp(one ** 9):12d}   {bp(one ** 27):14d}   {bp(one ** 45):14d}   {bp(one ** 63):14d}")
    print()
    print("C. the winner's curse: N points of equal true p, chosen on a search block of R=10, confirmed on 50")
    print("   p      N    best passes search (SR2, n=10)   chosen passes confirm (SR2, n=50)   confirm if search had been reused")
    for p in (F(80, 100), F(85, 100), F(90, 100)):
        a = p_at_least(10, sr2_need(10), p)
        c = p_at_least(50, sr2_need(50), p)
        for N in (16, 64):
            best = 1 - (1 - a) ** N
            print(f"   {p!s:6s} {N:3d} {bp(best):16d}                     {bp(c):12d}                      {bp(best):8d}")
    print()
    print("D. attempts: a point of true p confirmed on a fresh block each attempt (n=50, SR2); chance it passes")
    print("   at least once in k attempts (the reason attempts are counted and the gate is a final, separate check)")
    print("   p      1 attempt   2 attempts   3 attempts   gate (n=10, SR2) once")
    for p in (F(85, 100), F(90, 100), F(93, 100), F(95, 100)):
        c = p_at_least(50, sr2_need(50), p)
        g = p_at_least(10, sr2_need(10), p)
        print(f"   {p!s:6s} {bp(c):8d} {bp(1 - (1 - c) ** 2):11d} {bp(1 - (1 - c) ** 3):11d} {bp(g):12d}")


if __name__ == "__main__":
    assert sr2_need(10) == 9 and sr2_need(50) == 45 and sr2_need(40) == 36 and sr2_need(4) == 4
    # exactness check against hand arithmetic: P(X>=9), n=10, p=1/2 = (10 + 1) / 1024
    assert p_at_least(10, 9, F(1, 2)) == F(11, 1024)
    table()
