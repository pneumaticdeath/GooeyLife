package examples

import (
	"fmt"
	"github.com/pneumaticdeath/golife"
)

type Example struct {
	Title       string
	Author      string
	Source      string
	Explanation string
	Cells       golife.CellList
}

var Examples []Example = []Example{
	{"Glider", "", "", "Simple patern that glides across the grid", []golife.Cell{{0, 0}, {1, 0}, {2, 0}, {0, 1}, {1, 2}}},
	{"Blinker", "", "", "Simple oscilator that switches between two states", []golife.Cell{{0, 0}, {0, 1}, {0, 2}}},
	{"Gosper Glider Gun", "Bill Gosper, et. al.", "", "First known pattern that grows indefinitely",
		[]golife.Cell{{25, 1}, {23, 2}, {25, 2}, {13, 3}, {14, 3}, {21, 3}, {22, 3}, {35, 3}, {36, 3},
			{12, 4}, {16, 4}, {21, 4}, {22, 4}, {35, 4}, {36, 4}, {1, 5}, {2, 5}, {11, 5}, {17, 5},
			{21, 5}, {22, 5}, {1, 6}, {2, 6}, {11, 6}, {15, 6}, {17, 6}, {18, 6}, {23, 6}, {25, 6},
			{11, 7}, {17, 7}, {25, 7}, {12, 8}, {16, 8}, {13, 9}, {14, 9}}}}

func LoadExample(e Example) *golife.Game {
	g := golife.NewGame()

	g.Filename = e.Title // hack, since this isn't really the title of a file
	g.Comments = append(g.Comments, fmt.Sprintf("N %s", e.Title))
	if e.Author != "" {
		g.Comments = append(g.Comments, fmt.Sprintf("O %s", e.Author))
	}
	if e.Source != "" {
		g.Comments = append(g.Comments, fmt.Sprintf("C %s", e.Source))
	}
	g.AddCells(e.Cells)

	return g
}
