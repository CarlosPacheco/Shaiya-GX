package app

import (
	"image"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func (a *App) auctionArt(file string, src image.Rectangle, x, y, w, h int, alpha float32) bool {
	if a.assets == nil {
		return false
	}
	t := a.assets.Texture("interface/" + file + ".png")
	if t == nil || !src.In(image.Rect(0, 0, t.W, t.H)) {
		return false
	}
	u, v := float32(t.W), float32(t.H)
	a.sprites.Draw(t, float32(x), float32(y), float32(w), float32(h),
		float32(src.Min.X)/u, float32(src.Min.Y)/v, float32(src.Max.X)/u, float32(src.Max.Y)/v, [4]float32{1, 1, 1, alpha})
	return true
}

func (a *App) auctionFrame(x, y, width, height int) {
	file := "auction_objectsearch"
	switch a.auction.view {
	case proto.AuctionSelling, proto.AuctionBidding:
		file = "auction_progress"
	case proto.AuctionItems, proto.AuctionGold:
		file = "auction_completio"
	case proto.AuctionWatch:
		file = "auction_interest_list"
	}
	a.drawAuctionFrameStrip(file, x, y, width, 2, 48)
	a.drawAuctionFrameStrip(file, x, y+height-11, width, 594, 605)
}

func (a *App) drawAuctionFrameStrip(file string, x, y, width, top, bottom int) {
	const sourceLeft, sourceRight, capWidth = 2, 737, 78
	height := bottom - top
	a.auctionArt(file, image.Rect(sourceLeft, top, sourceLeft+capWidth, bottom), x, y, capWidth, height, 1)
	a.auctionArt(file, image.Rect(sourceLeft+capWidth, top, sourceRight-capWidth, bottom),
		x+capWidth, y, width-2*capWidth, height, 1)
	a.auctionArt(file, image.Rect(sourceRight-capWidth, top, sourceRight, bottom),
		x+width-capWidth, y, capWidth, height, 1)
}

// The cells contain their own labels. Fit them inside the GX control bounds.
func (a *App) auctionSpriteButton(file string, col, state, cellW, cellH, artW, artH, x, y, width int) bool {
	b := image.Rect(col*cellW, state*cellH, col*cellW+artW, state*cellH+artH)
	drawW := min(width, min(artW, artW*26/artH))
	drawH := max(1, artH*drawW/artW)
	return a.auctionArt(file, b, x+(width-drawW)/2, y+(26-drawH)/2, drawW, drawH, 1)
}

func (a *App) auctionButtonState(x, y, w int, enabled bool) int {
	if !enabled {
		return 3
	}
	if a.lastMouseX >= x && a.lastMouseX < x+w && a.lastMouseY >= y && a.lastMouseY < y+26 {
		return 1
	}
	return 0
}

func (v *auctionView) auctionActionArt(action string) (file string, col int) {
	switch action {
	case "bid":
		return "auction_objectsearch_button", 0
	case "buy":
		return "auction_objectsearch_button", 1
	case "watch":
		return "auction_objectsearch_button", 2
	case "exact":
		return "auction_objectsearch_button", 3
	case "setup":
		return "auction_progress_button", 0
	case "cancel":
		return "auction_progress_button", 1
	case "collect", "collectall":
		if v.view == proto.AuctionGold {
			col = 2
		}
		if action == "collectall" {
			col++
		}
		return "auction_completion_button", col
	case "refresh":
		return "auction_interestlist_button", 0
	case "details":
		return "auction_interestlist_button", 1
	case "unwatch":
		return "auction_interestlist_button", 2
	case "clearwatch":
		return "auction_interestlist_button", 3
	default:
		return "", 0
	}
}
