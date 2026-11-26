# Client text and SysMsg

The base content catalog is `data/sysmsg-uni.txt` (UTF-16LE).

- **Native IDs** (1–11100) are the original rows.
- **GX IDs** (12001+) and defaults for native rows missing in older files are
  embedded from `internal/data/text/sysmsg-gx.txt`.
  `tools/patch_sysmsg.py <data dir> [--native]` can install these rows in the
  English catalog for editing.

## Translation catalogs

Select a language in User Options and press OK to apply it immediately. Press
OK again after editing the active catalog to reload the changes. English (ID 1) uses
only the base catalog. For other languages, the client opens one file at
`data/Languages/Messages/sysmgtxt-<code>.txt`.

| ID | Language | Code |
| --- | --- | --- |
| 2 | Español | `es` |
| 3 | Português (Brasil) | `pt-BR` |

Saved IDs outside English, Spanish, and Brazilian Portuguese resolve to
English. These IDs remain reserved for future variants.

Use UTF-8 and the same numeric IDs and quoted rows as the English table.
For example, `2<TAB><TAB>"<p> ha muerto"` is a Spanish row for the
English message `<p> died`. Keep substitutions such as `<p>` and `<v>`,
literal `\n` line breaks, and formatting tags. The file can contain any subset
of rows. Missing files and missing, empty or invalid entries use the English
row. Menu labels still use the English catalog. Existing chat lines keep
their original language.

## SData text overlays

The same selector loads UTF-8 CSV overlays from `data/Languages/`. The
Spanish catalog uses the original PS0085 text where it matches GX identities
and manually translated GX text for the remaining usable fields. SData remains
the authority for gameplay and supplies the
English name or description whenever the file, row or field is missing.

| File | Header |
| --- | --- |
| `cash/cash_<code>.csv` | `code,name,description` |
| `item/item_<code>.csv` | `type,id,name,description` |
| `monster/monster_<code>.csv` | `id,name` |
| `npc/npcnames_<code>.csv` | `type,id,name` |
| `npc/npcdescription_<code>.csv` | `type,id,description` |
| `npc/npcquest.csv` | `language,id,name,summary,offer,journal,reminder,alternate,completion1,completion2,completion3` |
| `skills/skills_<code>.csv` | `id,level,name,description` |
| `npcskills/npcskills_<code>.csv` | `id,level,name,description` |

Item IDs are the type and type ID pair. Monster IDs are zero-based SData
indices. NPC IDs are one-based within their type and quest IDs are one-based
GX quest indices. Player skill rows use the exact skill ID and rank from
`Character/Skill.SData`. NPC skill rows use their own IDs from
`Monster/NpcSkill.SData` and rank 1. The packet-level NPC marker 100 is
resolved to rank 1 before lookup. Skill translations change only names and
descriptions. The quest file is shared and only rows with the selected
language code apply. Quote fields containing commas, quotes or line breaks
using standard CSV rules. The next OK in User Options rereads the active
language from disk.

Character and server selection, character creation, and faction selection
read from the selected SysMsg catalog. Button artwork stays shared. Loading
tips can use `Languages/Tips/tip_<code>.ini` in UTF-8. Match the keys in the
`[Tips]` section of `Tip.INI` to replace individual tips. Missing keys use the
English tip.

How code asks for text:

| Place | Call |
| --- | --- |
| App | `a.tr(id)`, `a.msg.Format(id, map[string]string{"v": ...})`, `a.systemMessage(style, id, vals)`, `a.questMessageID(id)`, `a.backToLoginID(id)` |
| UI package | `ui.Tr(id)`, `ui.Trf(id, vals...)` (placeholders `<v>`, `<v2>`...) |

`TestGXSysMsgIDsMatchTheEmbeddedRows` keeps code and rows in step: a GX id used in code needs
a row, and a row nobody uses is rejected.

An internal migration script performed the one-off conversion of helper calls (`backToLogin`, `questMessage`,
`personalMarketNotice`...). Text still in code, deliberately: internal keys compared as text
(`Bid refund`, `Expired`, `Buy:`, `Sell:`), file and section names, fonts and the QA preview fixtures.
