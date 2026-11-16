# Mob properties: database, Monster.SData and original runtime

Audit date: 2026-11-16. This is a field inventory and a bounded parity audit,
not a claim that every native AI or combat path has been reproduced.

## Evidence and authority

| Source | Identification |
| --- | --- |
| Original server with PDB | `ps_game.exe`, SHA-256 `0fb7c9f13b5bcf186118f8bfe73ab214226b93e8aa502b8e6fded4a8312bc547` |
| Original EP5 client | `Game.exe`, SHA-256 `8e5ef14a2a3bfef15bae61b03d185e551b6c3fd44d09a2861d57428400bb3263` |
| Native server definitions | `CGameData::LoadConstMob` `0x004c1720`; PDB `stMOBInfo`, 268 bytes |
| Native client definitions | `0x00461470`, lookup `0x00460900`, runtime record 44 bytes |
| Native visual scale | `0x00431869..0x004318ea` within `0x00431770`; scaling matrix helper `0x00410470` |
| AI dispatch | `CMob::Init` `0x00490040`, `CAIFactory::SetFSM` `0x004a9130`, `CMob::InitLuaFSM` `0x004b4c20` |

The server decides health, damage, drops, AI and movement. The client uses its
own localized definitions, model/animation references and presentation metadata.
Shared-looking fields need not contain identical values. The audited binaries
and local DB/assets span different episodes; differences alone do not establish
which definition should be changed. No data import or DB rewrite was performed.

## Complete Monster.SData payload

Each record has a length-prefixed name followed by 31 bytes; its record index
is the MobID. Offsets below begin after the name. The native loader copies these
into a 44-byte runtime structure, inserting alignment gaps.

| Payload offset | Width | Field | GX use / finding |
| --- | --- | --- | --- |
| 0 | 2 | Model | Index into `monster.MON`; not the MobID |
| 2 | 2 | Level | Client definition / presentation; server level remains authoritative |
| 4 | 1 | AI | Now retained; does not drive client combat AI |
| 5 | 4 | HP | Retained; live server health takes precedence |
| 9 | 1 | Day / ScaleCode | **Previously discarded; native model scale restored** |
| 10 | 1 | Size | Combat reach metadata; independent of visual scaling |
| 11 | 1 | Attrib | Client elemental metadata, normalized by `MobElement` |
| 12 / 16 | 4 / 1 | NormalTime / NormalStep | Walking interpolation metadata |
| 17 / 21 | 4 / 1 | ChaseTime / ChaseStep | Running interpolation metadata; GX authoritative duration packets take precedence |
| 22 / 24 / 26 | 1 each | AttackType1/2/3 | Retained attack metadata; does not replace server skill selection |
| 23 / 25 / 27 | 1 each | Attack animation selectors | Select MON attack clips, separately from skill IDs |
| 28 | 1 | AttackPlus3 | Now retained; no additional client behavior claimed |
| 29 | 2 | QuestItemID | Now retained; drops remain server-owned |

`Day` does not mean a day/night spawn restriction in this client. Runtime byte
`+0x10` selects uniform scale: `0=.6`, `1=1`, `2=1.3`, `3=1.6`, `4=2`, `5=5`,
`6=.4`, otherwise `1`. Native number height is `MON.Height * scale + .2`.
The server loader explicitly clears its own `byDay`; importing client scaling
into server combat size would be incorrect.

GX now applies the scale before world placement, without scaling position or
accumulating it across frames. Skinning, bounds, picking, shadows and bone
attachments use the same actor transform. NPC scaling is unchanged. Death and
respawn preserve the established stationary-corpse and fixed spawn-effect rules.

## Database field inventory

`PS_GameDefs_Mobs` has 70 columns: RowID plus 69 definition columns. The following
groups cover all of them; a consumed field does not imply all native edge cases
have been validated.

| Columns | Native meaning / GX audit |
| --- | --- |
| RowID; MobID; MobName | SQL surrogate; template ID; server name. Localized client name/model comes from SData/MON |
| Level, Exp, HP, SP, MP | Level, reward and resource maxima. GX uses HP live and MP/SP maxima in formulas; MP/SP are not evidence of native resource spending |
| AI | 1/3 acquire enemies. **10..99 may attach a native FSM; >=100 initializes Lua FSM. GX does not implement these dispatch systems** |
| Money1, Money2 | Base plus random addition, not minimum/maximum endpoints; GX already uses this convention |
| QuestItemID | **Quest-only index, previously decoded incorrectly in GX; corrected below** |
| Dex, Wis, Luc, Defense, Magic | Combat stats; PDB/compiler labels alone are insufficient evidence for formulas; see original-parity combat entries |
| Day, Size, Attrib | Server loader clears Day; Size affects reach; Attrib is normalized elemental identity |
| ResistState1..15, ResistSkill1..6 | Status / skill immunity lists. PDB stores skill IDs as bytes; GX retains wider DB values. Audit truncation before changing behavior for IDs >255 |
| NormalTime, NormalStep, ChaseTime, ChaseStep, ChaseRange | Movement timing/step and acquisition radius, separate from the return-home leash |
| AttackType1/2/3 | Zero is basic attack; otherwise native loader resolves a level-100 mob skill and admits specific effect categories. GX needs a further audit of rejected/missing skill definitions |
| AttackTime1/2/3 | Slot timing; AttackTime3 also supplies shield defense |
| Attackrange1/2/3 | Reach metadata. Native loader caps enabled basic-attack range at 60; GX currently retains the DB range |
| Attack1/2/3, Attackplus1/2/3 | Damage/variance metadata; slot 3 is overloaded: Attack3 recovery, Attackplus3 obelisk marker |
| Attackattrib1/2/3 | First two elemental metadata; third maps country |
| Attackspecial1/2/3 | First two retained codes require a dedicated use-site audit; third selects classic respawn delay |
| Attackok1/2/3 | Slot enabled/conditions; GX scans 3→1 with HP windows and per-slot cooldowns; code 6 retargets weakest eligible enemy |

### Corrected quest index

Native loader maps `1..1000` to type 27, `1001..2000` to type 28, and larger
positive values to type 29. The respective index offset is 0, 1000 or 2000;
TypeID is the low byte of that result, as in the executable. This is not the
ordinary `Type*1000+TypeID` encoding.

Examples: mob 108 uses index 248 → item `(27,248)`; mob 216 uses 1001 → `(28,1)`.
GX previously looked these up as `(0,248)` and `(1,1)`. The corrected lookup
retains quest eligibility and item-quality chance checks. The audited local
seed contains 271 definitions with a positive quest index.

## Outstanding discrepancies and next investigations

1. **Special AI dispatch:** the local seed has 84 definitions with AI 10..99
   and nine with AI >=100. Native initialization selects an FSM or Lua route;
   GX currently runs generic mob behavior instead. Examples include mob 2464
   (AI 111). Exact states, transitions, inputs and script assets must be recovered
   before implementing these bosses; changing aggression alone is insufficient.
2. **Stationary attack routing:** native `CMob::AttackAI` `0x004a1ab0` treats
   AI=4, ChaseStep=0 or an obelisk as stationary. GX's shared `stationary()` checks
   both movement steps for zero and also serves event population placement.
   Eight local obelisk-marked definitions have nonzero movement metadata.
   Audit combat and event placement separately before changing that helper.
3. **Skill admission and basic range normalization:** reproduce LoadConstMob's
   accepted categories, rejected-slot behavior and the 60-unit cap using native
   fixtures; do not silently classify an unknown skill as a physical attack.
4. **Cross-source definitions:** the seed has 2,747 definitions; client SData
   has 3,087 entries including placeholders. Among overlapping client entries
   with positive HP and level, there are 228 HP, 125 level, 217 Size and 213 AI
   differences (each count is independent). Example mob 63: server HP 110000,
   client HP 36500. These are dataset differences, not proven GX logic bugs.
   Size discrepancies deserve particular attention because client approach
   distance can disagree with server reach. Preserve server authority and
   inspect episode/spawn coverage before synchronizing data.
5. **Remaining client metadata:** AI, AttackPlus3 and the quest tail are now
   retained rather than discarded; their native use sites still need individual
   tracing. No speculative effects or combat behavior were added for them.

## Validation and limits

The scale switch and number-height arithmetic were executed in an isolated x86
emulator for codes 0..7, stopping before the graphics call. Native constants agree
with GX. A GL test loads eight original monster models spanning all codes,
checks scaled posed bounds and name anchors, and complements transform tests
for repeated placements. World tests cover quest index boundaries, all three
quest item families and eligibility. Existing movement and stationary corpse /
spawn-effect GL regressions remain applicable.

Local evidence is in ignored `work/mob-properties-audit/`: Ghidra output, native
scale oracle, field export and test logs. Proprietary binaries/data are not
included. This audit did not execute the full original server/client or establish
dynamic parity for the outstanding special AI paths.
