package main

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"slices"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/cash"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/studio"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type assets struct {
	storeIcons map[uint32]image.Image
	root       string
	items      map[sdata.ItemKey]*sdata.Item
	ids        []uint32
	atlases    map[string]image.Image
	warning    string
}

func loadAssets(root string) *assets {
	a := &assets{root: root, atlases: map[string]image.Image{}}
	path, err := studio.FindFile(root, "Item/Item.SData")
	if err == nil {
		var raw []byte
		raw, err = os.ReadFile(path)
		if err == nil {
			a.items, err = sdata.ParseItems(raw)
		}
	}
	if err != nil {
		a.warning = "Item previews unavailable • " + err.Error()
	}
	for key := range a.items {
		a.ids = append(a.ids, uint32(key.Type)*1000+uint32(key.TypeID))
	}
	slices.Sort(a.ids)
	return a
}

func (a *assets) item(id uint32) *sdata.Item {
	if id/1000 > 255 || id%1000 > 255 {
		return nil
	}
	return a.items[sdata.ItemKey{Type: uint8(id / 1000), TypeID: uint8(id % 1000)}]
}

func (a *assets) itemTitle(id uint32) string {
	if def := a.item(id); def != nil {
		return fmt.Sprintf("%d · %s", id, def.Name)
	}
	return fmt.Sprint(id)
}

func (a *assets) productIcon(p cash.Product) image.Image {
	if p.Icon == 0 {
		for _, item := range p.Items {
			if item.ItemID != 0 {
				return a.itemIcon(item.ItemID)
			}
		}
	}
	file, uv, ok := ui.MallIcon(p.Icon)
	if ok {
		return a.crop(file, uv)
	}
	return nil
}

func (a *assets) usableStoreIcon(id uint32) image.Image {
	if a.storeIcons == nil {
		a.storeIcons = make(map[uint32]image.Image)
		for _, base := range []uint32{1, 1001} {
			tiles := make([]*image.NRGBA, 256)
			for i := range tiles {
				tiles[i], _ = a.productIcon(cash.Product{Icon: base + uint32(i)}).(*image.NRGBA)
			}
			last := tiles[len(tiles)-1]
			end := len(tiles)
			// The stock atlas pads unused trailing cells with the same
			// opaque empty-slot artwork. Keep isolated duplicate icons.
			if last != nil {
				for end > 0 && tiles[end-1] != nil && bytes.Equal(tiles[end-1].Pix, last.Pix) {
					end--
				}
				if len(tiles)-end < 8 {
					end = len(tiles)
				}
			}
			for i, tile := range tiles[:end] {
				if tile != nil && hasVisiblePixels(tile) {
					a.storeIcons[base+uint32(i)] = tile
				}
			}
		}
	}
	return a.storeIcons[id]
}

func hasVisiblePixels(tile *image.NRGBA) bool {
	for i := 3; i < len(tile.Pix); i += 4 {
		if tile.Pix[i] != 0 {
			return true
		}
	}
	return false
}

func (a *assets) itemIcon(id uint32) image.Image {
	if item := a.item(id); item != nil {
		file, uv, ok := ui.ItemIcon(item.Type, item.Icon)
		if ok {
			return a.crop(file, uv)
		}
	}
	return nil
}

func (a *assets) crop(file string, uv ui.UV) image.Image {
	atlas, cached := a.atlases[file]
	if !cached {
		path, err := studio.FindFile(a.root, "interface/"+file)
		if err == nil {
			f, err := os.Open(path)
			if err == nil {
				config, err := png.DecodeConfig(f)
				if err == nil && config.Width <= 4096 && config.Height <= 4096 {
					if _, err := f.Seek(0, 0); err == nil {
						atlas, _ = png.Decode(f)
					}
				}
				f.Close()
			}
		}
		a.atlases[file] = atlas
	}
	if atlas == nil {
		return nil
	}
	bounds := atlas.Bounds()
	w, h := float32(bounds.Dx()), float32(bounds.Dy())
	region := image.Rect(int(uv.U0*w), int(uv.V0*h), int(uv.U1*w), int(uv.V1*h)).Add(bounds.Min)
	tile := image.NewNRGBA(image.Rect(0, 0, region.Dx(), region.Dy()))
	draw.Draw(tile, tile.Bounds(), atlas, region.Min, draw.Src)
	return tile
}

func includes(text, search string) bool {
	return strings.Contains(strings.ToLower(text), search)
}
