# Shaiya GX gx-v0.1.23

This release includes the native Linux and Windows amd64 client and GX Server
packages, a combined binary portable and SHA-256 checksums. See the
[upgrade guide](https://github.com/Spelunkern/Shaiya-GX/blob/gx-v0.1.23/docs/releases/gx-v0.1.23-upgrade.md).

## Gameplay

- Monsters and guards move over the original `.svmap` walk mask with the
  native line test, grid A* and chase trail, as the original server does.
  Monsters also approach targets standing on blocked tiles.
- Killing a monster prints the original System messages for you and your
  party members.
- The false Exiel holds its position until it vanishes into its guards.
- `[RandomRespawn]` in `Obelisk.ini` is now an optional appearance chance on
  top of the boss's normal timing. The boss appears at one of its listed
  coordinates; in instances the chance is rolled when the instance is
  created. Evilmidget Clavi has a 1% chance per instance.
- New GM command: `/bless` grants your country the goddess blessing.

## Quests

- Hidden item quests are offered in the original quest window with NPC face
  55, and finished branches close as in the original.
- Quest windows show up to three rewards with item icons. Click one to
  choose it when taking or completing the quest.
- The quest journal shows required item icons with tooltips.

## Interface

- Item Mall: wider window with a single Points panel, the product list on
  the left and a character preview on the right. The selected product is
  shown equipped (helmet, armor, gloves, boots, weapon, shield or cloak);
  mounts show the mount. Drag in the preview to rotate it.
- Graphics options add VSync and anisotropic filtering. The NPC Sound
  option, which had no sound files in EP5 data, is removed. The Sub Skill
  window no longer shows the unused craft-skill tab.
- Multi-language support is removed. The client uses the base data text.

## Rendering and performance

- Cloaks collide with the body and terrain and react to the native wind.
  Character previews in the interface no longer stretch the cloak.
- Monster effects follow the monster, use its model space and scale like the
  original.
- Effect batching merges alpha and additive runs with premultiplied alpha,
  contributed by Gexur (#41), with a fallback for fragmented passes.
  Crowd skin groups are sorted to reduce state changes, which helps AMD
  drivers.
- Player collision queries are about twice as fast and use 18-22% less
  memory per map. Navigation meshes are decoded on first use.
