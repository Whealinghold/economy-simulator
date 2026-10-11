"""Parse the registry of spec-calibration.md (section 6) as plain text and run the
checks of its section 11.1 against the parameter-owner table of README.md, the facts
and scenarios of spec-stylised-facts.md, the register of spec-stability.md, the
presets of spec-indicators.md and the owners' parameter tables.

Standard library only. Prints one line per check, then a list of findings; exits 1
on a failure. Run in Docker (never the host interpreter):
  docker run --rm -e PYTHONDONTWRITEBYTECODE=1 -v "$PWD/doc/specs":/s:ro \
    -w /s/models/clb python:3.12-slim python registry.py
"""
import re
import sys

ROOT = "../.."


def read(name):
    with open(f"{ROOT}/{name}", encoding="utf-8") as f:
        return f.read()


CLB = read("spec-calibration.md")
README = read("README.md")
STY = read("spec-stylised-facts.md")
STB = read("spec-stability.md")
IND = read("spec-indicators.md")

fails = []
notes = []


def check(name, ok, detail=""):
    print(("ok   " if ok else "FAIL ") + name + ((" :: " + detail) if detail and not ok else ""))
    if not ok:
        fails.append(name)


def cells(line):
    parts = re.split(r"(?<!\\)\|", line.strip())
    return [p.strip().strip("`") for p in parts[1:-1]]


knob_rows, group_rows = [], []
for line in CLB.splitlines():
    if line.startswith("| K-"):
        knob_rows.append(cells(line))
    elif line.startswith("| S-"):
        group_rows.append(cells(line))

check("knob rows have 12 cells", all(len(r) == 12 for r in knob_rows), str([r[0] for r in knob_rows if len(r) != 12]))
check("group rows have 7 cells", all(len(r) == 7 for r in group_rows), str([r[0] for r in group_rows if len(r) != 7]))
ids = [r[0] for r in knob_rows] + [r[0] for r in group_rows]
check("ids are unique", len(ids) == len(set(ids)), str({i for i in ids if ids.count(i) > 1}))
kd = [r for r in knob_rows if r[2] in ("K", "D")]
print(f"     {len(knob_rows)} knob rows (K {sum(r[2]=='K' for r in knob_rows)}, D {sum(r[2]=='D' for r in knob_rows)}, "
      f"S {sum(r[2]=='S' for r in knob_rows)}, P {sum(r[2]=='P' for r in knob_rows)}), {len(group_rows)} groups")
srch = {}
for r in kd:
    k = "U" if r[5] == "U" else ("E" if r[5].startswith("E") else "I")
    srch[k] = srch.get(k, 0) + 1
print(f"     K and D rows {len(kd)}: search kinds {srch}")

UNITS = {"bp", "ticks", "units", "minor", "count", "years", "share", "name", "flag", "list"}
CLASSES = {"K", "D", "S", "P"}
STAGES = {f"C{i}" for i in range(1, 10)} | {"-"}
FLAGS = re.compile(r"^(in|chk|out|tbl|d|z|v1x|off=(min|max|flat|-|\d+))$")
MOVE = re.compile(r"^([A-Z]\d\d)/([A-Za-z0-9_.]+):([+\-0~][SMHE]|\?)$")

# ---- IND presets and the closed field spelling ------------------------------------
ind_presets = set(re.findall(r"^\| \d+ \| `([A-Za-z]+)` \|", IND, re.M))
FIELD_OK = {
    "Pinned": {"Share"}, "Turnover": {"Volume", "Velocity"}, "PriceRange": {""}, "PriceIndex": {"Fisher"},
    "Offset": {"", "abs"}, "Dispersion": {""}, "RoleShares": {"Swing"}, "Households": {""}, "Unfed": {""},
    "WelfareDistribution": {"Mean"}, "PriceGap": {"Spread", "GapAfterFreight"}, "StockCover": {""},
    "Sectors": {"M"}, "HouseholdIncome": {""},
}
OBS_EVENTS = {"UpkeepShortfall"}

fact_ids = set(re.findall(r"\*\*STY-([A-Z]\d\d)\b", STY))
scen_ids = set(re.findall(r"^\| (S\d\d) \|", STY, re.M))
loops = set(re.findall(r"\*\*STB-(L\d+)\b", STB))
dampers = set(re.findall(r"^\| (D\d\d) \|", STB, re.M))
interactions = set(re.findall(r"^\| (I\d\d) \|", CLB, re.M))
aliases = set(re.findall(r"^\| (A\d\d) \|", CLB, re.M))
anchors = set(re.findall(r"^\| (E\d) \|", CLB, re.M))
print(f"     IND presets {len(ind_presets)}, facts {len(fact_ids)}, scenarios {len(scen_ids)}, loops {len(loops)}, "
      f"dampers {len(dampers)}, I {len(interactions)}, A {len(aliases)}, E {len(anchors)}")

# loop and damper entry text, for the 'parameter named in the entry' check
loop_text = {}
for m in re.finditer(r"\*\*STB-(L\d+)\b(.*?)(?=\n\*\*STB-L\d+|\n## |\n### )", STB, re.S):
    loop_text[m.group(1)] = m.group(2)
damper_text = {}
for m in re.finditer(r"^\| (D\d\d) \|(.*)$", STB, re.M):
    damper_text[m.group(1)] = m.group(2)
# the parameter table of section 11 of STB names parameters per damper too
for m in re.finditer(r"^\| (D\d\d)(?:[, ]+D\d\d)* [^|]*\|([^|]*)\|", STB, re.M):
    damper_text[m.group(1)] = damper_text.get(m.group(1), "") + m.group(2)
for m in re.finditer(r"^\| ((?:D\d\d(?:, )?)+(?: to D\d\d)?) \|([^|]*)\|", STB, re.M):
    for d in re.findall(r"D\d\d", m.group(1)):
        damper_text[d] = damper_text.get(d, "") + m.group(2)

bad = []
for r in knob_rows:
    i, key, cls, owner, unit, search, grid, stage, flags, moves, lps, links = r
    scope, _, field = key.rpartition(".")
    if cls not in CLASSES: bad.append((i, "class", cls))
    if unit not in UNITS: bad.append((i, "unit", unit))
    if not (search in ("U", "-") or re.fullmatch(r"E\d", search) or re.fullmatch(r"I\d\d", search)):
        bad.append((i, "search", search))
    if re.fullmatch(r"E\d", search) and search not in anchors: bad.append((i, "anchor", search))
    if re.fullmatch(r"I\d\d", search) and search not in interactions: bad.append((i, "search-I", search))
    if not (grid in ("-", "geo") or re.fullmatch(r"lin\d+", grid) or grid.startswith("set:")):
        bad.append((i, "grid", grid))
    if stage not in STAGES: bad.append((i, "stage", stage))
    toks = [] if flags == "-" else flags.split()
    for t in toks:
        if not FLAGS.match(t): bad.append((i, "flag", t))
    if cls == "D" and not any(t.startswith("off=") for t in toks): bad.append((i, "D without off", flags))
    if cls != "D" and any(t.startswith("off=") for t in toks): bad.append((i, "off on non-D", flags))
    if cls in ("S", "P") and not (search == "-" or re.fullmatch(r"E\d", search)): bad.append((i, "S/P with a search kind", search))
    if cls in ("K", "D") and search == "-": bad.append((i, "K/D without a search kind", search))
    if cls in ("K", "D") and grid == "-" and stage != "-" and not any(l.startswith("A") for l in links.split()):
        bad.append((i, "searched K/D without a grid or a composite link", links))
    if moves != "-":
        mt = [m.strip() for m in moves.split(";")]
        if len(mt) > 4: bad.append((i, "more than four moves", moves))
        for m in mt:
            mm = MOVE.match(m)
            if not mm: bad.append((i, "move", m)); continue
            fact, meas, sign = mm.groups()
            if fact not in fact_ids and fact not in scen_ids: bad.append((i, "unknown fact/scenario", fact))
            preset, _, fld = meas.partition(".")
            if preset in FIELD_OK:
                if preset not in ind_presets: bad.append((i, "preset not in IND", preset))
                if fld not in FIELD_OK[preset]: bad.append((i, "field", meas))
            elif meas in OBS_EVENTS or re.fullmatch(r"[a-z]\d\d?\.[a-z_]+", meas):
                if re.fullmatch(r"[a-z]\d\d?\.[a-z_]+", meas) and meas not in STY: bad.append((i, "band key not in STY", meas))
            else:
                bad.append((i, "unknown measure", meas))
    for t in ([] if lps == "-" else lps.split()):
        b = t.split("=")[0]
        if re.fullmatch(r"L\d+", b):
            if b not in loops: bad.append((i, "unknown loop", b))
            elif field not in loop_text.get(b, ""): notes.append((i, f"{b} entry does not name {field}"))
        elif re.fullmatch(r"D\d\d", b):
            if b not in dampers: bad.append((i, "unknown damper", b))
            elif field not in damper_text.get(b, "") and field not in loop_text.get("", ""):
                notes.append((i, f"{b} row/table does not name {field}"))
        else:
            bad.append((i, "loop token", t))
    for t in ([] if links == "-" else links.split()):
        if re.fullmatch(r"I\d\d", t):
            if t not in interactions: bad.append((i, "unknown I", t))
        elif re.fullmatch(r"A\d\d", t):
            if t not in aliases: bad.append((i, "unknown A", t))
        else:
            bad.append((i, "link token", t))
check("knob rows use the closed sets and known references", not bad, str(bad[:15]))

badg = []
for r in group_rows:
    i, scope, fields, cls, owner, stage, flags = r
    if cls not in CLASSES: badg.append((i, "class", cls))
    if stage not in STAGES: badg.append((i, "stage", stage))
    for t in ([] if flags == "-" else flags.split()):
        if not FLAGS.match(t): badg.append((i, "flag", t))
check("structure groups use the closed sets", not badg, str(badg[:10]))

# ---- owner tables -------------------------------------------------------------------
code_to_file = {}
for m in re.finditer(r"^\| `([A-Z]{3})` \| `(spec-[a-z\-]+\.md)`", README, re.M):
    code_to_file[m.group(1)] = m.group(2)
code_to_file["CUR"] = "spec-currency.md"
code_to_file["CAR"] = "spec-carriage.md"

# README owner table: (scope token, parameter) -> owner file and section
sect = README.split("## Parameter owners", 1)[1].split("## New domain terms", 1)[0]
owner_pairs = {}
for line in sect.splitlines():
    if not line.startswith("| `"): continue
    cs = [c.strip() for c in line.strip().strip("|").split("|")]
    param = re.sub(r" \(.*", "", re.match(r"`([^`]+)`", cs[0]).group(1))
    scope = re.split(r";| \(", cs[1])[0].strip()
    om = re.search(r"`(spec-[a-z\-]+\.md)` section ([0-9][0-9.]*)", cs[2])
    owner_pairs[(scope, param)] = (om.group(1), om.group(2)) if om else (None, None)

unres, mism = [], []
file_code = {v: k for k, v in code_to_file.items()}
for r in knob_rows:
    m = re.fullmatch(r"([A-Z]{3}) ([0-9][0-9.]*)", r[3])
    scope, _, field = r[1].rpartition(".")
    if not m or m.group(1) not in code_to_file:
        unres.append((r[0], r[3])); continue
    text = read(code_to_file[m.group(1)])
    num = re.escape(m.group(2))
    if not re.search(rf"^(#+ )?{num}[. ]", text, re.M) and not re.search(rf"^(#+ )?{num}$", text, re.M):
        unres.append((r[0], r[3]))
    own = owner_pairs.get((scope, field))
    if own and own[0]:
        ocode = file_code.get(own[0])
        if ocode != m.group(1) or not (m.group(2) == own[1] or m.group(2).startswith(own[1] + ".") or own[1].startswith(m.group(2) + ".")):
            mism.append((r[0], r[3], f"README {ocode} {own[1]}"))
check("owner sections resolve in the owner spec", not unres, str(unres))
check("owner cell matches the README owner section", not mism, str(mism[:20]))
check("flag z only on defaults of 0; geo only on defaults of 1 or more (owner table)", True)

# default from the owner's parameter table
def owner_default(code, scope, field):
    f = code_to_file.get(code)
    if not f: return None
    lines = read(f).splitlines()
    for n, ln in enumerate(lines):
        if re.match(rf"^\|[^|]*\|?\s*`{re.escape(field)}`\s*\|", ln) or re.match(rf"^\|\s*`{re.escape(field)}`\s*\|", ln):
            # nearest header above
            for k in range(n, max(0, n - 40), -1):
                if lines[k].startswith("|") and "Default" in lines[k]:
                    hdr = cells(lines[k]); row = cells(ln)
                    if "Default" in hdr and len(row) == len(hdr):
                        return row[hdr.index("Default")]
                    break
    return None

geo_issues, z_issues = [], []
for r in knob_rows:
    cls, grid, flags = r[2], r[6], r[8].split() if r[8] != "-" else []
    if grid != "geo" and "z" not in flags: continue
    m = re.fullmatch(r"([A-Z]{3}) ", r[3] + " ")
    code = r[3].split()[0]
    scope, _, field = r[1].rpartition(".")
    d = owner_default(code, scope, field)
    if d is None:
        notes.append((r[0], "default not found in the owner's table (check by hand)")); continue
    zero = bool(re.match(r"(0\b|none|nil|absent|empty|\(none\))", d.strip(), re.I))
    num = re.fullmatch(r"[0-9 ]+", d.strip())
    if grid == "geo" and "z" not in flags and (zero or (num and int(d.replace(" ", "")) < 1)):
        geo_issues.append((r[0], d))
    if "z" in flags and not (zero or d.strip() in ("0",)):
        z_issues.append((r[0], d))
check("a geo row has an owner default of at least 1, or the flag z", not geo_issues, str(geo_issues))
check("the flag z only where the owner default is 0 or none", not z_issues, str(z_issues))

# ---- coverage -------------------------------------------------------------------------
covered = set()
for r in knob_rows:
    scope, _, field = r[1].rpartition(".")
    covered.add((scope, field))
for r in group_rows:
    for fld in [x.strip() for x in r[2].split(",")]:
        covered.add((r[1], fld))
missing = sorted(p for p in owner_pairs if p not in covered)
check("every owner row is a knob row or in a group", not missing, str(missing))
extra = sorted(p for p in covered if p not in owner_pairs)
unexpected = [p for p in extra if p[0] not in {"Solve option", "Presentation"} and p != ("World", "Clearing")]
check("registry-only pairs are the expected scopes", not unexpected, str(unexpected))
print(f"     {len(owner_pairs)} owner pairs, {len(extra)} registry-only")

heads = [l for l in CLB.splitlines() if l.startswith("| ID | Key |")]
check("no Default or Bound column", all("Default" not in h and "Bound" not in h for h in heads))

# group rows: each field's README owner section must match the group's owner cell
gmism = []
for r in group_rows:
    items = []
    for part in r[4].split(";"):
        t = part.split()
        if len(t) >= 2: items += [(t[0], n) for n in t[1:]]
    for fld in [x.strip() for x in r[2].split(",")]:
        own = owner_pairs.get((r[1], fld))
        if own and own[0]:
            oc = file_code.get(own[0])
            if not any(c == oc and (n == own[1] or own[1].startswith(n + ".") or n.startswith(own[1] + ".")) for c, n in items):
                gmism.append((r[0], fld, r[4], own[1]))
check("group owner cells match the README owner sections", not gmism, str(gmism[:30]))

# ---- back links: a knob named in an I or A row links to it --------------------------
back = []
for line in CLB.splitlines():
    m = re.match(r"^\| ((?:I|A)\d\d) \|(.*)\|\s*$", line)
    if not m: continue
    rid, rest = m.group(1), m.group(2)
    for r in knob_rows:
        field = r[1].rpartition(".")[2]
        if re.search(rf"`{re.escape(field)}`", rest) or re.search(rf"\b{re.escape(field)}\b", rest.split("|")[-1]):
            if rid not in r[11].split():
                back.append((r[0], rid))
print(f"     back links missing (informational): {len(back)}")
for b in back[:40]:
    print("       ", b)

print()
print("notes (informational, to be read):", len(notes))
for n in notes[:80]:
    print("   ", n)
print()
print("FAILED: " + ", ".join(fails) if fails else "all checks passed")
sys.exit(1 if fails else 0)
