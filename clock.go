package main

import (
	"time"
)

// This is a simple mechanism to make the animations of the running
// simulation smooth and consistent.  It starts a background
// goroutine thread (doTicks() which does nothing but sleep wait on
// a boolean channel, and every time I reads a value (always true)
// it moves the simulation on to the next generation while timing
// the result.  Then every time the game wants to move to the next
// generation, it calls the "Tick()" method on the clock, and
// that allows the game to progress to the next generation.
// if the next generation hasn't finished calculating yet, then the
// "Tick()" method will block.

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
