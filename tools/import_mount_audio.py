#!/usr/bin/env python3
"""Import the native mount cues omitted from MON sound slots; convert WAV to OGG."""
import argparse
from pathlib import Path
import shutil
import subprocess

NAMES=('ef_in_atrbu01','ef_dmg_transf01','run_horse-roar01','run_pumav-roar01','run_baslk-roar01')
def main():
    ap=argparse.ArgumentParser(description=__doc__)
    ap.add_argument('--source',type=Path,required=True,help='Original client archive directory')
    ap.add_argument('--data',type=Path,required=True,help='Installed loose GX data directory')
    ap.add_argument('--backup-dir',type=Path,required=True)
    args=ap.parse_args();root=Path(__file__).resolve().parents[1]
    args.backup_dir.mkdir(parents=True,exist_ok=False)
    assets=root/'build/assets'
    command=[str(assets)] if assets.exists() else ['go','run','./cmd/assets']
    for name in NAMES:
        raw=subprocess.run(command+['-data',str(args.source),'cat','Sound/'+name+'.wav'],cwd=root,capture_output=True,check=True).stdout
        dest=args.data/'Sound'/(name+'.ogg');dest.parent.mkdir(parents=True,exist_ok=True)
        temp=dest.with_suffix('.mount-audio.tmp')
        try:
            subprocess.run(['ffmpeg','-v','error','-y','-i','pipe:0','-c:a','libvorbis','-q:a','5','-f','ogg',str(temp)],input=raw,check=True)
            if dest.exists():shutil.copy2(dest,args.backup_dir/dest.name)
            temp.replace(dest)
        finally:temp.unlink(missing_ok=True)
    print('Imported five original mount cues as OGG.')
if __name__=='__main__':main()
