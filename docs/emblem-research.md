# Cloak emblem research (EP4.5 server, EP5.4 client)

Evidence for the cloak emblem service: an NPC customises an equipped or carried
cloak with an emblem, for a fixed gold fee. Nothing here changes game behaviour yet.

## Server (EP4.5 `ps_game.exe`, PDB names)

Both packets are handled by `CUser::PacketGem` (`0x0046bd40`), the same 0x08xx
group as gems and lapisia (`0801` gem add, `0802` gem remove, `0803` cloak add,
`0804` cloak remove).

### `0803` apply emblem — `CUser::ItemCloakAdd` (`0x00464ad0`)

Client to server, 10 bytes: `op(2) bag(1) slot(1) data[6]`.

Validation (silent drop on failure): `bag <= bag count`, `slot < 24`, an item
exists there and its ItemDef Type is 24 (cloak; compare with the `0x18` check).
The NPC distance/type is not checked here.

* Gold below **450000**: reply `0803` result 0 and nothing changes.
* Otherwise: item bytes `0x46..0x4b` (the six "gem" bytes) are overwritten with
  `data[6]`, gold is reduced by 450000, the DB agent gets `0711`
  (billing id, bag, slot, 6 bytes, gold) and the client gets the success reply.
  If the item is the equipped cloak (bag 0, slot 7) `SendEquipment` runs, so
  nearby players see the emblem (the `0507`/equipment packet already carries the
  six cloak bytes). A game-log entry `0412` is written.

Server to client reply, 15 bytes:
`op(2)=0803 result(1) bag(1) slot(1) data[6] gold(4)`; result 1 = applied,
0 = not enough gold (data echoed, gold unchanged).

### `0804` remove emblem — `CUser::ItemCloakRemove` (`0x00464d40`)

Client to server: `op(2) bag(1) slot(1)`. Same validation. Zeroes the six bytes,
sends `0711`, replies `0804 bag slot` (4 bytes), no fee, and `SendEquipment` for
bag 0 slot 7.

## Client (EP5.4 `Game.exe`; the EP4.5 client holds the same strings)

* Data: `data/cloak/Emblem/EmblemList.dat`, `EmblemBack.dat`, `EmblemFront.dat`
  (`int32 count`, then `count` records of 260-byte NUL-padded names, 0xCC fill)
  loaded by `FUN_0055ac10`; textures in `EmblemTex/` (emblems, 33 entries here)
  and `BackgroundTex/` (back/front shapes, 20 each).
* UI textures: `emblem.tga` (1024x512), `emblem_select_mark.tga`,
  `emblem_button.tga`, `emblem_slect_arrow.tga` (+ `emblem_button2.tga`).
* Window: `FUN_00480a00` (build), `FUN_0047f400` (update/click), opens as window
  state `0x6d` from `FUN_0048e050` when the clicked NPC (target kind 1, action 2)
  has merchant type **8** (see below). Cost field `0x35cc` = 450000.
* The window picks a cloak from the inventory (client requires it to be Type 24),
  three shape selectors with left/right arrows (front, back, emblem; `-1` = none)
  and a 36-entry colour grid (`0x24`, 6 columns, 34 px cells; palette table at
  `0x007091a8`, 12 bytes per colour) assigning colours to three slots.
* Byte layout sent and stored (`FUN_0059c380`, item bytes `+0x398..0x39d`):
  `[0..2] = selected index + 1` for the three shapes (0 = none), `[3..5]` =
  palette index of each shape (default `0x23`). A shape with index 0 forces its
  colour to `0x23`.
* Buttons: apply sends `0803` when gold >= cost, else SysMsg `0x14f1` (5361); an
  empty window gives `0x14ef` (5359); `0x14ee` (5358) is the level 15 check on
  the dropped cape. Remove sends `0804`.
* Result handlers copy the six bytes back into the local inventory item.

## Confirmed details (all implemented)

* The emblem NPC is a merchant (NPC type 1) with **MerchantType 8** ("Badge
  merchant", SysMsg 0x14f3, e.g. Allen Igla and Shuti Ceraiss); the window is
  Game.exe state `0x6d`.
* Byte order: `[0] front, [1] back, [2] emblem` (index + 1, 0 none) and
  `[3] front colour, [4] back colour, [5] emblem colour`. Front reads
  `EmblemFront.dat` and back reads `EmblemBack.dat`, both from `BackgroundTex/`;
  the emblem reads `EmblemList.dat` from `EmblemTex/`.
* Palette: 36 RGB triples at `0x007091a8`, index 35 white (the default). They
  are now `internal/data/emblem`.
* Window art `emblem.png` is 652x512. Item slot 80x78 at (0x218,0x3a), cape icon
  at (0x230,0x51), pattern squares 64x64 at x 0x123 and y 0x57/0xe4/0x171,
  arrows every 0x27 from x 0x123 at y 0x9b/0x128/0x1b5, palette 6x6 of 32 px
  swatches every 34 from (0x199,0xdd), Combine/Delete at (0x1a,0x1c7) and
  (0x94,0x1c7), price right-aligned at x 0xf3, y 0x198 in the GoldRGB colour.
* Dropping a cape needs type 24 or 39 and character level 15 or more (SysMsg
  5358). An empty window answers Combine with 5359 and too little gold with
  5361. The server only accepts type 24 and never checks the NPC.
* In-world render (`0x55a4d0`): nine cloth nodes around the centre given by
  four ints of the cloak `.PC` record at offsets 0x18..0x24 (column, row, column
  span, row span; `[2 2 1 1]` or `[2 3 1 1]`) form a 3x3 grid with UV 0/0.5/1
  and the native 24-index buffer. Three alpha-blended layers (back, front,
  emblem) are drawn in that order, each multiplied by its palette colour.
  The patch is about 15 cm wide on every shipped cape. Cloths whose shoulder
  piece covers the patch (model 0 for example) hide it exactly as designed.
