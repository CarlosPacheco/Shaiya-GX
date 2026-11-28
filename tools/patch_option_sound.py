#!/usr/bin/env python3
"""Patch interface/option_sound.png of a GX data folder (idempotent).

Removes the "NPC Sound" checkbox and label from the Etc. Option panel. The
original option (VOICE_NPC, Game.exe 0x4cb350) plays Sound/Ment/ment_*.wav
greetings that the EP5 data does not ship, so it never had an effect.
The area is covered with the blank panel interior just below it.

Usage: tools/patch_option_sound.py <data dir> [pristine.png]
The pristine art is kept at work/option_sound.original.png (outside the
client folder) and copied from the data folder on first use.
"""
import shutil
import sys
from pathlib import Path
from PIL import Image

data = Path(sys.argv[1])
root = Path(__file__).resolve().parents[1]
target = data / "interface/option_sound.png"
backup = Path(sys.argv[2]) if len(sys.argv) > 2 else root / "work/option_sound.original.png"
if not backup.exists():
    backup.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy(target, backup)
out = Image.open(backup).convert("RGBA")
# Checkbox at (129,286) and label up to x 240, rows 281..304; the blank
# interior right of "Risk Sensor" (rows 308..331) has the same backdrop.
out.paste(out.crop((124, 308, 242, 331)), (124, 281))
out.save(target)
print("patched", target)
