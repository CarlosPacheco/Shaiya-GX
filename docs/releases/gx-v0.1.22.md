# Shaiya GX gx-v0.1.22

This release includes the native Linux and Windows amd64 client and GX Server
packages, a combined binary portable and SHA-256 checksums. See the
[upgrade guide](https://github.com/Spelunkern/Shaiya-GX/blob/gx-v0.1.22/docs/releases/gx-v0.1.22-upgrade.md).

## Gameplay

- Equipment appearance changes preserve the current character animation.
  Unequip an item before dropping it.
- Monsters react immediately when hit during an idle walk. Their attack reach
  uses horizontal distance, and the native AI cadence and proximity aggro are
  restored for scripted monsters such as Cryptic the Immortal.
- Boss announcements use the raid banner and appear in violet in the upper
  chat pane, without a character prefix.
- Caelum Sacra's guarded portal opens after both guards die. Internal portals
  stay in the current instance, and resurrection returns each faction to its
  own Jungle base.

## Client and setup

- English is the default. Spanish and Brazilian Portuguese content overlays
  can be selected in Options. Restart the client to apply the language.
  Missing files and rows fall back to English. The loose data pack determines
  which translations and interface overrides are available.
- All cape customization colors can be selected. Returning to character
  selection clears the previous map's loading artwork.
- The default TCP ports are 31000 for login and 31010 for World, so GX and a
  legacy server can run on the same machine.
- [GX Studio](https://github.com/Spelunkern/GX-Studio/releases/tag/v0.1.0-beta.4)
  remains available as a standalone editor.
