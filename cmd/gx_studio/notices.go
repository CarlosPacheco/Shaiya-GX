package main

import (
	"path/filepath"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/Spelunkern/Shaiya-GX/internal/catalogfiles"
	"github.com/Spelunkern/Shaiya-GX/internal/studio"
)

func (e *editor) openNotices(path string, preferences studio.Preferences) {
	file, err := studio.OpenServerFile(path)
	if e.showError(err) {
		return
	}
	if _, err := catalogfiles.ParseNotices([]byte(file.Text())); e.showError(err) {
		return
	}
	e.preferences = preferences
	if err := studio.SavePreferences(e.root, preferences); e.showError(err) {
		return
	}
	e.project, e.definitions = nil, nil
	e.definitionDraft, e.itemFormDirty = nil, nil
	e.status = wrapped("")
	entry := widget.NewMultiLineEntry()
	entry.SetText(file.Text())
	e.serverDraftDirty = func() bool { return entry.Text != file.Text() }
	save := widget.NewButtonWithIcon("Save server file", theme.DocumentSaveIcon(), func() {
		if e.busy {
			return
		}
		next := []byte(entry.Text)
		if _, err := catalogfiles.ParseNotices(next); e.showError(err) {
			return
		}
		e.run("Backing up and saving notices…", func() func() {
			backup, err := studio.SaveServerFile(e.root, file, next)
			return func() {
				if !e.showError(err) {
					e.status.SetText("Saved. Restart GX World to apply. Backup · " + backup)
				}
			}
		})
	})
	top := container.NewVBox(container.NewHBox(widget.NewButton("Open another installation", func() { e.leave(e.start) }), e.switcher(studio.Notices), save), wrapped(file.Path),
		wrapped(filepath.Base(path)+" · [Notice_N] · Message · Frequency (minutes) · FixedHour1–5 (day/hour/minute)"))
	e.window.SetContent(container.NewPadded(container.NewBorder(top, e.status, nil, nil, entry)))
}
