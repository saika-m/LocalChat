package ui

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

type InformationField struct {
	View              *tview.TextView
	bleAvailable      bool
	natAvailable      bool
	internetAvailable bool
}

func NewInformationField() *InformationField {
	view := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	view.SetTitle("Connection Modes").SetBorder(true)

	return &InformationField{
		View:              view,
		bleAvailable:      false,
		natAvailable:      false,
		internetAvailable: false,
	}
}

// UpdateModes updates the display to show which connection modes are available
func (i *InformationField) UpdateModes(bleAvailable, natAvailable, internetAvailable bool) {
	// Always update, don't skip if values are the same (in case of errors)
	i.bleAvailable = bleAvailable
	i.natAvailable = natAvailable
	i.internetAvailable = internetAvailable

	var indicators []string

	if bleAvailable {
		indicators = append(indicators, fmt.Sprintf("[green]●[white] BLE"))
	} else {
		indicators = append(indicators, fmt.Sprintf("[gray]○[white] BLE"))
	}

	if natAvailable {
		indicators = append(indicators, fmt.Sprintf("[green]●[white] NAT"))
	} else {
		indicators = append(indicators, fmt.Sprintf("[gray]○[white] NAT"))
	}

	if internetAvailable {
		indicators = append(indicators, fmt.Sprintf("[green]●[white] Internet"))
	} else {
		indicators = append(indicators, fmt.Sprintf("[gray]○[white] Internet"))
	}

	text := strings.Join(indicators, "  |  ")
	if text == "" {
		// Fallback: show something even if all checks fail
		text = "[gray]○[white] BLE  |  [gray]○[white] NAT  |  [gray]○[white] Internet"
	}
	i.View.SetText(text)
}
