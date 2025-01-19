package main

import (
	"errors"
	"fmt"
	"net/url"
	"runtime"
	"time"

	"github.com/pneumaticdeath/golife"
	"github.com/pneumaticdeath/golife/examples"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

const (
	zoomFactor  = 1.1
	shiftFactor = 0.2
)

type HelpPage struct {
	Title  string
	URLstr string
}

var HelpPages []HelpPage = []HelpPage{
	{"Getting Started", "https://github.com/pneumaticdeath/GooeyLife/wiki/Getting_Started"},
	{"Tour", "https://github.com/pneumaticdeath/GooeyLife/wiki/Tour"},
	{"Examples", "https://github.com/pneumaticdeath/GooeyLife/wiki/Examples"},
	{"Creating your own", "https://github.com/pneumaticdeath/GooeyLife/wiki/Creating"},
	{"Loading from the web", "https://github.com/pneumaticdeath/GooeyLife/wiki/Loading"},
	{"Reporting bugs", "https://github.com/pneumaticdeath/GooeyLife/wiki/Bugs"},
}

func BuildHelpMenuItems(app fyne.App) []*fyne.MenuItem {
	items := make([]*fyne.MenuItem, 0, len(HelpPages))
	for _, page := range HelpPages {
		url, err := url.Parse(page.URLstr)
		if err != nil {
			fyne.LogError("Unable to parse help url ", err)
		} else {
			mi := fyne.NewMenuItem(page.Title, func() {
				app.OpenURL(url)
			})
			items = append(items, mi)
		}
	}

	return items
}

func BuildExampleMenuItems(loader func(examples.Example) func()) []*fyne.MenuItem {
	exList := examples.ListExamples()
	items := make([]*fyne.MenuItem, 0, 10)
	subItems := make([]*fyne.MenuItem, 0, 10)
	lastCategory := exList[0].Category // assumes at least one example
	for _, ex := range exList {
		if ex.Category != lastCategory {
			newMenuItem := fyne.NewMenuItem(lastCategory, nil)
			newMenuItem.ChildMenu = fyne.NewMenu(lastCategory, subItems...)
			items = append(items, newMenuItem)
			subItems = make([]*fyne.MenuItem, 0, 10)
		}
		subItems = append(subItems, fyne.NewMenuItem(ex.Title, loader(ex)))
		lastCategory = ex.Category
	}
	newMenuItem := fyne.NewMenuItem(lastCategory, nil)
	newMenuItem.ChildMenu = fyne.NewMenu(lastCategory, subItems...)
	items = append(items, newMenuItem)

	return items
}

var mainWindow fyne.Window

func main() {
	myApp := app.NewWithID("io.patenaude.gooeylife")
	InitConfig(myApp)
	mainWindow = myApp.NewWindow("Conway's Game of Life")

	GooeyLifeIconImage := canvas.NewImageFromResource(myApp.Metadata().Icon)
	GooeyLifeIconImage.SetMinSize(fyne.NewSize(128, 128))
	GooeyLifeIconImage.FillMode = canvas.ImageFillContain

	var currentLC *LifeContainer

	var modKey fyne.KeyModifier
	if runtime.GOOS == "darwin" {
		modKey = fyne.KeyModifierSuper
	} else {
		modKey = fyne.KeyModifierControl
	}

	updateSimMenu := func() { /* to be filled in later */ }

	simEditCheckMI := fyne.NewMenuItem("Edit Mode", func() {
		if currentLC != nil {
			currentLC.Sim.SetEditMode(!currentLC.Sim.IsEditable())
			updateSimMenu()
		}
	})
	simEditCheckMI.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyE, Modifier: modKey}

	simAutoZoomCheckMI := fyne.NewMenuItem("Auto Zoom", func() {
		if currentLC != nil {
			currentLC.Sim.SetAutoZoom(!currentLC.Sim.IsAutoZoom())
			updateSimMenu()
		}
	})
	simAutoZoomCheckMI.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyA, Modifier: modKey}

	simZoomFitMI := fyne.NewMenuItem("Zoom To Fit", func() {
		if currentLC != nil {
			currentLC.Sim.ResizeToFit()
		}
	})
	simZoomFitMI.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: modKey}

	updateSimMenu = func() {
		if currentLC != nil {
			simEditCheckMI.Checked = currentLC.Sim.IsEditable()
			simAutoZoomCheckMI.Checked = currentLC.Sim.IsAutoZoom()
		}
	}

	lc := NewLifeContainer(updateSimMenu)

	tabs := NewLifeTabs(lc)
	currentLC = tabs.CurrentLifeContainer()
	displayClock := StartDisplayUpdateClock(tabs)

	tabs.DocTabs.OnSelected = func(ti *container.TabItem) {
		currentLC = tabs.CurrentLifeContainer()
		updateSimMenu()
	}

	tabs.DocTabs.OnClosed = func(ti *container.TabItem) {
		if len(tabs.DocTabs.Items) == 0 {
			if fyne.CurrentDevice().IsMobile() {
				tabs.NewTab(NewLifeContainer(updateSimMenu))
				updateSimMenu()
				tabs.Refresh()
			} else {
				displayClock.Running = false
				// allow the displayClock thread to gracefully exit before we call Quit()
				time.Sleep(50 * time.Millisecond)
				myApp.Quit()
			}
		} else {
			obj := ti.Content
			oldLC, ok := obj.(*LifeContainer)
			if ok {
				// Clean up LC update thread
				oldLC.StopClocks()
			}
			currentLC = tabs.CurrentLifeContainer()
			updateSimMenu()
			tabs.Refresh()
		}
	}

	lifeFileExtensionsFilter := &LongExtensionsFileFilter{Extensions: []string{".rle", ".rle.txt", ".life", ".life.txt", ".cells", ".cells.txt"}}
	saveLifeExtensionsFilter := &LongExtensionsFileFilter{Extensions: []string{".rle", ".rle.txt"}}

	fileOpenCallback := func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, mainWindow)
		} else if reader != nil {
			lifeReader := golife.FindReader(reader.URI().Name())
			newGame, readErr := lifeReader(reader)
			defer reader.Close()
			if readErr != nil {
				dialog.ShowError(readErr, mainWindow)
			} else {
				newGame.Filename = reader.URI().Path()
				tabs.SetCurrentGame(newGame)
				tabs.Refresh()
			}
			// Now we save where we opend this file so that we can default to it next time.
			if reader.URI().Scheme() == "file" {
				Config.SetLastUsedDirURI(reader.URI())
			}
		}
	}

	fileSaveCallback := func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, mainWindow)
		} else if writer != nil && writer.URI().Scheme() == "file" && !saveLifeExtensionsFilter.Matches(writer.URI()) {
			dialog.ShowError(errors.New(fmt.Sprintln("File doesn't have proper extension: ", writer.URI())), mainWindow)
			// writer.Close()  // hack for android save
		}
		if writer != nil {
			write_err := currentLC.Sim.Game.WriteRLE(writer)
			if write_err != nil {
				dialog.ShowError(write_err, mainWindow)
			}
			if writer.URI().Scheme() == "file" {
				Config.SetLastUsedDirURI(writer.URI())
			}
			writer.Close()
		}
	}

	newTabMenuItem := fyne.NewMenuItem("New Tab", func() {
		newlc := NewLifeContainer(updateSimMenu)
		tabs.NewTab(newlc)
	})
	newTabMenuItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyN, Modifier: modKey}

	closeTabMenuItem := fyne.NewMenuItem("Close current tab", func() {
		// clean up LC update thread
		currentLC.StopClocks()
		tabs.DocTabs.RemoveIndex(tabs.DocTabs.SelectedIndex())
		if len(tabs.DocTabs.Items) == 0 {
			if fyne.CurrentDevice().IsMobile() {
				tabs.NewTab(NewLifeContainer(updateSimMenu))
				updateSimMenu()
				tabs.Refresh()
			} else {
				displayClock.Running = false
				// allow the displayClock thread to gracefully exit before we call Quit()
				time.Sleep(60 * time.Millisecond)
				myApp.Quit()
			}
		} else {
			currentLC = tabs.CurrentLifeContainer()
			updateSimMenu()
			tabs.Refresh()
		}
	})
	closeTabMenuItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyW, Modifier: modKey}

	fileOpenMenuItem := fyne.NewMenuItem("Open", func() {
		currentLC.Control.StopSim()
		fileOpen := dialog.NewFileOpen(fileOpenCallback, mainWindow)
		fileOpen.SetFilter(lifeFileExtensionsFilter)
		fileOpen.SetLocation(Config.LastUsedDirURI())
		fileOpen.Show()
	})
	fileOpenMenuItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyO, Modifier: modKey}

	fileSaveMenuItem := fyne.NewMenuItem("Save", func() {
		currentLC.Control.StopSim()
		fileSave := dialog.NewFileSave(fileSaveCallback, mainWindow)
		fileSave.SetFilter(saveLifeExtensionsFilter)
		fileSave.SetLocation(Config.LastUsedDirURI())
		fileSave.Show()
	})
	fileSaveMenuItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: modKey}

	fileInfoMenuItem := fyne.NewMenuItem("Get info", func() {
		title, content := currentLC.Sim.GetGameInfo()
		dialog.ShowInformation(title, content, mainWindow)
	})
	fileInfoMenuItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyI, Modifier: modKey}

	fileAboutMenuItem := fyne.NewMenuItem("About", func() {
		aboutContent := container.New(layout.NewVBoxLayout(), GooeyLifeIconImage,
			widget.NewLabel(myApp.Metadata().Name), widget.NewLabel("Copyright 2024,2025"),
			widget.NewLabel(fmt.Sprintf("Version %s (build %d)", myApp.Metadata().Version, myApp.Metadata().Build)),
			widget.NewLabel("by Mitch Patenaude"),
			widget.NewLabel("Examples copyright of their respective discoverers"))
		aboutDialog := dialog.NewCustom("About GooeyLife", "ok", aboutContent, mainWindow)
		aboutDialog.Show()
	})

	fileSettingsMenuItem := fyne.NewMenuItem("Settings", func() {
		Config.ShowPreferencesDialog(tabs, displayClock)
	})
	fileSettingsMenuItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeySemicolon, Modifier: modKey}

	fileMenu := fyne.NewMenu("File", newTabMenuItem, closeTabMenuItem, fyne.NewMenuItemSeparator(),
		fileOpenMenuItem, fileSaveMenuItem, fyne.NewMenuItemSeparator(), fileInfoMenuItem, fileSettingsMenuItem,
		fileAboutMenuItem)

	exampleLoader := func(e examples.Example) func() {
		return func() {
			newGame := examples.LoadExample(e)
			tabs.SetCurrentGame(newGame)
			tabs.Refresh()
		}
	}
	allExamplesMI := fyne.NewMenuItem("Open all examples", func() {
		exList := examples.ListExamples()
		games := make([]*golife.Game, 0, len(exList))
		for _, ex := range exList {
			games = append(games, examples.LoadExample(ex))
		}
		remaining := games
		if len(currentLC.Sim.Game.Population) == 0 {
			tabs.SetCurrentGame(games[0])
			remaining = games[1:]
		}
		for gameIndex := range remaining {
			lc = NewLifeContainer(updateSimMenu)
			lc.SetGame(remaining[gameIndex])
			tabs.NewTab(lc)
		}
		tabs.Refresh()
	})

	examplesMenu := fyne.NewMenu("Examples", BuildExampleMenuItems(exampleLoader)...)
	examplesMenu.Items = append(examplesMenu.Items, fyne.NewMenuItemSeparator(), allExamplesMI)

	helpMenu := fyne.NewMenu("Help", BuildHelpMenuItems(myApp)...)

	updateSimMenu() // set initial state

	simClearMI := fyne.NewMenuItem("Clear Pattern", func() {
		dialog.ShowConfirm("Are you sure?", "Are you sure you want to clear the current pattern?",
			func(yes bool) {
				if yes {
					tabs.SetCurrentGame(golife.NewGame())
				}
			}, mainWindow)
	})

	simMenu := fyne.NewMenu("Sim", simAutoZoomCheckMI, simZoomFitMI, simEditCheckMI, simClearMI)

	mainMenu := fyne.NewMainMenu(fileMenu, simMenu, examplesMenu, helpMenu)

	mainWindow.SetMainMenu(mainMenu)

	mainWindow.SetContent(tabs)

	toggleRun := func(shortcut fyne.Shortcut) {
		if currentLC.Control.IsRunning() {
			currentLC.Control.StopSim()
		} else {
			currentLC.Control.StartSim()
		}
	}

	mainWindow.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyR, Modifier: modKey}, toggleRun)
	keyPressHandler := func(keyEvent *fyne.KeyEvent) {
		switch keyEvent.Name {
		case fyne.KeyUp:
			currentLC.Sim.ShiftUp()
		case fyne.KeyDown:
			currentLC.Sim.ShiftDown()
		case fyne.KeyLeft:
			currentLC.Sim.ShiftLeft()
		case fyne.KeyRight:
			currentLC.Sim.ShiftRight()
		case fyne.KeyR:
			toggleRun(nil)
		default:
			// fmt.Println("Got unexpected key", keyEvent.Name)
		}
	}
	mainWindow.Canvas().SetOnTypedKey(keyPressHandler)

	mainWindow.SetOnDropped(func(pos fyne.Position, files []fyne.URI) {
		if len(files) >= 1 {
			games := make([]*golife.Game, 0, len(files))
			for index := range files {
				gameParser := golife.FindReader(files[index].Name())
				gameReader, err := storage.Reader(files[index])
				if err != nil {
					dialog.ShowError(err, mainWindow)
					continue
				}
				newGame, err := gameParser(gameReader)
				if err != nil {
					dialog.ShowError(err, mainWindow)
				} else if newGame != nil {
					newGame.Filename = files[index].Path()
					games = append(games, newGame)
				}
			}

			remaining := games
			if len(currentLC.Sim.Game.Population) == 0 {
				currentLC.Control.StopSim()
				tabs.SetCurrentGame(games[0])
				remaining = games[1:]
			}
			for index := range remaining {
				lc = NewLifeContainer(updateSimMenu)
				lc.SetGame(remaining[index])
				tabs.NewTab(lc)
			}
			tabs.Refresh()
		}
	})

	if Config.ShowGuidedTour() {
		ShowGuidedTour()
	}

	// This is a workaround for a bug in Linux
	// initial layout.
	mainWindow.Resize(fyne.NewSize(1028, 770))
	mainWindow.Show()
	mainWindow.Resize(fyne.NewSize(1024, 768))
	myApp.Run()
	displayClock.Running = false
}
