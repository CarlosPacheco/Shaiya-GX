package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/Spelunkern/Shaiya-GX/internal/studio"
)

type catalogSection struct {
	name  string
	count int
}

// sectionBrowser keeps filters visible beside the catalog and its inspector.
func sectionBrowser(title, source string, sections []catalogSection, selected int, onSelected func(int), content fyne.CanvasObject) *container.Split {
	list := widget.NewList(func() int { return len(sections) }, func() fyne.CanvasObject {
		name := widget.NewLabel("")
		name.Truncation = fyne.TextTruncateEllipsis
		count := widget.NewLabelWithStyle("", fyne.TextAlignTrailing, fyne.TextStyle{})
		return container.NewBorder(nil, nil, nil, container.NewGridWrap(fyne.NewSize(42, 28), count), name)
	}, func(id widget.ListItemID, object fyne.CanvasObject) {
		section := sections[id]
		children := object.(*fyne.Container).Objects
		children[0].(*widget.Label).SetText(section.name)
		children[1].(*fyne.Container).Objects[0].(*widget.Label).SetText(strconv.Itoa(section.count))
	})
	list.Select(selected)
	list.OnSelected = onSelected
	heading := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	sidebar := container.NewBorder(container.NewVBox(heading, small(source), widget.NewSeparator()), nil, nil, nil, list)
	split := container.NewHSplit(sidebar, content)
	split.SetOffset(.135)
	return split
}

func definitionSections(catalog *studio.Definitions, drafts studio.DefinitionEdits) ([]catalogSection, map[int]int, []int) {
	values := make([]int, len(catalog.Records))
	counts := make(map[int]int)
	var effect studio.Field
	for _, field := range catalog.Kind.Fields() {
		if field.Source == "TypeEffect" {
			effect = field
			break
		}
	}
	for i := range catalog.Records {
		raw := catalog.Value(i, effect)
		if value, ok := drafts[i][effect.Source]; ok {
			raw = value
		}
		key, _ := strconv.Atoi(raw)
		values[i] = key
		counts[key]++
	}
	keys := make([]int, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	sections := []catalogSection{{name: "All records", count: len(catalog.Records)}}
	rows := make(map[int]int, len(counts))
	for _, key := range keys {
		name := fmt.Sprintf("Effect %d", key)
		for _, option := range readableValues("TypeEffect") {
			code, _, _ := strings.Cut(option, " · ")
			if code == strconv.Itoa(key) {
				name = option
				break
			}
		}
		rows[key] = len(sections)
		sections = append(sections, catalogSection{name: name, count: counts[key]})
	}
	return sections, rows, values
}
