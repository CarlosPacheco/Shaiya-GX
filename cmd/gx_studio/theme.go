package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

const (
	mutedText    fyne.ThemeColorName = "gxMutedText"
	propertyFill fyne.ThemeColorName = "gxPropertyFill"
	propertyEdge fyne.ThemeColorName = "gxPropertyEdge"
)

var themeOptions = []string{"Light", "Dark", "Dark blue"}
var themeIDs = []string{"light", "dark", "dark-blue"}

type studioTheme struct {
	fyne.Theme
	id string
}

var darkBlueColors = map[fyne.ThemeColorName]color.Color{
	theme.ColorNameBackground:          rgb(17, 28, 45),
	theme.ColorNameOverlayBackground:   rgb(24, 39, 61),
	theme.ColorNameMenuBackground:      rgb(24, 39, 61),
	theme.ColorNameHeaderBackground:    rgb(24, 39, 61),
	theme.ColorNameInputBackground:     rgb(32, 49, 73),
	theme.ColorNameButton:              rgb(37, 56, 82),
	theme.ColorNameSeparator:           rgb(55, 77, 105),
	theme.ColorNameForeground:          rgb(229, 237, 250),
	theme.ColorNamePrimary:             rgb(123, 179, 255),
	theme.ColorNameForegroundOnPrimary: rgb(14, 30, 54),
	theme.ColorNameHyperlink:           rgb(146, 194, 255),
	theme.ColorNameFocus:               rgb(113, 166, 237),
	theme.ColorNameSelection:           rgb(45, 80, 123),
	theme.ColorNameHover:               rgb(41, 62, 91),
	mutedText:                          rgb(168, 189, 217),
	propertyFill:                       rgb(22, 36, 56),
	propertyEdge:                       rgb(55, 77, 105),
}

func rgb(r, g, b uint8) color.NRGBA { return color.NRGBA{R: r, G: g, B: b, A: 255} }

func (t studioTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	variant := theme.VariantLight
	if t.id == "dark" || t.id == "dark-blue" {
		variant = theme.VariantDark
	}
	if t.id == "dark-blue" {
		if value, ok := darkBlueColors[name]; ok {
			return value
		}
	}
	switch name {
	case mutedText:
		if variant == theme.VariantLight {
			return rgb(83, 96, 111)
		}
		return rgb(180, 188, 199)
	case propertyFill:
		if variant == theme.VariantLight {
			return rgb(245, 247, 250)
		}
		return rgb(41, 43, 47)
	case propertyEdge:
		if variant == theme.VariantLight {
			return rgb(207, 215, 225)
		}
		return rgb(77, 81, 88)
	}
	return t.Theme.Color(name, variant)
}
