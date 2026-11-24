#!/usr/bin/env python3
"""Install the pinned server physics resources used by native release builds."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import shutil
import tarfile
import tempfile
import time
import urllib.error
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
URL = ('https://github.com/Spelunkern/Shaiya-GX/releases/download/'
       'gx-geometry-rules3-20261123/gx-server-geometry-rules3.tar.gz')
SHA256 = '4feb61ac11b1ed52979f5414fcdb58acee4ee94cc5858857afe92f86fa501f95'
ARCHIVE_MANIFESTS_SHA256 = '1bc09817cdaf5c3106712ebbd724845f31dd77019a29cfa1a3dfde004f523e84'
MANIFESTS_SHA256 = 'b1750b8a0bd4188291b4bacd54ef284b90aa3f821478ff9d6121f7c581d71ac8'
RETIRED_MAPS = frozenset(range(59, 64))
PORTAL_EDITS = (19, 20, 28, 29)
COLLISION_RULES_VERSION = 3  # internal/mapnav.CollisionRulesVersion


def digest(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def manifest_digest(catalog, manifests):
    canonical = json.dumps([catalog, manifests], sort_keys=True, separators=(',', ':'))
    return hashlib.sha256(canonical.encode('utf-8')).hexdigest()


def read_manifests(directory):
    catalog = json.loads((directory / 'catalog.json').read_text())
    maps = catalog.get('maps', [])
    if (catalog.get('version') != 1 or not maps
            or any(type(map_id) is not int or not 0 <= map_id <= 65535 for map_id in maps)
            or maps != sorted(set(maps))):
        raise ValueError('Invalid server physics catalog')
    manifests = [json.loads((directory / f'{map_id}.manifest.json').read_text())
                 for map_id in maps]
    return catalog, manifests


def validate(directory, *, require_pin=True):
    catalog, manifests = read_manifests(directory)
    maps = catalog['maps']
    for map_id, manifest in zip(maps, manifests):
        if (manifest.get('version'), manifest.get('map_id'), manifest.get('player_physics')) != (1, map_id, 1):
            raise ValueError(f'Incompatible physics generation: {map_id}')
        rules = manifest.get('collision_rules')
        if rules != COLLISION_RULES_VERSION:
            raise ValueError(f'Map {map_id} uses collision rules {rules}; '
                             f'GX Server requires {COLLISION_RULES_VERSION}. Rebuild the geometry catalog.')
    if require_pin and manifest_digest(catalog, manifests) != MANIFESTS_SHA256:
        raise ValueError(f'{directory} contains a different physics generation. '
                         'Install the pinned generation into a fresh directory.')
    for map_id, manifest in zip(maps, manifests):
        for extension in ('gxnav', 'gxcol', 'gxphysics'):
            name = f'{map_id}.{extension}'
            if digest(directory / name) != manifest['files'].get(name):
                raise ValueError(f'Physics checksum mismatch: {name}')
    return catalog, manifests


def project_archive(directory):
    # These portal-only SVMAP edits leave the walk masks used by the bake intact.
    catalog, manifests = validate(directory, require_pin=False)
    if manifest_digest(catalog, manifests) != ARCHIVE_MANIFESTS_SHA256:
        raise ValueError('Unexpected source catalog for the geometry projection')
    retained = {map_id: manifest for map_id, manifest in zip(catalog['maps'], manifests)
                if map_id not in RETIRED_MAPS}
    catalog['maps'] = list(retained)
    for map_id in PORTAL_EDITS:
        name = f'{map_id}.svmap'
        retained[map_id]['sources'][f'server/maps/{name}'] = digest(ROOT / 'server/data/maps' / name)
    if manifest_digest(catalog, list(retained.values())) != MANIFESTS_SHA256:
        raise ValueError('Projected catalog differs from the pinned generation')
    for map_id in PORTAL_EDITS:
        path = directory / f'{map_id}.manifest.json'
        path.write_text(json.dumps(retained[map_id], indent=2) + '\n')
    (directory / 'catalog.json').write_text(json.dumps(catalog, indent=2) + '\n')
    for map_id in RETIRED_MAPS:
        for ext in ('manifest.json', 'gxnav', 'gxcol', 'gxphysics'):
            (directory / f'{map_id}.{ext}').unlink()


def install(destination, archive=None):
    if destination.exists():
        catalog, manifests = read_manifests(destination)
        if manifest_digest(catalog, manifests) == ARCHIVE_MANIFESTS_SHA256:
            project_archive(destination)
            print('Updated the existing server physics catalog.', flush=True)
            return
        validate(destination)
        print('Existing server physics verified; no files replaced.', flush=True)
        return
    destination.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='.gx-geometry-', dir=destination.parent) as temporary:
        temporary = Path(temporary)
        if archive is None:
            archive = temporary / 'download.tar.gz'
            # A source push can start CI while its pinned resource upload finishes.
            for attempt in range(8):
                try:
                    with urllib.request.urlopen(URL, timeout=120) as response, archive.open('wb') as target:
                        shutil.copyfileobj(response, target)
                    break
                except (urllib.error.URLError, TimeoutError):
                    if attempt == 7:
                        raise
                    time.sleep(15)
        if digest(archive) != SHA256:
            raise ValueError('Pinned physics archive checksum mismatch')
        with tarfile.open(archive, 'r:gz') as source:
            members = source.getmembers()
            allowed = re.compile(r'geometry/(?:catalog\.json|\d+\.(?:manifest\.json|gxnav|gxcol|gxphysics))\Z')
            if sum(member.size for member in members) > 1536 * 1024 * 1024:
                raise ValueError('Oversized physics archive')
            for member in members:
                if member.isdir() and member.name == 'geometry':
                    continue
                if not member.isfile() or not allowed.fullmatch(member.name):
                    raise ValueError(f'Unexpected physics archive member: {member.name}')
                target = temporary / member.name
                target.parent.mkdir(parents=True, exist_ok=True)
                with source.extractfile(member) as stream, target.open('wb') as output:
                    shutil.copyfileobj(stream, output)
        project_archive(temporary / 'geometry')
        (temporary / 'geometry').rename(destination)
    print('Installed the pinned server physics catalog.', flush=True)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--destination', type=Path, default=ROOT / 'server/data/geometry')
    parser.add_argument('--archive', type=Path, help='Use an already downloaded pinned archive')
    args = parser.parse_args()
    install(args.destination.resolve(), args.archive)
