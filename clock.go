package main

import (
	"time"
)

// This is a simple mechanism to make the animations of the running
// simulation smooth and consistent.  It starts a background
// goroutine thread (doLifeTicks() which does nothing but wait on
// a boolean channel, and every time it reads a value (always true)
// it moves the simulation on to the next generation while timing
// the result.  Then every time the game wants to move to the next
// generation, it calls the "LifeTick()" method on the clock, and
// that allows the game to progress to the next generation.
// if the next generation hasn't finished calculating yet, then the
// "LifeTick()" method will block.

type LifeSimClock struct {
	lifeTicker           chan bool
	life                 *LifeSim
	Running              bool
	DisplayUpdateCadence time.Duration
}

func NewLifeSimClock(sim *LifeSim) *LifeSimClock {
	clk := &LifeSimClock{make(chan bool, 1), sim, true, time.Second / 60.0}
	go clk.doLifeTicks()
	go clk.doDisplayRedraws()
	return clk
}

func (clk *LifeSimClock) doLifeTicks() {
	for clk.Running {
		<-clk.lifeTicker // Will block waiting for a clock tick
		start := time.Now()
		clk.life.Game.Next()
		clk.life.LastStepTime = time.Since(start)
		clk.life.Dirty = true
	}
}

func (clk *LifeSimClock) LifeTick() {
	clk.lifeTicker <- true // Will block if the last tick hasn't been consumed yet
}

func (clk *LifeSimClock) doDisplayRedraws() {
	for clk.Running {
		time.Sleep(clk.DisplayUpdateCadence)
		clk.life.Draw() // the Draw routine checks/clears the Dirty flag
	}
}
