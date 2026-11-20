import csv
import sqlite3
from contextlib import closing
import tempfile
import unittest
from pathlib import Path

from migrate_item_stacks import migrate


class StackMigrationTests(unittest.TestCase):
    def test_only_stackable_definitions_and_mall_quantities_change(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            dbpath=root/'seed.sqlite'
            with closing(sqlite3.connect(dbpath)) as db, db:
                db.executescript('CREATE TABLE PS_GameDefs_Items(Type,TypeID,Count);'
                                 'INSERT INTO PS_GameDefs_Items VALUES(25,1,10),(1,1,1);'
                                 'CREATE TABLE Inventory(Count); INSERT INTO Inventory VALUES(7);')
            catalog=root/'cash.csv'
            with catalog.open('w',newline='') as f:
                csv.writer(f).writerows([['index']*55,
                    ['1','1','0','0','SKU','Test','']+['25001','10','1001','1']+['0','0']*22])
            client=root/'client'
            client.mkdir()
            migrate([dbpath],catalog,client)
            with closing(sqlite3.connect(dbpath)) as db, db:
                self.assertEqual(db.execute('SELECT Count FROM PS_GameDefs_Items ORDER BY Type').fetchall(),[(1,),(255,)])
                self.assertEqual(db.execute('SELECT Count FROM Inventory').fetchone(),(7,))
            with catalog.open() as f:
                row=list(csv.reader(f))[1]
            self.assertEqual((row[8],row[10]),('255','1'))
            self.assertEqual(catalog.read_bytes(),(client/'cash.csv').read_bytes())
            self.assertFalse((root/'item-stack-limits.csv').exists())
            self.assertEqual((client/'item-stack-limits.csv').read_text(),
                             'item_id,max_stack\n1001,1\n25001,255\n')
