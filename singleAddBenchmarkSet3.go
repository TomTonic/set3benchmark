package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"time"

	set3 "github.com/TomTonic/Set3"
	misc "github.com/TomTonic/set3benchmark/misc"
)

type SingleAddBenchmarkConfig struct {
	Number      uint64  `help:"Number of values to add to an empty set." default:"20" short:"n"`
	Capacity    uint32  `help:"Initial capacity allocation value to create an empty set." default:"30" short:"c"`
	Experiments uint64  `help:"Number of experiments or rounds of measurments." default:"2000" short:"e"`
	Iterations  uint64  `help:"Number of iterations per experiment. The default of 25 million menas roughly 200ms per measurement." default:"25000000" short:"i"`
	HistLower   float64 `help:"Histogram lower bound value in nanoseconds (inclusive)." default:"7.0" short:"l"`
	HistUpper   float64 `help:"Histogram upper bound value in nanoseconds (inclusive)." default:"9.5" short:"u"`
	HistSteps   uint32  `help:"Number of steps in histogram between upper and lower bound." default:"50" short:"s"`
	HistWidth   uint32  `help:"With of the historam bars on the console." default:"40" short:"w"`
	Xlsx        bool    `help:"Generate XLSX file containing the results in current direcory. (Default is 'true'.)" default:"true" short:"x"`
	RandRt      float64 `help:"Runtime of random number generation in nanoseconds, i.e. one call to prng.Uint64(). If this parameter is omited it will automatically be determined, loweing the startup time." short:"r"`
	Precision   float64 `help:"Maximum precision of system timer (quantization error) in nanoseconds. If this parameter is omited it will automatically be determined, loweing the startup time." short:"p"`
	PrngSeed    uint64  `help:"Seed value for the pseudo random number generator." default:"3571113171923"`
	//RuntimePerAdd float64 `help:"Expected runtime per single Add(prng.Uint64()) instruction in nanoseconds. Used to calculate the necessary number of iterations to meet the target runtime per experiment." short:"a" default:"8.0"`
	//Target        time.Duration `help:"Target runtime per experiment. (Refer to https://pkg.go.dev/time#ParseDuration for syntax.)" short:"t" default:"10ms"`
}

func doSingleAddBenchmarkSet3(cfg SingleAddBenchmarkConfig) *misc.Histo {
	prng := misc.PRNG{State: cfg.PrngSeed}
	setSize := cfg.Number
	iter := cfg.Iterations
	set := set3.EmptyWithCapacity[uint64](cfg.Capacity)
	histo := misc.MakeHisto(cfg.HistLower, cfg.HistUpper, int(cfg.HistSteps))
	avgClear, _ := measureAvgClear(cfg.Iterations, cfg.Precision, set)
	runtime.GC()
	debug.SetGCPercent(-1)
	for range cfg.Experiments {
		//prng.State = cfg.PrngSeed
		//prng.Round = 0
		startTime := misc.SampleTime()
		for range iter {
			set.Clear()
			for range setSize {
				set.Add(prng.Uint64())
			}
		}
		endTime := misc.SampleTime()
		diff := misc.DiffTimeStamps(startTime, endTime)
		timeForClearAndAddsAndRng := (float64(diff) - cfg.Precision) / float64(iter)
		timeForAddsAndRng := timeForClearAndAddsAndRng - avgClear
		timeForOneAddAndRng := timeForAddsAndRng / float64(setSize)
		timeForOneAdd := timeForOneAddAndRng - cfg.RandRt
		histo.Add(timeForOneAdd)
	}
	debug.SetGCPercent(100)
	return histo
}

func measureAvgClear(iterations uint64, precision float64, set *set3.Set3[uint64]) (avgClear, quantizationError float64) {
	clearRounds := iterations * 10
	runtime.GC()
	debug.SetGCPercent(-1)
	startTime := misc.SampleTime()
	for range clearRounds {
		set.Clear()
	}
	endTime := misc.SampleTime()
	debug.SetGCPercent(100)
	diff := misc.DiffTimeStamps(startTime, endTime)
	avgClear = (float64(diff) - precision) / float64(clearRounds)
	quantizationError = misc.GetSampleTimePrecision() / (avgClear * float64(clearRounds))
	fmt.Printf("avgClear: %.3fns (measuring runtime: %v, iterations: %d, quantization error: %e)\n", avgClear, time.Duration(int(avgClear*float64(clearRounds))), clearRounds, quantizationError)
	return
}
