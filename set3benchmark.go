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
	misc "github.com/TomTonic/set3benchmark/misc"
	//	"github.com/loov/hrtime"
)

var rngOverhead = getPRNGOverhead()

func getPRNGOverhead() float64 {
	calibrationCalls := 2_000_000_000 // prng.Uint64() is about 1-2ns, timer resolution is 100ns (windows)
	prng := misc.PRNG{State: 0x1234567890abcde}
	debug.SetGCPercent(-1)
	start := misc.SampleTime()
	for i := 0; i < calibrationCalls; i++ {
		prng.Uint64()
	}
	stop := misc.SampleTime()
	debug.SetGCPercent(100)
	totalTimeForRandomNumbers := float64(misc.DiffTimeStamps(start, stop))
	sampleTimeOverhead := misc.GetSampleTimeRuntime()
	result := (totalTimeForRandomNumbers - sampleTimeOverhead) / float64(calibrationCalls)
	return result
}

func addBenchmark(cfg singleAddBenchmarkConfig) (measurements []float64) {
	prng := misc.PRNG{State: cfg.seed}
	numberOfSets := cfg.numOfSets
	setSize := cfg.finalSetSize
	set := make([]*set3.Set3[uint64], numberOfSets)
	for i := range numberOfSets {
		set[i] = set3.EmptyWithCapacity[uint64](cfg.initSize)
	}
	timePerRound := make([]float64, cfg.rounds)
	runtime.GC()
	debug.SetGCPercent(-1)
	defer debug.SetGCPercent(100)
	for r := range cfg.rounds {
		for s := range numberOfSets {
			set[s].Clear()
		}
		startTime := misc.SampleTime()
		for s := range numberOfSets {
			currentSet := set[s]
			for range setSize {
				currentSet.Add(prng.Uint64())
			}
		}
		endTime := misc.SampleTime()
		diff := float64(misc.DiffTimeStamps(startTime, endTime))
		timePerRound[r] = diff - misc.GetSampleTimeRuntime() - (rngOverhead * float64(numberOfSets*setSize))
	}
	return timePerRound
}

func toNanoSecondsPerAdd(measurements []float64, addsPerRound uint32) []float64 {
	result := make([]float64, len(measurements))
	div := 1.0 / float64(addsPerRound)
	for i, m := range measurements {
		result[i] = float64(m) * div
	}
	return result
}

func printTotalRuntime(start time.Time) string {
	end := time.Now()
	return fmt.Sprintf("\nTotal runtime of benchmark: %v\n", end.Sub(start))
}

func getNumberOfConfigs(setup benchmarkSetup) uint32 {
	result := uint32(0)
	for setSize := range setup.setSizes() {
		for range initSizes2(setSize, setup.fromSetSize, setup.toSetSize, setup.Pstep, setup.Istep, setup.RelativeLimit, setup.AbsoluteLimit) {
			result++
		}
	}
	return result
}

type programParametrization struct {
	fromSetSize, toSetSize, targetAddsPerRound uint32
	expRuntimePerAdd, secondsPerConfig         float64
	Pstep                                      *float64
	Istep                                      *uint32
	RelativeLimit                              *float64
	AbsoluteLimit                              *uint32
	//step                                       Step
}

type benchmarkSetup struct {
	programParametrization
	totalAddsPerConfig uint32
}

func benchmarkSetupFrom(p programParametrization) (benchmarkSetup, error) {
	result := benchmarkSetup{
		programParametrization: p,
		totalAddsPerConfig:     uint32(p.secondsPerConfig * 1_000_000_000.0 / p.expRuntimePerAdd),
	}
	if p.Pstep != nil && p.Istep != nil {
		return result, errors.New("Pstep and Istep are both defined")
	}
	if p.Pstep == nil && p.Istep == nil {
		one := uint32(1)
		p.Istep = &one
	}
	if p.RelativeLimit != nil && p.AbsoluteLimit != nil {
		return result, errors.New("RelativeLimit and AbsoluteLimit are both defined")
	}
	if p.RelativeLimit == nil && p.AbsoluteLimit == nil {
		onhundret := 100.0
		p.RelativeLimit = &onhundret
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

func (setup *benchmarkSetup) setSizes() func(yield func(uint32) bool) {
	return func(yield func(uint32) bool) {
		for setSize := setup.fromSetSize; setSize <= setup.toSetSize; setSize++ {
			if !yield(setSize) {
				return
			}
		}
	}
}

func stepsHeadings(setSizeFrom, setSizeTo uint32, Pstep *float64, Istep *uint32, RelativeLimit *float64, AbsoluteLimit *uint32) ([]string, error) {
	result := make([]string, 0, 100)
	if Pstep == nil && Istep == nil {
		return nil, errors.New("Pstep == nil && Istep == nil")
	}
	if Pstep != nil && Istep != nil {
		return nil, errors.New("Pstep != nil && Istep != nil")
	}
	if RelativeLimit == nil && AbsoluteLimit == nil {
		return nil, errors.New("RelativeLimit == nil && Istep == nil")
	}
	if RelativeLimit != nil && AbsoluteLimit != nil {
		return nil, errors.New("RelativeLimit != nil && AbsoluteLimit != nil")
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
		start := uint32(0)
		limit := uint32((math.Round(*RelativeLimit * float64(setSizeTo) / 100.0))) + (*Istep)
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
		start := uint32(0)
		limit := (*AbsoluteLimit) - setSizeFrom + (*Istep)
		for i := start; i < limit; i += (*Istep) {
			result = append(result, fmt.Sprintf("+%d ", i))
		}
	}
	return result, nil
}

func initSizes2(setSize uint32, setSizeFrom, setSizeTo uint32, Pstep *float64, Istep *uint32, RelativeLimit *float64, AbsoluteLimit *uint32) func(yield func(uint32) bool) {
	if Pstep != nil && RelativeLimit != nil {
		start := float64(0)
		limit := (*RelativeLimit) + (*Pstep)
		return func(yield func(uint32) bool) {
			for f := start; f < limit; f += (*Pstep) {
				retval := setSize + uint32(math.Round(float64(setSize)*f/100.0))
				if !yield(retval) {
					return
				}
			}
		}
	}
	if Pstep != nil && AbsoluteLimit != nil {
		start := float64(0)
		factor := float64(*AbsoluteLimit) / float64(setSizeFrom)
		limit := (factor-1.0)*100.0 + *Pstep
		return func(yield func(uint32) bool) {
			for f := start; f < limit; f += (*Pstep) {
				retval := setSize + uint32(math.Round(float64(setSize)*f/100.0))
				if !yield(retval) {
					return
				}
			}
		}
	}
	if Istep != nil && RelativeLimit != nil {
		start := uint32(0)
		limit := uint32((math.Round(*RelativeLimit * float64(setSizeTo) / 100.0))) + (*Istep)
		return func(yield func(uint32) bool) {
			for i := setSize + start; i < setSize+limit; i += (*Istep) {
				if !yield(i) {
					return
				}
			}
		}
	}
	if Istep != nil && AbsoluteLimit != nil {
		start := uint32(0)
		limit := (*AbsoluteLimit) - setSizeFrom + (*Istep)
		return func(yield func(uint32) bool) {
			for i := setSize + start; i < setSize+limit; i += (*Istep) {
				if !yield(i) {
					return
				}
			}
		}

	}
	panic("4tapgpo438tnaowghp")
}

type singleAddBenchmarkConfig struct {
	initSize           uint32
	finalSetSize       uint32
	targetAddsPerRound uint32
	totalAddsPerConfig uint32
	numOfSets          uint32
	actualAddsPerRound uint32
	rounds             uint32
	seed               uint64
}

func makeSingleAddBenchmarkConfig(initSize, setSize, targetAddsPerRound, totalAddsPerConfig uint32, seed uint64) singleAddBenchmarkConfig {
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
	result.numOfSets = uint32(math.Round(float64(targetAddsPerRound) / float64(setSize)))
	result.actualAddsPerRound = result.numOfSets * setSize // actualAddsPerRound ~ targetAddsPerRound
	result.rounds = uint32(math.Round(float64(totalAddsPerConfig) / float64(result.actualAddsPerRound)))
	return result
}

var cli struct {
	Add struct {
		Loadfactor struct {
			From             uint32        `arg:"" help:"First set size to benchmark (inclusive)." short:"f"`
			To               uint32        `arg:"" help:"Last set size to benchmark (inclusive)." short:"t"`
			AddsPerRound     uint32        `help:"Number of Add(prng.Uint64()) instructions between two time measurements. Balance the value between memory consumption (cache size/speed) and timer precision of your runtime environment (e.g., Windows=100ns)." short:"r" default:"50000"`
			RuntimePerConfig time.Duration `help:"Benchmarking time per configuration (combination of initial set size and number of values to add)." short:"c" default:"1.5s"`
			RuntimePerAdd    time.Duration `help:"Expected runtime per single Add(prng.Uint64()) instruction. Used to calculate the necessary number of iterations to meet the runtime-per-config and to predcict the total runtime of the benchmark." short:"a" default:"8ns"`
			Pstep            *float64      `help:"Uses percentage value to increase the initial set size in benchmark configurations until the limit is reached (see relative-limit or absolute-limit). You can either specify a pstep or an istep. Default is an istep of size 1." short:"p" xor:"Pstep, Istep"`
			Istep            *uint32       `help:"Uses integer value to increase the initial set size in benchmark configurations until the limit is reached (see relative-limit or absolute-limit). You can either specify a pstep or an istep. Default is an istep of size 1." short:"i" xor:"Pstep, Istep"`
			RelativeLimit    *float64      `help:"Increase initial (pre-allocated) set sizes until at least x% headroom are reached. You can either specify a relative-limit or an absolute-limit. Default is a relative-limit of 100%." default:"100.0" xor:"RelativeLimit, AbsoluteLimit"`
			AbsoluteLimit    *uint32       `help:"Increase initial (pre-allocated) set sizes until at least an initial set size of x is reached. You can either specify a relative-limit or an absolute-limit. Default is a relative-limit of 100%." xor:"RelativeLimit, AbsoluteLimit"`
		} `cmd:"" help:"Perform a loadfactor test using different initial (pre-allocated) set sizes. This benchmark creates empty sets of a defined size x and then adds y random numers via Add(prng.Uint64()). Any combination of x and y is called 'configration'."`
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
		fmt.Println(cli.Add.Loadfactor.From, cli.Add.Loadfactor.To, cli.Add.Loadfactor.AddsPerRound, cli.Add.Loadfactor.RuntimePerConfig, cli.Add.Loadfactor.RuntimePerAdd)
		fmt.Println(cli.Add.Loadfactor.Pstep, cli.Add.Loadfactor.Istep, cli.Add.Loadfactor.RelativeLimit, cli.Add.Loadfactor.AbsoluteLimit)
	default:
		fmt.Print("Check ctx.Command(): ")
		fmt.Println(ctx.Command())
		panic("done.")
	}

	var pp programParametrization
	pp.fromSetSize = cli.Add.Loadfactor.From
	pp.toSetSize = cli.Add.Loadfactor.To
	pp.targetAddsPerRound = cli.Add.Loadfactor.AddsPerRound
	pp.secondsPerConfig = float64(cli.Add.Loadfactor.RuntimePerConfig.Nanoseconds() / 1_000_000_000_000.0)
	pp.expRuntimePerAdd = float64(cli.Add.Loadfactor.RuntimePerAdd.Nanoseconds())

	setup, err := benchmarkSetupFrom(pp)

	if err != nil {
		panic(err)
	}

	printSetup(setup)

	start := time.Now()
	defer fmt.Print(printTotalRuntime(start))

	fmt.Printf("setSize ")
	headings, _ := stepsHeadings(setup.fromSetSize, setup.toSetSize, setup.Pstep, setup.Istep, setup.RelativeLimit, setup.AbsoluteLimit)
	for _, columnH := range headings {
		fmt.Print(columnH)
	}
	fmt.Print("\n")
	for setSize := range setup.setSizes() {
		fmt.Printf("%d ", setSize)
		for initSize := range initSizes2(setSize, setup.fromSetSize, setup.toSetSize, setup.Pstep, setup.Istep, setup.RelativeLimit, setup.AbsoluteLimit) {
			cfg := makeSingleAddBenchmarkConfig(initSize, setSize, setup.targetAddsPerRound, setup.totalAddsPerConfig, 0xABCDEF0123456789)
			measurements := addBenchmark(cfg)
			nsValues := toNanoSecondsPerAdd(measurements, cfg.actualAddsPerRound)
			median := misc.Median(nsValues)
			fmt.Printf("%.3f ", median)
			// fmt.Printf("%d ", initSize)
		}
		fmt.Printf("\n")
	}
}

func printSetup(p benchmarkSetup) {
	fmt.Printf("Architecture:\t\t\t%s\n", runtime.GOARCH)
	fmt.Printf("OS:\t\t\t\t%s\n", runtime.GOOS)
	fmt.Printf("Max timer precision:\t\t%.2fns\n", misc.GetSampleTimePrecision())
	fmt.Printf("SampleTime() runtime:\t\t%.2fns (informative, already subtracted from below measurement values)\n", misc.GetSampleTimeRuntime())
	fmt.Printf("prng.Uint64() runtime:\t\t%.2fns (informative, already subtracted from below measurement values)\n", rngOverhead)
	fmt.Printf("Exp. Add(prng.Uint64()) rt:\t%.2fns\n", p.expRuntimePerAdd)
	quantizationError := calcQuantizationError(p)
	fmt.Printf("Add()'s per round:\t\t%d (expect a quantization error of %.3f%%, i.e. %.3fns per Add)\n", p.targetAddsPerRound, quantizationError, quantizationError*p.expRuntimePerAdd)
	fmt.Printf("Add()'s per config:\t\t%d (should result in a benchmarking time of %.2fs per config)\n", p.totalAddsPerConfig, p.secondsPerConfig)
	//	fmt.Printf("Set3 sizes:\t\t\tfrom %d to %d, stepsize %v\n", p.fromSetSize, p.toSetSize, p.step.String())
	fmt.Printf("Set3 sizes:\t\t\tfrom %d to %d\n", p.fromSetSize, p.toSetSize)
	numberOfConfigs := getNumberOfConfigs(p)
	fmt.Printf("Number of configs:\t\t%d\n", numberOfConfigs)
	totalduration := predictTotalDuration(p)
	fmt.Printf("Expected total runtime:\t\t%v (assumption: %.2fns per Add(prng.Uint64()) and 12%% overhead for housekeeping)\n\n", totalduration, p.expRuntimePerAdd)
}

func calcQuantizationError(p benchmarkSetup) float64 {
	quantizationError := misc.GetSampleTimePrecision() * 100.0 / (p.expRuntimePerAdd * float64(p.targetAddsPerRound))
	return quantizationError
}

func predictTotalDuration(p benchmarkSetup) time.Duration {
	numberOfStepsPerSetSize := getNumberOfConfigs(p)
	totalduration := time.Duration(uint32(p.expRuntimePerAdd * float64(p.totalAddsPerConfig)))
	totalduration *= time.Duration(numberOfStepsPerSetSize)
	totalduration = time.Duration(float64(totalduration) * 1.12)
	return totalduration
}
