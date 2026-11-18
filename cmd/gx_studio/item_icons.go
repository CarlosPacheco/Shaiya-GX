package main

import (
	"bytes"
	"image"
	"slices"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func (a *assets) itemIconOptions(itemType uint8) map[uint8]image.Image {
	type tile struct {
		id    uint8
		image *image.NRGBA
	}
	sheets := map[string][]tile{}
	for id := 1; id <= 255; id++ {
		file, uv, ok := ui.ItemIcon(itemType, uint8(id))
		if ok {
			pixels, _ := a.crop(file, uv).(*image.NRGBA)
			if pixels != nil {
				sheets[file] = append(sheets[file], tile{uint8(id), pixels})
			}
		}
	}
	options := make(map[uint8]image.Image)
	for _, tiles := range sheets {
		end := len(tiles)
		last := tiles[end-1].image
		for end > 0 && bytes.Equal(tiles[end-1].image.Pix, last.Pix) {
			end--
		}
		if len(tiles)-end < 8 {
			end = len(tiles)
		}
		for _, option := range tiles[:end] {
			if hasVisiblePixels(option.image) {
				options[option.id] = option.image
			}
		}
	}
	return options
}

func (e *editor) pickItemIcon(itemType uint8, apply func(uint8)) {
	options := e.assets.itemIconOptions(itemType)
	ids := make([]uint8, 1, len(options)+1)
	for id := range options {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	var d *dialog.CustomDialog
	grid := widget.NewGridWrap(func() int { return len(ids) }, func() fyne.CanvasObject {
		return container.NewVBox(newIcon(36), widget.NewButton("255", nil))
	}, func(index widget.GridWrapItemID, obj fyne.CanvasObject) {
		id := ids[index]
		children := obj.(*fyne.Container).Objects
		showIcon(children[0].(*canvas.Image), options[id])
		button := children[1].(*widget.Button)
		button.SetText(strconv.Itoa(int(id)))
		if id == 0 {
			button.SetText("None")
		}
		button.OnTapped = func() {
			d.Hide()
			apply(id)
		}
	})
	d = dialog.NewCustom("Inventory icons · SData.Icon", "Cancel", grid, e.window)
	d.Resize(fyne.NewSize(610, 460))
	d.Show()
}
