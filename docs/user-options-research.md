# User Option, "Other Setup": what the original does

EP5.4 `Game.exe`. The User Option page (`option_user.tga`) has three groups:
Control Setup, Name/Chat Setup and **Other Setup** (nine checkboxes). All are stored
in the client's own INI under `[USER]` as `TRUE`/`FALSE`: loaded by `0x408bf0`,
written by `0x4f9000` when OK is pressed, copied to the window by `0x4f8c40`
(page 3). Defaults apply when a key is missing.

| Checkbox | `[USER]` key | Global | Default | What it does |
|---|---|---|---|---|
| All Windows | `SHOW_INTERFACE` | `0x719938`, applied to the live flag `0x7198c4` on OK | TRUE | The same switch F11 flips: every window and HUD element is drawn only while it is set. Unchecking hides the whole interface; the choice is saved. |
| Reject Battle | `REJECT_FIGHT` | `0x719940` | FALSE | An incoming battle (duel) request (`0x562a7e`) is declined at once with `0x2402 {0}` and no prompt. It is also declined while the character is busy (`+0x2fc`). |
| Reject Party | `REJECT_PARTY` | `0x719944` | FALSE | Incoming party invitations (`0x572ce0`, `0x572e40`, prompt state 4) and union/raid invitations (`0x573160`, `0x5731f0`, state 0x19) are declined at once with the "no" answer and no prompt or sound. Same auto-decline while busy. |
| Warning Msg. | `WARNING_MSG` | `0x719948` | TRUE | On: spending a skill point to learn a skill first asks SysMsg 10086 "do you want to spend the point? (unable to restore it once it's been spent.)" (`0x502682`). Off: the learn request (`0x0209`, id, level) is sent at once. |
| Reject Trade | `REJECT_TRADE` | `0x71994c` | FALSE | An incoming exchange request (`0x56c920`, prompt state 5) is declined at once, also while busy or with another dialog open (`0x21fe735`). Silent. |
| Re-open UI | `UI_REOPEN` | `0x719950` | TRUE | Only loaded, shown and saved. Nothing in the executable reads it, so it has no effect in EP5.4. |
| Shake Screen | `SCREEN_SWING` | `0x719954` | TRUE | Gates **every** camera shake (see the next section): the shake timer still runs, but `0x54fe70` only adds the random offset while the option is on. |
| Word Balloon On/Off | `TALK_BALLOON` | `0x719958` | TRUE | The speech balloon above a character is updated and drawn only while set (`0x411e10`). Off: no balloons. |
| Hide Damage | `HIDE_DAMAGE` | `0x71995c` | FALSE | The floating combat numbers (damage, misses, heals) are not drawn (`0x44ae0e`, loop over the list at `+0x13d864`). |

The Control and Name/Chat groups keep `USE_MOUSE`, `INV_MOUSE`, `INV_MOUSE_UPDOWN`,
`MOUSE_SENSITIVITY` (default 3), `MY_NAME`, `USER_NAME`, `MONSTER_NAME` and
`USE_FILTER` in the same section.

## Camera shake: sources and mechanics

The camera object (`0x220e060`) keeps one shake timer (`+0x1f4`). A trigger,
`0x54fe20(seconds)`, only raises it to the larger of the old and new value. Each
frame `0x54fe70` lowers the timer by the frame time and, while it is above zero
and Shake Screen is on, adds an independent random offset in [-0.2, 0.2] on
each axis to both the eye and the target. With the option off the offsets are
zero. There is no distance check: a trigger shakes the local camera wherever it
happens.

| Source | Duration | Where |
|---|---|---|
| Critical hit in which the player is attacker or target (player skills and basic attacks) | 0.2 s | `0x418030`, result kind 1 |
| Monster critical hit on the player | 0.2 s | `0x431da0`, `0x4338f0` |
| Each falling shard of a **shower** touching the ground (`0x430a80`): Hailstorm 53, Meteor 68, Blizzard 71, Shower Time 162, Bloody Shower 163 | 0.5 s per impact | the five shards per second of the shower controller `CMeteor` |
| Earthquake (skill 69) when its result is shown | 2.0 s | `0x577020` |
| Earth Spikes (skill 83) when its result is shown | 0.5 s | `0x577020` |

So Blizzard's shake is the impact of its falling shards: for the five seconds of
the shower each landing renews a 0.5 s shake, which feels like a continuous
tremor. Turning Shake Screen off removes all of the above at once.

## GX today

Seven are implemented (see [User Options](wiki/user-options.md)). All Windows and
Re-open UI had no effect in the original and were removed from the page.
