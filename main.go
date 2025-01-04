package main

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pneumaticdeath/golife"
	"github.com/pneumaticdeath/golife/examples"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	xlayout "fyne.io/x/fyne/layout"
)

const (
	zoomFactor  = 1.1
	shiftFactor = 0.2
	historySize = 50 // really should be configurable
)

var (
	pausedCellColor  color.Color = color.NRGBA{R: 0, G: 0, B: 255, A: 255}
	runningCellColor color.Color = color.NRGBA{R: 0, G: 255, B: 0, A: 255}
	editingCellColor color.Color = color.NRGBA{R: 255, G: 255, B: 0, A: 255}
)

//go:embed Icon.png
var iconPNGData []byte

// LifeContainer is the overall container managing a single
// simulation.  There is one per tab currently.  This has
// three major components.

type LifeContainer struct {
	widget.BaseWidget

	container *fyne.Container

	// The Sim element contains the basic logic of
	// the simulation, and encapsulates the logic
	// from the golife.Game class and drawing
	// the field of cells.
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

	lc.container = container.NewBorder(lc.Control, lc.Status, nil, nil, container.NewScroll(lc.Sim))

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

// LifeSim - encapsulates everything about the simulation and displaying it on
// a canvas/container, but doesn't handle the animation, control or reporting

type LifeSim struct {
	widget.BaseWidget

	Game                         *golife.Game    // The underlying GameOfLife engine
	BoxDisplayMin, BoxDisplayMax fyne.Position   // The viewport into the game in the coordinates of the sim
	Scale                        float32         // points per cell
	LastStepTime                 time.Duration   // Statistic of time taken to calculate the last generation
	LastDrawTime                 time.Duration   // How long it to draw the last frame
	drawingSurface               *fyne.Container // The actual drawing surface
	CellColor                    color.Color     // Color the cells should be draw in
	useAlphaDensity              bool            // whether to use alpha to adjust color for aggregate pixels
	GlyphStyle                   string          // One of "Rectange", "RoundedRectangle" or "Circle"
	BackgroundColor              color.Color     // Should probably be derived from the theme
	autoZoom                     binding.Bool    // Should the viewport automatically expand (but never contract) to fit the full population
	EditMode                     binding.Bool    // Whether the sim is in editable mode
	drawLock                     sync.Mutex      // Make sure only one goroutine is drawing at any given time
}

func (ls *LifeSim) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(ls.drawingSurface)
}

func NewLifeSim() *LifeSim {
	sim := &LifeSim{}
	sim.Game = golife.NewGame()
	sim.Game.SetHistorySize(historySize)
	sim.BoxDisplayMin = fyne.NewPos(0.0, 0.0)
	sim.BoxDisplayMax = fyne.NewPos(10.0, 10.0)
	sim.drawingSurface = container.NewWithoutLayout()
	sim.CellColor = pausedCellColor
	sim.useAlphaDensity = false
	sim.GlyphStyle = "RoundedRectangle"
	sim.BackgroundColor = color.Black
	sim.autoZoom = binding.NewBool()
	sim.autoZoom.Set(true)
	sim.autoZoom.AddListener(binding.NewDataListener(func() { sim.Draw() }))
	sim.EditMode = binding.NewBool()
	sim.EditMode.Set(false)
	sim.ExtendBaseWidget(sim)
	return sim
}

func (ls *LifeSim) MinSize() fyne.Size {
	return fyne.NewSize(150, 150)
}

func (ls *LifeSim) SetAutoZoom(az bool) {
	ls.autoZoom.Set(az)
}

func (ls *LifeSim) IsAutoZoom() bool {
	az, _ := ls.autoZoom.Get()
	return az
}

func (ls *LifeSim) IsEditable() bool {
	em, _ := ls.EditMode.Get()
	return em
}

func (ls *LifeSim) Resize(size fyne.Size) {
	ls.Draw()
	ls.BaseWidget.Resize(size)
}

func (ls *LifeSim) Dragged(e *fyne.DragEvent) {
	if e.Dragged.IsZero() {
		return
	}
	ls.SetAutoZoom(false)
	dx, dy := e.Dragged.Components()
	cells_x := dx / ls.Scale
	cells_y := dy / ls.Scale

	ls.BoxDisplayMin.X, ls.BoxDisplayMax.X = ls.BoxDisplayMin.X-cells_x, ls.BoxDisplayMax.X-cells_x
	ls.BoxDisplayMin.Y, ls.BoxDisplayMax.Y = ls.BoxDisplayMin.Y-cells_y, ls.BoxDisplayMax.Y-cells_y
	ls.Draw()
}

func (ls *LifeSim) DragEnd() {
	// Not much to do here
}

func (ls *LifeSim) Tapped(e *fyne.PointEvent) {
	if ls.IsEditable() {
		// Slightly non-obvious, but the upper left
		// corner of the dislay box is not  necessarily
		// aligned at the uppper left corner, but the
		// center of the display box is always the same
		// as the center of the window
		windowSize := ls.drawingSurface.Size()
		windowCenter_x := windowSize.Width / 2.0
		windowCenter_y := windowSize.Height / 2.0
		boxCenter_x := (ls.BoxDisplayMax.X + ls.BoxDisplayMin.X) / 2.0
		boxCenter_y := (ls.BoxDisplayMax.Y + ls.BoxDisplayMin.Y) / 2.0
		x, y := e.Position.Components()
		cell_x := golife.Coord(math.Floor(float64((x-windowCenter_x)/ls.Scale + boxCenter_x + 0.5)))
		cell_y := golife.Coord(math.Floor(float64((y-windowCenter_y)/ls.Scale + boxCenter_y + 0.5)))
		cell := golife.Cell{cell_x, cell_y}
		if ls.Game.Population[cell] {
			delete(ls.Game.Population, cell)
		} else {
			ls.Game.Population[cell] = true
		}
		ls.Draw()
	}
}

func (ls *LifeSim) GetGameInfo() (string, string) {
	var title string = "Blank Game"
	if ls.Game.Filename != "" {
		title = ls.Game.Filename
	}
	var content strings.Builder
	for _, comment := range ls.Game.Comments {
		if strings.HasPrefix(comment, "#") {
			comment = comment[1:]
		}
		switch {
		case strings.HasPrefix(comment, "N "):
			content.WriteString("Name: ")
			content.WriteString(comment[2:])
		case strings.HasPrefix(comment, "O "):
			content.WriteString("Author: ")
			content.WriteString(comment[2:])
		case strings.HasPrefix(comment, "C "):
			content.WriteString(comment[2:])
		case strings.HasPrefix(comment, " "):
			content.WriteString(comment[1:])
		default:
			content.WriteString(comment)
		}
		content.WriteString("\n")
	}

	return title, content.String()
}

func (ls *LifeSim) Draw() {
	ls.AutoZoom()

	windowSize := ls.drawingSurface.Size()
	if windowSize.Width == 0 || windowSize.Height == 0 {
		// fmt.Println("Can't draw on a zero_sized window")
		return
	}

	start := time.Now()

	ls.drawLock.Lock()
	defer ls.drawLock.Unlock()

	population := ls.Game.Population // saving the current population in case the underlying population changes during draw

	displayWidth := ls.BoxDisplayMax.X - ls.BoxDisplayMin.X + float32(1.0)
	displayHeight := ls.BoxDisplayMax.Y - ls.BoxDisplayMin.Y + float32(1.0)

	ls.Scale = min(windowSize.Width/displayWidth, windowSize.Height/displayHeight)

	cellSize := fyne.NewSize(ls.Scale, ls.Scale)

	displayCenter := fyne.NewPos((ls.BoxDisplayMax.X+ls.BoxDisplayMin.X)/2.0,
		(ls.BoxDisplayMax.Y+ls.BoxDisplayMin.Y)/2.0)

	windowCenter := fyne.NewPos(windowSize.Width/2.0, windowSize.Height/2.0)

	background := canvas.NewRectangle(ls.BackgroundColor)
	background.Resize(windowSize)
	background.Move(fyne.NewPos(0, 0))

	newObjects := make([]fyne.CanvasObject, 0, 1024)

	newObjects = append(newObjects, background)

	pixels := make(map[golife.Cell]int32)
	maxDens := 1

	for cell, _ := range population {
		window_x := windowCenter.X + ls.Scale*(float32(cell.X)-displayCenter.X) - ls.Scale/2.0
		window_y := windowCenter.Y + ls.Scale*(float32(cell.Y)-displayCenter.Y) - ls.Scale/2.0
		cellPos := fyne.NewPos(window_x, window_y)

		if window_x >= -ls.Scale && window_y >= -ls.Scale && window_x < windowSize.Width+ls.Scale && window_y < windowSize.Height+ls.Scale {
			if ls.Scale < 2.0 {
				pixelPos := golife.Cell{golife.Coord(window_x), golife.Coord(window_y)}
				pixels[pixelPos] += 1
				if int(pixels[pixelPos]) > maxDens {
					maxDens = int(pixels[pixelPos])
				}
			} else {
				var cellGlyph fyne.CanvasObject
				switch ls.GlyphStyle {
				case "Rectangle":
					cellGlyph = canvas.NewRectangle(ls.CellColor)
				case "RoundedRectangle":
					tmpRect := canvas.NewRectangle(ls.CellColor)
					tmpRect.CornerRadius = ls.Scale / 5.0
					cellGlyph = tmpRect
				case "Circle":
					cellGlyph = canvas.NewCircle(ls.CellColor)
				default:
					cellGlyph = canvas.NewLine(ls.CellColor)
				}
				cellGlyph.Resize(cellSize)
				cellGlyph.Move(cellPos)

				newObjects = append(newObjects, cellGlyph)
			}
		}
	}

	if ls.Scale < 2.0 && len(pixels) > 0 {
		for pixelPos, count := range pixels {
			var pixelColor color.Color
			if ls.useAlphaDensity {
				density := max(float32(count)/float32(maxDens), float32(0.25))
				r, g, b, a := ls.CellColor.RGBA()
				pixelColor = color.NRGBA{R: uint8(r),
					G: uint8(g),
					B: uint8(b),
					A: uint8(float32(a) * density)}
			} else {
				pixelColor = ls.CellColor
			}
			pixel := canvas.NewRectangle(pixelColor)
			pixel.Resize(fyne.NewSize(2, 2))
			pixel.Move(fyne.NewPos(float32(pixelPos.X), float32(pixelPos.Y)))
			newObjects = append(newObjects, pixel)
		}
	}

	// By reducing the timespan between the removal and re-adding of the objects,
	// flicker is reduced or eliminated.
	ls.drawingSurface.RemoveAll()
	for _, obj := range newObjects {
		ls.drawingSurface.Add(obj)
	}

	ls.drawingSurface.Refresh()
	ls.LastDrawTime = time.Since(start)
}

func (ls *LifeSim) SetDisplayBox(minCorner, maxCorner fyne.Position) {
	if minCorner.X > maxCorner.X {
		ls.BoxDisplayMin = fyne.NewPos(0, 0)
		ls.BoxDisplayMax = fyne.NewPos(10, 10)
	} else {
		ls.BoxDisplayMin, ls.BoxDisplayMax = minCorner, maxCorner
	}
}

func (ls *LifeSim) Zoom(factor float32) {
	ls.BoxDisplayMin.X, ls.BoxDisplayMax.X = scale(ls.BoxDisplayMin.X, ls.BoxDisplayMax.X, factor)
	ls.BoxDisplayMin.Y, ls.BoxDisplayMax.Y = scale(ls.BoxDisplayMin.Y, ls.BoxDisplayMax.Y, factor)
}

func scale(min_v, max_v float32, factor float32) (float32, float32) {
	mid_v := (max_v + min_v) / 2.0
	new_min := (mid_v - (mid_v-min_v)*factor)
	new_max := (mid_v + (max_v-mid_v)*factor)

	if new_max > new_min {
		return new_min, new_max
	} else {
		return min_v, max_v
	}
}

func shift(min_v, max_v, factor float32) (float32, float32) {
	amount := (max_v - min_v) * factor
	if amount == 0 {
		if factor < 0.0 {
			amount = -0.5
		} else if factor > 0.0 {
			amount = 0.5
		}
	}

	return min_v + amount, max_v + amount
}

func (ls *LifeSim) ShiftLeft() {
	ls.BoxDisplayMin.X, ls.BoxDisplayMax.X = shift(ls.BoxDisplayMin.X, ls.BoxDisplayMax.X, -1.0*shiftFactor)
	ls.Draw()
}

func (ls *LifeSim) ShiftRight() {
	ls.BoxDisplayMin.X, ls.BoxDisplayMax.X = shift(ls.BoxDisplayMin.X, ls.BoxDisplayMax.X, shiftFactor)
	ls.Draw()
}

func (ls *LifeSim) ShiftUp() {
	ls.BoxDisplayMin.Y, ls.BoxDisplayMax.Y = shift(ls.BoxDisplayMin.Y, ls.BoxDisplayMax.Y, -1*shiftFactor)
	ls.Draw()
}

func (ls *LifeSim) ShiftDown() {
	ls.BoxDisplayMin.Y, ls.BoxDisplayMax.Y = shift(ls.BoxDisplayMin.Y, ls.BoxDisplayMax.Y, shiftFactor)
	ls.Draw()
}

func (ls *LifeSim) AutoZoom() {
	if !ls.IsAutoZoom() {
		return
	}

	gameCoordMin, gameCoordMax := ls.Game.Population.BoundingBox()

	if float32(gameCoordMin.X) < ls.BoxDisplayMin.X {
		ls.BoxDisplayMin.X = float32(gameCoordMin.X)
	}

	if float32(gameCoordMin.Y) < ls.BoxDisplayMin.Y {
		ls.BoxDisplayMin.Y = float32(gameCoordMin.Y)
	}

	if float32(gameCoordMax.X) > ls.BoxDisplayMax.X {
		ls.BoxDisplayMax.X = float32(gameCoordMax.X)
	}

	if float32(gameCoordMax.Y) > ls.BoxDisplayMax.Y {
		ls.BoxDisplayMax.Y = float32(gameCoordMax.Y)
	}
}

func (ls *LifeSim) ResizeToFit() {
	boxMin, boxMax := ls.Game.Population.BoundingBox()
	newMin, newMax := fyne.NewPos(float32(boxMin.X), float32(boxMin.Y)), fyne.NewPos(float32(boxMax.X), float32(boxMax.Y))
	ls.SetDisplayBox(newMin, newMax)
}

type StatusBar struct {
	widget.BaseWidget
	life                *LifeSim
	control             *ControlBar
	GenerationDisplay   *widget.Label
	CellCountDisplay    *widget.Label
	ScaleDisplay        *widget.Label
	LastStepTimeDisplay *widget.Label
	LastDrawTimeDisplay *widget.Label
	TargetFPSDisplay    *widget.Label
	ActualFPSDisplay    *widget.Label
	UpdateCadence       time.Duration
	bar                 *fyne.Container
}

func NewStatusBar(sim *LifeSim, cb *ControlBar) *StatusBar {
	genDisp := widget.NewLabel("")
	cellCountDisp := widget.NewLabel("")
	scaleDisp := widget.NewLabel("")
	lastStepTimeDisp := widget.NewLabel("")
	lastDrawTimeDisp := widget.NewLabel("")
	targetFPSDisp := widget.NewLabel("")
	actualFPSDisp := widget.NewLabel("")
	statBar := &StatusBar{life: sim, control: cb, GenerationDisplay: genDisp, CellCountDisplay: cellCountDisp,
		ScaleDisplay: scaleDisp, LastStepTimeDisplay: lastStepTimeDisp,
		LastDrawTimeDisplay: lastDrawTimeDisp, TargetFPSDisplay: targetFPSDisp,
		ActualFPSDisplay: actualFPSDisp, UpdateCadence: 20.0 * time.Millisecond}

	statBar.bar = container.New(layout.NewVBoxLayout(),
		container.New(layout.NewHBoxLayout(), widget.NewLabel("Generation:"), statBar.GenerationDisplay,
			layout.NewSpacer(), widget.NewLabel("Live Cells:"), statBar.CellCountDisplay,
			layout.NewSpacer(), widget.NewLabel("Scale:"), statBar.ScaleDisplay),
		container.New(layout.NewHBoxLayout(), widget.NewLabel("Last step time:"), statBar.LastStepTimeDisplay,
			layout.NewSpacer(), widget.NewLabel("Last draw time:"), statBar.LastDrawTimeDisplay,
			layout.NewSpacer(), widget.NewLabel("Target FPS:"), statBar.TargetFPSDisplay,
			widget.NewLabel("Actual FPS:"), statBar.ActualFPSDisplay))

	statBar.ExtendBaseWidget(statBar)

	go func() {
		for {
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
	statBar.CellCountDisplay.SetText(fmt.Sprintf("%d", len(statBar.life.Game.Population)))
	statBar.ScaleDisplay.SetText(fmt.Sprintf("%.3f", statBar.life.Scale))
	statBar.LastStepTimeDisplay.SetText(fmt.Sprintf("%v", statBar.life.LastStepTime))
	statBar.LastDrawTimeDisplay.SetText(fmt.Sprintf("%v", statBar.life.LastDrawTime))
	targetUpdateCadence := time.Duration(math.Pow(10.0, statBar.control.speedSlider.Value)) * time.Millisecond
	statBar.TargetFPSDisplay.SetText(fmt.Sprintf("%.1f", 1.0/targetUpdateCadence.Seconds()))
	statBar.ActualFPSDisplay.SetText(fmt.Sprintf("%.1f", 1.0/statBar.control.updateCadence.Seconds()))
}

func (statBar *StatusBar) Refresh() {
	statBar.Update()
	statBar.BaseWidget.Refresh()
}

type LifeSimClock struct {
	ticker chan bool
	life   *LifeSim
}

func NewLifeSimClock(sim *LifeSim) *LifeSimClock {
	clk := &LifeSimClock{make(chan bool, 1), sim}
	go clk.doTicks()
	return clk
}

func (clk *LifeSimClock) doTicks() {
	for {
		<-clk.ticker // Will block waiting for a clock tick
		start := time.Now()
		clk.life.Game.Next()
		clk.life.LastStepTime = time.Since(start)
		clk.life.Draw()
	}
}

func (clk *LifeSimClock) Tick() {
	clk.ticker <- true // Will block if the last tick hasn't been consumed yet
}

type ControlBar struct {
	widget.BaseWidget
	life               *LifeSim
	clk                *LifeSimClock
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

	controlBar.clk = NewLifeSimClock(sim)

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

	controlBar.zoomFitButton = widget.NewButtonWithIcon("", theme.ZoomFitIcon(), func() { controlBar.life.ResizeToFit(); controlBar.life.Draw() })

	controlBar.zoomInButton = widget.NewButtonWithIcon("", theme.ZoomInIcon(), func() { controlBar.ZoomIn() })

	controlBar.glyphSelector = widget.NewSelect([]string{"Rectangle", "RoundedRectangle", "Circle"}, func(selection string) { controlBar.life.GlyphStyle = selection; controlBar.life.Draw() })
	controlBar.glyphSelector.SetSelected(controlBar.life.GlyphStyle)

	controlBar.editCheckBox = widget.NewCheckWithData("Edit mode", controlBar.life.EditMode)
	controlBar.life.EditMode.AddListener(binding.NewDataListener(func() {
		if controlBar.life.IsEditable() {
			controlBar.StopSim()
			controlBar.life.CellColor = editingCellColor
			controlBar.life.Draw()
		} else {
			controlBar.life.CellColor = pausedCellColor
			controlBar.life.Draw()
		}
	}))

	controlBar.speedSlider = widget.NewSlider(0.5, 3.0) // log_10 scale in milliseconds
	controlBar.speedSlider.SetValue(2.0)                // default to 100ms clock tick time
	controlBar.speedSlider.Step = (3.0 - 0.5) / 12

	fasterLabel := widget.NewLabelWithStyle("faster", fyne.TextAlignTrailing, fyne.TextStyle{})
	controlBar.bar = container.New(layout.NewGridLayout(2),
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
	controlBar.life.Draw()
}

func (controlBar *ControlBar) ZoomOut() {
	controlBar.DisableAutoZoom()
	controlBar.life.Zoom(zoomFactor)
	controlBar.life.Draw()
}

func (controlBar *ControlBar) setRunStopText(label string, icon fyne.Resource) {
	controlBar.runStopButton.SetIcon(icon)
	controlBar.runStopButton.SetText(label)
}

func (controlBar *ControlBar) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(controlBar.bar)
}

func (controlBar *ControlBar) RunGame() {
	controlBar.life.CellColor = runningCellColor
	for controlBar.IsRunning() {
		controlBar.StepForward()
		time.Sleep(time.Duration(math.Pow(10.0, controlBar.speedSlider.Value)) * time.Millisecond)
	}
	if controlBar.life.IsEditable() {
		controlBar.life.CellColor = editingCellColor
	} else {
		controlBar.life.CellColor = pausedCellColor
	}
	controlBar.life.Draw()
}

func (controlBar *ControlBar) StepForward() {
	controlBar.autoZoomCheckBox.SetChecked(controlBar.life.IsAutoZoom())
	controlBar.updateCadence = time.Since(controlBar.lastUpdateTime)
	controlBar.lastUpdateTime = time.Now()
	controlBar.clk.Tick()
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
	controlBar.life.Draw()
}

// The standard FileExtensionFilter only handles simple
// extensinos (e.g. ".rle") but not compound extensions
// like ".rle.txt" that are sometimes the result of
// browsers saving RLE files
type LongExtensionsFileFilter struct {
	storage.FileFilter
	Extensions []string
}

func (filter *LongExtensionsFileFilter) Matches(uri fyne.URI) bool {
	for _, ext := range filter.Extensions {
		if strings.HasSuffix(uri.Name(), ext) {
			return true
		}
	}
	return false
}

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
	co := lt.DocTabs.Selected().Content
	lc, ok := co.(*LifeContainer)
	if !ok {
		fmt.Println("Unable to convert tab content to LifeContainer")
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
			widget.NewLabel(""), widget.NewLabel("written by Mitch Patenaude"))
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
