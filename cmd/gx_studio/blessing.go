package main

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/Spelunkern/Shaiya-GX/internal/blessingrules"
	"github.com/Spelunkern/Shaiya-GX/internal/studio"
)

func (e *editor) blessingCatalog() {
	catalog := e.blessing
	draft := slices.Clone(catalog.Rules)
	e.status = wrapped("")
	type inputKey struct{ tier, field int }
	invalid := make(map[inputKey]string)
	e.databaseDraftDirty = func() bool {
		return len(invalid) != 0 || !slices.Equal(draft, catalog.Rules)
	}
	var active int
	var render func()
	details := container.NewStack()
	save := widget.NewButtonWithIcon("Save database", theme.DocumentSaveIcon(), func() {
		if e.busy {
			return
		}
		pending := slices.Clone(draft)
		details.Hide()
		e.run("Backing up and saving blessing tiers…", func() func() {
			backup, err := studio.SaveBlessing(context.Background(), e.root, catalog, pending)
			return func() {
				if e.showError(err) {
					details.Show()
					return
				}
				e.blessing = &studio.BlessingCatalog{DBPath: catalog.DBPath, Rules: pending}
				e.blessingCatalog()
				e.status.SetText("Saved. Restart GX World to apply. Backup · " + backup)
			}
		})
	})
	save.Importance = widget.HighImportance
	save.Disable()
	refreshSave := func() {
		if len(invalid) != 0 {
			e.status.SetText("Enter whole numbers in the allowed range before saving.")
		} else if err := draft.Validate(); err != nil {
			e.status.SetText(err.Error())
		} else {
			e.status.SetText("")
			if e.databaseDraftDirty() {
				save.Enable()
				return
			}
		}
		save.Disable()
	}
	list := widget.NewList(func() int { return len(draft) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, item fyne.CanvasObject) {
			item.(*widget.Label).SetText(fmt.Sprintf("%2d    %d", i+1, draft[i].RequiredKills))
		})
	render = func() {
		index := active
		values := make([]*widget.Label, len(blessingrules.Effects))
		refreshPreview := func() {
			effective := draft.Effective(index)
			for i, out := range values {
				if out != nil {
					out.SetText(strconv.Itoa(int(effective[i])) + " " + blessingrules.Effects[i].Unit)
				}
			}
		}
		entry := func(field int, initial uint64, bits int, set func(uint64)) *widget.Entry {
			key := inputKey{index, field}
			text := strconv.FormatUint(initial, 10)
			if bad, ok := invalid[key]; ok {
				text = bad
			}
			w := inspectorEntry(text)
			w.OnChanged = func(text string) {
				n, err := strconv.ParseUint(strings.TrimSpace(text), 10, bits)
				if err != nil || (field < 0 && n == 0) {
					invalid[key] = text
				} else {
					delete(invalid, key)
					set(n)
					refreshPreview()
				}
				refreshSave()
			}
			return w
		}
		threshold := entry(-1, uint64(draft[index].RequiredKills), 32, func(n uint64) {
			draft[index].RequiredKills = uint32(n)
			list.RefreshItem(index)
		})
		rows := container.NewVBox(
			fieldHeading(fmt.Sprintf("Tier %d", index+1), "Tier"),
			fieldHeading("Kills required", "RequiredKills"), threshold,
			wrapped("The same rules apply to Light and Fury. Effective bonuses include the highest value from earlier tiers."),
		)
		group := ""
		for i, effect := range blessingrules.Effects {
			if group != effect.Group {
				group = effect.Group
				rows.Add(widget.NewSeparator())
				rows.Add(widget.NewLabelWithStyle(group, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
				rows.Add(container.NewGridWithColumns(3, widget.NewLabel("Bonus"), widget.NewLabel("Defined in this tier"), widget.NewLabel("Effective total")))
			}
			current := entry(i, uint64(draft[index].Bonuses[i]), 16, func(n uint64) {
				draft[index].Bonuses[i] = uint16(n)
			})
			values[i] = widget.NewLabel("")
			rows.Add(container.NewGridWithColumns(3, fieldHeading(effect.Label, fmt.Sprintf("%s · Type %d", effect.Column, i+1)), current, values[i]))
		}
		refreshPreview()
		details.Objects = []fyne.CanvasObject{container.NewVScroll(rows)}
		details.Refresh()
	}
	list.OnSelected = func(i widget.ListItemID) {
		if !e.busy {
			active = i
			render()
		}
	}
	left := container.NewBorder(widget.NewLabel("Tier   Required kills"), nil, nil, nil, list)
	top := container.NewVBox(
		container.NewHBox(widget.NewButton("Open another installation", func() { e.leave(e.start) }), e.switcher(studio.Blessing), save),
		wrapped(catalog.DBPath),
		wrapped(blessingrules.Table),
	)
	split := container.NewHSplit(left, details)
	split.SetOffset(0.22)
	e.window.SetContent(container.NewPadded(container.NewBorder(top, e.status, nil, nil, split)))
	list.Select(0)
}
