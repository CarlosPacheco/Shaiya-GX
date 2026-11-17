# GX Server installation

GX Server is now part of the parent Shaiya GX repository. Follow the maintained guides:

- [First run on Windows and Linux](../../docs/quickstart.md): downloads, hashes, account creation and local play.
- [Building and testing](../../docs/building.md): native Windows/Linux toolchains and cross-compilation.
- [Operation](../../docs/server.md): terminal console, LAN, logs, persistence, backup and updates.
- [Portable archives](distribution.md): package layout and reproduction.
- [Development database](sqlite.md): snapshots, backups and mutable runtime storage.

**GX Server** operates through the terminal console. The server ships the requested development database, not an empty production database. SQLite is the primary production database; its embedded operation requires no separate database service.
