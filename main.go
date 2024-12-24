package main

import (
    "fmt"
    "image/color"
    "os"
    "time"

    "github.com/pneumaticdeath/golife"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/canvas"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/layout"
    "fyne.io/fyne/v2/theme"
    "fyne.io/fyne/v2/widget"
)

type LifeSim struct {
    Game                                *golife.Game
    BoxDisplayMin, BoxDisplayMax        golife.Cell
    Scale                               float32 // pixel per cell
    Surface                             *fyne.Container
    CellColor                           color.Color
    Running                             bool
    StepTime                            float64
}

func (ls *LifeSim) Init() {
    ls.Game = golife.NewGame()
    ls.BoxDisplayMin = golife.Cell{0, 0}
    ls.BoxDisplayMax = golife.Cell{10, 10}
    ls.Surface = container.NewWithoutLayout()
    ls.CellColor = color.NRGBA{R: 0, G: 0, B: 180, A: 255}
}

func (ls *LifeSim) Draw() {
    // Need to implement
    windowSize := ls.Surface.Size()

    displayWidth := float32(ls.BoxDisplayMax.X - ls.BoxDisplayMin.X + 1)
    displayHeight := float32(ls.BoxDisplayMax.Y - ls.BoxDisplayMin.Y + 1)

    ls.Scale = min(windowSize.Width / displayWidth, windowSize.Height / displayHeight)

    cellSize := fyne.NewSize(ls.Scale, ls.Scale)

    displayCenter := fyne.NewPos(float32(ls.BoxDisplayMax.X + ls.BoxDisplayMin.X)/2.0, 
                                float32(ls.BoxDisplayMax.Y + ls.BoxDisplayMin.Y)/2.0)
    
    windowCenter := fyne.NewPos(windowSize.Width/2.0, windowSize.Height/2.0)

    // cells := make([]fyne.CanvasObject, 0, len(ls.Game.Population))

    ls.Surface.RemoveAll()
    for cell, _ := range ls.Game.Population {
        window_x := windowCenter.X + ls.Scale * (float32(cell.X) - displayCenter.X) - ls.Scale/2.0
        window_y := windowCenter.Y + ls.Scale * (float32(cell.Y) - displayCenter.Y) - ls.Scale/2.0
        cellPos := fyne.NewPos(window_x, window_y)

        if ls.Scale < 2.0 {
            fmt.Println("Can't display that far zoomed in yet")
        } else {
            cellCircle := canvas.NewCircle(ls.CellColor)
            cellCircle.Resize(cellSize)
            cellCircle.Move(cellPos)

            // cells = append(cells, cellCircle)
            ls.Surface.Add(cellCircle)

            fmt.Printf("Cell at %v\n", cellPos)
        }

    }

    ls.Surface.Refresh()

}

func (ls *LifeSim) SetDisplayBox(minCorner, maxCorner golife.Cell) {
    if minCorner.X > maxCorner.X {
        ls.BoxDisplayMin = golife.Cell{0, 0}
        ls.BoxDisplayMax = golife.Cell{10, 10}
    } else {
        ls.BoxDisplayMin, ls.BoxDisplayMax = minCorner, maxCorner
    }
}

func (ls *LifeSim) ResizeToFit() {
    boxDisplayMin, boxDisplayMax := ls.Game.Population.BoundingBox()
    if boxDisplayMin.X > boxDisplayMax.X {  // null field condition
        ls.BoxDisplayMin = golife.Cell{0, 0}
        ls.BoxDisplayMax = golife.Cell{10, 10}
    } else {
        ls.BoxDisplayMin = boxDisplayMin
        ls.BoxDisplayMax = boxDisplayMax
    }
}

func main() {
    myApp := app.New()
    myWindow := myApp.NewWindow("Conway's Game of Life")

    red := color.NRGBA{R: 180, G: 0, B: 0, A: 255}
    blue := color.NRGBA{R: 0, G: 0, B: 180, A: 255}
    // colors := []color.Color{color.Black, red, blue, color.White}
    // colorIndex := 0

    speedSlider := widget.NewSlider(100.0, 1000.0)
    speedSlider.SetValue(300.0)

    // rectangle := canvas.NewRectangle(colors[colorIndex])
    // mainContent := container.New(layout.NewGridLayout(1), rectangle)
    // mainContent := container.NewWithoutLayout()

    var lifeSim LifeSim

    lifeSim.Init()
    if len(os.Args) > 1 {
        lifeSim.Game = golife.Load(os.Args[1])
    } else {
        lifeSim.Game = golife.Load("glider.rle")
    }
    // lifeSim.Surface = mainContent
    fmt.Printf("Game type: %T\n", lifeSim.Game)
    fmt.Println("Cells:", lifeSim.Game.Population)
    lifeSim.ResizeToFit()
    // lifeSim.CellColor = blue
    lifeSim.Draw()

    running := false

    runGame := func() {
        for running {
            lifeSim.Draw()
            lifeSim.Game.Next()
            lifeSim.ResizeToFit()
            time.Sleep(time.Duration(speedSlider.Value)*time.Millisecond)
        }
    }

    // Stub so we can pass it as part of the button
    // action.  Will be replaced later.
    setRunStopText := func(label string, icon fyne.Resource) {}

    runStopButton := widget.NewButtonWithIcon("Run", theme.MediaPlayIcon(), func() {
        running = ! running
        if running {
            setRunStopText("Pause", theme.MediaPauseIcon())
            lifeSim.CellColor = red
            go runGame()
        } else {
            setRunStopText("Run", theme.MediaPlayIcon())
            lifeSim.CellColor = blue
            lifeSim.Draw()
        }})

    setRunStopText = func(label string, icon fyne.Resource) {
        runStopButton.SetIcon(icon)
        runStopButton.SetText(label)
    }

    topBar := container.New(layout.NewHBoxLayout(), runStopButton, layout.NewSpacer(),
                            canvas.NewText("faster", color.Black), speedSlider, canvas.NewText("slower", color.Black))
    content := container.NewBorder(topBar, nil, nil, nil, lifeSim.Surface)
    myWindow.Resize(fyne.NewSize(250, 250))
    myWindow.SetContent(content)
    // speedSlider.Resize(fyne.NewSize(30, 200)) // doesn't seem to have much effect
    myWindow.ShowAndRun()
}

