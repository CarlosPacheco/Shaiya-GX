import struct
import unittest
from update_recreation_content import Native, patch_items, patch_cash, read_cash, runes, string, clean_rune_body, rune_aliases

class RecreationContentTests(unittest.TestCase):
    def test_native_items_preserve_fields_and_add_twenty_eight_runes(self):
        body=bytearray(range(81));body[0],body[1],body[35],body[80]=100,171,62,255
        gem=body[:];gem[0],gem[1],gem[80]=30,1,1
        raw=struct.pack('<II',1,2)+string('Rune')+string('Description')+body+string('Gem')+string('Gem description')+gem
        first=patch_items(raw)
        self.assertEqual(patch_items(first),first)
        r=Native(first);self.assertEqual(r.word(),101)
        records={}
        for _ in range(101):
            for _ in range(r.word()):
                name,desc,b=r.string(),r.string(),r.take(81);records[b[0],b[1]]=(name,desc,b)
        r.end()
        self.assertEqual(records[100,171][2],clean_rune_body(body,struct.unpack_from("<H",body,28)[0]))
        expected=clean_rune_body(gem,struct.unpack_from("<H",gem,28)[0]);expected[80]=255
        self.assertEqual(records[30,1][2],expected)
        self.assertEqual(len(runes()),28)
        for rune in runes():
            b=records[101,rune['type_id']][2]
            self.assertEqual(struct.unpack_from('<H',b,28)[0],rune['req_vg'])
            self.assertEqual(b[35],62);self.assertEqual(b[80],255)
            self.assertEqual(records[101,rune['type_id']][0].decode(),rune['name'])

    def test_native_mall_updates_bundles_and_is_idempotent(self):
        raw=struct.pack('<I4I',1,1,1,0,0)
        raw+=struct.pack('<IBIB',30001,1,95001,1)+bytes(22*5)
        raw+=string('Supplies')+string('Supplies_1')+string('Description')
        result,products=patch_cash(raw)
        self.assertEqual(len(products),29)
        self.assertEqual(products[0]['items'][:2],[[30001,255],[95001,255]])
        self.assertEqual(patch_cash(result)[0],result)
        for p in read_cash(result)[1:]:
            self.assertEqual(p['items'][0][1],255)
            self.assertLessEqual(len(p['fields'][1]),20)


class LegacyRuneTests(unittest.TestCase):
    def test_all_legacy_variants_receive_explicit_selector_and_no_residual_stats(self):
        body=bytearray(81);body[0],body[1],body[35],body[80]=100,171,62,255
        body[18],body[20],body[53],body[55],body[57]=3,4,5,6,7
        records=string('Rune')+string('Description')+body
        for old in rune_aliases():
            legacy=body[:];legacy[1]=old
            records+=string('Old rune')+string('Old description')+legacy
        raw=struct.pack('<II',1,1+len(rune_aliases()))+records
        patched=patch_items(raw);reader=Native(patched);actual={}
        for _ in range(reader.word()):
            for _ in range(reader.word()):
                reader.string();reader.string();b=reader.take(81);actual[b[0],b[1]]=b
        for old,definition in rune_aliases().items():
            b=actual[100,old]
            self.assertEqual(struct.unpack_from('<H',b,28)[0],definition['req_vg'])
            self.assertEqual(b[16:28],bytes(12));self.assertEqual(b[47:65],bytes(18))
        self.assertEqual(patch_items(patched),patched)

class ForgeCostContentTests(unittest.TestCase):
    def test_authoritative_reqig_changes_only_forge_gold_selector(self):
        body=bytearray(81);body[0],body[1],body[35],body[80]=100,171,62,255
        gem=bytearray(81);gem[0],gem[1],gem[31],gem[80]=30,7,39,255
        raw=struct.pack('<II',1,2)+string('Rune')+string('Normal')+body+string('Lapis')+string('Original')+gem
        result=patch_items(raw,{30007:38});reader=Native(result);found=None
        for _ in range(reader.word()):
            for _ in range(reader.word()):
                reader.string();reader.string();record=reader.take(81)
                if record[0:2]==bytes([30,7]):found=record
        expected=gem[:];expected[31]=38
        self.assertEqual(found,expected)

if __name__=='__main__':unittest.main()
