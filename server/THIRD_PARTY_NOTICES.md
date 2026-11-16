# Ground navigation dependencies

The offline `gx_recast_baker` uses official Recast/Detour v1.6.0,
commit `6dc1667f580357e8a2154c28b7867bea7e8ad3a7`, under the zlib license.
The server queries generated meshes with `github.com/arl/go-detour` v0.1.3
(zlib), which uses `gogeo` (MIT), `math32` (Go BSD license) and `assertgo`
(Unlicense). Unmodified notices are included in `licenses/`.

The GX baker integration is project code, not an upstream Recast modification.
Recast is needed only when generating navigation files; running GX Server
requires neither the C++ compiler nor Recast libraries.

# Modern Lua runtime

Mob policies run on `github.com/arnodel/golua` v0.4.1 (Apache-2.0), with
`github.com/arnodel/strftime` v0.1.6 (MIT). Unmodified license texts are in
`licenses/golua.txt` and `licenses/arnodel-strftime.txt`. Lua runs inside Go;
no external Lua executable, shared library or Python AI checker is required.
