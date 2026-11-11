#!/usr/bin/env bash
# coverage-check.sh <profile> <thresholds> [srcroot]
#
# Checks a Go coverage profile against per-package minimums (doc/rules/testing.md 9.1).
# Blocks are de-duplicated (a block is covered if any copy has a count above zero),
# a package is the directory of its files, and covered*100 >= min*total is integer
# arithmetic. Files that are generated or in package main (read from srcroot, default
# the current directory) are left out. Fails on an empty or malformed profile and on
# a package with no row in the thresholds file. Exit 0 pass, 1 below threshold, 2 bad input.
set -euo pipefail

[ $# -ge 2 ] || { echo "usage: coverage-check.sh <profile> <thresholds> [srcroot]" >&2; exit 2; }
[ -r "$1" ] || { echo "coverage-check: cannot read profile $1" >&2; exit 2; }
[ -r "$2" ] || { echo "coverage-check: cannot read thresholds $2" >&2; exit 2; }

awk -v thr="$2" -v srcroot="${3:-.}" '
function bad(msg) { print "coverage-check: " msg > "/dev/stderr"; broken = 1; exit 2 }
function excluded(src,    line, gen, main, incomment) {
	if (src in skip) return skip[src]
	gen = 0; main = 0
	if ((getline line < src) < 0) bad("cannot read source " src)
	do {
		if (incomment) { if (line !~ /\*\//) continue; sub(/^.*\*\//, "", line); incomment = 0 }
		sub(/^[ \t]*\/\*.*\*\/[ \t]*/, "", line)
		if (line ~ /^[ \t]*\/\*/ && line !~ /\*\//) { incomment = 1; continue }
		if (line ~ /^\/\/ Code generated .* DO NOT EDIT\.$/) gen = 1
		if (line ~ /^package /) { main = (line ~ /^package main([ \t]|$)/); break }
	} while ((getline line < src) > 0)
	close(src)
	return skip[src] = (gen || main)
}
FILENAME == thr {
	if ($0 ~ /^[ \t]*(#|$)/) next
	if ($1 == "base" && NF == 2) { base = $2; next }
	if (NF != 2 || $2 !~ /^[0-9]+$/) bad("bad thresholds row " FNR ": " $0)
	min[$1] = $2
	next
}
FNR == 1 {
	if (base == "") bad("thresholds have no base row")
	if ($0 !~ /^mode: (set|count|atomic)$/) bad("profile line 1 is not a mode line: " $0)
	next
}
/^[ \t]*$/ { next }
{
	if (NF != 3 || $2 !~ /^[0-9]+$/ || $3 !~ /^[0-9]+$/ ||
	    !match($1, /:[0-9]+\.[0-9]+,[0-9]+\.[0-9]+$/))
		bad("malformed profile line " FNR ": " $0)
	file = substr($1, 1, RSTART - 1)
	if (index(file, base "/") != 1) bad("profile line " FNR " is outside " base ": " $0)
	blocks++
	if (excluded(srcroot "/" substr(file, length(base) + 2))) next
	if (!($1 in stmts)) { stmts[$1] = $2; sub(/\/[^\/]*$/, "", file); pkg[$1] = file }
	if ($3 > 0) hit[$1] = 1
}
END {
	if (broken) exit 2
	if (blocks == 0) bad("empty profile: no coverage blocks")
	for (k in stmts) { total[pkg[k]] += stmts[k]; if (k in hit) covered[pkg[k]] += stmts[k] }
	for (p in total) {
		rel = (p == base) ? "." : substr(p, length(base) + 2)
		if (!(rel in min)) { printf "FAIL %s: no threshold row for this package\n", rel; fails++; continue }
		if (total[p] == 0) { printf "skip %s: no statements to cover\n", rel; continue }
		c = covered[p] + 0
		ok = (c * 100 >= min[rel] * total[p])
		pc = int(c * 1000 / total[p])
		printf "%s %s: %d.%d%% (%d/%d statements, minimum %d%%)\n", ok ? "ok  " : "FAIL", rel, pc / 10, pc % 10, c, total[p], min[rel]
		if (!ok) fails++
	}
	if (fails) { print "coverage-check: " fails " scope(s) below threshold" > "/dev/stderr"; exit 1 }
}' "$2" "$1" | sort -k2
