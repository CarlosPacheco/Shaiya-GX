package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/Spelunkern/Shaiya-GX/internal/enchantrules"
	"github.com/Spelunkern/Shaiya-GX/internal/studio"
)

type enchantRow struct {
	level         uint8
	chance, bonus *widget.Entry
}

func (e *editor) enchantCatalog() {
	e.status = wrapped("")
	catalog := e.enchant
	var rows [2][]enchantRow
	read := func() (enchantrules.Catalog, uint8, uint8, error) {
		var rules enchantrules.Catalog
		for i, group := range rows {
			target := &rules.Attack
			if i == 1 {
				target = &rules.Defense
			}
			for _, row := range group {
				chance, chanceErr := strconv.ParseUint(strings.TrimSpace(row.chance.Text), 10, 32)
				bonus, bonusErr := strconv.ParseUint(strings.TrimSpace(row.bonus.Text), 10, 32)
				if chanceErr != nil || bonusErr != nil {
					return rules, 0, 0, fmt.Errorf("invalid number at level +%d", row.level)
				}
				*target = append(*target, enchantrules.Level{Number: row.level, SuccessChance: uint32(chance), Bonus: uint32(bonus)})
			}
		}
		attack, defense, err := rules.Caps()
		return rules, attack, defense, err
	}
	e.enchantDirty = func() bool {
		rules, _, _, err := read()
		return err != nil || !rules.Equal(catalog.Rules)
	}
	capLabel := widget.NewLabel("")
	tabs := container.NewAppTabs()
	var update func()
	save := widget.NewButtonWithIcon("Save database", theme.DocumentSaveIcon(), func() {
		if e.busy {
			return
		}
		rules, attack, defense, err := read()
		if e.showError(err) {
			return
		}
		message := fmt.Sprintf("Attack cap +%d, defense cap +%d. Restart GX World to load the saved rules.", attack, defense)
		if len(rules.Attack) < len(catalog.Rules.Attack) || len(rules.Defense) < len(catalog.Rules.Defense) {
			message += "\nEquipment at removed levels will lose the corresponding bonuses."
		}
		dialog.ShowConfirm("Save enchant rules?", message, func(ok bool) {
			if ok && !e.busy {
				tabs.Hide()
				savePlaceholder := func() { tabs.Show(); update() }
				e.saveEnchant(rules, savePlaceholder)
			}
		}, e.window)
	})
	save.Importance = widget.HighImportance
	update = func() {
		_, attack, defense, err := read()
		if err != nil {
			capLabel.SetText(err.Error())
		} else {
			capLabel.SetText(fmt.Sprintf("Attack cap +%d  ·  Defense cap +%d  ·  Maximum +%d", attack, defense, enchantrules.MaxLevel))
		}
		if err == nil && e.enchantDirty() && !e.busy {
			save.Enable()
		} else {
			save.Disable()
		}
	}
	for i, levels := range [2][]enchantrules.Level{catalog.Rules.Attack, catalog.Rules.Defense} {
		for _, level := range levels {
			rows[i] = append(rows[i], enchantInputs(level))
		}
	}
	tabs.Append(container.NewTabItem("Attack", enchantTable(&rows[0], enchantrules.AttackTable, "Attack bonus", "AttackBonus", update)))
	tabs.Append(container.NewTabItem("Defense", enchantTable(&rows[1], enchantrules.DefenseTable, "Absorption bonus", "AbsorptionBonus", update)))
	update()
	top := container.NewVBox(
		container.NewHBox(widget.NewButton("Open another installation", func() { e.leave(e.start) }),
			e.switcher(studio.Enchant), save),
		wrapped(catalog.DBPath), capLabel,
		wrapped("Chances are per million for the next level. Bonuses are totals at the listed level."),
	)
	e.window.SetContent(container.NewPadded(container.NewBorder(top, e.status, nil, nil, tabs)))
}

func enchantInputs(level enchantrules.Level) enchantRow {
	return enchantRow{level: level.Number, chance: inspectorEntry(strconv.FormatUint(uint64(level.SuccessChance), 10)),
		bonus: inspectorEntry(strconv.FormatUint(uint64(level.Bonus), 10))}
}

func enchantTable(rows *[]enchantRow, table, bonusTitle, bonusField string, update func()) fyne.CanvasObject {
	body := container.NewVBox()
	grid := func(cells ...fyne.CanvasObject) fyne.CanvasObject {
		return container.NewGridWithColumns(4, cells...)
	}
	add, remove := widget.NewButton("Add next level", nil), widget.NewButton("Remove highest level", nil)
	render := func() {
		body.Objects = nil
		for _, row := range *rows {
			percent := widget.NewLabel("")
			updateChance := func(text string) {
				value, err := strconv.ParseUint(strings.TrimSpace(text), 10, 32)
				if err != nil || value > 1000000 {
					percent.SetText("—")
				} else {
					percent.SetText(fmt.Sprintf("%.4f%%", float64(value)/10000))
				}
				update()
			}
			row.chance.OnChanged = updateChance
			row.bonus.OnChanged = func(string) { update() }
			updateChance(row.chance.Text)
			body.Add(grid(widget.NewLabel(fmt.Sprintf("+%d", row.level)), row.chance, percent, row.bonus))
		}
		body.Refresh()
		add.Enable()
		remove.Enable()
		if len(*rows) == enchantrules.MaxLevel+1 {
			add.Disable()
		}
		if len(*rows) == 1 {
			remove.Disable()
		}
		update()
	}
	add.OnTapped = func() {
		*rows = append(*rows, enchantInputs(enchantrules.Level{Number: uint8(len(*rows))}))
		render()
	}
	remove.OnTapped = func() {
		*rows = (*rows)[:len(*rows)-1]
		render()
	}
	render()
	header := grid(fieldHeading("Level", "Level"), fieldHeading("Success chance", "SuccessChance"),
		widget.NewLabel("Percent"), fieldHeading(bonusTitle, bonusField))
	top := container.NewVBox(wrapped(table), container.NewHBox(add, remove), header, widget.NewSeparator())
	return container.NewBorder(top, nil, nil, nil, container.NewVScroll(body))
}

func fieldHeading(label, source string) fyne.CanvasObject {
	return container.NewVBox(widget.NewLabel(label), small(source))
}

func (e *editor) saveEnchant(rules enchantrules.Catalog, failed func()) {
	catalog := e.enchant
	e.run("Backing up and saving enchant rules…", func() func() {
		ctx := context.Background()
		backup, err := studio.SaveEnchant(ctx, e.root, catalog, rules)
		return func() {
			if e.showError(err) {
				failed()
				return
			}
			e.enchant = &studio.EnchantCatalog{DBPath: catalog.DBPath, Rules: rules}
			e.enchantCatalog()
			e.status.SetText("Saved. Restart GX World to apply. Backup · " + backup)
		}
	})
}
