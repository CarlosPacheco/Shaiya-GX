#!/usr/bin/env python3
"""Synchronize native mall quantities and stack definitions in explicit offline DBs.

Uses the installed Item.SData for mall stackability (including custom items),
keeps non-mall stackability from either source, and preserves inventory rows.
Backups and disposable validation precede writes. Never replaces a runtime DB.
"""
import argparse
from contextlib import closing
import csv
import hashlib
import io
import json
import os
from pathlib import Path
import re
import shutil
import sqlite3
import struct

from cash_path import existing_cash_path
from update_recreation_content import Native, read_cash, string


def read_items(raw):
    src = Native(raw)
    groups = []
    items = {}
    for _ in range(src.word()):
        group = []
        for _ in range(src.word()):
            name, description, body = src.string(), src.string(), bytearray(src.take(81))
            identity = body[0] * 1000 + body[1]
            if identity in items:
                raise ValueError(f'duplicate native item {identity}')
            record = [name, description, body]
            items[identity] = record
            group.append(record)
        groups.append(group)
    src.end()
    return groups, items


def encode_items(groups):
    out = bytearray(struct.pack('<I', len(groups)))
    for group in groups:
        out.extend(struct.pack('<I', len(group)))
        for name, description, body in group:
            out.extend(string(name) + string(description) + body)
    return bytes(out)


def encode_cash(products):
    out = bytearray(struct.pack('<I', len(products)))
    for p in products:
        out.extend(struct.pack('<4I', *p['nums']))
        for entry in p['items']:
            out.extend(struct.pack('<IB', *entry))
        for field in p['fields']:
            out.extend(string(field))
    return bytes(out)


def text(raw):
    return raw.rstrip(b'\0').decode('utf-8')


def prepare(item_raw, cash_raw, counts):
    groups, items = read_items(item_raw)
    products = read_cash(cash_raw)
    sold = {identity for p in products for identity, count in p['items'] if identity}
    codes = set()
    for p in products:
        code = text(p['fields'][1])
        if not code or len(code.encode()) > 20 or code in codes:
            raise ValueError(f'invalid/duplicate wire product code {code!r}')
        codes.add(code)
        if not any(i for i, n in p['items']):
            raise ValueError(f'empty product {code}')
        for identity, quantity in p['items']:
            if bool(identity) != bool(quantity):
                raise ValueError(f'incomplete product slot {code}')
            if identity and (identity not in counts or identity not in items or not items[identity][2][80]):
                raise ValueError(f'undefined mall item {code}: {identity}')
    for record in items.values():
        if record[2][80] > 1:
            record[2][80] = 255
    limits = {}
    for identity, original in counts.items():
        record = items.get(identity)
        if record is None:
            raise ValueError(f'SQL item missing in native table {identity}')
        # Native mall entries intentionally include stackable custom items that
        # occupy formerly unused SQL rows. Equipment stays unstackable.
        native = record[2][80]
        stackable = native > 1 if identity in sold else native > 1 or original > 1
        limit = 255 if stackable else (1 if native or original else 0)
        record[2][80] = limit
        limits[identity] = limit
    changed = 0
    for p in products:
        for entry in p['items']:
            identity, quantity = entry
            if identity and limits[identity] > 1:
                changed += quantity != 255
                entry[1] = 255
        name = re.sub(r'\s+x\d+$', '', text(p['fields'][0]))
        entries = [(i, n) for i, n in p['items'] if i]
        if len(entries) == 1:
            name += f' x{entries[0][1]}'
        contents = '; '.join(f'{text(items[i][0])} x{n}' for i, n in entries)
        # Quantity text is derived from actual slots, never from old prose.
        p['fields'][0] = name.encode()
        p['fields'][2] = ('Contains: ' + contents + '.').encode()
        need = sum((n + max(limits[i], 1) - 1) // max(limits[i], 1) for i, n in entries)
        if need > 120:
            raise ValueError(f'product exceeds empty inventory: {text(p["fields"][1])}, {need} slots')
    return encode_items(groups), encode_cash(products), products, limits, changed


def apply_database(path, products, limits, item_names):
    with closing(sqlite3.connect(path)) as db, db:
        def mutable_fingerprint():
            result = {}
            for (table,) in db.execute("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name").fetchall():
                if table in ('PS_GameDefs_Items', 'PS_GameDefs_ProductList'):
                    continue
                rows = db.execute('SELECT * FROM "'+table.replace('"','""')+'"').fetchall()
                result[table] = hashlib.sha256('\n'.join(sorted(map(repr, rows))).encode()).hexdigest()
            return result
        before = mutable_fingerprint()
        db.execute('BEGIN IMMEDIATE')
        db.executemany('UPDATE PS_GameDefs_Items SET Count=? WHERE ItemID=?', [(n, i) for i, n in limits.items()])
        # Placeholder labels are not functional identities. Fill only their
        # names; no unrelated combat/effect properties are inferred here.
        db.executemany("UPDATE PS_GameDefs_Items SET ItemName=? WHERE ItemID=? AND ItemName LIKE '???%'", [(n, i) for i, n in item_names.items()])
        columns = ['ProductName', 'BuyCost'] + [c for i in range(1, 25) for c in (f'ItemID{i}', f'ItemCount{i}')]
        assignments = ','.join(f'{c}=?' for c in columns)
        for p in products:
            values = [text(p['fields'][0]), p['nums'][3]] + [v for entry in p['items'] for v in entry]
            code = text(p['fields'][1])
            result = db.execute('UPDATE PS_GameDefs_ProductList SET '+assignments+' WHERE RTRIM(ProductCode)=?', values+[code])
            if result.rowcount != 1:
                raise ValueError(f'product not unique in SQL {code}')
        if db.execute('SELECT COUNT(*) FROM PS_GameDefs_ProductList').fetchone()[0] != len(products):
            raise ValueError('SQL-only products require explicit reconciliation')
        if mutable_fingerprint() != before:
            raise ValueError('migration modified tables outside Items/ProductList')
        if db.execute('PRAGMA quick_check').fetchone()[0] != 'ok':
            raise ValueError('database integrity check failed')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--client-data', type=Path, required=True)
    parser.add_argument('--database', type=Path, action='append', required=True)
    parser.add_argument('--backup-dir', type=Path, required=True)
    args = parser.parse_args()
    args.backup_dir.mkdir(parents=True, exist_ok=False)
    item_path, cash_path = args.client_data/'Item/Item.SData', existing_cash_path(args.client_data)
    item_raw, cash_raw = item_path.read_bytes(), cash_path.read_bytes()
    original_counts = []
    for path in args.database:
        if not path.is_file():
            raise ValueError(f'missing explicit DB {path}')
        with closing(sqlite3.connect(path.resolve().as_uri()+'?mode=ro', uri=True)) as db:
            original_counts.append(dict(db.execute('SELECT ItemID,Count FROM PS_GameDefs_Items')))
    if any(c != original_counts[0] for c in original_counts):
        raise ValueError('DB stack definitions differ; review before synchronizing')
    patched_items, patched_cash, products, limits, changed = prepare(item_raw, cash_raw, original_counts[0])
    _, records = read_items(patched_items)
    names = {i:text(r[0]) for i,r in records.items() if i in limits and text(r[0])}
    shutil.copy2(item_path, args.backup_dir/'Item.SData')
    shutil.copy2(cash_path, args.backup_dir/'Cash.SData')
    for i, path in enumerate(args.database):
        with closing(sqlite3.connect(path)) as source, closing(sqlite3.connect(args.backup_dir/f'database-{i}.sqlite')) as backup:
            source.backup(backup)
        trial = args.backup_dir/f'trial-{i}.sqlite'
        shutil.copy2(args.backup_dir/f'database-{i}.sqlite', trial)
        apply_database(trial, products, limits, names)
        trial.unlink()
    for path in args.database:
        apply_database(path, products, limits, names)
        manifest = path.with_suffix('.manifest.json')
        if manifest.exists():
            doc = json.loads(manifest.read_text())
            doc['sha256'] = hashlib.sha256(path.read_bytes()).hexdigest()
            manifest.write_text(json.dumps(doc, indent=2)+'\n')
    out = io.StringIO(); writer = csv.writer(out, lineterminator='\n')
    writer.writerow(['item_id','max_stack'])
    writer.writerows((i, max(n,1)) for i,n in sorted(limits.items()))
    targets = [(item_path, patched_items), (cash_path, patched_cash), (args.client_data/'item-stack-limits.csv', out.getvalue().encode())]
    for path, raw in targets:
        staged = path.with_suffix(path.suffix+'.gx-new')
        staged.write_bytes(raw); os.replace(staged, path)
    print(json.dumps({'products':len(products), 'stack_definitions':len(limits), 'quantity_changes':changed, 'backup':str(args.backup_dir)}))

if __name__ == '__main__':
    main()
