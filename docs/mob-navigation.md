# Authoritative ground navigation

GX Server owns player positions, gravity and jump impulses, and mob foot height and movement paths. The same implementation
handles terrain fields, placed buildings, bridges and dungeon floors. Map 3 is
the initial dungeon regression fixture; map 1 is the outdoor regression fixture.
Player packets request horizontal movement; client Y is not used as a ground or jump input.

## Server files

`data/maps/<id>.svmap` remains the source of spawn areas, their original Y
volumes, portals and the legacy blocked-cell mask. No database schema change is
required. Generated files are independent of the graphics engine:

- `data/geometry/<id>.gxnav`: tiled, layered navigation mesh.
- `data/geometry/<id>.gxcol`: compressed original collision surfaces and terrain
  samples, used for exact support height and line-of-sight checks.
- `data/geometry/<id>.gxphysics`: terrain and original collision geometry, including water flags, for authoritative player sweeps.
- `data/geometry/<id>.manifest.json`: format, agent profile, source hashes, player physics and collision-construction versions, and SHA-256 checksums for all three files.
- `data/geometry/catalog.json`: complete map set for the installed generation.

The runtime verifies checksums, format and collision construction revision 2 before use. Older construction manifests require regeneration; matching hashes alone cannot certify obsolete geometry. A missing catalogued map
or corrupt generation fails startup with a diagnostic. A map with no reachable
spawn surface logs rejected spawns; mobs are never placed under the layout to
hide the error. Player entry into a map without generated physics is refused with a diagnostic; there is no unchecked movement fallback. Portable packaging requires and validates the complete catalog.
Generated geometry belongs to the distributable data, not Git source history.

## Building data

Use the same loose client resources that will accompany the client. Original
archives, meshes and textures are not copied into the server package. The
shared `internal/mapgeometry` loader applies the original collision-group rules:
Grass, VANI and MANI instances remain excluded. Buildings/Shapes without
authored collision use the native display-triangle fallback; Trees retain
their authored-box construction policy. Static model placements remain solid
when an animated instance separately reuses their model.
The offline baker also applies the original `.svmap` blocked cells before
walkable-area erosion, allowing routes to detour around those restrictions.

The wrapper builds the offline baker from the pinned official Recast v1.6.0
checkout and then generates the requested maps. It requires Go 1.27.4, Python,
Git and a C++ compiler. The runtime remains Go and needs no Recast installation.

```sh
python3 server/tools/mapbuild/generate.py \
  --data /path/to/Client/data --server-maps server/data/maps \
  --maps all --output work/mapbuild/generated
```

For a future map (for example ID 68), first install its matching client WLD,
terrain/layout and referenced collision resources; add the matching server
`maps/68.svmap` and map configuration. Generate into a copy of the current
geometry directory using `--maps 68`. Numeric IDs are not limited to the present
catalog. The command merges the selected IDs into `catalog.json`. Install the
complete directory with the world stopped and ship the same client dataset.
Use `--maps 3` for one dungeon, or `--maps 1,3` for selected maps. Regenerate
whenever client geometry, terrain, server masks or the agent profile changes.
The baker pin is checked; licenses accompany server packages. Dense tiles retry larger 256/512-cell partitions when the Detour polygon-reference limit is reached, retaining the same cell resolution and agent profile. The manifest records `tile_size`.

An existing unchanged navigation generation can gain player physics with
`--physics-only --output /path/to/existing/geometry`. This checks original source hashes and collision construction rules. Changed content or older construction rules require a full bake. This mode
does not replace `.gxnav` or `.gxcol`. Neither mode writes into the client data.

## Movement contract

The initial ground-agent profile uses height 1.8, radius 0.3, maximum climb 0.5
and slope 50 degrees. Collision geometry determines clearance and connected
surfaces. `MobDef.Size` remains combat reach; it is not assumed to be a physical
radius. This profile does not introduce flight, swimming, ladders or moving
platform passenger attachment. Mob routes bake animated rotors in their base pose. Player sweeps use their actual animated pose with a server-synchronized monotonic clock. Moving-platform passenger attachment and dynamic mob rerouting are not implemented.

Spawn selects the first supporting surface under the original spawn volume,
with a small upward tolerance. Buried outdoor volumes fall back to the native
terrain base. It never chooses a global highest surface. Missing clearance or
missing physical support rejects the spawn with map-level diagnostics. When a
mobile mob has an authored spawn area, the server prefers a point in that area
with a short route out to nearby walkable ground. If no such point exists, it
keeps the first grounded candidate so narrow spawn areas retain their mobs.
Respawn, roaming, chase, home return, GM spawning and instanced copies use the
same ground navigation. A* must reach the requested polygon; partial paths are
rejected. Movement consumes collinear polygon crossings together and stops at actual
horizontal turns. It samples support along every segment and checks the server mask. Portal Y is resolved from the corridor polygon's detailed surface before original support checks, preserving the floor on steep terrain instead of rejecting a valid return route. Returning home follows a path and cannot teleport
through a wall. Height-aware distance and original-surface line of sight keep
mobs from attacking through walls or intervening floors.

GX packets `0x7F05` (mob enter) and `0x7F06` (mob movement) carry XYZ for grounded
mobs. Factioned spawns append country U8 (0 Light, 1 Fury, 2 neutral) and
a height-known U8 flag. The optional two-byte extension is GX-specific; native
0601 and earlier GX XYZ-only spawns remain readable. Stationary factioned actors
can transmit country without claiming an authoritative Y. Client and server
reject offensive actions against the actor's own country before cast/attack
prediction; the client uses localized SysMsg 317 (Wrong target), style 17.
Movement additionally carries its server-selected duration in milliseconds
as a trailing uint32. A 50 ms scheduler and matching client interpolation keep
short corner segments from becoming long stop-and-go pauses. Original `0x0601`/`0x0603` layouts remain unchanged for legacy entities.
The shared client revision is `0x11122209`; update Linux client and server
together. The client retains server Y and only refines interpolation against a
nearby original surface. It no longer replaces dungeon or bridge height with
zero or terrain height. The decoded map cache is bounded to four maps; mobs
retain lightweight map references rather than keeping every map resident.

## Player authority

The server limits horizontal travel using elapsed server time and the current
server movement speed. Packet floods, client timestamps and forged Y cannot
increase that allowance. Every accepted segment sweeps the original player
ellipsoid against terrain, authored collision and animated instances. Model support uses the original ellipsoid contact against upward-facing triangles, including rounded contact at stair/platform edges. Native edge sliding and bounded swept step assistance permit small changes of support height within 0.5 units while retaining body clearance checks. Terrain uses the recovered three-probe height/normal controller and progressive downhill slide instead of a hard angle gate. The 50-degree profile above applies to mob navigation baking. Model wall sliding cannot add height. Jump impulses require support;
gravity continues without incoming movement packets. Water movement remains
bounded by server speed, its surface and the physical floor. GM speed mode
changes speed only, not collision rules; explicit authorized teleports use a
separate destination validation path.

The client predicts the same ground sweeps and sends movement at 10 Hz. The
server sends `0x7F08` corrections with authoritative XYZ and vertical state,
preserving the jump trajectory during reconciliation. `0x7F07` synchronizes
animated collision timing on map entry. Login and trusted relocations select a
real supporting floor instead of the globally highest overlapping surface.
The existing map coordinate footer shows current character XYZ; cursor coordinates stay XZ. The compact minimap server-time plate remains a clock.

## Validation and content limitations

Tests cover separated floors, walls with a doorway, partial paths, gaps,
clearance, climb limits, terrain support, blocked masks, generation corruption,
protocol decoding and real spawn/path/home behavior on maps 3 and 1. Native
Linux GL tests and isolated login/world probes validate client integration.
These checks do not establish Windows runtime validation.

The local source set has an existing mismatch on maps 59 and 61: both client
WLDs reference `ev_wedding.dg`, while the server spawn areas describe another
layout. Spawns without matching surfaces are diagnosed and rejected. Restoring
those maps requires the matching authored server/client files, not an invented
collision floor or a hardcoded map-specific navigation rule.

## Reproducible release resources

Run `python3 tools/fetch_server_geometry.py` before packaging or running a
server from a fresh source checkout. The script downloads the pinned
[68-map physics resource archive](https://github.com/Spelunkern/Shaiya-GX/releases/tag/gx-geometry-rules2-20261122).
This generation was built from the gx-v0.1.16 client data and server walk
masks. The installer verifies the archive, the complete set of map manifests,
and each generated file by SHA-256. It requires a fresh directory when a
different generation is present. Both native CI platforms use the same archive.
Server portables include all the generated resources.
For future or changed maps, regenerate the resources with the map builder
described above and distribute the new generation with the server.


## Native spell sight ray and monster effects

The shared skill line-of-sight query follows native `00438010`: raise both logical
actor origins by 0.5. GX stores feet, 0.9 below those origins, so its ray passes
1.4 units above each endpoint. Low decorations can be shot over; tall walls,
intervening floors, terrain and animated colliders still block skills. This is
not a blanket exclusion of small meshes and does not change movement collision.

MON bone and attack effects follow the actor's visibility. Native `00432890`
creates attachments with the monster; `0044f2f0` / `00432bc0` use actor/frustum
visibility rather than a separate effect-distance cutoff. GX's former camera
radius of 50 units (55 after creation) has been removed. The shared per-frame
uncached emitter creation budget remains; it does not impose a view radius.


Stationary/native mob spawns can omit Y. Client skill reach, sight and approach
therefore resolve that unknown height from the displayed ground (or terrain
before a model is prepared). An absent Y must not become a target at world Y=0.
Confirmed XYZ positions retain their authoritative floor, including a different
floor; this does not bypass line of sight or faction eligibility.


VANI/MANI correction requires rebaking navigation and physics from the same
loose assets; `animation_rules: 1` identifies regenerated manifests. Keeping an
older navigation/physics generation can retain its already baked animation
obstacles even when client collision loading has been fixed.
