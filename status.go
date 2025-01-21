package main

import (
	"fmt"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// The StatusBar is a simple mechanism for displaying the status of
// simulation/game.

type StatusBar struct {
	widget.BaseWidget
	life                *LifeSim
	control             *ControlBar
	GenerationDisplay   *widget.Label
	CellCountDisplay    *widget.Label
	HistorySizeDisplay  *widget.Label
	ScaleDisplay        *widget.Label
	LastStepTimeDisplay *widget.Label
	LastDrawTimeDisplay *widget.Label
	TargetGPSDisplay    *widget.Label
	ActualGPSDisplay    *widget.Label
	UpdateCadence       time.Duration
	ClockRunning        bool
	bar                 *fyne.Container
}

func NewStatusBar(sim *LifeSim, cb *ControlBar) *StatusBar {
	genDisp := widget.NewLabel("")
	cellCountDisp := widget.NewLabel("")
	histSizeDisp := widget.NewLabel("")
	scaleDisp := widget.NewLabel("")
	lastStepTimeDisp := widget.NewLabel("")
	lastDrawTimeDisp := widget.NewLabel("")
	targetGPSDisp := widget.NewLabel("")
	actualGPSDisp := widget.NewLabel("")
	statBar := &StatusBar{life: sim, control: cb, GenerationDisplay: genDisp, CellCountDisplay: cellCountDisp,
		HistorySizeDisplay: histSizeDisp, ScaleDisplay: scaleDisp, LastStepTimeDisplay: lastStepTimeDisp,
		LastDrawTimeDisplay: lastDrawTimeDisp, TargetGPSDisplay: targetGPSDisp,
		ActualGPSDisplay: actualGPSDisp, UpdateCadence: 50.0 * time.Millisecond, ClockRunning: true}

	if fyne.CurrentDevice().IsMobile() {
		statBar.bar = container.New(layout.NewVBoxLayout(),
			container.New(layout.NewHBoxLayout(), widget.NewLabel("Gen:"), statBar.GenerationDisplay,
				layout.NewSpacer(), widget.NewLabel("Cells:"), statBar.CellCountDisplay),
			container.New(layout.NewHBoxLayout(), widget.NewLabel("Target GPS:"), statBar.TargetGPSDisplay,
				layout.NewSpacer(), widget.NewLabel("Actual GPS:"), statBar.ActualGPSDisplay))
	} else {
		statBar.bar = container.New(layout.NewVBoxLayout(),
			container.New(layout.NewHBoxLayout(), widget.NewLabel("Generation:"), statBar.GenerationDisplay,
				layout.NewSpacer(), widget.NewLabel("Available history"), statBar.HistorySizeDisplay,
				layout.NewSpacer(), widget.NewLabel("Live Cells:"), statBar.CellCountDisplay,
				layout.NewSpacer(), widget.NewLabel("Scale:"), statBar.ScaleDisplay),
			container.New(layout.NewHBoxLayout(), widget.NewLabel("Last step time:"), statBar.LastStepTimeDisplay,
				layout.NewSpacer(), widget.NewLabel("Last draw time:"), statBar.LastDrawTimeDisplay,
				layout.NewSpacer(), widget.NewLabel("Target GPS:"), statBar.TargetGPSDisplay,
				widget.NewLabel("Actual GPS:"), statBar.ActualGPSDisplay))
	}

	statBar.ExtendBaseWidget(statBar)

	go func() {
		for statBar.ClockRunning {
			statBar.Refresh()
			time.Sleep(statBar.UpdateCadence)
		}
	}()

	return statBar
}

func (statBar *StatusBar) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(statBar.bar)
}

func (statBar *StatusBar) Update() {
	statBar.GenerationDisplay.SetText(fmt.Sprintf("%d", statBar.life.Game.Generation))
	statBar.CellCountDisplay.SetText(fmt.Sprintf("%d", statBar.life.Game.Size()))
	statBar.HistorySizeDisplay.SetText(fmt.Sprintf("%d of %d", len(statBar.life.Game.History), statBar.life.Game.HistorySize))
	statBar.ScaleDisplay.SetText(fmt.Sprintf("%.3f", statBar.life.Scale))
	statBar.LastStepTimeDisplay.SetText(fmt.Sprintf("%7v", statBar.life.LastStepTime))
	statBar.LastDrawTimeDisplay.SetText(fmt.Sprintf("%7v", statBar.life.LastDrawTime))
	targetUpdateCadence := time.Duration(math.Pow(10.0, statBar.control.speedSlider.Value)) * time.Millisecond
	statBar.TargetGPSDisplay.SetText(fmt.Sprintf("%.1f", 1.0/targetUpdateCadence.Seconds()))
	statBar.ActualGPSDisplay.SetText(fmt.Sprintf("%.1f", 1.0/statBar.control.updateCadence.Seconds()))
}

func (statBar *StatusBar) Refresh() {
	statBar.Update()
	statBar.BaseWidget.Refresh()
}

func (statBar *StatusBar) StopClocks() {
	statBar.ClockRunning = false
}
