package main

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/alecthomas/kong"

	set3 "github.com/TomTonic/Set3"
	rtcompare "github.com/TomTonic/rtcompare"
	nms "github.com/TomTonic/set3benchmark/nativemapset"
)

func addBenchmarkSet3(cfg addBenchmarkConfig) (measurements []float64) {
	prng := rtcompare.NewDPRNG(cfg.seed)
	numberOfSets := cfg.numOfSets
	setSize := cfg.finalSize
	setAry := make([]*set3.Set3[uint64], numberOfSets)
	timePerRound := make([]float64, cfg.rounds)

	for r := range cfg.rounds {
		for i := range numberOfSets {
			setAry[i] = set3.EmptyWithCapacity[uint64](uint32(cfg.initSize))
		}
		debug.SetGCPercent(-1)
		runtime.GC()

		prng.State = cfg.seed
		prng.Round = 0
		startTime := rtcompare.SampleTime()
		for s := range numberOfSets {
			currentSet := setAry[s]
			for range setSize {
				currentSet.Add(prng.Uint64())
			}
		}
		endTime := rtcompare.SampleTime()
		diff := float64(rtcompare.DiffTimeStamps(startTime, endTime))
		timePerRound[r] = diff
		debug.SetGCPercent(100)
	}
	return timePerRound
}

func addBenchmarkNMS(cfg addBenchmarkConfig) (measurements []float64) {
	prng := rtcompare.NewDPRNG(cfg.seed)
	numberOfSets := cfg.numOfSets
	setSize := cfg.finalSize
	setAry := make([]*nms.NativeMapSet[uint64], numberOfSets)
	timePerRound := make([]float64, cfg.rounds)

	for r := range cfg.rounds {
		for i := range numberOfSets {
			setAry[i] = nms.EmptyNativeMapSetWithCapacity[uint64](uint32(cfg.initSize))
		}
		debug.SetGCPercent(-1)
		runtime.GC()

		prng.State = cfg.seed
		prng.Round = 0
		startTime := rtcompare.SampleTime()
		for s := range numberOfSets {
			currentSet := setAry[s]
			for range setSize {
				currentSet.Add(prng.Uint64())
			}
		}
		endTime := rtcompare.SampleTime()
		diff := float64(rtcompare.DiffTimeStamps(startTime, endTime))
		timePerRound[r] = diff
		debug.SetGCPercent(100)
	}
	return timePerRound
}

type addBenchmarkConfig struct {
	rounds    uint64
	numOfSets uint64
	initSize  uint64
	finalSize uint64
	seed      uint64
}

var cli struct {
	Add struct {
		Rounds    uint64 `help:"Number of rounds/samples for the benchmark." short:"r" default:"128"`
		NumOfSets uint64 `help:"Number of sets to fill per round." short:"n" default:"1000"`
		InitSize  uint64 `help:"Initially allocated set size." short:"i" default:"100"`
		FinalSize uint64 `help:"Total number of random values to add to each set." short:"f" default:"10000"`
		Seed      uint64 `help:"Random seed. Choose a value for deterministic random number generation or zero for random random seeds." short:"s" default:"0"`
	} `cmd:"" help:"Benchmark adding random uint64 to an empty set."`
}

func main() {

	ctx := kong.Parse(&cli,
		kong.Name("set3benchmark"),
		kong.Description("A benchmark program to compare set implementations."),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}))

	switch ctx.Command() {
	case "add":
		abc := addBenchmarkConfig{
			rounds:    cli.Add.Rounds,
			numOfSets: cli.Add.NumOfSets,
			initSize:  cli.Add.InitSize,
			finalSize: cli.Add.FinalSize,
			seed:      cli.Add.Seed,
		}
		fmt.Printf("Running add benchmark with config: %+v\n", abc)

		fmt.Println("Benchmarking Set3...")
		set3Measurements := addBenchmarkSet3(abc)
		fmt.Println("Benchmarking NativeMapSet...")
		nmsMeasurements := addBenchmarkNMS(abc)

		sumSet3 := 0.0
		for _, v := range set3Measurements {
			sumSet3 += v
		}
		avgSet3 := sumSet3 / float64(len(set3Measurements))
		precisionVal := float64(rtcompare.GetSampleTimePrecision())
		ratio := avgSet3 / precisionVal
		fmt.Printf("Average measurement per round (Set3): %.2fns. Sample time precision: %.2fns. Average = %.2fx precision\n", avgSet3, precisionVal, ratio)
		if ratio < 10.0 {
			fmt.Println("Warning: average measurements are close to clock precision — results may be noisy.")
		}

		sumNMS := 0.0
		for _, v := range nmsMeasurements {
			sumNMS += v
		}
		avgNMS := sumNMS / float64(len(nmsMeasurements))
		ratio = avgNMS / precisionVal
		fmt.Printf("Average measurement per round (NativeMapSet): %.2fns. Sample time precision: %.2fns. Average = %.2fx precision\n", avgNMS, precisionVal, ratio)
		if ratio < 10.0 {
			fmt.Println("Warning: average measurements are close to clock precision — results may be noisy.")
		}

		relativeSpeedups := []float64{-0.5, -0.4, -0.3, -0.2, -0.1, -0.05, 0.0, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5} // relative speedups to test

		results, err := rtcompare.CompareSamplesDefault(set3Measurements, nmsMeasurements, relativeSpeedups)
		if err != nil {
			panic(err)
		}

		// Report results
		fmt.Println("⏱️ Runtime comparison: Set3 vs. NativeMapSet")
		for _, r := range results {
			fmt.Printf("Speedup ≥ %.2f%% → Confidence: %.3f%%\n", r.RelativeSpeedupSampleAvsSampleB*100.0, r.Confidence*100.0)
		}

	default:
		fmt.Print("Check ctx.Command(): ")
		fmt.Println(ctx.Command())
		panic("done.")
	}

}
