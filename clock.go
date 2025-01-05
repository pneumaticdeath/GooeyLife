package main

import (
	"time"
)

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
