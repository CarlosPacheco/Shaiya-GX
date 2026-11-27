#!/usr/bin/env python3
"""Append the GX SysMsg rows to a data folder's sysmsg-uni.txt (idempotent).

internal/data/text/sysmsg-gx.txt is the canonical list of rows the client adds
(ids 12001+) and of default texts for native ids. The client already embeds
them as fallbacks; this writes the missing ones into the data file so each
language's table is complete and can be translated in place.

Usage: tools/patch_sysmsg.py <data dir> [--native]   (--native also adds the
native ids the file lacks)
"""
import re, sys
from pathlib import Path

root = Path(__file__).resolve().parents[1]
rows = {}
for line in (root / "internal/data/text/sysmsg-gx.txt").read_text(encoding="utf8").splitlines():
    m = re.match(r'^(\d+)\s+"(.*)"\s*$', line)
    if m:
        rows[int(m.group(1))] = m.group(2)
target = Path(sys.argv[1]) / "sysmsg-uni.txt"
raw = target.read_bytes()
if raw[:2] != b"\xff\xfe":
    sys.exit("sysmsg-uni.txt must be UTF-16LE with BOM")
text = raw[2:].decode("utf-16-le")
have = {int(m.group(1)) for m in re.finditer(r"^(\d+)\s", text, re.M)}
eol = "\r\n" if "\r\n" in text else "\n"
add = [(i, t) for i, t in sorted(rows.items()) if i not in have and (i >= 12001 or "--native" in sys.argv)]
# GX ids that no longer exist in the canonical list are removed from the data file
stale = {i for i in have if i >= 12001 and i not in rows}
if stale:
    text = "".join(l for l in text.splitlines(True) if not (re.match(r"^(\d+)\s", l) and int(re.match(r"^(\d+)", l).group(1)) in stale))
    print(f"removed {len(stale)} stale GX rows")
# GX rows follow the canonical list: an id reused for a new text replaces the
# old row instead of keeping a stale meaning.
changed = 0
def refresh(m):
    global changed
    i = int(m.group(1))
    if i >= 12001 and i in rows and m.group(2) != rows[i]:
        changed += 1
        return f'{i}\t\t"{rows[i]}"'
    return m.group(0)
text = re.sub(r'^(\d+)\s+"(.*)"[ \t]*(?=\r?$)', refresh, text, flags=re.M)
if changed:
    print(f"updated {changed} GX rows")
if not add and not stale and not changed:
    print("nothing to add")
    sys.exit()
if add and not text.endswith(eol):
    text += eol
text += "".join(f'{i}\t\t"{t}"{eol}' for i, t in add)
target.write_bytes(b"\xff\xfe" + text.encode("utf-16-le"))
print(f"added {len(add)} rows to {target}")
