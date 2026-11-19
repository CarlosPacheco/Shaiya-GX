// GX Studio is the standalone Shaiya GX desktop data editor.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/Spelunkern/Shaiya-GX/internal/data/cash"
	"github.com/Spelunkern/Shaiya-GX/internal/enchantrules"
	"github.com/Spelunkern/Shaiya-GX/internal/studio"
)

type editor struct {
	window                 fyne.Window
	root                   string
	preferences            studio.Preferences
	dataDir, dbPath        string
	definitions            *studio.Definitions
	definitionDraft        studio.DefinitionEdits
	cashQuery, cashCode    string
	cashSection            int
	cashFormDirty          func() bool
	itemFormDirty          func() bool
	enchant                *studio.EnchantCatalog
	enchantDirty           func() bool
	itemSection, itemFocus int
	itemQuery              string
	project                *studio.Project
	draft                  []cash.Product
	assets                 *assets
	status                 *widget.Label
	pending                bool
	busy                   bool
}

func main() {
	executable, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		log.Fatal(err)
	}
	instance := app.NewWithID("com.shaiyagx.studio")
	window := instance.NewWindow("GX Studio")
	e := &editor{window: window, root: filepath.Dir(executable)}
	window.Resize(fyne.NewSize(1120, 760))
	window.SetCloseIntercept(func() { e.leave(window.Close) })
	e.start()
	window.ShowAndRun()
}

func (e *editor) leave(next func()) {
	if e.busy {
		return
	}
	if !e.pending && ((e.project != nil && (e.dirty() || e.cashFormDirty != nil && e.cashFormDirty())) || len(e.definitionDraft) > 0 || e.itemFormDirty != nil && e.itemFormDirty() || e.enchantDirty != nil && e.enchantDirty()) {
		dialog.ShowConfirm("Discard draft?", "Your unapplied catalog edits will be discarded.", func(ok bool) {
			if ok {
				next()
			}
		}, e.window)
		return
	}
	next()
}

func (e *editor) start() {
	e.project, e.draft, e.assets = nil, nil, nil
	e.enchant, e.enchantDirty = nil, nil
	e.definitions, e.definitionDraft = nil, nil
	e.itemFormDirty, e.itemSection, e.itemFocus, e.itemQuery = nil, 0, -1, ""
	e.cashQuery, e.cashCode, e.cashSection = "", "", 0
	preferences, preferencesErr := studio.LoadPreferences(e.root)
	themeIndex := max(slices.Index(themeIDs, preferences.Theme), 0)
	preferences.Theme = themeIDs[themeIndex]
	e.preferences = preferences
	fyne.CurrentApp().Settings().SetTheme(studioTheme{Theme: theme.DefaultTheme(), id: preferences.Theme})
	pending, err := studio.Pending(e.root)
	e.pending = pending
	if err != nil {
		e.window.SetContent(container.NewPadded(wrapped(err.Error())))
		return
	}
	if pending {
		e.showRecovery()
		return
	}

	data := widget.NewEntry()
	data.SetPlaceHolder("Client data folder")
	db := widget.NewEntry()
	db.SetPlaceHolder("SQLite database")
	data.SetText(preferences.DataDir)
	db.SetText(preferences.Database)
	appearance := widget.NewSelect(themeOptions, nil)
	appearance.SetSelectedIndex(themeIndex)

	detect := widget.NewButtonWithIcon("Detect SQLite", theme.SearchIcon(), func() { e.detect(data, db) })
	selectFolder := widget.NewButton("Browse…", func() {
		dialog.ShowFolderOpen(func(dir fyne.ListableURI, err error) {
			if e.showError(err) || dir == nil {
				return
			}
			data.SetText(dir.Path())
			db.SetText("")
			e.detect(data, db)
		}, e.window)
	})
	selectDB := widget.NewButton("Browse…", func() {
		d := dialog.NewFileOpen(func(file fyne.URIReadCloser, err error) {
			if e.showError(err) || file == nil {
				return
			}
			db.SetText(file.URI().Path())
			e.showError(file.Close())
		}, e.window)
		d.Resize(fyne.NewSize(850, 550))
		d.Show()
	})
	mode := widget.NewSelect(catalogNames, nil)
	mode.SetSelectedIndex(0)
	open := widget.NewButton("Open catalog", nil)
	open.Importance = widget.HighImportance
	open.OnTapped = func() {
		if e.busy {
			return
		}
		dataDir, dbPath := data.Text, db.Text
		kind := studio.Kind(mode.SelectedIndex())
		if strings.TrimSpace(dbPath) == "" || (kind != studio.Enchant && strings.TrimSpace(dataDir) == "") {
			if kind == studio.Enchant {
				e.showError(fmt.Errorf("select the server SQLite database"))
			} else {
				e.showError(fmt.Errorf("select the client data folder and the server SQLite database"))
			}
			return
		}
		var err error
		if strings.TrimSpace(dataDir) != "" {
			dataDir, err = filepath.Abs(dataDir)
			if e.showError(err) {
				return
			}
		}
		dbPath, err = filepath.Abs(dbPath)
		if e.showError(err) {
			return
		}
		e.dataDir, e.dbPath = dataDir, dbPath
		e.openKind(kind)
	}
	e.status = wrapped("")
	if preferencesErr != nil {
		e.status.SetText("Could not read gx-studio.ini: " + preferencesErr.Error())
	}
	render := func() {
		title := widget.NewLabelWithStyle("Open an installation", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		fields := container.NewVBox(container.NewBorder(nil, nil, nil, container.NewHBox(widget.NewLabel("Theme"), appearance), title))
		dbSource := studio.ProductTable
		var dbActions fyne.CanvasObject = container.NewHBox(detect, selectDB)
		if studio.Kind(mode.SelectedIndex()) == studio.Enchant {
			dbSource = enchantrules.AttackTable + " · " + enchantrules.DefenseTable
			if data.Text == "" {
				dbActions = selectDB
			}
		} else {
			fields.Add(field("Client data folder", studio.Cash.RelativePath(), container.NewBorder(nil, nil, nil, selectFolder, data)))
		}
		fields.Add(field("Server database", dbSource, container.NewBorder(nil, nil, nil, dbActions, db)))
		fields.Add(container.NewHBox(mode, open))
		fields.Add(e.status)
		e.window.SetContent(container.NewPadded(container.NewBorder(nil, nil, nil, nil, container.NewVBox(fields))))
	}
	mode.OnChanged = func(string) { render() }
	appearance.OnChanged = func(label string) {
		index := slices.Index(themeOptions, label)
		if index < 0 || e.preferences.Theme == themeIDs[index] {
			return
		}
		if e.busy {
			appearance.SetSelectedIndex(slices.Index(themeIDs, e.preferences.Theme))
			return
		}
		e.preferences.Theme = themeIDs[index]
		fyne.CurrentApp().Settings().SetTheme(studioTheme{Theme: theme.DefaultTheme(), id: e.preferences.Theme})
		render()
		if err := studio.SavePreferences(e.root, e.preferences); err != nil {
			e.status.SetText("Could not save gx-studio.ini: " + err.Error())
		}
	}
	render()
}

var catalogNames = []string{"Item Mall", "Items", "Player skills", "NPC skills", "Enchant"}

func (e *editor) switcher(current studio.Kind) *widget.Select {
	control := widget.NewSelect(catalogNames, nil)
	control.SetSelectedIndex(int(current))
	control.OnChanged = func(string) {
		kind := studio.Kind(control.SelectedIndex())
		if kind == current {
			return
		}
		control.SetSelectedIndex(int(current))
		e.leave(func() { e.openKind(kind) })
	}
	return control
}

func (e *editor) openKind(kind studio.Kind) {
	if e.busy {
		return
	}
	if kind != studio.Enchant && strings.TrimSpace(e.dataDir) == "" {
		e.showError(fmt.Errorf("open an installation with a client data folder to edit that catalog"))
		return
	}
	dataDir, dbPath, existingAssets := e.dataDir, e.dbPath, e.assets
	preferences := studio.Preferences{DataDir: dataDir, Database: dbPath, Theme: e.preferences.Theme}
	message := "Reading " + kind.RelativePath() + " and SQLite…"
	if kind == studio.Enchant {
		message = "Reading enchant rules from SQLite…"
	}
	e.run(message, func() func() {
		ctx := context.Background()
		var project *studio.Project
		var definitions *studio.Definitions
		var enchant *studio.EnchantCatalog
		var err error
		if kind == studio.Cash {
			project, err = studio.Open(ctx, dataDir, dbPath)
		} else if kind == studio.Enchant {
			enchant, err = studio.OpenEnchant(ctx, dbPath)
		} else {
			definitions, err = studio.OpenDefinitions(ctx, dataDir, dbPath, kind)
		}
		images := existingAssets
		var preferencesErr error
		if err == nil {
			if images == nil && kind != studio.Enchant {
				images = loadAssets(dataDir)
			}
			preferencesErr = studio.SavePreferences(e.root, preferences)
		}
		return func() {
			if e.showError(err) {
				return
			}
			e.preferences = preferences
			e.itemFormDirty = nil
			e.project, e.definitions, e.assets = project, definitions, images
			e.enchant, e.enchantDirty = enchant, nil
			e.draft, e.definitionDraft = nil, make(studio.DefinitionEdits)
			if kind == studio.Cash {
				e.draft = append([]cash.Product(nil), project.Products...)
				e.catalog()
			} else if kind == studio.Enchant {
				e.enchantCatalog()
			} else {
				e.definitionCatalog()
			}
			if preferencesErr != nil {
				warning := "Could not save gx-studio.ini: " + preferencesErr.Error()
				e.status.SetText(strings.TrimSpace(e.status.Text + "\n" + warning))
			}
		}
	})
}

func (e *editor) detect(source, target *widget.Entry) {
	dataDir := source.Text
	if strings.TrimSpace(dataDir) == "" || e.busy {
		return
	}
	e.run("Looking for the runtime database…", func() func() {
		found := studio.Detect(context.Background(), dataDir)
		return func() {
			if source.Text != dataDir {
				return
			}
			switch len(found) {
			case 0:
				e.status.SetText("Choose the SQLite database manually.")
			case 1:
				target.SetText(found[0])
				e.status.SetText("Runtime database detected. Verify the path before opening.")
			default:
				choice := widget.NewSelect(found, nil)
				d := dialog.NewCustomConfirm("Select the runtime database", "Use database", "Cancel", choice, func(ok bool) {
					if ok && choice.Selected != "" {
						target.SetText(choice.Selected)
					}
				}, e.window)
				d.Resize(fyne.NewSize(850, 230))
				d.Show()
			}
		}
	})
}

func (e *editor) showRecovery() {
	e.status = wrapped("")
	offline := widget.NewCheck("The affected GX client and server are stopped", nil)
	recover := widget.NewButton("Recover last save", nil)
	recover.Importance = widget.HighImportance
	recover.Disable()
	offline.OnChanged = func(checked bool) {
		if checked && !e.busy {
			recover.Enable()
		} else {
			recover.Disable()
		}
	}
	recover.OnTapped = func() {
		if !offline.Checked || e.busy {
			return
		}
		recover.Disable()
		e.run("Checking the committed catalog…", func() func() {
			backup, err := studio.Recover(context.Background(), e.root)
			return func() {
				if e.showError(err) {
					e.status.SetText("Recovery is still pending. Backup folder • " + backup)
					recover.Enable()
					return
				}
				e.start()
				dialog.ShowInformation("Recovery complete", "Both catalogs were verified. Backup folder\n"+backup, e.window)
			}
		})
	}
	e.window.SetContent(container.NewPadded(container.NewVBox(
		wrapped("A previous save was interrupted. GX Studio must recover the catalog before opening an installation."),
		offline, container.NewHBox(recover), e.status)))
}

func (e *editor) run(message string, work func() func()) {
	e.busy = true
	e.status.SetText(message)
	go func() {
		done := work()
		fyne.Do(func() {
			e.busy = false
			e.status.SetText("")
			done()
		})
	}()
}

func (e *editor) showError(err error) bool {
	if err != nil {
		dialog.ShowError(err, e.window)
	}
	return err != nil
}

func wrapped(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	return label
}
