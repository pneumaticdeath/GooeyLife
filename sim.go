package main

import (
	"image/color"
	"math"
	"path/filepath"
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

const (
	simPaused  = iota
	simRunning = iota
	simEditing = iota
)

// glyphScaleThreshold is the minimum Scale at which individual cell glyphs
// are used instead of the raster path. At high zoom, fewer cells are visible
// so the per-object cost is acceptable; at low zoom the raster is used instead.
const glyphScaleThreshold = float32(6.0)

// LifeSim - encapsulates everything about the simulation and displaying it on
// a canvas/container, including the amount of the population that is visible
// (zoom level), but doesn't handle the animation, control or reporting

type LifeSim struct {
	widget.BaseWidget

	Game                         *golife.Game        // The underlying GameOfLife engine
	BoxDisplayMin, BoxDisplayMax fyne.Position       // The viewport into the game in the coordinates of the sim
	Scale                        float32             // points per cell
	LastStepTime                 time.Duration       // Statistic of time taken to calculate the last generation
	LastDrawTime                 time.Duration       // How long it to draw the last frame
	drawingSurface               *fyne.Container     // The actual drawing surface
	State                        binding.Int         // State the game is in.
	useAlphaDensity              bool                // whether to use alpha to adjust color for aggregate pixels
	GlyphStyle                   string              // One of "Rectange", "RoundedRectangle" or "Circle"
	autoZoom                     binding.Bool        // Should the viewport automatically expand (but never contract) to fit the full population
	EditMode                     binding.Bool        // Whether the sim is in editable mode
	drawLock                     sync.Mutex          // Make sure only one goroutine is drawing at any given time
	Dirty                        bool                // Does the screen need to be redrawn
	raster                       *canvas.Raster      // single persistent raster for zoomed-out rendering
	screenCells                  []bool              // flat bool array indexed by py*screenCols+px
	screenCols                   int                 // logical width of screenCells grid
	screenRows                   int                 // logical height of screenCells grid
	rasterCellColor              color.Color         // cell color read by raster pixel function
	rasterBgColor                color.Color         // background color read by raster pixel function
	usingRaster                  bool                // tracks which path was used last frame
	background                   *canvas.Rectangle   // reusable background rectangle for glyph path
	cellPool                     []fyne.CanvasObject // reusable pool of cell glyphs for glyph path
	poolStyle                    string              // GlyphStyle the pool was built for
}

func (ls *LifeSim) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(ls.drawingSurface)
}

func NewLifeSim(menuUpdateCallback func()) *LifeSim {
	sim := &LifeSim{}
	sim.Game = golife.NewGame()
	sim.Game.SetHistorySize(Config.HistorySize())
	sim.drawingSurface = container.NewWithoutLayout()
	sim.ResizeToFit()
	sim.State = binding.NewInt()
	sim.State.Set(simPaused)
	sim.useAlphaDensity = false
	sim.GlyphStyle = "RoundedRectangle"
	sim.autoZoom = binding.NewBool()
	sim.autoZoom.Set(Config.AutoZoomDefault())
	sim.autoZoom.AddListener(binding.NewDataListener(func() { sim.Draw() }))
	sim.autoZoom.AddListener(binding.NewDataListener(menuUpdateCallback))
	sim.EditMode = binding.NewBool()
	sim.EditMode.AddListener(binding.NewDataListener(menuUpdateCallback))
	sim.EditMode.Set(sim.Game.Size() == 0)
	sim.ExtendBaseWidget(sim)
	sim.Dirty = true
	sim.screenCells = make([]bool, 1)
	sim.rasterCellColor = color.White
	sim.rasterBgColor = color.Black
	sim.raster = canvas.NewRasterWithPixels(func(x, y, w, h int) color.Color {
		if sim.screenCols == 0 || sim.screenRows == 0 || w == 0 || h == 0 {
			return sim.rasterBgColor
		}
		px := x * sim.screenCols / w
		py := y * sim.screenRows / h
		if px >= 0 && px < sim.screenCols && py >= 0 && py < sim.screenRows && sim.screenCells[py*sim.screenCols+px] {
			return sim.rasterCellColor
		}
		return sim.rasterBgColor
	})
	sim.drawingSurface.Objects = []fyne.CanvasObject{sim.raster}
	sim.usingRaster = true
	sim.background = canvas.NewRectangle(color.Black)
	sim.cellPool = make([]fyne.CanvasObject, 0, 256)
	sim.poolStyle = ""
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

func (ls *LifeSim) SetEditMode(em bool) {
	ls.EditMode.Set(em)
}

func (ls *LifeSim) IsEditable() bool {
	em, _ := ls.EditMode.Get()
	return em
}

func (ls *LifeSim) SetState(state int) {
	ls.State.Set(state)
}

func (ls *LifeSim) GetState() int {
	state, _ := ls.State.Get()
	return state
}

func (ls *LifeSim) Resize(size fyne.Size) {
	ls.Dirty = true
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
	ls.Dirty = true
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
		ls.Dirty = true
	}
}

func (ls *LifeSim) Scrolled(se *fyne.ScrollEvent) {
	if Config.ScrollAsZoom() {
		// Slightly non-obvious... this will zoom in if DY is negative
		// and zoom out if DY is positive.  -0.25 was chosen to taste
		// dx := float64(se.Scrolled.DX)  // not sure what to so with this
		dy := float64(se.Scrolled.DY)
		if math.Abs(dy) > 0.01 {
			ls.SetAutoZoom(false)
		}
		ls.Zoom(float32(math.Pow(zoomFactor, -0.25*dy)))
		ls.Dirty = true
	} else {
		// We're going to treat this as equivalent to a Dragged event
		de := &fyne.DragEvent{PointEvent: se.PointEvent, Dragged: se.Scrolled}
		ls.Dragged(de)
	}
}

func (ls *LifeSim) GetGameInfo() (string, string) {
	var title string = "Blank Game"
	if ls.Game.Filename != "" {
		title = filepath.Base(ls.Game.Filename)
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

func (ls *LifeSim) ModeColor() color.Color {
	switch ls.GetState() {
	case simPaused:
		return Config.PausedCellColor()
	case simRunning:
		return Config.RunningCellColor()
	case simEditing:
		return Config.EditCellColor()
	default:
		return color.White
	}
}

func (ls *LifeSim) StateLabel() string {
	switch ls.GetState() {
	case simPaused:
		return "Paused"
	case simRunning:
		return "Running"
	case simEditing:
		return "Editing"
	default:
		return "Unknown"
	}
}

func (ls *LifeSim) Draw() {
	ls.AutoZoom()

	windowSize := ls.drawingSurface.Size()
	if windowSize.Width == 0 || windowSize.Height == 0 {
		// fmt.Println("Can't draw on a zero_sized window")
		return
	}

	if !ls.Dirty {
		return
	}
	ls.Dirty = false

	start := time.Now()

	ls.drawLock.Lock()
	defer ls.drawLock.Unlock()

	population := ls.Game.Population // saving the current population in case the underlying population changes during draw

	displayWidth := ls.BoxDisplayMax.X - ls.BoxDisplayMin.X + float32(1.0)
	displayHeight := ls.BoxDisplayMax.Y - ls.BoxDisplayMin.Y + float32(1.0)

	ls.Scale = min(windowSize.Width/displayWidth, windowSize.Height/displayHeight)

	displayCenter := fyne.NewPos((ls.BoxDisplayMax.X+ls.BoxDisplayMin.X)/2.0,
		(ls.BoxDisplayMax.Y+ls.BoxDisplayMin.Y)/2.0)

	windowCenter := fyne.NewPos(windowSize.Width/2.0, windowSize.Height/2.0)

	if ls.Scale >= glyphScaleThreshold {
		// Glyph path: individual canvas objects per cell.
		// Used at high zoom where fewer cells are visible and visual quality matters.
		cellSize := fyne.NewSize(ls.Scale*0.9, ls.Scale*0.9)
		bgColor := Config.BackgroundColor()
		cellColor := ls.ModeColor()

		if ls.GlyphStyle != ls.poolStyle {
			ls.cellPool = ls.cellPool[:0]
			ls.poolStyle = ls.GlyphStyle
		}
		for len(ls.cellPool) < len(population) {
			switch ls.GlyphStyle {
			case "Rectangle", "RoundedRectangle":
				ls.cellPool = append(ls.cellPool, canvas.NewRectangle(cellColor))
			case "Circle":
				ls.cellPool = append(ls.cellPool, canvas.NewCircle(cellColor))
			default:
				ls.cellPool = append(ls.cellPool, canvas.NewLine(cellColor))
			}
		}

		// Pre-compute visible cell positions off the main thread.
		type cellPos struct {
			obj fyne.CanvasObject
			pos fyne.Position
		}
		visible := make([]cellPos, 0, len(population))
		poolIdx := 0
		for cell := range population {
			window_x := windowCenter.X + ls.Scale*(float32(cell.X)-displayCenter.X) - ls.Scale/2.0
			window_y := windowCenter.Y + ls.Scale*(float32(cell.Y)-displayCenter.Y) - ls.Scale/2.0
			if window_x >= -ls.Scale && window_y >= -ls.Scale && window_x < windowSize.Width+ls.Scale && window_y < windowSize.Height+ls.Scale {
				visible = append(visible, cellPos{ls.cellPool[poolIdx], fyne.NewPos(window_x+ls.Scale/20, window_y+ls.Scale/20)})
				poolIdx++
			}
		}

		// All canvas mutations must happen on the main goroutine.
		fyne.Do(func() {
			ls.background.FillColor = bgColor
			ls.background.Resize(windowSize)
			ls.background.Move(fyne.NewPos(0, 0))

			newObjects := make([]fyne.CanvasObject, 0, len(visible)+1)
			newObjects = append(newObjects, ls.background)

			for _, cp := range visible {
				switch ls.GlyphStyle {
				case "Rectangle":
					rect := cp.obj.(*canvas.Rectangle)
					rect.FillColor = cellColor
					rect.CornerRadius = 0
				case "RoundedRectangle":
					rect := cp.obj.(*canvas.Rectangle)
					rect.FillColor = cellColor
					rect.CornerRadius = ls.Scale / 5.0
				case "Circle":
					cp.obj.(*canvas.Circle).FillColor = cellColor
				default:
					cp.obj.(*canvas.Line).StrokeColor = cellColor
				}
				cp.obj.Resize(cellSize)
				cp.obj.Move(cp.pos)
				newObjects = append(newObjects, cp.obj)
			}

			ls.drawingSurface.RemoveAll()
			for _, obj := range newObjects {
				ls.drawingSurface.Add(obj)
			}
		})
		ls.usingRaster = false
	} else {
		// Raster path: single canvas object for all cells.
		// Used at low zoom where many cells may be visible and efficiency matters.
		ls.rasterBgColor = Config.BackgroundColor()
		ls.rasterCellColor = ls.ModeColor()

		newCols := int(windowSize.Width) + 2
		newRows := int(windowSize.Height) + 2
		if newCols != ls.screenCols || newRows != ls.screenRows {
			ls.screenCols = newCols
			ls.screenRows = newRows
			ls.screenCells = make([]bool, newCols*newRows)
		} else {
			clear(ls.screenCells)
		}

		for cell := range population {
			window_x := windowCenter.X + ls.Scale*(float32(cell.X)-displayCenter.X) - ls.Scale/2.0
			window_y := windowCenter.Y + ls.Scale*(float32(cell.Y)-displayCenter.Y) - ls.Scale/2.0

			if window_x >= -ls.Scale && window_y >= -ls.Scale && window_x < windowSize.Width+ls.Scale && window_y < windowSize.Height+ls.Scale {
				x0 := max(0, int(window_x+ls.Scale/20))
				y0 := max(0, int(window_y+ls.Scale/20))
				x1 := min(ls.screenCols-1, int(window_x+ls.Scale*0.9))
				y1 := min(ls.screenRows-1, int(window_y+ls.Scale*0.9))
				for py := y0; py <= y1; py++ {
					for px := x0; px <= x1; px++ {
						ls.screenCells[py*ls.screenCols+px] = true
					}
				}
			}
		}

		if !ls.usingRaster {
			ls.drawingSurface.Objects = []fyne.CanvasObject{ls.raster}
			ls.usingRaster = true
		}
		fyne.Do(func() { ls.raster.Resize(windowSize) })
	}

	fyne.Do(ls.drawingSurface.Refresh)
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
	ls.Dirty = true
}

func (ls *LifeSim) StopClocks() {
	// Nothing to do here
}
