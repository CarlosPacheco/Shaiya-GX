package main

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/Spelunkern/Shaiya-GX/internal/studio"
	"github.com/Spelunkern/Shaiya-GX/internal/worlddrops"
)

func (e *editor) worldDropCatalog() {
	catalog := e.worldDrops
	draft := slices.Clone(catalog.Rules)
	type fieldKey struct {
		id     int64
		column string
	}
	invalid := make(map[fieldKey]string)
	e.databaseDraftDirty = func() bool { return len(invalid) != 0 || !slices.Equal(catalog.Rules, draft) }
	e.status = wrapped("")
	details := container.NewStack(wrapped("Select or add a rule."))
	save := widget.NewButtonWithIcon("Save database", theme.DocumentSaveIcon(), func() {
		if e.busy {
			return
		}
		pending := slices.Clone(draft)
		details.Hide()
		e.run("Backing up and saving world drops…", func() func() {
			backup, err := studio.SaveWorldDrops(context.Background(), e.root, catalog, pending)
			return func() {
				if e.showError(err) {
					details.Show()
					return
				}
				e.worldDrops = &studio.WorldDropCatalog{DBPath: catalog.DBPath, Rules: pending}
				e.worldDropCatalog()
				e.status.SetText("Saved. Restart GX World to apply. Backup · " + backup)
			}
		})
	})
	save.Importance = widget.HighImportance
	refreshSave := func() {
		save.Disable()
		if len(invalid) > 0 {
			e.status.SetText("Enter valid numbers and a percentage with up to 8 decimal places.")
		} else if err := worlddrops.Validate(draft); err != nil {
			e.status.SetText(err.Error())
		} else {
			e.status.SetText("")
			if e.databaseDraftDirty() {
				save.Enable()
			}
		}
	}
	list := widget.NewList(func() int { return len(draft) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			r := draft[i]
			obj.(*widget.Label).SetText(fmt.Sprintf("%d · %s", r.Index, r.Description))
		})
	nextID := int64(0)
	if len(draft) > 0 {
		nextID = draft[len(draft)-1].Index
	}
	add := widget.NewButtonWithIcon("Add rule", theme.ContentAddIcon(), func() {
		if e.busy || nextID == math.MaxInt64 {
			return
		}
		nextID++
		draft = append(draft, worlddrops.Rule{Index: nextID, Description: "New rule", Grade: 1, MinLevel: 1, MaxLevel: 100})
		list.Refresh()
		list.Select(len(draft) - 1)
		refreshSave()
	})
	list.OnSelected = func(i widget.ListItemID) {
		if e.busy {
			return
		}
		r := draft[i]
		input := func(label, column, value string, set func(string) bool) fyne.CanvasObject {
			key := fieldKey{r.Index, column}
			if raw, ok := invalid[key]; ok {
				value = raw
			}
			entry := inspectorEntry(value)
			entry.OnChanged = func(text string) {
				if set(text) {
					delete(invalid, key)
				} else {
					invalid[key] = text
				}
				refreshSave()
			}
			return container.NewVBox(fieldHeading(label, column), entry)
		}
		integer := func(label, column string, value uint16, set func(uint16)) fyne.CanvasObject {
			return input(label, column, strconv.Itoa(int(value)), func(text string) bool {
				n, err := strconv.ParseUint(strings.TrimSpace(text), 10, 16)
				if err != nil {
					return false
				}
				set(uint16(n))
				return true
			})
		}
		remove := widget.NewButtonWithIcon("Remove rule", theme.DeleteIcon(), func() {
			if e.busy {
				return
			}
			for key := range invalid {
				if key.id == r.Index {
					delete(invalid, key)
				}
			}
			list.UnselectAll()
			draft = slices.Delete(draft, i, i+1)
			list.Refresh()
			if len(draft) > 0 {
				list.Select(min(i, len(draft)-1))
			} else {
				details.Objects = []fyne.CanvasObject{wrapped("Select or add a rule.")}
				details.Refresh()
			}
			refreshSave()
		})
		editor := container.NewVBox(
			fieldHeading(fmt.Sprintf("Rule %d", r.Index), "Index"),
			input("Internal description", "Description", r.Description, func(text string) bool {
				draft[i].Description = text
				list.RefreshItem(i)
				return true
			}),
			integer("Item grade", "Grade", r.Grade, func(n uint16) { draft[i].Grade = n }),
			container.NewGridWithColumns(2,
				integer("Minimum mob level", "MinLevel", r.MinLevel, func(n uint16) { draft[i].MinLevel = n }),
				integer("Maximum mob level", "MaxLevel", r.MaxLevel, func(n uint16) { draft[i].MaxLevel = n }),
			),
			input("Drop chance (%)", "DropChance", worlddrops.FormatChance(r.DropChance), func(text string) bool {
				n, err := worlddrops.ParseChance(text)
				if err != nil {
					return false
				}
				draft[i].DropChance = n
				return true
			}),
			wrapped("0 disables the rule. The smallest positive chance is 0.00000001%."),
			container.NewHBox(remove),
		)
		details.Objects = []fyne.CanvasObject{container.NewVScroll(editor)}
		details.Refresh()
	}
	header := container.NewVBox(
		container.NewHBox(widget.NewButton("Open another installation", func() { e.leave(e.start) }), e.switcher(studio.WorldDrops), save),
		wrapped(catalog.DBPath), wrapped(worlddrops.Table),
	)
	split := container.NewHSplit(container.NewBorder(add, nil, nil, nil, list), details)
	split.SetOffset(0.35)
	e.window.SetContent(container.NewPadded(container.NewBorder(header, e.status, nil, nil, split)))
	if len(draft) > 0 {
		list.Select(0)
	}
	refreshSave()
}
