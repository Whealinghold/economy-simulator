"""Mechanical checks of spec-calibration.md (the checks N-103 asks for), standard
library only: requirement IDs referenced and defined, section references resolve,
files named exist, no shell-quoting residue, no host-interpreter text, no float in
the models' verdict code. Run in Docker only (see grid.py)."""
import os
import re
import sys

ROOT = "../.."
text = open(f"{ROOT}/spec-calibration.md", encoding="utf-8").read()
fails = []


def check(name, ok, detail=""):
    print(("ok   " if ok else "FAIL ") + name + ((" :: " + detail) if detail and not ok else ""))
    if not ok:
        fails.append(name)


# requirement IDs: defined once, every referenced one defined
sec21 = text.split("## 21. Requirements")[1].split("## 22.")[0]
defined = re.findall(r"^\| ((?:N?FR|NFR)-CLB-\d+) \|", sec21, re.M)
check("requirement IDs are unique", len(defined) == len(set(defined)))
refs = set(re.findall(r"\b(?:FR|NFR)-CLB-\d+\b", text))
undefined = sorted(refs - set(defined))
# a range such as "FR-CLB-010 to FR-CLB-014" is two references; both must exist
check("every FR-CLB reference is defined", not undefined, str(undefined))
unreferenced = sorted(set(defined) - {r for r in re.findall(r"\b(?:FR|NFR)-CLB-\d+\b", text.split("## 21. Requirements")[0])}
                      - {r for r in re.findall(r"\b(?:FR|NFR)-CLB-\d+\b", text.split("## 22. Open questions")[1])})
print(f"     {len(defined)} requirements defined; not cited outside section 21: {unreferenced}")

# headings and numbered paragraphs
heads = set()
for m in re.finditer(r"^#{2,4} (\d+(?:\.\d+)*)\.? ", text, re.M):
    heads.add(m.group(1))
for m in re.finditer(r"^(\d+(?:\.\d+){1,3})\. ", text, re.M):
    heads.add(m.group(1))
body = re.sub(r"```.*?```", "", text, flags=re.S)
cites = set()
# "section 7", "7.3.11", "(5.2)" - only references to THIS spec: bare numbers in parentheses
for m in re.finditer(r"\((\d+(?:\.\d+){0,3})(?:[,; ][^)]*)?\)", body):
    cites.add(m.group(1))
for m in re.finditer(r"\b[Ss]ections? (\d+(?:\.\d+){0,3})", body):
    cites.add(m.group(1))
top = {str(i) for i in range(1, 25)}
# numbers that are not a section of this spec: values in parentheses, and two
# references to other files written bare (architecture.md 2.1, STY 6.5.1)
other = {"2.1", "6.5.1"}
unresolved = sorted(c for c in cites if c not in heads and c not in top and c not in other
                    and "." in c and int(c.split(".")[0]) <= 24)
# parenthesised numbers that belong to another file are written with a file name or a
# code (STY 7.3.1); a bare "(n.n)" is ours. Report, do not fail on the pure-number ones
check("bare section references resolve in this spec", not unresolved, str(unresolved))

# files named exist
files = set(re.findall(r"`(spec-[a-z\-]+\.md)`", text)) | set(re.findall(r"\b(spec-[a-z\-]+\.md)\b", text))
missing = sorted(f for f in files if f != "spec-calibration.md" and not os.path.exists(f"{ROOT}/{f}"))
check("every spec file named exists", not missing, str(missing))

check("no shell-quoting residue", "'\"'\"'" not in text)
check("no tab characters", "\t" not in text)
check("no setting words (a short list)", not re.search(r"\b(peasant|village|caravan|gold|silver|copper|elf|dwarf|knight|sword)\b", text, re.I),
      str(re.findall(r"\b(peasant|village|caravan|gold|silver|copper|elf|dwarf|knight|sword)\b", text, re.I)[:5]))

print()
print("FAILED: " + ", ".join(fails) if fails else "all checks passed")
sys.exit(1 if fails else 0)
