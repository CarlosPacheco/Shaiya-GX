#!/usr/bin/env python3
"""Add a verified dual-OS client/server portable to native release archives."""
import argparse
from pathlib import Path
import shutil
import tempfile

from package_bundle import digest, prepare
from release_geometry import verify_tag


def package(archives, tag):
    verify_tag(tag)
    archives = archives.resolve()
    with tempfile.TemporaryDirectory(prefix='gx-complete-release-') as temporary:
        portable = Path(temporary) / ('Shaiya GX ' + tag + ' Portable')
        prepare(archives, None, portable, require_pin=True)
        shutil.make_archive(str(archives / 'gx-portable-linux-windows'), 'zip',
                            root_dir=portable.parent, base_dir=portable.name)
    files = sorted(path for path in archives.iterdir()
                   if path.name.endswith(('.tar.gz', '.zip')))
    (archives / 'SHA256SUMS.txt').write_text(''.join(
        digest(path) + '  ' + path.name + '\n' for path in files))
    print('Complete portable and release checksums ready.', flush=True)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--archives', type=Path, required=True)
    parser.add_argument('--tag', required=True)
    args = parser.parse_args()
    package(args.archives, args.tag)
