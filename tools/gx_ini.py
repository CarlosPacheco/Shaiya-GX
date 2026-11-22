#!/usr/bin/env python3
"""Convert the native WorldMap.cfg / Dungeon_MiniMap.cfg / filter.txt to GX INI files.

    gx_ini.py DATA_DIR     rewrites WorldMap.INI, DungeonMiniMaps.INI and Filter.INI
                           next to the old files (the old files are not removed).
"""
import configparser
import re
import sys
from pathlib import Path

COUNTRIES = {0: 'Both', 1: 'Light', 2: 'Fury'}

WORLD_HEADER = """; World map regions, listed in display order.
;   Country  Both, Light or Fury (who can preview the region)
;   Map      connected map ID, -1 for none
;   Click    hover and click box in pixels of the 512x512 wm_map image: left, top, right, bottom
;   Image    top-left of the region overlay image: x, y
"""

DUNGEON_HEADER = """; Dungeon minimap calibration: pixels of the numbered MiniMap image per world unit.
; One [Map ID] section per map; ScaleX and ScaleZ are the horizontal scales.
"""

FILTER_TEMPLATE = """; Banned words filter. The User Options checkbox masks every match in received
; player chat and speech bubbles. One word or phrase per Word= line; matching is
; literal and ignores case. Restart the client after editing.
[Filter]
;Word=example
"""


def read(path):
    raw = path.read_bytes()
    for encoding in ('utf-8-sig', 'cp949', 'latin1'):
        try:
            return raw.decode(encoding)
        except UnicodeDecodeError:
            pass


def parse(text):
    parser = configparser.ConfigParser(comment_prefixes=('#', ';'), inline_comment_prefixes=None)
    parser.optionxform = str.lower
    parser.read_string(text)
    return parser


def comments_by_section(text):
    """Comment line that directly precedes each numbered section, if any."""
    names, last = {}, None
    for line in text.splitlines():
        line = line.strip()
        if line.startswith('#'):
            last = line[1:].strip()
        elif re.fullmatch(r'\[\d+\]', line):
            names[int(line[1:-1])] = last
            last = None
        elif line:
            last = None
    return names


def worldmap_to_ini(text):
    parsed, names = parse(text), comments_by_section(text)
    out = [WORLD_HEADER]
    for n in range(parsed.getint('WorldMap', 'count')):
        row = parsed[str(n)]
        number = row.getint('mapnum')
        out.append('')
        if names.get(n):
            out.append(f'; {names[n]}')
        out.append(f'[Map {number}]' if number >= 0 else f'[Region {n}]')
        out.append(f"Country={COUNTRIES[row.getint('country')]}")
        out.append(f'Map={number}')
        out.append(f"Click={row.getint('click_left')},{row.getint('click_top')},{row.getint('click_right')},{row.getint('click_bottom')}")
        out.append(f"Image={row.getint('imgpos_x')},{row.getint('imgpos_y')}")
    return '\n'.join(out) + '\n'


def dungeon_to_ini(text):
    parsed = parse(text)
    out = [DUNGEON_HEADER]
    for n in range(parsed.getint('WorldMap', 'count')):
        row = parsed[str(n)]
        out += ['', f"[Map {row.getint('mapnum')}]",
                f"ScaleX={row.getint('imgpos_x') / 1000:.3f}", f"ScaleZ={row.getint('imgpos_y') / 1000:.3f}"]
    return '\n'.join(out) + '\n'


def filter_to_ini(text):
    words = []
    for line in text.splitlines():
        match = re.match(r'^\s*\d+\s+"(.*)"\s*$', line)
        if match and match.group(1):
            words.append(match.group(1))
    return FILTER_TEMPLATE + ''.join(f'Word={w}\n' for w in words)


def main():
    data = Path(sys.argv[1])
    for old, new, convert in (('WorldMap.cfg', 'WorldMap.INI', worldmap_to_ini),
                              ('Dungeon_MiniMap.cfg', 'DungeonMiniMaps.INI', dungeon_to_ini),
                              ('filter.txt', 'Filter.INI', filter_to_ini)):
        source = next((p for p in data.iterdir() if p.name.casefold() == old.casefold()), None)
        if source is None:
            print('missing', old)
            continue
        (data / new).write_text(convert(read(source)), encoding='utf-8', newline='\n')
        print('wrote', new)


if __name__ == '__main__':
    main()
