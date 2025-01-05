package main

import (
	"image/color"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/pneumaticdeath/golife"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

/*
var (
	pausedCellColor  color.Color = color.NRGBA{R: 0, G: 0, B: 255, A: 255}
	runningCellColor color.Color = color.NRGBA{R: 0, G: 255, B: 0, A: 255}
	editingCellColor color.Color = color.NRGBA{R: 255, G: 255, B: 0, A: 255}
)
*/

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
	return fyne.NewSize(150, 150) // This probably shouldn't be hard-coded
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
		if ls.Game.HasCell(cell) {
			ls.Game.RemoveCell(cell)
		} else {
			ls.Game.AddCell(cell)
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

	for cell := range population {
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