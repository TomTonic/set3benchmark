package main

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/alecthomas/kong"

	set3 "github.com/TomTonic/Set3"
	rtcompare "github.com/TomTonic/rtcompare"
	misc "github.com/TomTonic/set3benchmark/misc"
)

var prngOverhead = -1.0
var prngOverheadActualQuantizationError = -1.0

func getPRNGOverhead() (prngOverheadInNS, quantizationError float64) {
	if prngOverhead != -1.0 {
		return prngOverhead, prngOverheadActualQuantizationError
	}
	//	roughlyExpectedRuntimeForOneCallInNS := 1.2
	desiredErrorMargin := 1.0 / (1 << 16)
	timerPrecisionInNS := rtcompare.GetSampleTimePrecision()
	//	calibrationCalls := int64(timerPrecisionInNS / (roughlyExpectedRuntimeForOneCallInNS * desiredErrorMargin))
	calibrationCalls := int64(float64(timerPrecisionInNS) / desiredErrorMargin)
	prng := rtcompare.NewDPRNG(0x1234567890abcde)
	rounds := 1001
	times := make([]float64, rounds)
	debug.SetGCPercent(-1)
	for r := range rounds {
		start := rtcompare.SampleTime()
		for i := int64(0); i < calibrationCalls; i++ {
			prng.Uint64()
		}
		stop := rtcompare.SampleTime()
		times[r] = float64(rtcompare.DiffTimeStamps(start, stop))
	}
	debug.SetGCPercent(100)
	medTimeForOneRound := rtcompare.QuickMedian(times)
	prngOverhead = medTimeForOneRound / float64(calibrationCalls)
	//	prngOverheadActualQuantizationError = timerPrecisionInNS / (float64(calibrationCalls) * medTimeForOneRound)
	prngOverheadActualQuantizationError = float64(timerPrecisionInNS) / float64(calibrationCalls)
	return prngOverhead, prngOverheadActualQuantizationError
}

func addBenchmark(cfg singleAddBenchmarkConfig) (measurements []float64) {
	prng := rtcompare.NewDPRNG(cfg.seed)
	numberOfSets := cfg.numOfSets
	setSize := cfg.finalSetSize
	set := make([]*set3.Set3[uint64], numberOfSets)
	for i := range numberOfSets {
		set[i] = set3.EmptyWithCapacity[uint64](uint32(cfg.initSize))
	}
	timePerRound := make([]float64, cfg.rounds)
	debug.SetGCPercent(-1)
	runtime.GC()
	for r := range cfg.rounds {
		prng.State = cfg.seed
		prng.Round = 0
		for s := range numberOfSets {
			set[s].Clear()
		}
		startTime := rtcompare.SampleTime()
		for s := range numberOfSets {
			currentSet := set[s]
			for range setSize {
				currentSet.Add(prng.Uint64())
			}
		}
		endTime := rtcompare.SampleTime()
		diff := float64(rtcompare.DiffTimeStamps(startTime, endTime))
		// timePerRound[r] = diff - misc.GetSampleTimeRuntime() - (rngOverhead * float64(numberOfSets*setSize))
		timePerRound[r] = diff
	}
	debug.SetGCPercent(100)
	return timePerRound
}

func addBenchmark2(cfg singleAddBenchmarkConfig) (measurements []float64) {
	prng := rtcompare.NewDPRNG(cfg.seed)
	numberOfSets := cfg.numOfSets
	setSize := cfg.finalSetSize
	set := set3.EmptyWithCapacity[uint64](uint32(cfg.initSize))
	avgClear := 0.0
	timePerRoundClearAndAdd := make([]float64, cfg.rounds)
	debug.SetGCPercent(-1)
	runtime.GC()
	clearRounds := cfg.rounds * numberOfSets * 10
	{
		startTime := rtcompare.SampleTime()
		for range clearRounds {
			set.Clear()
		}
		endTime := rtcompare.SampleTime()
		diff := float64(rtcompare.DiffTimeStamps(startTime, endTime))
		avgClear = diff / float64(clearRounds)
	}
	debug.SetGCPercent(100)
	quantizationError := float64(rtcompare.GetSampleTimePrecision()) / (avgClear * float64(clearRounds))
	fmt.Printf("avgClear: %.3fns (measuring runtime: %v, iterations: %d, quantization error: %e)\n", avgClear, time.Duration(int(avgClear*float64(clearRounds))), clearRounds, quantizationError)
	debug.SetGCPercent(-1)
	runtime.GC()
	for r := range cfg.rounds {
		//prng.State = cfg.seed
		//prng.Round = 0
		startTime := rtcompare.SampleTime()
		for range numberOfSets {
			set.Clear()
			for range setSize {
				set.Add(prng.Uint64())
			}
		}
		endTime := rtcompare.SampleTime()
		diff := float64(rtcompare.DiffTimeStamps(startTime, endTime))
		timePerRoundClearAndAdd[r] = diff - avgClear
	}
	debug.SetGCPercent(100)
	return timePerRoundClearAndAdd
}

func toNanoSecondsPerAdd(timePerRound []float64, addsPerRound uint64) []float64 {
	nsPerAdd := make([]float64, len(timePerRound))
	for i, tpr := range timePerRound {
		nsPerAdd[i] = float64(tpr) / float64(addsPerRound)
	}
	return nsPerAdd
}

func printTotalRuntime(start time.Time) string {
	end := time.Now()
	return fmt.Sprintf("\nTotal runtime of benchmark: %v\n", end.Sub(start))
}

func getNumberOfConfigs(setSizeFrom, setSizeTo uint64, Pstep *float64, Istep *uint64, RelativeLimit *float64, AbsoluteLimit *uint64) uint64 {
	result := uint64(0)
	for setSize := range setSizes(setSizeFrom, setSizeTo) {
		for range initSizes2(setSize, Pstep, Istep, RelativeLimit, AbsoluteLimit) {
			result++
		}
	}
	return result
}

type programParametrization struct {
	fromSetSize, toSetSize, targetAddsPerRound uint64
	expRuntimePerAdd, secondsPerConfig         float64
	Pstep                                      *float64
	Istep                                      *uint64
	RelativeLimit                              *float64
	AbsoluteLimit                              *uint64
	//step                                       Step
}

type benchmarkSetup struct {
	programParametrization
	totalAddsPerConfig uint64
}

func benchmarkSetupFrom(p programParametrization) (benchmarkSetup, error) {
	result := benchmarkSetup{
		programParametrization: p,
		totalAddsPerConfig:     uint64(p.secondsPerConfig * 1_000_000_000.0 / p.expRuntimePerAdd),
	}
	if p.Pstep != nil && p.Istep != nil {
		return result, errors.New("parameter error: Pstep and Istep are both defined")
	}
	if p.Pstep == nil && p.Istep == nil {
		one := uint64(1)
		result.Istep = &one
	}
	if p.RelativeLimit != nil && p.AbsoluteLimit != nil {
		return result, errors.New("parameter error: RelativeLimit and AbsoluteLimit are both defined")
	}
	if p.RelativeLimit == nil && p.AbsoluteLimit == nil {
		onhundret := 100.0
		result.RelativeLimit = &onhundret
	}
	if p.toSetSize < p.fromSetSize {
		return result, errors.New("parameter error: 'to' < 'from'")
	}
	if p.toSetSize > 1<<28 {
		return result, errors.New("parameter error: value of 'to' too big")
	}
	if p.secondsPerConfig <= 0 {
		return result, errors.New("parameter error: value of 'spc' too low")
	}
	if p.targetAddsPerRound > result.totalAddsPerConfig {
		return result, errors.New("parameter error: AddsPerRound 'apr' too big for SecondsPerConfig 'spc'")
	}
	if p.targetAddsPerRound < result.toSetSize {
		return result, errors.New("parameter error: AddsPerRound 'apr' too small for 'to'")
	}
	return result, nil
}

func setSizes(setSizeFrom, setSizeTo uint64) func(yield func(uint64) bool) {
	return func(yield func(uint64) bool) {
		for setSize := setSizeFrom; setSize <= setSizeTo; setSize++ {
			if !yield(setSize) {
				return
			}
		}
	}
}

func stepsHeadings(setSizeFrom, setSizeTo uint64, Pstep *float64, Istep *uint64, RelativeLimit *float64, AbsoluteLimit *uint64) ([]string, error) {
	result := make([]string, 0, 100)
	if Pstep == nil && Istep == nil {
		return nil, errors.New("parameter error: Pstep == nil && Istep == nil")
	}
	if Pstep != nil && Istep != nil {
		return nil, errors.New("parameter error: Pstep != nil && Istep != nil")
	}
	if RelativeLimit == nil && AbsoluteLimit == nil {
		return nil, errors.New("parameter error: RelativeLimit == nil && Istep == nil")
	}
	if RelativeLimit != nil && AbsoluteLimit != nil {
		return nil, errors.New("parameter error: RelativeLimit != nil && AbsoluteLimit != nil")
	}
	if Pstep != nil && RelativeLimit != nil {
		/*
			Example:
			setSize	Pstep	RelativeLimit	Expected
			5		10%		100%			0%	10%	20%	30%	40%	50%	60%	70%	80%	90%	100%
			6		10%		100%			0%	10%	20%	30%	40%	50%	60%	70%	80%	90%	100%
			7		10%		100%			0%	10%	20%	30%	40%	50%	60%	70%	80%	90%	100%

			=> setSizeFrom/setSizeTo are irrelevant
		*/
		start := float64(0)
		limit := (*RelativeLimit) + (*Pstep)
		for f := start; f < limit; f += (*Pstep) {
			result = append(result, fmt.Sprintf("+%.1f%% ", f))
		}
	}
	if Pstep != nil && AbsoluteLimit != nil {
		/*
			Example:
			setSize	Pstep	AbsoluteLimit	Expected
			50		10%		100				0%	10%	20%	30%	40%	50%	60%	70%	80%	90%	100%
			51		10%		100				0%	10%	20%	30%	40%	50%	60%	70%	80%	90%	100%
			52		10%		100				0%	10%	20%	30%	40%	50%	60%	70%	80%	90%	100%
			53		10%		100				0%	10%	20%	30%	40%	50%	60%	70%	80%	90%
			54		10%		100				0%	10%	20%	30%	40%	50%	60%	70%	80%	90%
			55		10%		100				0%	10%	20%	30%	40%	50%	60%	70%	80%	90%
			56		10%		100				0%	10%	20%	30%	40%	50%	60%	70%	80%
			57		10%		100				0%	10%	20%	30%	40%	50%	60%	70%	80%
			58		10%		100				0%	10%	20%	30%	40%	50%	60%	70%	80%
			59		10%		100				0%	10%	20%	30%	40%	50%	60%	70%
			60		10%		100				0%	10%	20%	30%	40%	50%	60%	70%
			61		10%		100				0%	10%	20%	30%	40%	50%	60%	70%
			62		10%		100				0%	10%	20%	30%	40%	50%	60%	70%
			63		10%		100				0%	10%	20%	30%	40%	50%	60%

			=> use setSizeFrom for longest Sequence
		*/
		start := float64(0)
		factor := float64(*AbsoluteLimit) / float64(setSizeFrom)
		limit := (factor-1.0)*100.0 + *Pstep
		for f := start; f < limit; f += (*Pstep) {
			result = append(result, fmt.Sprintf("+%.1f%% ", f))
		}
	}
	if Istep != nil && RelativeLimit != nil {
		/*
			Example A:
			setSize	Istep	RelativeLimit	Expected
			5		1		100%			+0	+1	+2	+3	+4	+5
			6		1		100%			+0	+1	+2	+3	+4	+5	+6
			7		1		100%			+0	+1	+2	+3	+4	+5	+6	+7

			Example B:
			setSize	Istep	RelativeLimit	Expected
			5		2		100%			+0	+2	+4	+6
			6		2		100%			+0	+2	+4	+6
			7		2		100%			+0	+2	+4	+6	+8
			8		2		100%			+0	+2	+4	+6	+8
			9		2		100%			+0	+2	+4	+6	+8	+10

			=> use setSizeTo for longest sequence
		*/
		start := uint64(0)
		limit := uint64((math.Round(*RelativeLimit * float64(setSizeTo) / 100.0))) + (*Istep)
		for i := start; i < limit; i += (*Istep) {
			result = append(result, fmt.Sprintf("+%d ", i))
		}
	}
	if Istep != nil && AbsoluteLimit != nil {
		/*
			Example:
			setSize	Istep	AbsoluteLimit	Expected
			5		2		20				+0	+2	+4	+6	+8	+10	+12	+14	+16
			6		2		20				+0	+2	+4	+6	+8	+10	+12	+14
			7		2		20				+0	+2	+4	+6	+8	+10	+12	+14
			8		2		20				+0	+2	+4	+6	+8	+10	+12

			=> use setSizeFrom for longest Sequence
		*/
		start := uint64(0)
		limit := (*AbsoluteLimit) - setSizeFrom + (*Istep)
		for i := start; i < limit; i += (*Istep) {
			result = append(result, fmt.Sprintf("+%d ", i))
		}
	}
	return result, nil
}

func initSizes2(setSize uint64, Pstep *float64, Istep *uint64, RelativeLimit *float64, AbsoluteLimit *uint64) func(yield func(uint64) bool) {
	if Pstep != nil && RelativeLimit != nil {
		start := float64(0)
		limit := (*RelativeLimit) + (*Pstep)
		return func(yield func(uint64) bool) {
			for f := start; f < limit; f += (*Pstep) {
				retval := setSize + uint64(math.Round(float64(setSize)*f/100.0))
				if !yield(retval) {
					return
				}
			}
		}
	}
	if Pstep != nil && AbsoluteLimit != nil {
		start := float64(0)
		factor := float64(*AbsoluteLimit) / float64(setSize)
		limit := (factor-1.0)*100.0 + *Pstep
		return func(yield func(uint64) bool) {
			for f := start; f < limit; f += (*Pstep) {
				retval := setSize + uint64(math.Round(float64(setSize)*f/100.0))
				if !yield(retval) {
					return
				}
			}
		}
	}
	if Istep != nil && RelativeLimit != nil {
		start := uint64(0)
		limit := uint64((math.Round(*RelativeLimit * float64(setSize) / 100.0))) + (*Istep)
		return func(yield func(uint64) bool) {
			for i := start; i < limit; i += (*Istep) {
				retval := setSize + i
				if !yield(retval) {
					return
				}
			}
		}
	}
	if Istep != nil && AbsoluteLimit != nil {
		start := uint64(0)
		limit := (*AbsoluteLimit) - setSize + (*Istep)
		return func(yield func(uint64) bool) {
			for i := start; i < limit; i += (*Istep) {
				retval := setSize + i
				if !yield(retval) {
					return
				}
			}
		}

	}
	panic("4tapgpo438tnaowghp")
}

type singleAddBenchmarkConfig struct {
	initSize           uint64
	finalSetSize       uint64
	targetAddsPerRound uint64
	totalAddsPerConfig uint64
	numOfSets          uint64
	actualAddsPerRound uint64
	rounds             uint64
	seed               uint64
}

func makeSingleAddBenchmarkConfig(initSize, setSize, targetAddsPerRound, totalAddsPerConfig uint64, seed uint64) singleAddBenchmarkConfig {
	if setSize > targetAddsPerRound {
		targetAddsPerRound = setSize
	}
	if targetAddsPerRound > totalAddsPerConfig {
		totalAddsPerConfig = targetAddsPerRound
	}
	result := singleAddBenchmarkConfig{
		initSize:           initSize,
		finalSetSize:       setSize,
		targetAddsPerRound: targetAddsPerRound,
		totalAddsPerConfig: totalAddsPerConfig,
		seed:               seed,
	}
	result.numOfSets = uint64(math.Round(float64(targetAddsPerRound) / float64(setSize)))
	result.actualAddsPerRound = result.numOfSets * setSize // actualAddsPerRound ~ targetAddsPerRound
	result.rounds = uint64(math.Round(float64(totalAddsPerConfig) / float64(result.actualAddsPerRound)))
	return result
}

/*
type LF2 struct {
	Ssf  uint64   `help:"Set Size From. First set size to benchmark (inclusive)." default:"20"`
	Sst  uint64   `help:"Set Size To. Last set size to benchmark (inclusive)." default:"30"`
	Sssi *uint64  `help:"Set Size Step Integer. Calculate next set size to benchmark by adding an integer value. Default behaviour is to increment the set size by 1." xor:"Sssp, Sssi"`
	Sssp *float64 `help:"Set Size Step Percentage. Calculate next set size to benchmark by adding a percentage value. Default behaviour is to increment the set size by 1." xor:"Sssp, Sssi"`
	Isf  uint64   `help:"Init Size From. First set size to benchmark (inclusive)." default:"20"`
	Ist  uint64   `help:"Init Size To. Last set size to benchmark (inclusive)." default:"30"`
	Issi *uint64  `help:"Init Size Step Integer. Calculate next set size to benchmark by adding an integer value. Default behaviour is to increment the set size by 1." xor:"Sssp, Sssi"`
	Issp *float64 `help:"Init Size Step Percentage. Calculate next set size to benchmark by adding a percentage value. Default behaviour is to increment the set size by 1." xor:"Sssp, Sssi"`
}
*/

var cli struct {
	Add struct {
		Loadfactor struct {
			From             uint64        `arg:"" help:"First set size to benchmark (inclusive)." short:"f"`
			To               uint64        `arg:"" help:"Last set size to benchmark (inclusive)." short:"t"`
			AddsPerRound     uint64        `help:"Number of Add(prng.Uint64()) instructions between two time measurements. Balance the value between memory consumption (cache size/speed) and timer precision of your runtime environment (e.g., Windows=100ns)." short:"r" default:"50000"`
			RuntimePerConfig time.Duration `help:"Benchmarking time per configuration (combination of initial set size and number of values to add)." short:"c" default:"1.5s"`
			RuntimePerAdd    time.Duration `help:"Expected runtime per single Add(prng.Uint64()) instruction. Used to calculate the necessary number of iterations to meet the runtime-per-config and to predcict the total runtime of the benchmark." short:"a" default:"8ns"`
			Pstep            *float64      `help:"Uses percentage value to increase the initial set size in benchmark configurations until the limit is reached (see relative-limit or absolute-limit). You can either specify a pstep or an istep. Default is an istep of size 1." short:"p" xor:"Pstep, Istep"`
			Istep            *uint64       `help:"Uses integer value to increase the initial set size in benchmark configurations until the limit is reached (see relative-limit or absolute-limit). You can either specify a pstep or an istep. Default is an istep of size 1." short:"i" xor:"Pstep, Istep"`
			RelativeLimit    *float64      `help:"Increase initial (pre-allocated) set sizes until at least x% headroom are reached. You can either specify a relative-limit or an absolute-limit. Default is a relative-limit of 100%." xor:"RelativeLimit, AbsoluteLimit"`
			AbsoluteLimit    *uint64       `help:"Increase initial (pre-allocated) set sizes until at least an initial set size of x is reached. You can either specify a relative-limit or an absolute-limit. Default is a relative-limit of 100%." xor:"RelativeLimit, AbsoluteLimit"`
		} `cmd:"" help:"Perform a loadfactor test using different initial (pre-allocated) set sizes. This benchmark creates empty sets of a defined size x and then adds y random numers via Add(prng.Uint64()). Any combination of x and y is called 'configration'."`
		// Loadfactor2 LF2 `cmd:"" help:"Perform a loadfactor test using different initial (pre-allocated) set sizes. This benchmark creates empty sets of a defined size x and then adds y random numers via Add(prng.Uint64()). Any combination of x and y is called 'configration'."`
		SingleConf SingleAddBenchmarkConfig `cmd:"" help:"Perform a benchmark adding random uint64 to an empty set. This benchmark creates an empty set of a defined initial capacity and then adds a defined random numers via Add(prng.Uint64()). This procedure is repeated many times during an experiment to compensate for the quantisation error introduced by the systems limited timer precision."`
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
	case "add loadfactor <from> <to>":
		// fmt.Println(cli.Add.Loadfactor.From, cli.Add.Loadfactor.To, cli.Add.Loadfactor.AddsPerRound, cli.Add.Loadfactor.RuntimePerConfig, cli.Add.Loadfactor.RuntimePerAdd)
		// fmt.Println(cli.Add.Loadfactor.Pstep, cli.Add.Loadfactor.Istep, cli.Add.Loadfactor.RelativeLimit, cli.Add.Loadfactor.AbsoluteLimit)
		doLoadfactor()
	case "add single-conf":
		doBenchmark3()
	default:
		fmt.Print("Check ctx.Command(): ")
		fmt.Println(ctx.Command())
		panic("done.")
	}

}

func doBenchmark3() {
	hist := doSingleAddBenchmarkSet3(cli.Add.SingleConf)
	hist.Width = 120
	fmt.Printf("%v\n", hist.String())
}

func doLoadfactor() {
	var pp programParametrization
	pp.fromSetSize = cli.Add.Loadfactor.From
	pp.toSetSize = cli.Add.Loadfactor.To
	pp.targetAddsPerRound = cli.Add.Loadfactor.AddsPerRound
	pp.secondsPerConfig = float64(cli.Add.Loadfactor.RuntimePerConfig) / 1_000_000_000.0
	pp.expRuntimePerAdd = float64(cli.Add.Loadfactor.RuntimePerAdd)
	pp.Pstep = cli.Add.Loadfactor.Pstep
	pp.Istep = cli.Add.Loadfactor.Istep
	pp.RelativeLimit = cli.Add.Loadfactor.RelativeLimit
	pp.AbsoluteLimit = cli.Add.Loadfactor.AbsoluteLimit

	setup, err := benchmarkSetupFrom(pp)

	if err != nil {
		panic(err)
	}

	printSetup(setup)

	eo := NewExcelOutput("result.xlsx")
	eo.WriteConfigSheet(setup)

	start := time.Now()

	fmt.Printf("setSize ")
	headings, _ := stepsHeadings(setup.fromSetSize, setup.toSetSize, setup.Pstep, setup.Istep, setup.RelativeLimit, setup.AbsoluteLimit)
	for _, columnH := range headings {
		fmt.Print(columnH)
	}
	fmt.Print("\n")
	ho := misc.HistogramOptions{
		BinCount:     51,
		NiceRange:    false,
		ClampMinimum: 7.0,
		ClampMaximum: 9.55,
		// ClampPercentile: 0.99,
	}
	eo.WriteLine("Results", 1, "initSize", "setSize", "Average", "Minimum", "P25", "P50", "P75", "P90", "P99", "P999", "P9999", "Maximum",
		"7.00ns",
		"7.05ns",
		"7.10ns",
		"7.15ns",
		"7.20ns",
		"7.25ns",
		"7.30ns",
		"7.35ns",
	)

	for setSize := range setSizes(setup.fromSetSize, setup.toSetSize) {
		//fmt.Printf("%d ", setSize)
		for initSize := range initSizes2(setSize, setup.Pstep, setup.Istep, setup.RelativeLimit, setup.AbsoluteLimit) {
			fmt.Printf("%d/%d:\n", setSize, initSize)
			cfg := makeSingleAddBenchmarkConfig(initSize, setSize, setup.targetAddsPerRound, setup.totalAddsPerConfig, 0xABCDEF0123456789)
			timePerRound := addBenchmark2(cfg)
			misc.AssertPositive(timePerRound, "A")
			nsPerAdd := toNanoSecondsPerAdd(timePerRound, cfg.actualAddsPerRound)
			misc.AssertPositive(nsPerAdd, "B")
			h := misc.NewHistogram(nsPerAdd, &ho)
			h.Width = 120
			fmt.Printf("%v\n", h.String())
			eo.WriteLine("Results", 1, initSize, setSize, h.Average, h.Minimum, h.P25, h.P50, h.P75, h.P90, h.P99, h.P999, h.P9999, h.Maximum,
				h.Bins[0].Count,
				h.Bins[1].Count,
				h.Bins[2].Count,
				h.Bins[3].Count,
				h.Bins[4].Count,
				h.Bins[5].Count,
				h.Bins[6].Count,
				h.Bins[7].Count,
				h.Bins[8].Count,
				h.Bins[9].Count,
				h.Bins[10].Count,
				h.Bins[11].Count,
				h.Bins[12].Count,
				h.Bins[13].Count,
				h.Bins[14].Count,
				h.Bins[15].Count,
				h.Bins[16].Count,
				h.Bins[17].Count,
				h.Bins[18].Count,
				h.Bins[19].Count,
				h.Bins[20].Count,
				h.Bins[21].Count,
				h.Bins[22].Count,
				h.Bins[23].Count,
				h.Bins[24].Count,
				h.Bins[25].Count,
				h.Bins[26].Count,
				h.Bins[27].Count,
				h.Bins[28].Count,
				h.Bins[29].Count,
				h.Bins[30].Count,
				h.Bins[31].Count,
				h.Bins[32].Count,
				h.Bins[33].Count,
				h.Bins[34].Count,
				h.Bins[35].Count,
				h.Bins[36].Count,
				h.Bins[37].Count,
				h.Bins[38].Count,
				h.Bins[39].Count,
				h.Bins[40].Count,
				h.Bins[41].Count,
				h.Bins[42].Count,
				h.Bins[43].Count,
				h.Bins[44].Count,
				h.Bins[45].Count,
				h.Bins[46].Count,
				h.Bins[47].Count,
				h.Bins[48].Count,
				h.Bins[49].Count,
				h.Bins[50].Count,
			)
			eo.Save()
			//median := misc.Median(nsValues)
			//fmt.Printf("%.3f ", median)
			// fmt.Printf("%d ", initSize)
		}
		//fmt.Printf("\n")
	}
	fmt.Print(printTotalRuntime(start))
	eo.Close()
}

func printSetup(p benchmarkSetup) {
	fmt.Printf("Architecture:\t\t\t%s\n", runtime.GOARCH)
	fmt.Printf("OS:\t\t\t\t%s\n", runtime.GOOS)
	fmt.Printf("Max timer precision:\t\t%.2fns\n", rtcompare.GetSampleTimePrecision())
	overhead, qerror := getPRNGOverhead()
	fmt.Printf("prng.Uint64() runtime:\t\t%.3fns (quantization error: %ens)\n", overhead, qerror*overhead)
	fmt.Printf("Exp. Add(prng.Uint64()) rt:\t%.2fns\n", p.expRuntimePerAdd)
	quantizationError := calcQuantizationError(p)
	fmt.Printf("Add()'s per round:\t\t%d (expect a quantization error of %e, i.e. %ens per Add)\n", p.targetAddsPerRound, quantizationError, quantizationError*p.expRuntimePerAdd)
	fmt.Printf("Add()'s per config:\t\t%d (should result in a benchmarking time of %.2fs per config)\n", p.totalAddsPerConfig, p.secondsPerConfig)
	//	fmt.Printf("Set3 sizes:\t\t\tfrom %d to %d, stepsize %v\n", p.fromSetSize, p.toSetSize, p.step.String())
	fmt.Printf("Set3 sizes:\t\t\tfrom %d to %d\n", p.fromSetSize, p.toSetSize)
	numberOfConfigs := getNumberOfConfigs(p.fromSetSize, p.toSetSize, p.Pstep, p.Istep, p.RelativeLimit, p.AbsoluteLimit)
	fmt.Printf("Number of configs:\t\t%d\n", numberOfConfigs)
	fmt.Printf("Rounds per config:\t\t%d\n", p.totalAddsPerConfig/p.targetAddsPerRound)
	totalduration := predictTotalDuration(p)
	fmt.Printf("Expected total runtime:\t\t%v (assumption: %.2fns per Add(prng.Uint64()) and 12%% overhead for housekeeping)\n\n", totalduration, p.expRuntimePerAdd)
}

func calcQuantizationError(p benchmarkSetup) float64 {
	//	quantizationError := misc.GetSampleTimePrecision() / (p.expRuntimePerAdd * float64(p.targetAddsPerRound))
	quantizationError := float64(rtcompare.GetSampleTimePrecision()) / float64(p.targetAddsPerRound)
	return quantizationError
}

func predictTotalDuration(p benchmarkSetup) time.Duration {
	numberOfStepsPerSetSize := getNumberOfConfigs(p.fromSetSize, p.toSetSize, p.Pstep, p.Istep, p.RelativeLimit, p.AbsoluteLimit)
	totalduration := time.Duration(uint64(p.expRuntimePerAdd * float64(p.totalAddsPerConfig)))
	totalduration *= time.Duration(numberOfStepsPerSetSize)
	totalduration = time.Duration(float64(totalduration) * 1.12)
	return totalduration
}
