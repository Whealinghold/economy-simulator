"""Parse the registry of spec-calibration.md (section 6) as plain text and run the
checks of its section 11.1 against the parameter-owner table of README.md, the facts
and scenarios of spec-stylised-facts.md, the register of spec-stability.md, the
presets of spec-indicators.md, the event table of observability-and-logging.md, the catalogue of
spec-config.md 3.7 and the owners' parameter tables.

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
OBS = read("../rules/observability-and-logging.md")
CFG = read("spec-config.md")
import grid as gridmod          # the grid and cut rule of 5.2 (models/clb/grid.py)

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
_obs_sec = OBS.split("## 4. Events", 1)[1].split("## 5. Metrics", 1)[0]
OBS_EVENTS = set()
for _ln in _obs_sec.splitlines():
    if _ln.startswith("| `"):
        OBS_EVENTS |= set(re.findall(r"`([A-Z][A-Za-z]+)`", _ln.split("|")[1]))

fact_ids = set(re.findall(r"\*\*STY-([A-Z]\d\d)\b", STY))
scen_ids = set(re.findall(r"^\| (S\d\d) \|", STY, re.M))
scen_text = {m.group(1): m.group(2) for m in re.finditer(r"^\| (S\d\d) \|(.*)$", STY, re.M)}
loops = set(re.findall(r"\*\*STB-(L\d+)\b", STB))
dampers = set(re.findall(r"^\| (D\d{2,3}) \|", STB, re.M))
interactions = set(re.findall(r"^\| (I\d\d) \|", CLB, re.M))
aliases = set(re.findall(r"^\| (A\d\d) \|", CLB, re.M))
anchors = set(re.findall(r"^\| (E\d) \| anchor \|", CLB, re.M))
leads = set(re.findall(r"^\| (E\d) \| lead \|", CLB, re.M))
print(f"     IND presets {len(ind_presets)}, facts {len(fact_ids)}, scenarios {len(scen_ids)}, loops {len(loops)}, "
      f"dampers {len(dampers)}, I {len(interactions)}, A {len(aliases)}, E {len(anchors)}")

# loop and damper entry text, for the 'parameter named in the entry' check
loop_text = {}
for m in re.finditer(r"\*\*STB-(L\d+)\b(.*?)(?=\n\*\*STB-L\d+|\n## |\n### )", STB, re.S):
    loop_text[m.group(1)] = m.group(2)
damper_text = {}
for m in re.finditer(r"^\| (D\d{2,3}) \|(.*)$", STB, re.M):
    damper_text[m.group(1)] = m.group(2)
# the parameter table of section 11 of STB names parameters per damper too
for m in re.finditer(r"^\| (D\d{2,3})(?:[, ]+D\d{2,3})* [^|]*\|([^|]*)\|", STB, re.M):
    damper_text[m.group(1)] = damper_text.get(m.group(1), "") + m.group(2)
for m in re.finditer(r"^\| ((?:D\d{2,3}(?:, )?)+(?: to D\d{2,3})?) \|([^|]*)\|", STB, re.M):
    for d in re.findall(r"D\d{2,3}", m.group(1)):
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
            if fact not in fact_ids and fact in scen_ids and meas not in scen_text[fact]:
                bad.append((i, "an S<nn> token needs the band key written in the STY 5.11 row", f"{fact}/{meas}"))
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
        elif re.fullmatch(r"D\d{2,3}", b):
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

# README owner table: (scope token, parameter) -> owner file and ALL its sections ("section 4.2" or
# "sections 4.2, 11.1"); a row that parses no section FAILS (11.1 "owner")
sect = README.split("## Parameter owners", 1)[1].split("## New domain terms", 1)[0]
owner_pairs = {}
no_section = []
for line in sect.splitlines():
    if not line.startswith("| `"): continue
    cs = [c.strip() for c in line.strip().strip("|").split("|")]
    param = re.sub(r" \(.*", "", re.match(r"`([^`]+)`", cs[0]).group(1))
    scope = re.split(r";| \(", cs[1])[0].strip()
    om = re.search(r"`(spec-[a-z\-]+\.md)` sections? ([0-9][0-9., and]*)", cs[2])
    if om:
        secs = re.findall(r"[0-9]+(?:\.[0-9]+)*", om.group(2))
        owner_pairs[(scope, param)] = (om.group(1), secs)
    else:
        owner_pairs[(scope, param)] = (None, [])
        no_section.append((scope, param))
    # a remark "(also <Scope>, ...)" names a second scope of the same row (4.7): its own pair
    am = re.search(r"\(also ([A-Z][A-Za-z ]*?)[,)]", cs[1])
    if am:
        owner_pairs[(am.group(1).strip(), param)] = owner_pairs[(scope, param)]
check("every README owner row parses an owner file and a section", not no_section, str(no_section[:10]))


def sec_ok(mine, theirs):
    return mine in theirs or any(mine.startswith(t + ".") or t.startswith(mine + ".") for t in theirs)


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
        if ocode != m.group(1) or not sec_ok(m.group(2), own[1]):
            mism.append((r[0], r[3], f"README {ocode} {own[1]}"))
check("owner sections resolve in the owner spec", not unres, str(unres))
check("owner cell matches a README owner section", not mism, str(mism[:20]))


# default and bound from the owner's parameter table
def owner_row(code, field, scope=None):
    f = code_to_file.get(code)
    if not f: return None
    lines = read(f).splitlines()
    for n, ln in enumerate(lines):
        if scope and len(cells(ln)) > 1 and cells(ln)[0] != scope and any(cells(l2)[0] == scope for l2 in lines if l2.startswith("|") and len(cells(l2)) > 1 and cells(l2)[1] == field):
            continue
        if re.match(rf"^\|[^|]*\|?\s*`{re.escape(field)}`\s*\|", ln) or re.match(rf"^\|\s*`{re.escape(field)}`\s*\|", ln):
            for k in range(n, max(0, n - 40), -1):
                if lines[k].startswith("|") and "Default" in lines[k]:
                    hdr = cells(lines[k]); row = cells(ln)
                    if "Default" in hdr and len(row) == len(hdr):
                        bound = row[hdr.index("Bound")] if "Bound" in hdr else ""
                        return row[hdr.index("Default")], bound
                    break
    return None


def num_of(s):
    s = s.strip().replace(" ", "")
    return int(s) if re.fullmatch(r"-?[0-9]+", s) else None


def bound_range(bound):
    m = re.match(r"\s*(-?[0-9][0-9 ]*|`?[A-Za-z]+`?)\s+to\s+(.+)$", bound)
    if not m: return None, None
    return num_of(m.group(1)), num_of(m.group(2).split(",")[0].split(";")[0])


geo_issues, z_issues, req_issues, off_issues = [], [], [], []
cut_rows = []
for r in knob_rows:
    cls, kind, flags = r[2], r[6], r[8].split() if r[8] != "-" else []
    code = r[3].split()[0]
    scope, _, field = r[1].rpartition(".")
    got = owner_row(code, field, scope.split(".")[-1])
    if got is None:
        if kind == "geo" or "z" in flags:
            notes.append((r[0], "default not found in the owner's table (check by hand)"))
        continue
    d, bound = got
    ds = d.strip()
    zero = bool(re.match(r"(0\b|none|nil|absent|empty|\(none\))", ds, re.I))
    worlds = bool(re.match(r"the world", ds, re.I))
    required = bool(re.match(r"required", ds, re.I))
    n = num_of(ds)
    lo, hi = bound_range(bound)
    if kind == "geo":
        if required:
            if "z" in flags: z_issues.append((r[0], d, "z on a required default"))
            if lo is None or lo < 1: req_issues.append((r[0], d, bound))
        elif zero or worlds or (n is not None and n < 1):
            if "z" not in flags: geo_issues.append((r[0], d))
    if "z" in flags and not (zero or worlds):
        z_issues.append((r[0], d, "z needs 0, none or the world's"))
    # 11.1 "off value, after the cut": the off number of a D row is the owner's bound only as min/max
    for t in flags:
        m = re.fullmatch(r"off=(-?\d+)", t)
        if m and lo is not None and hi is not None:
            v = int(m.group(1))
            if v == hi: off_issues.append((r[0], t, "equals the owner's highest value: write off=max"))
            if v == lo and v not in (0, 1): off_issues.append((r[0], t, "equals the owner's lowest value: write off=min"))
        if t in ("off=min", "off=max") and (lo is None if t == "off=min" else hi is None):
            notes.append((r[0], f"{t}: the owner's bound is a named constant or absent (check by hand)"))
    # the cut: build the grid on the owner's default and count rows where a level was cut
    if cls == "D" and kind != "-" and n is not None and n >= 0 and lo is not None and hi is not None:
        off = None
        for t in flags:
            if t == "off=min": off = lo
            elif t == "off=max": off = hi
            else:
                m = re.fullmatch(r"off=(-?\d+)", t)
                if m: off = int(m.group(1))
        base_ok = lo <= n <= hi and (kind != "geo" or n >= 1)
        if off is not None and base_ok:
            lv = gridmod.geo_levels(n, lo, hi) if kind == "geo" else gridmod.lin_levels(n, int(kind[3:]), lo, hi)
            kept = gridmod.cut(lv, n, off)
            assert n in kept
            if len(kept) != len(lv):
                cut_rows.append((r[0], n, off))
            if any(v == off and v != n for v in kept):
                off_issues.append((r[0], f"off={off}", "an off level survived the cut"))
check("a geo row has an owner default of at least 1, or the flag z", not geo_issues, str(geo_issues))
check("a required default of a geo row has a lowest legal value of at least 1", not req_issues, str(req_issues))
check("the flag z only where the owner default is 0, none or the world's", not z_issues, str(z_issues))
check("an off number is not the owner's bound (min/max), and no off level survives the cut", not off_issues, str(off_issues))
print(f"     the cut removed a level on {len(cut_rows)} rows of the owners' defaults: {cut_rows[:12]}")

# `out` only where the solver writes (VIA 8, REF position 4)
OUT_OK = {("Commodity", "ReferencePrice"), ("RegionPrice", "Price"), ("WarmStart", "Prices"), ("WarmStart", "Wages"),
          ("WarmStart", "Groups"), ("Config section", "WarmStart")}      # not Pools: the table may seed it (VIA 8.3.3)
out_bad = []
for r in knob_rows:
    key = r[1].rpartition(".")
    if "out" in (r[8].split() if r[8] != "-" else []) and (key[0], key[2]) not in OUT_OK:
        out_bad.append((r[0], r[1]))
for r in group_rows:
    if "out" in (r[6].split() if r[6] != "-" else []):
        fields = [x.strip() for x in r[2].split(",")]
        if not all((r[1], f) in OUT_OK for f in fields):
            out_bad.append((r[0], r[1], r[2]))
check("`out` only on a field the solver writes (WarmStart Prices, Wages, Groups; ReferencePrice; RegionPrice.Price)", not out_bad, str(out_bad))

# ---- the flags `in` and `chk` against the first table of VIA 11 (11.1 "solver flags", FR-CLB-009) ----------
VIA = read("spec-viability.md")
_via = VIA.split("**The configuration paths the programme reads.**", 1)[1].split("Not read:", 1)[0]
via_rows = []
for _ln in _via.splitlines():
    if _ln.startswith("| `"):
        _cs = [c.strip() for c in _ln.strip().strip("|").split("|")]
        via_rows.append((_cs[0], _cs[2]))
check("VIA 11 has a table of paths", len(via_rows) >= 10, str(len(via_rows)))
STAR = "*"
PATHMAP = {   # the table-prefix rule of EXP 11.3.1 and the snake form, written out for each path of the table
    "world.upkeep": [("World", "Upkeep")], "role.<id>.upkeep": [("Role", "Upkeep")],
    "region.<id>.extra_upkeep": [("Region", "ExtraUpkeep")],
    "region.<id>.group.<role>.households": [("Group", "Households")],
    "recipe.<id>.duration": [("Recipe", "Duration")], "recipe.<id>.inputs": [("Recipe", "Inputs")],
    "recipe.<id>.outputs": [("Recipe", "Outputs"), ("Recipe output", STAR)],
    "recipe.<id>.requires": [("Recipe", "Requires"), ("Recipe tool", STAR)],
    "recipe.<id>.draws": [("Recipe", "Draws")], "recipe.<id>.sited": [("Recipe", "Sited")],
    "commodity.<id>.durability": [("Commodity", "Durability")], "commodity.<id>.shelf_life": [("Commodity", "ShelfLife")],
    "commodity.<id>.raw": [("Commodity", "Raw")],
    "region.<id>.yield.<recipe>": [("Yield", "All"), ("Yield", "Seasons"), ("Region and recipe", "YieldProfile")],
    "calendar.*": [("Calendar", STAR)],
    "role.<id>.expected_work_ticks": [("Role", "ExpectedWorkTicks")], "role.<id>.recipes": [("Role", "Recipes")],
    "resources.resource.<id>.*": [("Resource", STAR), ("AbundanceKnot", STAR), ("SeasonRegen", STAR)],
    "resources.endowment.<region>.<resource>.*": [("Endowment", STAR)],
    "resources.horizon_years": [("Resources", "HorizonYears")],
    "demand.want.<id>.*": [("Want", STAR), ("Want item", STAR)],
    "demand.taste.<role>.<want>.*": [("Taste", STAR)], "demand.region_taste.<region>.<want>.*": [("RegionTaste", STAR)],
    "modules.transport.modes.*": [("Mode", STAR)], "region.<id>.edge.<id>.distance": [("Edge", "Distance")],
    "module.external.node.*": [("OutsideNode", STAR), ("OutsideGate", STAR), ("OutsideLine", STAR),
                               ("OutsideSide", STAR), ("SeasonFactor", STAR)],
    "commodity.<id>.reference_price": [("Commodity", "ReferencePrice")],
    "region.<id>.reference_price.<commodity>": [("RegionPrice", "Price")],
    "world.belief_step": [("World", "BeliefStep")], "world.cost_floor": [("World", "CostFloor")],
    "world.cost_ceiling": [("World", "CostCeiling")], "world.ref_band_low": [("World", "RefBandLow")],
    "world.ref_band_high": [("World", "RefBandHigh")],
    "role.<id>.target_ratio": [("Role", "TargetRatio")], "role.<id>.target_spread": [("Role", "TargetSpread")],
    "world.welfare_window": [("World", "WelfareWindow")], "modules.infrastructure.*": [("Infrastructure", STAR), ("FacilityClass", STAR), ("InitialFacility", STAR)], "modules.storage.*": [("Storage", STAR), ("RoleRoom", STAR), ("StoreClass", STAR), ("Capacity", STAR)], "modules.institutions.control.<id>.*": [("Control", STAR)], "modules.tenure.terms.*": [("TenureTerm", STAR)], "world.window_cycles": [("World", "WindowCycles")],
}
via_paths, via_unmapped = [], []
for cell, moves in via_rows:
    stem = ""
    for tok in re.findall(r"`([^`]+)`", cell):
        if tok.startswith("."):
            tok = stem + tok
        elif "." not in tok:
            continue                       # `cost_share`, `wear_per_use`, `all`, `seasons`: annotations
        stem = tok.rsplit(".", 1)[0]
        kind = "in" if (moves.startswith("yes") or moves.startswith("the jobs")) else ("chk" if moves.startswith("no") else None)
        if tok not in PATHMAP or kind is None:
            via_unmapped.append(tok)
        else:
            via_paths.append((tok, kind))
check("every path of VIA 11's table is mapped to a registry (scope, field)", not via_unmapped, str(via_unmapped))
reg_fields = {}
for r in knob_rows:
    s, _, fld = r[1].rpartition(".")
    reg_fields[(s, fld)] = set(r[8].split()) if r[8] != "-" else set()
for r in group_rows:
    for fld in [x.strip() for x in r[2].split(",")]:
        reg_fields[(r[1], fld)] = set(r[6].split()) if r[6] != "-" else set()
exp_in, exp_chk = set(), set()
for tok, kind in via_paths:
    for sc, fld in PATHMAP[tok]:
        pairs = [k for k in reg_fields if k[0] == sc] if fld == STAR else [(sc, fld)]
        (exp_in if kind == "in" else exp_chk).update(pairs)
KEYS = {"ID", "Region", "Role", "Recipe", "Resource", "Season", "Want", "From", "To", "Commodity"}
CONTAINERS = {"Wants", "Flows", "Modes", "Nodes", "Lines", "Gates", "Buy", "Sell", "Resources", "Endowments",
              "Edges", "RegionPrices", "Items", "Roles", "Regions", "WarmStart"}
flag_bad = []
for (sc, fld), fl in sorted(reg_fields.items()):
    if fld in KEYS or fld in CONTAINERS or sc == "Solve option":
        continue
    if ("in" in fl) != ((sc, fld) in exp_in):
        flag_bad.append((sc, fld, "in" if "in" in fl else "-", "VIA says in" if (sc, fld) in exp_in else "VIA says not"))
    if ("chk" in fl) != ((sc, fld) in exp_chk):
        flag_bad.append((sc, fld, "chk" if "chk" in fl else "-", "VIA says chk" if (sc, fld) in exp_chk else "VIA says not"))
missing = [p for p in sorted(exp_in | exp_chk) if p not in reg_fields and p[1] not in KEYS and p[1] not in CONTAINERS]
check("`in` and `chk` are on exactly the data fields VIA 11 lists (a key and a container are not checked)",
      not flag_bad and not missing, str(flag_bad[:20]) + str(missing[:10]))
print(f"     VIA 11: {len(via_paths)} paths, {len(exp_in)} fields moving the table, {len(exp_chk)} fields read by a check or a report")

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
CFG_ONLY = {("Taste", "Role"), ("RegionTaste", "Region"), ("RegionTaste", "Want"), ("RegionTaste", "Scale")}   # owed to the README (23, X7)
unexpected = [p for p in extra if p[0] not in {"Solve option", "Presentation"} and p != ("World", "Clearing") and p not in CFG_ONLY]
check("registry-only pairs are the expected scopes", not unexpected, str(unexpected))
print(f"     {len(owner_pairs)} owner pairs ({sum(1 for v in owner_pairs.values() if len(v[1]) > 1)} with several owner sections), {len(extra)} registry-only")

# ---- coverage against the catalogue of spec-config.md 3.7 (11.1 "coverage, CFG 3.7") ----------
cat37 = CFG.split("### 3.7 The catalogue of fields", 1)[1].split("## 4. Constants and bounds", 1)[0]
cfg_names, cfg_nested = set(), []   # field names; (nested table name, field name)
for line in cat37.splitlines():
    if not line.startswith("| `"): continue
    cs = [c.strip() for c in line.strip().strip("|").split("|")]
    text = cs[1]
    # nested tables: `Name[]` ( ... ) or `Name` ( ... ), parentheses may nest once
    nested_spans = []
    for m in re.finditer(r"`([A-Z][A-Za-z]+)(?:\[\])?` \(((?:[^()]|\([^()]*\))*)\)", text):
        nested_spans.append((m.start(), m.end()))
        inner = m.group(2)
        if inner.strip() == "p":
            continue
        for t in re.finditer(r"`([A-Z][A-Za-z]+)`(?!\s*\(p\))", inner):
            cfg_nested.append((m.group(1), t.group(1)))
    for t in re.finditer(r"`([A-Z][A-Za-z]+)(?:\[\])?`(?!\s*\(p\))", text):
        if any(a <= t.start() < b for a, b in nested_spans):
            continue
        cfg_names.add(t.group(1))
    for name, _ in [(None, None)]:
        pass
PRESENTATION = {"Label", "UnitLabel", "DistanceLabel"}
all_fields = {f for (_, f) in covered}
miss_cfg = sorted(n for n in cfg_names if n not in all_fields and n not in PRESENTATION)
check("every field name of CFG 3.7 is a registry key or a group field", not miss_cfg, str(miss_cfg))


def singular(n):
    return n[:-3] + "y" if n.endswith("ies") else (n[:-1] if n.endswith("s") else n)


# a table of the catalogue whose registry scopes are not named for it (the catalogue flattens deeper tables)
ALIAS = {"DeathSchedule": ["knot"], "BirthSchedule": ["knot"], "Flows": ["moneyflow", "flowtarget"], "Nodes": ["outside", "seasonfactor"],
         "Modes": ["mode"], "Resources": ["resource", "abundance", "seasonregen"], "Requires": ["recipe tool"],
         "Wants": ["want"], "Clearing": ["world"]}
# tokens of a sub-structure that the catalogue flattens into its parent's parentheses: checked by name only
FLATTENED = {"Region", "Role", "Kind", "Season", "BP", "Share", "Factor", "Commodity", "Quantity", "Buy", "Sell", "ID", "Window"}
nested_miss, by_name = [], []
for tbl, fld in cfg_nested:
    if fld in PRESENTATION or fld in ("Items",) or tbl == "Clearing":
        continue
    sc = {s for (s, f) in covered if f == fld}
    subs = ALIAS.get(tbl, [singular(tbl).lower()])
    if any(any(x in s.lower() for x in subs) for s in sc):
        continue
    if tbl in ALIAS and fld in FLATTENED and sc:
        by_name.append((tbl, fld)); continue
    nested_miss.append((tbl, fld))
print(f"     nested fields checked by name only (the catalogue flattens them): {len(by_name)}")
check("every nested field of CFG 3.7 is under a scope named for its table", not nested_miss, str(sorted(set(nested_miss))))

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
            if not any(c == oc and sec_ok(n, own[1]) for c, n in items):
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
