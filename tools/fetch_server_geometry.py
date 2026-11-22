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
       'gx-geometry-rules2-20261122/gx-server-geometry-rules2.tar.gz')
SHA256 = '020c170f908517961799a33e749dbc8dec8a2d17c6049b5ad2857fec90208073'
MANIFESTS_SHA256 = '980578e3aa7ffbf3e8d393d694669e2fc4131b7645311d7caaa704fcee213316'


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
    if require_pin and manifest_digest(catalog, manifests) != MANIFESTS_SHA256:
        raise ValueError(f'{directory} contains a different physics generation. '
                         'Install the pinned generation into a fresh directory.')
    maps = catalog['maps']
    for map_id, manifest in zip(maps, manifests):
        if (manifest.get('version'), manifest.get('map_id'),
                manifest.get('player_physics'), manifest.get('collision_rules')) != (1, map_id, 1, 2):
            raise ValueError(f'Incompatible physics generation: {map_id}')
        for extension in ('gxnav', 'gxcol', 'gxphysics'):
            name = f'{map_id}.{extension}'
            if digest(directory / name) != manifest['files'].get(name):
                raise ValueError(f'Physics checksum mismatch: {name}')
    return catalog, manifests


def install(destination, archive=None):
    if destination.exists():
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
        validate(temporary / 'geometry')
        (temporary / 'geometry').rename(destination)
    print('Installed the pinned server physics catalog.', flush=True)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--destination', type=Path, default=ROOT / 'server/data/geometry')
    parser.add_argument('--archive', type=Path, help='Use an already downloaded pinned archive')
    args = parser.parse_args()
    install(args.destination.resolve(), args.archive)
