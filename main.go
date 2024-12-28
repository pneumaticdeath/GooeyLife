package main

import (
    "errors"
    "fmt"
    "image/color"
    "math"
    "os"
    "runtime"
    "strings"
    "sync"
    "time"

    "github.com/pneumaticdeath/golife"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/canvas"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/driver/desktop"
    "fyne.io/fyne/v2/layout"
    "fyne.io/fyne/v2/storage"
    "fyne.io/fyne/v2/storage/repository"
    "fyne.io/fyne/v2/theme"
    "fyne.io/fyne/v2/widget"
)

const (
    zoomFactor = 1.1
    shiftFactor = 0.2
    historySize = 100
)

type LifeSim struct {
    widget.BaseWidget
    Game                                *golife.Game
    BoxDisplayMin, BoxDisplayMax        golife.Cell
    Scale                               float32 // pixel per cell
    LastStepTime                        time.Duration
    LastDrawTime                        time.Duration
    drawingSurface                      *fyne.Container
    CellColor                           color.Color
    BackgroundColor                     color.Color
    autoZoom                            bool
    drawLock                            sync.Mutex
}

func (ls *LifeSim) CreateRenderer() fyne.WidgetRenderer {
    return widget.NewSimpleRenderer(ls.drawingSurface)
}

func NewLifeSim() *LifeSim {
    sim := &LifeSim{}
    sim.Game = golife.NewGame()
    sim.Game.SetHistorySize(historySize)
    sim.BoxDisplayMin = golife.Cell{0, 0}
    sim.BoxDisplayMax = golife.Cell{10, 10}
    sim.drawingSurface = container.NewWithoutLayout()
    sim.CellColor = color.NRGBA{R: 0, G: 0, B: 255, A: 255}
    sim.BackgroundColor = color.Black
    // sim.BackgroundColor = color.White
    sim.autoZoom = true
    sim.ExtendBaseWidget(sim)
    return sim
}

func (ls *LifeSim) MinSize() fyne.Size {
    return fyne.NewSize(150, 150)
}

func (ls *LifeSim) SetAutoZoom(az bool) {
    ls.autoZoom = az
}

func (ls *LifeSim) IsAutoZoom() bool {
    return ls.autoZoom
}

func (ls *LifeSim) Resize(size fyne.Size) {
    ls.Draw()
    ls.BaseWidget.Resize(size)
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

    displayWidth := float32(ls.BoxDisplayMax.X - ls.BoxDisplayMin.X + 1)
    displayHeight := float32(ls.BoxDisplayMax.Y - ls.BoxDisplayMin.Y + 1)

    ls.Scale = min(windowSize.Width / displayWidth, windowSize.Height / displayHeight)

    cellSize := fyne.NewSize(ls.Scale, ls.Scale)

    displayCenter := fyne.NewPos(float32(ls.BoxDisplayMax.X + ls.BoxDisplayMin.X)/2.0, 
                                float32(ls.BoxDisplayMax.Y + ls.BoxDisplayMin.Y)/2.0)
    
    windowCenter := fyne.NewPos(windowSize.Width/2.0, windowSize.Height/2.0)

    pixels := make(map[golife.Cell]int32)
    maxDens := 1

    ls.drawingSurface.RemoveAll()
    background := canvas.NewRectangle(ls.BackgroundColor)
    background.Resize(windowSize)
    background.Move(fyne.NewPos(0,0))

    ls.drawingSurface.Add(background)

    for cell, _ := range population {
        window_x := windowCenter.X + ls.Scale * (float32(cell.X) - displayCenter.X) - ls.Scale/2.0
        window_y := windowCenter.Y + ls.Scale * (float32(cell.Y) - displayCenter.Y) - ls.Scale/2.0
        cellPos := fyne.NewPos(window_x, window_y)

        if window_x >= -0.5 && window_y >= -0.5 && window_x < windowSize.Width - ls.Scale/2.0 && window_y < windowSize.Height - ls.Scale/2.0 {
            if ls.Scale < 2.0 {
                pixelPos := golife.Cell{golife.Coord(window_x), golife.Coord(window_y)}
                pixels[pixelPos] += 1
                if int(pixels[pixelPos]) > maxDens {
                    maxDens = int(pixels[pixelPos])
                }
            } else {
                cellCircle := canvas.NewCircle(ls.CellColor)
                cellCircle.Resize(cellSize)
                cellCircle.Move(cellPos)

                ls.drawingSurface.Add(cellCircle)
            }
        }
    }

    if ls.Scale < 2.0 && len(pixels) > 0 {
        for pixelPos, count := range pixels {
            density := max(float32(count)/float32(maxDens), float32(0.25))
            r, g, b, a := ls.CellColor.RGBA()
            pixelColor := color.NRGBA{R: uint8(r),
                                      G: uint8(g),
                                      B: uint8(b),
                                      A: uint8(float32(a)*density)}
            pixel := canvas.NewRectangle(pixelColor)
            pixel.Resize(fyne.NewSize(2, 2))
            pixel.Move(fyne.NewPos(float32(pixelPos.X), float32(pixelPos.Y)))
            ls.drawingSurface.Add(pixel)
        }
    }

    ls.drawingSurface.Refresh()
    ls.LastDrawTime = time.Since(start)
}

func (ls *LifeSim) SetDisplayBox(minCorner, maxCorner golife.Cell) {
    if minCorner.X > maxCorner.X {
        ls.BoxDisplayMin = golife.Cell{0, 0}
        ls.BoxDisplayMax = golife.Cell{10, 10}
    } else {
        ls.BoxDisplayMin, ls.BoxDisplayMax = minCorner, maxCorner
    }
}

func (ls *LifeSim) Zoom(factor float32) {
    ls.BoxDisplayMin.X, ls.BoxDisplayMax.X = scale(ls.BoxDisplayMin.X, ls.BoxDisplayMax.X, factor)
    ls.BoxDisplayMin.Y, ls.BoxDisplayMax.Y = scale(ls.BoxDisplayMin.Y, ls.BoxDisplayMax.Y, factor)
}

func scale(min_v, max_v golife.Coord, factor float32) (golife.Coord, golife.Coord) {
    mid_v := float32(max_v + min_v)/2.0
    new_min := golife.Coord(mid_v - (mid_v - float32(min_v))*factor)
    new_max := golife.Coord(mid_v + (float32(max_v) - mid_v)*factor)

    if new_min == min_v {
        if factor < 1.0 {
            new_min += 1
        } else if factor > 1.0 {
            new_min -= 1
        }
    }

    if new_max == max_v {
        if factor < 1.0 {
            new_max -= 1
        } else if factor > 1.0 {
            new_max += 1
        }
    }

    if new_max > new_min {
        return new_min, new_max
    } else {
        return min_v, max_v
    }
}

func shift(min_v, max_v golife.Coord, factor float64) (golife.Coord, golife.Coord) {
    amount := golife.Coord(math.Floor(float64(max_v - min_v)*factor + 0.5))
    if amount == 0 {
        if factor < 0.0 {
            amount = golife.Coord(-1)
        } else if factor > 0.0 {
            amount = golife.Coord(1)
        }
    }

    return min_v + amount, max_v + amount
}

func(ls *LifeSim) ShiftLeft() {
    ls.BoxDisplayMin.X, ls.BoxDisplayMax.X = shift(ls.BoxDisplayMin.X, ls.BoxDisplayMax.X, -1*shiftFactor)
    ls.Draw()
}

func(ls *LifeSim) ShiftRight() {
    ls.BoxDisplayMin.X, ls.BoxDisplayMax.X = shift(ls.BoxDisplayMin.X, ls.BoxDisplayMax.X, shiftFactor)
    ls.Draw()
}

func(ls *LifeSim) ShiftUp() {
    ls.BoxDisplayMin.Y, ls.BoxDisplayMax.Y = shift(ls.BoxDisplayMin.Y, ls.BoxDisplayMax.Y, -1*shiftFactor)
    ls.Draw()
}

func(ls *LifeSim) ShiftDown() {
    ls.BoxDisplayMin.Y, ls.BoxDisplayMax.Y = shift(ls.BoxDisplayMin.Y, ls.BoxDisplayMax.Y, shiftFactor)
    ls.Draw()
}

func (ls *LifeSim) AutoZoom() {
    if ! ls.autoZoom {
        return 
    }

    gameBoxMin, gameBoxMax := ls.Game.Population.BoundingBox()

    if gameBoxMin.X < ls.BoxDisplayMin.X {
        ls.BoxDisplayMin.X = gameBoxMin.X
    }

    if gameBoxMin.Y < ls.BoxDisplayMin.Y {
        ls.BoxDisplayMin.Y = gameBoxMin.Y
    }

    if gameBoxMax.X > ls.BoxDisplayMax.X {
        ls.BoxDisplayMax.X = gameBoxMax.X
    }

    if gameBoxMax.Y > ls.BoxDisplayMax.Y {
        ls.BoxDisplayMax.Y = gameBoxMax.Y
    }
}

func (ls *LifeSim) ResizeToFit() {
    boxDisplayMin, boxDisplayMax := ls.Game.Population.BoundingBox()
    ls.SetDisplayBox(boxDisplayMin, boxDisplayMax)
}

type StatusBar struct {
    widget.BaseWidget
    life                *LifeSim
    GenerationDisplay   *widget.Label
    CellCountDisplay    *widget.Label
    ScaleDisplay        *widget.Label
    LastStepTimeDisplay *widget.Label
    LastDrawTimeDisplay *widget.Label
    UpdateCadence       time.Duration
    bar                 *fyne.Container
}

func NewStatusBar(sim *LifeSim) (*StatusBar) {
    genDisp := widget.NewLabel("")
    cellCountDisp := widget.NewLabel("")
    scaleDisp := widget.NewLabel("")
    lastStepTimeDisp := widget.NewLabel("")
    lastDrawTimeDisp := widget.NewLabel("")
    statBar := &StatusBar{life: sim, GenerationDisplay: genDisp, CellCountDisplay: cellCountDisp,
                          ScaleDisplay: scaleDisp, LastStepTimeDisplay: lastStepTimeDisp,
                          LastDrawTimeDisplay: lastDrawTimeDisp, UpdateCadence: 20*time.Millisecond}
    statBar.bar = container.New(layout.NewHBoxLayout(), widget.NewLabel("Generation:"), statBar.GenerationDisplay,
                                layout.NewSpacer(), widget.NewLabel("Live Cells:"), statBar.CellCountDisplay,
                                layout.NewSpacer(), widget.NewLabel("Scale:"), statBar.ScaleDisplay,
                                layout.NewSpacer(), widget.NewLabel("Last step time:"), statBar.LastStepTimeDisplay,
                                layout.NewSpacer(), widget.NewLabel("Last draw time:"), statBar.LastDrawTimeDisplay)

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
    clk := &LifeSimClock{ make(chan bool, 1), sim }
    go clk.doTicks()
    return clk
}

func (clk *LifeSimClock) doTicks() {
    for {
        <-clk.ticker
        start := time.Now()
        clk.life.Game.Next()
        clk.life.LastStepTime = time.Since(start)
        clk.life.Draw()
    }
}

func (clk *LifeSimClock) Tick() {
    clk.ticker <- true
}

type ControlBar struct {
    widget.BaseWidget
    life                *LifeSim
    clk                 *LifeSimClock
    backwardStepButton  *widget.Button
    runStopButton       *widget.Button
    forwardStepButton   *widget.Button
    zoomOutButton       *widget.Button
    autoZoomCheckBox    *widget.Check
    zoomFitButton       *widget.Button
    zoomInButton        *widget.Button
    speedSlider         *widget.Slider
    bar                 *fyne.Container
    running             bool
}

func (controlBar *ControlBar) IsRunning() bool {
    return controlBar.running
}

func NewControlBar(sim *LifeSim) *ControlBar {
    controlBar := &ControlBar{}
    controlBar.life = sim

    controlBar.clk = NewLifeSimClock(sim)

    // Haven't implemented this functionality yet
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
        }})


    controlBar.forwardStepButton = widget.NewButtonWithIcon("", theme.MediaSkipNextIcon(), func() {
        if controlBar.IsRunning() {
            controlBar.StopSim()  // If we're running, we've probably already calculated the next step
        } else {
            controlBar.StepForward()
        }
    })

    controlBar.zoomOutButton = widget.NewButtonWithIcon("", theme.ZoomOutIcon(), func () {controlBar.ZoomOut()})

    controlBar.autoZoomCheckBox = widget.NewCheck("Auto Zoom", func(checked bool) { 
        controlBar.life.SetAutoZoom(checked) 
        if controlBar.life.IsAutoZoom() {
            controlBar.life.Draw()
        }
    })
    controlBar.autoZoomCheckBox.SetChecked(controlBar.life.IsAutoZoom())

    controlBar.zoomFitButton = widget.NewButtonWithIcon("", theme.ZoomFitIcon(), func() {controlBar.life.ResizeToFit(); controlBar.life.Draw()})

    controlBar.zoomInButton = widget.NewButtonWithIcon("", theme.ZoomInIcon(), func () {controlBar.ZoomIn()})

    controlBar.speedSlider = widget.NewSlider(1.5, 500.0) // in milliseconds
    controlBar.speedSlider.SetValue(200.0)

    controlBar.bar = container.New(layout.NewHBoxLayout(), 
                                   controlBar.backwardStepButton, controlBar.runStopButton, controlBar.forwardStepButton, layout.NewSpacer(),
                                   controlBar.zoomOutButton, controlBar.autoZoomCheckBox, controlBar.zoomFitButton, controlBar.zoomInButton, layout.NewSpacer(),
                                   canvas.NewText("faster", color.Black), controlBar.speedSlider, canvas.NewText("slower", color.Black))

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
}

func (controlBar *ControlBar) DisableAutoZoom() {
    controlBar.autoZoomCheckBox.SetChecked(false)
    controlBar.life.SetAutoZoom(false)
}

func (controlBar *ControlBar) ZoomIn() {
    controlBar.DisableAutoZoom()
    controlBar.life.Zoom(1.0/zoomFactor)
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
    // red := color.NRGBA{R: 255, G: 0, B: 0, A: 255}
    blue := color.NRGBA{R: 0, G: 0, B: 255, A: 255}
    green := color.NRGBA{R: 0, G: 255, B: 0, A: 255}

    controlBar.life.CellColor = green
    for controlBar.IsRunning() {
        controlBar.StepForward()
        time.Sleep(time.Duration(controlBar.speedSlider.Value)*time.Millisecond)
    }
    controlBar.life.CellColor = blue
    controlBar.life.Draw()
}

func (controlBar *ControlBar) StepForward() {
    controlBar.clk.Tick()
    controlBar.backwardStepButton.Enable()
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

/* The standard FileExtensionFilter only handles simple 
   extensinos (e.g. ".rle") but not compound extensions
   like ".rle.txt" that are sometimes the result of 
   browsers saving RLE files
*/
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

func main() {
    myApp := app.NewWithID("com.github.pneumaticdeath.guiLife")
    myWindow := myApp.NewWindow("Conway's Game of Life")

    lifeSim := NewLifeSim()

    if len(os.Args) > 1 {
        newGame, err := golife.Load(os.Args[1])
        if err != nil {
            dialog.ShowError(err, myWindow)
        } else {
            lifeSim.Game = newGame
            lifeSim.Game.SetHistorySize(historySize)
        }
    } else {
        lifeSim.Game = golife.NewGame()
    }
    lifeSim.ResizeToFit()

    lifeFileExtensionsFilter := &LongExtensionsFileFilter{Extensions: []string{".rle",".rle.txt",".life",".life.txt"}}
    saveLifeExtensionsFilter := &LongExtensionsFileFilter{Extensions: []string{".rle",".rle.txt"}}

    cwd, err := os.Getwd()
    if err != nil {
        dialog.ShowError(err, myWindow)
    }
    tmpURI := repository.NewFileURI(cwd)
    cwdURI, err := storage.ListerForURI(tmpURI)
    if err != nil {
        dialog.ShowError(err, myWindow)
    }

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
                lifeSim.Game = newGame
                lifeSim.Game.Filename = reader.URI().Name()
                lifeSim.Game.SetHistorySize(historySize)
                lifeSim.ResizeToFit()
                lifeSim.Draw()
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
            write_err := lifeSim.Game.WriteRLE(writer)
            if write_err != nil {
                dialog.ShowError(write_err, myWindow)
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

    fileOpenMenuItem := fyne.NewMenuItem("Open", func () {
        fileOpen := dialog.NewFileOpen(fileOpenCallback, myWindow)
        fileOpen.SetFilter(lifeFileExtensionsFilter)
        fileOpen.SetLocation(cwdURI)
        fileOpen.Show()
    })
    fileOpenMenuItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyO, Modifier: modKey}

    fileSaveMenuItem := fyne.NewMenuItem("Save", func() {
        fileSave := dialog.NewFileSave(fileSaveCallback, myWindow)
        fileSave.SetFilter(saveLifeExtensionsFilter)
        fileSave.SetLocation(cwdURI)
        fileSave.Show()
    })
    fileSaveMenuItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: modKey}

    fileMenu := fyne.NewMenu("File", fileOpenMenuItem, fileSaveMenuItem)
    mainMenu := fyne.NewMainMenu(fileMenu)

    myWindow.SetMainMenu(mainMenu)

    controlBar := NewControlBar(lifeSim)
    statusBar := NewStatusBar(lifeSim)
    content := container.NewBorder(controlBar, statusBar, nil, nil, lifeSim)
    myWindow.SetContent(content)
    myWindow.Resize(fyne.NewSize(800, 500))
    toggleRun := func(shortcut fyne.Shortcut) {
        if controlBar.IsRunning() {
            controlBar.StopSim()
        } else {
            controlBar.StartSim()
        }
    }
    myWindow.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyR, Modifier: modKey}, toggleRun)
    keyPressHandler := func(keyEvent *fyne.KeyEvent) {
        switch keyEvent.Name {
        case fyne.KeyUp:
            lifeSim.ShiftUp()
        case fyne.KeyDown:
            lifeSim.ShiftDown()
        case fyne.KeyLeft:
            lifeSim.ShiftLeft()
        case fyne.KeyRight:
            lifeSim.ShiftRight()
        case fyne.KeyR:
            toggleRun(nil)
        default:
            fmt.Println("Got unexpected key", keyEvent.Name)
        }
    }
    myWindow.Canvas().SetOnTypedKey(keyPressHandler)

    myWindow.ShowAndRun()
}

