# Walking collision: GX assists versus the original controller

Evidence: the Unicorn replay of the original x86 controller
(`0x442d9a..0x44305f` calling the ellipsoid resolver `0x4383d0`) against GX's
`mapgeometry` code, on 137 routes (93 synthetic, 44 cut from original SMOD/DG
assets) at 30, 60 and 120 Hz. The harness lives in
`work/asset-collision-research` (`native-replay.py`, `probe.go`).

## Result

| Walking model | Routes that differ by more than 0.1 unit | Sum of final separations |
| --- | ---: | ---: |
| GX today (`GroundMove` assists + `StandingPosition`) | 61 of 137 | 121 |
| Original shape: pure ellipsoid slide, then the vertical fall step | 1 of 137 (max 0.25, stairway07) | 0.57 |

So the collision core (`MoveFeet`, i.e. `0x4383d0`) is already right. The
differences are the decisions GX adds around it.

## Where GX blocks what the original allows

GX advances at least 0.5 unit less than the original in 19 routes:

- Ramps of 51, 55, 60, 65 and 70 degrees going up at every frame rate. The
  original climbs them (4.3 to 5.6 units, rising 2.4 to 5.5); GX stops after
  about 1.4 units with no rise.
- Curbs of 0.60 (30 and 60 Hz) and 0.75 (30 Hz). The original steps onto them
  (0.45 to 0.73 rise); GX stops after 1.2 units.
- A 1.82 high ceiling over a curb at 120 Hz.

Conversely GX passes in 8 routes where the original is blocked at 120 Hz
(curbs of 0.45 and 0.50): the original's step climbing depends on the frame
step, which GX hides.

The rules that cause it (`internal/mapgeometry`):

1. `GroundMove` reverts the whole sweep (`moved = p`) when the result rises and
   the model support is missing, more than `MaxGroundStep` (0.5) above the old
   support, or `clearBody` fails. `clearBody` uses a 0.998 threshold, so a
   grazing contact against a doorframe or tread edge can freeze the character.
2. The "lift, advance, settle" assist runs whenever progress is under 98 percent
   and can pick a different lateral line than the slide.
3. `modelSupport` and `floorSurface` discard faces steeper than 50 degrees, so a
   steep ramp is never support even though the original slides up it.
4. `StandingPosition` snaps the feet up to any support within 0.13, something
   the original never does; it grounds by the vertical contact of the next
   gravity step (-2.98 units/s, accelerating by 2.8 per second until contact).

## What a drop-in test showed

Replacing only `GroundMove` by the plain sweep plus a fixed grounded step got
the climbing cases right (61 routes down to 38) but the steep descents and two
real stairways (stairway06, Bridge00) then drift, because the original keeps a
persistent fall velocity across frames. The vertical state has to be shared
with the existing airborne code instead of being a per-call constant.

## Proposed plan

1. Add a native controller to `mapgeometry`: pure horizontal sweep per fixed 60 Hz
   step, then the vertical fall step with a persistent fall state (reset to -2.98
   on contact). It replaces the assisted `GroundMove`/`StandingPosition` path.
2. Use it in client prediction (`GroundPlayer`) and in World's movement
   validation, so both agree. Keep `GX_NATIVE_CONTACT=off` as a rollback switch.
3. Re-run the 137-route replay (target: at most 1 route off) and the existing
   wall, ceiling, jump and stair tests, adding gravity to the tests that today
   push into a wall without any.
4. Measure the real-map spots listed in the research report (map 1
   stairway05/06/08, Bridge00) on the client against World before and after.

## Implemented (permanent, confirmed in play 2026-11-21)

`mapgeometry.GroundMoveDT` now walks with the plain ellipsoid slide plus the
grounded fall of each step (-2.98 units/s times that step's duration), with no
support, step-height, slope or clearance veto, no lift assist and a model slope
limit of 85 degrees instead of 50. The client passes the frame time; World walks
a movement batch one 60 Hz frame at a time at full speed toward the reported
target, like the client predicts it. The shared vertical code (persistent fall
speed, `Supported`, `StandingPosition`) is unchanged. In the 137-route replay
with that flow the sum of final separations dropped from 130 to 25 units and the
steep ramps and 0.60 / 0.75 curbs now climb. `GX_NATIVE_CONTACT=off` (client and
World) restores the previous assisted controller.

World also walks airborne batches one 60 Hz frame at a time, horizontal then
vertical, the way the client predicts a jump, and the jump clip is left to finish
after landing (see terrain.md). `GX_NATIVE_CONTACT=off` remains only as a
rollback switch.
