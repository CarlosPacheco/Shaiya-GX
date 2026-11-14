# Shaiya GX gx-v0.1.7 — stable world surfaces, complete rune variants and reliable Item Mall

Update client and server together. Native Linux and Windows client/server
archives include checksums. Client archives include licensed external fonts;
proprietary game artwork, maps and audio remain separate. Use the updated loose
client data definitions with this release. Preserve the existing server runtime
database when upgrading; the distributed development seed contains intentional
test accounts and is not an empty production database.

## World and character rendering

- Fix black armor patches on normal NPCs by applying clockwise back-face
  culling to opaque skinned-model draws, with explicit raster-state cleanup.
- Fix wood walls and other world surfaces alternating between light and dark
  while the camera moves. Authored front/back sheets can occupy the same plane
  with different triangle diagonals. Identify their opposite coplanar coverage
  and draw only the correct facing sheet, preventing depth and lighting fights.
- Preserve unpaired two-sided sheets, cutout vegetation, terrain, original UVs,
  textures, normals and collision geometry. Mixed parts use compact buffers;
  streamed CPU preparation, instancing and LOD retain the same surface contract.
  No texture replacements, depth-bias workaround or anti-aliasing setting is
  required.

## Independent gameplay windows

- Inventory, Status, Skills and Actions can stay open together. Hotkeys and
  bottom-bar buttons use the same independent toggles.
- New frames use available horizontal space while manually moved frames retain
  their positions. Overlapping controls, title dragging and Escape respect the
  drawing order; Escape closes one panel at a time without clicking through to
  a covered window or the world.

## Recreation runes and blacksmith feedback

- Add nine stat-removal runes for STR, DEX, REC, INT, WIS, LUC, HP, MP and SP,
  plus a general remover that clears all nine recreation stats.
- Removal preserves base equipment properties, enchantment, socketed lapis,
  quality and item identity. An absent selected stat, an already empty item or
  malformed recreation data is rejected without consuming a rune.
- Complete the 28 GX variants: nine random, nine perfect, nine stat removers
  and one general remover. All have dedicated free 255-unit Item Mall products.
- Repair the 27 older variants still sold in legacy rune bundles. Client and
  server now agree on their recreation Special and explicit ReqVg selectors.
  Clear obsolete requirements and intrinsic bonuses inherited from the former
  stat/mode encoding, including residues on the ordinary and GX runes.
- Insufficient gold uses the original operation-specific system text for
  linking, extraction and enhancement, with the required amount. Rejected
  attempts consume nothing; paid random failures remain distinct from lack of
  funds.
- Show the lapisia's static gold cost in yellow beneath the enhancement
  instructions, without waiting for a server quote. Share the original ReqIg
  cost table between client and server and align outdated client selectors.
- Successful recreation displays only the native `Used <item>` notice. Remove
  temporary Before, After and recreated-attributes chat messages.

## Item Mall and inventory safety

- Synchronize all 310 products, their 24 bundle slots and 255-unit stackable
  supplies with client and server definitions. Correct obsolete quantity text
  and bundles that previously demanded hundreds of inventory slots. Equipment
  remains individual pieces, including bundles containing several copies.
- Commit point deductions, purchase logs and complete deliveries atomically.
  Gifts likewise commit their charge and full contents together. Failed
  delivery, insufficient balance or gift-box capacity rolls back the operation.
- Split gifted equipment bundles into individual entries. Claiming an older
  unsplit bundle preserves its remainder instead of deleting undelivered pieces.
- Reject invalid, empty, duplicate and missing-item product definitions during
  server startup. Offline content tools validate disposable copies, retain
  backups and preserve accounts, characters, inventory and unrelated tables.

## Portables and upgrading

- Publish fresh native client and server packages for Linux and Windows,
  including GX Server GUI and console, startup scripts, server maps, the updated
  development seed, notices, licenses and checksums.
- Support a combined `Cliente/` and `Servidor/` portable folder without copying
  local client game resources (`tools/package_bundle.py --without-data`).
- Older loose resource packs need updated Item.SData, Cash.SData and matching
  stack limits. The explicit offline tools `update_recreation_content.py` and
  `sync_mall_content.py` synchronize selected data directories and databases;
  stop affected components first and keep their backups. Do not overwrite a
  personal runtime with the distributed seed.

## Validation

Linux validation includes unit tests, integration vetting, server integration,
Python storage/content tools, race checks, real-asset OpenGL tests and the full
isolated N2 client/server probe. Dedicated encrypted probes cover all 55 rune
variants through Item Mall purchase, blacksmith use, exact stat changes,
consumption and relog persistence, plus purchase/gift/claim quantities for all
310 products. Graphics regressions cover camera motion at the map 1 market,
both depth modes, paired surfaces, cutout holes, unpaired foliage and state
cleanup. Native Windows/Linux CI builds and validates relocated server portables;
CI without proprietary game assets does not establish Windows visual parity.
