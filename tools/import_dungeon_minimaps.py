#!/usr/bin/env python3
"""Import missing original minimaps into loose GX data; never replace existing art."""
import argparse
import configparser
import hashlib
import json
from pathlib import Path
import struct

from PIL import Image

from gx_ini import dungeon_to_ini, read


def child(parent, name):
    return next((p for p in parent.iterdir() if p.name.casefold() == name.casefold()), None)


def import_minimaps(source, data, server_maps):
    source, data, server_maps = map(Path, (source, data, server_maps))
    cfg = child(source, 'Dungeon_MiniMap.cfg')
    interface = child(source, 'Interface')
    images = child(interface, 'MiniMap') if interface else None
    if cfg is None or images is None:
        raise ValueError('source needs Dungeon_MiniMap.cfg and Interface/MiniMap')
    parsed = configparser.ConfigParser()
    parsed.read_string(cfg.read_text(encoding='latin1'))
    count = parsed.getint('WorldMap', 'count')
    if not 1 <= count <= 4096:
        raise ValueError('invalid dungeon calibration count')
    identities = set()
    for n in range(count):
        row = parsed[str(n)]
        identity = row.getint('MapNum')
        if identity in identities or not 0 <= identity <= 65535 or min(row.getint('ImgPos_x'), row.getint('ImgPos_y')) <= 0:
            raise ValueError('invalid or duplicate dungeon calibration')
        identities.add(identity)
    world = child(data, 'world')
    if world is None:
        raise ValueError('destination needs the playable world files')
    playable = sorted(int(p.stem) for p in world.iterdir() if p.suffix.lower() == '.wld' and p.stem.isdigit())
    dest_interface = child(data, 'interface') or data / 'interface'
    dest = child(dest_interface, 'MiniMap') if dest_interface.exists() else None
    dest = dest or dest_interface / 'MiniMap'
    dest.mkdir(parents=True, exist_ok=True)
    available = {p.name.casefold(): p for p in images.iterdir() if p.is_file()}
    records, missing, preserved, incompatible = [], [], [], []
    source_world = child(source, 'World')
    for identity in playable:
        if child(dest, f'{identity}.png'):
            preserved.append(identity)
            continue
        original = next((available[f'{identity}{ext}'] for ext in ('.png', '.tga', '.dds') if f'{identity}{ext}' in available), None)
        if original is None:
            missing.append(identity)
            continue
        source_layout = child(source_world, f'{identity}.wld') if source_world else None
        target_layout = child(world, f'{identity}.wld')
        if source_layout:
            with source_layout.open('rb') as stream:
                core_header = stream.read(260)
            with target_layout.open('rb') as stream:
                target_header = stream.read(260)
            different_kind = core_header[:4] != target_header[:4]
            different_dungeon = (core_header[:3] == b'DUN' and
                                 core_header[4:260].split(b'\0')[0].lower() !=
                                 target_header[4:260].split(b'\0')[0].lower())
            if different_kind or different_dungeon:
                incompatible.append(identity)
                continue
        target = dest / f'{identity}.png'
        temporary = target.with_suffix('.importing')
        with Image.open(original) as image:
            rgba = image.convert('RGBA')
            rgba.save(temporary, format='PNG')
            with Image.open(temporary) as check:
                if check.size != rgba.size or check.convert('RGBA').tobytes() != rgba.tobytes():
                    raise ValueError(f'pixel mismatch: {identity}')
            records.append(dict(map=identity, dimensions=list(rgba.size),
                                rgba_sha256=hashlib.sha256(rgba.tobytes()).hexdigest(),
                                source_sha256=hashlib.sha256(original.read_bytes()).hexdigest(),
                                png_sha256=hashlib.sha256(temporary.read_bytes()).hexdigest()))
        temporary.replace(target)
    # Native CFG copied intact. This separate GX metadata only inverts GM image
    # clicks to the existing server-normalized request; it grants no authority.
    sizes = {}
    for p in server_maps.iterdir():
        if p.suffix.lower() != '.svmap' or not p.stem.isdigit():
            continue
        with p.open('rb') as stream:
            raw = stream.read(4)
        if len(raw) != 4:
            raise ValueError(f'truncated server map: {p.name}')
        size, = struct.unpack('<I', raw)
        if not 0 < size <= 8192:
            raise ValueError(f'invalid server map size: {p.name}')
        sizes[int(p.stem)] = size
    sizes_file = dest / 'map-sizes.json.importing'
    sizes_file.write_text(json.dumps(dict(sorted(sizes.items())), indent=2) + '\n')
    sizes_file.replace(dest / 'map-sizes.json')
    cfg_file = data / 'DungeonMiniMaps.INI.importing'
    cfg_file.write_text(dungeon_to_ini(read(cfg)), encoding='utf-8', newline='\n')
    cfg_file.replace(data / 'DungeonMiniMaps.INI')
    return dict(imported=records, preserved=preserved, missing=missing, incompatible=incompatible,
                calibration_sha256=hashlib.sha256(cfg.read_bytes()).hexdigest())


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source', type=Path, required=True, help='original files extracted with Parsec')
    parser.add_argument('--data', type=Path, required=True, help='loose GX data directory (stage before installing)')
    parser.add_argument('--server-maps', type=Path, required=True, help='matching server data/maps directory')
    parser.add_argument('--report', type=Path, required=True)
    args = parser.parse_args()
    report = import_minimaps(args.source, args.data, args.server_maps)
    args.report.write_text(json.dumps(report, indent=2) + '\n')
    print(f"Imported {len(report['imported'])}; preserved {len(report['preserved'])}; unavailable {report['missing']}; incompatible {report['incompatible']}")


if __name__ == '__main__':
    main()
