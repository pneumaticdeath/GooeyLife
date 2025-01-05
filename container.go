package main

import (
	"github.com/pneumaticdeath/golife"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// LifeContainer is the overall container managing a single
// simulation.  There is one per tab currently.  This has
// three major components.

type LifeContainer struct {
	widget.BaseWidget

	container *fyne.Container

	// The Sim element contains the basic logic of
	// the simulation, and encapsulates the logic
	// from the golife.Game class and drawing
	// the field of cells. It also dealth with
	// the logic of managing the part of the
	// population of cells that is visible on screen.
	Sim *LifeSim

	// The Control element manages all aspects of
	// running and controlling the simulation: e.g.
	// starting/stopping and controlling the speed
	Control *ControlBar

	// The Status object is responisble for providing
	// the user with information about the simulation.
	Status *StatusBar
}

func NewLifeContainer() *LifeContainer {
	lc := &LifeContainer{}

	lc.Sim = NewLifeSim()
	lc.Control = NewControlBar(lc.Sim)
	lc.Status = NewStatusBar(lc.Sim, lc.Control)

	scroll := container.NewScroll(lc.Sim)
	scroll.Direction = container.ScrollNone
	lc.container = container.NewBorder(lc.Control, lc.Status, nil, nil, scroll)

	lc.ExtendBaseWidget(lc)
	return lc
}

func (lc *LifeContainer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(lc.container)
}

func (lc *LifeContainer) SetGame(game *golife.Game) {
	lc.Control.StopSim()
	lc.Sim.Game = game
	lc.Sim.Game.SetHistorySize(historySize)
	lc.Sim.ResizeToFit()
	lc.Sim.Draw()
}
