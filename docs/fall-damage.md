# GX fall damage

The feature is implemented and tested, but **disabled by default**. Enable it
explicitly with `RAPTURE_FALL_DAMAGE=true` in the server environment (or
`.local/runtime.env`) and restart the world server. Missing, false or invalid
values keep it inactive; disabled landings never change HP or emit damage notices.

Fall damage is a GX rule, chosen independently of native EP5 damage formulas.
The world server tracks the highest point of an airborne interval and evaluates
its vertical drop once a solid supported landing is reached.

- Drops up to six world units cause no damage; normal jumps remain harmless.
- Above six units, damage is `ceil(maxHP * min((drop - 6) / 24, 1)^2)`.
- A 12-unit drop takes about 6.25% of maximum HP; 18 takes 25%; 24 takes 56.25%.
- At thirty units the calculated damage reaches maximum HP and can kill.
- Damage is capped to current HP. Actual damage uses current server stats,
  ends stealth and cancels preparation. Existing GM immunity remains effective.

Entry/teleport initialization does not carry an earlier fall into the new
position. Falls are armed only after server-observed support; a character
initially loading above ground does not receive spawn-settling damage. Entering
deep water clears the tracked drop instead of treating swimming as a hard
landing. Landings clear their interval before applying damage, so repeated
stationary updates cannot charge it twice.

The server derives height, support and elapsed time from its own physics. It
accepts no client-reported fall distance or damage amount. Native `0x0521`
updates health; additive server notification `GXFallDamage` (`0x7F0C`) carries
target ID and actual damage, both uint32. It uses ordinary length framing and
has no client request counterpart. The client shows the native SysMsg **5**
with the confirmed amount and its normal floating-number presentation.

A lethal fall uses environmental killer ID zero, stops combat and removes the
usual death-sensitive buffs. It grants no PvP credit or inventory drops; its
respawn is marked loss-free instead of charging XP or Ultimate deletion.
Client and server must be updated together for matching notices and respawn
presentation. The loose asset pack and distributed seed database are unchanged.
