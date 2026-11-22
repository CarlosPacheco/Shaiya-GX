# Shaiya GX gx-v0.1.17

Changes since the gx-v0.1.16 full portable.

## Movement, combat and maps

- W+Q and W+E keep their initial heading while the camera orbits. Right-dragging while standing preserves the character's facing. GM turbo preserves the normal A/D turn rate.
- Explicit movement cancels basic attacks. Damage numbers, pain sounds and hit reactions follow the swing's contact markers. Skill effects appear in front of the caster.
- Skills that require a weapon reject incompatible equipment. Learning weapon or shield mastery refreshes the affected stats immediately.
- Safe zones now follow the original boundaries on maps 45, 46 and 47. Dungeon `/town` uses the server's respawn rules, and World detects portals with their original activation bounds.
- Dead characters appear as spirits with a free camera. Returning to character selection or exiting revives the character.

## Items, appearance and interface

- Consumables covered by the stack update can stack to 255. The client reads limits from Item.SData, and the server applies the corresponding catalog update with SQLite migration 36.
- The appearance window shows a live preview with the original hair and face limits. Sex-change stones require removing sex-restricted equipment first.
- Right-click uses items from the inventory and the Z/X potion slots. The skill window shows skills unlocked by the character's level.
- The HUD scales for smaller windows. Loading art fills the window, and each loading screen picks one message from `data/Tip.INI`.
- The client supports `WorldMap.INI`, `DungeonMiniMaps.INI`, `MapScreens.INI` and `Filter.INI`. More interface text can be translated from `sysmsg-uni.txt`.

## Graphics diagnostics and downloads

- `logs/client.log` records the OpenGL context, GPU capabilities and the name of each shader program before compilation and linking. The performance HUD averages available GPU timing samples.
- Server packages include the verified collision, player physics and navigation for all 68 maps. The distributed development seed is at schema 36.

GitHub includes Windows and Linux binaries for Shaiya GX, GX Studio and GX Server, plus fonts and server resources. The combined binary portable takes compatible loose client data in `Client/data/`.

[Upgrade guide](https://github.com/Spelunkern/Shaiya-GX/blob/main/docs/releases/gx-v0.1.17-upgrade.md) · [Quickstart](https://github.com/Spelunkern/Shaiya-GX/blob/main/docs/quickstart.md)
