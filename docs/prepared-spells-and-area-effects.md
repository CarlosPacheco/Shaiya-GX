# Prepared spells and area effects

## Original evidence

These findings come from the original client/server identified in
[spell-fluidity-research.md](spell-fluidity-research.md), inspected statically.
They are not timing measurements of a running original pair.

Server:

- `EnableUseSkillMob` 0x44ee50 and `EnableUseSkillUser` 0x44eb40 call
  `EnableAttackRange` before beginning preparation.
- `UpdateAttack` 0x48ca50, status 3, resolves the existing target through
  `EnableAttackMob` 0x44cd80 / `EnableAttackUser` 0x456780 and calls the skill
  execution wrappers 0x450cf0 / 0x451c80. These routes do not repeat
  `EnableAttackRange`. The target must still exist, be alive (except supported
  resurrection skills), and satisfy targeting rules. There is no separate
  2.5-unit or 100-unit release allowance in this path.
- The launch helpers 0x44bf90/0x44c030 use horizontal distance and
  `GetAttackEnableRange` 0x48a550, a movement/time-dependent allowance. GX keeps
  its existing shared launch calculation and static/dynamic geometry validation;
  this change restores the prepared-release distance behavior specifically.
- `PacketPC` accepts movement before invoking `StatusCastingStopByMove`
  0x48a090, and only when motion bit 0x80 is clear. GX previously cleared a cast
  before even validating the movement, and failed to republish the attack speed.

Client:

- The detailed area selector 0x577020 is separate from the target-impact selector
  0x577390. Blizzard 71 has an area library (`DA_area_blizzard.EFT`) and no
  target library. Using only the target selector silently loses the effect.
- `0x57bdc0` / `0x57e320` use the area selector for target-centered spells.
  Blizzard 71, Hailstorm 53 and Meteor 68 use the world shower path 0x450780;
  Blood Shower 162 and Shower 163 use that path too. Other target-centered area
  effects attach their area library to the recipients, rather than multiplying
  a full shower for each server hit.
- `0x430530`, constructor 0x447df0 and updater 0x430a80 implement the shower:
  fixed release center, five-second emission at five particles/second, signed
  integer X/Z offsets bounded by ApplyRange, 15-unit starting height, downward
  speed 15 and sideways speed/offset 7.5 along the perpendicular direction.
  Sequence zero supplies falling visuals; sequence one plays at terrain contact.
  Native runtime limits the live falling list to 50 and retains landed particles
  for a second. Hailstorm uses its separate native branch: two seconds at ten particles/second,
  height/downward speed 20 and sideways speed/offset 10.

## CMeteor placement metadata

`effect.SkillMeteor` identifies the five original world-shower routes (53, 68,
71, 162 and 163) and supplies their existing duration, rate, height, lateral
speed, sequences, contact offset and drain time. This is controller metadata;
it is not an extra EFT layer, shader, particle emitter or render pass. Radius
continues to come from the skill's original ApplyRange.

Native placement uses a fixed release center C and horizontal cast direction D:

- Side vector S = D cross (0,1,0).
- Initial position = C + signed integer X/Z offsets + (0,H,0) - S * L.
- Velocity = S * L + (0,-H,0); facing = -D, independent of velocity.
- A ground crossing triggers sequence 1 at terrain height + 1. The falling body
  remains at the sampled crossing position during its one-second drain, rather
  than snapping up to the ground. The +1 value was read from the original float
  constant at 0x6bc4dc; Phoenix's later +0.1 tuning is not copied.

The original 0x4307f0/0x430590 paths distinguish the moving mesh body from
world-space particle trails. Mesh simulation stays local to the controller;
previously emitted billboard particles remain in world space. GX carries this
placement through `SetMeteorTransform` in the existing renderer. Before this
correction, the wrapper moved but mesh particles could stay at their birth
coordinates, and the cast-facing direction was not supplied.

Phoenix's older CMeteor explanation and its current rain controller were used as
reference; native code remains authoritative. Ten isolated executions of the
original x86 block 0x430c73–0x430d52 cover cardinal/diagonal directions and both
height/speed profiles. They match GX's placement/velocity rules. This executes
only the audited calculation block, not the original game's graphics or network.

A real-assets GL comparison covers falling and contact sequences of all five
families. Changing the CMeteor placement metadata preserves emitter/template
counts and draw calls in every case: 1→1 / 2→2 (53), 2→2 / 4→4 (68),
1→1 / 2→2 (71), 2→2 / 4→4 (162), 3→3 / 3→3 (163). It adds no textures,
meshes, shaders or passes. These are draw-work checks, not a claim of identical
GPU elapsed time for differently positioned pixels.

## Result and limits

Blizzard now renders falling ice across the original data-defined area and ground
contact effects, independently of how many enemies are hit. The release center
stays fixed when the selected enemy moves. Separate recipient hit/debuff effects
remain separate. The complete detailed area library selector is restored, while
caster-centered spin presentation retains its existing target/activation routes.
The visual shower is presentation only: the server owns recipients, damage,
resources and reuse. Random offsets follow the native distribution; the original
process-wide random sequence is not reproduced.

Prepared casts survive a fleeing target without an arbitrary release-distance
cap. GX still checks target validity and visibility, and still rejects an
out-of-range initial request. GX sends motion-zero position/height refreshes even while idle. These do not
cancel a cast: they carry no locomotion intent. During preparation, motion-zero
packets cannot translate the caster horizontally, so disguising walking as an
idle refresh cannot bypass interruption. Accepted locomotion with bit 0x80 clear
still interrupts and restores attack speed. The client publishes its stop state
before the skill request and uses the same distinction for remote cast animation
and hand effects. This GX synchronization adaptation supersedes the earlier rule
that treated every accepted packet with bit 0x80 clear as an interruption.

A regression reproduced the reported loss on the first idle packet before the
fix. Afterward, twenty consecutive casts using the real Blizzard level-three
definition survive repeated idle/height updates, spend MP and resolve against a stationary
mob (normal hit/miss rolls still apply). Forged motion-zero displacement is rejected, real walking still interrupts,
and an original-assets GL test keeps the cast animation through idle updates.

Validation includes mob release well beyond launch range, player range helpers,
obstructed releases, malformed/non-finite/rejected/non-interrupting movement,
interruption speed recovery, full server integration, and original-assets OpenGL
Blizzard levels 1–3 with a fixed center and no duplicate storm per recipient.
