package main

import (
	"fmt"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	xlayout "fyne.io/x/fyne/layout"
)

// The ControlBar structure controls all aspects
// of the animaiton and manipulation of the
// running simulation.  It allows the user to
// step the game forward (to the next generation
// of cells), backward (if the hisotry has any
// previous generations), or to run automatically
// at a given speed.  Some functions (like the
// zoom functions) have to be passed down to the
// LifeSim object that encapsulates the game.

type ControlBar struct {
	widget.BaseWidget
	life               *LifeSim
	Clock              *LifeSimClock
	lastUpdateTime     time.Time
	updateCadence      time.Duration
	backwardStepButton *widget.Button
	runStopButton      *widget.Button
	forwardStepButton  *widget.Button
	zoomOutButton      *widget.Button
	autoZoomCheckBox   *widget.Check
	zoomFitButton      *widget.Button
	zoomInButton       *widget.Button
	glyphSelector      *widget.Select
	editCheckBox       *widget.Check
	speedSlider        *widget.Slider
	bar                *fyne.Container
	running            bool
}

func (controlBar *ControlBar) IsRunning() bool {
	return controlBar.running
}

func NewControlBar(sim *LifeSim) *ControlBar {
	controlBar := &ControlBar{}
	controlBar.life = sim

	controlBar.Clock = NewLifeSimClock(sim)

	controlBar.lastUpdateTime = time.Now()
	controlBar.updateCadence = 100 * time.Millisecond

	controlBar.backwardStepButton = widget.NewButtonWithIcon("", theme.MediaSkipPreviousIcon(), func() {
		controlBar.StepBackward()
	})
	if len(controlBar.life.Game.History) == 0 {
		controlBar.backwardStepButton.Disable()
	}

	controlBar.runStopButton = widget.NewButtonWithIcon("Run", theme.MediaPlayIcon(), func() {
		if controlBar.IsRunning() {
			controlBar.StopSim()
		} else {
			controlBar.StartSim()
		}
	})

	controlBar.forwardStepButton = widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), func() {
		if controlBar.IsRunning() {
			controlBar.StopSim() // If we're running, we've probably already calculated the next step
		} else {
			controlBar.StepForward()
		}
	})

	controlBar.zoomOutButton = widget.NewButtonWithIcon("", theme.ZoomOutIcon(), func() { controlBar.ZoomOut() })

	controlBar.autoZoomCheckBox = widget.NewCheckWithData("Auto Zoom", controlBar.life.autoZoom)

	// controlBar.autoZoomCheckBox.SetChecked(controlBar.life.IsAutoZoom())

	controlBar.zoomFitButton = widget.NewButtonWithIcon("", theme.ZoomFitIcon(), func() { controlBar.life.ResizeToFit(); controlBar.life.Dirty = true })

	controlBar.zoomInButton = widget.NewButtonWithIcon("", theme.ZoomInIcon(), func() { controlBar.ZoomIn() })

	controlBar.glyphSelector = widget.NewSelect([]string{"Rectangle", "RoundedRectangle", "Circle"}, func(selection string) { controlBar.life.GlyphStyle = selection; controlBar.life.Dirty = true })
	controlBar.glyphSelector.SetSelected(controlBar.life.GlyphStyle)

	controlBar.editCheckBox = widget.NewCheckWithData("Edit mode", controlBar.life.EditMode)
	controlBar.life.EditMode.AddListener(binding.NewDataListener(func() {
		if controlBar.life.IsEditable() {
			controlBar.StopSim()
			if controlBar.life.Scale < 4.0 {
				confirm := dialog.NewConfirm("Scale is very small",
					"Each cell is very small on the screen.  Would you like to zoom in?",
					func(answer bool) {
						if answer {
							// if we don't disable auto-zoom, it will just zoom right back out
							controlBar.life.SetAutoZoom(false)
							controlBar.life.Zoom(controlBar.life.Scale / 5)
							controlBar.life.Dirty = true
						}
					},
					mainWindow)
				confirm.Show()
			}
			controlBar.life.State = simEditing
			controlBar.life.Dirty = true
		} else {
			controlBar.life.State = simPaused
			controlBar.life.Dirty = true
		}
	}))

	controlBar.speedSlider = widget.NewSlider(0.5, 3.0) // log_10 scale in milliseconds
	controlBar.speedSlider.SetValue(2.0)                // default to 100ms clock tick time
	controlBar.speedSlider.Step = (3.0 - 0.5) / 12

	fasterLabel := widget.NewLabelWithStyle("faster", fyne.TextAlignTrailing, fyne.TextStyle{})
	controlBar.bar = container.New(layout.NewAdaptiveGridLayout(2),
		container.New(layout.NewHBoxLayout(), controlBar.backwardStepButton, controlBar.runStopButton,
			controlBar.forwardStepButton, controlBar.zoomOutButton, controlBar.autoZoomCheckBox,
			controlBar.zoomFitButton, controlBar.zoomInButton, controlBar.glyphSelector, controlBar.editCheckBox),
		container.New(xlayout.NewHPortion([]float64{0.2, 0.6, 0.2}), fasterLabel, controlBar.speedSlider, widget.NewLabel("slower")))

	controlBar.ExtendBaseWidget(controlBar)
	return controlBar
}

func (controlBar *ControlBar) StopSim() {
	if controlBar.IsRunning() {
		controlBar.running = false
		controlBar.setRunStopText("Run", theme.MediaPlayIcon())
	}
}

func (controlBar *ControlBar) StartSim() {
	if !controlBar.IsRunning() {
		controlBar.setRunStopText("Pause", theme.MediaPauseIcon())
		controlBar.running = true
		go controlBar.RunGame()
	}
	if controlBar.life.IsEditable() {
		controlBar.life.EditMode.Set(false)
	}
}

func (controlBar *ControlBar) DisableAutoZoom() {
	controlBar.autoZoomCheckBox.SetChecked(false)
	controlBar.life.SetAutoZoom(false)
}

func (controlBar *ControlBar) ZoomIn() {
	controlBar.DisableAutoZoom()
	controlBar.life.Zoom(1.0 / zoomFactor)
	controlBar.life.Dirty = true
}

func (controlBar *ControlBar) ZoomOut() {
	controlBar.DisableAutoZoom()
	controlBar.life.Zoom(zoomFactor)
	controlBar.life.Dirty = true
}

func (controlBar *ControlBar) setRunStopText(label string, icon fyne.Resource) {
	controlBar.runStopButton.SetIcon(icon)
	controlBar.runStopButton.SetText(label)
}

func (controlBar *ControlBar) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(controlBar.bar)
}

func (controlBar *ControlBar) RunGame() {
	controlBar.life.State = simRunning
	for controlBar.IsRunning() {
		controlBar.StepForward()
		time.Sleep(time.Duration(math.Pow(10.0, controlBar.speedSlider.Value)) * time.Millisecond)
	}
	if controlBar.life.IsEditable() {
		controlBar.life.State = simEditing
	} else {
		controlBar.life.State = simPaused
	}
	controlBar.life.Dirty = true
}

func (controlBar *ControlBar) StepForward() {
	controlBar.autoZoomCheckBox.SetChecked(controlBar.life.IsAutoZoom())
	controlBar.updateCadence = time.Since(controlBar.lastUpdateTime)
	controlBar.lastUpdateTime = time.Now()
	controlBar.Clock.LifeTick()
	if len(controlBar.life.Game.History) > 0 {
		controlBar.backwardStepButton.Enable() // We might have history now
	}
}

func (controlBar *ControlBar) StepBackward() {
	if controlBar.IsRunning() {
		controlBar.StopSim()
	}
	err := controlBar.life.Game.Previous()
	if err != nil {
		fmt.Println("Got error trying to step backwards", err)
	}
	if len(controlBar.life.Game.History) == 0 {
		controlBar.backwardStepButton.Disable()
	}
	controlBar.life.Dirty = true
}
