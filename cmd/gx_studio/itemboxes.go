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
	"github.com/Spelunkern/Shaiya-GX/internal/itemboxes"
	"github.com/Spelunkern/Shaiya-GX/internal/studio"
	"github.com/Spelunkern/Shaiya-GX/internal/worlddrops"
)

func (e *editor) itemBoxCatalog() {
	catalog := e.itemBoxes
	draft := itemboxes.Clone(catalog.Rules)
	type entryKey struct {
		box      int16
		position int
		chance   bool
	}
	invalid := map[entryKey]string{}
	e.databaseDraftDirty = func() bool { return len(invalid) > 0 || !itemboxes.Equal(draft, catalog.Rules) }
	e.status = wrapped("")
	details := container.NewStack(wrapped("Select a box or add an index."))
	save := widget.NewButtonWithIcon("Save database", theme.DocumentSaveIcon(), func() {
		if e.busy {
			return
		}
		pending := itemboxes.Clone(draft)
		details.Hide()
		e.run("Backing up and saving item boxes…", func() func() {
			backup, err := studio.SaveItemBoxes(context.Background(), e.root, catalog, pending)
			return func() {
				if e.showError(err) {
					details.Show()
					return
				}
				e.itemBoxes = &studio.ItemBoxCatalog{DBPath: catalog.DBPath, Rules: pending}
				e.itemBoxCatalog()
				e.status.SetText("Saved. Restart GX World to apply. Backup · " + backup)
			}
		})
	})
	save.Importance = widget.HighImportance
	refreshSave := func() {
		save.Disable()
		if len(invalid) > 0 {
			e.status.SetText("Correct the invalid item IDs, grades or chances before saving.")
		} else if err := itemboxes.Validate(draft); err != nil {
			e.status.SetText(err.Error())
		} else {
			e.status.SetText("")
			if e.databaseDraftDirty() {
				save.Enable()
			}
		}
	}
	list := widget.NewList(func() int { return len(draft) }, func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(fmt.Sprintf("%d · %s", draft[i].Index, draft[i].Description))
		})
	add := widget.NewButtonWithIcon("Add box", theme.ContentAddIcon(), func() {
		if e.busy {
			return
		}
		index := widget.NewEntry()
		d := dialog.NewForm("Add item box", "Add", "Cancel", []*widget.FormItem{widget.NewFormItem("Index (ReqVg)", index)}, func(ok bool) {
			if !ok {
				return
			}
			n, err := strconv.ParseInt(strings.TrimSpace(index.Text), 10, 16)
			if err != nil || n < 1 {
				e.showError(fmt.Errorf("enter an index from 1 to 32767"))
				return
			}
			i, exists := slices.BinarySearchFunc(draft, int16(n), func(b itemboxes.Box, id int16) int { return int(b.Index) - int(id) })
			if exists {
				e.showError(fmt.Errorf("box %d already exists", n))
				return
			}
			list.UnselectAll()
			draft = slices.Insert(draft, i, itemboxes.Box{Index: int16(n), Description: "New box", BoxType: itemboxes.Normal, ItemType: itemboxes.Grade})
			list.Refresh()
			list.Select(i)
			refreshSave()
		}, e.window)
		d.Show()
	})
	var render func(int)
	render = func(i int) {
		b := &draft[i]
		id := b.Index
		description := inspectorEntry(b.Description)
		description.OnChanged = func(text string) { b.Description = text; list.RefreshItem(i); refreshSave() }
		roll := widget.NewSelect([]string{"Normal (one reward)", "Independent rolls"}, nil)
		roll.SetSelectedIndex(int(b.BoxType) - 1)
		roll.OnChanged = func(string) {
			mode := uint8(roll.SelectedIndex() + 1)
			if b.BoxType == mode {
				return
			}
			dialog.ShowConfirm("Change roll mode?", "Changing between weights and percentages resets the reward chances to zero.", func(ok bool) {
				if ok {
					b.BoxType = mode
					for j := range b.Rewards {
						b.Rewards[j].Chance = 0
						delete(invalid, entryKey{id, j, true})
					}
				}
				render(i)
				refreshSave()
			}, e.window)
		}
		target := widget.NewSelect([]string{"Grade", "Item ID"}, nil)
		target.SetSelectedIndex(int(b.ItemType) - 1)
		target.OnChanged = func(string) { b.ItemType = uint8(target.SelectedIndex() + 1); render(i); refreshSave() }
		rows := container.NewVBox()
		live := make([]*widget.Label, len(b.Rewards))
		refreshChances := func() {
			total := b.Total()
			for j, r := range b.Rewards {
				switch {
				case b.BoxType == itemboxes.Independent:
					live[j].SetText(worlddrops.FormatChance(r.Chance) + " %")
				case total > 0:
					live[j].SetText(fmt.Sprintf("%.8g %%", 100*float64(r.Chance)/float64(total)))
				default:
					live[j].SetText("—")
				}
			}
		}
		input := func(key entryKey, value string, set func(string) bool) *widget.Entry {
			if raw, ok := invalid[key]; ok {
				value = raw
			}
			entry := inspectorEntry(value)
			entry.OnChanged = func(text string) {
				if set(strings.TrimSpace(text)) {
					delete(invalid, key)
				} else {
					invalid[key] = text
				}
				refreshChances()
				refreshSave()
			}
			return entry
		}
		itemLabel, chanceLabel := "Grade", "Weight (0–1000)"
		if b.ItemType == itemboxes.Direct {
			itemLabel = "Item ID"
		}
		if b.BoxType == itemboxes.Independent {
			chanceLabel = "Chance (%)"
		}
		rows.Add(container.NewGridWithColumns(4, fieldHeading(itemLabel, "Item"), fieldHeading(chanceLabel, "Chance"),
			widget.NewLabel("Effective chance"), widget.NewLabel("Position")))
		for j, r := range b.Rewards {
			item := input(entryKey{id, j, false}, strconv.FormatInt(int64(r.Item), 10), func(text string) bool {
				n, err := strconv.ParseInt(text, 10, 32)
				if err != nil || n < 1 {
					return false
				}
				b.Rewards[j].Item = int32(n)
				return true
			})
			chanceText := strconv.FormatUint(r.Chance, 10)
			if b.BoxType == itemboxes.Independent {
				chanceText = worlddrops.FormatChance(r.Chance)
			}
			chance := input(entryKey{id, j, true}, chanceText, func(text string) bool {
				var n uint64
				var err error
				if b.BoxType == itemboxes.Independent {
					n, err = worlddrops.ParseChance(text)
				} else {
					n, err = strconv.ParseUint(text, 10, 64)
				}
				if err != nil || (b.BoxType == itemboxes.Normal && n > 1000) {
					return false
				}
				b.Rewards[j].Chance = n
				return true
			})
			live[j] = widget.NewLabel("")
			remove := widget.NewButtonWithIcon(strconv.Itoa(j+1), theme.DeleteIcon(), func() {
				for _, chanceField := range []bool{false, true} {
					for k := j; k < len(b.Rewards)-1; k++ {
						key := entryKey{id, k, chanceField}
						if raw, ok := invalid[entryKey{id, k + 1, chanceField}]; ok {
							invalid[key] = raw
						} else {
							delete(invalid, key)
						}
					}
					delete(invalid, entryKey{id, len(b.Rewards) - 1, chanceField})
				}
				b.Rewards = slices.Delete(b.Rewards, j, j+1)
				render(i)
				refreshSave()
			})
			rows.Add(container.NewGridWithColumns(4, item, chance, live[j], remove))
		}
		refreshChances()
		addReward := widget.NewButtonWithIcon("Add reward", theme.ContentAddIcon(), func() {
			b.Rewards = append(b.Rewards, itemboxes.Reward{Item: 1})
			render(i)
			refreshSave()
		})
		deleteBox := widget.NewButton("Remove box", func() {
			dialog.ShowConfirm("Remove item box?", fmt.Sprintf("Remove configuration %d and its rewards?", id), func(ok bool) {
				if !ok {
					return
				}
				for k := range invalid {
					if k.box == id {
						delete(invalid, k)
					}
				}
				list.UnselectAll()
				draft = slices.Delete(draft, i, i+1)
				list.Refresh()
				if len(draft) > 0 {
					list.Select(min(i, len(draft)-1))
				} else {
					details.Objects = []fyne.CanvasObject{wrapped("Select a box or add an index.")}
					details.Refresh()
				}
				refreshSave()
			}, e.window)
		})
		editor := container.NewVBox(fieldHeading(fmt.Sprintf("Box %d (ReqVg)", id), "Index / BoxIndex"),
			fieldHeading("Internal description", "Description"), description,
			container.NewGridWithColumns(2, container.NewVBox(fieldHeading("Roll mode", "BoxType"), roll), container.NewVBox(fieldHeading("Reward target", "ItemType"), target)),
			container.NewHBox(addReward, deleteBox), rows)
		details.Objects = []fyne.CanvasObject{container.NewVScroll(editor)}
		details.Refresh()
	}
	list.OnSelected = func(i widget.ListItemID) {
		if !e.busy {
			render(i)
		}
	}
	header := container.NewVBox(container.NewHBox(widget.NewButton("Open another installation", func() { e.leave(e.start) }), e.switcher(studio.ItemBoxes), save),
		wrapped(catalog.DBPath), wrapped(itemboxes.Table+" · "+itemboxes.RewardsTable))
	split := container.NewHSplit(container.NewBorder(add, nil, nil, nil, list), details)
	split.SetOffset(.29)
	e.window.SetContent(container.NewPadded(container.NewBorder(header, e.status, nil, nil, split)))
	if len(draft) > 0 {
		list.Select(0)
	}
	refreshSave()
}
