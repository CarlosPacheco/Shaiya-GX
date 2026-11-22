import tempfile
from pathlib import Path
import unittest
from unittest.mock import patch

from fetch_server_geometry import MANIFESTS_SHA256, digest
import release_geometry


class ReleaseGeometryTests(unittest.TestCase):
    def test_server_walk_mask_must_match_even_without_client_data(self):
        with tempfile.TemporaryDirectory() as temporary:
            maps = Path(temporary)
            mask = maps / '7.svmap'
            mask.write_bytes(b'walk mask')
            catalog = {'maps': [7]}
            manifests = [{'map_id': 7, 'sources': {'server/maps/7.svmap': digest(mask)}}]
            release_geometry.verify_sources(catalog, manifests, maps)
            mask.write_bytes(b'changed walk mask')
            with self.assertRaisesRegex(ValueError, 'server/maps/7.svmap'):
                release_geometry.verify_sources(catalog, manifests, maps)

    def test_release_tag_requires_an_annotation_with_the_current_pin(self):
        with tempfile.TemporaryDirectory() as temporary, patch.object(release_geometry, 'ROOT', Path(temporary)):
            git = release_geometry.git
            git('init', '-q')
            git('config', 'user.name', 'GX test')
            git('config', 'user.email', 'gx-test@example.invalid')
            git('config', 'commit.gpgsign', 'false')
            git('config', 'tag.gpgsign', 'false')
            git('commit', '--allow-empty', '-qm', 'Release commit')
            git('tag', 'gx-v1')
            with self.assertRaisesRegex(ValueError, 'annotated release tag'):
                release_geometry.verify_tag('gx-v1')
            git('tag', '-a', 'gx-v2', '-m', release_geometry.TAG_PREFIX + '0' * 64)
            with self.assertRaisesRegex(ValueError, 'pinned generation'):
                release_geometry.verify_tag('gx-v2')
            git('tag', '-a', 'gx-v3', '-m', release_geometry.TAG_PREFIX + MANIFESTS_SHA256)
            release_geometry.verify_tag('gx-v3')
            checkout = Path(temporary) / 'shallow'
            checkout.mkdir()
            commit = git('rev-parse', 'HEAD')
            with patch.object(release_geometry, 'ROOT', checkout):
                git('init', '-q')
                remote = Path(temporary).as_uri()
                ref = 'refs/tags/gx-v3'
                git('fetch', '--no-tags', '--depth=1', remote, f'{commit}:{ref}')
                git('checkout', '-q', '--detach', commit)
                with self.assertRaisesRegex(ValueError, 'annotated release tag'):
                    release_geometry.verify_tag('gx-v3')
                git('fetch', '--no-tags', '--depth=1', '--force', remote, f'{ref}:{ref}')
                release_geometry.verify_tag('gx-v3')
            git('commit', '--allow-empty', '-qm', 'Another commit')
            with self.assertRaisesRegex(ValueError, 'checked out commit'):
                release_geometry.verify_tag('gx-v3')


if __name__ == '__main__':
    unittest.main()
