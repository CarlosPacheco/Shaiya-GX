**Shaiya GX v0.1.11 — Client & Server Update**
Changes since v0.1.10:

• **Combat:** corrected looping Wind Spin combos, Stinger dash/reach, Blizzard effects, prepared-cast range and casts interrupted by idle-position updates.
• **Presentation:** corrected meteor rotation/contact, fixed spawn-effect anchors and corpse positions, restored monster scales and quest-drop indexing. Combat stance now refreshes only on basic attacks; stack counts appear only on stackable items.
• **Monsters:** modern Lua is now the default server AI, with 46 documented policies. Restored classic group respawn timing and retaliation after missed attacks; recharge refusals now show feedback.
• **Persistence:** trades, quest rewards and economy/inventory changes commit atomically before success is reported. Added recovery, rollback, concurrency and disconnect tests.
• **Server:** improved world scheduling, bounded queues, SQLite durability/migrations, rotating logs and diagnostics. Added Argon2id passwords, session revocation, long-password support and TLS for remote connections.
• **Terminal-only GX Server:** the GUI and its icon are gone. Use `gx_console`: `start server` / `shutdown server`.
• **Performance:** reduced repeated animation/bounds work and effect setup while preserving visible animation, reflections, shadows and labels. Gains vary by scene; benchmark CPU timings are not FPS guarantees.
• **Development:** expanded the shared Wiki and protocol reference, added reproducible performance tools and updated native Linux/Windows CI.

**Update client + server together. Back up and keep your runtime database, configuration and private TLS files.**
Download: https://github.com/Spelunkern/Shaiya-GX/releases/tag/gx-v0.1.11
