package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/Spelunkern/Shaiya-GX/internal/studio"
)

var itemFamilies = []string{
	"Light gear", "Fury gear", "Light weapons", "Fury weapons", "Shared weapons",
	"Accessories", "Lapis / Lapisia", "Capes", "Mounts", "Consumables",
	"Quest items", "Pets", "Wings", "Costumes", "Other items",
}

func itemFamily(kind, country uint32) int {
	switch {
	case slices.Contains([]uint32{16, 17, 18, 20, 21, 66, 67, 68, 70, 71, 72, 73, 74, 76, 77}, kind):
		return 0
	case slices.Contains([]uint32{31, 32, 33, 35, 36, 81, 82, 83, 85, 86, 87, 88, 89, 91, 92}, kind):
		return 1
	case slices.Contains([]uint32{1, 2, 14, 19, 45, 46, 49, 51, 53, 55, 60, 62, 64, 69, 75}, kind):
		return 2
	case slices.Contains([]uint32{3, 4, 11, 34, 47, 48, 50, 52, 54, 56, 59, 61, 63, 84, 90}, kind):
		return 3
	case kind >= 5 && kind <= 15 || kind == 57 || kind == 58 || kind == 65:
		if country < 3 {
			return 2
		}
		if country < 6 {
			return 3
		}
		return 4
	case slices.Contains([]uint32{22, 23, 37, 40, 96, 97}, kind):
		return 5
	case kind == 30 || kind == 95 || kind == 98:
		return 6
	case kind == 24 || kind == 39:
		return 7
	case kind == 42:
		return 8
	case slices.Contains([]uint32{25, 38, 41, 43, 44, 78, 79, 80, 94, 100, 101, 102, 103}, kind):
		return 9
	case kind >= 27 && kind <= 29 || kind == 99:
		return 10
	case kind == 120:
		return 11
	case kind == 121:
		return 12
	case kind == 150:
		return 13
	default:
		return 14
	}
}

type itemBrowserTheme struct{ fyne.Theme }

func (t itemBrowserTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameScrollBar, theme.SizeNameScrollBarSmall:
		return 20
	case theme.SizeNameInnerPadding:
		return 5
	case theme.SizeNameSplitThickness:
		return 13
	default:
		return t.Theme.Size(name)
	}
}

func (e *editor) itemCatalog() {
	catalog := e.definitions
	e.itemFormDirty = nil
	e.status = wrapped("")
	fields := make(map[string]studio.Field)
	for _, f := range catalog.Kind.Fields() {
		fields[f.Source] = f
	}
	value := func(index int, source string) string {
		if changed, ok := e.definitionDraft[index][source]; ok {
			return changed
		}
		return catalog.Value(index, fields[source])
	}
	groups := make([][]int, len(itemFamilies))
	for index, record := range catalog.Records {
		country, _ := strconv.ParseUint(value(index, "Country"), 10, 8)
		group := itemFamily(record.ID/1000, uint32(country))
		groups[group] = append(groups[group], index)
	}
	sections := make([]catalogSection, 0, len(groups))
	groupKeys := make([]int, 0, len(groups))
	for index, records := range groups {
		if len(records) != 0 {
			sections = append(sections, catalogSection{itemFamilies[index], len(records)})
			groupKeys = append(groupKeys, index)
		}
	}
	if len(sections) == 0 {
		sections = append(sections, catalogSection{name: "Other items"})
		groupKeys = append(groupKeys, len(itemFamilies)-1)
	}
	section := slices.Index(groupKeys, e.itemSection)
	if section < 0 {
		section = 0
	}
	e.itemSection = groupKeys[section]

	search := widget.NewEntry()
	search.SetPlaceHolder("Search by name or ID")
	search.SetText(e.itemQuery)
	draft := widget.NewLabel(fmt.Sprintf("%d edited items", len(e.definitionDraft)))
	save := widget.NewButtonWithIcon("Save both", theme.DocumentSaveIcon(), func() {
		if e.busy {
			return
		}
		if e.itemFormDirty != nil && e.itemFormDirty() {
			e.status.SetText("Apply the item changes to the draft first.")
			return
		}
		e.reviewDefinitions()
	})
	save.Importance = widget.HighImportance
	if len(e.definitionDraft) == 0 {
		save.Disable()
	}
	inspector := container.NewStack(wrapped("Select an item to edit its fields."))
	selected := -1
	var visible []int
	row := func(header bool) fyne.CanvasObject {
		name := widget.NewLabel("")
		name.Truncation = fyne.TextTruncateEllipsis
		columns := container.NewHBox()
		for i, title := range []string{"Type", "TypeID", "Lvl", "Model", "Icon"} {
			label := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{})
			if header {
				label.SetText(title)
			}
			columns.Add(container.NewGridWrap(fyne.NewSize([]float32{39, 51, 32, 47, 38}[i], 27), label))
		}
		if header {
			name.SetText("Name")
		}
		return container.NewBorder(nil, nil, container.NewGridWrap(fyne.NewSize(22, 27), newIcon(20)), columns, name)
	}
	list := widget.NewList(func() int { return len(visible) }, func() fyne.CanvasObject { return row(false) },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			index := visible[id]
			record := catalog.Records[index]
			parts := obj.(*fyne.Container).Objects
			parts[0].(*widget.Label).SetText(value(index, "Name"))
			iconID, _ := strconv.ParseUint(value(index, "Icon"), 10, 8)
			showIcon(parts[1].(*fyne.Container).Objects[0].(*canvas.Image), e.definitionIcon(studio.Items, record.ID, uint32(iconID)))
			for i, text := range []string{fmt.Sprint(record.ID / 1000), fmt.Sprint(record.ID % 1000), value(index, "MinLevel"), value(index, "Model"), value(index, "Icon")} {
				parts[2].(*fyne.Container).Objects[i].(*fyne.Container).Objects[0].(*widget.Label).SetText(text)
			}
		})
	open := func(index int) {
		form, pending := e.definitionForm(index, true, func(values map[string]string) {
			if len(values) == 0 {
				delete(e.definitionDraft, index)
			} else {
				e.definitionDraft[index] = values
			}
			country, _ := strconv.ParseUint(value(index, "Country"), 10, 8)
			e.itemSection = itemFamily(catalog.Records[index].ID/1000, uint32(country))
			e.itemCatalog()
		})
		if form != nil {
			selected, e.itemFocus, e.itemFormDirty = index, index, pending
			inspector.Objects = []fyne.CanvasObject{form}
			inspector.Refresh()
		}
	}
	restoring := false
	list.OnSelected = func(id widget.ListItemID) {
		index := visible[id]
		if restoring || selected == index {
			return
		}
		if e.itemFormDirty != nil && e.itemFormDirty() {
			dialog.ShowConfirm("Discard unapplied changes?", "Apply to draft to keep the current item edits.", func(ok bool) {
				if ok {
					open(index)
					return
				}
				restoring = true
				list.UnselectAll()
				if prior := slices.Index(visible, selected); prior >= 0 {
					list.Select(prior)
				}
				restoring = false
			}, e.window)
			return
		}
		open(index)
	}
	refresh := func() {
		list.UnselectAll()
		if e.itemFormDirty == nil || !e.itemFormDirty() {
			selected, e.itemFormDirty = -1, nil
			inspector.Objects = []fyne.CanvasObject{wrapped("Select an item to edit its fields.")}
			inspector.Refresh()
		}
		e.itemQuery = search.Text
		query := strings.ToLower(strings.TrimSpace(e.itemQuery))
		visible = visible[:0]
		for _, index := range groups[e.itemSection] {
			record := catalog.Records[index]
			if includes(value(index, "Name"), query) || includes(fmt.Sprint(record.ID), query) {
				visible = append(visible, index)
			}
		}
		list.Refresh()
		if position := slices.Index(visible, e.itemFocus); position >= 0 {
			list.Select(position)
		}
	}
	search.OnChanged = func(string) { refresh() }
	tools := container.NewHBox(widget.NewButton("Open another installation", func() { e.leave(e.start) }),
		e.switcher(studio.Items), save, draft)
	top := container.NewVBox(tools, wrapped(catalog.Path+"\n"+catalog.DBPath+" · "+catalog.Kind.Table()))
	browser := container.NewBorder(container.NewVBox(search, row(true), widget.NewSeparator()), nil, nil, nil, list)
	split := container.NewHSplit(browser, inspector)
	split.SetOffset(.43)
	body := sectionBrowser("Item families", "SData.Type", sections, section, func(row int) {
		e.itemSection = groupKeys[row]
		refresh()
	}, split)
	body.SetOffset(.167)
	e.window.SetContent(container.NewPadded(container.NewBorder(top, e.status, nil, nil,
		container.NewThemeOverride(body, itemBrowserTheme{Theme: theme.Current()}))))
	refresh()
}
