package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/Spelunkern/Shaiya-GX/internal/data/cash"
	"github.com/Spelunkern/Shaiya-GX/internal/studio"
)

var categories = []string{"Hidden (legacy)", "Popular", "Essentials", "Buffs", "Packages"}
var blankIcon = image.NewNRGBA(image.Rect(0, 0, 1, 1))

func small(text string) *canvas.Text {
	label := canvas.NewText(text, color.NRGBA{R: 83, G: 96, B: 111, A: 255})
	label.TextSize = theme.TextSize() * .8
	return label
}

func field(title, source string, input fyne.CanvasObject) fyne.CanvasObject {
	return container.NewVBox(container.NewHBox(widget.NewLabel(title), small(source)), input)
}

func newIcon(size float32) *canvas.Image {
	icon := canvas.NewImageFromImage(blankIcon)
	icon.FillMode = canvas.ImageFillContain
	icon.SetMinSize(fyne.NewSize(size, size))
	return icon
}

func showIcon(icon *canvas.Image, target image.Image) {
	if target == nil {
		target = blankIcon
	}
	icon.Image = target
	icon.Refresh()
}

func category(n uint32) string {
	if n >= uint32(len(categories)) {
		return fmt.Sprintf("Unknown (%d)", n)
	}
	return categories[n]
}

func (e *editor) dirty() bool { return !slices.Equal(e.draft, e.project.Products) }

func (e *editor) catalog() {
	e.status = wrapped(e.assets.warning)
	e.cashFormDirty = nil
	inspector := container.NewStack(wrapped("Select a product to edit its fields."))
	formPending := func() bool { return e.cashFormDirty != nil && e.cashFormDirty() }
	requireApplied := func() bool {
		if formPending() {
			e.status.SetText("Apply the product changes to the draft first.")
			return false
		}
		return true
	}
	paths := wrapped(e.project.CashPath + "\n" + e.project.DBPath + "  ·  " + studio.ProductTable)
	search := widget.NewEntry()
	search.SetPlaceHolder("Search products by name or code")
	search.SetText(e.cashQuery)
	sections := []catalogSection{{name: "All products", count: len(e.draft)}}
	for _, name := range categories[1:] {
		sections = append(sections, catalogSection{name: name})
	}
	for _, product := range e.draft {
		if product.Bag > 0 && product.Bag < uint32(len(categories)) {
			sections[product.Bag].count++
		}
	}
	selected := -1
	var visible []int
	var list *widget.List

	remove := widget.NewButton("Remove", func() {
		if selected < 0 || e.busy {
			return
		}
		index := selected
		dialog.ShowConfirm("Remove product?", "Remove "+e.draft[index].Code+" from the draft catalog?", func(ok bool) {
			if ok {
				e.draft = slices.Delete(e.draft, index, index+1)
				e.catalog()
			}
		}, e.window)
	})
	add := widget.NewButtonWithIcon("New product", theme.ContentAddIcon(), func() {
		if !e.busy && requireApplied() {
			e.editProduct(cash.Product{Index: e.nextIndex(), Bag: 1}, func(p cash.Product) {
				e.draft = append(e.draft, p)
				e.cashQuery, e.cashCode, e.cashSection = "", p.Code, 0
				e.catalog()
			})
		}
	})
	move := func(delta int) {
		target := selected + delta
		if e.busy || selected < 0 || target < 0 || target >= len(e.draft) || !requireApplied() {
			return
		}
		e.draft[selected], e.draft[target] = e.draft[target], e.draft[selected]
		e.catalog()
	}
	up := widget.NewButtonWithIcon("Move up", theme.MoveUpIcon(), func() { move(-1) })
	down := widget.NewButtonWithIcon("Move down", theme.MoveDownIcon(), func() { move(1) })
	remove.Disable()
	up.Disable()
	down.Disable()
	changes := studio.Differences(e.draft, e.project.Server)
	review := widget.NewButton(fmt.Sprintf("SQLite differences (%d)", len(changes)), func() {
		if !e.busy && requireApplied() {
			e.review(changes, false)
		}
	})
	save := widget.NewButtonWithIcon("Save both", theme.DocumentSaveIcon(), func() {
		if !e.busy && requireApplied() {
			e.review(changes, true)
		}
	})
	save.Importance = widget.HighImportance
	if e.project.Encrypted {
		add.Disable()
		save.Disable()
		e.status.SetText("Encrypted Cash.SData • read-only. " + e.assets.warning)
	} else if !e.dirty() && len(changes) == 0 {
		save.Disable()
	}
	close := widget.NewButton("Open another installation", func() { e.leave(e.start) })

	rowLayout := func(title, code, points string, thumbnail fyne.CanvasObject) fyne.CanvasObject {
		name := widget.NewLabel(title)
		name.Truncation = fyne.TextTruncateEllipsis
		columns := container.NewHBox(
			container.NewGridWrap(fyne.NewSize(125, 28), widget.NewLabel(code)),
			container.NewGridWrap(fyne.NewSize(70, 28), widget.NewLabelWithStyle(points, fyne.TextAlignTrailing, fyne.TextStyle{})),
		)
		return container.NewBorder(nil, nil, thumbnail, columns, name)
	}
	list = widget.NewList(func() int { return len(visible) }, func() fyne.CanvasObject {
		return rowLayout("", "", "", newIcon(24))
	}, func(row widget.ListItemID, object fyne.CanvasObject) {
		product := e.draft[visible[row]]
		children := object.(*fyne.Container).Objects
		children[0].(*widget.Label).SetText(product.Name)
		showIcon(children[1].(*canvas.Image), e.assets.productIcon(product))
		columns := children[2].(*fyne.Container).Objects
		columns[0].(*fyne.Container).Objects[0].(*widget.Label).SetText(product.Code)
		columns[1].(*fyne.Container).Objects[0].(*widget.Label).SetText(fmt.Sprint(product.Cost))
	})
	showProduct := func(index int) {
		selected = index
		p := e.draft[index]
		e.cashCode = p.Code
		form, pending := e.productForm(p, func(p cash.Product) {
			e.draft[index] = p
			e.cashCode = p.Code
			if e.cashSection != 0 && p.Bag != uint32(e.cashSection) {
				e.cashSection = int(p.Bag)
			}
			e.catalog()
		})
		e.cashFormDirty = pending
		inspector.Objects = []fyne.CanvasObject{form}
		inspector.Refresh()
		if !e.project.Encrypted {
			remove.Enable()
			if selected > 0 {
				up.Enable()
			} else {
				up.Disable()
			}
			if selected < len(e.draft)-1 {
				down.Enable()
			} else {
				down.Disable()
			}
		}
	}
	restoring := false
	list.OnSelected = func(id widget.ListItemID) {
		next := visible[id]
		if restoring || next == selected {
			return
		}
		if formPending() {
			dialog.ShowConfirm("Discard unapplied changes?", "Apply to draft to keep the current product edits.", func(ok bool) {
				if ok {
					showProduct(next)
					return
				}
				restoring = true
				list.UnselectAll()
				if row := slices.Index(visible, selected); row >= 0 {
					list.Select(row)
				}
				restoring = false
			}, e.window)
			return
		}
		showProduct(next)
	}
	refresh := func() {
		list.UnselectAll()
		if !formPending() {
			selected = -1
			e.cashFormDirty = nil
			remove.Disable()
			up.Disable()
			down.Disable()
			inspector.Objects = []fyne.CanvasObject{wrapped("Select a product to edit its fields.")}
			inspector.Refresh()
		}
		query := strings.ToLower(strings.TrimSpace(search.Text))
		section := e.cashSection
		e.cashQuery = search.Text
		visible = visible[:0]
		for i, product := range e.draft {
			if (section <= 0 || product.Bag == uint32(section)) &&
				(includes(product.Name, query) || includes(product.Code, query)) {
				visible = append(visible, i)
			}
		}
		list.Refresh()
		for row, index := range visible {
			if e.draft[index].Code == e.cashCode {
				list.Select(row)
				break
			}
		}
	}
	search.OnChanged = func(string) { refresh() }
	refresh()
	details := container.NewBorder(container.NewHBox(remove, up, down), nil, nil, nil, inspector)
	header := rowLayout("Product", "Code", "Points", newIcon(24))
	left := container.NewBorder(container.NewVBox(search, header, widget.NewSeparator()), nil, nil, nil, list)
	split := container.NewHSplit(left, details)
	split.SetOffset(.51)
	toolbar := container.NewHBox(close, e.switcher(studio.Cash), add, review, save)
	e.window.SetContent(container.NewPadded(container.NewBorder(container.NewVBox(toolbar, paths), e.status, nil, nil, sectionBrowser("Sections", "Cash.Bag", sections, e.cashSection, func(section int) {
		e.cashSection = section
		refresh()
	}, split))))
}

func (e *editor) nextIndex() uint32 {
	used := make(map[uint32]bool, len(e.draft))
	for _, p := range e.draft {
		used[p.Index] = true
	}
	index := uint32(1)
	for used[index] {
		index++
	}
	return index
}

func (e *editor) review(changes []studio.Change, saving bool) {
	var d *dialog.CustomDialog
	lines := wrapped("Both catalogs already agree on the shared fields.")
	if saving && len(changes) == 0 {
		lines.SetText("The shared fields agree. This save updates the client-only fields and creates backups of both catalogs.")
	}
	list := widget.NewList(func() int { return len(changes) }, func() fyne.CanvasObject {
		return widget.NewLabel("")
	}, func(i widget.ListItemID, o fyne.CanvasObject) {
		o.(*widget.Label).SetText(changes[i].Code)
	})
	useDB := widget.NewButton("Use SQLite fields in draft", nil)
	useDB.Disable()
	list.OnSelected = func(i widget.ListItemID) {
		change := changes[i]
		lines.SetText(change.Code + "\n\n" + change.Summary)
		server, exists := e.project.Server[change.Code]
		if exists && !e.project.Encrypted {
			useDB.Enable()
			useDB.OnTapped = func() {
				index := slices.IndexFunc(e.draft, func(p cash.Product) bool { return p.Code == change.Code })
				if index < 0 {
					e.draft = append(e.draft, server.Apply(cash.Product{Index: e.nextIndex()}))
				} else {
					e.draft[index] = server.Apply(e.draft[index])
				}
				d.Hide()
				e.catalog()
			}
		} else {
			useDB.Disable()
		}
	}
	split := container.NewHSplit(list, container.NewVScroll(lines))
	split.SetOffset(.33)
	cancel := widget.NewButton("Close", func() { d.Hide() })
	actions := container.NewHBox(cancel, useDB)
	header := wrapped("SQLite → proposed value. Select a product to inspect the changes.")
	title := "SQLite differences"
	if saving {
		title = "Review coordinated save"
		if len(changes) > 0 {
			header.SetText("Saving applies the draft to both catalogs, including all listed SQLite additions, changes and removals.")
		} else {
			header.SetText("Review the catalog before saving.")
		}
		offline := widget.NewCheck("The affected GX client and server are stopped", nil)
		save := widget.NewButton("Apply to both catalogs", func() {
			if offline.Checked && !e.busy {
				d.Hide()
				e.save()
			}
		})
		save.Importance = widget.HighImportance
		save.Disable()
		offline.OnChanged = func(ok bool) {
			if ok {
				save.Enable()
			} else {
				save.Disable()
			}
		}
		actions = container.NewHBox(cancel, useDB, offline, save)
	}
	d = dialog.NewCustomWithoutButtons(title, container.NewBorder(header, actions, nil, nil, split), e.window)
	d.Resize(fyne.NewSize(900, 490))
	d.Show()
}

func (e *editor) save() {
	project, draft := e.project, slices.Clone(e.draft)
	e.run("Validating and backing up both catalogs…", func() func() {
		backup, err := studio.Save(context.Background(), e.root, project, draft)
		var updated *studio.Project
		saved := err == nil
		if saved {
			updated, err = studio.Open(context.Background(), project.DataDir, project.DBPath)
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
			e.project, e.draft = updated, slices.Clone(updated.Products)
			e.catalog()
			e.status.SetText("Saved and verified both catalogs. Backups • " + backup)
		}
	})
}
