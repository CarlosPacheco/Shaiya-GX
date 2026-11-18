package main

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/Spelunkern/Shaiya-GX/internal/data/cash"
)

func editText(value string, multiline bool) *widget.Entry {
	var box *widget.Entry
	if multiline {
		box = widget.NewMultiLineEntry()
		box.SetMinRowsVisible(3)
		box.Wrapping = fyne.TextWrapWord
	} else {
		box = widget.NewEntry()
	}
	box.SetText(value)
	return box
}

func number(value uint64) *widget.Entry { return editText(strconv.FormatUint(value, 10), false) }

type productSlot struct {
	id, count *widget.Entry
}

func (e *editor) editProduct(original cash.Product, apply func(cash.Product)) {
	readOnly := e.project.Encrypted
	name, code := editText(original.Name, false), editText(original.Code, false)
	description := editText(original.Description, true)
	index, cost, icon := number(uint64(original.Index)), number(uint64(original.Cost)), number(uint64(original.Icon))
	section := widget.NewSelect(categories, nil)
	if original.Bag < uint32(len(categories)) {
		section.SetSelectedIndex(int(original.Bag))
	}
	preview := newIcon(48)
	var slots [cash.Slots]productSlot
	readIcon := func() {
		id, _ := strconv.ParseUint(icon.Text, 10, 32)
		p := original
		p.Icon = uint32(id)
		for i, slot := range slots {
			if slot.id != nil {
				item, _ := strconv.ParseUint(slot.id.Text, 10, 32)
				p.Items[i].ItemID = uint32(item)
			}
		}
		showIcon(preview, e.assets.productIcon(p))
	}
	pickIcon := widget.NewButton("Browse icons…", func() {
		e.iconPicker(func(id uint32) { icon.SetText(strconv.FormatUint(uint64(id), 10)) })
	})
	icon.OnChanged = func(string) { readIcon() }
	properties := container.NewVBox(
		field("Name", "Cash.Name  ·  ProductName", name),
		field("Product code", "Cash.Code  ·  ProductCode", code),
		field("Description", "Cash.Description", description),
		field("Section", "Cash.Bag  ·  0 hidden, 1–4 visible", section),
		field("Record index", "Cash.Index", index),
		field("Price in points", "Cash.Cost  ·  BuyCost", cost),
		field("Store icon", "Cash.Icon  ·  0 uses first item", container.NewBorder(nil, nil, preview, pickIcon, icon)),
	)
	tabs := container.NewAppTabs(container.NewTabItem("Product", container.NewVScroll(properties)))
	for start := 0; start < cash.Slots; start += 8 {
		group := container.NewVBox()
		for i := start; i < start+8; i++ {
			item := original.Items[i]
			slot := productSlot{number(uint64(item.ItemID)), number(uint64(item.Count))}
			slots[i] = slot
			itemImage := newIcon(30)
			name := wrapped("")
			update := func(string) {
				id, _ := strconv.ParseUint(slot.id.Text, 10, 32)
				showIcon(itemImage, e.assets.itemIcon(uint32(id)))
				if id == 0 {
					name.SetText("Empty")
				} else if def := e.assets.item(uint32(id)); def != nil {
					name.SetText(fmt.Sprintf("%s  ·  Type %d / TypeID %d", def.Name, def.Type, def.TypeID))
				} else {
					name.SetText("Item definition unavailable")
				}
				readIcon()
			}
			slot.id.OnChanged = update
			update(slot.id.Text)
			pick := widget.NewButton("Find item…", func() {
				e.itemPicker(func(id uint32) {
					slot.id.SetText(strconv.FormatUint(uint64(id), 10))
					if id == 0 {
						slot.count.SetText("0")
					} else if slot.count.Text == "0" {
						slot.count.SetText("1")
					}
				})
			})
			if readOnly || len(e.assets.ids) == 0 {
				pick.Disable()
			}
			if readOnly {
				slot.id.Disable()
				slot.count.Disable()
			}
			idField := field("Item ID", fmt.Sprintf("Cash.Items[%d].ItemID  ·  ItemID%d", i, i+1), slot.id)
			countField := field("Quantity", fmt.Sprintf("Cash.Items[%d].Count  ·  ItemCount%d", i, i+1), slot.count)
			group.Add(widget.NewSeparator())
			group.Add(container.NewBorder(nil, nil, itemImage, pick, name))
			group.Add(container.NewGridWithColumns(2, idField, countField))
		}
		tabs.Append(container.NewTabItem(fmt.Sprintf("Slots %d–%d", start+1, start+8), container.NewVScroll(group)))
	}
	if readOnly {
		for _, box := range []*widget.Entry{name, code, description, index, cost, icon} {
			box.Disable()
		}
		section.Disable()
		pickIcon.Disable()
	}
	readIcon()
	var d *dialog.CustomDialog
	errorText := wrapped("")
	cancel := widget.NewButton("Close", func() { d.Hide() })
	actions := container.NewHBox(cancel)
	if !readOnly {
		cancel.SetText("Cancel")
		button := widget.NewButton("Apply to draft", func() {
			p := original
			p.Name, p.Code, p.Description = name.Text, code.Text, description.Text
			var err error
			read := func(box *widget.Entry, label string, bits int) uint64 {
				if err != nil {
					return 0
				}
				n, parseErr := strconv.ParseUint(strings.TrimSpace(box.Text), 10, bits)
				if parseErr != nil {
					err = fmt.Errorf("%s needs an unsigned %d-bit integer", label, bits)
				}
				return n
			}
			if section.SelectedIndex() < 0 {
				errorText.SetText("Select a section.")
				return
			}
			p.Bag = uint32(section.SelectedIndex())
			p.Index, p.Cost, p.Icon = uint32(read(index, "Index", 32)), uint32(read(cost, "Price", 32)), uint32(read(icon, "Icon", 32))
			for i, slot := range slots {
				p.Items[i] = cash.Item{ItemID: uint32(read(slot.id, fmt.Sprintf("Slot %d item ID", i+1), 32)), Count: uint16(read(slot.count, fmt.Sprintf("Slot %d quantity", i+1), 8))}
			}
			if err == nil {
				err = cash.Validate([]cash.Product{p})
			}
			if err != nil {
				errorText.SetText(err.Error())
				return
			}
			d.Hide()
			apply(p)
		})
		button.Importance = widget.HighImportance
		actions.Add(button)
	}
	d = dialog.NewCustomWithoutButtons("Product fields", container.NewBorder(nil, container.NewVBox(errorText, actions), nil, nil, tabs), e.window)
	d.Resize(fyne.NewSize(840, 655))
	d.Show()
}

func (e *editor) iconPicker(apply func(uint32)) {
	var d *dialog.CustomDialog
	grid := widget.NewGridWrap(func() int { return 513 }, func() fyne.CanvasObject {
		return container.NewVBox(newIcon(36), widget.NewButton("1000", nil))
	}, func(i widget.GridWrapItemID, object fyne.CanvasObject) {
		id := uint32(i)
		if i > 256 {
			id += 744
		}
		children := object.(*fyne.Container).Objects
		showIcon(children[0].(*canvas.Image), e.assets.productIcon(cash.Product{Icon: id}))
		button := children[1].(*widget.Button)
		button.SetText(strconv.FormatUint(uint64(id), 10))
		if id == 0 {
			button.SetText("Auto")
		}
		button.OnTapped = func() {
			d.Hide()
			apply(id)
		}
	})
	d = dialog.NewCustom("Store icons · Cash.Icon", "Cancel", grid, e.window)
	d.Resize(fyne.NewSize(720, 490))
	d.Show()
}

func (e *editor) itemPicker(apply func(uint32)) {
	var d *dialog.CustomDialog
	search := widget.NewEntry()
	search.SetPlaceHolder("Search by ID or item name")
	var ids []uint32
	list := widget.NewList(func() int { return len(ids) }, func() fyne.CanvasObject {
		return container.NewBorder(nil, nil, newIcon(28), nil, widget.NewLabel(""))
	}, func(i widget.ListItemID, object fyne.CanvasObject) {
		children := object.(*fyne.Container).Objects
		children[0].(*widget.Label).SetText(e.assets.itemTitle(ids[i]))
		showIcon(children[1].(*canvas.Image), e.assets.itemIcon(ids[i]))
	})
	list.OnSelected = func(i widget.ListItemID) {
		d.Hide()
		apply(ids[i])
	}
	search.OnChanged = func(query string) {
		query = strings.ToLower(strings.TrimSpace(query))
		ids = ids[:0]
		for _, id := range e.assets.ids {
			if includes(e.assets.itemTitle(id), query) {
				ids = append(ids, id)
			}
		}
		list.Refresh()
	}
	search.OnChanged("")
	empty := widget.NewButton("Clear slot", func() {
		d.Hide()
		apply(0)
	})
	d = dialog.NewCustom("Find item", "Cancel", container.NewBorder(search, empty, nil, nil, list), e.window)
	d.Resize(fyne.NewSize(720, 495))
	d.Show()
}
