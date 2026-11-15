# Mounts

Click the equipped mount in equipment slot 13, or use a quick slot pointing to
that equipped item. Mounting takes two seconds and uses the ordinary cast bar
and the native Town Portal preparation effect. `U` has no mount binding.
Movement, damage and other actions cancel preparation. Ordinary damage does
not dismount an established rider; death, hard crowd control, map changes and
removing the equipped mount do. The object's `ReqIg` cooldown, in seconds,
starts when preparation completes and survives dismounting and reconnecting
within the running world service.

Type 42 items reuse existing definition fields in both Item.SData and server
Items definitions:

| Field | Meaning |
| --- | --- |
| `Range` | Direct, zero-based record index in the driver's racial Vehicle MON |
| `AttackTime` | Seat count, 1 or 2 |
| `Speed` | Absolute mounted movement tier: 4 = 7 units/s, 5 = 8 units/s |
| `ReqIg` | Item cooldown in seconds, retained from original content |

Level 40–45 mounts use tier 4; level 50+ use tier 5. Earlier native mounts keep
tier 4. Mount metadata does not contribute to movement or attack speed on foot.
Characters remain capped at movement tier 4 regardless of ordinary bonuses.

## Bones and effects

Install the four source-controlled `resources/vehicle/<race>/bones.ini` files
under `data/Vehicle/<race>/bones.ini`. Race folders are `Human`, `Elf`, `Vail`
and `DeathEater`. Each section is a racial MON record index:

```ini
[1]
rider=25
passenger=27
```

An omitted `passenger`, or `passenger=-1`, means no second seat. Bindings must
refer to valid skeleton bones. The initial tables cover all eight available
native models per race, using Phoenix's verified seat bindings. Phoenix's
global deduplicated vehicle indices must not be used as racial MON indices.
Seats follow the animated skeleton pose, shared with rendering. Vehicle MON
bone effects use the same preparation, particle settings and rendering path
as monster MON bone effects.

## Two riders

Right-click an allied player and select the native **go together** action.
Either the driver or the prospective passenger can request it. The recipient
gets the native confirmation prompt. A two-seat driver must have a free seat;
the passenger must be on foot. Invitations expire after 30 seconds. The server
checks the same map, faction, life state and a 3-unit three-dimensional range
both when requesting and accepting. Acceptance cannot move a distant passenger
to the mount. Jump detaches the passenger. Driver departure detaches both.
The server owns their relationship and gameplay position; only the driver
controls movement.

Native request opcodes 0x0216/0x0217 and 0x021C–0x021E remain in use. GX sends
explicit state in 0x7F10 so MON index zero, seat count, speed and passenger links
are independent of the native appearance byte. This requires matching GX
client and server versions.

## Updating original content

Stop the installation before migrating its definitions. The offline tool backs
up the named databases, client table and existing bindings; it changes only the
three Type 42 metadata fields and preserves accounts, characters, item instances
and `ReqIg`. It also installs the four bone files. Original client assets are
not included in the repository.

```sh
python3 tools/update_mount_content.py --data /path/to/data \
  --database /path/to/rapture.sqlite --backup-dir /path/to/new-backup-directory
```

Use `--database` repeatedly to update multiple explicit databases. The tool
validates client/server correspondence before writing and supports already
migrated content. Do not substitute a personal runtime database for the public
seed. The distributed seed definitions are updated independently.

Validation: `tools/check.sh`; `python3 -m unittest discover -s tools -p
 'test_update_mount_content.py'`; `python3 tools/n2_mounts.py` with local original
assets and a working Linux graphics session. The latter creates disposable
accounts and a database copy, checks two-client invitation/secondary-seat/jump
behavior, and loads all 32 race/model combinations with real OpenGL.
