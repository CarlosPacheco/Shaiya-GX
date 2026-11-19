package ui

import "github.com/Spelunkern/Shaiya-GX/internal/render"

// PanelScroll is the 0x52c430 scrollbar used by native list windows.
type PanelScroll = panelScroll

func NewPanelScroll(texture *render.Texture, x, y, height int) PanelScroll {
	return newPanelScroll(texture, x, y, height)
}

// NativeClose is the 22px talk1_close_button control at (x, y).
func NativeClose(texture *render.Texture, x, y int) Button { return nativeClose(texture, x, y) }

// ItemNameRGB is the 0x708d24 grade colour table used for item names.
func ItemNameRGB(grade uint8) [3]uint8 {
	if int(grade) >= len(itemGradeRGB) {
		return [3]uint8{255, 255, 255}
	}
	return itemGradeRGB[grade]
}

var itemGradeRGB = [...][3]uint8{
	{255, 255, 255}, {128, 255, 255}, {0, 128, 255}, {0, 255, 64}, {255, 255, 0}, {255, 128, 0},
	{200, 200, 255}, {255, 0, 255}, {128, 0, 255}, {255, 128, 255}, {128, 0, 255},
}
