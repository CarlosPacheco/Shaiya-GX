# Client text and SysMsg

All user-visible text of the client lives in SysMsg tables, never in code, so a
language is a single `sysmsg-uni.txt`:

- **Native ids** (1..11100) are the original rows. Code asks for them by id and prefers them.
- **GX ids** (12001+) are rows the client adds where the original has no text. They are
  listed in `internal/data/text/sysmsg-gx.txt` (same row format, UTF-8), embedded in the
  executable as defaults, and written into the data folder's `sysmsg-uni.txt` by
  `tools/patch_sysmsg.py <data dir> [--native]` so they can be translated in place.
  The same file also holds default text for native ids an older table may lack
  (`--native` writes those too).
- `sysmsg-uni.txt` always wins; the embedded rows only answer for ids it lacks
  (`SysMsg.Lookup`, `App.tr`, `ui.Tr`).

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
