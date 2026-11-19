package main

import (
	"context"
	"path/filepath"
	"slices"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/Spelunkern/Shaiya-GX/internal/mobaimessages"
	"github.com/Spelunkern/Shaiya-GX/internal/studio"
)

type mobAIMessageTheme struct {
	fyne.Theme
	font fyne.Resource
}

func (t mobAIMessageTheme) Font(fyne.TextStyle) fyne.Resource { return t.font }

func (e *editor) mobAIMessageCatalog() {
	e.status = wrapped("")
	catalog := e.mobAIMessages
	draft := slices.Clone(catalog.Messages)
	e.databaseDraftDirty = func() bool { return !slices.Equal(draft, catalog.Messages) }
	rows := container.NewVBox()
	save := widget.NewButtonWithIcon("Save database", theme.DocumentSaveIcon(), func() {
		if e.busy {
			return
		}
		pending := slices.Clone(draft)
		saveDraft := func() func() {
			backup, err := studio.SaveMobAIMessages(context.Background(), e.root, catalog, pending)
			return func() {
				if e.showError(err) {
					rows.Show()
					return
				}
				e.mobAIMessages = &studio.MobAIMessageCatalog{DBPath: catalog.DBPath, Messages: pending}
				e.mobAIMessageCatalog()
				e.status.SetText("Saved. Restart GX World to apply. Backup · " + backup)
			}
		}
		rows.Hide()
		e.run("Backing up and saving mob AI messages…", saveDraft)
	})
	save.Importance = widget.HighImportance
	save.Disable()
	font := e.mobAIMessageFont()
	for i, message := range catalog.Messages {
		entry := inspectorEntry(message.Text)
		entry.OnChanged = func(text string) {
			draft[i].Text = text
			if e.databaseDraftDirty() {
				save.Enable()
			} else {
				save.Disable()
			}
		}
		var field fyne.CanvasObject = entry
		if font != nil {
			field = container.NewThemeOverride(entry, mobAIMessageTheme{fyne.CurrentApp().Settings().Theme(), font})
		}
		index := widget.NewLabel(strconv.Itoa(int(message.Index)))
		rows.Add(container.NewBorder(nil, nil, index, nil, field))
	}
	top := container.NewVBox(
		container.NewHBox(widget.NewButton("Open another installation", func() { e.leave(e.start) }),
			e.switcher(studio.MobAIMessages), save),
		wrapped(catalog.DBPath), wrapped(mobaimessages.Table),
		container.NewHBox(fieldHeading("Message ID", "Index"), fieldHeading("Message", "Text")),
		widget.NewSeparator(),
	)
	e.window.SetContent(container.NewPadded(container.NewBorder(top, e.status, nil, nil, container.NewVScroll(rows))))
}

func (e *editor) mobAIMessageFont() fyne.Resource {
	const filename = "NotoSansCJKsc-Regular.otf"
	paths := []string{filepath.Join(e.root, "fonts", filename)}
	if e.dataDir != "" {
		paths = append(paths, filepath.Join(e.dataDir, "fonts", "unicode", filename))
	}
	for _, path := range paths {
		if font, err := fyne.LoadResourceFromPath(path); err == nil {
			return font
		}
	}
	return nil
}
