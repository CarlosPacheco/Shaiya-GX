# Movement speeds (original code)

Units per second, from the EP5.4 client and the EP4.5 server.

| Case | Speed |
| --- | --- |
| Run | 5 plus the move index offset (-2..+2), so 3, 4, 5, 6, 7 (`0x419f60`, `0x43bfa0`) |
| Walk | half of run |
| Strafe and step back | 0.5, 1.0, 1.5, 2.0, 2.5 by move index (`0x43e8ad`, `0x43f2aa`) |
| Mounted forward | 7 (8 for the fast mount variant) |
| Mounted directional step | The on-foot step speed plus 2, so 2.5, 3.0, 3.5, 4.0, 4.5 (`0x43e9f1`, `0x43eeac`, `0x43f341`) |
| Climbing a ladder | 0.8, 1.3, 1.8, 2.3, 2.8 by move index (`0x43aea0`, `0x419f60`) |
| Swimming | no separate speed: the same tables as on land |
| Blocked (index 255) | 0 |
| Server check (`ps_game` table `0x542508`) | 3, 4, 5, 6, 7 by move index; vehicle mode 2 gives 8 |

Monsters (`0x4312a0`): the client walks them toward the newest destination at
`(Step + offset) / (Time in seconds) - 0.3`, with offset -4, -2, 0, +2, +4 by
the move index and the normal or chase Step/Time pair. The speed is constant, so
a hop of random length takes proportionally longer.

Holding S with Q or E selects the lateral step (`0x43f064`). Double-tapping
S uses the backward step tier until ALT 13 becomes active, then uses 2.5
(`0x43f2a1`). PS107 uses the same base step tiers and mounted bonus.

GX applies these in client prediction, the server movement check and the hop
durations the server sends for monsters. By project design, GX uses the full
forward speed for mounted lateral movement. Earlier GX versions swam at a
fixed 2.8, climbed at a fixed 2.5 and gave normal wandering a fixed duration.
