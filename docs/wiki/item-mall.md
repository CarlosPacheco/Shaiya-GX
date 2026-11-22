# Item Mall catalog

The client reads `data/Item/Cash.SData`. Older loose packs can still
use `data/Cash.SData`. The world server reads
`PS_GameDefs_ProductList` from its configured database at startup. SQL determines
prices and the items actually purchased or gifted. Keep product codes, prices
and all 24 bundle slots aligned when editing the two catalogs. Restart the client
and world after changing their respective data. CSV is no longer runtime input;
there is no catalog hash handshake.

Cash.SData retains its native binary layout: product count (uint32), followed by
index/category/icon/price (four uint32 values), 24 item ID (uint32) and quantity
(uint8) pairs, and name/code/description (each byte length uint32 plus text).
Encrypted original SData and plaintext binary tables are supported. Categories
1–4 are visible tabs; category 0 preserves hidden legacy products. Codes fit
20 ASCII bytes. Packed item IDs use type * 1000 + type ID.

The local testing catalog retains all 310 products, including the added supplies
and kits; 226 are visible and every price is zero. Stackable supplies deliver up
to 255 per occupied bundle slot; non-stackable items remain single. Purchase,
gift and claim storage remain authoritative in SQL. Native inventory icons are
used when a testing product has no authored mall icon.

The maximum item stack is 255. The client uses the native `Item.SData`
definitions, and the full portable packager synchronizes their stack fields
with the included SQLite catalog. Quantity packets use native opcodes and
one-byte counts. Internal arithmetic may use
wider integers to avoid overflow; login revision `0x11122202` requires matching
updated client and server binaries.

`tools/restore_native_cash.py` is a one-time importer for a former CSV catalog.
Use explicit stopped database paths and a client data directory. It writes the
native Cash.SData table and matching SQL ProductList, caps catalog entries at
255 and splits existing carried stacks above 255
without losing items. It aborts when there is insufficient room or unsupported
oversized storage. Keep backups before running it. It never replaces a personal
database with the distributed seed. The normal asset importer copies the binary catalog to
`Item/Cash.SData`.

Lapis and lapisias all have a 255 definition stack limit. Every occupied mall
bundle slot containing either family delivers 255. The catalog also contains
28 recreation runes (9 random, 9 perfect, 9 stat removal and 1 general removal), each delivered as 255; see the recreation guide.


Use `tools/sync_mall_content.py` for an explicit offline synchronization of
existing native client tables and named databases. It normalizes stackable
Item.SData definitions and occupied stackable mall slots to 255, exports the
same SQL stack limits, and derives product quantity text from the actual bundle
contents. Equipment remains unstackable; bundles can contain several pieces.
The tool validates all targets on disposable copies before applying changes,
backs up both native tables and each database, and verifies that every table
except Items and ProductList is unchanged. It never replaces a runtime database.
Stop affected clients and services before using it, and restart them afterward:

```sh
python3 tools/sync_mall_content.py \
  --client-data /path/to/client/data \
  --database /path/to/server/data/rapture.sqlite \
  --database /path/to/server/runtime/rapture.sqlite \
  --backup-dir /path/to/new/private-backup
```

Purchases commit the point charge, point log and complete item delivery in one
transaction. Gifts commit the charge and entire gift delivery together. Gifted
equipment bundles occupy one box slot per piece; claims of older unsplit bundles
preserve the remainder for subsequent claims. A failed delivery or insufficient
balance rolls back the entire operation. Invalid, empty, duplicate and unknown
item products are rejected while loading definitions.

The opt-in `TestEntireMallCatalogAgainstIsolatedGX` probe purchases, gifts and
claims all 310 native products over encrypted connections, comparing actual item
identities and totals with Cash.SData and client stack limits. It requires two
disposable GM accounts and `SHAIYA_TEST_ALL_MALL=1`; never use personal accounts.
This checks delivery, quantities and persistence, not every item's gameplay effect.
