# Two-player progression and restart journey

This opt-in integration probe follows two newly created normal-mode fighters
through an earned early-game progression sequence. It uses the real world
server entry point in a separate process, encrypted native requests and the
client's `game.State` packet decoder. Its accounts, SQLite database, handoff
files, sockets and server processes are disposable. No personal installation or
runtime is used.

## Run

From the repository root, with the ordinary server seed, map and physics
resources available:

```sh
. tools/go-env.sh
GX_RUN_PARTY_JOURNEY=1 go test -tags integration ./server/cmd/gx_world -run '^TestPartyProgressionJourney$' -count=1 -timeout=15m -v
```

It walks in real time, so it is excluded from routine integration runs unless
explicitly enabled. `TestJourneyWorldProcess` is a subprocess helper and also
skips when invoked ordinarily. For private logs, set `GX_JOURNEY_ARTIFACTS` to
an existing or creatable directory outside versioned source; do not publish the
resulting runtime artifacts.

## Observed run

The Linux run completed in 145.7 seconds: five shared kills, two item pickups,
one shared-gold event, both characters reaching level 3 and learning skill 1/1.
The same committed progression survived a world-process restart. A dungeon mob
killed the second character, and native party resurrection returned it beside
the leader after the 30-second wait. Final SQLite integrity was `ok`.

The position-only portal checkpoint below remains an explicit coverage limit.

## Assertions

1. Create two accounts in the private fixture, then choose faction and create
   level-1 characters through the native world requests. Original starter items,
   attributes and growth rules apply; the probe grants no levels, stats, items,
   money, skill points or quest progress.
2. Form a party, walk to the original quest-1 NPC (8/83), accept the quest for
   both characters and walk to an original reachable mob-2008 spawn.
3. Defeat real mobs together. Each confirmed death advances both quests and
   experience totals. Observe actual item pickup/delivery and equal party gold
   shares. Original random drop rates remain unchanged; at most 30 kills are
   attempted to observe both item loot and shared gold.
4. Walk back and turn in quest 1 at NPC 8/87. Both earn levels and unspent stat
   and skill points. Learn an eligible skill from the real definitions and check
   its persisted skill-point debit. The application normally completes some
   content-dependent client bookkeeping; the headless probe checks durable quest
   and skill records rather than inventing those fields in the packet decoder.
5. Walk to gatekeeper 2/32 and use its original free destination 1. Then apply
   the **position-only fixture checkpoint** described below. Form the party
   again and enter the original map-1 portal to dungeon map 5.
6. Snapshot the committed level, experience, gold, six base attributes, unspent
   points, item rows/quantities/metadata, learned skills, finished quests and
   dungeon map and X/Z position. Close the test sockets, stop the world process and start a
   new world process against the same database, without reseeding it.
7. Reconnect both characters and compare stored and client-visible progression.
   Re-form the party: membership is session state, so this test does not require
   a party to survive a server restart.
8. Approach a real dungeon combat mob. If the approach itself is lethal, count
   that real death; otherwise provoke it, stop attacking after the first response
   and let it kill the second player. Request the native party resurrection,
   wait its original 30 seconds and verify that the player returns alive beside
   the living leader, retaining the completed quest.
9. Log out both characters normally and require SQLite `integrity_check=ok`.

## Explicit route limitation

**The complete overland walk from the gatekeeper to the dungeon entrance is
not validated.** The probe's mob-navigation path query did not connect that
approach. This alone does not establish that a human-controlled player cannot
walk the route; mob navigation is not the player's movement controller.

After both players log out, the fixture changes only their map/position to a
point in front of the original portal. It verifies that experience, level,
gold, inventory, learned skills and completed quests remain unchanged, reconnects
and then tests the actual portal transition. This checkpoint is printed
explicitly in the test log. It must not be represented as proof of an uninterrupted
walk from character creation to the dungeon.

Map 18 is a field/PvP zone in the original Map.ini, and is not counted as dungeon
coverage. Map 5 is explicitly `MapType=D` and has a usable level range for these
new characters.

## Other limits

This is headless protocol/gameplay validation, not a rendered UI, camera,
animation, sound or manual-play test. Authentication is supplied by disposable
login handoffs; the full login service is covered separately by N2. The test
covers earned early levels, not a level-1-to-60 playthrough, broad class balance,
a long soak or maximum player capacity. Item pickup is exercised in the party's
default mode; this is not coverage of every loot policy or raid distribution mode. It does not kill the server inside an
uncommitted gameplay transaction; transaction rollback and forced-writer tests
are separate checks in the [persistence audit](persistence-audit.md).
