# Movement speeds (original code)

Units per second, from the EP5.4 client and the EP4.5 server.

| Case | Speed |
| --- | --- |
| Run | 5 plus the move index offset (-2..+2), so 3, 4, 5, 6, 7 (`0x419f60`, `0x43bfa0`) |
| Walk | half of run |
| Strafe and step back | 0.5, 1.0, 1.5, 2.0, 2.5 by move index (`0x43e8ad`, `0x43f2aa`) |
| Mounted | 7 (8 for the fast mount variant); stepping back 2 |
| Climbing a ladder | 0.8, 1.3, 1.8, 2.3, 2.8 by move index (`0x43aea0`, `0x419f60`) |
| Swimming | no separate speed: the same tables as on land |
| Blocked (index 255) | 0 |
| Server check (`ps_game` table `0x542508`) | 3, 4, 5, 6, 7 by move index; vehicle mode 2 gives 8 |

Monsters (`0x4312a0`): the client walks them toward the newest destination at
`(Step + offset) / (Time in seconds) - 0.3`, with offset -4, -2, 0, +2, +4 by
the move index and the normal or chase Step/Time pair. The speed is constant, so
a hop of random length takes proportionally longer.

GX applies these in the client prediction, in the server movement check and in
the hop durations the server sends for monsters. Earlier GX versions swam at a
fixed 2.8, climbed at a fixed 2.5 and gave normal wandering a fixed duration.
