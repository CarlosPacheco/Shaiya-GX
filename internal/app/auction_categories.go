package app

import "github.com/Spelunkern/Shaiya-GX/internal/proto"

const auctionCategoryY = 138
const auctionCategoryPitch = 26

type auctionCategoryRow struct {
	category, subcategory int
	name                  string
}

func (v *auctionView) categoryRows() []auctionCategoryRow {
	rows := make([]auctionCategoryRow, 0, len(proto.AuctionCategories)+len(proto.AuctionSubcategories[v.category]))
	for i, name := range proto.AuctionCategories {
		rows = append(rows, auctionCategoryRow{category: i, name: name})
		if i == v.category {
			for j, sub := range proto.AuctionSubcategories[i] {
				rows = append(rows, auctionCategoryRow{i, j + 1, sub.Name})
			}
		}
	}
	return rows
}

func auctionCategoryCapacity(height int) int {
	return auctionRowPitch(height) * proto.AuctionPageSize / auctionCategoryPitch
}

func (a *App) updateAuctionCategories(y, height int, wheel float64) {
	v := &a.auction
	rows := v.categoryRows()
	capacity := auctionCategoryCapacity(height)
	lastOffset := max(0, len(rows)-capacity)
	offset := min(v.categoryScroll, lastOffset)
	if wheel != 0 {
		if wheel > 0 {
			v.categoryScroll = max(0, offset-1)
		} else {
			v.categoryScroll = min(lastOffset, offset+1)
		}
		return
	}
	visible := (y - auctionCategoryY) / auctionCategoryPitch
	if visible >= capacity || offset+visible >= len(rows) {
		return
	}
	row := rows[offset+visible]
	if row.category != v.category {
		v.categoryScroll = min(row.category, offset)
	}
	v.category, v.subcategory = row.category, row.subcategory
	v.page, v.selected, v.exactType = 0, -1, 0
	a.requestAuction(proto.AuctionRequest{})
}

func (a *App) drawAuctionCategories(x, y, width, height int) {
	v := &a.auction
	railWidth := auctionListX(proto.AuctionBrowse, width) - 21
	a.font.Draw(a.sprites, "Categories", x+14, y+116, auctionLabelText, railWidth)
	rows := v.categoryRows()
	capacity := auctionCategoryCapacity(height)
	offset := min(v.categoryScroll, max(0, len(rows)-capacity))
	for i, row := range rows[offset:min(offset+capacity, len(rows))] {
		rowY := y + auctionCategoryY + i*auctionCategoryPitch
		selected := row.category == v.category && row.subcategory == v.subcategory
		if row.subcategory > 0 {
			a.drawAuctionControl(row.name, x+24, rowY, railWidth-10, selected, true)
			continue
		}
		state := a.auctionButtonState(x+14, rowY, railWidth, true)
		if selected {
			a.auctionBox(x+14, rowY, railWidth, 26, auctionSelectedColor)
		}
		if row.category == 0 || !a.auctionSpriteButton("auction_completion", row.category-1, state, 128, 32, 127, 28, x+14, rowY, railWidth) {
			a.drawAuctionControl(row.name, x+14, rowY, railWidth, selected, true)
		}
	}
}
