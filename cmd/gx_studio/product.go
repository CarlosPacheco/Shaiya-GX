package main

import (
	"fmt"
	"image"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
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

type productSlot struct {
	id, count *widget.Entry
}

func (e *editor) editProduct(original cash.Product, apply func(cash.Product)) {
	var d *dialog.CustomDialog
	form, _ := e.productForm(original, func(p cash.Product) {
		d.Hide()
		apply(p)
	})
	d = dialog.NewCustom("New product", "Cancel", form, e.window)
	d.Resize(fyne.NewSize(600, 680))
	d.Show()
}

// Fields sit inside the inspector scroll. Disable the Entry's own
// scroller so hovering a value never consumes the inspector's wheel events.
func inspectorEntry(value string) *widget.Entry {
	entry := widget.NewEntry()
	entry.Wrapping = fyne.TextWrapOff
	entry.Scroll = container.ScrollNone
	entry.SetText(value)
	return entry
}

func propertyBox(title, source string, input fyne.CanvasObject) fyne.CanvasObject {
	background := canvas.NewRectangle(theme.Color(propertyFill))
	background.StrokeColor = theme.Color(propertyEdge)
	background.StrokeWidth = 1
	return container.NewStack(background, container.NewPadded(container.NewVBox(small(title+" · "+source), input)))
}

func (e *editor) productForm(original cash.Product, apply func(cash.Product)) (fyne.CanvasObject, func() bool) {
	readOnly := e.project.Encrypted
	name, code := inspectorEntry(original.Name), inspectorEntry(original.Code)
	description := original.Description
	index := inspectorEntry(strconv.FormatUint(uint64(original.Index), 10))
	cost := inspectorEntry(strconv.FormatUint(uint64(original.Cost), 10))
	icon := inspectorEntry(strconv.FormatUint(uint64(original.Icon), 10))
	options := categories[1:]
	firstBag := 1
	if original.Bag == 0 {
		options, firstBag = categories, 0
	}
	section := widget.NewSelect(options, nil)
	if original.Bag < uint32(len(categories)) {
		section.SetSelectedIndex(int(original.Bag) - firstBag)
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
	pickIcon := widget.NewButton("", func() {
		e.iconPicker(func(id uint32) { icon.SetText(strconv.FormatUint(uint64(id), 10)) })
	})
	icon.OnChanged = func(string) { readIcon() }
	iconButton := container.NewStack(pickIcon, preview)
	iconPreview := container.NewBorder(nil, nil, iconButton, nil, widget.NewLabel("Click the icon to choose"))
	if readOnly {
		iconPreview = container.NewBorder(nil, nil, preview, nil, widget.NewLabel("Store icon"))
	}
	descriptionPreview := widget.NewLabel(description)
	descriptionPreview.Truncation = fyne.TextTruncateEllipsis
	editDescription := widget.NewButton("Edit…", func() {
		entry := editText(description, true)
		if readOnly {
			entry.Disable()
			d := dialog.NewCustom("Description · Cash.Description", "Close", entry, e.window)
			d.Resize(fyne.NewSize(540, 260))
			d.Show()
			return
		}
		d := dialog.NewCustomConfirm("Description · Cash.Description", "Apply", "Cancel", entry, func(ok bool) {
			if ok {
				description = entry.Text
				descriptionPreview.SetText(description)
			}
		}, e.window)
		d.Resize(fyne.NewSize(540, 260))
		d.Show()
	})
	if readOnly {
		editDescription.SetText("View…")
	}
	descriptionField := container.NewBorder(nil, nil, nil, editDescription, descriptionPreview)
	properties := container.NewVBox(
		propertyBox("Name", "Cash.Name / ProductName", name),
		container.NewGridWithColumns(2,
			propertyBox("Code", "Cash.Code / ProductCode", code),
			propertyBox("Section", "Cash.Bag", section)),
		container.NewGridWithColumns(3,
			propertyBox("Index", "Cash.Index", index),
			propertyBox("Points", "Cash.Cost / BuyCost", cost),
			propertyBox("Icon ID", "Cash.Icon", icon)),
		propertyBox("Description", "Cash.Description", descriptionField),
	)

	fixed := func(width float32, content fyne.CanvasObject) *fyne.Container {
		return container.NewGridWrap(fyne.NewSize(width, 30), content)
	}
	group := container.NewVBox()
	for i, item := range original.Items {
		slot := productSlot{
			id:    inspectorEntry(strconv.FormatUint(uint64(item.ItemID), 10)),
			count: inspectorEntry(strconv.FormatUint(uint64(item.Count), 10)),
		}
		slots[i] = slot
		itemImage := newIcon(22)
		itemName := widget.NewLabel("")
		itemName.Truncation = fyne.TextTruncateEllipsis
		update := func(string) {
			id, _ := strconv.ParseUint(slot.id.Text, 10, 32)
			showIcon(itemImage, e.assets.itemIcon(uint32(id)))
			switch def := e.assets.item(uint32(id)); {
			case id == 0:
				itemName.SetText("Empty")
			case def != nil:
				itemName.SetText(def.Name)
			default:
				itemName.SetText("Unknown item")
			}
			readIcon()
		}
		slot.id.OnChanged = update
		update(slot.id.Text)
		pick := widget.NewButton("Find", func() {
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
		identity := container.NewHBox(fixed(29, widget.NewLabel(fmt.Sprintf("%02d", i+1))), itemImage)
		inputs := container.NewHBox(fixed(78, slot.id), fixed(56, slot.count), fixed(58, pick))
		group.Add(container.NewBorder(nil, nil, identity, inputs, itemName))
	}
	slotsHeader := container.NewBorder(nil, nil, fixed(55, widget.NewLabel("Slot")),
		container.NewHBox(fixed(78, widget.NewLabel("Item ID")), fixed(56, widget.NewLabel("Qty")), fixed(58, widget.NewLabel(""))),
		widget.NewLabel("Item"))
	slotsPane := container.NewBorder(container.NewVBox(
		slotsHeader, small("Cash.Items[n−1] · SQLite ItemIDn / ItemCountn"), widget.NewSeparator()),
		nil, nil, nil, container.NewVScroll(group))
	tabs := container.NewAppTabs(
		container.NewTabItem("Product", container.NewVScroll(properties)),
		container.NewTabItem("Slots", slotsPane))

	if readOnly {
		for _, box := range []*widget.Entry{name, code, index, cost, icon} {
			box.Disable()
		}
		section.Disable()
		pickIcon.Disable()
	}
	readIcon()
	errorText := wrapped("")
	errorText.Hide()
	showError := func(message string) {
		errorText.SetText(message)
		errorText.Show()
	}
	actions := container.NewHBox()
	if !readOnly {
		button := widget.NewButton("Apply to draft", func() {
			p := original
			p.Name, p.Code, p.Description = name.Text, code.Text, description
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
				showError("Select a section.")
				return
			}
			p.Bag = uint32(section.SelectedIndex() + firstBag)
			p.Index, p.Cost, p.Icon = uint32(read(index, "Index", 32)), uint32(read(cost, "Price", 32)), uint32(read(icon, "Icon", 32))
			for i, slot := range slots {
				p.Items[i] = cash.Item{ItemID: uint32(read(slot.id, fmt.Sprintf("Slot %d item ID", i+1), 32)), Count: uint16(read(slot.count, fmt.Sprintf("Slot %d quantity", i+1), 8))}
			}
			if err == nil && p.Icon != original.Icon && p.Icon != 0 && e.assets.usableStoreIcon(p.Icon) == nil {
				err = fmt.Errorf("icon %d is unavailable in the client atlas", p.Icon)
			}
			if err == nil {
				err = cash.Validate([]cash.Product{p})
			}
			if err != nil {
				showError(err.Error())
				return
			}
			apply(p)
		})
		button.Importance = widget.HighImportance
		actions.Add(button)
	}
	hasPending := func() bool {
		if readOnly {
			return false
		}
		if name.Text != original.Name || code.Text != original.Code || description != original.Description ||
			section.SelectedIndex()+firstBag != int(original.Bag) ||
			index.Text != strconv.FormatUint(uint64(original.Index), 10) ||
			cost.Text != strconv.FormatUint(uint64(original.Cost), 10) ||
			icon.Text != strconv.FormatUint(uint64(original.Icon), 10) {
			return true
		}
		for i, slot := range slots {
			if slot.id.Text != strconv.FormatUint(uint64(original.Items[i].ItemID), 10) ||
				slot.count.Text != strconv.FormatUint(uint64(original.Items[i].Count), 10) {
				return true
			}
		}
		return false
	}
	return container.NewBorder(iconPreview, container.NewVBox(errorText, actions), nil, nil, tabs), hasPending
}

func (e *editor) iconPicker(apply func(uint32)) {
	var d *dialog.CustomDialog
	type choice struct {
		id      uint32
		preview image.Image
	}
	choices := []choice{{}}
	for id := uint32(1); id <= 1256; id++ {
		if preview := e.assets.usableStoreIcon(id); preview != nil {
			choices = append(choices, choice{id, preview})
		}
	}
	grid := widget.NewGridWrap(func() int { return len(choices) }, func() fyne.CanvasObject {
		return container.NewVBox(newIcon(36), widget.NewButton("1000", nil))
	}, func(i widget.GridWrapItemID, object fyne.CanvasObject) {
		selected := choices[i]
		children := object.(*fyne.Container).Objects
		showIcon(children[0].(*canvas.Image), selected.preview)
		button := children[1].(*widget.Button)
		button.SetText(strconv.FormatUint(uint64(selected.id), 10))
		if selected.id == 0 {
			button.SetText("Auto")
		}
		button.OnTapped = func() {
			d.Hide()
			apply(selected.id)
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
