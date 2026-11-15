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
       'gx-geometry-rules2-20261114/gx-server-geometry-rules2.tar.gz')
SHA256 = '2acf507e99ba2e8ae41c037c6f8b58443179c5becf2d30c0d5bd0820cf3718d5'


def digest(path):
    with path.open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def validate(directory):
    catalog = json.loads((directory / 'catalog.json').read_text())
    if catalog.get('version') != 1 or catalog.get('maps') != list(range(68)):
        raise ValueError('Expected the complete 68-map physics generation')
    for map_id in catalog['maps']:
        manifest = json.loads((directory / f'{map_id}.manifest.json').read_text())
        if (manifest.get('version'), manifest.get('map_id'),
                manifest.get('player_physics'), manifest.get('collision_rules')) != (1, map_id, 1, 2):
            raise ValueError(f'Incompatible physics generation: {map_id}')
        for extension in ('gxnav', 'gxcol', 'gxphysics'):
            name = f'{map_id}.{extension}'
            if digest(directory / name) != manifest['files'].get(name):
                raise ValueError(f'Physics checksum mismatch: {name}')


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
    print('Installed verified server physics for maps 0–67.', flush=True)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--destination', type=Path, default=ROOT / 'server/data/geometry')
    parser.add_argument('--archive', type=Path, help='Use an already downloaded pinned archive')
    args = parser.parse_args()
    install(args.destination.resolve(), args.archive)
