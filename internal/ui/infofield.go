package ui

import (
	"github.com/rivo/tview"
)

type InformationField struct {
	View *tview.TextView
}

func NewInformationField() *InformationField {
	view := tview.NewTextView().
		SetText("♡ " + "Donate Me" + " ♡").
		SetTextAlign(tview.AlignCenter)

	view.SetTitle("Created by xxx Lin").SetBorder(true)

	return &InformationField{
		View: view,
	}
}
