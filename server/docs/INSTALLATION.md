# GX Server installation

GX Server is now part of the parent Shaiya GX repository. Follow the maintained guides:

- [First run on Windows and Linux](../../docs/quickstart.md): downloads, hashes, account creation and local play.
- [Building and testing](../../docs/building.md): native Windows/Linux toolchains and cross-compilation.
- [Operation](../../docs/server.md): terminal console, LAN, logs, persistence, backup and updates.
- [Portable archives](distribution.md): package layout and reproduction.
- [Development database](sqlite.md): snapshots, backups and mutable runtime storage.

**GX Server** operates through the terminal console. The server ships the requested development database, not an empty production database. SQLite is the primary production database; its embedded operation requires no separate database service.

### Mob navigation in dungeons

Field mobs use the terrain heightfield for movement. Dungeon mobs can use a
separate navigation generation built from the dungeon structure and WLD
blockers. Generate it from loose client data with the standard map builder:

```sh
python3 server/tools/mapbuild/generate.py \
  --data /path/to/client/data --maps all --mob-dungeons-only \
  --output /path/to/server/data/geometry
```

This writes `geometry/mob-dungeons/` alongside the full geometry. Restart
`gx_world` to load it. Published server packages include the pinned companion
catalog. Maps without a companion mesh use the full dungeon geometry for
mobs. Player collision uses the main geometry generation.
