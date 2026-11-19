# Auction House

Talk to an auction board to access the realm-wide gold market. Boards of subtype `117` share their listings across both factions, all zones and instances. Map 42 already has four Notice Boards. For another existing type-1 NPC, set the merchant subtype in the NPC definition using GX Studio. The field label shows `MerchantType`. Deploy the matching NPC data and NPC placement to the client and server.

| Tab | Actions |
|---|---|
| Search Items | Search by item or seller name, filter by category and level, sort active listings, look up the exact item from a selected result, bid, buy or watch |
| Auction Progress | View current sales and bids in separate lists. Open Setup Auction, drag an inventory stack onto the item field and enter quantity, prices and duration. Cancel your own listing here. |
| Ended Auctions | View item and gold claims in separate lists. Collect a selected result or collect all of that kind. Claims that do not fit in the inventory or wallet remain available. |
| Interest List | Review up to ten watched listings per character. Refresh, inspect, remove one or clear the list. |

The market requires a live character near the board they last interacted with. Item data and prices in the window update from the server. The app shows a confirmation before posting, bidding, buying or canceling. The world server rejects a bid, buy or cancel against a stale listing version. Reopen the listing to review its new price or status.

The client reads the auction PNG atlases from `data/interface` when available. The local import uses the sixteen distinct `auction_*.tga` resources as loose PNGs, preserving the original filename stems, including `auction_completio` and `auction_buy_search_botton`. GX uses the extracted frames and control states within its existing panel layout. Missing atlases fall back to GX-rendered controls.

## Listing and settlement

Listings require a buyout higher than the starting price. The starting price is 500–3,800,000,000 gold and the buyout is at most 4,000,000,000. The duration choices are 7, 24, 72 and 168 hours. The listing deposit is five gold per duration hour per started 500,000 gold of starting price.

Bidding requires at least 110% of the current price, rounded down, and less than the buyout. The first bid uses the starting price as its basis. The current bidder has the full bid held in custody. Being outbid creates a gold claim. Buyout charges the full buyout price and places any existing bid in the refund queue, including a bid from the buyer.

A sale creates an item claim for the buyer and a gold claim for the seller containing proceeds and the deposit, less a 2% commission on proceeds. Cancellation or expiry without a bid gives the item back as a claim and consumes the deposit. Canceling a listing with a bid also queues a full refund. When a listing ends by time with a bid, the highest bidder wins.

GX writes a 14-day claim deadline for continuity with the legacy market. Overdue assets remain claimable while the original purge rules are unresolved. Item claims use an available bag slot or sufficient room in matching stacks. Bulk collection processes up to 100 results in a request and reports any attempted results that remain.

## Storage and operations

Migration 23 creates `GX_Auctions`, `GX_AuctionClaims` and `GX_AuctionWatch` inside the normal GX SQLite runtime. It preserves the original `PS_GameData_Market*` tables. The GX runtime uses the new tables and does not import legacy listings. Use normal SQLite backups to preserve market state with characters and inventory.

Posting and collection change the character's wallet or inventory and the market in a single transaction. Pending item ownership is recorded by its unique item UID. Bidding/buyout validates the persisted listing version in the transaction. A unique posting nonce prevents duplicate posting, while settlement state and claim ownership prevent duplicate awards. The expiry worker makes an initial pass at startup and then processes overdue listings in bounded batches.

GX client and server exchange bounded requests and paginated replies under opcode `0x7F1F`. Use matching client and server builds. See the [original market research](auction-house-research.md) for the legacy UI, packets and recovered rules.


The Search Items category rail expands to show item families. Scroll over
the rail to see additional choices and over the results to page the listing
results. The expanded selections filter by catalog item type before paging.
