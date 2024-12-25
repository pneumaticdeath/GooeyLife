package main

import (
    // "fmt"
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

    pixels := make(map[golife.Cell]int32)
    maxDens := 1

    ls.Surface.RemoveAll()
    for cell, _ := range ls.Game.Population {
        window_x := windowCenter.X + ls.Scale * (float32(cell.X) - displayCenter.X) - ls.Scale/2.0
        window_y := windowCenter.Y + ls.Scale * (float32(cell.Y) - displayCenter.Y) - ls.Scale/2.0
        cellPos := fyne.NewPos(window_x, window_y)

        if ls.Scale < 2.0 {
            if window_x >= 0 && window_y >= 0 && window_x < displayWidth && window_y < displayHeight {
                pixelPos := golife.Cell{golife.Coord(window_x), golife.Coord(window_y)}
                pixels[pixelPos] += 1
                if int(pixels[pixelPos]) > maxDens {
                    maxDens = int(pixels[pixelPos])
                }
            }
        } else {
            cellCircle := canvas.NewCircle(ls.CellColor)
            cellCircle.Resize(cellSize)
            cellCircle.Move(cellPos)

            ls.Surface.Add(cellCircle)
        }
    }

    if ls.Scale < 2.0 && len(pixels) > 0 {
        for pixelPos, count := range pixels {
            density := max(float32(count)/float32(maxDens), float32(0.5))
            r, g, b, a := ls.CellColor.RGBA()
            pixelColor := color.NRGBA{R: uint8(float32(r)*density),
                                      G: uint8(float32(g)*density),
                                      B: uint8(float32(b)*density),
                                      A: uint8(a)}
            pixel := canvas.NewRectangle(pixelColor)
            pixel.Resize(fyne.NewSize(2, 2))
            pixel.Move(fyne.NewPos(float32(pixelPos.X), float32(pixelPos.Y)))
            ls.Surface.Add(pixel)
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

func (ls *LifeSim) AutoZoom() {
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

func main() {
    myApp := app.New()
    myWindow := myApp.NewWindow("Conway's Game of Life")

    // red := color.NRGBA{R: 180, G: 0, B: 0, A: 255}
    blue := color.NRGBA{R: 0, G: 0, B: 180, A: 255}
    green := color.NRGBA{R: 0, G: 180, B: 0, A: 255}
    // colors := []color.Color{color.Black, red, blue, color.White}
    // colorIndex := 0

    speedSlider := widget.NewSlider(1.5, 1000.0)
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
    lifeSim.ResizeToFit()
    // lifeSim.CellColor = blue
    lifeSim.Draw()

    running := false

    runGame := func() {
        for running {
            lifeSim.Draw()
            lifeSim.Game.Next()
            // lifeSim.ResizeToFit()
            lifeSim.AutoZoom()
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
            lifeSim.CellColor = green
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
    myWindow.Resize(fyne.NewSize(500, 500))
    myWindow.SetContent(content)
    lifeSim.Draw()
    // speedSlider.Resize(fyne.NewSize(30, 200)) // doesn't seem to have much effect

    myWindow.ShowAndRun()
}

