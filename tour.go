package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func ShowGuidedTour() {
	checkbox := widget.NewCheck("Don't show this again", func(checked bool) {
		Config.SetShowGuidedTour(!checked)
	})
	checkbox.Checked = !Config.ShowGuidedTour()

	var text *widget.Label
	if fyne.CurrentDevice().IsMobile() {
		text = widget.NewLabel("The Hamburger menu in the upper left allows you to control the simulation.\nTry Help->Tour to get started.")
	} else {
		text = widget.NewLabel("You might want to take the tour of the program under the Help menu.")
	}

	content := container.New(layout.NewVBoxLayout(), text, checkbox)

	dialog.ShowCustom("Welcome to GooeyLife!", "Let me play", content, mainWindow)
}
