# Original world collisions

The local development client predicts movement against the original EP5 world
collision geometry. `/cview` lets a GM inspect the same surfaces in game.
GX Server now enforces player movement against the same original geometry and owns mob height/navigation. Client coordinates are prediction inputs, never proof of a valid path.

## Recovered construction rules

Reference executable SHA-256:
`8e5ef14a2a3bfef15bae61b03d185e551b6c3fd44d09a2861d57428400bb3263`.
The executable and original assets remain local and outside source history.

The movement dispatcher at `0x437b60` queries the dungeon tree, five selected
placement trees and the explicit WLD blocker manager. Its selected placement
slots are `+0x13d02c`, `+0x13d0b8`, `+0x13d144`, `+0x13d25c` and
`+0x13d374`. The WLD loader `0x4518a0` identifies the groups and their directories.

| Source | Movement collision |
|---|---|
| WLD Buildings and Shapes (mode 1) | Authored SMOD collision; if absent, original `0x5ae590` builds collision from display triangles, up to 65,535 total vertices |
| WLD Trees (mode 3) | Authored SMOD collision; if absent, `0x5ae540` builds the authored collision-box faces |
| WLD Grass | Excluded by the original movement dispatcher, even when a file contains triangles |
| Empty SMOD collision | Follow the group construction mode; absence alone does not imply traversability |
| VANI-A (mode 3) | Authored final collision bounds become a box (`0x4466c0` → `0x5adbc0`); pose-zero triangle construction belongs to another loading mode |
| VANI-B | Decorative group excluded by the movement dispatcher |
| Dungeon DG, including placed field dungeons | Explicit node collision meshes with the complete placement transform |
| MANI buildings | Building collision meshes at the current authored rotor transform |
| Explicit WLD blocker volumes | Original eight corners and twelve outward-facing faces, including authored yaw |
| Terrain | Existing original height-fan surface; displayed separately by `/cview` |
| Interactive Objects such as ladders/ivy | Separate native interaction controllers; no inferred solid hull is added |

The WLD list formerly skipped as "monster-restricted zones" is actually read by
`0x5388f0` into the blocker manager at `0x2207140`, which `0x537f40` queries during
movement. Its boxes are explicitly authored collision data. `0x537840` constructs
the corners and `0x5b2fd0` uses the twelve-face index table at `0x70c3f0`.
This is distinct from substituting model display bounds for absent collision.

## Movement queries and lifetime

Original `0x4383d0` uses ellipsoid radii `(0.3, 0.9, 0.3)`, regardless of rendered
body/equipment. It scales into unit-sphere space, resolves horizontal movement
and sliding first, rejects a downward horizontal result, then resolves vertical
motion without recursive sliding. `0x437b60` uses a 0.05 sphere-space clearance
and a recursion limit of five. Triangle sweep `0x5a0ef0` tests a one-sided face,
then vertices and edges, retaining the nearest hit. The native winding remains
meaningful, including the underside of a ceiling.

`internal/collision` contains the graphics-free query/index core. The scene
adapter converts GX foot positions to the ellipsoid centre and back. Gameplay
map loading prepares all selected CPU collision resources before enabling the
world. Camera visibility, object LOD, graphics detail and GPU eviction cannot
remove blockers. SMOD collision readers retain display positions/faces for the original fallback; VANI and DG readers skip unused display buffers.
Missing or malformed required collision data reports a map-loading error.

Keyboard movement, strafing, autorun, click-to-move displacement, jumping and
vertical water motion use the same model/blocker queries. Ground settling lets
characters stand on authored model floors and fall after leaving them. Existing
terrain height fans, water thresholds and server position reconciliation remain
part of the movement flow. Click-to-move does not add a new pathfinder around
obstacles. Native ladder/ivy interaction and moving-platform passenger attachment remain
outside the validated scope.

## Step edges and model support

Support uses a short downward sweep of the original player ellipsoid against
upward-facing authored triangles. At a stair or platform edge, feet may be
below the supporting top face while the rounded bottom still touches its edge;
a point-height probe incorrectly classifies that position as airborne.

Horizontal movement retains the native upward edge slide when the result has
real model support. Blocked ground movement may attempt a swept lift, advance
and settle, bounded by the existing 0.5-unit change between supporting surfaces.
The same contact check handles jump landings and walking on small objects.
The complete body must fit below ceilings. Vertical walls cannot provide jump
support, client Y cannot create a step, and tall obstacles still require a jump
or a valid route. Both client and server use this shared resolution.

## Terrain controller

The original keeps terrain height handling separate from model ellipsoid sweeps.
Adding height-fan triangles to that sweep made slopes behave like invisible
walls. GX now applies the native height surface after sweeping authored models.
It retains actual model-step and wall checks, rather than interpreting every
terrain height change as a 0.5-unit stair.

The controller at `0x442996..0x442c98` samples three points at offsets 0.1,
converts XZ to half-world units, truncates interpolated raw heights through
`0x5cd414`, and constructs the terrain normal. Below normal Y 0.5 it adds a
progressive downhill force. The factor is `(1-normal.Y)`, multiplied by 3 below
0.3 or by 5 below 0.1. There is no immediate uphill movement veto at that
threshold: an uphill attempt slows and loses to increasing slide pressure.

GX shares this rule between client prediction and server simulation. Pressure
advances at a fixed 60 Hz equivalent using elapsed time, so display FPS and
packet counts cannot accelerate it. Server corrections include slope pressure.
Model support still rejects wall contacts as jump support; terrain does not use
the navigation agent's separate 50-degree baking cutoff.

World construction modes are set at `0x444493..0x444514`, and selected by
`0x4462d0` / `0x4466c0`. Earlier local documentation incorrectly treated every
empty collision block as traversable and VANI-A as pose-zero geometry. The
mode-specific rules above replace those assumptions. Inverted empty-bound
sentinels do not create usable collision boxes.

## GM collision view

Use `/cview` to toggle, or `/cview on` / `/cview off` for an explicit state.
The client checks the GM role supplied by the server; the command is handled
locally and never sent as public chat. Revoking the role disables the overlay.
Movement continues to collide while the view is enabled.

- Gold: walls and other blocking faces.
- Cyan: upward-facing model support surfaces.
- Blue: original terrain fan triangles.
- Magenta: the player's actual ellipsoid.

The x-ray line overlay shows nearby collision triangles, including invisible
blockers, below normal UI. It uses bounded geometry around the player, restores
OpenGL state and does not write scene depth, reflection or shadow buffers.
No texture replacement or modification of original assets is required.

## Validation and server stages

Checked-in synthetic fixtures execute the original x86 arithmetic in isolation:
304 triangle queries and 283 complete ellipsoid motions, including walls,
corners, floors, ceilings, ramps and high-speed displacement. The Go results
match within 0.0001 world/query units. Tests cover excluded Grass/VANI-B, mode-specific empty
SMOD construction, transformed solids, authored blocker yaw, missing/corrupt collision,
renderer-independent queries, GM gating and actual OpenGL output/state in both
depth modes. Local content auditing covers all 68 numeric WLDs without GL.

The Linux N2 probe uses isolated login/world processes and a disposable database
for controls, jump/landing, remote motion, travel and persistence. Separate
movement checks verify float32 integration, terrain traversal and submerged
position persistence. Windows runtime parity is not established by these Linux
checks; no Windows collision build has been produced in this local phase.

The server uses versioned resources generated from the same authored sources,
checks their hashes, limits horizontal travel using server time, and integrates
vertical motion itself. Walkable slope and step rules prevent horizontal wall
sliding from granting height. Missing player physics blocks map entry with a
diagnostic instead of silently enabling unchecked movement. Personal database
and client asset files are not changed by generation or isolated tests.

See [authoritative ground navigation](mob-navigation.md) for player/mob
ownership, protocol, regeneration commands for current and future maps, and
remaining content limitations.
