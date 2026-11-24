"""Storage checks for publishing the distributable development database."""
from contextlib import closing
import hashlib
import json
from pathlib import Path
import sqlite3
import tempfile
import unittest

from snapshot_sqlite import snapshot
from package import GAME_DATA, check_navigation, copy_runtime_data


class SnapshotTests(unittest.TestCase):
    def test_snapshot_includes_committed_wal_and_preserves_source(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root / "runtime.sqlite"
            destination = root / "data/rapture.sqlite"
            with closing(sqlite3.connect(source)) as live:
                live.execute("PRAGMA journal_mode=WAL")
                live.execute("PRAGMA wal_autocheckpoint=0")
                live.execute("PRAGMA user_version=1")
                live.execute("CREATE TABLE sample(id INTEGER PRIMARY KEY,value TEXT,payload BLOB)")
                live.execute("INSERT INTO sample VALUES(?,?,?)", (2**60 + 1, "área", b"\x00\xff"))
                live.commit()
                self.assertTrue(Path(str(source) + "-wal").stat().st_size > 0)
                original = source.read_bytes()
                self.assertEqual(snapshot(source, destination), {"sample": 1})
                self.assertEqual(source.read_bytes(), original)
                self.assertEqual(live.execute("SELECT COUNT(*) FROM sample").fetchone()[0], 1)
            with closing(sqlite3.connect(destination)) as copy:
                self.assertEqual(copy.execute("SELECT * FROM sample").fetchone(), (2**60 + 1, "área", b"\x00\xff"))
                self.assertEqual(copy.execute("PRAGMA journal_mode").fetchone()[0], "delete")
            metadata = json.loads(destination.with_suffix(".manifest.json").read_text())
            self.assertEqual(metadata["sha256"], hashlib.sha256(destination.read_bytes()).hexdigest())
            self.assertFalse(Path(str(destination) + "-wal").exists())

    def test_failed_snapshot_preserves_previous_distribution(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source, destination = root / "runtime.sqlite", root / "seed.sqlite"
            destination.write_bytes(b"previous distribution")
            manifest = destination.with_suffix(".manifest.json")
            manifest.write_bytes(b"previous manifest")
            with closing(sqlite3.connect(source)) as database:
                database.execute("PRAGMA user_version=99")
            with self.assertRaises(ValueError):
                snapshot(source, destination)
            self.assertEqual(destination.read_bytes(), b"previous distribution")
            self.assertEqual(manifest.read_bytes(), b"previous manifest")
            with self.assertRaises(ValueError):
                snapshot(source, source)


class NavigationDistributionTests(unittest.TestCase):
    def test_complete_generation_and_corruption(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            directory = root / "data/geometry"
            directory.mkdir(parents=True)
            (directory / "catalog.json").write_text(json.dumps({"version": 1, "maps": [3]}))
            files = {}
            for extension in ("gxnav", "gxcol", "gxphysics"):
                name = f"3.{extension}"
                payload = name.encode()
                (directory / name).write_bytes(payload)
                files[name] = hashlib.sha256(payload).hexdigest()
            path = directory / "3.manifest.json"
            manifest = {"version": 1, "map_id": 3, "player_physics": 1, "collision_rules": 2, "files": files}
            path.write_text(json.dumps(manifest))
            with self.assertRaisesRegex(ValueError, "collision rules 2; GX Server requires 3"):
                check_navigation(root)
            manifest["collision_rules"] = 3
            path.write_text(json.dumps(manifest))
            check_navigation(root)
            (directory / "3.gxcol").write_bytes(b"corrupt")
            with self.assertRaisesRegex(ValueError, "checksum"):
                check_navigation(root)
            (directory / "3.gxcol").unlink()
            with self.assertRaises(FileNotFoundError):
                check_navigation(root)


class PackageDataTests(unittest.TestCase):
    def test_stages_only_runtime_data(self):
        with tempfile.TemporaryDirectory() as temporary:
            source = Path(temporary) / "data"
            target = Path(temporary) / "staged"
            expected = {"rapture.sqlite", "rapture.manifest.json", "maps/1.svmap",
                        "geometry/catalog.json", "geometry/1.gxnav", "geometry/1.gxcol",
                        "geometry/1.gxphysics", "geometry/1.manifest.json"}
            expected.update("game/" + name for name in GAME_DATA)
            residues = {"game/Map/1.svmap", "game/MapWar.ini", "game/README.txt", "maps/README.txt", "item-stack-limits.csv",
                        "rapture.sqlite-wal", "maps/old.svmap", "geometry/9.gxnav"}
            for name in expected | residues:
                path = source / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text('{"maps":[1]}' if name == "geometry/catalog.json" else name)
            copy_runtime_data(source, target)
            self.assertEqual({p.relative_to(target).as_posix() for p in target.rglob("*") if p.is_file()}, expected)


if __name__ == "__main__":
    unittest.main()
