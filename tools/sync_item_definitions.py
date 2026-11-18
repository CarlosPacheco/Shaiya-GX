#!/usr/bin/env python3
"""Synchronize SQL Items from the installed, already customized native Item.SData.

Dry-run unless --apply is given. Preserves explicitly customized forge/rune SQL definitions in both catalogs.
Never removes SQL rows or replaces a runtime database with a seed. Backups precede every write.
"""
import argparse
from contextlib import closing
import csv
import hashlib
import io
import json
import os
import tempfile
from pathlib import Path
import sqlite3
import struct

from update_recreation_content import Native, rune_aliases, runes

TABLE = 'PS_GameDefs_Items'
CLASSES = ('Attackfighter', 'Defensefighter', 'Patrolrogue', 'Shootrogue', 'Attackmage', 'Defensemage')

def decode_name(raw):
    raw = raw.split(b'\0', 1)[0]
    try:
        return raw.decode('utf-8')
    except UnicodeDecodeError:
        return raw.decode('cp1252', errors='replace')


def parse_items(raw):
    src = Native(raw)
    groups = src.word()
    if not 1 <= groups <= 256:
        raise ValueError('Expected plaintext EP5 Item.SData')
    rows = {}
    for group in range(1, groups + 1):
        count = src.word()
        if count > 255:
            raise ValueError('Too many definitions in an item family')
        for _ in range(count):
            name = decode_name(src.string())
            src.string()  # Description is client-only and remains untouched.
            b = src.take(81)
            typ, ident = b[0], b[1]
            key = typ * 1000 + ident
            if typ != group or ident == 0 or key in rows:
                raise ValueError(f'Invalid/duplicate item identity {typ}/{ident}')
            u16 = lambda off: struct.unpack_from('<H', b, off)[0]
            u32 = lambda off: struct.unpack_from('<I', b, off)[0]
            row = dict(ItemID=key, ItemName=name, Type=typ, TypeID=ident,
                       Reqlevel=u16(4), Country=b[6], Grow=b[13],
                       ReqVg=u16(28), ReqOg=b[30], ReqIg=b[31], Range=b[32],
                       AttackTime=b[33], Attrib=b[34], Special=b[35], Slot=b[36],
                       Quality=u16(37), Speed=b[65], Exp=b[66], Buy=u32(67),
                       Sell=u32(71), Grade=u16(75), Drop=u16(77), Server=b[79],
                       Count=255 if b[80] > 1 else b[80])
            row.update(zip(CLASSES, b[7:13]))
            for off, field in enumerate(('ReqStr', 'ReqDex', 'ReqRec', 'ReqInt', 'ReqWis', 'Reqluc')):
                row[field] = u16(16 + off * 2)
            for off, field in enumerate(('Effect1', 'Effect2', 'Effect3', 'Effect4',
                                         'ConstHP', 'ConstSP', 'ConstMP', 'ConstStr',
                                         'ConstDex', 'ConstRec', 'ConstInt', 'ConstWis', 'ConstLuc')):
                row[field] = u16(39 + off * 2)
            # SQL loader uses int16 for these native fields, int32 for prices.
            signed = ('ReqStr', 'ReqDex', 'ReqRec', 'ReqInt', 'ReqWis', 'ReqVg',
                      'Effect1', 'Effect2', 'Effect3', 'Effect4', 'ConstHP', 'ConstSP',
                      'ConstMP', 'ConstStr', 'ConstDex', 'ConstRec', 'ConstInt', 'ConstWis', 'ConstLuc')
            if any(row[f] > 32767 for f in signed) or max(row['Buy'], row['Sell']) > 2147483647:
                raise ValueError(f'Item {key} exceeds the server loader numeric range')
            rows[key] = row
    src.end()
    if not rows:
        raise ValueError('Empty item catalog')
    return rows


def validate_custom(rows):
    expected = {(101, r['type_id']): r for r in runes()}
    expected.update({(100, ident): r for ident, r in rune_aliases().items()})
    for (typ, ident), r in expected.items():
        item = rows.get(typ * 1000 + ident)
        if not item or item['Special'] != 62 or item['ReqVg'] != r['req_vg'] or item['Count'] != 255:
            raise ValueError(f'Customized recreation rune changed: {typ}/{ident}')
    ordinary = rows.get(100171)
    if not ordinary or ordinary['Special'] != 62 or ordinary['ReqVg'] != 0:
        raise ValueError('Ordinary recreation rune changed')
    patches = Path(__file__).parent / 'data/forge-item-patches.csv'
    forge = []
    with patches.open(newline='') as f:
        for r in csv.DictReader(f):
            key = int(r['type']) * 1000 + int(r['type_id'])
            item = rows.get(key)
            if not item or item['ReqVg'] != int(r['req_vg']) or int(r['stack']) and item['Count'] != int(r['stack']):
                raise ValueError(f'Customized forge object changed: {key}')
            forge.append(key)
    return {'recreation_runes': len(expected) + 1, 'forge_objects': len(forge),
            'protected_ids': sorted({100171, *[t * 1000 + i for t, i in expected], *forge})}


def patch_custom_records(raw, protected):
    """Preserve native names/descriptions/model/icon and every unprotected byte."""
    offsets = {'Type': (0, 'B'), 'TypeID': (1, 'B'), 'Reqlevel': (4, 'H'),
               'Country': (6, 'B'), 'Grow': (13, 'B'), 'ReqVg': (28, 'H'),
               'ReqOg': (30, 'B'), 'ReqIg': (31, 'B'), 'Range': (32, 'B'),
               'AttackTime': (33, 'B'), 'Attrib': (34, 'B'), 'Special': (35, 'B'),
               'Slot': (36, 'B'), 'Quality': (37, 'H'), 'Speed': (65, 'B'),
               'Exp': (66, 'B'), 'Buy': (67, 'I'), 'Sell': (71, 'I'),
               'Grade': (75, 'H'), 'Drop': (77, 'H'), 'Server': (79, 'B'), 'Count': (80, 'B')}
    offsets.update({k: (7+i, 'B') for i, k in enumerate(CLASSES)})
    offsets.update({k: (16+2*i, 'H') for i, k in enumerate(('ReqStr','ReqDex','ReqRec','ReqInt','ReqWis','Reqluc'))})
    offsets.update({k: (39+2*i, 'H') for i, k in enumerate(('Effect1','Effect2','Effect3','Effect4','ConstHP','ConstSP','ConstMP','ConstStr','ConstDex','ConstRec','ConstInt','ConstWis','ConstLuc'))})
    src = Native(raw)
    groups = src.word()
    out = bytearray(struct.pack('<I', groups))
    changed = []
    for _ in range(groups):
        n = src.word()
        out.extend(struct.pack('<I', n))
        for _ in range(n):
            start = src.pos
            src.string()
            src.string()
            end = src.pos
            body = bytearray(src.take(81))
            original = bytes(body)
            key = body[0]*1000+body[1]
            if key in protected:
                for field, (off, fmt) in offsets.items():
                    struct.pack_into('<'+fmt, body, off, protected[key][field])
            if body != original:
                changed.append(key)
            out.extend(raw[start:end])
            out.extend(body)
    src.end()
    return bytes(out), changed


def preserve_skill_references(rows, baseline):
    # EP5 Item.SData Range is one byte; SQL may hold a complete item-skill
    # ID above 255. Keep its high byte only when both low byte and level match.
    kept = {}
    for key, row in rows.items():
        old = baseline.get(key)
        if old and row['Type'] in (25, 44, 100, 101, 102) and 255 < old['Range'] <= 32767 and row['Range'] == old['Range'] & 255 and row['AttackTime'] == old['AttackTime']:
            row['Range'] = old['Range']
            kept[key] = old['Range']
    return kept


def connect_ro(path):
    return sqlite3.connect(path.resolve().as_uri() + '?mode=ro', uri=True)


def read_rows(db):
    db.row_factory = sqlite3.Row
    return {r['ItemID']: dict(r) for r in db.execute(f'SELECT * FROM {TABLE}')}


def plan(db, rows):
    current = read_rows(db)
    fields = set(next(iter(rows.values())))
    columns = {r[1] for r in db.execute(f'PRAGMA table_info({TABLE})')}
    if not fields <= columns:
        raise ValueError('Incompatible Items schema')
    changed, changed_fields, added = [], {}, []
    for key, row in rows.items():
        if key not in current:
            added.append(key)
        else:
            differences = [f for f, v in row.items() if current[key][f] != v]
            if differences:
                changed.append(key)
                changed_fields[key] = differences
    return {'before': len(current), 'after': len(current.keys() | rows.keys()),
            'updated': len(changed), 'added': len(added), 'changed_ids': changed,
            'added_ids': added, 'changed_fields': changed_fields,
            'server_only_preserved': sorted(current.keys() - rows.keys())}


def other_tables(db):
    """Hash every non-Items row; private values never leave the database."""
    result = {}
    tables = db.execute("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name").fetchall()
    for (name,) in tables:
        if name == TABLE:
            continue
        h, count = hashlib.sha256(), 0
        quoted = '"' + name.replace('"', '""') + '"'
        for row in db.execute(f'SELECT * FROM {quoted} ORDER BY rowid'):
            h.update(json.dumps(list(row), ensure_ascii=True, default=lambda x: x.hex(), separators=(',', ':')).encode())
            h.update(b'\n')
            count += 1
        result[name] = {'rows': count, 'sha256': h.hexdigest()}
    return result


def apply_rows(db, rows):
    fields = list(next(iter(rows.values())))
    quoted = ','.join('"' + f + '"' for f in fields)
    changed = ','.join(f'"{f}"=excluded."{f}"' for f in fields if f != 'ItemID')
    sql = f'INSERT INTO {TABLE}(RowID,{quoted}) VALUES({",".join("?" for _ in range(len(fields)+1))}) ON CONFLICT(ItemID) DO UPDATE SET {changed}'
    current = read_rows(db)
    next_id = db.execute(f'SELECT COALESCE(MAX(RowID),0)+1 FROM {TABLE}').fetchone()[0]
    values = []
    for key, row in rows.items():
        if key in current and all(current[key][f] == row[f] for f in fields):
            continue
        row_id = current[key]['RowID'] if key in current else next_id
        next_id += key not in current
        values.append([row_id, *[row[f] for f in fields]])
    db.executemany(sql, values)
    actual = read_rows(db)
    if any(any(actual[key][f] != v for f, v in row.items()) for key, row in rows.items()):
        raise ValueError('Post-sync definitions disagree with source')


def update_manifest(path):
    manifest = path.with_suffix('.manifest.json')
    if not manifest.exists():
        return
    data = json.loads(manifest.read_text())
    with closing(connect_ro(path)) as db:
        for table in data['tables']:
            q = '"' + table.replace('"', '""') + '"'
            data['tables'][table] = db.execute(f'SELECT COUNT(*) FROM {q}').fetchone()[0]
    data['sha256'] = hashlib.sha256(path.read_bytes()).hexdigest()
    temp = manifest.with_name(manifest.name + '.sync-next')
    temp.write_text(json.dumps(data, indent=2) + '\n')
    temp.replace(manifest)


def atomic_write(path, content):
    # Replacing the directory entry also avoids modifying other hardlinked
    # portable copies of a client's stack table.
    with tempfile.NamedTemporaryFile(dir=path.parent, prefix='.'+path.name+'.sync-', delete=False) as f:
        temp = Path(f.name)
        f.write(content)
        f.flush()
        os.fsync(f.fileno())
    try:
        temp.chmod(path.stat().st_mode & 0o777 if path.exists() else 0o644)
        temp.replace(path)
    finally:
        temp.unlink(missing_ok=True)


def require_stopped(source, databases):
    paths = {str(source.parent.parent.parent / 'Shaiya GX')}
    for db in databases:
        for name in ('gx_world', 'gx_login', 'gx_console'):
            paths.add(str(db.parent.parent / 'build' / name))
    proc = Path('/proc')
    if proc.is_dir():
        for entry in proc.iterdir():
            if not entry.name.isdecimal():
                continue
            try:
                exe = os.readlink(entry / 'exe')
            except OSError:
                continue
            if exe in paths:
                raise ValueError(f'Stop the GX process before synchronization: {exe}')


def synchronize(source, databases, backup_dir=None, apply=False, reference=None):
    raw = source.read_bytes()
    rows = parse_items(raw)
    custom = validate_custom(rows)
    databases = list(dict.fromkeys(p.resolve() for p in databases))
    with closing(connect_ro(reference or databases[0])) as db:
        baseline = read_rows(db)
    protected = {}
    for key in custom['protected_ids']:
        if key not in baseline:
            raise ValueError(f'Missing customized SQL definition {key}')
        protected[key] = {f: baseline[key][f] for f in rows[key] if f != 'ItemName'}
    patched, custom_patches = patch_custom_records(raw, protected)
    rows = parse_items(patched)
    validate_custom(rows)
    skill_refs = preserve_skill_references(rows, baseline)
    report = {'source_sha256': hashlib.sha256(raw).hexdigest(), 'definitions': len(rows),
              'preserved_custom': custom, 'preserved_wide_skill_references': skill_refs, 'custom_client_records_patched': custom_patches,
              'result_source_sha256': hashlib.sha256(patched).hexdigest(), 'applied': apply, 'databases': []}
    # Read-only validation of every target before any mutation.
    for path in databases:
        with closing(connect_ro(path)) as db:
            entry = dict(path=str(path), **plan(db, rows))
            current = read_rows(db)
            for key, values in protected.items():
                if key not in current or any(current[key][f] != v for f, v in values.items()):
                    raise ValueError(f'Customized SQL definitions differ between targets: {key}')
            entry['custom_behavior_changes'] = {}
            report['databases'].append(entry)
    if not apply:
        return report
    if backup_dir is None:
        raise ValueError('--backup-dir is required to apply')
    require_stopped(source, databases)
    if source.read_bytes() != raw:
        raise ValueError('Client definitions changed during preflight')
    backup_dir.mkdir(parents=True, exist_ok=False, mode=0o700)
    (backup_dir / 'Item.SData').write_bytes(raw)
    client_stack = source.parent.parent / 'item-stack-limits.csv'
    if client_stack.exists():
        (backup_dir / 'client.item-stack-limits.csv').write_bytes(client_stack.read_bytes())
    for n, path in enumerate(databases):
        with closing(connect_ro(path)) as db, closing(sqlite3.connect(backup_dir / f'database-{n}.sqlite')) as backup:
            db.backup(backup)
        manifest = path.with_suffix('.manifest.json')
        if manifest.exists():
            (backup_dir / f'database-{n}.manifest.json').write_bytes(manifest.read_bytes())
    for n, path in enumerate(databases):
        with closing(sqlite3.connect(path)) as db:
            db.execute('BEGIN IMMEDIATE')
            try:
                before = other_tables(db)
                apply_rows(db, rows)
                if other_tables(db) != before:
                    raise ValueError('A non-Items table changed')
                if [r[0] for r in db.execute('PRAGMA integrity_check')] != ['ok']:
                    raise ValueError('SQLite integrity check failed')
                db.commit()
            except BaseException:
                db.rollback()
                raise
            report['databases'][n]['other_tables_unchanged'] = True
            report['databases'][n]['other_tables'] = before
            report['databases'][n]['remaining_changes'] = plan(db, rows)['updated'] + plan(db, rows)['added']
        update_manifest(path)
    if source.read_bytes() != raw:
        raise ValueError('Client definitions changed during synchronization')
    if patched != raw:
        atomic_write(source, patched)
    # Match the selected server's stack export without touching Cash/products.
    with closing(connect_ro(databases[0])) as db:
        limits = db.execute(f'SELECT ItemID,CASE WHEN Count>1 THEN 255 ELSE 1 END FROM {TABLE} ORDER BY ItemID').fetchall()
    output = io.StringIO(newline='')
    writer = csv.writer(output, lineterminator='\n')
    writer.writerow(['item_id', 'max_stack'])
    writer.writerows(limits)
    atomic_write(client_stack, output.getvalue().encode())
    report['client_customizations_preserved'] = True
    report['noncustom_client_bytes_unchanged'] = True
    return report


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument('--item-sdata', type=Path, required=True)
    ap.add_argument('--database', type=Path, action='append', required=True)
    ap.add_argument('--backup-dir', type=Path)
    ap.add_argument('--reference-database', type=Path, help='Optional pre-sync SQL baseline for custom/extended references')
    ap.add_argument('--apply', action='store_true')
    ap.add_argument('--report', type=Path, required=True)
    args = ap.parse_args()
    result = synchronize(args.item_sdata, args.database, args.backup_dir, args.apply, args.reference_database)
    args.report.parent.mkdir(parents=True, exist_ok=True)
    args.report.write_text(json.dumps(result, indent=2) + '\n')
    print(json.dumps({k: v for k, v in result.items() if k != 'databases'}))
    for db in result['databases']:
        print({k: db[k] for k in ('path', 'before', 'after', 'updated', 'added', 'server_only_preserved')})


if __name__ == '__main__':
    main()
