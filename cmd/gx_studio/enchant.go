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
	level                  uint8
	chance, percent, bonus *widget.Entry
}

const (
	enchantChanceScale = 10_000
	enchantChanceMax   = 100 * enchantChanceScale
)

func (e *editor) enchantCatalog() {
	e.status = wrapped("")
	catalog := e.enchant
	var rows [2][]enchantRow
	read := func() (enchantrules.Catalog, uint8, uint8, error) {
		var rules enchantrules.Catalog
		for i, group := range rows {
			target, kind := &rules.Attack, "Attack"
			if i == 1 {
				target, kind = &rules.Defense, "Defense"
			}
			for _, row := range group {
				chance, err := strconv.ParseUint(strings.TrimSpace(row.chance.Text), 10, 32)
				if err != nil || chance > enchantChanceMax {
					return rules, 0, 0, fmt.Errorf("%s +%d: enter a chance from 0 to 100%% with up to four decimals, or 0 to 1000000 per million", kind, row.level)
				}
				bonus, err := strconv.ParseUint(strings.TrimSpace(row.bonus.Text), 10, 31)
				if err != nil {
					return rules, 0, 0, fmt.Errorf("%s +%d: enter a whole bonus from 0 to 2147483647", kind, row.level)
				}
				*target = append(*target, enchantrules.Level{Number: row.level, SuccessChance: uint32(chance), Bonus: uint32(bonus)})
			}
		}
		attack, defense, err := rules.Caps()
		return rules, attack, defense, err
	}
	e.serverDraftDirty = func() bool {
		rules, _, _, err := read()
		return err != nil || !rules.Equal(catalog.Rules)
	}
	capLabel := widget.NewLabel("")
	tabs := container.NewAppTabs()
	var update func()
	save := widget.NewButtonWithIcon("Save server file", theme.DocumentSaveIcon(), func() {
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
		rules, attack, defense, err := read()
		if err != nil {
			capLabel.SetText(err.Error())
		} else {
			capLabel.SetText(fmt.Sprintf("Attack cap +%d  ·  Defense cap +%d  ·  Maximum +%d", attack, defense, enchantrules.MaxLevel))
		}
		if err == nil && !rules.Equal(catalog.Rules) && !e.busy {
			save.Enable()
		} else {
			save.Disable()
		}
	}
	for i, levels := range [2][]enchantrules.Level{catalog.Rules.Attack, catalog.Rules.Defense} {
		for _, level := range levels {
			rows[i] = append(rows[i], enchantInputs(level, update))
		}
	}
	tabs.Append(container.NewTabItem("Attack", enchantTable(&rows[0], "Weapon", "Attack bonus", update)))
	tabs.Append(container.NewTabItem("Defense", enchantTable(&rows[1], "Defense", "Absorption bonus", update)))
	update()
	top := container.NewVBox(
		container.NewHBox(widget.NewButton("Open another installation", func() { e.leave(e.start) }),
			e.switcher(studio.Enchant), save),
		wrapped(catalog.File.Path), capLabel,
		wrapped("Edit either chance column. Chances apply when reaching the listed level, and bonuses are totals at that level."),
	)
	e.window.SetContent(container.NewPadded(container.NewBorder(top, e.status, nil, nil, tabs)))
}

func formatEnchantPercent(chance uint64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%d.%04d", chance/enchantChanceScale, chance%enchantChanceScale), "0"), ".")
}

func parseEnchantPercent(text string) (uint64, bool) {
	text = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(text), "%"))
	whole, fraction, _ := strings.Cut(text, ".")
	if whole == "" {
		if fraction == "" {
			return 0, false
		}
		whole = "0"
	}
	integer, err := strconv.ParseUint(whole, 10, 8)
	if err != nil || len(fraction) > 4 {
		return 0, false
	}
	decimal, err := strconv.ParseUint(fraction+strings.Repeat("0", 4-len(fraction)), 10, 16)
	chance := integer*enchantChanceScale + decimal
	return chance, err == nil && chance <= enchantChanceMax
}

func enchantInputs(level enchantrules.Level, update func()) enchantRow {
	row := enchantRow{
		level:   level.Number,
		chance:  inspectorEntry(strconv.FormatUint(uint64(level.SuccessChance), 10)),
		percent: inspectorEntry(formatEnchantPercent(uint64(level.SuccessChance))),
		bonus:   inspectorEntry(strconv.FormatUint(uint64(level.Bonus), 10)),
	}
	syncing := false
	syncEntry := func(entry *widget.Entry, text string) {
		syncing = true
		if entry.Text != text {
			entry.SetText(text)
		}
		syncing = false
		update()
	}
	row.chance.OnChanged = func(text string) {
		if syncing {
			return
		}
		chance, err := strconv.ParseUint(strings.TrimSpace(text), 10, 32)
		percent := ""
		if err == nil && chance <= enchantChanceMax {
			percent = formatEnchantPercent(chance)
		}
		syncEntry(row.percent, percent)
	}
	row.percent.OnChanged = func(text string) {
		if syncing {
			return
		}
		raw := ""
		if chance, ok := parseEnchantPercent(text); ok {
			raw = strconv.FormatUint(chance, 10)
		}
		syncEntry(row.chance, raw)
	}
	row.bonus.OnChanged = func(string) { update() }
	return row
}

func enchantTable(rows *[]enchantRow, prefix, bonusTitle string, update func()) fyne.CanvasObject {
	body := container.NewVBox()
	grid := func(cells ...fyne.CanvasObject) fyne.CanvasObject {
		return container.NewGridWithColumns(4, cells...)
	}
	add, remove := widget.NewButton("Add next level", nil), widget.NewButton("Remove highest level", nil)
	render := func() {
		body.Objects = nil
		for _, row := range *rows {
			rateKey := small(fmt.Sprintf("%sStep%02d", prefix, row.level-1))
			bonusKey := small(fmt.Sprintf("%sStep%02d", prefix, row.level))
			body.Add(grid(widget.NewLabel(fmt.Sprintf("+%d", row.level)),
				container.NewVBox(rateKey, row.chance), container.NewBorder(nil, row.percent, nil, nil),
				container.NewVBox(bonusKey, row.bonus)))
		}
		body.Refresh()
		add.Enable()
		remove.Enable()
		if len(*rows) == enchantrules.MaxLevel {
			add.Disable()
		}
		if len(*rows) == 0 {
			remove.Disable()
		}
		update()
	}
	add.OnTapped = func() {
		*rows = append(*rows, enchantInputs(enchantrules.Level{Number: uint8(len(*rows) + 1)}, update))
		render()
	}
	remove.OnTapped = func() {
		*rows = (*rows)[:len(*rows)-1]
		render()
	}
	render()
	header := grid(fieldHeading("Level", "[Enchant]"), fieldHeading("Chance per million", "[LapisianEnchantPercentRate]"),
		widget.NewLabel("Chance (%)"), fieldHeading(bonusTitle, "[LapisianEnchantAddValue]"))
	top := container.NewVBox(container.NewHBox(add, remove), header, widget.NewSeparator())
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
			e.enchant = &studio.EnchantCatalog{DBPath: catalog.DBPath, File: catalog.File, Rules: rules}
			e.enchantCatalog()
			e.status.SetText("Saved. Restart GX World to apply. Backup · " + backup)
		}
	})
}
