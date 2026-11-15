#!/usr/bin/env python3
"""Build the offline baker from a pinned official Recast checkout."""
import argparse
import pathlib
import subprocess

PIN = '6dc1667f580357e8a2154c28b7867bea7e8ad3a7'
p = argparse.ArgumentParser()
p.add_argument('--source', type=pathlib.Path, required=True)
p.add_argument('--output', type=pathlib.Path, required=True)
a = p.parse_args()
revision = subprocess.check_output(['git', '-C', str(a.source), 'rev-parse', 'HEAD'], text=True).strip()
if revision != PIN:
    raise SystemExit('Expected official Recast v1.6.0 commit ' + PIN)
a.output.parent.mkdir(parents=True, exist_ok=True)
command = ['g++', '-std=c++17', '-O2', '-DNDEBUG',
           '-I' + str(a.source / 'Recast/Include'),
           '-I' + str(a.source / 'Detour/Include'),
           '-I' + str(a.source / 'RecastDemo/Include'),
           str(pathlib.Path(__file__).with_name('baker.cpp'))]
command += [str(f) for part in ('Recast', 'Detour') for f in sorted((a.source / part / 'Source').glob('*.cpp'))]
command += [str(a.source / 'RecastDemo/Source/ChunkyTriMesh.cpp'), '-o', str(a.output)]
subprocess.run(command, check=True)
