#!/usr/bin/env python3
"""Rebuild server physics from local client data before tagging a GX release."""
import argparse
from functools import cache
from pathlib import Path
import re
import subprocess
import sys
import tarfile
import tempfile

if __package__:
    from .fetch_server_geometry import MANIFESTS_SHA256, ROOT, digest, manifest_digest, validate
else:
    from fetch_server_geometry import MANIFESTS_SHA256, ROOT, digest, manifest_digest, validate

TAG_PREFIX = 'GX-Geometry-Manifests-SHA256: '
TAG_PATTERN = re.compile(r'gx-v[0-9][a-zA-Z0-9._-]*')
WORLD_MAP = re.compile(r'world/([0-9]+)\.wld')


def verify_sources(catalog, manifests, server_maps, client_data=None):
    client_files = {}
    if client_data is not None:
        for path in client_data.rglob('*'):
            if not path.is_file():
                continue
            relative = path.relative_to(client_data).as_posix().casefold()
            if relative in client_files:
                raise ValueError(f'Ambiguous client data path: {relative}')
            client_files[relative] = path
        maps = sorted(int(match[1]) for name in client_files
                      if (match := WORLD_MAP.fullmatch(name)) and int(match[1]) <= 65535)
        if maps != catalog['maps']:
            raise ValueError('Numbered client world maps do not match the server physics catalog')

    source_digest = cache(digest)
    for map_id, manifest in zip(catalog['maps'], manifests):
        if manifest.get('map_id') != map_id or not isinstance(manifest.get('sources'), dict):
            raise ValueError(f'Map {map_id} is missing its geometry source manifest')
        sources = manifest['sources']
        mask = server_maps / f'{map_id}.svmap'
        mask_key = f'server/maps/{mask.name}'
        if mask.exists() or mask_key in sources:
            if not mask.is_file() or source_digest(mask) != sources.get(mask_key):
                raise ValueError(f'Map {map_id} server geometry source mismatch: {mask_key}')
        if client_data is not None:
            if f'world/{map_id}.wld' not in sources:
                raise ValueError(f'Map {map_id} world file is missing from its geometry manifest')
            for name, expected in sorted(sources.items()):
                if name == mask_key:
                    continue
                path = client_files.get(name.casefold())
                if path is None or source_digest(path) != expected:
                    raise ValueError(f'Map {map_id} client geometry source mismatch: {name}')


def git(*args):
    return subprocess.check_output(['git', *args], cwd=ROOT, text=True).strip()


def tag_ref(tag):
    if not TAG_PATTERN.fullmatch(tag):
        raise ValueError(f'Invalid GX release tag: {tag}')
    ref = f'refs/tags/{tag}'
    git('check-ref-format', ref)
    return ref


def verify_tag(tag):
    ref = tag_ref(tag)
    tag_type = git('for-each-ref', '--format=%(objecttype)', ref)
    if (tag_type != 'tag' or git('rev-parse', f'{ref}^{{commit}}') != git('rev-parse', 'HEAD')):
        raise ValueError(f'{tag} must be an annotated release tag for the checked out commit')
    annotation = git('for-each-ref', '--format=%(contents)', ref)
    fingerprints = [line for line in annotation.splitlines() if line.startswith(TAG_PREFIX)]
    if fingerprints != [TAG_PREFIX + MANIFESTS_SHA256]:
        raise ValueError(f'{tag} needs a geometry rebuild against the pinned generation. '
                         'Prepare it with tools/release_geometry.py --data CLIENT_DATA --tag TAG.')
    print(f'{tag} matches the pinned server physics manifests.', flush=True)


def export_candidate(directory, candidate, fingerprint):
    if candidate.exists():
        raise ValueError(f'Choose a new candidate directory: {candidate}')
    candidate.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='.gx-physics-candidate-', dir=candidate.parent) as staging:
        staged = Path(staging) / 'resource'
        staged.mkdir()
        archive = staged / 'gx-server-geometry-rules2.tar.gz'
        with tarfile.open(archive, 'w:gz', compresslevel=4) as output:
            output.add(directory, arcname='geometry')
        (staged / 'SHA256SUMS.txt').write_text(f'{digest(archive)}  {archive.name}\n')
        (staged / 'MANIFESTS-SHA256.txt').write_text(fingerprint + '\n')
        staged.rename(candidate)
    print(f'Generated physics archive for review in {candidate}.', flush=True)


def rebuild(data, tag=None, candidate=None, recast_source=None):
    if not data.is_dir():
        raise ValueError(f'Client data directory is missing: {data}')
    if candidate is not None and candidate.is_relative_to(data):
        raise ValueError('Choose a candidate location outside the client data')
    if tag is not None:
        if git('for-each-ref', '--format=%(refname)', tag_ref(tag)):
            raise ValueError(f'Tag already exists: {tag}')
        head = git('rev-parse', 'HEAD')
        if git('status', '--porcelain'):
            raise ValueError('Use a clean checkout of the release commit to prepare a tag')
    work = ROOT / 'work'
    work.mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='release-geometry-', dir=work) as temporary:
        directory = Path(temporary) / 'geometry'
        server_maps = ROOT / 'server/data/maps'
        command = [sys.executable, str(ROOT / 'server/tools/mapbuild/generate.py'),
                   '--data', str(data), '--server-maps', str(server_maps),
                   '--maps', 'all', '--output', str(directory)]
        if recast_source is not None:
            command.extend(['--recast-source', str(recast_source)])
        subprocess.run(command, cwd=ROOT, check=True)
        catalog, manifests = validate(directory, require_pin=False)
        verify_sources(catalog, manifests, server_maps, data)
        fingerprint = manifest_digest(catalog, manifests)
        if fingerprint != MANIFESTS_SHA256:
            guidance = 'Use --candidate to export the replacement archive.'
            if candidate is not None:
                export_candidate(directory, candidate, fingerprint)
                guidance = 'Publish the reviewed archive and update the release pin before tagging.'
            raise ValueError(f'Rebuilt geometry differs from the pinned generation ({fingerprint}). {guidance}')
    print(f'Rebuilt and verified {len(manifests)} maps against the pinned release generation.', flush=True)
    if tag is not None:
        if git('rev-parse', 'HEAD') != head or git('status', '--porcelain'):
            raise ValueError('The release checkout changed during the rebuild')
        git('tag', '-a', tag, head, '-m', f'Shaiya GX {tag}\n\n{TAG_PREFIX}{fingerprint}')
        print(f'Created verified local release tag {tag}.', flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument('--data', type=Path, help='Loose client data for this release')
    mode.add_argument('--verify-tag', metavar='TAG', help='Verify a release tag in CI')
    parser.add_argument('--tag', help='Create this local release tag if the rebuild matches')
    parser.add_argument('--candidate', type=Path, help='Export a changed generation to a new directory')
    parser.add_argument('--recast-source', type=Path, help='Reuse a checkout of the pinned Recast source')
    args = parser.parse_args()
    if args.verify_tag is not None and any((args.tag, args.candidate, args.recast_source)):
        parser.error('--tag, --candidate and --recast-source require --data')
    try:
        if args.verify_tag is not None:
            verify_tag(args.verify_tag)
        else:
            rebuild(args.data.resolve(), args.tag, args.candidate and args.candidate.resolve(),
                    args.recast_source and args.recast_source.resolve())
    except (ValueError, OSError, subprocess.CalledProcessError) as error:
        parser.exit(1, f'{error}\n')


if __name__ == '__main__':
    main()
