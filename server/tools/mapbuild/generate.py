#!/usr/bin/env python3
"""Generate complete GX server resources for selected current or future maps."""
import argparse
import os
from pathlib import Path
import shutil
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[3]
p = argparse.ArgumentParser(description=__doc__)
p.add_argument('--data', type=Path, required=True, help='loose client data directory (read only)')
p.add_argument('--server-maps', type=Path, default=ROOT / 'server/data/maps')
p.add_argument('--maps', default='all', help='all or comma-separated numeric map IDs')
p.add_argument('--output', type=Path, default=ROOT / 'work/mapbuild/generated')
mode = p.add_mutually_exclusive_group()
mode.add_argument('--physics-only', action='store_true', help='extend an unchanged existing generation without rebaking navigation')
mode.add_argument('--mob-dungeons-only', action='store_true', help='build structural dungeon mob navigation')
p.add_argument('--recast-source', type=Path, default=ROOT / 'work/mapbuild/recast')
a = p.parse_args()
go = Path.home() / '.local/opt/go-1.27.4/bin/go'
if not go.is_file():
    located = shutil.which('go')
    if not located:
        raise SystemExit('Go 1.27.4 is required to build gx_mapbuild.')
    go = Path(located)
version = subprocess.check_output([str(go), 'version'], text=True)
if 'go1.27.4 ' not in version:
    raise SystemExit('Expected Go 1.27.4; found ' + version.strip())
env = dict(os.environ, GOTOOLCHAIN='local')
if not a.physics_only:
    if not a.recast_source.exists():
        a.recast_source.parent.mkdir(parents=True, exist_ok=True)
        subprocess.run(['git', 'clone', '--depth', '1', '--branch', 'v1.6.0',
                        'https://github.com/recastnavigation/recastnavigation.git', str(a.recast_source)], check=True)
    helper = ROOT / 'work/mapbuild/gx_recast_baker'
    subprocess.run([sys.executable, str(Path(__file__).with_name('build.py')),
                    '--source', str(a.recast_source.resolve()), '--output', str(helper)], check=True)
    env['GX_RECAST_BAKER'] = str(helper)
command = [str(go), 'run', './server/cmd/gx_mapbuild', '-data', str(a.data.resolve()),
           '-server-maps', str(a.server_maps.resolve()), '-maps', a.maps, '-output', str(a.output.resolve())]
if a.physics_only:
    command += ['-physics-only']
if a.mob_dungeons_only:
    command += ['-mob-dungeons-only']
subprocess.run(command, cwd=ROOT, env=env, check=True)
print('Generated resources: ' + str(a.output.resolve()))
print('Install the complete generation with the world stopped; include its catalog and manifests.')
