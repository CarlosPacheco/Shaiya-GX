"""Keep stack limits in the native item table and check them against SQLite."""
from contextlib import closing
from pathlib import Path
import sqlite3

from update_recreation_content import Native

RETIRED_STACK_CSV = 'item-stack-limits.csv'


def item_count_offsets(raw):
    source = Native(raw)
    groups = source.word()
    if not 1 <= groups <= 256:
        raise ValueError('Expected plaintext EP5 Item.SData')
    for _ in range(groups):
        for _ in range(source.word()):
            source.string()
            source.string()
            body = source.take(81)
            yield body[0] * 1000 + body[1], source.pos - 1
    source.end()


def patch_stack_limits(raw, limits):
    result = bytearray(raw)
    offsets = dict(item_count_offsets(raw))
    for item, limit in limits.items():
        if item not in offsets:
            raise ValueError(f'Item {item} is missing from Item.SData')
        result[offsets[item]] = limit
    return bytes(result)


def item_sdata_path(data):
    for directory in data.iterdir():
        if directory.name.casefold() == 'item' and directory.is_dir():
            for path in directory.iterdir():
                if path.name.casefold() == 'item.sdata' and path.is_file():
                    return path
    raise ValueError(f'Item/Item.SData is missing in {data}')


def reconcile_stack_limits(data, database):
    path = item_sdata_path(data)
    raw = path.read_bytes()
    offsets = dict(item_count_offsets(raw))
    with closing(sqlite3.connect(Path(database).resolve().as_uri() + '?mode=ro', uri=True)) as db:
        limits = {t * 1000 + i: 255 if (n or 0) > 1 else 1 for t, i, n in
                  db.execute('SELECT Type, TypeID, Count FROM PS_GameDefs_Items')}
    shared = offsets.keys() & limits.keys()
    if not shared:
        raise ValueError('Item.SData and SQLite have no item definitions in common')
    changes = {item: limits[item] for item in shared if (raw[offsets[item]] > 1) != (limits[item] > 1)}
    if changes:
        path.write_bytes(patch_stack_limits(raw, changes))
    return len(changes)
