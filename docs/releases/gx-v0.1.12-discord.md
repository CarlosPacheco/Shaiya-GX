**Shaiya GX v0.1.12 — Teleport Hotfix**

• Fixed the blocked commands/disconnection after repeated map changes, reported with `/cmove`. The destination appeared, but the client was not acknowledging completion to the server.
• Corrected the shared map-transfer acknowledgement for the graphical client, CLI and protocol fixtures.
• Verified seven consecutive map commands, including a same-map teleport, with a gameplay command after every transfer and clean logout.

**For v0.1.11 users: no new game data or database migration is required. Keep your runtime DB and configuration.** Matching Linux/Windows client and terminal-only server packages are included, along with the server seed, maps, physics and Lua AI.

Download: https://github.com/Spelunkern/Shaiya-GX/releases/tag/gx-v0.1.12
