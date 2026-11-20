package main

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/Spelunkern/Shaiya-GX/internal/studio"
	"github.com/Spelunkern/Shaiya-GX/internal/teleportrunes"
)

func (e *editor) teleportRuneCatalog() {
	catalog := e.teleportRunes
	draft := slices.Clone(catalog.Rules)
	invalid := make(map[int16]map[string]string)
	e.serverDraftDirty = func() bool { return len(invalid) > 0 || !slices.Equal(catalog.Rules, draft) }
	e.status = wrapped("")
	selectPrompt := wrapped("Select or add a rune.")
	details := container.NewStack(selectPrompt)
	save := widget.NewButtonWithIcon("Save server file", theme.DocumentSaveIcon(), func() {
		if e.busy || len(invalid) > 0 {
			return
		}
		pending := slices.Clone(draft)
		details.Hide()
		e.run("Backing up and saving teleport runes…", func() func() {
			backup, err := studio.SaveTeleportRunes(context.Background(), e.root, catalog, pending)
			return func() {
				if e.showError(err) {
					details.Show()
					return
				}
				catalog.Rules = pending
				e.teleportRuneCatalog()
				e.status.SetText("Saved. Restart GX World to apply. Backup · " + backup)
			}
		})
	})
	save.Importance = widget.HighImportance
	refreshSave := func() {
		save.Disable()
		e.status.SetText("")
		if len(invalid) > 0 {
			e.status.SetText("Map IDs must be integers from 0 to 999. Coordinates must be finite numbers.")
		} else if e.serverDraftDirty() {
			save.Enable()
		}
	}
	list := widget.NewList(func() int { return len(draft) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			r := draft[i]
			obj.(*widget.Label).SetText(fmt.Sprintf("%d  ·  Light %d  ·  Fury %d", r.Index, r.Light.MapID, r.Fury.MapID))
		})
	add := widget.NewButtonWithIcon("Add rune", theme.ContentAddIcon(), func() {
		if e.busy {
			return
		}
		entry := widget.NewEntry()
		entry.SetPlaceHolder("1–32767")
		entry.Validator = func(raw string) error {
			n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 16)
			if err != nil || n < 1 {
				return fmt.Errorf("enter an index from 1 to 32767")
			}
			if slices.ContainsFunc(draft, func(r teleportrunes.Rune) bool { return r.Index == int16(n) }) {
				return fmt.Errorf("index already exists")
			}
			return nil
		}
		dialog.ShowForm("Add teleport rune", "Add", "Cancel",
			[]*widget.FormItem{widget.NewFormItem("Rune index (Index)", entry)}, func(ok bool) {
				if !ok || entry.Validate() != nil {
					return
				}
				n, _ := strconv.ParseInt(strings.TrimSpace(entry.Text), 10, 16)
				index := int16(n)
				i := 0
				for i < len(draft) && draft[i].Index < index {
					i++
				}
				list.UnselectAll()
				draft = slices.Insert(draft, i, teleportrunes.Rune{Index: index})
				list.Refresh()
				list.Select(i)
				refreshSave()
			}, e.window)
	})
	list.OnSelected = func(i widget.ListItemID) {
		if e.busy {
			return
		}
		r := draft[i]
		input := func(label, column, value string, set func(string) bool) fyne.CanvasObject {
			if raw, ok := invalid[r.Index][column]; ok {
				value = raw
			}
			entry := inspectorEntry(value)
			entry.OnChanged = func(raw string) {
				if set(strings.TrimSpace(raw)) {
					delete(invalid[r.Index], column)
					if len(invalid[r.Index]) == 0 {
						delete(invalid, r.Index)
					}
					list.RefreshItem(i)
				} else {
					if invalid[r.Index] == nil {
						invalid[r.Index] = make(map[string]string)
					}
					invalid[r.Index][column] = raw
				}
				refreshSave()
			}
			side := "Fury"
			if strings.HasPrefix(column, "Light") {
				side = "Light"
			}
			return container.NewVBox(fieldHeading(label, side+" · "+strings.TrimPrefix(column, side)), entry)
		}
		destination := func(faction string, d *teleportrunes.Destination) fyne.CanvasObject {
			mapID := input("Map ID", faction+"MapID", strconv.Itoa(int(d.MapID)), func(raw string) bool {
				n, err := strconv.ParseUint(raw, 10, 16)
				if err != nil || n > 999 {
					return false
				}
				d.MapID = uint16(n)
				return true
			})
			coordinate := func(axis string, value *float64) fyne.CanvasObject {
				return input(axis+" coordinate", faction+axis, strconv.FormatFloat(*value, 'g', -1, 64), func(raw string) bool {
					n, err := strconv.ParseFloat(raw, 64)
					if err != nil || !teleportrunes.ValidCoordinate(n) {
						return false
					}
					*value = n
					return true
				})
			}
			return container.NewVBox(widget.NewLabelWithStyle(faction, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				container.NewGridWithColumns(2, mapID, coordinate("X", &d.X), coordinate("Y", &d.Y), coordinate("Z", &d.Z)))
		}
		remove := widget.NewButtonWithIcon("Remove rune", theme.DeleteIcon(), func() {
			dialog.ShowConfirm("Remove teleport rune",
				fmt.Sprintf("Items with Variant %d will lose their destination.", r.Index), func(ok bool) {
					if !ok {
						return
					}
					list.UnselectAll()
					delete(invalid, r.Index)
					draft = slices.Delete(draft, i, i+1)
					list.Refresh()
					details.Objects = []fyne.CanvasObject{selectPrompt}
					details.Refresh()
					if len(draft) != 0 {
						list.Select(min(i, len(draft)-1))
					}
					refreshSave()
				}, e.window)
		})
		editor := container.NewVBox(fieldHeading(fmt.Sprintf("Rune %d", r.Index), fmt.Sprintf("[Rune_%d]", r.Index)),
			wrapped("Items with Special 47 use this index as their Variant (ReqVg)."),
			destination("Light", &draft[i].Light), destination("Fury", &draft[i].Fury),
			container.NewHBox(remove))
		details.Objects = []fyne.CanvasObject{container.NewVScroll(editor)}
		details.Refresh()
	}
	header := container.NewVBox(
		container.NewHBox(widget.NewButton("Open another installation", func() { e.leave(e.start) }), e.switcher(studio.TeleportRunes), save),
		wrapped(catalog.File.Path),
	)
	split := container.NewHSplit(container.NewBorder(add, nil, nil, nil, list), details)
	split.SetOffset(0.28)
	e.window.SetContent(container.NewPadded(container.NewBorder(header, e.status, nil, nil, split)))
	if len(draft) > 0 {
		list.Select(0)
	}
	refreshSave()
}
