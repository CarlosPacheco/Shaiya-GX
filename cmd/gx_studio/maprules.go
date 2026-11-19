package main

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/Spelunkern/Shaiya-GX/internal/maprules"
	"github.com/Spelunkern/Shaiya-GX/internal/studio"
)

func (e *editor) mapRulesCatalog() {
	catalog := e.mapRules
	clone := func(rows []maprules.Row) []maprules.Row {
		out := slices.Clone(rows)
		for i := range out {
			out[i] = slices.Clone(out[i])
		}
		return out
	}
	draft := maprules.Catalog{Maps: clone(catalog.Rules.Maps), Starts: clone(catalog.Rules.Starts)}
	e.databaseDraftDirty = func() bool { return !reflect.DeepEqual(draft, catalog.Rules) }
	e.status = wrapped("")
	var content *container.AppTabs
	save := widget.NewButtonWithIcon("Save database", theme.DocumentSaveIcon(), func() {
		if e.busy {
			return
		}
		content.Hide()
		e.run("Backing up and saving map rules…", func() func() {
			backup, err := studio.SaveMapRules(context.Background(), e.root, catalog, draft)
			return func() {
				if e.showError(err) {
					content.Show()
					return
				}
				e.mapRules = &studio.MapRulesCatalog{DBPath: catalog.DBPath, Rules: draft}
				e.mapRulesCatalog()
				e.status.SetText("Saved. Restart GX World to apply. Backup · " + backup)
			}
		})
	})
	save.Importance = widget.HighImportance
	save.Disable()
	edit := func(title, table string, row maprules.Row, fields []maprules.Field, first, end int, done func()) {
		panel := container.NewVBox()
		inputs := make([]*widget.Entry, 0, end-first)
		original := make([]string, 0, end-first)
		for i, f := range fields[first:end] {
			label := npcQuestLabel(f.Name)
			switch f.Name {
			case "WeatherType":
				label = "Weather (0 clear, 1 rain, 2 snow)"
			case "WeatherFrequency":
				label = "Weather interval in minutes (0 = 60)"
			case "MinLevel", "MaxLevel":
				label += " (0 = no limit)"
			}
			val := ""
			switch x := row[first+i].(type) {
			case float64:
				val = strconv.FormatFloat(x, 'g', -1, 32)
			case nil:
			default:
				val = fmt.Sprint(x)
			}
			original = append(original, val)
			input := inspectorEntry(val)
			inputs = append(inputs, input)
			panel.Add(container.NewVBox(fieldHeading(label, table+"."+f.Name), input))
		}
		status := wrapped("")
		var popup *dialog.CustomDialog
		apply := widget.NewButton("Apply to draft", func() {
			values := make([]any, len(inputs))
			for i, input := range inputs {
				if input.Text == original[i] {
					values[i] = row[first+i]
					continue
				}
				v, err := maprules.ParseValue(fields[first+i], input.Text)
				if err != nil {
					status.SetText(fields[first+i].Name + ": " + err.Error())
					return
				}
				values[i] = v
			}
			copy(row[first:end], values)
			if e.databaseDraftDirty() {
				save.Enable()
			} else {
				save.Disable()
			}
			done()
			popup.Hide()
		})
		apply.Importance = widget.HighImportance
		popup = dialog.NewCustom(title, "Cancel", container.NewBorder(nil, container.NewVBox(status, apply), nil, nil, container.NewVScroll(panel)), e.window)
		popup.Resize(fyne.NewSize(650, 595))
		popup.Show()
	}
	details := container.NewStack(wrapped("Select a map."))
	search := widget.NewEntry()
	search.SetPlaceHolder("Filter by map ID")
	var rows []int
	list := widget.NewList(func() int { return len(rows) }, func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < len(rows) {
				obj.(*widget.Label).SetText(fmt.Sprintf("Map %v", draft.Maps[rows[i]][0]))
			}
		})
	filter := func() {
		list.UnselectAll()
		rows = rows[:0]
		query := strings.TrimSpace(search.Text)
		for i, row := range draft.Maps {
			if strings.Contains(fmt.Sprint(row[0]), query) {
				rows = append(rows, i)
			}
		}
		list.Refresh()
		details.Objects = []fyne.CanvasObject{wrapped("Select a map.")}
		details.Refresh()
	}
	list.OnSelected = func(i widget.ListItemID) {
		if e.busy || i >= len(rows) {
			return
		}
		row := draft.Maps[rows[i]]
		var refresh func()
		refresh = func() {
			title := fmt.Sprintf("Map %v", row[0])
			fields := maprules.MapFields
			panel := container.NewVBox(fieldHeading(title, "MapID"),
				wrapped(fmt.Sprintf("Level %v–%v · Weather %v", row[4], row[5], row[9])))
			for _, g := range []struct {
				title      string
				first, end int
			}{
				{"Rules and instance", 1, 9}, {"Weather", 9, 11}, {"Map curses", 11, 14}, {"Map modifiers (reserved)", 14, 17},
				{"Light rebirth", 17, 21}, {"Fury rebirth", 21, 25}, {"Instance schedules", 25, len(fields)},
			} {
				panel.Add(widget.NewButton(g.title, func() { edit(g.title, maprules.Properties, row, fields, g.first, g.end, refresh) }))
			}
			details.Objects = []fyne.CanvasObject{container.NewVScroll(panel)}
			details.Refresh()
		}
		refresh()
	}
	search.OnChanged = func(string) { filter() }
	filter()
	split := container.NewHSplit(container.NewBorder(search, nil, nil, nil, list), details)
	split.SetOffset(.24)
	starts := container.NewVBox()
	var showStarts func()
	showStarts = func() {
		starts.Objects = nil
		for i, label := range []string{"Human", "Elf", "Vail", "Death Eater"} {
			row := draft.Starts[i]
			editStart := widget.NewButton("Edit", func() { edit(label, maprules.Starts, row, maprules.StartFields, 1, len(row), showStarts) })
			starts.Add(container.NewBorder(nil, nil, nil, editStart,
				fieldHeading(fmt.Sprintf("%s · Map %v", label, row[1]), fmt.Sprintf("Race %v · X %v · Y %v · Z %v", row[0], row[2], row[3], row[4]))))
		}
		starts.Refresh()
	}
	showStarts()
	content = container.NewAppTabs(container.NewTabItem("Map properties", split),
		container.NewTabItem("Character starts", container.NewVScroll(starts)))
	top := container.NewVBox(container.NewHBox(widget.NewButton("Open another installation", func() { e.leave(e.start) }), e.switcher(studio.MapRules), save),
		wrapped(catalog.DBPath))
	e.window.SetContent(container.NewPadded(container.NewBorder(top, e.status, nil, nil, content)))
}
