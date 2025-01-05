package main

import (
	"bytes"
	_ "embed"
	"errors"
	"image/color"
	"os"
	"runtime"

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
	historySize = 10 // really should be configurable
)

var (
	pausedCellColor  color.Color = color.NRGBA{R: 0, G: 0, B: 255, A: 255}
	runningCellColor color.Color = color.NRGBA{R: 0, G: 255, B: 0, A: 255}
	editingCellColor color.Color = color.NRGBA{R: 255, G: 255, B: 0, A: 255}
)

//go:embed Icon.png
var iconPNGData []byte

func BuildExampleMenuItems(loader func(examples.Example) func()) []*fyne.MenuItem {
	exList := examples.ListExamples()
	items := make([]*fyne.MenuItem, 0, len(exList))

	for _, ex := range exList {
		items = append(items, fyne.NewMenuItem(ex.Title, loader(ex)))
	}

	return items
}

func main() {
	myApp := app.NewWithID("io.patenaude.guiLife")
	myWindow := myApp.NewWindow("Conway's Game of Life")

	pngReader := bytes.NewReader(iconPNGData)
	GuiLifeIconImage := canvas.NewImageFromReader(pngReader, "Icon.png")
	GuiLifeIconImage.SetMinSize(fyne.NewSize(128, 128))
	GuiLifeIconImage.FillMode = canvas.ImageFillContain

	lc := NewLifeContainer()

	if len(os.Args) > 1 {
		newGame, err := golife.Load(os.Args[1])
		if err != nil {
			dialog.ShowError(err, myWindow)
		} else {
			lc.SetGame(newGame)
		}
	}

	tabs := NewLifeTabs(lc)
	currentLC := tabs.CurrentLifeContainer()

	tabs.DocTabs.OnSelected = func(ti *container.TabItem) {
		currentLC = tabs.CurrentLifeContainer()
	}

	tabs.DocTabs.OnClosed = func(ti *container.TabItem) {
		if len(tabs.DocTabs.Items) == 0 {
			myApp.Quit()
		} else {
			tabs.Refresh()
			currentLC = tabs.CurrentLifeContainer()
		}
	}

	if len(os.Args) > 2 {
		remaining := os.Args[2:]
		for index := range remaining {
			newGame, err := golife.Load(remaining[index])
			if err != nil {
				dialog.ShowError(err, myWindow)
			} else {
				nextlc := NewLifeContainer()
				nextlc.SetGame(newGame)
				tabs.NewTab(nextlc)
			}
		}
	}

	lifeFileExtensionsFilter := &LongExtensionsFileFilter{Extensions: []string{".rle", ".rle.txt", ".life", ".life.txt", ".cells", ".cells.txt"}}
	saveLifeExtensionsFilter := &LongExtensionsFileFilter{Extensions: []string{".rle", ".rle.txt"}}

	var lastDirURI fyne.ListableURI

	fileOpenCallback := func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, myWindow)
		} else if reader != nil {
			lifeReader := golife.FindReader(reader.URI().Name())
			newGame, readErr := lifeReader(reader)
			defer reader.Close()
			if readErr != nil {
				dialog.ShowError(readErr, myWindow)
			} else {
				newGame.Filename = reader.URI().Path()
				tabs.SetCurrentGame(newGame)
				tabs.Refresh()
			}
			// Now we save where we opend this file so that we can default to it next time.
			parentURI, parErr := storage.Parent(reader.URI())
			if parErr != nil {
				dialog.ShowError(parErr, myWindow)
			} else {
				tmpURI, uriErr := storage.ListerForURI(parentURI)
				if uriErr != nil {
					dialog.ShowError(uriErr, myWindow)
				} else {
					lastDirURI = tmpURI
				}
			}
		}
	}

	fileSaveCallback := func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, myWindow)
		} else if writer != nil && !saveLifeExtensionsFilter.Matches(writer.URI()) {
			dialog.ShowError(errors.New("File doesn't have proper extension"), myWindow)
			writer.Close()
			/* // Don't actually delete for now
			   delErr := storage.Delete(writer.URI())
			   if delErr != nil {
			       dialog.ShowError(delErr, myWindow)
			   }
			*/
		} else if writer != nil {
			write_err := currentLC.Sim.Game.WriteRLE(writer)
			if write_err != nil {
				dialog.ShowError(write_err, myWindow)
			}
			parURI, parErr := storage.Parent(writer.URI())
			if parErr != nil {
				dialog.ShowError(parErr, myWindow)
			} else {
				tmpURI, uriErr := storage.ListerForURI(parURI)
				if uriErr != nil {
					dialog.ShowError(uriErr, myWindow)
				} else {
					lastDirURI = tmpURI
				}
			}
			writer.Close()
		}
	}

	var modKey fyne.KeyModifier
	if runtime.GOOS == "darwin" {
		modKey = fyne.KeyModifierSuper
	} else {
		modKey = fyne.KeyModifierControl
	}

	newTabMenuItem := fyne.NewMenuItem("New Tab", func() {
		newlc := NewLifeContainer()
		tabs.NewTab(newlc)
	})
	newTabMenuItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyN, Modifier: modKey}

	closeTabMenuItem := fyne.NewMenuItem("Close current tab", func() {
		tabs.DocTabs.RemoveIndex(tabs.DocTabs.SelectedIndex())
		if len(tabs.DocTabs.Items) == 0 {
			myApp.Quit()
		} else {
			tabs.Refresh()
			currentLC = tabs.CurrentLifeContainer()
		}
	})
	closeTabMenuItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyW, Modifier: modKey}

	fileOpenMenuItem := fyne.NewMenuItem("Open", func() {
		currentLC.Control.StopSim()
		fileOpen := dialog.NewFileOpen(fileOpenCallback, myWindow)
		fileOpen.SetFilter(lifeFileExtensionsFilter)
		fileOpen.SetLocation(lastDirURI)
		fileOpen.Show()
	})
	fileOpenMenuItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyO, Modifier: modKey}

	fileSaveMenuItem := fyne.NewMenuItem("Save", func() {
		currentLC.Control.StopSim()
		fileSave := dialog.NewFileSave(fileSaveCallback, myWindow)
		fileSave.SetFilter(saveLifeExtensionsFilter)
		fileSave.SetLocation(lastDirURI)
		fileSave.Show()
	})
	fileSaveMenuItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: modKey}

	fileInfoMenuItem := fyne.NewMenuItem("Get info", func() {
		title, content := currentLC.Sim.GetGameInfo()
		dialog.ShowInformation(title, content, myWindow)
	})
	fileInfoMenuItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyI, Modifier: modKey}

	fileAboutMenuItem := fyne.NewMenuItem("About", func() {
		aboutContent := container.New(layout.NewVBoxLayout(), GuiLifeIconImage,
			widget.NewLabel("GuiLife"), widget.NewLabel("Copyright 2024,2025"),
			widget.NewLabel("by Mitch Patenaude"),
			widget.NewLabel("Examples copyright of their respective discoverers"))
		aboutDialog := dialog.NewCustom("About GuiLife", "ok", aboutContent, myWindow)
		aboutDialog.Show()
	})

	fileMenu := fyne.NewMenu("File", newTabMenuItem, closeTabMenuItem, fyne.NewMenuItemSeparator(),
		fileOpenMenuItem, fileSaveMenuItem, fyne.NewMenuItemSeparator(), fileInfoMenuItem, fileAboutMenuItem)

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
			lc = NewLifeContainer()
			lc.SetGame(remaining[gameIndex])
			tabs.NewTab(lc)
		}
		tabs.Refresh()
	})

	examplesMenu := fyne.NewMenu("Examples", BuildExampleMenuItems(exampleLoader)...)
	examplesMenu.Items = append(examplesMenu.Items, fyne.NewMenuItemSeparator(), allExamplesMI)

	mainMenu := fyne.NewMainMenu(fileMenu, examplesMenu)

	myWindow.SetMainMenu(mainMenu)

	myWindow.SetContent(tabs)

	toggleRun := func(shortcut fyne.Shortcut) {
		if currentLC.Control.IsRunning() {
			currentLC.Control.StopSim()
		} else {
			currentLC.Control.StartSim()
		}
	}

	myWindow.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyR, Modifier: modKey}, toggleRun)
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
	myWindow.Canvas().SetOnTypedKey(keyPressHandler)

	myWindow.SetOnDropped(func(pos fyne.Position, files []fyne.URI) {
		if len(files) >= 1 {
			games := make([]*golife.Game, 0, len(files))
			for index := range files {
				gameParser := golife.FindReader(files[index].Name())
				gameReader, err := storage.Reader(files[index])
				if err != nil {
					dialog.ShowError(err, myWindow)
					continue
				}
				newGame, err := gameParser(gameReader)
				if err != nil {
					dialog.ShowError(err, myWindow)
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
				lc = NewLifeContainer()
				lc.SetGame(remaining[index])
				tabs.NewTab(lc)
			}
			tabs.Refresh()
		}
	})

	// This is a workaround for a bug in Linux
	// initial layout.
	myWindow.Resize(fyne.NewSize(1028, 770))
	myWindow.Show()
	myWindow.Resize(fyne.NewSize(1024, 768))
	myApp.Run()
}
