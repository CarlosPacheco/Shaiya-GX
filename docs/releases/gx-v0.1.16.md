# Shaiya GX gx-v0.1.16

Changes since the gx-v0.1.15 full portable.

## World, movement and combat

- Diving gives you 30 seconds of air. An oxygen bar empties underwater, refills at the surface and disappears when full. Staying submerged after it empties causes drowning damage.
- Ground movement uses the original contact controller. Low obstacles, curbs and ramps that the original allows are easier to cross. Climbers are drawn along the ladder guide.
- Jump animations finish through landing, including uphill jumps and delayed combat replies. An attack or skill ordered during a jump in place waits until landing and the end of the animation. Horizontal movement during the jump discards that order.
- Routine monster spell casts now vary between attack opportunities. Mobile spawns stay on connected ground. Monster loot lands on dungeon floors, and cross-map dungeon revival goes straight to the loading screen.
- Fixed misplaced terrain textures on maps 43, 44 and 46. Concealed characters keep their cloaks, emblems and cloak shoulder pieces hidden.
- The server schedules attacks and skills against their shared animation cadence. Trap and Demon Hushing play a hit sound when they deal damage, and Ground Shock faces the right way.

## Equipment, controls and interface

- Gear and lapis attack and movement speeds recover after revival. The Status sheet refreshes after a failed lapis link, attribute allocation or stat reset. Hold an attribute button to distribute points at ten per second, or tap to change one point. The draft shows its contribution next to the current base stat.
- Merchant, warehouse and other NPC service windows close when you walk out of range. The inventory opens temporarily to the left of merchant and warehouse windows. Items move between warehouse bags, and dragged icons stay above the inventory. Warehouse Call opens personal storage and can be reopened from its active buff.
- Drag service and character windows from their empty areas. The Revolver keeps its center when rotated. Personal buff tooltips show remaining duration. Interface typography follows the original EP4 reference.
- Forward and backward movement follow the camera. Targeting acts on press, an empty click keeps your selection, and clicking a selected target engages it. Same-faction players can be followed, and `F` copies a selected group member's target. `H` toggles separate catalog type and ID displays. The `N` shortcut for finding an NPC was removed.
- `F12` and Print Screen save a PNG in the `screenshot` folder beside the client. Saving runs in the background and the game displays the original capture message.

## GM commands

- `/cure` clears current debuffs. `/char` and `/attack` control concealment and attackability. Protected GMs are ignored by guards, and `/attack on` also prevents monster aggro. `/turbo` reports that fall damage and the server's horizontal speed limit are disabled while active.

The GitHub downloads include native Windows and Linux builds of Shaiya GX, GX Studio and GX Server, plus fonts, server resources and the schema-35 development seed. The [complete gx-v0.1.16 portable on Google Drive](https://drive.google.com/file/d/1-yK2bZAsWv3nKK-RBryVa-tstWfU3pRz/view?usp=sharing) includes the matching loose client game data and interface artwork.

**Server geometry update, November 22.** The GitHub server packages and combined portable include freshly generated collision, player physics and navigation for all 68 maps, using the gx-v0.1.16 client data. If you installed the portable before this update, use the [server update instructions](https://github.com/Spelunkern/Shaiya-GX/blob/main/docs/releases/gx-v0.1.16-upgrade.md#server-geometry-update) to apply the corrected resources.

[Upgrade guide](https://github.com/Spelunkern/Shaiya-GX/blob/main/docs/releases/gx-v0.1.16-upgrade.md) · [Quickstart](https://github.com/Spelunkern/Shaiya-GX/blob/main/docs/quickstart.md)
