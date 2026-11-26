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
| 4 | Français | `fr` |
| 5 | Deutsch | `de` |
| 6 | Italiano | `it` |
| 7 | Polski | `pl` |
| 8 | Türkçe | `tr` |
| 9 | Русский | `ru` |
| 10 | Tiếng Việt | `vi` |

Use UTF-8 and the same numeric IDs and quoted rows as the English table.
For example, `2<TAB><TAB>"<p> ha muerto"` is a Spanish row for the
English message `<p> died`. Keep substitutions such as `<p>` and `<v>`,
literal `\n` line breaks, and formatting tags. The file can contain any subset
of rows. Missing files and missing, empty or invalid entries use the English
row. Menu labels still use the English catalog. Existing chat lines keep
their original language.

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
