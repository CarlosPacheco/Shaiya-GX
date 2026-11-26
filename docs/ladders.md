# Functional ladders

Native WLD `Objects` (the Ladder section) now supports climbing, in addition to
its existing rendering. Face the rungs and hold **W** to climb or **S** to descend.
Releasing movement holds position. **Space**, lateral strafing or a ground travel
request detaches the character and resumes normal collision and gravity.
Sitting, mounts, death and movement locks prevent attachment. Attack and skill
movement pauses also stop vertical climbing for the existing 800ms interval.

The original `018_ladder` character clip begins immediately, locally and
for nearby players. Its locomotion clock stops while holding. Confirmed
actions can still finish.
The original speed table uses 0.8, 1.3, 1.8, 2.3 and 2.8 world units per
second for movement tiers 0–4. Game.exe faces the climber against the WLD
object's authored forward vector at both ends (`0x443893` and `0x443977`).

Client and server use shared CPU ladder physics. Attachment requires proximity,
facing and a clear collision path on the rung face. The back of the guide
can admit descent from a supported upper landing onto that same face.
Vertical travel is integrated using server
time; movement samples cannot choose height, escape the guide or gain speed by
sending extra packets. Walls and ceilings still block movement. Nearby real
walkable landings permit automatic dismount, including below the model's raised
rails. Narrow landings centered on the guide are supported too; the guide
permits up to 0.55 units above its visual endpoint to reach a slightly raised
authored floor, with the same collision checks. A blocked ascent may move
outward along the guide normal (up to 1.45 units, bounded by climb speed) to
clear a platform lip, while walls and broad ceilings still block it. Position
corrections determine this offset directly, avoiding separate unsynchronized
client/server offset state. Boarding from an upper landing
crosses to the open side for descent.
Where the original collision geometry provides no safe landing, the character
holds on the guide until descending or explicitly detaching.

The client predicts a landing and waits for the server's exit confirmation
before resuming ground input. A short reattachment cooldown prevents queued
heartbeats from recapturing a character who has just dismounted. Missing intent
heartbeats expire after one second; teleport, map entry and death reset climbing.

`GXLadder` (`0x7F0D`) is an additive GX extension, with ordinary length framing:

- Client payload after opcode: guide index `uint16`, mode `uint8`.
- Server payload after opcode: character ID `uint32`, guide index `uint16`, mode
  `uint8`. Notifications reach the owner, current viewers and new viewers.
- Modes: 0 detach, 1 hold, 2 ascend, 3 descend. Invalid lengths/modes are rejected.
- Existing native movement opcodes, coordinates and flag meanings are unchanged.
  Client and server must be updated together.

The small embedded `internal/mapgeometry/native_ladders.json` contains only 71
derived placement facts across 12 original maps, not models or textures. It lets
the server validate the same guides without requiring client assets or rebuilding
the large navigation resources. The client derives guides from its own loose
WLD/Object assets; an integration check compares them with the server catalog.
Regenerate the catalog after an authorized map/asset change:

```sh
go run ./cmd/laddercatalog -data /path/to/client/data
SHAIYA_DATA=/path/to/client/data go test -tags integration ./internal/mapgeometry -run OriginalLadder
```

Unit checks cover ascent/descent, holding, upper and lower landings, facing,
ceilings, remote/invalid requests, packet floods and attack pauses. The disposable
N2 suite includes a client/server climb, hold, descent and upper landing with the
original map-1 guide and assets.
