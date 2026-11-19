# Guild house, guild NPCs and etin: original behaviour

Evidence for the 1:1 port. Server rules come from EP4.5 `ps_game.exe` (PDB names,
Eternity's Ghidra project, `tools/ghidra/env.sh` → `ghidra_run_server`); wire
layouts and UI come from the EP5.4 `Game.exe` (`ghidra_run`). Private decompiles
live in `work/guild-house/` (not in Git).

## Guild state (`CGuild` offsets)

| Offset | Field | Notes |
|---|---|---|
| 0x40 | country | clamped to 1 when used as a table index |
| 0x44 | points | ranking result |
| 0x48 | rank | 1-30 ranked, 31 unranked |
| 0x4c | etin | stored guild etin |
| 0x50 | keep etin | upkeep, `CalculateKeepEtin` 0x430a50 |
| 0x54 | UseHouse | 0 off, 1 active, 2 suspended (etin < keep at settlement) |
| 0x55 | BuyHouse | set by 0x0D34 |
| 0x47c.. | 8 NPC slots | `{type u16, level u16}` each |
| 0x98/0x9c/0xa0 | ranking points / etin return count / entries | reset every ranking |

`EnableGuildHouse` 0x42f0c0: `rank < 31 && UseHouse == 1 && BuyHouse != 0`.
`IsGuildHouseMap`: `CreateType == 6` (`H`). `CWorld::GetZoneByMapID` 0x413df0
case 5/6 creates the instance only when `EnableGuildHouse` holds.

## Wire (all little-endian, after the 2-byte opcode)

| Opcode | Dir | Body | Source |
|---|---|---|---|
| 0x0D23 | C→S | empty: answer with etin | `PacketGuild` 0x4764a0 |
| 0x0D23 | S→C | u32 etin (also broadcast after upgrades and settlement) | `SendEtin` 0x430eb0 |
| 0x0D24 | S→C | u8 count, count × {u16 type, u8 level, u8 slot}; omitted if empty | `SendNpcLvList` 0x42f0e0, sent at join/login |
| 0x0D25 | C→S | u16 type, u8 level (guild rank ≤ 2 only) | `NpcUpgrade` 0x430640 |
| 0x0D25 | S→C | u8 result, u16 type, u8 level, u8 slot | same |
| 0x0D33 | S→C | u32 keep etin (no EP5.4 reader) | `SendHouseKeepEtin` |
| 0x0D34 | C→S | u32 ignored: buy the house (master only) | `PacketGuild` |
| 0x0D34 | S→C | u8 result, u32 money | same |
| 0x0D35 | C→S | u32 ignored: donate every etin item in the bags | `PacketGuild` |
| 0x0D35 | S→C | u8 count, count × {u8 bag, u8 slot} consumed | same |
| 0x0D36 | S→C | u8 | `GuildPvPWin` |
| 0x0D38 | S→C | u32 | EP5.4 reader only |

EP5.4 receive reducers: 0x0D24 `FUN_005977d0`, 0x0D25 `FUN_00597940`,
0x0D34 `FUN_005978f0`, 0x0D35 `FUN_00597780`; senders `FUN_0059c730` (0x0D35),
`FUN_0059c760` (0x0D34), `FUN_0059c790` (0x0D23), `FUN_0059c7c0` (0x0D25).

### 0x0D34 buy house (0x477211)

Result 3: no guild or rank ≠ 1. 1: already bought. 2: money below
`HouseBuyMoney`. 0: `BuyHouse = 1`, money −= price, `KeepEtin = HouseKeepEtin`,
reply with the new money, DB 0x0B10 `{guild, keep}`, game log 0x0B08.

### 0x0D25 NPC upgrade (`NpcUpgrade`)

Rejected with 3 when the guild is unranked, `UseHouse != 1`, `BuyHouse == 0`,
level 0 or > 5, unknown type, or level > 3 for types 2/0x101/0x201/0x401.
Rank limits (result 4): type 3 needs rank ≤ 14, type 0x401 rank ≤ 6, types 6 and
0x301 rank ≤ 30. `etin - keep` below 1 or below the level's cost: result 1. An
existing NPC only accepts `level == current + 1` (result 2); a new one must be
level 1 in the first free slot. Then: slot written, 0x0D25 result 0 broadcast to
the guild, keep etin recalculated (default + upkeep of every slot), etin −= cost,
0x0D23 to the caller, DB 0x0B0F, game log 0x0B09.

NPC types: 2 gatekeeper, 0x101 weapon shop, 0x201 defense shop, 0x301 general
shop, 3 blacksmith, 0x401 potion shop, 6 warehouse (bank).

### 0x0D35 donation

Walks bags 1-9 slots 0-23; an item counts only when its craft name starts with
`+` and its Type is 52/53/54/55 (×1/×10/×100/×1000 per count). Each is deleted
(DB 0x0702, game log 0x0415). The total is added to etin, the guild's return
count is incremented, `SendEtin`, DB 0x0B12 `{guild, total, amount, char,
returnCount}`, game log 0x0B0A.

### Ranking settlement (`UpdateGuildPointRank` 0x432b70)

Ranked guilds with `BuyHouse == 1`: `etin < keep` → `UseHouse = 2`; otherwise
`etin −= keep`, `UseHouse = 1`, 0x0D23 to the guild. Unranked guilds:
`UseHouse = 0`, rank 31.

## `GuildHouse.ini` (`InitGuildHouseIniFile` 0x4319a0)

`[DefaultSet] HouseBuyMoney, HouseKeepEtin`; `[GateKeeper|ShopWeapon|ShopDefen|
General|Smith|Bank|ShopPosion] UpgradeEtin/KeepEtin` tables; `[ShopDefault]
BuyDecreRate, BuyPotionDecreRate`; `[Smith] RepairDecreRate, RapiceMixDecreRate,
RapiceMixPercentRate`; `[Bank] PermitSlot`; `[NpcLevel_Light|Dark]
GateKeeperN, ShopWeaponN, ShopDefenN, ShopGeneralN` (NPC type ids per level).
Tokens split on ` ,=()[]{}` and missing tokens keep their default.

Consumers: `PacketShop` (buy rate, NPC level too low → result 2),
`PacketMaininterface` 0x020C (gate rate), `ItemRepair` (repair rate, guild
rank ≤ 14), `ItemGemAdd`/`ItemGemRemove*` (rapice mix cost and success bonus,
only with the house enabled), the bank (`PermitSlot` by NPC level).

## Client (EP5.4)

- `data/Npc/GuildHouse.SData` (`FUN_004cc920`): u32 version, u32 house price,
  u32 default keep etin, 36 × 15-byte service records `{u8 discount%, u8 level,
  u8, u8, u8 rank limit, u16 icon, u16 text id, u16 upgrade etin, u16 keep etin,
  u16 type}`, then four 2×3 u32 tables of NPC ids (gatekeeper, weapon, defense,
  general; light then dark). The client ships built-in defaults for all 36 and
  overrides them from the file (feature 0x37 doubles several defaults).
- Guild manager window `FUN_004820e0` (mode at +0x2c: 0 list, 3 services/upgrade,
  4 buy house); panels `FUN_004815c0` (list), `FUN_00481950` (services),
  `FUN_00481500` (buy house). Assets: `guild_manager_*`, `guild_point_alarm`,
  `guild_manage_button`, `npcarrow`.
