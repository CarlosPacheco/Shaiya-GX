import tempfile
from pathlib import Path
import unittest

from package_bundle import merge, verify_cash_copies


class NativeBundleMergeTests(unittest.TestCase):
    def test_windows_ansi_readme_matches_linux_utf8_without_changing_linux_bytes(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source, destination = root/'windows', root/'shared'
            source.mkdir(); destination.mkdir()
            text = 'GX Server — portable\nGM commands: /nmake 1 1\n'
            (source/'README.txt').write_bytes(text.replace('\n', '\r\n').encode('cp1252'))
            (source/'gx_world.exe').write_bytes(b'PE\0windows')
            (destination/'README.txt').write_bytes(text.encode('utf8'))
            merge(source, destination)
            self.assertEqual((destination/'README.txt').read_bytes(), text.encode('utf8'))
            self.assertEqual((destination/'gx_world.exe').read_bytes(), b'PE\0windows')

    def test_different_binary_content_is_rejected(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source, destination = root/'a', root/'b'
            source.mkdir(); destination.mkdir()
            (source/'seed.sqlite').write_bytes(b'SQLite\0one')
            (destination/'seed.sqlite').write_bytes(b'SQLite\0two')
            with self.assertRaises(ValueError):
                merge(source, destination)


class CashCatalogBundleTests(unittest.TestCase):
    def test_independent_native_catalogs_are_required(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            client, server = root / 'client', root / 'server'
            client.mkdir(); server.mkdir()
            with self.assertRaises(ValueError): verify_cash_copies(client, server)
            (client / 'Item').mkdir()
            (client / 'Item/Cash.SData').write_bytes(b'native table')
            with self.assertRaises(ValueError): verify_cash_copies(client, server)
            (server / 'rapture.sqlite').write_bytes(b'database')
            verify_cash_copies(client, server)

class BinaryOnlyBundleTests(unittest.TestCase):
    def test_dual_os_bundle_keeps_client_resources_separate(self):
        import hashlib
        import json
        import tarfile
        import zipfile
        from unittest.mock import patch
        from package_bundle import ARCHIVES, digest, prepare
        with tempfile.TemporaryDirectory() as temporary:
            root=Path(temporary);archives=root/'archives';archives.mkdir()
            for name in ARCHIVES:
                stem=name.removesuffix('.tar.gz').removesuffix('.zip')
                source=root/'source'/stem;source.mkdir(parents=True)
                windows='windows' in name
                if name.startswith('gx-client-'):
                    (source/('Shaiya GX.exe' if windows else 'Shaiya GX')).write_bytes(b'native executable')
                    fonts=source/'data/fonts';fonts.mkdir(parents=True)
                    (fonts/'LICENSE.txt').write_text('packaged licensed fonts')
                    (fonts/'README.md').write_text('font notes')
                    (fonts/'pack.json').write_text(json.dumps({'files': {'README.md': {}, 'LICENSE.txt': {}}}))
                    bindings=source/'data/Vehicle';bindings.mkdir()
                    (bindings/'bones.ini').write_text('[Human]\n0=25,-1\n')
                else:
                    (source/'build').mkdir();(source/'data').mkdir()
                    (source/'build'/('gx_world.exe' if windows else 'gx_world')).write_bytes(b'native server')
                    (source/'data/rapture.sqlite').write_bytes(b'approved development seed')
                    record={'files':{p.relative_to(source).as_posix():digest(p) for p in source.rglob('*') if p.is_file()}}
                    (source/'package.manifest.json').write_text(json.dumps(record))
                if name.endswith('.zip'):
                    with zipfile.ZipFile(archives/name,'w') as archive:
                        for path in source.rglob('*'):
                            if path.is_file():archive.write(path,path.relative_to(source.parent))
                else:
                    with tarfile.open(archives/name,'w:gz') as archive:archive.add(source,arcname=stem)
            (archives/'SHA256SUMS.txt').write_text(''.join(digest(archives/n)+'  '+n+'\n' for n in ARCHIVES))
            output=root/'portable'
            with patch('package_bundle.validate') as validate:
                prepare(archives,None,output)
                validate.assert_called_once()
                self.assertEqual(validate.call_args.args[0].parts[-2:],('data','fonts'))
            self.assertTrue((output/'Client/Shaiya GX').is_file())
            self.assertTrue((output/'Client/Shaiya GX.exe').is_file())
            self.assertEqual({p.name for p in (output/'Client/data').iterdir()},{'fonts','Vehicle','filter.txt'})
            self.assertEqual((output/'Client/data/filter.txt').read_bytes(), b'')
            self.assertFalse((output/'Client/data/Cash.SData').exists())
            self.assertTrue((output/'Server/data/rapture.sqlite').is_file())
            self.assertEqual(list(output.rglob('README*')), [])
            fonts = output/'Client/data/fonts'
            self.assertEqual(set(json.loads((fonts/'pack.json').read_text())['files']), {'LICENSE.txt'})

            data=root/'local-data';(data/'fonts').mkdir(parents=True)
            (data/'fonts/LICENSE.txt').write_text('local licensed font data')
            (data/'Cash.SData').write_bytes(b'local original catalog')
            (data/'fonts/Arial.TTF').write_bytes(b'local regular face')
            (data/'fonts/arialbd.ttf').write_bytes(b'local bold face')
            with patch('package_bundle.validate'):
                prepare(archives,data,root/'with-data')
            for relative in ('fonts/LICENSE.txt', 'Cash.SData'):
                self.assertEqual((data/relative).read_bytes(), (root/'with-data/Client/data'/relative).read_bytes())
            fonts = root/'with-data/Client/data/fonts'
            self.assertFalse((fonts/'Arial.TTF').exists())
            self.assertFalse((fonts/'arialbd.ttf').exists())
