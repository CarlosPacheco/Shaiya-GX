import struct
import unittest
from sync_mall_content import prepare, read_items
from update_recreation_content import read_cash, string

class MallSyncTests(unittest.TestCase):
    def fixture(self, quantity=255, code='Pack'):
        body=bytearray(81);body[0],body[1],body[80]=100,48,255
        gear=bytearray(body);gear[0],gear[1],gear[80]=17,46,1
        items=struct.pack('<II',1,2)+string('Reset Stone')+string('Resets stats')+body+string('Armor')+string('Equipment')+gear
        cash=struct.pack('<I4I',1,1,1,0,0)+struct.pack('<IBIB',100048,quantity,17046,4)+bytes(22*5)+string('Kit x10')+string(code)+string('Old grants 1')
        return items,cash

    def test_stackable_custom_sql_count_one_and_equipment_bundle(self):
        items,cash=self.fixture(1)
        patched,native,products,limits,changes=prepare(items,cash,{100048:1,17046:1})
        self.assertEqual(limits,{100048:255,17046:1})
        self.assertEqual(products[0]['items'][:2],[[100048,255],[17046,4]])
        self.assertEqual(changes,1)
        self.assertIn(b'Reset Stone x255',products[0]['fields'][2])
        self.assertNotIn(b'grants 1',native)
        self.assertEqual(read_items(patched)[1][100048][2][80],255)
        self.assertEqual(prepare(patched,native,limits)[:2],(patched,native))

    def test_unknown_item_and_long_wire_code_abort(self):
        items,cash=self.fixture()
        with self.assertRaises(ValueError):prepare(items,cash,{17046:1})
        items,cash=self.fixture(code='a'*21)
        with self.assertRaises(ValueError):prepare(items,cash,{100048:1,17046:1})

if __name__=='__main__':unittest.main()
