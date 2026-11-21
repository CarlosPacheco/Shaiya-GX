# User Options: Other Setup

The **User Option** page (`O` → User) has three groups: Control Setup,
Name/Chat Setup and **Other Setup**. This page documents the nine checkboxes of
the original Other Setup (GX keeps seven): what the original EP5.4 client does with each (recovered from
`Game.exe`) and how GX implements it. The settings are saved with the rest of the
client settings and apply when OK is pressed. **Reset** on the page restores the
original defaults.

## The nine options

| Checkbox | `[USER]` key | Global | Default | What it does |
|---|---|---|---|---|
| All Windows | `SHOW_INTERFACE` | `0x719938` / `0x7198c4` | TRUE | The same switch F11 flips: every window and HUD element is drawn only while it is set. Unchecking hides the whole interface; the choice is saved. |
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


## GX behaviour

The page art (`option_user.png`) was edited so the remaining seven checkboxes fill the
grid in the original order, row by row: Reject Battle, Reject Party, Warning Msg.,
Reject Trade, Shake Screen, Word Balloon, Hide Damage.

| Option | In GX |
|---|---|
| All Windows | F11 toggles the interface in GX. |
| Reject Battle | An incoming duel request is declined at once with the "no" answer; no prompt. |
| Reject Party | Party and raid invitations are declined at once; no prompt. |
| Reject Trade | An incoming trade request is declined at once, silently. |
| Warning Msg. | Learning a skill asks SysMsg 10086 first (OK / Cancel); off sends the request at once. |
| Re-open UI | This checkbox is omitted in GX. |
| Shake Screen | Enables the camera shake described below. |
| Word Balloon | Off hides the speech balloons above characters. |
| Hide Damage | Off by default; on hides every floating combat number. |

Defaults are the original ones: Reject Battle, Reject Party, Reject Trade and
Hide Damage off; Warning Msg., Shake Screen and Word Balloon on. Existing settings files keep working: keys
that are missing take the defaults.

## Camera shake

Shake Screen gates every camera shake of the client. The camera keeps one shake
timer; a trigger only raises it to the longer of the old and new value. While it
runs, each frame draws one random offset in [-0.2, 0.2] per axis and adds it to the
wanted eye and target together. The drawn camera eases toward the wanted one
(`0x550090`, 10 times the frame time per frame), so the view only shakes: it is a
smoothed translation, never a rotation or a distortion. With the option off the timer still runs and no offset is added.
There is no distance check.

| Source | Duration |
|---|---|
| Each falling shard of a shower touching the ground: Hailstorm, Meteor, Blizzard, Shower Time, Bloody Shower | 0.5 s per impact |
| Earthquake, when its result is shown | 2.0 s |
| Earth Spikes, when its result is shown | 0.5 s |

During the five seconds of Blizzard each landing shard renews a 0.5 s shake, which
reads as a continuous tremor.

## Sound Options

Risk Sensor controls the original battle music that plays while nearby enemies
activate the warning. The minimap warning remains visible with the music off.
Character Sound controls nearby player voices for attacks, hits and deaths.
Weapon, spell and interface sounds use their usual audio settings.
NPC Sound is reserved and currently has no effect.

## Evidence

Client settings are read by `0x408bf0` and written by `0x4f9000`; the window copy
is `0x4f8c40`. The consumers are the auto-decline paths `0x562a7e` (battle),
`0x572ce0`, `0x572e40`, `0x573160`, `0x5731f0` (party and raid), `0x56c920`
(trade), the skill window `0x502682` (Warning Msg.), the balloon update `0x411e10`,
the number loop in `0x447ef0`, and the camera `0x54fe20`/`0x54fe70` with the
triggers in `0x418030`, `0x431da0`, `0x4338f0`, `0x430a80` and `0x577020`.
Behaviour of Warning Msg. and Shake Screen was derived from the code, not observed
in the original client.
