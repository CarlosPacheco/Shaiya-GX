# Original parity

GX Server follows the original EP4.5 server. The rules come from
`ps_game.exe` (shipped with its PDB, so functions keep their names), read
through the [Eternity](https://github.com/Spelunkern/Eternity) reconstruction
or directly in Ghidra. Packet layouts come from the 5.4 client's readers
(shaiya-client-decomp), because Eternity targets the EP4 client and some
fields are narrower there.

Status: **done** matches the original; **partial** follows it with the noted
gaps; **invented** is still a GX Server rule to replace.

| System | Original | Status |
|---|---|---|
| Start position, rebirth points | Original `Map.ini`, `CMap::LoadGameMapConfig` 0x004270a0 | done: SQLite map and racial start definitions, with faction respawn and birth fallback |
| Character creation | `CUser::ProcessCharacterCreate` 0x00480e90, `CGameData::LoadBaseItem` 0x004bf800 | done |
| Base HP/MP/SP | `CUser::InitEquipment` 0x00454560, `StatusDefs` | done |
| Attack, defense, accuracy, evasion, critical | `CUser::SetAttack` 0x00454280, `CUser::SetSkillAbility` 0x00485e00, `CUser::AddSkillPassive` 0x00489a80 | done: buff abilities 1-9, 13-18, 20-31; passives detail 1 (weapon mastery, with the original running-total quirk), 4, 8, 9 (cross road), 10 (shield); gem reach/speed; armour (slot 1) element. Passive details 2 and 5 (Backpack Upgrade, Interpretation) are discarded. Passive detail 6 (the five Potential skills) is a GX implementation: below the skill's LimitHP percent it applies its abilities as a 30 second buff (KeepTime is zero in the data) and starts a 180 second reuse (ResetTime is zero in the data), announced to the client with GXSkillCooldown (0x7F16); it cannot be cast. Detail 84 (Fire Thorn, TypeEffect 4) pulses every 2 s while the buff lasts: each hostile mob and user within ApplyRange of the caster (measured on the ground plane like the original area loops, plus a lenient height check of 4 units) takes the skill's DamageHP as fixed damage (no hit roll, stats, defense or absorption; mobs also gain aggro), announced with 0x0607 (mobs) or 0x050F (users); the client plays on each target the element damage effect of the skill, DA_target_Adamage sequence by AttribType (0x577b40; fire is sequence 2). Ability 19 is GX's cooldown reduction (seconds): applied, it takes that many seconds off every spell already cooling down at once and off the reuse of spells cast while it lasts (a reduction covering a spell's reuse leaves it with none); Potentials keep their fixed reuse; the server announces changed timers with GXSkillCooldown (skill, remaining, total) so bars and timers stay coherent |
| Level-up | stat/skill points by mode, +1 main stat | done |
| Exp per kill | `CMob::GetExp` 0x00491690, `award_experience` | done: x1.5 with exp doubling (ability 35), +0.2 with a charm (40-42), quest rewards skip combat modifiers. GX reads a common EXP curve and cap from SQLite. At the cap, rewards credit zero EXP. |
| Recovery | `CUser::UpdateRecover` 0x0048cfb0; `InitEquipment`, `GemEquipmentAdd`, abilities 10–12 and KC tiers | done: previous tick mode selects sitting (5% plus skill percent, flat 28 and recovery gems), standing (flat 4 and recovery gems), or combat (KC modifiers); blessing recovery uses current maximum pools. Dead actors do not recover. |
| Lapisia enchant | Original target/material fields and Craftname encoding retained | GX policy: SQLite attack and defense enchant tables with independent caps derived from consecutive levels (default +20), native ReqIg gold costs, material count does not change chance, no equipment destruction. See [feature wiki](../../docs/wiki/blacksmith.md). |
| Craft bonuses | `CUser::ItemEquipmentAdd`, `CItem::CopyCraftExpassion` 0x004ba670 | done: Craftname stats only on items with Server set and ReqWis 1-99; enchant up to 50 adds attack, above 50 absorption; equipped Items.Exp and active linked lapis Exp absorb damage |
| Gems | Native capacity, target flags and result layouts retained | GX policy: native ReqIg costs without Vg/level terms, additive linking hammers up to100%, selective extraction with direct hammer ReqVg probability, equipment always preserved. See [feature wiki](../../docs/wiki/blacksmith.md). |
| Hit chance | `CUser::AttackSuccess` 0x0044d010 / `CMob::AttackSuccess` 0x004a23f0 | done: skills check the mob's ResistState/ResistSkill and SuccessType 1-3; the magic term reads WIS on both sides (CUser +0x122c, stMOBInfo +0x3a), SuccessType 3 reads LUC (Eternity labels these DEX). Defender blocks (buff 85 by count, 86 by chance, 92 all) and mirror reflection (buff 83, `CUser::ChkMirror` 0x0044e080: SuccessType 1 SuccessValue%, 2 LUC x 10 / value, 3 WIS x 10 / value; 0x051B back to the mob) |
| Damage | `CUser::AttackDamage` 0x0044d540, `CMob::AttackDamage` 0x004a2920 / 0x004a3070, element table `CGameData::m_AttackAttr` 0x00542260 | done: skill DamageType 0-5, skill AttrType element, detail 24 ignores the defender element, bow x0.7 within 3 units unless cross road, mob normal attacks use element row 0 against the armour element; the random terms and the mob critical read LUC (+0x1234 / stMOBInfo +0x3c, Eternity labels them WIS) |
| Mob stats and attacks | `Mobs` Dex/Wis/Luc/Defense/Magic/Attrib/Size, `CMob::AttackAI` 0x004a1ab0 | done: slots 3 to 1 by Attackok (3: HP 20-40%, 4: HP <= 5%, 5: HP < 20%), per-slot cooldowns, AttackTypeN is a mob skill (Skills level 100): attack (0x060D first hit, 0x060B area/repeat, 0x060C drain, 0x060E AddDamage), self heal; areas 6/7 by ApplyRange. Their debuff skills stay on the player (below). Attackok 6 turns to the weakest attacker of another country (FindAttackUserMinHP) |
| Mob movement and targeting | `CMob::UpdateIdle` 0x004a7e60, `FindTarget` 0x0049fd20, `Chase` 0x004a6de0, `ReturnHome`, `ReTargetByAggro` 0x00490ab0, `UpdateRecover` 0x004a8210 | done: wander within 10 of the spawn point (NormalTime/NormalStep, speed offsets); only AI 1/3 look for targets, 5 s after spawning, the first living user of another country (Attackattrib3) within ChaseRange over the 3x3 cells; aggro ledger per hit = the attacker's REC x 10 plus the skill's Arrow (`CMob::Damage` 0x00491... via `CUser::AttackMob`; damage dealt does not count), the attack turns to its top (`CDamageMob::GetMaxAggro` 0x004b9000 compares the aggro field only); Taunt (buff 4/69, `CUser::TauntMob` 0x004877f0 from `AddApplySkillBuff`): every living mob within ApplyRange (strict) turns to the caster (`SetTarget`), its ledger gains Arrow + REC x 10 with no damage, then `ReTargetByAggro`; leash 45 from home (walk home) or from the mob (idle); chase ChaseStep every ChaseTime; return completes within 10 of home and clears effects and aggro without refilling HP; recovery Attack3 (nRecoverValue) every 5 s. No pathfinding around walls (blocked lines walk home) |
| Mob respawn | `CZone::UpdateMobDeath` 0x00421a80, `CZone::UpdateMobRespawn` 0x0041d240, `CGameData::LoadConstMob` respawn table | done: one group per svmap area entry; the first corpse removal (after the 5 s corpse lifetime) of a full group starts its timer (Attackspecial3: 5 s, 30 s, 1, 3, 5, 10, 15, 30 min, 1, 4, 12, 24 h, 3 days, 2 h, immediate, 7 days; +-60 min from selector 9); each expiry brings one mob back at a random walkable spot of the area and restarts the timer |
| Drops | `MobItems` orders 1..9 (1-6 per cent, 7-9 per million), grade pick with `Drop` limits, original `SequenceItemCreate.ini`, 30% gold `Money1 + rand(Money2)`, `CMob::GetDropPosRand` 0x00490c70 | done: positions within half a unit with Y 0 (the client places them on the ground). GX replaces the sequence rule with independent World Drops chances by grade and mob level |
| Skills | `CGameData::LoadConstSkill` 0x004bffe0, `CUser::SkillAttackSelect` / `SkillAttackRange` / `SkillAttackRangeSelect`, `CUser::AttackMob(skill)` 0x0044fd90, `CUser::HealUser` 0x00450670 | done: load-time normalization (kind = clamp(TypeAttack,1,3)-1, AttrType, StateType, MultiAttack, FixRange hours, detail 93/122), dispatch by TypeEffect, details 21-27 (22 multi-hit, 23 sacrifice, 27 drain with 0x0518; detail 25, an attack with extra HP/MP/SP damage, was folded into 21 by migration 7 since AddDamage comes from its own fields), area announce 0x0511, AddDamage 0x051A, cost reductions (cost 1 spends the whole pool), heal + 2 x WIS, buffs 82/91/94/95, ClearApplySkillByDeath. Item skills (`CUser::UseItemSkill`: Range/AttackTime name the skill; candies, remedies, charms) |
| Skills on mobs | `CMob::AddApplySkill` 0x004a0aa0, `RemApplySkill`, `ChkKeepSkill` (every 3 s), `ChkEndTimeSkill` | done: 61 damage over time (TimeDamageType 0/3/4/5 x elapsed/3000), 62/67 ability loss, 63 whole-HP hit on expiry, 64 stun, 65 stop, 66 sleep (woken by damage), 68 no element, 72 target fixed on the caster; speed changes as 0x0609, damage as 0x0607. Selected-target effects use native 0x030B/C/D layouts: reader 0x5667a0 and target HUD 0x50b6a0, now recovered and implemented; GX subscribes with 0x7F0F |
| Skills on players | `CUser::AddApplySkill` 0x00484850, `AddApplySkillDebuff` 0x004855b0, `ChkKeepSkill` 0x00487190, `TimeDamage` 0x00486290 | done: 0x050D (id, skill, level: 9 bytes), 0x050E, 61 damage over time (0x050F), 62/67, 63, 64/66 block actions and 65 movement (0x051C speeds 255), 66 woken by damage, 68, abilities 32-34 disable a category; level rules (same level refreshes, lower does not replace) |
| Death | `CUser::Death` 0x004586a0, `CUser::KillSoul` 0x004599e0, `CUser::ExpDrop` 0x0045ad50 | done: buffs cleared, revive per `0x0553` (`CUser::PacketPC`): 2 bind point (KillSoul: exp loss, full pools), 3 wait 30 s for the party leader (KillSoulParty: at the leader if alive on the map, else KillSoul), 4/5 resurrection rune (KillSoulByItem / NoMove, real type 26 Special 37, ReqVg set preferred to move and 0 to stay; no exp loss, half pools, at the bind point / in place), automatic KillSoul 200 s after death (UpdateRebirth); ultimate mode needs ability 46 or a rune (no exp loss, half pools) or the character is deleted and disconnected; 0x0406 corpse leave to the view, 0x0551 id/kind/exp/position. Bind point: `CMap::GetBindPos` 0x004295a0: uses the faction destination from SQLite, then the nearest eligible SVMAP group, then the racial birth position, 5% of the level's exp lost above level 10 (1% with the Blue Dragon or White Tiger charm, 0% with Red Phoenix or an exp-protection ability 38, decided at death) |
| Attack range and speed | `CUser::EnableAttackRange`, `g_ConstUserAttackSpeed` 0x005424e0 | done: weapon reach + mob size + 2.3 + movement allowance (taken as its 2-unit floor); swing index moved by gems, passives and buffs |
| Visibility | `CZone::EnterUser` / `MoveUserLocal` 0x00416370 / `MoveMob` 0x004169d0 / `SendMoveUser` 0x0041f350 / `SendMoveMob` 0x0041f860 / `SendCellInfoView`, `CCell::SendCellInfoEnter/Leave`, `CZone::PSendView` | done: 40-unit cells kept while inside the 60-unit view window (no flip-flop on borders), 5x5 cell view for users, mobs, NPCs, map items and other users; enter/leave only on cell changes, for both sides (0x0201/0x0202 users, 0x0601/0x0602, 0x0E01/0x0E02, 0x0401/0x0402); moves to the 5x5 view; combat and effects to 60 units; mob corpses leave 5 s after death (`CZone::UpdateMobDeath`). Before this GX Server used circles, re-sent mobs at every border crossing, never sent 0x0202 and sent every map item on entry |
| NPC patrol | `CNpc::UpdateMove` 0x004bbeb0, `NpcQuest.SData` MoveDistance/MoveSpeed | done: steps of MoveDistance every MoveSpeed ms (16 units / 8 s in EP4.5 data), points walked forward then back, rest rand(10 s) + MoveSpeed + 10 s at each point |
| NPC guards | `CNpc::UpdateGuard` 0x004bc0a0, `FindTarget` 0x004bc180, `Chase` 0x004bca00 | Type 8 pursues enemies, detects stealth and uses the original guard attack packets |
| Skill learning | `CUser::LearnSkill` 0x004842e0, `EnableLearnSkill` 0x00484080 | done: levels 1-3, new skills at level 1 and known ones to any higher level, errors 1 points / 2 level / 3 otherwise, Country by family, job, growth mode (easy only Grow 0, normal up to 1, hard all but 3), TypeShow 0 hidden; no prerequisite skill (GX Server's PrevSkillID check was invented) |
| Mounts | `CUser::PacketMaininterface` 0x00468420, `UpdateRebirth` 0x0048d240, `SendShapeVehicle2` 0x00482660; native client 0x004caae0 / 0x0057ace0; Phoenix seat catalog | implemented with GX metadata: fixed 2-second native cast bar/Town Portal effect, retained ReqIg cooldown, absolute tiers 4/5, explicit racial MON index and seat count, validated two-rider invitations, jump departure and MON bone effects. Normal damage cancels preparation without dismounting established riders; hard CC/death/map departure detach them. See [mounts](../../docs/mounts.md) for fields, external bones.ini and migration. |
| Mob gold pickup | `ItemGet` 0x0045d2e0 | done: mob gold gets ability 44 percent, +20 White Tiger, +50 Red Phoenix |
| Trade | `CUser::PacketExchange` 0x0046ec90, `ExchangeItemAddEnable`, `ExchangeMoneyAddEnable` | done: same zone and country, not already trading; offers from the bags only, count 1..stack, no quest items (real type 27), no item twice; room and gold checks on completion |
| NPC shop sale | `CUser::PacketShop` 0x0703 | done: bags only, count 1..stack else ignored, quest items pay nothing (a partial sale keeps the stack), no sale while attacking or on gold overflow |
| Warehouse | `CUser::ItemBankToBag` 0x0045bf30, `ItemBagToBank` 0x0045c4e0, `IsBank` | done: 240 slots, withdrawing costs 5% of Sell x count (type 94 free), quest items cannot be stored |
| Equip requirements | `CUser::EnableEquipment` 0x0045b450 | done |
| NPC shop buy | `CUser::PacketShop` 0x0046af10, `NpcQuest.SData` | done (no guild-house discounts) |
| Gatekeepers | `CUser::PacketMaininterface` case 0x020C, `NpcQuest.SData` | done (no guild-house rates; teleport is immediate) |
| Item durability and repair | `CGameData::LoadConstItem` (wQuality/wEffect tables), `CItem::GetQualityLevel` 0x004540a0, `CUser::QualityByAttack` 0x0045b260, `QualityByDefense` 0x0045b290, `QualityDec` 0x0045b350, `ItemEquipmentAttackAddQ`, `CUser::ItemRepair` 0x00464340 (0x0554) | done: the weapon loses 1 per hit (normal and skill), one armour piece per hit taken (shield, helmet, upper, gloves, lower, shoes from the rotating index to the end), none with ability 36 or while mounted; level 0 above half the item's Quality, 1 at or below half (effects halved), 2 at or below a tenth (no effect); accessories and gems always level 0; a level change resends the stats; qualities saved with the position. Repair by bag/slot or 0xFF/0xFF for the equipment: quality = item + gems' Quality, price round(missing / 400 * Buy / 30), at least 1, nothing without the money; 0x0554 bag, slot, money. Not modelled: guild-house repair rates |
| Return to town | `CUser::PacketPC` 0x0556, `CUser::UpdateResetPosition` 0x0048d470 (type 0), `CUser::CancleMove` 0x0048a010 | done: alive with no move pending, the bind point (GetBindPos) 10 s later; moving, auto-attacking, using a skill or being hit cancels it silently. GX Server's `/town` chat command stays as its own escape |
| Priest services | `CUser::PacketPriest` 0x00467560 (0x0F01/0x0F02, 10 gold) | excluded: never implemented by the game (user decision) |
| Chat | `CUser::PacketChatE` 0x00470490, `PSendViewChat`, `PSendViewRange`, `PSendAllCountry`, `CZone::SendAllCountry` | done: length byte 2-128 (0-1 ignored, more closes the connection), 0x1101 within 30 units, 0x1102 whisper by name in the same country (0x1106 1 otherwise), 0x1103 trade to the country, 0x1104 guild, 0x1105 party, 0x1107 shout within 100 units every 30 s, 0x1111 area to the map's country. Not wired: 0x1108 world shout (needs the item-granted permission), 0x1112 raid leader notice, chat bans and GM listening; the `/town` command is GX Server's own |
| Party | `party_runtime*`, `mob_death` (solo exp ×1.2, party count factor, level share, 100-unit range) | done: invites, roster, leader, expel, chat (0x1105), pools/level updates, exp and gold split, shared loot |
| Raids | `party_runtime_raid*`, raid exp table | done: convert, 30 wire slots, leader/sub-leader, slot swaps, invites, auto-join, expel, disband, chunked roster; the item-division mode is stored and sent as the original does (`CParty::SetItemDiv`, +0x104, read only by EnterRaidParty's roster); EP4.5 never applies it to loot (`CParty::GetRaidItemDivSeq` has no caller), so pickup follows `CItem::PickEnable` for raids too |
| Quests | `NpcQuest.SData` (1720 quests, item links), `CUser::QuestEnableStart` 0x0047f080, `QuestStartRequest` 0x0047f220, `QuestFindItem` 0x0047f8e0, end/reward/progress routines, `CharQuests` procedures | done: GX loads the six SQLite NPC and quest tables at startup; start by NPC or item, level/race/job/mode/prerequisite checks, start items, item/mob/player-kill objectives, timed and minimum-time quests, reward variants and player choice, repeatable quests, abandon, party kill credit, quest item drops (`Mobs.QuestItemID`); player kills count through QuestUpdateKillPC, item-use starts (type 4) not wired |
| NPC talk | `0x0E04` | done (the original only records the NPC in use) |
| Item mall | `0x2601`-`0x2607`, `PS_GameDefs.dbo.ProductList`, NCash procedures (linked server/MSDTC, reproduced locally), `usp_Update_UserPoint`, `PointLog`, `UserStoredPointItems` | done: points, buy, gift, gift box and claim; the client catalog (`Cash.SData`) must be in ProductList (`tools/sql/client_cash_catalog.sql`) |
| Heals and buffs on users | `CUser::ChkTargetUser` 0x0044e140, `IsTeamUsable` 0x0044bf30, `SkillHeal` / `SkillBuff` and their Party / NParty variants, `HealUser` 0x00450670, `BuffUser`, `RemApplySkill` | done: 0x0511 is a skill on a user (0x0517 on a mob); TargetType 3/7 reaches one friendly user (the caster, or alive on the map, of the same country or on a WarType P map, within reach; otherwise 0x0511 result 11), 4/5 the party members on the map within ApplyRange (a raid only the caster's group of 5), other target types the caster; heal 41 (HealHP + 2 x WIS, HealMP, HealSP), cure 42 (debuffs removed) and buffs are applied in each target's session; a single target answers 0x0511 to the view, a party skill announces 0x0511 result 4 and sends each member's 0x0513; 0x0511 0xFF cancels an own buff. sacrifice 43 (Sacrifice, the data has no heal values): the caster pays its whole SP/MP pool as the cost and the target recovers the caster's maximum SP (level 1), SP and MP (2) or HP (3) |
| Normal attacks between users | `CUser::PacketPC` 0x0502, `UpdateAttack` 0x0048ca50, `IsTeam` 0x0044bdf0, `EnableAttackRange` 0x0044bf90, `AttackUser` 0x00456870, `AttackSuccess` 0x00456b10, `AttackDamage` 0x00457030, `CriticalSuccess` 0x00456fd0, `ChkMirror` 0x0044e080, `Damage` 0x0044c1e0, `Death` 0x004586a0 (type 1) | done: 0x0502 starts an attack only on a hostile user (alive, on the map, other country, not a WarType P map); reach is the move allowance + the attacker's range + 2.3; the swing is resolved in the target's session: block 85/86/92, hit chance as the mob version but clamped 1..99, damage without a level factor (attack - guard + variance, close-range bow cut unless cross-road, element, LUC/10, critical 1..99 from the attacker's critical - target LUC x 0.034, x1.5 + LUC x 1.4, the crit repeats the bow cut), absorption, armour wear on the target, weapon wear on the attacker only on a critical; 0x0502 to the view (0 normal, 1 critical, 3 miss); a mirrored hit shows no damage, then hurts the attacker with 0x0515 (mirror owner, attacker, HP/SP/MP); a kill announces 0x0504 type 1 with the killer. Duels, guild duels and hostile skills are their own rows. Not yet: the shape-13 exemption |
| Death by a user | `CUser::Death` 0x004586a0, `CDamage::Add` / `GetMax` / `Reset`, `GetExp` 0x004581b0, `AddExpFromUser`, `KillPCFrom` 0x0045b050, `KillPCTo` 0x0045aed0, `QuestUpdateKillPC` 0x00480020, `ItemDropByUser` 0x0045a180, `ItemDropByUserDeath` 0x0045a5f0, `SetSkillAbility` 0x00485e00 | done: every HP hit is recorded per attacker (users and mobs) until the next death; the attacker with the most damage (first on a tie) is rewarded, nobody when it is a mob or the user itself. Solo killer: GetExp (dead level x 50 x ((dead - killer + 1) x 0.2 + 1), at least 1, 0 when the dead user is 10+ levels above) x 1.2 with exp charms, a kill (QuestUpdateKillPC on the first player-kill quest, kills + 1 up to the GX cap 1000000, 0x020E 0, 0x0522 to the view) and a death for the dead user (deaths + 1 up to the GX cap 1000000, 0x020E 1); party killer: each member within 100 units gets GetExp(own level) x party/raid factor x level share and a kill, the first one also books the death. Drops above level 10 unless ability 37: a regular account loses one item (chance by Vg: 65-100%) among the first 10 inventory items with RealType 1-25, ReqOg 0 and Type not 94, a GM account 1-3 items among the first 25 not of Type 94; 30% of the time 1..level x 5 gold, split among the killer's party within 100 units or dropped for the killer. The item follows the killer party's division (2 a random member in range directly or lost, 1 the next member in range directly, 3 owned by the next member in range, else the ground for the killer and party). Deaths now only count deaths by users, as the original (mob deaths no longer add to them). Duel deaths (PvPGetDeathType 4/5) and the country tally are in the Duels, Guild duels and Country kill count rows |
| Hostile skills on users | `CUser::SkillAttack` 0x00451d90 / `SkillDebuff` 0x00452bf0 and their Select / Range / RangeSelect variants (user and mob centers), `ChkTargetUser` 0x0044e140, `AttackUser(skill)` 0x0044f4c0, `AttackSuccess` 0x00456b10, `AttackDamage` 0x00457030 | done: an attack (TypeEffect 1) or debuff (3) sent as 0x0511 targets users: TargetType 3 one hostile user (else 0x0511 result 11, out of reach 5), 6 the hostile users around the caster, 7 around the target user, each area announced with 0x0511 result 4 and followed by the mobs in ApplyRange; 0x0517 TargetType 7 around a mob also hits the hostile users there. Per user: block, SuccessType 1-3 (2 LUC, 3 WIS) or the category's chance, mirror (0x0515 back), DamageType 0-5 on the user-target weapon formula, including its magic branch (magic attack reduced by a 1..50 level term plus the target's magic guard over weapon attack + INT, 1..70%), SP/MP damage, absorption, weapon wear on a critical; 0x0511 first hit, 0x0512 drain (the caster heals the damage), 0x0513 later and area hits, result 3 on a miss; AddDamage with 0x0514; detail 22 repeats, 23 sacrifices the caster; a debuff stays on a surviving target and its damage over time counts as the caster's (death by user). Also fixed for mobs: SuccessType 2 reads LUC and 3 WIS (they were swapped). Not modeled: skill 56's move-speed change, the HP-ability shields (7/8) and state resistances in AttackSuccess |
| Duels | `CUser::PacketPVP` 0x0047d0f0, `ExchangePvP` 0x0047caf0, `ExchangeCancleReadyPvP`, `ItemDropPvP` 0x0047c990, `PvPModeEnd`, `PvPChkLoseDeath` 0x0047b2c0, `PvPChkLeaveByTime` / `ByMove` / `ByAttack` / `ByDeath` / `ByZoneLeave`, `PvPHaveDebuff` / `PvPClearDebuff`, `IsTeam` / `IsTeamUsable` | done: request 0x2401 (free users of the same country, any country on maps 39/40; 0x2402 2 refused, 4 with a debuff; 10 s to answer, then 0x2402 3), answer 0x2402 (0 refuses; else the wager window 0x2407); wager slots 0x240A/0x240D (8, no quest items), 0x240B, gold 0x240C, every change cancels ready (0x2409 3,1); both ready (0x2409 0) take the wagers out of the inventories (0x0213/0x0206, 0x2408 0), the center is the midpoint, 5 s countdown 0x2403 then 0x2404; while fighting the two are foes (any country) and nobody in a fight is team-usable; give up 0x2406 2, leaving the 40-unit circle 0x2406 1, a third attacker, a mob hit or another death 0x2406 3, leaving the zone or logging out 0x2406 0 to the opponent, each returning the wagers at their owners' feet; the window closes with 0x2408 2/3. A death to the opponent drops both wagers for the winner, adds a victory and a defeat (0x2405 1/2), no exp/kill/drop, and the loser keeps exp, the ultimate mode and skills above 200. As the original, moving with the window open measures from the last duel center and closes it. |
| Guild duels | `CUser::PacketPVPGuild` 0x0047da70, `CGuild::PvPUserSet` 0x00437ed0 / `PvPUserLeave` 0x00438160 / `PvPUserClear` / `SendPvP` / `SendPvPJoinUser` / `PvPReset`, `PvPGModeEnd`, the guild parts of `PvPChkLeaveByTime` / `ByMove` / `ByDeath` / `ByZoneLeave` / `ByGuildLeave` and `PvPChkLoseDeath` (type 5), `IsTeam` / `IsTeamUsable` | done: 0x2501 from a rank 1-4 member of a free guild to a rank 1-4 member of another free guild on the zone (else 0x2502 2), 10 s to answer 0x2502 (0 refuses, else 5 s countdown 0x2503 center/opposing guild, then 0x2504); the online members on the map within 40 units of the leaders' midpoint enroll; the two sides are foes and only team-usable within their side; leaving the circle, the zone or the guild (reason 1) or dying (3) takes a member out (0x2506), a side without members or whose leader leaves for reason 1 loses (0x2505 1 winners / 2 losers) and both reset; a death by the other side gives no rewards and keeps the exp; before the countdown the leaders' leaves cancel with 0x2502 3. GuildPvPWin / GuildPvPLoss (guild points) and the ranking's duel bonus have no writer in EP4.5. Not handled: a disband during a guild duel |
| Country kill count (God's blessing) | `CGameData::LoadKCStatus` 0x004bf4a0 (data/KillStatus.SData), `CWorld::AddKillCount` 0x00413600 / `ResetKillCount` / `UpdateKillCount` / `SetKillCount` / `ApplyKillCount`, `CWorldThread::SendAllKillCount`, `CUser::SendKCStatus` 0x00481920, `UpdateKCStatus` 0x0048b780, `usp_Read_GodBless_R` / `usp_Save_GodBless_E` | done: 30 tiers per country, each keeping the earlier tiers' values; each kill counted by KillPCTo adds 1 to the killer's country unless either country holds its top tier; the top tier holds 10 minutes and resets the other country at once, then itself; 0x020F (country, count up to the top tier) to the country on a tier transition/debit and every 10 minutes, 0x0211 (country, count, ms left) at world entry; saved with usp_Save_GodBless_E (creating the WorldInfo row with usp_Save_WorldInfo_E when missing). Tier types on every user of the country: 1-3 percent of max HP/MP/SP, 4-9 STR/REC/INT/WIS/DEX/LUC, 10-12 percent of max HP/SP/MP to sitting and standing recovery, 13-15 to in-combat recovery, 16 critical, 17 melee accuracy, 18/19 percent of melee/ranged defense, 20/23 magic evasion, 21/22 melee/ranged evasion; country-wide 24 adds to the gem chances and 25 cuts the death exp loss (types 26/27 have no reader). Below the first tier nothing applies (the original reads the slot before its table). Relic captures use the native per-country amount (final threshold / eligible relic count + 1), debit the previous owner and reset to configured base ownership on activation; see the obelisk source audit below. GX stores the shared tiers in PS_GameDefs_GoddessBlessing and sends the bar/countdown, cumulative SysMsg tooltips and activation state via 0x7F1B; see [blessing](../../docs/wiki/goddess-blessing.md). |
| 5-second recovery | `CUser::UpdateRecover` 0x0048cfb0, `InitEquipment`, `GemEquipmentAdd` | done: the mode seen at the previous tick decides the amount: sitting 5% of the max + sitting abilities % + the flat sitting recovery (28 base), standing the flat standing recovery (4 base), in combat the flat combat recovery (0 base), all truncated; Special 16 gems turn their HP/SP/MP into sitting and standing recovery instead of max pools; plus the country tier |
| World calendar | ps_dbAgent `SetWorldDay`, `CWorld::GetWorldDay` 0x0046c2f0, `usp_Read_WorldInfo_E` / `usp_Save_WorldInfo_E` | done: the base is WorldInfo.LastWorldTime (packed (year-2000)<<26 \| month<<22 \| day<<17 \| hour<<12 \| minute<<6 \| second), created from the local time on the first start; 0x0404 at character entry sends the seconds elapsed since it (it was a fixed packed date) |
| Gem skills | `CUser::AttackItemOption` 0x0044c630 / 0x0044c720, `BeAttackedItemOption` 0x0044c970 / 0x0044cae0, `AttackItemUser` / `AttackItemMob` | done: a gem with Special 1-15 names a mob skill (Skills id = Special, level 100); after a landed hit the attacker's weapon gems cast on the target (user or mob) and, against a user or from a mob, the target's armour gems (shield, helmet, upper, gloves, lower, shoes, stopping at the first empty slot) cast on the attacker; TypeEffect 1 as a selected attack, 3 as a debuff, each with its own success roll; gem casts cast no gems (m_bItemAttack) |
| Movement Rune (item Special 32) | `CUser::ItemUse` case 0x20 at 0x465e55 (move type 5, 0x1388 ms later, destination = the party member's map and position) | done: used on the party member selected in the target window (the server follows the selection through the target-effects watch), five second cast shared with the summon rune's cast bar, one rune consumed, interrupted like any travel cast; a dead member is still a valid destination; a character outside the party or of the other faction, or an instance map, is refused. The GM command /amove does the same with no rune or cast |
| PvP kill rewards (`CUser::Death` 0x004586a0) | GetExp x 1.2 to the killer or the party, KillPCFrom | GX rule: killing a player gives no experience; the kill is still credited (kill count, quests, PvP reward stages) and the killer is told with GXKillNotice (0x7F1A), shown in red as SysMsg 8 |
| Degeneration (debuff 3/90, skill 97) | `CUser::AddApplySkillDebuff` case 0x5a stores ChangeType and the monster id of the skill (`m_wTransFormTargetMobID`, `SendSpeed`, `SendShape`) | done for the effect: the target moves at the lowest tier (slowest, whatever its buffs), cannot attack or use spells (`cannotAct`, also for prepared casts and dashes), keeps consumables and equipment changes, any damage ends it (like a sleep) and it cannot be cast upon a monster (SysMsg 317). The monster named by ChangeLevel is shown in GX's own way: the server announces it with GXTransform (0x7F18: character, monster; 0 ends it; repeated for characters entering a view) and the client hides the character model and plays that monster's model in its place, walking while it moves |
| Trap (TypeEffect 5, detail 101: Trap 113, Dragnet 169) | `CUser::AttackUser` 0x00451c80 → `CUser::SkillField` 0x004534e0 (a `CTrab` at the caster's feet, owner, country, end = KeepTime), `CZone::UpdateTrab` 0x004226b0, `TrabDamage` 0x00422480 | done: the cast is announced with 0x0511 (result 0, no target) and needs no victim; the trap is invisible to everybody; a sweep every 250 ms springs it on the nearest enemy user or mob within ApplyRange; a sprung trap is consumed and an expired one is dropped; users lose their stealth, take TrabDamage on HP, SP and MP (DamageType 4: % of the maximum, 5: % of the current, otherwise fixed; at least 1, at most 30000) with 0x0513, and mobs with 0x0519 (immune when `ResistState[StateType-1]`, the skill's Arrow as aggro, the mob turns to the owner if it survives). GX has no limit of traps (the original allowed one): KeepTime and ResetTime of the skill are the only limits. Traps belong to the owner's session and vanish with it or when it leaves the map. Fire Bush (detail 102, skill 48, `SkillField` too) is a curtain of fire in GX: it stays for its KeepTime and every 0.5 s hurts each hostile mob and user within ApplyRange with the skill's DamageHP as fixed damage, like Fire Thorn, with the element damage effect on each target; it is cast with its ready time and needs no victim, and the view is told where it stands and for how long with GXField (0x7F17) so the client shows its area library DA_area_fire there once, as authored (about eleven seconds) |
| Health Assistant (buff 4/87) | `CUser::AddApplySkillBuff` case 0x57 stores the level in CUser +0x1338; `CUser::Damage` 0x0044c259 and `DamageByKeepSkill` read it | done: level 1 spends SP against each hit, level 2 MP, level 3 SP and then MP, point for point; HP takes what is left and the result packets show the part moved to SP / MP. Applies to mob hits, user hits and skills, periodic damage and Fire Thorn; fall damage (a GX rule) stays on HP |
| Detection (buff 4/93, normalized from effect 6) | `CUser::ChkKeepSkill` 0x00487190 case 0x5d → `CUser::ChkFindHide` 0x00486e00 → `RemApplySkillHide` | done: while the buff lasts, every second (and when it is applied) each living enemy user within ApplyRange of the caster's current position (so the area follows the caster) loses its stealth; allies, concealed GMs and users that are not hidden are untouched. The original skips only party members (`IsTeam`); GX uses its team rule (same country or free-PvP zone), as for every other hostile query |
| Skill resurrection and other TypeEffect 6 skills | `CUser::AttackUser` 0x00451c80 → `SkillOnce` 0x00453610 (Type 121-125), `LifeSkill` 0x00459d20, `AttackSuccess` 0x00456b10, `UpdateResetPosition` | done: 121 resurrection (dead target only, else result 11; revives at the caster with full pools and no exp loss), 122 soul kill (KillSoul on a non-ultimate target), 123 detection (enemy only, else 11; 0x0558 defence element, level, mode), 124 roll only, 125 return to the bind point; result 0x0511 to the view, failure 3, costs and reset time spent either way, 0x0510 cast start to the view. Not modelled: the target's own block/resist (cross-session) and the level-based default success. `LifeSoul` (0x0552, needs the soul state CUser::Soul sets) not wired |
| Guilds (core) | `CUser::PacketGuild` 0x004764a0, `CGuildMgr::CreateGuild` 0x004321e0 / `FireGuild` / `SendGuildList` / `ConnectUser`, `CGuild::*` (CreateAddUser, ConnectUser, DisconnectUser, SendGuildUserList, SendGuildJoinList, JoinRequest, JoinRequestCancle, JoinOK, JoinNO, JoinUser, JoinLeave, JoinExpel, ChangeGuildUserLevel, ChangeUserLevel, UpdateRemark, Fire, SendAllGuildChat), `CGuildCreate::*` | done: creation by a party of exactly 7 on the map (level 11+, not easy mode, no guild, 72 h after the last join, unique name), every member accepts (0x0D22), 1000 gold from the master, master rank 1 and members 9, 0x0D21 / 0x0D30 to the country; disband at once with at most 2 members or by vote of the master and two party admins (0x0D03/0x0D04, 0x0D0C 101, 0x0D11); invitation by an admin (0x0D05/0x0D06, 30 s); applications (0x0D07, rank 0 rows, 0x0D16/0x0D17/0x0D18) accepted or declined by admins (0x0D08); leave (0x0D09, the master only alone, an empty guild disbands); expel of lower ranks (0x0D0A); rank steps 2-9 with at most 7 admins, sub-masters not to their own rank (0x0D0C 200+rank); remark by the master, no quotes (0x0D20); online/offline/level/left/expelled status (0x0D0C 0x68/0x69/level/0x66/0x67); member lists at entry (0x0D12 online, 0x0D13 offline, admins also 0x0D16); guild list of the country (0x0D0D, 0x0D2F x14, 0x0D0E); guild chat 0x1104; guild name/id in 0x0105, 0x0201 and 0x0303. DB through the original procedures; usp_Read_Guild2_R and usp_Save_GuildRemark_E run as plain SQL (they reference a Guilds.Rank column and a GuildHouses table the EP4.5 schema lacks); the linked-server Guilds triggers are disabled and GX Server writes their PS_UserData.dbo.CreatedGuilds copy. Approximations: a creation/vote member leaving the world counts as a refusal (7); an expired invitation is dropped silently. The guild bank, the ranking battle and guild duels are their own rows. Not yet: house/NPCs/etin (0x0D23 sends the stored etin only, 0x0D24-0x0D27, 0x0D34/0x0D35) |
| Guild bank | `CGuild::ItemBagToGuildBank` 0x0042f390, `ItemGuildBankToGuildBank` 0x0042fb40, `ItemGuildBankToBag` 0x0042fea0, `SendBankItemList` 0x0042f250, `GetGuildHouseBankAllowSlotCount`, `CUser::IsGuildBank` (bag 0xFF, slots < 240), GuildStoredItems procedures | done: only guilds ranked 1-30 (0x0D28 1/30 otherwise); usable slots from GuildHouse.ini [Bank] PermitSlot at the bank NPC level (40 without a house; 0x0D28 2/page beyond); deposit from the bags (not equipment, quest items or ReqOg items), rank 9 refused (0x0D28 4/8), a taken slot only for admins (swap); withdraw and rearrange by admins only (0x0D28 2/2), withdraw fee trunc(5% of Sell x count) except type 94; 0x0204 to the mover (rearrange to every member), 0x0D31 placed / 0x0D32 taken (slot, item, char id) to the others; 0x0D29 list (33-byte entries, 50 per packet) at entry and on joining. Ranks come from GuildDetails.Rank: every guild stays at 31 until the guild ranking battle exists (its points are added in `CGuild::SendRankingZone`; `CGuildMgr::UpdateGuildPointRank` ranks them), so the bank is closed for now |
| Instanced zones | `CWorld::GetZoneByMapID` 0x00413df0, Map.ini CreateType (R/H/P), CreateTime | partial: zones are uint16 ids, real maps keep theirs and instances get virtual ids from 1000 that alias the map's svmap data (fresh mobs/NPCs) and grid; the client and Chars.Map only see the real map; a character saved in an instanced map starts at its bind point. Used by the guild ranking battle and the party dungeons; an instance with nobody in it gets an expiry of ExpireTime minutes + 30 s the first time it empties (CWorldThread::Update; not reset when users return) and is destroyed once empty past it. H zones (guild houses) are not wired |
| Party dungeons | `CUser::PacketMaininterface` 0x020A, `CMap::IsPartyInsZoneOpenTime` 0x00429020, `CWorld::FindPartyInsZone` 0x00414430, `GetZoneByMapID` (CreateType P) | done: a portal to a P map (64-66) is refused with 0x020A (portal, result) outside its CreateTime slots (9), without a party (4), when the party's instance holds EnterMaxUser (11) or the party has fewer than CreateMinUser members (10); otherwise the party gets its own instance, created on the first entry |
| Guild ranking battle and ranking | `CMap::IsGuildRankMapOpenTime` 0x00428d20, `GetGuildRankNotices` 0x00428eb0, `CUser::ZoneEnterUserMove`, `CMob::ItemDrop` -> `CGuild::SendRankingZone` 0x0042c6e0, `CZone::UpdateCloseGuildRanking`, `CWorldSendThread::Run`, `CGuildMgr::UpdateGuildPointRank` 0x00432b70, usp_Save_GuildInfo_E2 | done: CreateType R maps (map 50, Sunday 21:00 for 1 hour) open per CreateTime (two slots, past-midnight wrap); notices 0x0D26 (map, start hour at 30/10 minutes before, 100 open, 101/102 ten/one minute before the close); each guild enters its own instance during the battle (no guild or closed: the move fails), every entry counted; Money1 0 / Money2 > 0 mobs give Money2 points to the killer's guild instead of gold, 0x0D27 (guild points, country best and its guild) to the members inside; at the close everyone goes to the bind point and empty instances are removed; 30 minutes later each guild's value is points + min(25 x entries + duel bonus, points / 2), each country ranks its guilds with a value (equal values keep the first, as the original map insert), top 30 ranked 1-30 and the rest 31, stored with usp_Save_GuildInfo_E2 and sent per country (0x0D37). Not modelled: the duel bonus (no guild duels yet) and the house keep-etin settlement |
| Guilds (rest) | house, NPC upgrades, etin | missing |

## NPC guards and patrol

The EP4.5 server uses NPC type 8 for guards. `CNpc::FindTarget` checks a
20-unit acquisition radius and a 30-unit limit from the current route point.
`Chase` uses one quarter of the patrol movement interval. Both attack paths
pass 2 to the chase routine, which adds 1 to establish the 3-unit contact
range. `ReturnHome` rejoins the current route point within 1.5 units.
The original attack packets are 0x0E05 for players and 0x0E06 for mobs.
The death source type is 3.

GX keeps the NPC and mob factions from their existing definitions. It
prioritizes enemy players and otherwise selects a nearby hostile or neutral
mob. A neutral guard follows the native faction comparison. The guard kills
on contact, including when the victim is stealthed or protected, and never
awards player kill credit. Route steps respect the map walkability mask and
interpolate height toward authored patrol points. Pursuit and return use the
GX map navigation geometry where available. GX revalidates contact in the
player session before applying a queued strike. A guard spawned with
`/nmake 8 <row>` stands at its spawn between pursuits. The spawn anchors
its range and return even though it has no patrol route.

## Spin combinations and Stinger

Wind Spin (38) and Reinforced Wind Spin (156) use their recovered looping
client state. By explicit user requirement, a distinct spin or instant hostile
caster-centred area may replace a spin after its first contact (4/30 seconds),
without waiting the ordinary attack-speed interval. Own reuse, MP/SP, reach and
obstructions remain enforced; early requests retain only the latest eligible
follow-up. This boundary is not claimed as a measured EP4.5 server interval.

Stinger (56) uses server-authored swept movement and 0x7F11 replication rather
than the original client-trusted distance credit. Native windup/speed/contact
frames are preserved. Client destinations, speed floods and movement cannot
steer a dash. See [evidence and limitations](../../docs/spell-fluidity-research.md).

## PvP rank titles and rewards

The client reference is EP5.4 SHA256
`8e5ef14a2a3bfef15bae61b03d185e551b6c3fd44d09a2861d57428400bb3263`.
The 31 kill thresholds at 0x0070b2e8 drive both `grade2` nameplate icons
(0x00443f90) and Status titles (0x00504285). Native title IDs are 5450/5451
plus 5389-5403, with 5404 alone for the last rank. The full nameplate path
compensates its temporary -6px offset before the final name draw
(0x0044bd3c, 0x0044be19, 0x0044bfbf-0x0044c068); icons sit 1px beyond the
name and 3px above it. GX loads the atlas only for a visible ranked label.

The server reference is SHA256
`0fb7c9f13b5bcf186118f8bfe73ab214226b93e8aa502b8e6fded4a8312bc547`.
Its `g_tKillInfo` at 0x00541d70 and `g_tDeathInfo` at 0x00541eb0 each have
16 stages ending at 200,000; `g_nMaxKillGrow` at 0x00541f70 is 16 for all
three eligible modes. GX deliberately extends `KillPCTo` / `KillPCFrom` to
1,000,000 and all 31 reward stages by the user's explicit 2026-11-15 request.
The amounts for every mode and all high ranks are recovered from the client
tables at 0x0070b164/0x0070b1e4/0x0070b264/0x0070b364, not placeholders.
The inactive NEW_SERVER doubling option is not enabled in GX.

`CUser::PacketMaininterface` admits native reward requests 0x0218-0x021A
for non-easy characters at level 16+. `StatusKillDeathResult` (0x0048c8a0)
reports next stages. `StatusKillResult` (0x0048c640) and `StatusDeathResult`
(0x0048c750) redeem the next earned stage only, saving `KillLevel`/`DeadLevel`
with the updated stat pool/gold. The reply layouts are two u8 stage fields,
result u8 plus stat delta u8, and result u8 plus gold delta u32. A failed
claim grants nothing; GX initializes its failed delta to zero. Gold addition
saturates at uint32 maximum. No client-supplied count, point amount or gold
amount is accepted.

`StatResetStatus` (0x00480660, store 0x0048089e to CUser+0x15c) clears
claimed kill rewards and restores level-derived points, leaving earned kills
and death claims intact. Their PvP stat rewards can be claimed again after
the reset. Claim stage 32 marks completion of the 31 available rewards; it is
not an extra display rank. The final rank remains visible with disabled claim
controls, including after reconnecting. The client updates Title, stat pool,
gold and claim controls from confirmed packets, including re-entry and reset.

## Obelisk source audit

GX implements the relic lifecycle, persistent ownership, linked portals and
linked boss populations described below. The reference executable is the local
EP4.5 `ps_game.exe`, 1,323,008 bytes, SHA256
`0fb7c9f13b5bcf186118f8bfe73ab214226b93e8aa502b8e6fded4a8312bc547`.
The executable recorded by the read-only Ghidra project has that same hash.
PR #10 cites a different 1,581,568-byte server; its addresses and claims must
be reconciled against this reference before integration.

The supplied `Obelisk.ini` has SHA256
`e3c72654021dddf318b6a2cdee48055ddf2e58ed5cef2b29c44d1f2b8db62164`.
It defines nine relics: seven on map 0 and one each on maps 18 and 30.
Its mob triplets are neutral/Light/Fury. The wire uses owners 0/1/2 in that
order, while the reference runtime uses Light/Fury/neutral as 0/1/2.

Recovered reference behavior:

- `CMap::LoadObelisk` (0x00427300) keeps the configuration identity
  `map * 65536 + entry` separately from an incrementing runtime identifier.
  Instructions 0x0042814b-0x00428155 assign the latter. `SendObeliskList`
  (0x0042b040) sends this runtime identifier, owner and X/Z in 13-byte records
  under 0x2101, in batches of at most 100. The list also carries portal marker
  states 100-103; it is not exclusively a relic list.
- `CMap::ObeliskDestroy` (0x0042a410) removes dependent guards, changes owner,
  emits map ownership changes (0x2102) and global destruction notices (0x2103),
  updates last-conquering-country state and reevaluates linked portals.
  A neutral destruction request falls back to a nonneutral configured base
  country. A reset path can suppress the ordinary capture scoring and notice.
- `CMap::UpdateRespawnObelisk` (0x00429da0) respawns the owner's relic variant
  when its deadline expires. Guard population and deadlines are maintained
  separately; guards are replenished only while the parent stands and their
  current country variant exists. A zero variant is an absent population.
- `CMap::ObeliskReset` (0x0042aa40) removes the standing relic, schedules a
  600,000 ms delay, restores the configured base country and publishes the
  resulting state to clients and the DB agent. This is not a universal rule
  that every relic immediately returns neutral when blessing ends.
- `CWorld::SetKeepObelisk` (0x00413bd0), `CMap::SendDBObeliskList`
  (0x0042b370) and DB-agent opcode 0x0207 restore/publish ownership and timing.
  The persistent record also packs last-conquering-country state. Loading INI
  defaults and restoring saved state are distinct steps.
- The loader writes both blessing contributions at 0x00428557/0x00428575:
  integer division of the country's final KC threshold by the eligible relic
  count, plus one. `ObeliskDestroy` consumes these amounts through
  `CWorld::AddKillCount` / `SubKillCount`. The prior claim that these globals
  were never written was incorrect. GX's existing PvP blessing implementation
  now connects relic captures to this path, without a separate altar cap.
- `CMap::UpdatePortalLink` (0x0042a000) tests configured dependency sets against
  the last-conquering country, rather than proximity to a relic. All matching
  dependencies open that faction's side; no matching dependencies close it.
  Partial ownership can retain the previous open state.

Phoenix's blessing implementation is useful prior work, but its documented
contract includes explicit custom decisions: neutral startup, a 50% altar
contribution cap, provisional PvP scoring and an XP bonus. Those choices do not
establish original-server parity. The source also has separate `bothmob` boss
phase/portal and random-respawn records; loading nine relics alone does not
implement that event system. GX reuses the contribution’s client foundation with explicit corrections
for the reference versions, runtime IDs, restart/reset timing, capture scoring,
DB restoration, portals and boss chains. Unit, race, original-world boot,
OpenGL pixel checks and encrypted disposable-DB client/server tests cover the
implemented lifecycle. See [Obelisks](../../docs/wiki/obelisks.md) for behavior,
protocol, persistence and content limitations.

## Element table

`CGameData::m_AttackAttr` (attacker attribute × defender attribute):

```
1.0 0.8 0.8 0.8 0.8 0.7 0.7 0.7 0.7
1.2 1.0 0.5 1.0 1.4 1.0 0.4 1.0 1.3
1.2 1.4 1.0 0.5 1.0 1.3 1.0 0.4 1.0
1.2 1.0 1.4 1.0 0.5 1.0 1.3 1.0 0.4
1.2 0.5 1.0 1.4 1.0 0.4 1.0 1.3 1.0
1.3 1.0 0.5 1.0 1.6 1.0 0.5 1.0 1.4
1.3 1.6 1.0 0.5 1.0 1.4 1.0 0.5 1.0
1.3 1.0 1.6 1.0 0.5 1.0 1.4 1.0 0.5
1.3 0.5 1.0 1.6 1.0 0.5 1.0 1.4 1.0
```

Current element-specific evidence and event-state corrections are documented in
`docs/wiki/elements.md`. Shared matrix: `internal/element`; all81 float32 bits
checked against immutable original server SHA256
`0fb7c9f13b5bcf186118f8bfe73ab214226b93e8aa502b8e6fded4a8312bc547`.
Item normalization0..8 and mob base normalization0..4 follow native loaders;
normal mob attacks remain neutral. Temporary element fields now follow apply/
remove events, including weapon removal and native restoration quirks. Detection
reads the live defensive snapshot, rather than equipment without active effects.

## Reading the original

Ghidra 12.1.3 opens Eternity's project read-only:

```sh
export JAVA_HOME=~/.cache/phoenix-reverse/ghidra/jdk-21.0.12.1+1
~/.cache/phoenix-reverse/ghidra/ghidra_12.1.3_PUBLIC/support/analyzeHeadless \
  ~/Projects/Eternity/work/ghidra-12.1.3 Eternity -process ps_game.exe \
  -readOnly -noanalysis -scriptPath tools/ghidra -postScript Decomp.java "CUser::AttackDamage"
```

Other scripts in `tools/ghidra`: `ListFuncs.java <substring>` lists functions
with their address and size, `Fields.java <struct>` prints a PDB structure's
fields, and `Insns.java <address>... <filter>` prints a function's
instructions containing the filter text (for example `"0x12"` to see which
`CUser` fields it reads). The PDB's `CUser` field names are shifted against
the shipped binary; field meanings above come from the instruction offsets
(Str 0x1220, Dex 0x1224, Int 0x1228, Wis 0x122c, Rec 0x1230, Luc 0x1234,
HP 0x1238, MP 0x123c, SP 0x1240), cross-checked with `CUser::SetAttack`.


GX item affinity intentionally overrides original conflict-to-neutral merging:
intrinsic element wins, otherwise the first elemental lapis in six-socket order.
Later elemental lapis stay linked/inactive (including bonuses and triggers);
non-elemental lapis remain active. Client socket rows gray the inactive gems.
See docs/wiki/elements.md for this requested rule and its original provenance.

## Mob property audit follow-up

The [field inventory and native evidence](../../docs/mob-properties-research.md)
record outstanding special AI dispatch, stationary attack routing and loader
normalization work. Existing mob rows above describe implemented generic paths,
not complete parity for native FSM/Lua bosses. QuestItemID now uses the native
quest-only type 27/28/29 mapping; it is not Type*1000+TypeID. Client visual Day
scaling is independent of server combat Size.

## Lapis NPC effects and incoming absorption

Lapis Special selects the NpcSkill namespace (SQL level 100, client
Monster/NpcSkill.SData), never the same ID in player Skills. High-Special
Lv6/Lv7 debuffs now use their NPC-defined success, duration and effect values.
Restrictions expire, sleep wakes on real HP damage, and HP/SP/MP periodic
loss changes the appropriate resources. NPC and player effect identities stay
separate. Defensive state-resistance lapis and Endure durability are excluded.
Direct mob skills apply flat absorption before their result packet and drain
healing; periodic damage is not absorbed. PvP skill result/drain values also
reflect absorption already applied to the defender.
