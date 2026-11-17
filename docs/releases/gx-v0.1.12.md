# Shaiya GX gx-v0.1.12

Hotfix since **gx-v0.1.11**.

- Fix repeated map transfers: the client now sends the required `0x0201` load acknowledgement after every self `0x020B` placement that changes the map. Previously, the destination could appear normally while the server still considered the session loading, blocking subsequent commands and eventually disconnecting it.
- Apply the same acknowledgement boundary to the graphical client, CLI and protocol-test clients. Preserve the initial entry/encryption sequence; same-map moves, other characters, malformed packets and placement echoes do not generate extra acknowledgements.
- Add regression coverage for seven consecutive `/cmove` commands across maps 18, 45 and 1, including a same-map command, a gameplay-command barrier after each transfer and clean logout. Unit tests cover initial entry and duplicate/foreign/malformed packets.
- Refresh protocol documentation and the shared Wiki references.

The fix is client-side. The release includes matching Linux/Windows client and
terminal-only server packages with the distributed seed DB, server maps/physics,
Lua AI policies, resources and tools. Existing v0.1.11 installations need no new
data or database migration for this hotfix. Keep your runtime and configuration.

GitHub binary packages omit original client game artwork/audio/maps. The complete
[Drive package](https://drive.google.com/file/d/1YoUifTj5D-nn378cyFmyT7H4Dw4Vyegp/view?usp=sharing)
now includes **v0.1.12** for Windows and Linux, with the teleport fix, complete
client data and all server resources.

[Upgrade guide](https://github.com/Spelunkern/Shaiya-GX/blob/main/docs/releases/gx-v0.1.12-upgrade.md).
