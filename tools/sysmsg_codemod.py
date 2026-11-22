#!/usr/bin/env python3
"""Move hard-coded user text in internal/app into SysMsg ids.

Idempotent codemod used for the client text review: every literal passed to a
known message helper is replaced by an id. Native ids are preferred (exact
text match in the real sysmsg-uni.txt); otherwise a GX id from 12001 is
allocated and its text is stored in internal/data/text/sysmsg-gx.txt (embedded
defaults, also written into the data folder by tools/patch_sysmsg.py).

Usage: tools/sysmsg_codemod.py <path to sysmsg-uni.txt>
"""
import ast, os, re, sys
ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
GX = os.path.join(ROOT, "internal/data/text/sysmsg-gx.txt")
APP = os.path.join(ROOT, "internal/app")
BASE = 12001

def read_rows(text):
    rows = {}
    for line in text.splitlines():
        m = re.match(r'^(\d+)\s+"(.*)"\s*$', line)
        if m:
            rows[int(m.group(1))] = m.group(2).replace("\\n", "\n")
    return rows

native = read_rows(open(sys.argv[1], "rb").read().decode("utf-16"))
gx = read_rows(open(GX, encoding="utf8").read()) if os.path.exists(GX) else {}
by_native = {}
for k, v in native.items():
    by_native.setdefault(v.strip().lower().rstrip("."), k)
by_gx = {v: k for k, v in gx.items() if k >= BASE}
next_id = max([BASE - 1] + [k for k in gx if k >= BASE]) + 1

L = r'"((?:[^"\\]|\\.)*)"'
def unq(s): return ast.literal_eval('"' + s + '"')

def id_for(text):
    global next_id
    key = text.strip().lower().rstrip(".")
    if key in by_native:
        return by_native[key]
    if text in by_gx:
        return by_gx[text]
    gx[next_id] = text; by_gx[text] = next_id
    next_id += 1
    return next_id - 1

def register(i, text):
    """Default text for an id the call site already named."""
    if i in native:
        return  # the real sysmsg-uni.txt owns native ids
    gx.setdefault(i, text)

def sub_simple(pat, fmt):
    def f(m):
        return fmt.format(id_for(unq(m.group(1))))
    return re.compile(pat), f

simple = [
    (r'a\.backToLogin\(' + L + r'\)', 'a.backToLoginID({})'),
    (r'a\.questMessage\(' + L + r'\)', 'a.questMessageID({})'),
    (r'a\.accountMessage\.Open\(' + L + r',', 'a.accountMessage.Open(a.tr({}),'),
]
changed = 0
for f in sorted(os.listdir(APP)):
    if not f.endswith(".go") or f.endswith("_test.go"):
        continue
    path = os.path.join(APP, f)
    src = open(path, encoding="utf8").read()
    out = src
    for pat, fmt in simple:
        out = re.sub(pat, lambda m, fmt=fmt: fmt.format(id_for(unq(m.group(1)))), out)
    # id + fallback helpers: the fallback moves to the table
    def notice(m):
        register(int(m.group(2)), unq(m.group(4))) if m.group(2).isdigit() else None
        return f"a.localizedNotice({m.group(1)}, {m.group(2)}, {m.group(3)})"
    out = re.sub(r'a\.localizedNotice\(([^,]+),\s*([0-9A-Za-z_.]+),\s*([^,]+),\s*' + L + r'\)', notice, out)
    def pm(m):
        i, text = int(m.group(1)), unq(m.group(2))
        if i == 0:
            i = id_for(text)
        else:
            register(i, text)
        return f"a.personalMarketNotice({i}, "
    out = re.sub(r'a\.personalMarketNotice\((\d+),\s*' + L + r',\s*', pm, out)
    def two(name):
        def g(m):
            register(int(m.group(1)), unq(m.group(2)))
            return f"a.tr({m.group(1)})"
        return re.compile(r'a\.' + name + r'\((\d+),\s*' + L + r'\)'), g
    for name in ("msgOr", "accountText"):
        pat, g = two(name)
        out = pat.sub(g, out)
    if out != src:
        open(path, "w", encoding="utf8").write(out)
        changed += 1
rows = sorted(gx.items())
with open(GX, "w", encoding="utf8") as fh:
    fh.write("// GX SysMsg rows: ids 12001+ are GX-only; lower ids are defaults for native ids that a\n// sysmsg-uni.txt may lack. Same format as sysmsg-uni.txt (UTF-8 here).\n")
    for k, v in rows:
        fh.write(f'{k}\t\t"{v.replace(chr(10), chr(92) + "n")}"\n')
print("files changed:", changed, "rows:", len(rows), "next id:", next_id)
