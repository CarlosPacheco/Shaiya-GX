# Shaiya GX portable client and server

## [Download the complete portable — Client + Server + Data](https://drive.google.com/file/d/1EF83MEmACo9xPYenPP59OBzPivRH9bMl/view?usp=sharing)

**gx-v0.1.10 for Windows and Linux.** This Google Drive package includes both
executables, GX Server, client game data, fonts and the development database.
**Everything is included; no separate data download is needed.**
Extract the entire package into a writable folder:

```text
Shaiya GX gx-v0.1.10 Portable/
  Client/
    Shaiya GX
    Shaiya GX.exe
    data/
    licenses/
  Server/
    build/
    data/
```

Start `Server/build/gx_server` on Linux or `Server/build/gx_server.exe` on Windows,
then click **Start server**. Open `Client/Shaiya GX` or `Client/Shaiya GX.exe`.
Keep `Client/data/` beside the client executables and preserve all licenses.

**Latest binary update: [gx-v0.1.10](https://github.com/Spelunkern/Shaiya-GX/releases/tag/gx-v0.1.10).**
The Drive download includes gx-v0.1.10 client/server builds and refreshed data
for mount sounds, dungeon minimaps and creature repairs. Update client and server
together. Preserve an existing
server `runtime/` and follow the [mount migration and upgrade guide](releases/gx-v0.1.10-upgrade.md).
Client packages now include `data/Vehicle/bones.ini` alongside fonts; merge both.
Server packages include all 68 physics/navigation maps, the approved development
seed and the optional offline mount-definition migration tool.

The [GitHub combined binary portable](https://github.com/Spelunkern/Shaiya-GX/releases/latest/download/gx-portable-linux-windows.zip)
has the same `Client/` and `Server/` layout, with fonts and the server seed/maps/physics,
and is intended for optional binary updates to an existing complete installation.
For a first installation, use the all-in-one Drive download above. Original client
resources are not included in the source repository or GitHub binary archives.

[GitHub releases](https://github.com/Spelunkern/Shaiya-GX/releases/latest) provide updated packages with external fonts for each OS, with `SHA256SUMS.txt`. To update, close the client, extract the appropriate package and replace that OS's executable and merge the included `data/fonts/` into your existing `data/`. Retain the accompanying notices and licenses. The GitHub client packages include fonts, authored mount bindings and licenses, but no original game artwork/audio/maps or server database; GX Server is already included in the complete Drive package. A future release may require a matching data update, so check its release notes.

Both executables can share the same folder and `data/` directory. Keep the included licenses and notices when redistributing. Start GX Server separately, then open the executable for your OS. The client defaults to login address 127.0.0.1:30800; use `-login IP:PORT` and `-world-port PORT` for another server. Fonts are external resources in `data/fonts/`, including lazy Unicode and colour-emoji fallbacks. Preserve that directory and its license files with the data pack. Vorbis decoding is built in. No launcher, Go installation or media tools are needed.

Windows requires a 64-bit OS and a GPU driver supporting OpenGL 4.5. Linux requires a 64-bit OS, OpenGL 4.5, ALSA and the X11/Wayland graphics libraries. The published Linux build targets the Ubuntu 24.04 runner; use glibc 2.39 or newer. An executable built locally on another Linux distribution may have different runtime requirements. If an archive tool loses execute permissions on Linux, run `chmod +x 'Shaiya GX'`.

The executable discovers adjacent data independently of the working directory. For development, `-data PATH` or `SHAIYA_DATA` overrides it. Compatible data uses PNG interface images and cursors, Ogg Vorbis audio, and an EP5 `Character/Skill.SData`. Legacy SAH/SAF archives do not load directly.

Source, build instructions and third-party font sources: https://github.com/Spelunkern/Shaiya-GX

Update GX Server and the client together. Use the complete package linked
above, containing matching client/server builds and game data; preserve existing server runtime
databases when upgrading. See the [current patch notes](releases/gx-v0.1.10.md).


## Optional maintainer packaging

The complete Drive download above needs no assembly. Maintainers who want a
binary-only folder from GitHub release archives can run:

```sh
python3 tools/package_bundle.py --archives /path/to/archives --without-data --output /path/to/new-folder --tag gx-v0.1.10
```

The resulting `Client/` and `Server/` directories contain both operating
systems. Add updated game resources to `Client/data/` separately, retaining the
packaged external fonts. The server includes its distributable maps and seed;
keep an existing personal runtime separate during upgrades.

## Startup diagnostics

Missing or damaged interface images use placeholders instead of aborting startup.
See `logs/client.log` beside the executable (and `client.previous.log` for the
preceding session). Read-only installations fall back to the user cache or
temporary directory. Blocking errors display the diagnostic location; see
[troubleshooting](wiki/troubleshooting.md#startup-and-incomplete-interface-resources).
