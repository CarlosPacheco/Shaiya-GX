# Original terrain surface

GX previously rendered every two-unit FLD height sample, with one fixed cell
diagonal. Valid original files can contain values between the native mesh
vertices that create spikes or pits when rendered this way.

The reference EP5 `Game.exe` SHA-256 is
`8e5ef14a2a3bfef15bae61b03d185e551b6c3fd44d09a2861d57428400bb3263`.
Direct execution in an isolated x86 emulator establishes:

- `0x5b3720` queries four triangle planes in a four-sample square. Its vertices
  are the four corners and the centre: an eight-unit square with a four-unit
  centre offset. It does not interpolate every source sample or blur neighbours.
- `0x5b75c0`, reached from `0x5b77b0`, limits ROAM subdivision when both base
  coordinate differences are below eight samples, after splitting that node.
- `0x5b6160` emits the selected near-camera mesh. A 32-unit block produces
  41 shared vertices and 64 triangles, exactly sixteen four-triangle fans.
  Original corner/centre heights survive unchanged; unused intermediate samples
  do not become vertices. An isolated spike fixture confirms this distinction.

GX now uses this **finest native surface** at every distance. Distance-based
ROAM collapse is not implemented by this change. Existing streaming, material
sections, visibility culling and batching remain in use. A 64-unit cell changes
from 1,089 vertices / 2,048 triangles to 145 vertices / 256 triangles: 87.5% fewer
terrain triangles, not an equivalent reduction in total scene cost.

Height queries, terrain drawing, camera/picking queries, water classification,
bounds, occlusion and character-shadow receivers use the same surface. Derived
height storage fills the unused lattice points from the five native anchors;
FLD source samples and all asset files remain unmodified. Unusual non-native
small grids retain the previous fallback. Edge queries clamp safely to the map
rather than reproducing the executable's out-of-buffer corner reads.

`tools/terrain_oracle.py` requires optional `pefile` and `unicorn` analysis
packages. It accepts only the reference hash and runs original height, variance,
near-camera selection and mesh emission. Its only substituted game operation is
heap allocation for nodes; it does not initialize the game or invoke OS,
network or graphics APIs.

```sh
python tools/terrain_oracle.py /path/to/original/Game.exe \
  --data /path/to/loose/data --output-dir work/terrain-native/oracle
```

Validation covered 1,000 synthetic height queries, 24 real outdoor-map crops
(16 native queries each), and exact synthetic mesh topology. Native float-store
rounding differs by less than 0.00024 world units in these height probes. Unit
fixtures retain 160 synthetic native queries and the original emitted synthetic
mesh; no original map data or executable bytes are included.

The complete map-1 sample comparison found a maximum 71.275-unit deviation at
X=1938, Z=1936 (old sample 220.54; native surface 149.265). This is a geometry
comparison, not a claim that every player-visible discontinuity is removed.
Real OpenGL checks cover maps 1, 2 and 19, shared borders, shadow receivers,
streaming, water, occlusion and AA. Local logs and before/after captures are kept
under `work/terrain-native/`.

## Model and authored-volume collisions

The terrain fan is distinct from world model blockers. See
[Original world collisions](collisions.md) for original group selection,
SMOD/DG/VANI collision data, WLD volumes, player movement and GM `/cview`.

## Moving jumps and airborne presentation

A moving client sends its pending validated position before the native jump
motion request, which carries no coordinates. Both sides then start the impulse
on the same support surface, including small steps. The client checks the shared
support geometry before predicting another jump; the server independently
requires support and retains ownership of vertical physics.

The server continues broadcasting gravity and landing to viewers. An active
owner no longer receives an unconditional landing correction containing X/Z
from its previous 10-Hz sample: that stale snapshot rewound newer predicted
movement. Invalid height/path/speed samples still receive corrections, and an
owner that stops sending input still receives authoritative gravity/landing
updates after the 300-ms idle threshold.

The predictor carries the sub-coordinate fraction between frames. World
subdivides the same horizontal displacement for collision checks while keeping
the rounded endpoint, and charges that endpoint against its elapsed-time
movement budget. This prevents per-frame rounding from accumulating at high
frame rates and triggering corrections on clear ground. Periodic samples use
the position at the end of the rendered frame. World retains bounded elapsed
time for delayed running samples and keeps the per-sample speed cap. A small
speed-only discrepancy on solid ground has one sample to resolve. Corrections
update the physical position immediately. While walking or running, the visible
model blends small corrections across clear ground over 180 ms. Falls, blocked
paths and large corrections snap to the server position.

Jump presentation plays original ALT action 8 once and lets it run to its end even after the character lands: the 0.8 s clip outlasts the 0.69 s of a flat jump, and the original ends the jump state only when the clip finishes (0x418030, 0x419f60), so landing shows its last frames instead of cutting to idle. Its animation duration is
independent of the physical time spent airborne. Once it finishes, continued
falling uses run or walk only while horizontal movement is active. Releasing
movement returns to the standard idle pose; uncommanded falls also use idle. Switching clips at action completion prevents a
wrapped first jump frame from appearing during the transition.

The WLD `Objects` group is Phoenix's `Ladder` section. Its original SMOD models
are loaded by both static and streamed rendering, with two-sided surfaces so
thin ladders and climbable ivy remain visible from either direction. They stay
non-solid as in Phoenix; this visual support does not introduce a climbing
movement mode or change server collision resources.

Server-authoritative falling can apply the optional, default-disabled [GX fall-damage
rules](fall-damage.md) on supported landings. Spawn settling, teleport resets
and deep-water entry do not inherit an earlier drop.

Native WLD Ladder/Object placements also support shared client/server climbing,
with server-owned vertical motion and collision-checked landings. See
[functional ladders](ladders.md) for controls and the asset-free guide catalog.

## Portal volumes

World uses the original WLD trigger bounds for the matching maps and routes
listed in `server/data/game/PortalBounds.json`. Each row is keyed by map and
SVMAP portal index. The destination map and source position must match before
World uses the bounds. Routes without a verified match keep the SVMAP point
trigger. World checks the actor origin above the feet, as in EP5. The C2
portal to Cloron's Lair C3 has an arrival height override for the upper entry
corridor at the authored X/Z position.
