# Map rules, weather and progression

GX World loads map and experience definitions from SQLite at startup. [GX Studio](gx-studio.md) edits map rules and birth positions. See [server operations](server-operations.md) for backups and restarts.

## Map access and respawn

`PS_GameDefs_MapProperties` stores map type, level limits, faction respawns,
instance configuration and schedules, weather and map curses. Its 68 initial
rows preserve the server definitions migrated from `Map.ini`.
`PS_GameDefs_CharacterStartPositions` stores the birth position of each race.

| Map | Inclusive level range |
| --- | --- |
| 18 | 1–15 |
| 30 | 20–30 |

Zero leaves that side of the level range open. The server checks destination
access during travel and rechecks saved locations on login. A GM can bypass
the range. A player who exceeds the range by leveling is transferred to their
racial birth position.

Respawn uses the faction's configured destination or the nearest eligible
SVMAP respawn group. If neither provides an eligible destination, the character
returns to their racial birth position. Instance schedules remain part of the
map row, while [obelisk rules](obelisks.md) retain their own configuration.

## Weather and map curses

| Column | Meaning |
| --- | --- |
| `WeatherType` | 0 clear, 1 rain, 2 snow |
| `WeatherFrequency` | Minutes between event starts, counted from GX World startup. Zero selects 60 minutes. |
| `MapCurse1`–`MapCurse3` | Zero disables the slot. A skill ID applies its supported level-one effect on entry, persists through death and dispels, and is removed when leaving. |
| `MapModifier1`–`MapModifier3` | Reserved for future server rules. The shipped values are zero. |

The initial catalog schedules weather on ten maps. Each event lasts ten
minutes. GX World sends a map-scoped snapshot on entry and when weather
changes. Compatible clients render PS107 precipitation resources, a gray rain
fog or snow fog, the storm sky and scrolling clouds.

A GM can preview rain with `/weather 1` and snow with `/weather 2` for ten
minutes in their own session. `/weather 0` returns that session to scheduled
weather immediately. See the [GM command list](gm-commands.md).

## Experience curve

`PS_GameDefs_ExpDefs` has one cumulative `Exp` threshold per `Level`, shared by
all characters. The rows start at level 1 and are consecutive. Each threshold
is the cumulative experience needed to reach the following level. The highest
row sets the server cap. The initial cap is **60**, reached at **295,000,000
EXP**. Characters at the cap receive zero experience. A reward that reaches the
cap credits only the amount needed to get there.

Migration 19 takes the highest threshold from the former mode rows at each
level and rounds in bands starting at 100,000 EXP. Existing levels, progress
within a level and unspent points are retained. Restart GX World after editing
the curve.
