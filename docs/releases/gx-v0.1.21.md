# Shaiya GX gx-v0.1.21

Native Linux and Windows amd64 packages for the client and GX Server, with a
combined binary portable and `SHA256SUMS.txt`. See the [upgrade guide](https://github.com/Spelunkern/Shaiya-GX/blob/gx-v0.1.21/docs/releases/gx-v0.1.21-upgrade.md).

## Gameplay and presentation

- Field mobs follow terrain in XZ and pass through placed scenery. Dungeon
  mobs use structural dungeon navigation, which preserves walls and floors
  while allowing them to pass through nonstructural props. Player movement
  continues to use the full collision geometry.
- Mob death animations start on the server death notice. Attack and projectile
  contact feedback continues on its own animation timing.
- World synchronizes character idle gestures, including for nearby players.
  Remote headings interpolate and stationary turns reach nearby players.
- Basic attacks retain their native cadence on busy connections. Dedicated
  skills use their own animation timing. Several skill sounds now play at the
  authored contact or release point.
- Speech bubbles track the rendered actor at a fixed offset. Mounts use the
  resting seat as a reference. Water footsteps and hooves require water contact.
- Ladders face the authored direction at both ends and accept supported
  upper-landing descents.

## Interface, economy and packaging

- Back or Escape can cancel a connection attempt. Merchant and inventory
  windows can move independently, and close controls play the native cue.
- A warehouse tab accepts an item drop into its first free slot. Withdrawals
  use the native truncated 5% fee, and Item Mall type 94 withdrawals are free.
- Guild House prices, upkeep and NPC upgrades come from World. The client
  can run without `Npc/GuildHouse.SData`.
- Blended foliage reuses resident transforms and batches draws while preserving
  blend order. OpenGL fallbacks retain batching where the driver permits it.
- Server packages include the approved SQLite development seed, the player
  geometry catalog and the structural dungeon catalog. The independent
  [GX Studio](https://github.com/Spelunkern/GX-Studio/releases) has its own releases.
