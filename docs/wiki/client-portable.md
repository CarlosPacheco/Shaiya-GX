# Shaiya GX portable client

The [complete current portable](https://github.com/Spelunkern/Shaiya-GX/releases/latest/download/gx-portable-linux-windows.zip)
contains `Client/` and `Server/`, both operating systems, external fonts and
the updated server seed/maps. Add the separately downloaded client data to
`Client/data/`; the GitHub archive contains no proprietary client resources.

Download the [updated client data/portable bundle](https://drive.google.com/file/d/1ash33lTYREdbvDZRFwlCfvVu0nC8b412/view?usp=sharing) from Google Drive and extract it to a writable location. Use its client `data/` directory with the current GitHub executables, even if the Drive archive contains an older build. The resulting client layout is:

```text
Shaiya GX/
  Shaiya GX
  Shaiya GX.exe
  data/
  licenses/
```

On Linux, open `Shaiya GX`; on Windows, open `Shaiya GX.exe`. Keep `data/` beside the executables and preserve the licenses. Client game data is hosted separately from this source repository.

[GitHub releases](https://github.com/Spelunkern/Shaiya-GX/releases/latest) provide updated packages with external fonts for each OS, with `SHA256SUMS.txt`. To update, close the client, extract the appropriate package and replace that OS's executable and merge the included `data/fonts/` into your existing `data/`. Retain the accompanying notices and licenses. The GitHub client packages include fonts and licenses, but no original game artwork/audio/maps or server database; GX Server is a separate download. A future release may require a matching data update, so check its release notes.

Both executables can share the same folder and `data/` directory. Keep the included licenses and notices when redistributing. Start GX Server separately, then open the executable for your OS. The client defaults to login address 127.0.0.1:30800; use `-login IP:PORT` and `-world-port PORT` for another server. Fonts are external resources in `data/fonts/`, including lazy Unicode and colour-emoji fallbacks. Preserve that directory and its license files with the data pack. Vorbis decoding is built in. No launcher, Go installation or media tools are needed.

Windows requires a 64-bit OS and a GPU driver supporting OpenGL 4.5. Linux requires a 64-bit OS, OpenGL 4.5, ALSA and the X11/Wayland graphics libraries. The published Linux build targets the Ubuntu 24.04 runner; use glibc 2.39 or newer. An executable built locally on another Linux distribution may have different runtime requirements. If an archive tool loses execute permissions on Linux, run `chmod +x 'Shaiya GX'`.

The executable discovers adjacent data independently of the working directory. For development, `-data PATH` or `SHAIYA_DATA` overrides it. Compatible data uses PNG interface images and cursors, Ogg Vorbis audio, and an EP5 `Character/Skill.SData`. Legacy SAH/SAF archives do not load directly.

Source, build instructions and third-party font sources: https://github.com/Spelunkern/Shaiya-GX

Update GX Server and the client together. Use the updated data bundle linked
above and the gx-v0.1.8 native executables; preserve existing server runtime
databases when upgrading. See the [current patch notes](releases/gx-v0.1.5.md).


A combined binary-only `Client/` + `Server/` folder can be generated from the
four verified native archives with `tools/package_bundle.py --without-data`.
Add updated loose client game resources separately and retain the packaged
external fonts. Server maps and the development seed remain included.
