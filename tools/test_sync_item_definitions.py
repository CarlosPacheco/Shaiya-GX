from contextlib import closing
import csv
import json
from pathlib import Path
import sqlite3
import struct
import tempfile
import unittest

import sync_item_definitions as sync
from update_recreation_content import runes, rune_aliases, string


def fixture_source():
    records = {}
    def add(typ, ident, special=0, selector=0, count=255):
        b = bytearray(81)
        b[0], b[1], b[35], b[80] = typ, ident, special, count
        struct.pack_into('<H', b, 28, selector)
        records[typ, ident] = b
        return b
    for r in runes():
        add(101, r['type_id'], 62, r['req_vg'])
    for ident, r in rune_aliases().items():
        add(100, ident, 62, r['req_vg'])
    add(100, 171, 62)
    with (Path(sync.__file__).parent / 'data/forge-item-patches.csv').open(newline='') as f:
        for r in csv.DictReader(f):
            typ, ident = int(r['type']), int(r['type_id'])
            special = 0 if typ == 95 else 58 if (typ, ident) in ((44,237),(100,140),(100,141),(100,204)) else 36
            add(typ, ident, special, int(r['req_vg']))
    # The known customized hammer mismatch must be preserved from SQL.
    records[100,204][35] = 73
    b = add(1, 1, count=1)
    struct.pack_into('<H', b, 26, 44082)  # ReqLuc is unsigned, above int16.
    struct.pack_into('<H', b, 53, 65)
    struct.pack_into('<H', b, 75, 3210)
    struct.pack_into('<H', b, 77, 102)
    add(30, 244)[66] = 150
    add(44, 213, special=50, count=1)[32:34] = bytes((17, 1))
    raw = bytearray(struct.pack('<I', 101))
    for typ in range(1, 102):
        entries = [(i, b) for (t, i), b in records.items() if t == typ]
        raw.extend(struct.pack('<I', len(entries)))
        for ident, b in sorted(entries):
            raw.extend(string(f'Item {typ}/{ident}') + string('Preserve description/icon/model') + b)
    return bytes(raw)


def fixture_db(path, rows):
    columns = list(next(iter(rows.values())))
    with closing(sqlite3.connect(path)) as db, db:
        ddl = ','.join(f'"{k}" ' + ('TEXT' if k == 'ItemName' else 'INTEGER') + (' PRIMARY KEY' if k == 'ItemID' else '') for k in columns)
        db.execute(f'CREATE TABLE {sync.TABLE}(RowID INTEGER,{ddl})')
        for n, (key, row) in enumerate(rows.items(), 1):
            if key == 30244:
                continue  # A definition missing from the old DB.
            row = dict(row)
            if key == 100204:
                row['Special'] = 58
            if key == 100049:
                row['Sell'], row['Grade'] = 1000, 944
            if key == 1001:
                row['ConstStr'] = 3
            if key == 44213:
                row['Range'] = 273
            q = ','.join('?' for _ in range(len(columns)+1))
            db.execute(f'INSERT INTO {sync.TABLE} VALUES({q})', [n, *[row[k] for k in columns]])
        db.executescript('CREATE TABLE Inventory(Owner,Count); INSERT INTO Inventory VALUES(42,7);')


class ItemSynchronizationTests(unittest.TestCase):
    def test_offsets_numeric_ranges_and_truncation(self):
        raw = fixture_source()
        rows = sync.parse_items(raw)
        self.assertEqual((rows[1001]['Reqluc'], rows[1001]['ConstStr'], rows[1001]['Grade'], rows[1001]['Drop']), (44082,65,3210,102))
        self.assertEqual(rows[30244]['Exp'],150)
        for bad in (raw[:-1], raw+b'x', b'not plaintext'):
            with self.assertRaises(ValueError):
                sync.parse_items(bad)

    def test_sync_preserves_customs_players_backups_and_is_idempotent(self):
        raw = fixture_source()
        rows = sync.parse_items(raw)
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source = root/'client/Item/Item.SData'
            source.parent.mkdir(parents=True)
            source.write_bytes(raw)
            database = root/'seed.sqlite'
            fixture_db(database,rows)
            manifest = database.with_suffix('.manifest.json')
            manifest.write_text(json.dumps({'sha256':'old', 'tables':{sync.TABLE:len(rows)-1,'Inventory':1}}))
            original = database.read_bytes()
            plan = sync.synchronize(source,[database])
            self.assertEqual(plan['databases'][0]['added'],1)
            self.assertEqual(database.read_bytes(),original)
            self.assertEqual(source.read_bytes(),raw)
            report = sync.synchronize(source,[database],root/'backup',True)
            self.assertTrue(report['databases'][0]['other_tables_unchanged'])
            self.assertEqual((root/'backup/Item.SData').read_bytes(),raw)
            self.assertEqual({path.name for path in source.parent.parent.iterdir()}, {'Item'})
            with closing(sqlite3.connect(root/'backup/database-0.sqlite')) as saved:
                self.assertEqual(saved.execute(f'SELECT ConstStr FROM {sync.TABLE} WHERE ItemID=1001').fetchone(),(3,))
                self.assertEqual(saved.execute('SELECT * FROM Inventory').fetchall(),[(42,7)])
            actual = sync.parse_items(source.read_bytes())
            self.assertEqual(actual[100204]['Special'],58)
            self.assertEqual((actual[100049]['Sell'],actual[100049]['Grade']),(1000,944))
            self.assertEqual(actual[1001], rows[1001])
            self.assertEqual(actual[30244], rows[30244])
            self.assertEqual(actual[44213]['Range'],17)
            self.assertEqual(report['preserved_wide_skill_references'][44213],273)
            with closing(sqlite3.connect(database)) as db, db:
                self.assertEqual(db.execute('SELECT * FROM Inventory').fetchall(),[(42,7)])
                self.assertEqual(db.execute(f'SELECT Range FROM {sync.TABLE} WHERE ItemID=44213').fetchone(),(273,))
                self.assertEqual(db.execute(f'SELECT ConstStr FROM {sync.TABLE} WHERE ItemID=1001').fetchone(),(65,))
                self.assertEqual(db.execute(f'SELECT Exp FROM {sync.TABLE} WHERE ItemID=30244').fetchone(),(150,))
            second = sync.synchronize(source,[database])
            self.assertEqual(second['databases'][0]['updated'],0)
            self.assertEqual(second['databases'][0]['added'],0)
            meta = json.loads(manifest.read_text())
            self.assertEqual(meta['tables'][sync.TABLE],len(rows))
            self.assertEqual(meta['sha256'],sync.hashlib.sha256(database.read_bytes()).hexdigest())

    def test_preflight_failure_and_trigger_side_effects_do_not_mutate_players(self):
        raw = fixture_source()
        rows = sync.parse_items(raw)
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            source=root/'client/Item/Item.SData';source.parent.mkdir(parents=True);source.write_bytes(raw)
            database=root/'seed.sqlite';fixture_db(database,rows)
            original=database.read_bytes()
            with self.assertRaises(sqlite3.OperationalError):
                sync.synchronize(source,[database,root/'missing.sqlite'],root/'backup',True)
            self.assertEqual(database.read_bytes(),original)
            with closing(sqlite3.connect(database)) as db, db:
                db.execute(f'CREATE TRIGGER unexpected AFTER UPDATE ON {sync.TABLE} BEGIN UPDATE Inventory SET Count=99; END')
            with self.assertRaisesRegex(ValueError,'non-Items table'):
                sync.synchronize(source,[database],root/'trigger-backup',True)
            with closing(sqlite3.connect(database)) as db, db:
                self.assertEqual(db.execute('SELECT Count FROM Inventory').fetchone(),(7,))
                self.assertEqual(db.execute(f'SELECT ConstStr FROM {sync.TABLE} WHERE ItemID=1001').fetchone(),(3,))
            self.assertEqual(source.read_bytes(),raw)


if __name__ == '__main__':
    unittest.main()
