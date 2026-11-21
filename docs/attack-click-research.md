# Clicking to attack: native flow

Evidence from `Game.exe` EP5.4, world update `0x43bfa0` (helper scripts in
`tools/ghidra`). The world keeps one action state (`DAT_0071d000`) and a target
id (`player+0x16c`).

- **Selection.** A left press picks, in order, ground items (`0x43b650`, state 1,
  acted on at once), other objects (`0x43baf0`, state 4), monsters (`0x43b8c0`,
  state 3), players (`0x43b420`, state 2) and NPCs (`0x43bd30`, state 5).
  Monsters, players and NPCs only **select** when the id changed. A click on the
  target that was already selected sets the state: this is the "double click".
  There is no time window or pointer-distance test between the two clicks.
- **Persistent chase.** States 2 and 3 run every frame. Out of reach the
  character moves straight toward the target's current position (re-aimed every
  frame, movement packets every 250 ms). In reach it turns, stops and sends the
  attack once (flag `0x71d018`/`0x71d01c`, further sends throttled to 250 ms).
  When the target leaves reach the flag clears, the chase resumes and the attack
  is sent again on arrival. The state ends when the target dies or is lost, the
  player moves manually or clicks the ground, or the attack send is refused.
- **Reach** (horizontal centre distance). Bare hands 1.0. Monster: weapon range
  (`item+0x26`) + monster `Size` - 0.3. Player: weapon range + 0.5, and 1.5 for
  a same-faction player outside duels (follow, no attack). A range buff adds
  `DAT_021fe5f2`. GX World accepts weapon + Size + 2.3 plus a 2 unit grace, so
  the native reach is always inside the server's.

GX now follows the same rules: the second click needs no timing, the chase keeps
following a moving target and swings again after it leaves reach, and reach uses
the weapon and monster Size. A same-faction player is followed at 1.5 units with
no packet of any kind (not even the heading), unless they are your duel opponent
or, during a guild duel, belong to another guild. The same applies to the F key.
Players without shape data keep the server's verdict. Still different: the
native range buff is not added, GX World's own WarType P maps (where everyone is
a teammate) are not known to the client, and the native client acts on mouse
press while GX acts on release.
