# Physical ranged projectiles

Bows, crossbows and javelins now share physical projectile presentation for
confirmed basic attacks and offensive physical ranged skills. This follows
Phoenix's weapon profiles, official model resources and the EP5.4 projectile
reference routines `0x450650`, `0x40e760` and `0x40f0a0`.

- Bows/crossbows use the original `Item/3DO/arrow.3DO` and `Item/DDS/arrow.dds`.
  Javelins reuse the equipped weapon mesh and texture, not an arrow substitute.
- The pending shot belongs to its own source/target/action. Native phase 30 is
  used for bows/crossbows, 15 for javelins, with the recovered phase-rate table
  selected by the published attack-speed index. Bows show a nocked arrow in
  the actual weapon attachment matrix during this preparation.
- Release uses the posed weapon's attachment origin, with a feet-relative
  fallback if no matrix is available. The actual 3DO arrow points down local
  negative Z; the flight matrix keeps its tip facing travel in every quadrant,
  including vertical paths. Javelins apply their extra native row rotations.
- Initial speed is initial distance multiplied by five. It stays fixed while
  the shot homes toward the recipient's chest. New selections, later attacks
  and body-animation changes do not redirect or cancel an already released shot.
  A disappearing recipient leaves the last known destination intact.
- A thrown weapon is hidden only for the matching action, then restored at
  arrival/cancellation. An older shot cannot restore a newer action's weapon.
- Numbers, hit audio/reaction and physical-skill recipient effects wait for
  visual arrival. Caster release effects retain their existing behavior.
  Health/resource results remain server-authoritative and are not delayed or
  recalculated by the client. A visual cancellation never refunds accepted damage.

There are at most 128 pending/flying shots. Their model cache is shared; arrivals,
cancellations and world teardown remove flight state. Physical weapons remain
visible independently of the Skill Effects display toggle. Spell EFT projectiles
continue using their existing separate presentation path.

Validation includes all native/alternate ranged item families, fixed-speed
homing, non-inverted tip directions across quadrants, and real OpenGL shots for
all three weapon families. The graphics test requires visible pixels from the
original arrow/javelin model, verifies arrival after a body-action change and
checks that physical-skill feedback is deferred until impact.
