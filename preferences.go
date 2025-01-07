package main

import (
	"fmt"
	"image/color"
	"strconv"

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
	defaultHistorySize    = 10
)

var (
	defaultPausedColor  color.Color = color.NRGBA{R: 0, G: 0, B: 255, A: 255}
	defaultRunningColor color.Color = color.NRGBA{R: 0, G: 255, B: 0, A: 255}
	defaultEditColor    color.Color = color.NRGBA{R: 255, G: 255, B: 0, A: 255}
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

func (c ConfigT) DisplayRefreshRate() int {
	return c.app.Preferences().IntWithFallback(displayRefreshRateKey, 60)
}

func (c ConfigT) SetDisplayRefreshRate(rate int) {
	c.app.Preferences().SetInt(displayRefreshRateKey, rate)
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

func (c ConfigT) ShowPreferencesDialog(clk *DisplayUpdateClock) {

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
	entries := []*widget.FormItem{
		widget.NewFormItem("Saved Generations", historySizeEntry),
		widget.NewFormItem("Auto-zoom enabled by default", autoZoomDefaultCheck),
		widget.NewFormItem("Diplay refresh rate", displayRefreshRateSelector),
		widget.NewFormItem("Paused Cell Color", pausedColorPickerButton),
		widget.NewFormItem("Running Cell Color", runningColorPickerButton),
		widget.NewFormItem("Editing Cell Color", editColorPickerButton)}

	dialog.ShowForm("Preferences", "Save", "Cancel", entries, func(save bool) {
		if save {
			historySize, err := strconv.Atoi(historySizeEntry.Text)
			if err != nil {
				fmt.Println("Prefernces let through a bad value: ", historySizeEntry.Text, err)
			} else {
				c.SetHistorySize(historySize)
			}
			c.SetAutoZoomDefault(autoZoomDefaultCheck.Checked)
			if displayRefreshRateSelector.SelectedIndex() == 0 {
				c.SetDisplayRefreshRate(30)
			} else {
				c.SetDisplayRefreshRate(60)
			}
			clk.DisplayUpdateHz = c.DisplayRefreshRate()
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
