-- Every consumable stacks to 255 (lapis, lapisia, potions, runes, etin and boxes).
-- Equipment, summon stones, quest items (types 27-29, relic pieces) keep their stack.
UPDATE PS_GameDefs_Items SET Count=255
WHERE Type IN (25,30,43,44,94,95,100,101) AND Count<>255
  AND ItemID NOT IN (25073,25074,25075,25076,25077,25078,25079,25180,25235,25236);
