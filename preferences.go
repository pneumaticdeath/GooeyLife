package main

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"github.com/pneumaticdeath/golife"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	gameHistorySizeKey    = "io.patenaude.gooeylife.history_size"
	autoZoomDefaultKey    = "io.patenaude.gooeylife.auto_zoom_default"
	displayRefreshRateKey = "io.patenaude.gooeylife.display_refresh_rate"
	lastUsedDirectoryKey  = "io.patenaude.gooeylife.last_dir_url"
	pausedCellColorKey    = "io.patenaude.gooeylife.paused_color"
	runningCellColorKey   = "io.patenaude.gooeylife.running_color"
	editCellColorKey      = "io.patenaude.gooeyLife.edit_color"
	backgroundColorKey    = "io.patenaude.gooeyLife.background_color"
	showGuidedTourKey     = "io.patenaude.gooeylife.guided_tour"
	scrollAsZoomKey       = "io.patenaude.gooeylife.scroll_as_zoom"
	savedGamesKey         = "io.patenaude.gooeylife.saved_games"
	defaultHistorySize    = 10
)

var (
	defaultPausedColor  color.Color = color.NRGBA{R: 0, G: 0, B: 255, A: 255}
	defaultRunningColor color.Color = color.NRGBA{R: 0, G: 255, B: 0, A: 255}
	defaultEditColor    color.Color = color.NRGBA{R: 255, G: 255, B: 0, A: 255}
	defaultBGColor      color.Color = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
)

type ConfigT struct {
	app fyne.App
}

var Config ConfigT

func InitConfig(app fyne.App) {
	Config.app = app
}

func (c ConfigT) HistorySize() int {
	return c.app.Preferences().IntWithFallback(gameHistorySizeKey, defaultHistorySize)
}

func (c ConfigT) SetHistorySize(value int) {
	c.app.Preferences().SetInt(gameHistorySizeKey, value)
}

func (c ConfigT) AutoZoomDefault() bool {
	return c.app.Preferences().BoolWithFallback(autoZoomDefaultKey, true)
}

func (c ConfigT) SetAutoZoomDefault(value bool) {
	c.app.Preferences().SetBool(autoZoomDefaultKey, value)
}

func (c ConfigT) ShowGuidedTour() bool {
	return c.app.Preferences().BoolWithFallback(showGuidedTourKey, true)
}

func (c ConfigT) SetShowGuidedTour(show bool) {
	c.app.Preferences().SetBool(showGuidedTourKey, show)
}

func (c ConfigT) ScrollAsZoom() bool {
	return c.app.Preferences().BoolWithFallback(scrollAsZoomKey, true)
}

func (c ConfigT) SetScrollAsZoom(saz bool) {
	c.app.Preferences().SetBool(scrollAsZoomKey, saz)
}

func (c ConfigT) DisplayRefreshRate() int {
	return c.app.Preferences().IntWithFallback(displayRefreshRateKey, 60)
}

func (c ConfigT) SetDisplayRefreshRate(rate int) {
	c.app.Preferences().SetInt(displayRefreshRateKey, rate)
}

func (c ConfigT) SavedGames() map[string]*golife.Game {
	gameStrs := c.app.Preferences().StringList(savedGamesKey)
	games := make(map[string]*golife.Game)
	blankCounter := 1
	for _, rleData := range gameStrs {
		reader := strings.NewReader(rleData)
		game, err := golife.ReadRLE(reader)
		if err != nil {
			fyne.LogError("Unable to decode saved game", err)
			continue
		}
		name := game.Name
		for name == "" {
			name = fmt.Sprintf("Blank game %d", blankCounter)
			_, present := games[name]
			if present {
				name = ""
			}
			blankCounter += 1
		}
		games[name] = game
	}

	return games
}

func (c ConfigT) SetSavedGames(games map[string]*golife.Game) {
	gameStrs := make([]string, 0, len(games))
	for name, game := range games {
		game.Name = name
		var writer strings.Builder
		err := game.WriteRLE(&writer)
		if err != nil {
			fyne.LogError(fmt.Sprintf("Unable to encode game %s", name), err)
			continue
		}
		gameStrs = append(gameStrs, writer.String())
	}
	c.app.Preferences().SetStringList(savedGamesKey, gameStrs)
}

func (c ConfigT) PausedCellColor() color.Color {
	return c.fetchColor(pausedCellColorKey, defaultPausedColor)
}

func (c ConfigT) SetPausedCellColor(clr color.Color) {
	c.setColor(pausedCellColorKey, clr)
}

func (c ConfigT) RunningCellColor() color.Color {
	return c.fetchColor(runningCellColorKey, defaultRunningColor)
}

func (c ConfigT) SetRunningCellColor(clr color.Color) {
	c.setColor(runningCellColorKey, clr)
}

func (c ConfigT) EditCellColor() color.Color {
	return c.fetchColor(editCellColorKey, defaultEditColor)
}

func (c ConfigT) SetEditCellColor(clr color.Color) {
	c.setColor(editCellColorKey, clr)
}

func (c ConfigT) BackgroundColor() color.Color {
	return c.fetchColor(backgroundColorKey, defaultBGColor)
}

func (c ConfigT) SetBackgroundColor(clr color.Color) {
	c.setColor(backgroundColorKey, clr)
}

func (c ConfigT) fetchColor(key string, def color.Color) color.Color {
	attr := c.app.Preferences().IntListWithFallback(key, make([]int, 0))
	if len(attr) != 4 {
		return def
	}
	// the values we get back at 16 bit scaled, but we neeed 8 bit values
	col := color.NRGBA{R: uint8(attr[0]), G: uint8(attr[1]), B: uint8(attr[2]), A: uint8(attr[3])}
	return col
}

func (c ConfigT) setColor(key string, clr color.Color) {
	r, g, b, a := clr.RGBA()
	attr := []int{int(r), int(g), int(b), int(a)}
	c.app.Preferences().SetIntList(key, attr)
}

func (c ConfigT) ShowPreferencesDialog(tabs *LifeTabs, clk *DisplayUpdateClock) {

	historySizeEntry := widget.NewEntry()
	historySizeEntry.Validator = validation.NewRegexp(`^\d+$`, "non-negative integers only")
	historySizeEntry.SetText(fmt.Sprintf("%d", c.HistorySize()))
	autoZoomDefaultCheck := widget.NewCheck("Auto Zoom by default", func(_ bool) {})
	autoZoomDefaultCheck.SetChecked(c.AutoZoomDefault())
	displayRefreshRateSelector := widget.NewSelect([]string{"30Hz", "60Hz"}, func(_ string) {})
	if clk.DisplayUpdateHz == 60 {
		displayRefreshRateSelector.SetSelectedIndex(1)
	} else {
		displayRefreshRateSelector.SetSelectedIndex(0)
	}
	scrollAsZoomRadioGroup := widget.NewRadioGroup([]string{"scroll", "zoom"}, nil)
	if c.ScrollAsZoom() {
		scrollAsZoomRadioGroup.SetSelected("zoom")
	} else {
		scrollAsZoomRadioGroup.SetSelected("scroll")
	}
	pausedColorPickerButton := widget.NewButtonWithIcon("Paused cells", theme.ColorPaletteIcon(), func() {
		picker := dialog.NewColorPicker("Paused Cell Color", "", func(clr color.Color) {
			c.SetPausedCellColor(clr)
			mainWindow.Canvas().Content().Refresh()
		}, mainWindow)
		picker.Advanced = true
		picker.SetColor(c.PausedCellColor())
		picker.Show()
	})
	runningColorPickerButton := widget.NewButtonWithIcon("Running cells", theme.ColorPaletteIcon(), func() {
		picker := dialog.NewColorPicker("Running Cell Color", "", func(clr color.Color) {
			c.SetRunningCellColor(clr)
			mainWindow.Canvas().Content().Refresh()
		}, mainWindow)
		picker.Advanced = true
		picker.SetColor(c.RunningCellColor())
		picker.Show()
	})
	editColorPickerButton := widget.NewButtonWithIcon("Editing cells", theme.ColorPaletteIcon(), func() {
		picker := dialog.NewColorPicker("Editing Cell Color", "", func(clr color.Color) {
			c.SetEditCellColor(clr)
			mainWindow.Canvas().Content().Refresh()
		}, mainWindow)
		picker.Advanced = true
		picker.SetColor(c.EditCellColor())
		picker.Show()
	})
	backgroundColorPickerButton := widget.NewButtonWithIcon("Background", theme.ColorPaletteIcon(), func() {
		picker := dialog.NewColorPicker("Background Color", "", func(clr color.Color) {
			c.SetBackgroundColor(clr)
			mainWindow.Canvas().Content().Refresh()
		}, mainWindow)
		picker.Advanced = true
		picker.SetColor(c.BackgroundColor())
		picker.Show()
	})
	entries := []*widget.FormItem{
		widget.NewFormItem("Saved Generations", historySizeEntry),
		widget.NewFormItem("Auto-zoom enabled by default", autoZoomDefaultCheck),
		widget.NewFormItem("Diplay refresh rate", displayRefreshRateSelector),
		widget.NewFormItem("Mouse wheel function", scrollAsZoomRadioGroup),
		widget.NewFormItem("Paused Cell Color", pausedColorPickerButton),
		widget.NewFormItem("Running Cell Color", runningColorPickerButton),
		widget.NewFormItem("Editing Cell Color", editColorPickerButton),
		widget.NewFormItem("Background Color", backgroundColorPickerButton)}

	dialog.ShowForm("Preferences", "Save", "Cancel", entries, func(save bool) {
		if save {
			historySize, err := strconv.Atoi(historySizeEntry.Text)
			if err != nil {
				fmt.Println("Prefernces let through a bad value: ", historySizeEntry.Text, err)
			} else {
				if historySize != c.HistorySize() {
					// update all existing games
					for _, lc := range tabs.GetLifeContainters() {
						lc.Sim.Game.SetHistorySize(historySize)
					}
					// persist the new value
					c.SetHistorySize(historySize)
				}
			}
			c.SetAutoZoomDefault(autoZoomDefaultCheck.Checked)
			if displayRefreshRateSelector.SelectedIndex() == 0 {
				c.SetDisplayRefreshRate(30)
			} else {
				c.SetDisplayRefreshRate(60)
			}
			clk.DisplayUpdateHz = c.DisplayRefreshRate()
			c.SetScrollAsZoom(scrollAsZoomRadioGroup.Selected == "zoom")
		}
	}, mainWindow)
}

func (c *ConfigT) LastUsedDirURI() fyne.ListableURI {
	lastUriString := c.app.Preferences().StringWithFallback(lastUsedDirectoryKey, "")
	if lastUriString == "" {
		return nil
	}
	uri := storage.NewURI(lastUriString)
	listable, err := storage.CanList(uri)
	if err != nil {
		return nil
	}
	for !listable {
		uri, err = storage.Parent(uri)
		if err != nil {
			fyne.LogError("Unable to walk up file tree", err)
			return nil
		}
		listable, err = storage.CanList(uri)
	}
	listableUri, err := storage.ListerForURI(uri)
	if err != nil {
		fyne.LogError("Can't make listable URI:", err)
		return nil
	}
	return listableUri
}

func (c *ConfigT) SetLastUsedDirURI(uri fyne.URI) {
	// parent := uri // make a copy of the URI
	listable, err := storage.CanList(uri)
	if err != nil {
		fyne.LogError("Can't check listability of URI:", err)
		return
	}
	for !listable {
		uri, err = storage.Parent(uri)
		if err != nil {
			fyne.LogError("Unable to walk up file tree", err)
			return
		}
		listable, _ = storage.CanList(uri)
	}
	c.app.Preferences().SetString(lastUsedDirectoryKey, uri.String())
}
