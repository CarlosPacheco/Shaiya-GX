# GX fall damage

The feature is **enabled by default**. Set `RAPTURE_FALL_DAMAGE=false` in the
server environment (or `.local/runtime.env`) to disable it, then restart the
world server. Invalid explicit values disable it as well; disabled landings
never change HP or emit damage notices.

Fall damage is a GX rule, chosen independently of native EP5 damage formulas.
The world server tracks the highest point of an airborne interval and evaluates
its vertical drop once a solid supported landing is reached.

- Drops up to six world units cause no damage; normal jumps remain harmless.
- Above six units, damage is `ceil(maxHP * min(0.5 * ((drop - 6) / 24)^2, 1))`.
- A 12-unit drop takes about 3.125% of maximum HP; 18 takes 12.5%; 24 takes 28.125%.
- At thirty units damage is 50% of maximum HP; around forty units it reaches
  maximum HP and can kill. This halves the previous uncapped damage curve.
- Damage is capped to current HP. Actual damage uses current server stats,
  ends stealth and cancels preparation. GMs with `/turbo` enabled land without taking fall damage.

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
