# Mounts

Click the equipped mount in equipment slot 13, or use a quick slot pointing to
that equipped item. Mounting takes two seconds and uses the ordinary cast bar
and the native Town Portal preparation effect attached to both casting hands. `U` has no mount binding.
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

Install `resources/vehicle/bones.ini` as `data/Vehicle/bones.ini`, beside the
racial MON files. Sections are `Human`, `Elf`, `Vail` and `DeathEater`. Each
entry is `MON index=primary rider bone,secondary rider bone`:

```ini
[Human]
1=25,27

[Elf]
1=32,46
```

An omitted secondary bone, or `-1`, means no second seat. Bindings must
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
to the mount. Driver departure detaches both.
Jump makes the driver and mount jump together; it still detaches the passenger.
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
and `ReqIg`. It also installs the shared bone file and removes the obsolete per-race files after backing them up. Original client assets are
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

## Native presentation and audio

Vehicle MON animation slots 0/1/2 are walk/run/jump, 6 is breathing and 8 is
an occasional idle gesture. Breathing is the resting loop; the gesture has an
11-second wait and a 3-second window and never loops within that window.
Backward movement keeps the mount facing forward and plays walk backwards.
Mounted lateral movement uses the same speed as forward movement. Ground
support, vertical motion and repeated-jump rejection remain server-authoritative.

Native client 0x417cc0 plays `ef_dmg_transf01` and transformation EFT catalog
124, sequence 0, on mount completion. It selects the appearance roar by race:
Human `run_horse-roar01`, Elf/Vail `run_pumav-roar01`, DeathEater
`run_baslk-roar01`. Standing boards omit the animal roar. These cues are
hardcoded in the original, rather than defined in the MON sound slots.
Preparation uses `ef_in_atrbu01` (0x57ace0). Existing mounted players entering
view do not replay their appearance effect or roar.

If these cues are absent from an older loose data installation, import the
five originals from a local original client, converting them to Ogg Vorbis:

```sh
python3 tools/import_mount_audio.py --source /path/to/original-client \
  --data /path/to/data --backup-dir /path/to/new-audio-backup
```

The importer requires ffmpeg and `build/assets` (or Go). Original audio is
external game data and is not distributed in the source repository.
