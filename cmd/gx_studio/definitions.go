package main

import (
	"context"
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
	"github.com/Spelunkern/Shaiya-GX/internal/studio"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func (e *editor) definitionCatalog() {
	e.showDefinitionSection(-1, "", false, -1)
}

func (e *editor) showDefinitionSection(sectionKey int, queryText string, linkedOnly bool, focus int) {
	catalog := e.definitions
	e.status = wrapped("")
	search := widget.NewEntry()
	search.SetPlaceHolder("Search by name or ID")
	search.SetText(queryText)
	paired := widget.NewCheck("Linked SQLite rows only", nil)
	paired.SetChecked(linkedOnly)
	sections, sectionRows, group := definitionSections(catalog, e.definitionDraft)
	if sectionKey != -1 {
		if _, exists := sectionRows[sectionKey]; !exists {
			sectionKey = -1
		}
	}
	keys := make([]int, len(sections))
	keys[0] = -1
	for key, row := range sectionRows {
		keys[row] = key
	}
	var visible []int
	selected := -1
	details := wrapped("Select a definition to inspect its fields.")
	preview := newIcon(48)
	edit := widget.NewButton("Edit fields", nil)
	edit.Disable()
	draft := widget.NewLabel("")
	save := widget.NewButtonWithIcon("Save both", theme.DocumentSaveIcon(), func() {
		if !e.busy && len(e.definitionDraft) > 0 {
			e.reviewDefinitions()
		}
	})
	save.Importance = widget.HighImportance

	name := func(index int) string {
		if changed, ok := e.definitionDraft[index]["Name"]; ok {
			return changed
		}
		return catalog.Records[index].Name
	}
	icon := func(index int) image.Image {
		r := catalog.Records[index]
		iconID := r.Icon
		if changed, ok := e.definitionDraft[index]["Icon"]; ok {
			n, _ := strconv.ParseUint(strings.TrimSpace(changed), 10, 16)
			iconID = uint32(n)
		}
		return e.definitionIcon(catalog.Kind, r.ID, iconID)
	}
	label := func(index int) string {
		r := catalog.Records[index]
		if catalog.Kind == studio.Items {
			return fmt.Sprintf("%d · %s", r.ID, name(index))
		}
		return fmt.Sprintf("%d / %d · %s", r.ID, r.Level, name(index))
	}

	list := widget.NewList(func() int { return len(visible) }, func() fyne.CanvasObject {
		label := widget.NewLabel("")
		label.Truncation = fyne.TextTruncateEllipsis
		return container.NewBorder(nil, nil, newIcon(26), nil, label)
	}, func(row widget.ListItemID, obj fyne.CanvasObject) {
		index := visible[row]
		ch := obj.(*fyne.Container).Objects
		ch[0].(*widget.Label).SetText(label(index))
		showIcon(ch[1].(*canvas.Image), icon(index))
	})
	showDetails := func(index int) {
		selected = index
		r := catalog.Records[index]
		showIcon(preview, icon(index))
		text := label(index)
		if r.DatabaseRow {
			switch catalog.Kind {
			case studio.Items:
				text += fmt.Sprintf("\n\nItemID %d  ·  Type %d  ·  TypeID %d", r.ID, r.ID/1000, r.ID%1000)
			case studio.NPCSkills:
				text += fmt.Sprintf("\n\nSkillID %d  ·  SkillLevel 100 in SQLite", r.ID)
			default:
				text += fmt.Sprintf("\n\nSkillID %d  ·  SkillLevel %d in SQLite", r.ID, r.Level)
			}
		} else {
			text += "\n\nNo unique SQLite row. Client-only fields are available."
		}
		if fields := e.definitionDraft[index]; len(fields) > 0 {
			text += fmt.Sprintf("\n\n%d draft field changes", len(fields))
		}
		details.SetText(text)
		edit.Enable()
	}
	list.OnSelected = func(i widget.ListItemID) { showDetails(visible[i]) }
	updateDraft := func() {
		draft.SetText(fmt.Sprintf("%d edited definitions", len(e.definitionDraft)))
		if len(e.definitionDraft) == 0 {
			save.Disable()
		} else {
			save.Enable()
		}
	}
	edit.OnTapped = func() {
		if selected < 0 || e.busy {
			return
		}
		index := selected
		e.editDefinition(index, func(values map[string]string) {
			if len(values) == 0 {
				delete(e.definitionDraft, index)
			} else {
				e.definitionDraft[index] = values
			}
			e.showDefinitionSection(sectionKey, search.Text, paired.Checked, index)
		})
	}
	filter := func() {
		list.UnselectAll()
		selected = -1
		edit.Disable()
		showIcon(preview, nil)
		details.SetText("Select a definition to inspect its fields.")
		query := strings.ToLower(strings.TrimSpace(search.Text))
		visible = visible[:0]
		for index, r := range catalog.Records {
			if (sectionKey == -1 || group[index] == sectionKey) && (!paired.Checked || r.DatabaseRow) && includes(label(index), query) {
				visible = append(visible, index)
			}
		}
		list.Refresh()
	}
	search.OnChanged = func(string) { filter() }
	paired.OnChanged = func(bool) { filter() }
	filter()
	for row, index := range visible {
		if index == focus {
			list.Select(row)
			break
		}
	}
	updateDraft()

	left := container.NewBorder(container.NewBorder(nil, nil, nil, paired, search), nil, nil, nil, list)
	right := container.NewBorder(container.NewHBox(preview, edit), nil, nil, nil, container.NewVScroll(details))
	split := container.NewHSplit(left, right)
	split.SetOffset(.66)
	top := container.NewVBox(container.NewHBox(
		widget.NewButton("Open another installation", func() { e.leave(e.start) }),
		e.switcher(catalog.Kind), save, draft),
		wrapped(catalog.Path+"\n"+catalog.DBPath+"  ·  "+catalog.Kind.Table()))
	if catalog.Encrypted {
		top.Add(small("SEED encrypted  ·  Encoding preserved on save"))
	}
	title, source := "Effect types", "SData.TypeEffect"
	if catalog.Kind == studio.Items {
		title, source = "Item types", "SData.Type"
	}
	e.window.SetContent(container.NewPadded(container.NewBorder(top, e.status, nil, nil,
		sectionBrowser(title, source, sections, sectionRows[sectionKey], func(row int) {
			sectionKey = keys[row]
			filter()
		}, split))))
}

func (e *editor) definitionIcon(kind studio.Kind, id, icon uint32) image.Image {
	var file string
	var uv ui.UV
	var ok bool
	if kind == studio.Items {
		if icon > 255 {
			return nil
		}
		file, uv, ok = ui.ItemIcon(uint8(id/1000), uint8(icon))
	} else {
		if icon > 65535 {
			return nil
		}
		file, uv, ok = ui.SkillIcon(uint16(icon))
	}
	if !ok {
		return nil
	}
	return e.assets.crop(file, uv)
}

func (e *editor) editDefinition(index int, apply func(map[string]string)) {
	catalog := e.definitions
	record := catalog.Records[index]
	server, err := catalog.SQLValues(context.Background(), index)
	if e.showError(err) {
		return
	}
	tabs := container.NewAppTabs()
	sections := make(map[string]*fyne.Container)
	inputs := make(map[string]*widget.Entry)
	for _, f := range catalog.Kind.Fields() {
		section := sections[f.Section]
		if section == nil {
			section = container.NewVBox()
			sections[f.Section] = section
			tabs.Append(container.NewTabItem(f.Section, container.NewVScroll(section)))
		}
		original := catalog.Value(index, f)
		value, edited := e.definitionDraft[index][f.Source]
		if !edited {
			value = original
		}
		input := editText(value, f.Source == "Description")
		inputs[f.Source] = input
		source := catalog.Kind.RelativePath() + " · " + f.Source
		if f.Column != "" {
			source += "  ·  " + f.Column
		}
		var fieldInput fyne.CanvasObject = input
		if f.Source == "Icon" {
			preview := newIcon(48)
			input.OnChanged = func(text string) {
				id, _ := strconv.ParseUint(strings.TrimSpace(text), 10, 16)
				showIcon(preview, e.definitionIcon(catalog.Kind, record.ID, uint32(id)))
			}
			input.OnChanged(input.Text)
			fieldInput = container.NewBorder(nil, nil, preview, nil, input)
		}
		disabled := f.Column != "" && !record.DatabaseRow
		if options := readableValues(f.Source); len(options) != 0 {
			choice := widget.NewSelect(options, nil)
			choice.PlaceHolder = "Choose a known value"
			input.OnChanged = func(raw string) {
				for _, option := range options {
					value, _, _ := strings.Cut(option, " · ")
					if value == strings.TrimSpace(raw) {
						choice.SetSelected(option)
						return
					}
				}
				choice.ClearSelected()
			}
			input.OnChanged(input.Text)
			choice.OnChanged = func(option string) {
				if value, _, ok := strings.Cut(option, " · "); ok && strings.TrimSpace(input.Text) != value {
					input.SetText(value)
				}
			}
			if disabled {
				choice.Disable()
			}
			fieldInput = container.NewBorder(nil, nil, nil, choice, input)
		}
		if disabled {
			input.Disable()
		}
		section.Add(field(f.Label, source, fieldInput))
		if dbValue, exists := server[f.Column]; exists && dbValue != original {
			section.Add(wrapped("Current SQLite value · " + dbValue))
		}

	}
	var d *dialog.CustomDialog
	errorText := wrapped("")
	confirm := widget.NewButton("Apply to draft", func() {
		changes := make(map[string]string)
		for _, f := range catalog.Kind.Fields() {
			value := inputs[f.Source].Text
			if !f.IsText() {
				value = strings.TrimSpace(value)
				if n, err := strconv.ParseInt(value, 10, 64); err == nil {
					value = strconv.FormatInt(n, 10)
				}
			}
			if value != catalog.Value(index, f) {
				changes[f.Source] = value
			}
		}
		if err := catalog.ValidateEdit(index, changes); err != nil {
			errorText.SetText(err.Error())
			return
		}
		d.Hide()
		apply(changes)
	})
	confirm.Importance = widget.HighImportance
	cancel := widget.NewButton("Cancel", func() { d.Hide() })
	identity := fmt.Sprintf("Skill ID %d · SkillLevel %d", record.ID, record.Level)
	switch catalog.Kind {
	case studio.Items:
		identity = fmt.Sprintf("ItemID %d · Type %d · TypeID %d", record.ID, record.ID/1000, record.ID%1000)
	case studio.NPCSkills:
		if record.Level == 1 {
			identity = fmt.Sprintf("Skill ID %d · SData level 1 → SQLite SkillLevel 100", record.ID)
		}
	}
	d = dialog.NewCustomWithoutButtons(record.Name, container.NewBorder(
		small(identity), container.NewVBox(errorText, container.NewHBox(cancel, confirm)), nil, nil, tabs), e.window)
	d.Resize(fyne.NewSize(860, 655))
	d.Show()
}

func (e *editor) reviewDefinitions() {
	catalog := e.definitions
	var summary []string
	for i, record := range catalog.Records {
		edit := e.definitionDraft[i]
		if len(edit) == 0 {
			continue
		}
		summary = append(summary, fmt.Sprintf("%d / %d · %s", record.ID, record.Level, record.Name))
		for _, f := range catalog.Kind.Fields() {
			if value, ok := edit[f.Source]; ok {
				change := fmt.Sprintf("  %s · SData.%s: %s → %s", f.Label, f.Source, catalog.Value(i, f), value)
				if f.Column != "" {
					stored, err := f.SQLValue(value)
					if e.showError(err) {
						return
					}
					change += fmt.Sprintf("  ·  SQLite.%s → %s", f.Column, stored)
				}
				summary = append(summary, change)
			}
		}
		summary = append(summary, "")
	}

	offline := widget.NewCheck("The affected GX client and server are stopped", nil)
	var d *dialog.CustomDialog
	cancel := widget.NewButton("Cancel", func() { d.Hide() })
	save := widget.NewButton("Apply coordinated save", func() {
		if offline.Checked && !e.busy {
			d.Hide()
			e.saveDefinitions()
		}
	})
	save.Importance = widget.HighImportance
	save.Disable()
	offline.OnChanged = func(checked bool) {
		if checked {
			save.Enable()
		} else {
			save.Disable()
		}
	}
	content := container.NewBorder(nil, container.NewHBox(cancel, offline, save), nil, nil,
		container.NewVScroll(wrapped(strings.Join(summary, "\n"))))
	d = dialog.NewCustomWithoutButtons("Review definition changes", content, e.window)
	d.Resize(fyne.NewSize(900, 510))
	d.Show()
}

func (e *editor) saveDefinitions() {
	catalog, changes := e.definitions, e.definitionDraft
	dataDir, dbPath := e.dataDir, e.dbPath
	e.run("Validating and backing up the definition files…", func() func() {
		ctx := context.Background()
		backup, err := studio.SaveDefinitions(ctx, e.root, catalog, changes)
		saved := err == nil
		var updated *studio.Definitions
		var images *assets
		if saved {
			updated, err = studio.OpenDefinitions(ctx, dataDir, dbPath, catalog.Kind)
			if catalog.Kind == studio.Items && err == nil {
				images = loadAssets(dataDir)
			}
		}
		return func() {
			if err != nil {
				pending, _ := studio.Pending(e.root)
				if pending {
					e.pending = true
					e.showRecovery()
				} else if saved {
					e.start()
				}
				e.showError(err)
				return
			}
			if images != nil {
				e.assets = images
			}
			e.definitions, e.definitionDraft = updated, make(studio.DefinitionEdits)
			e.definitionCatalog()
			e.status.SetText("Saved and verified both catalogs. Backups • " + backup)
		}
	})
}

func readableValues(source string) []string {
	switch source {
	case "Attribute", "AttrType":
		return []string{"0 · Neutral", "1 · Fire I", "2 · Water I", "3 · Earth I", "4 · Wind I", "5 · Fire II", "6 · Water II", "7 · Earth II", "8 · Wind II"}
	case "TypeAttack":
		return []string{"1 · Melee", "2 · Ranged", "3 · Magic"}
	case "TypeEffect":
		return []string{"0 · Passive", "1 · Attack", "2 · Heal", "3 · Debuff", "4 · Buff"}
	case "TargetType":
		return []string{"2 · Self", "3 · Other target", "4 · Party", "5 · Party (alternate)", "6 · Enemies near caster", "7 · Area near target / alternate target"}
	case "SuccessType":
		return []string{"1 · Fixed percent", "2 · Luck-based roll", "3 · Wisdom-based roll"}
	case "FixRange":
		return []string{"0 · Seconds, removed on death", "1 · Hours, persists on death"}
	default:
		return nil
	}
}
