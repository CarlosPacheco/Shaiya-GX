#!/usr/bin/env python3
"""Move map name rows in a loose GX client's UTF-16LE SysMsg file."""
import argparse
import os
from pathlib import Path
import re
import shutil
import tempfile

# Original map-to-message mapping, used only to migrate existing text catalogs.
LEGACY_MESSAGES = {
    0: 5068, 1: 5069, 2: 5070, 3: 5071, 4: 5071, 5: 5072, 6: 5072, 7: 5073,
    8: 5073, 9: 5096, 10: 5096, 11: 5096, 12: 5097, 13: 5097, 14: 5097, 15: 5098,
    16: 5098, 17: 5098, 18: 5074, 19: 5099, 20: 5100, 21: 5101, 22: 5101, 23: 5102,
    24: 5102, 25: 5103, 26: 5104, 27: 5104, 28: 5105, 29: 5106, 30: 5107, 31: 5108,
    32: 5109, 33: 5110, 34: 5111, 35: 5112, 36: 5113, 37: 5428, 38: 5429, 39: 5430,
    40: 5430, 41: 5431, 42: 5419, 43: 5375, 44: 5376, 45: 5378, 46: 5379, 47: 5380,
    48: 5405, 49: 5405, 50: 5406, 51: 5407, 52: 5407, 53: 5408, 54: 5408, 55: 5409,
    56: 5409, 57: 5410, 58: 5411, 59: 5412, 60: 5413, 61: 5414, 62: 5415, 63: 5416,
    64: 5432, 65: 5433, 66: 5433, 67: 5434, 68: 5425, 69: 5426, 70: 5427, 71: 5435,
    72: 5436, 73: 5427, 100: 5437, 101: 5437, 102: 5417, 103: 5417,
    104: 5417, 105: 5417, 106: 5417, 107: 5417, 108: 5417,
}

FIRST_MAP_MESSAGE = 11001
INITIAL_MAP_COUNT = 100
ROW = re.compile(r'^\s*(\d+)[ \t]+(.*?)\s*$')


def map_name_rows(source, map_ids):
    messages = {}
    for line in source.splitlines():
        row = ROW.fullmatch(line)
        if row:
            messages[int(row[1])] = row[2]

    rows = []
    extra_maps = sorted(i for i in LEGACY_MESSAGES if i >= INITIAL_MAP_COUNT and i in map_ids)
    for map_id in (*range(INITIAL_MAP_COUNT), *extra_maps):
        message_id = FIRST_MAP_MESSAGE + map_id
        if message_id in messages:
            continue
        name = messages.get(LEGACY_MESSAGES.get(map_id), '').strip('"').strip()
        if map_id not in map_ids or not name:
            name = f'Map {map_id}'
        rows.append(f'{message_id}\t\t"{name}"')
    return rows


def migrate(path, backup_path):
    raw = path.read_bytes()
    bom = b'\xff\xfe'
    if raw.startswith(b'\xfe\xff'):
        raise ValueError('SysMsg must use UTF-16LE')
    prefix = bom if raw.startswith(bom) else b''
    source = raw[len(prefix):].decode('utf-16le')
    worlds = next((entry for entry in path.parent.iterdir()
                   if entry.is_dir() and entry.name.casefold() == 'world'), None)
    if worlds is None:
        raise ValueError('World directory missing beside SysMsg')
    map_ids = {int(entry.stem) for entry in worlds.iterdir()
               if entry.is_file() and entry.suffix.casefold() == '.wld'
               and entry.stem.isascii() and entry.stem.isdecimal()}
    if not map_ids:
        raise ValueError('No numeric map files found in the World directory')

    rows = map_name_rows(source, map_ids)
    legacy_ids = set(LEGACY_MESSAGES.values())
    kept = []
    removed = 0
    for line in source.splitlines(keepends=True):
        match = ROW.fullmatch(line.rstrip('\r\n'))
        if match and int(match[1]) in legacy_ids:
            removed += 1
        else:
            kept.append(line)
    if not rows and not removed:
        return 0, 0
    result = ''.join(kept)
    if rows:
        ending = '\r\n' if '\r\n' in source else '\n'
        separator = '' if not result or result.endswith('\n') else ending
        result += separator + ending.join(rows) + ending
    backup_path.parent.mkdir(parents=True, exist_ok=True)
    with backup_path.open('xb') as backup:
        backup.write(raw)
    with tempfile.NamedTemporaryFile(dir=path.parent, delete=False) as temporary:
        temp = Path(temporary.name)
        try:
            temporary.write(prefix + result.encode('utf-16le'))
            temporary.close()
            shutil.copymode(path, temp)
            os.replace(temp, path)
        finally:
            temp.unlink(missing_ok=True)
    return len(rows), removed


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('sysmsg', type=Path)
    parser.add_argument('--backup', type=Path, required=True)
    args = parser.parse_args()
    added, removed = migrate(args.sysmsg, args.backup)
    print(f'Added {added} map name rows and removed {removed} legacy rows.')


if __name__ == '__main__':
    main()
