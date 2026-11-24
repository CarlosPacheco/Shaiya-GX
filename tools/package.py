#!/usr/bin/env python3
"""Build a private loose-data Linux client: Shaiya GX and its adjacent data/."""
import argparse
import os
from pathlib import Path
import shutil
import subprocess
from font_pack import FONT_PACK, validate
from gx_ini import FILTER_TEMPLATE
from item_sdata import RETIRED_STACK_CSV

ROOT = Path(__file__).resolve().parents[1]


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--source', type=Path, default=Path(os.environ.get('SHAIYA_DATA') or ROOT/'data'), help='converted loose data directory')
    p.add_argument('--output', type=Path, required=True, help='destination play folder')
    p.add_argument('--update', action='store_true', help='replace executable atomically and preserve installed data/')
    args = p.parse_args()
    source, output = args.source.expanduser().resolve(), args.output.expanduser().resolve()
    data = output/'data' if args.update else source
    if not data.is_dir(): p.error(f'missing loose data directory: {data}')
    if not args.update and (source == output/'data' or output.is_relative_to(source)): p.error('source and output overlap')
    output.mkdir(parents=True, exist_ok=True)
    env = dict(os.environ, PATH=str(Path.home()/'.local/opt/go-1.27.4/bin')+os.pathsep+os.environ.get('PATH',''), GOTOOLCHAIN='local', GOPATH=os.environ.get('GOPATH',str(Path.home()/'.local/share/go')))
    staged = output/'.Shaiya GX.next'
    subprocess.run(['go','build','-o',str(staged),'./cmd/shaiya'],cwd=ROOT,env=env,check=True)
    if not args.update:
        shutil.copytree(source, output/'data', ignore=lambda _, names: [
            name for name in names if name.casefold() == RETIRED_STACK_CSV])
    if not (output/'data/Filter.INI').exists():
        (output/'data/Filter.INI').write_text(FILTER_TEMPLATE, encoding='utf-8', newline='\n')
    if not any(path.name.casefold() == 'tip.ini' for path in (output/'data').iterdir()):
        shutil.copy2(ROOT/'resources/client/Tip.INI', output/'data')
    validate()
    shutil.copytree(FONT_PACK,output/'data/fonts',dirs_exist_ok=True)
    staged.replace(output/'Shaiya GX')
    for path in (output/'data').iterdir():
        if path.name.casefold() == RETIRED_STACK_CSV and (path.is_file() or path.is_symlink()):
            path.unlink()
    print(f'Client ready: {output}/Shaiya GX + data/',flush=True)


if __name__ == '__main__': main()
