#!/usr/bin/env python3
"""Offline migration of original mount definitions to GX's explicit MON metadata.

Only Type 42 Range, AttackTime and Speed change. ReqIg remains untouched.
Run against stopped installations; all inputs receive backups before writes.
"""
import argparse
import configparser
import hashlib
import json
from pathlib import Path
import shutil
import sqlite3
import struct
from update_recreation_content import Native, string

RACES = {0: 'Human', 1: 'Elf', 4: 'Vail', 3: 'DeathEater'}

def definition(level, grow, legacy_range, country, bindings):
    # Native shape = 14/15 + legacy offset, then native client shape->MON.
    shape = (15 if grow > 1 else 14) + (legacy_range*2 if legacy_range < 2 else legacy_range+7)
    indices = {14:0,15:1,16:2,17:3,24:4,25:5,26:6,27:7,29:8,30:9,32:11,33:11,34:10,35:10,36:12,37:13}
    if shape not in indices or country not in RACES:
        raise ValueError(f'Unsupported native mount shape {shape}, country {country}')
    index = indices[shape]
    bone = bindings[RACES[country]][str(index)]
    return index, 2 if bone.getint('passenger', fallback=-1) >= 0 else 1, 5 if level >= 50 else 4


def patch(raw, bindings):
    src=Native(raw); groups=src.word(); out=bytearray(struct.pack('<I',groups)); rows={}
    if not 1 <= groups <= 256: raise ValueError('Expected plaintext Item.SData')
    for _ in range(groups):
        count=src.word();out+=struct.pack('<I',count)
        for _ in range(count):
            name,desc,body=src.string(),src.string(),bytearray(src.take(81))
            if body[0]==42:
                if body[65] in (4,5) and body[33] in (1,2):
                    # Already explicit: validate external bindings, keep idempotence.
                    bone=bindings[RACES[body[6]]][str(body[32])]
                    expected=2 if bone.getint('passenger',fallback=-1)>=0 else 1
                    if body[33]!=expected: raise ValueError('Mount seat metadata disagrees with bones.ini')
                    meta=body[32],body[33],body[65]
                else:
                    meta=definition(struct.unpack_from('<H',body,4)[0],body[13],body[32],body[6],bindings)
                    body[32],body[33],body[65]=meta
                rows[body[1]]=meta
            out+=string(name)+string(desc)+body
    src.end()
    return bytes(out),rows


def main():
    ap=argparse.ArgumentParser(description=__doc__)
    ap.add_argument('--data',type=Path,required=True)
    ap.add_argument('--database',type=Path,action='append',required=True)
    ap.add_argument('--backup-dir',type=Path,required=True)
    args=ap.parse_args()
    root=Path(__file__).resolve().parents[1]
    bindings={}
    for race in RACES.values():
        config=configparser.ConfigParser();config.read(root/'resources/vehicle'/race/'bones.ini');bindings[race]=config
    item=next(p for p in args.data.rglob('*') if p.is_file() and p.name.lower()=='item.sdata')
    raw,rows=patch(item.read_bytes(),bindings)
    # Validate every DB before changing any input. Original and migrated records only.
    for path in args.database:
        with sqlite3.connect(path) as db:
            current=db.execute('SELECT TypeID,Range,AttackTime,Speed,ReqLevel,Grow,Country FROM PS_GameDefs_Items WHERE Type=42').fetchall()
            if {r[0] for r in current} != set(rows): raise ValueError('Client/server mount IDs differ')
            for ident,model,seats,speed,level,grow,country in current:
                expected=(model,seats,speed) if speed in (4,5) and seats in (1,2) else definition(level,grow,model,country,bindings)
                if expected != rows[ident]:raise ValueError(f'Client/server mount {ident} differs')
    args.backup_dir.mkdir(parents=True,exist_ok=False)
    shutil.copy2(item,args.backup_dir/'Item.SData')
    for i,path in enumerate(args.database):
        with sqlite3.connect(path) as source,sqlite3.connect(args.backup_dir/f'database-{i}.sqlite') as backup:source.backup(backup)
    for path in args.database:
        with sqlite3.connect(path) as db:
            db.executemany('UPDATE PS_GameDefs_Items SET Range=?,AttackTime=?,Speed=? WHERE Type=42 AND TypeID=?', [(*meta,ident) for ident,meta in rows.items()])
        manifest=path.with_suffix('.manifest.json')
        if manifest.exists():
            metadata=json.loads(manifest.read_text());shutil.copy2(manifest,args.backup_dir/(path.parent.name+'-manifest.json'))
            metadata['sha256']=hashlib.sha256(path.read_bytes()).hexdigest();manifest.write_text(json.dumps(metadata,indent=2)+'\n')
    temp=item.with_suffix('.mount.tmp');temp.write_bytes(raw);temp.replace(item)
    for race in RACES.values():
        dest=args.data/'Vehicle'/race/'bones.ini';dest.parent.mkdir(parents=True,exist_ok=True)
        if dest.exists():shutil.copy2(dest,args.backup_dir/(race+'-bones.ini'))
        shutil.copy2(root/'resources/vehicle'/race/'bones.ini',dest)
    print(f'Migrated {len(rows)} mount definitions and four bone bindings; ReqIg preserved.')

if __name__=='__main__':main()
