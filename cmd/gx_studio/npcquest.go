package main

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/Spelunkern/Shaiya-GX/internal/npcquest"
	"github.com/Spelunkern/Shaiya-GX/internal/studio"
)

var npcGroups = []string{"Stores", "Gatekeepers", "Blacksmiths", "PvP managers", "Gambling houses", "Warehouses",
	"Regular NPCs", "Guards", "Animals", "Apprentice taskmasters", "Guild masters", "Dead NPCs", "Combat commanders"}

type npcQuestEditor struct {
	editor      *editor
	project     *studio.NPCQuestCatalog
	draft       npcquest.Catalog
	texts       map[npcquest.TextKey]string
	refreshSave func()
}

type npcQuestInput struct {
	label, source, value string
	kind                 npcquest.ValueType
	multiline            bool
}

func (e *editor) npcQuestCatalog() {
	q := &npcQuestEditor{editor: e, project: e.npcQuests, draft: npcquest.Clone(e.npcQuests.Doc.Catalog), texts: map[npcquest.TextKey]string{}}
	e.databaseDraftDirty = func() bool { return len(q.texts) > 0 || !npcquest.Equal(q.draft, q.project.Doc.Catalog) }
	e.status = wrapped("")
	tabs := container.NewAppTabs(
		container.NewTabItem("NPCs", q.npcBrowser()),
		container.NewTabItem("Quests", q.questBrowser()),
		container.NewTabItem("Item quest links", q.itemLinks()))
	save := widget.NewButtonWithIcon("Save client and database", theme.DocumentSaveIcon(), func() {
		if e.busy {
			return
		}
		tabs.Hide()
		e.run("Backing up and saving NPCs and quests…", func() func() {
			backup, err := studio.SaveNPCQuest(context.Background(), e.root, q.project, q.draft, q.texts)
			var reopened *studio.NPCQuestCatalog
			if err == nil {
				reopened, err = studio.OpenNPCQuest(context.Background(), e.dataDir, e.dbPath)
			}
			return func() {
				if e.showError(err) {
					tabs.Show()
					return
				}
				e.npcQuests = reopened
				e.npcQuestCatalog()
				e.status.SetText("Saved. Restart GX World and the client to apply. Backup · " + backup)
			}
		})
	})
	save.Importance = widget.HighImportance
	q.refreshSave = func() {
		if e.databaseDraftDirty() {
			save.Enable()
		} else {
			save.Disable()
		}
	}
	q.refreshSave()
	top := container.NewVBox(
		container.NewHBox(widget.NewButton("Open another installation", func() { e.leave(e.start) }), e.switcher(studio.NPCQuest), save),
		wrapped(q.project.Path), wrapped(q.project.DBPath),
	)
	e.window.SetContent(container.NewPadded(container.NewBorder(top, e.status, nil, nil, tabs)))
}

func (q *npcQuestEditor) form(title string, fields []npcQuestInput, apply func([]any) error) {
	entries := make([]*widget.Entry, len(fields))
	rows := container.NewVBox()
	for i, field := range fields {
		var entry *widget.Entry
		if field.multiline {
			entry = widget.NewMultiLineEntry()
			entry.Wrapping = fyne.TextWrapWord
			entry.SetMinRowsVisible(4)
			entry.SetText(field.value)
		} else {
			entry = inspectorEntry(field.value)
		}
		entries[i] = entry
		rows.Add(container.NewVBox(fieldHeading(field.label, field.source), entry))
	}
	status := wrapped("")
	var d *dialog.CustomDialog
	submit := widget.NewButtonWithIcon("Apply to draft", theme.DocumentSaveIcon(), func() {
		values := make([]any, len(fields))
		for i, field := range fields {
			v, err := npcquest.ParseValue(field.kind, entries[i].Text)
			if err != nil {
				status.SetText(fmt.Sprintf("%s: %v", field.label, err))
				return
			}
			values[i] = v
		}
		if err := apply(values); err != nil {
			status.SetText(err.Error())
			return
		}
		q.refreshSave()
		d.Hide()
	})
	submit.Importance = widget.HighImportance
	scroll := container.NewVScroll(rows)
	d = dialog.NewCustom(title, "Cancel", container.NewBorder(nil, container.NewVBox(status, submit), nil, nil, scroll), q.editor.window)
	d.Resize(fyne.NewSize(650, 620))
	d.Show()
}

// Label native field names without changing their exact source identifier.
func npcQuestLabel(name string) string {
	switch name {
	case "NpcName":
		return "NPC name"
	case "QuestName":
		return "Quest name"
	case "NPCTypeID":
		return "Embedded NPC type ID"
	case "NpcWelcome":
		return "NPC dialogue"
	case "Direction":
		return "Direction (0 Start, 1 End)"
	case "QuestRow":
		return "Quest row"
	case "Money":
		return "Price (gold)"
	}
	var b strings.Builder
	var previous rune
	for i, r := range name {
		if i > 0 && (unicode.IsUpper(r) && unicode.IsLower(previous) || unicode.IsDigit(r) && !unicode.IsDigit(previous)) {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
		previous = r
	}
	return b.String()
}

func (q *npcQuestEditor) editRow(title string, table, index, start, end int, done func()) {
	t := npcquest.Tables[table]
	row := q.draft[table][index]
	var fields []npcQuestInput
	var columns []int
	for i := start; i < end; i++ {
		f := t.Fields[i]
		if table == npcquest.NPCs && f.Name == "MerchantType" && row[0] != int64(1) {
			continue
		}
		fields = append(fields, npcQuestInput{npcQuestLabel(f.Name), t.Name + "." + f.Name, npcquest.FormatValue(row[i]), f.Type, false})
		columns = append(columns, i)
	}
	q.form(title, fields, func(values []any) error {
		for i, col := range columns {
			row[col] = values[i]
		}
		done()
		return nil
	})
}

func (q *npcQuestEditor) editTexts(title string, table, index int, names []string, done func()) {
	var fields []npcQuestInput
	for _, name := range names {
		key := npcquest.TextKey{Table: table, Row: index, Field: name}
		text, ok := q.texts[key]
		if !ok {
			text = q.project.Doc.Text(key)
		}
		fields = append(fields, npcQuestInput{npcQuestLabel(name), "NpcQuest.SData · " + name, text, npcquest.Text, true})
	}
	q.form(title, fields, func(values []any) error {
		for i, name := range names {
			key := npcquest.TextKey{Table: table, Row: index, Field: name}
			value := values[i].(string)
			if value == q.project.Doc.Text(key) {
				delete(q.texts, key)
			} else {
				q.texts[key] = value
			}
		}
		done()
		return nil
	})
}

func (q *npcQuestEditor) browser(table int, extra fyne.CanvasObject, keep func(npcquest.Row) bool,
	inspect func(int, func()) fyne.CanvasObject, label func(npcquest.Row) string, register func(func())) fyne.CanvasObject {
	search := widget.NewEntry()
	search.SetPlaceHolder("Search by name or ID")
	var visible []int
	right := container.NewStack(wrapped("Select a row."))
	list := widget.NewList(func() int { return len(visible) }, func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < len(visible) {
				obj.(*widget.Label).SetText(label(q.draft[table][visible[i]]))
			}
		})
	var show func(int)
	show = func(index int) {
		right.Objects = []fyne.CanvasObject{inspect(index, func() { list.Refresh(); show(index) })}
		right.Refresh()
	}
	filter := func() {
		list.UnselectAll()
		visible = visible[:0]
		query := strings.ToLower(strings.TrimSpace(search.Text))
		for i, row := range q.draft[table] {
			if keep(row) && strings.Contains(strings.ToLower(label(row)), query) {
				visible = append(visible, i)
			}
		}
		right.Objects = []fyne.CanvasObject{wrapped("Select a row.")}
		right.Refresh()
		list.Refresh()
	}
	list.OnSelected = func(i widget.ListItemID) {
		if i < len(visible) && !q.editor.busy {
			show(visible[i])
		}
	}
	search.OnChanged = func(string) { filter() }
	header := container.NewVBox(search)
	if extra != nil {
		header.Add(extra)
	}
	if register != nil {
		register(filter)
	}
	filter()
	split := container.NewHSplit(container.NewBorder(header, nil, nil, nil, list), right)
	split.SetOffset(.32)
	return split
}

func (q *npcQuestEditor) npcBrowser() fyne.CanvasObject {
	group := widget.NewSelect(npcGroups, nil)
	group.SetSelectedIndex(0)
	return q.browser(npcquest.NPCs, group,
		func(row npcquest.Row) bool { return row[0] == int64(group.SelectedIndex()+1) },
		q.npcInspector, func(row npcquest.Row) string {
			return fmt.Sprintf("%d · %s", row[1], row[8])
		}, func(refresh func()) { group.OnChanged = func(string) { refresh() } })
}

func (q *npcQuestEditor) npcInspector(index int, done func()) fyne.CanvasObject {
	row := q.draft[npcquest.NPCs][index]
	g, n := row[0].(int64), row[1].(int64)
	title := fmt.Sprintf("%s · %d/%d", row[8], g, n)
	top := container.NewVBox(fieldHeading(title, "NpcGroup / NpcRow"),
		container.NewHBox(
			widget.NewButton("Edit NPC", func() { q.editRow(title, npcquest.NPCs, index, 2, len(row), done) }),
			widget.NewButton("Dialogue", func() { q.editTexts(title, npcquest.NPCs, index, []string{npcquest.NPCWelcome}, done) }),
		))
	tabs := container.NewAppTabs(
		container.NewTabItem("Offered quests", q.linkedRows(npcquest.NPCLinks, []int64{g, n, 0}, done)),
		container.NewTabItem("Received quests", q.linkedRows(npcquest.NPCLinks, []int64{g, n, 1}, done)))
	if g == 1 {
		tabs.Append(container.NewTabItem("Products", q.linkedRows(npcquest.Products, []int64{g, n}, done)))
		tabs.SelectIndex(2)
	}
	if g == 2 {
		tabs.Append(container.NewTabItem("Destinations", q.linkedRows(npcquest.Gates, []int64{g, n}, done)))
		tabs.SelectIndex(2)
	}
	return container.NewBorder(top, nil, nil, nil, tabs)
}

func (q *npcQuestEditor) linkedRows(table int, prefix []int64, done func()) fyne.CanvasObject {
	t := npcquest.Tables[table]
	panel := container.NewVBox()
	var matches []int
	for i, row := range q.draft[table] {
		if hasPrefix(row, prefix) {
			matches = append(matches, i)
		}
	}
	for _, i := range matches {
		row := q.draft[table][i]
		var label string
		switch table {
		case npcquest.Products:
			label = fmt.Sprintf("%d · Item %d/%d", row[2], row[3], row[4])
		case npcquest.Gates:
			label = fmt.Sprintf("%d · Map %d · (%s, %s, %s) · %d gold", row[2], row[3],
				npcquest.FormatValue(row[4]), npcquest.FormatValue(row[5]), npcquest.FormatValue(row[6]), row[7])
		}
		if table == npcquest.NPCLinks || table == npcquest.ItemLinks {
			ref := row[4].(int64)
			label = fmt.Sprintf("%d · Quest %d", row[3], ref)
			if ref > 0 && ref <= int64(len(q.draft[npcquest.Quests])) {
				label += fmt.Sprintf(" · %s", q.draft[npcquest.Quests][ref-1][2])
			}
		}
		actions := container.NewHBox(widget.NewButton("Edit", func() { q.editRow(label, table, i, t.Keys, len(row), done) }))
		if table == npcquest.Gates {
			actions.Add(widget.NewButton("Map label", func() { q.editTexts(label, table, i, []string{"MapName"}, done) }))
		} else {
			actions.Add(widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
				q.draft[table] = slices.Delete(q.draft[table], i, i+1)
				for _, r := range q.draft[table][i:] {
					if !hasPrefix(r, prefix) {
						break
					}
					r[t.Keys-1] = r[t.Keys-1].(int64) - 1
				}
				q.refreshSave()
				done()
			}))
		}
		panel.Add(container.NewBorder(nil, nil, nil, actions, wrapped(label)))
	}
	var add fyne.CanvasObject
	if table != npcquest.Gates {
		add = widget.NewButtonWithIcon("Add row", theme.ContentAddIcon(), func() {
			var inputs []npcQuestInput
			for _, f := range t.Fields[t.Keys:] {
				inputs = append(inputs, npcQuestInput{npcQuestLabel(f.Name), t.Name + "." + f.Name, "0", f.Type, false})
			}
			q.form("Add row", inputs, func(values []any) error {
				var row npcquest.Row
				for _, v := range prefix {
					row = append(row, v)
				}
				row = append(row, int64(len(matches)+1))
				row = append(row, values...)
				at, _ := slices.BinarySearchFunc(q.draft[table], row, func(a, b npcquest.Row) int {
					for k := 0; k < t.Keys; k++ {
						if n := a[k].(int64) - b[k].(int64); n != 0 {
							return int(n)
						}
					}
					return 0
				})
				q.draft[table] = slices.Insert(q.draft[table], at, row)
				done()
				return nil
			})
		})
	}
	return container.NewBorder(nil, add, nil, nil, container.NewVScroll(panel))
}

func hasPrefix(row npcquest.Row, prefix []int64) bool {
	for i, key := range prefix {
		if row[i] != key {
			return false
		}
	}
	return true
}

func (q *npcQuestEditor) questBrowser() fyne.CanvasObject {
	return q.browser(npcquest.Quests, nil, func(npcquest.Row) bool { return true },
		func(i int, done func()) fyne.CanvasObject {
			row := q.draft[npcquest.Quests][i]
			title := fmt.Sprintf("%d · %s", row[0], row[2])
			show := container.NewVBox(
				fieldHeading(title, "QuestRow / QuestName"),
				wrapped(fmt.Sprintf("Levels %d–%d · Faction %d · Mode %d", row[3], row[4], row[5], row[6])),
			)
			f := npcquest.Tables[npcquest.Quests].Fields
			start := slices.IndexFunc(f, func(f npcquest.Field) bool { return f.Name == "StartType" })
			end := slices.IndexFunc(f, func(f npcquest.Field) bool { return f.Name == "EndType" })
			rewards := slices.IndexFunc(f, func(f npcquest.Field) bool { return f.Name == "ResultType" })
			for _, group := range []struct {
				label      string
				start, end int
			}{
				{"Name and requirements", 1, start}, {"Start and starting items", start, end}, {"Objectives and completion", end, rewards}, {"Reward choices", rewards, len(f)},
			} {
				show.Add(widget.NewButton(group.label, func() { q.editRow(group.label, npcquest.Quests, i, group.start, group.end, done) }))
			}
			show.Add(widget.NewButton("Quest texts", func() { q.editTexts(title, npcquest.Quests, i, npcquest.QuestTexts[:], done) }))
			return container.NewVScroll(show)
		}, func(row npcquest.Row) string { return fmt.Sprintf("%d · %s", row[0], row[2]) }, nil)
}

func (q *npcQuestEditor) itemLinks() fyne.CanvasObject {
	typ, id := inspectorEntry("0"), inspectorEntry("0")
	body := container.NewStack()
	open := func() {
		t, err := strconv.ParseUint(strings.TrimSpace(typ.Text), 10, 8)
		if q.editor.showError(err) {
			return
		}
		i, err := strconv.ParseUint(strings.TrimSpace(id.Text), 10, 8)
		if q.editor.showError(err) {
			return
		}
		var refresh func()
		refresh = func() {
			body.Objects = []fyne.CanvasObject{container.NewAppTabs(
				container.NewTabItem("Starts", q.linkedRows(npcquest.ItemLinks, []int64{int64(t), int64(i), 0}, refresh)),
				container.NewTabItem("Ends", q.linkedRows(npcquest.ItemLinks, []int64{int64(t), int64(i), 1}, refresh)),
			)}
			body.Refresh()
		}
		refresh()
	}
	header := container.NewBorder(nil, nil, nil, widget.NewButton("Show links", open),
		container.NewGridWithColumns(2, container.NewVBox(fieldHeading("Item type", "ItemType"), typ),
			container.NewVBox(fieldHeading("Item type ID", "ItemTypeID"), id)))
	return container.NewBorder(header, nil, nil, nil, body)
}
