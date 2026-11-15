import json
from pathlib import Path
import struct
import tempfile
import unittest
from PIL import Image
from import_dungeon_minimaps import import_minimaps


class ImportMinimapsTest(unittest.TestCase):
    def test_preserves_existing_pixels_and_missing_maps(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source, data, maps = root/'source', root/'data', root/'maps'
            images = source/'Interface'/'MiniMap'
            images.mkdir(parents=True)
            (data/'world').mkdir(parents=True)
            (data/'interface'/'MiniMap').mkdir(parents=True)
            maps.mkdir()
            raw = b'# legacy \xb0\xa1\n[WorldMap]\ncount=1\n[0]\nMapNum=3\nImgPos_x=1036\nImgPos_y=1052\n'
            (source/'Dungeon_MiniMap.cfg').write_bytes(raw)
            for identity in (1, 3, 50, 60):
                (data/'world'/f'{identity}.wld').touch()
                (maps/f'{identity}.svmap').write_bytes(struct.pack('<I',1024))
            existing = data/'interface'/'MiniMap'/'1.png'
            existing.write_bytes(b'preserve this exact existing resource')
            Image.new('RGB',(8,8),(1,2,3)).save(images/'1.png')
            Image.new('RGBA',(8,8),(19,72,41,123)).save(images/'3.tga')
            (source/'World').mkdir()
            (source/'World'/'60.wld').write_bytes(b'DUN\0stadium.dg\0')
            (data/'world'/'60.wld').write_bytes(b'FLD\0')
            Image.new('RGB',(8,8),(1,2,3)).save(images/'60.png')
            report = import_minimaps(source,data,maps)
            self.assertEqual(existing.read_bytes(),b'preserve this exact existing resource')
            self.assertEqual(report['preserved'],[1])
            self.assertEqual(report['missing'],[50])
            self.assertEqual(report['incompatible'],[60])
            self.assertFalse((data/'interface'/'MiniMap'/'60.png').exists())
            self.assertEqual([r['map'] for r in report['imported']],[3])
            with Image.open(data/'interface'/'MiniMap'/'3.png') as converted:
                self.assertEqual(converted.convert('RGBA').getpixel((0,0)),(19,72,41,123))
            self.assertEqual((data/'Dungeon_MiniMap.cfg').read_bytes(),raw)
            self.assertEqual(json.loads((data/'interface'/'MiniMap'/'map-sizes.json').read_text()),{'1':1024,'3':1024,'50':1024,'60':1024})
            again = import_minimaps(source,data,maps)
            self.assertEqual(again['imported'],[])
            self.assertEqual(again['preserved'],[1,3])


if __name__ == '__main__':
    unittest.main()
