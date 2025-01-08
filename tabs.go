package main

import (
	"fmt"
	"path/filepath"

	"github.com/pneumaticdeath/golife"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// The LifeTabs structure allows the app to have multiple
// games/simulations loaded at once, and each can be
// controlled individually.

type LifeTabs struct {
	widget.BaseWidget

	DocTabs *container.DocTabs
}

func NewLifeTabs(lc *LifeContainer) *LifeTabs {
	lt := &LifeTabs{}

	title := "Blank Game"
	if lc.Sim.Game.Filename != "" {
		title = filepath.Base(lc.Sim.Game.Filename)
	}
	ti := container.NewTabItem(title, lc)
	lt.DocTabs = container.NewDocTabs(ti)

	lt.ExtendBaseWidget(lt)
	return lt
}

func (lt *LifeTabs) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(lt.DocTabs)
}

func (lt *LifeTabs) CurrentLifeContainer() *LifeContainer {
	if lt == nil || lt.DocTabs == nil {
		return nil
	}
	ti := lt.DocTabs.Selected()
	if ti == nil {
		return nil
	}
	co := ti.Content
	if co == nil {
		return nil
	}
	lc, ok := co.(*LifeContainer)
	if !ok {
		// Not sure how this might happen, but perhaps a race
		// condition when a tab is being created or destroyed
		fmt.Println("Unable to convert tab content to LifeContainer")
		return nil
	}
	return lc
}

func (lt *LifeTabs) NewTab(lc *LifeContainer) {
	title := "Blank Game"
	if lc.Sim.Game.Filename != "" {
		title = filepath.Base(lc.Sim.Game.Filename)
	}
	lt.DocTabs.Append(container.NewTabItem(title, lc))
	lt.DocTabs.SelectIndex(len(lt.DocTabs.Items) - 1)
}

func (lt *LifeTabs) SetCurrentGame(game *golife.Game) {
	lc := lt.CurrentLifeContainer()
	lc.SetGame(game)
	title := "Blank Game"
	if game.Filename != "" {
		title = filepath.Base(game.Filename)
	}
	lt.DocTabs.Selected().Text = title
}
