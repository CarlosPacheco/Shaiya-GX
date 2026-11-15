Shaiya GX gx-v0.1.9

• Collision and height: players and mobs respect terrain, dungeon floors and obstacles. Improved stairs, small steps and slopes; server validates movement and jumping. GM collision viewer: /cview.
• Movement and attack speeds are server controlled, including equipment, lapis, passives, buffs and consumables. Stats and item descriptions show their bonuses. Corrected Crun and cast-bar positions.
• Jumping while running no longer snaps backward on landing. Long falls return to run/walk after the jump animation.
• GM map-click teleport works on the minimap, full map and World Map previews, with server-resolved height. Drag to pan; no /goto command. Partially adapts PR #7 by juuma.
• OJ descriptions now include the maximum attribute count, e.g. Max OJ 13/1300 (4 stats). Server-supplied limits keep stale client descriptions aligned with drops; 94,300 catalog rolls verified.
• Removed /char and /attack GM commands; spell invisibility remains. Ground-item text labels stay hidden.
• Updated native Linux/Windows client/server portables, including physics resources for all 68 maps and the existing verified development DB seed.

Update client + server together (protocol 0x11122208). Keep your runtime DB and client game data. GitHub packages contain binaries, fonts and complete server resources; game artwork comes from the all-in-one Drive base linked in README. Windows CI validates native builds/startup; full visual gameplay still requires local game assets.

https://github.com/Spelunkern/Shaiya-GX/releases/tag/gx-v0.1.9
