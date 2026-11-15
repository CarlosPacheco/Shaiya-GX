# Shaiya GX portable client and server

## [Download the complete portable — Client + Server + Data](https://drive.google.com/file/d/1XU2Tk05VfO9hrtx3yIMevo9ALqFq4pW3/view?usp=sharing)

**gx-v0.1.8 for Windows and Linux.** This Google Drive package includes both
executables, GX Server, client game data, fonts and the development database.
**Everything is included; no separate data download is needed.**
Extract the entire package into a writable folder:

```text
Shaiya GX gx-v0.1.8 Portable/
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

**Latest binary update: [gx-v0.1.9](https://github.com/Spelunkern/Shaiya-GX/releases/tag/gx-v0.1.9).**
The Drive download remains the complete gx-v0.1.8 base. Close client and server,
apply the matching client/server update together, retain `Client/data/` and
preserve the existing server `runtime/` database. Merge the update's server
`data/geometry/` resources as well as its binaries and fonts. The update includes
physics/navigation for all 68 maps; no separate server resource download is needed.

The [GitHub combined binary portable](https://github.com/Spelunkern/Shaiya-GX/releases/latest/download/gx-portable-linux-windows.zip)
has the same `Client/` and `Server/` layout, with fonts and the server seed/maps/physics,
and is intended for optional binary updates to an existing complete installation.
For a first installation, use the all-in-one Drive download above. Original client
resources are not included in the source repository or GitHub binary archives.

[GitHub releases](https://github.com/Spelunkern/Shaiya-GX/releases/latest) provide updated packages with external fonts for each OS, with `SHA256SUMS.txt`. To update, close the client, extract the appropriate package and replace that OS's executable and merge the included `data/fonts/` into your existing `data/`. Retain the accompanying notices and licenses. The GitHub client packages include fonts and licenses, but no original game artwork/audio/maps or server database; GX Server is already included in the complete Drive package. A future release may require a matching data update, so check its release notes.

Both executables can share the same folder and `data/` directory. Keep the included licenses and notices when redistributing. Start GX Server separately, then open the executable for your OS. The client defaults to login address 127.0.0.1:30800; use `-login IP:PORT` and `-world-port PORT` for another server. Fonts are external resources in `data/fonts/`, including lazy Unicode and colour-emoji fallbacks. Preserve that directory and its license files with the data pack. Vorbis decoding is built in. No launcher, Go installation or media tools are needed.

Windows requires a 64-bit OS and a GPU driver supporting OpenGL 4.5. Linux requires a 64-bit OS, OpenGL 4.5, ALSA and the X11/Wayland graphics libraries. The published Linux build targets the Ubuntu 24.04 runner; use glibc 2.39 or newer. An executable built locally on another Linux distribution may have different runtime requirements. If an archive tool loses execute permissions on Linux, run `chmod +x 'Shaiya GX'`.

The executable discovers adjacent data independently of the working directory. For development, `-data PATH` or `SHAIYA_DATA` overrides it. Compatible data uses PNG interface images and cursors, Ogg Vorbis audio, and an EP5 `Character/Skill.SData`. Legacy SAH/SAF archives do not load directly.

Source, build instructions and third-party font sources: https://github.com/Spelunkern/Shaiya-GX

Update GX Server and the client together. Use the complete package linked
above, containing matching client/server builds and game data; preserve existing server runtime
databases when upgrading. See the [current patch notes](releases/gx-v0.1.5.md).


A combined binary-only `Client/` + `Server/` folder can be generated from the
four verified native archives with `tools/package_bundle.py --without-data`.
Add updated loose client game resources separately and retain the packaged
external fonts. Server maps and the development seed remain included.
