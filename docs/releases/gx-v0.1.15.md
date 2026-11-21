# Shaiya GX gx-v0.1.15

Changes since the gx-v0.1.14 full portable.

## Movement, maps and combat

- Movement holds up through slow frames and bursts of server packets. Collision checks stop doing extra slope work when an obstacle already blocks movement. Swimming uses the original movement speed, ladders use the original speed tiers, and monster hops use constant movement speed.
- Double-clicking an enemy keeps tracking the target as it moves. Monster picking uses native model bounds and placement, reducing interference from large bosses. Pickup, attack and talk cursors follow the original priority.
- Existing portal routes use 248 recovered original trigger volumes. Cloron's Lair uses its original C2-to-C1 volume, and the C2-to-C3 landing point has been moved onto walkable ground.
- Character and monster animation playback follows the original speed indices. NPCs face the player they are speaking to. Nameplates use native model heights and NPC role captions.
- Players and nearby observers can see spawn and faction-specific level-up effects. Leveling up and learning or upgrading a skill play their original feedback. Spawn effects stay anchored, including on monsters. GMs can use `/set level` to change their own level from 1 through the cap, with level-up feedback and equipment preserved.
- Skill casts and attacks respect server cadence under rapid inputs. Instant skill animations and effects begin together when the server confirms the skill. Character progression, EXP thresholds and revival rules are shared across former growth modes. Authored experience and combat chance bonuses use explicit percentages and stack according to their effect rules.

## Items and guilds

- Restored original status cures, protection scrolls, linked Anti-Lapis immunity, skill reset items and consumable skill links. Resurrection uses its skill or normal revival flow. Obsolete utility, protection and resurrection entries were retired from their catalogs. Anti-Lapis tooltips show guaranteed immunity and use the matching icons.
- Throw items onto the ground by dragging them out of the inventory. Inventory bags return with the right state after revival or teleport. Firecrackers play their effect and sound for nearby players and display the notice.
- The cape merchant offers emblem customization with a movable window, a rear-view preview, cloth decals and live updates. Capes are excluded from lapis linking.
- Found a guild solo or with an agreeing group. The configurable founding fee is charged once the guild is created, and characters can reapply immediately after leaving. Guild houses and their services can operate independently of ranking placement. Guild and personal warehouses have separate controls. The guild shop no longer stalls account logout after a purchase.
- A guild can reenter its ranking battle until the event closes. Eligible monster kills determine the score. GMs can test guild ranking outside the schedule and guilds founded by GMs receive an active house immediately.

## Interface, audio and graphics

- Character selection remembers the last played slot per account. Chat stays available while using auto-run or click-to-walk. Local word filtering, name visibility settings, public speech balloons and raid notices work in the live client. The custom chat auto-hide option was removed.
- The User Options menu can reject duels, trades and party or raid invitations, ask before spending skill points, toggle screen shake and speech balloons, and hide floating damage numbers. Earthquake, Earth Spikes and shower impacts use the shake setting.
- The Risk Sensor warns about nearby enemies on the minimap and plays original battle music when enabled in Sound Options. Character Sound controls original character voices independently of weapon and spell audio.
- The Revolver controls follow the bar when it rotates. The same right-button drag can close a player context menu and move the camera. Mouse controls apply correctly, the seated camera stays steady, and the normal zoom reaches farther.
- Geometry detail and texture quality are independent. The Details slider applies rendering presets live. Native helmets render their inner surfaces when visible, and movement remains smooth when remote characters change cosmetic settings.

GX Studio now opens native item catalogs containing reserved empty slots and preserves those slots when saving.

The GitHub archives contain native Windows and Linux builds of Shaiya GX, GX Studio and GX Server, with external fonts, server resources and the schema-35 development seed. Use matching loose client data for the updated interface and catalogs. The [complete gx-v0.1.14 portable on Google Drive](https://drive.google.com/file/d/1TPyHqlapcyItIbuVFsL5ISzapfQPbnPC/view?usp=sharing) is the previous data snapshot.

[Upgrade guide](https://github.com/Spelunkern/Shaiya-GX/blob/main/docs/releases/gx-v0.1.15-upgrade.md) · [Quickstart](https://github.com/Spelunkern/Shaiya-GX/blob/main/docs/quickstart.md)
