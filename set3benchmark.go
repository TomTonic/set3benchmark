package main

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	flag "github.com/spf13/pflag"

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

// Percent is a custom flag type for parsing percent values.
type Step struct {
	isSet       bool
	isPercent   bool
	percent     float64
	integerStep uint32
}

// Set parses the flag value and sets it.
func (p *Step) Set(value string) error {
	if strings.HasSuffix(value, "%") {
		value = strings.TrimSuffix(value, "%")
		p.isPercent = true
	} else {
		p.isPercent = false
	}
	if p.isPercent {
		parsedValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		p.percent = parsedValue
		p.isSet = true
	} else {
		parsedValue, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return err
		}
		p.integerStep = uint32(parsedValue) // #nosec G115
		p.isSet = true
	}
	return nil
}

// String returns the string representation of the flag value.
func (p *Step) String() string {
	if !p.isSet {
		return "1"
	}
	if p.isPercent {
		return fmt.Sprintf("%f%%", p.percent)
	}
	return fmt.Sprintf("%d", p.integerStep)
}

func (p *Step) Type() string {
	return "string"
}

func getNumberOfSteps(step Step, limit uint32) uint32 {
	if step.isPercent {
		pval := step.percent
		count := uint32(100.0 / pval)
		if float64(count)*pval < 100.0 {
			count++
		}
		count++
		return count
	}
	numberOfSteps := limit / step.integerStep
	if limit%step.integerStep != 0 {
		numberOfSteps++
	}
	return numberOfSteps + 1
}

func columnHeadings(step Step, limit uint32) []string {
	result := make([]string, 0, 100)
	if step.isPercent {
		pval := step.percent
		for f := 0.0; f < 100.0+pval; f += pval {
			result = append(result, fmt.Sprintf("+%.1f%% ", f))
		}
	} else {
		numberOfSteps := limit
		ival := step.integerStep
		for i := uint32(0); i < numberOfSteps+ival; i += ival {
			result = append(result, fmt.Sprintf("+%d ", i))
		}
	}
	return result
}

type programParametrization struct {
	fromSetSize, toSetSize, targetAddsPerRound uint32
	expRuntimePerAdd, secondsPerConfig         float64
	step                                       Step
}

type benchmarkSetup struct {
	programParametrization
	totalAddsPerConfig uint32
}

func benchmarkSetupFrom(p programParametrization) (benchmarkSetup, error) {
	if !p.step.isSet {
		p.step.isSet = true
		p.step.isPercent = true
		p.step.percent = 1.0
	}
	result := benchmarkSetup{
		programParametrization: p,
		totalAddsPerConfig:     uint32(p.secondsPerConfig * (1_000_000_000.0 / p.expRuntimePerAdd)),
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

func stepsHeadings(setSize uint32, Pstep *float64, Istep *uint32, RelativeLimit *float64, AbsoluteLimit *uint32) ([]string, error) {
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
		start := float64(0)
		limit := (*RelativeLimit) + (*Pstep)
		for f := start; f < limit; f += (*Pstep) {
			result = append(result, fmt.Sprintf("+%.1f%% ", f))
		}
	}
	if Pstep != nil && AbsoluteLimit != nil {
		start := float64(0)
		factor := float64(*AbsoluteLimit) / float64(setSize)
		limit := (factor-1.0)*100.0 + *Pstep
		for f := start; f < limit; f += (*Pstep) {
			result = append(result, fmt.Sprintf("+%.1f%% ", f))
		}
	}
	if Istep != nil && RelativeLimit != nil {
		start := uint32(0)
		limit := uint32((math.Round(*RelativeLimit * float64(setSize) / 100.0))) + (*Istep)
		for i := start; i < limit; i += (*Istep) {
			result = append(result, fmt.Sprintf("+%d ", i))
		}
	}
	if Istep != nil && AbsoluteLimit != nil {
		start := uint32(0)
		limit := (*AbsoluteLimit) + (*Istep)
		for i := start; i < limit; i += (*Istep) {
			result = append(result, fmt.Sprintf("+%d ", i))
		}
	}
	return result, nil
}

/*
func getNumberOfStepsNew(setSizeFrom, setSizeTo uint32, Pstep *float64, Istep *uint32, RelativeLimit *float64, AbsoluteLimit *uint32) uint32 {
	if Pstep != nil && RelativeLimit != nil {
		count := uint32(*RelativeLimit / *Pstep)
		if float64(count)*(*Pstep) < *RelativeLimit {
			count++
		}
		count++
		return count
	}
	if Pstep != nil && AbsoluteLimit != nil {
	}
	numberOfSteps := *AbsoluteLimit / *Istep
	if *AbsoluteLimit%*Istep != 0 {
		numberOfSteps++
	}
	return numberOfSteps + 1
}
*/

func (setup *benchmarkSetup) initSizes(setSize uint32) func(yield func(uint32) bool) {
	if setup.step.isPercent {
		return func(yield func(uint32) bool) {
			for f := 0.0; f < 100.0+setup.step.percent; f += setup.step.percent {
				retval := setSize + uint32(math.Round(f*float64(setSize)/100.0))
				if !yield(retval) {
					return
				}
			}
		}
	}
	return func(yield func(uint32) bool) {
		for i := setSize; i <= setSize+setup.toSetSize; i += setup.step.integerStep {
			if !yield(i) {
				return
			}
		}
	}
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

func main() {
	var pp programParametrization

	flag.Uint32Var(&pp.fromSetSize, "from", 100, "First set size to benchmark (inclusive)")
	flag.Uint32Var(&pp.toSetSize, "to", 200, "Last set size to benchmark (inclusive)")
	// 50_000 x ~8ns = ~400_000ns; Timer precision 100ns (Windows) => 0,025% error, i.e. 0,02ns per Add()
	flag.Uint32Var(&pp.targetAddsPerRound, "apr", 50_000, "Adds Per Round - instructions between two measurements. Balance between memory consumption (cache!) and timer precision (Windows: 100ns)")
	flag.Float64Var(&pp.secondsPerConfig, "spc", 1.0, "Seconds Per Config - estimated benchmark time per configuration in seconds")
	flag.Float64Var(&pp.expRuntimePerAdd, "erpa", 8.0, "Expected Runtime Per Add - in nanoseconds per instruction. Used to predcict runtimes")
	flag.Var(&pp.step, "step", "Step to increment headroom of pre-allocated sets. Either percent of set size (e.g. \"2.5%\") or absolut value (e.g. \"2\") (default: 1%)")

	flag.Parse()

	setup, err := benchmarkSetupFrom(pp)

	if err != nil {
		panic(err)
	}

	printSetup(setup)

	start := time.Now()
	defer fmt.Print(printTotalRuntime(start))

	fmt.Printf("setSize ")
	for _, columnH := range columnHeadings(setup.step, setup.toSetSize) {
		fmt.Print(columnH)
	}
	fmt.Print("\n")
	for setSize := range setup.setSizes() {
		fmt.Printf("%d ", setSize)
		for initSize := range setup.initSizes(setSize) {
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
	fmt.Printf("Set3 sizes:\t\t\tfrom %d to %d, stepsize %v\n", p.fromSetSize, p.toSetSize, p.step.String())
	numberOfStepsPerSetSize := getNumberOfSteps(p.step, p.toSetSize)
	fmt.Printf("Number of configs:\t\t%d\n", numberOfStepsPerSetSize*(p.toSetSize-p.fromSetSize+1))
	totalduration := predictTotalDuration(p)
	fmt.Printf("Expected total runtime:\t\t%v (assumption: %.2fns per Add(prng.Uint64()) and 12%% overhead for housekeeping)\n\n", totalduration, p.expRuntimePerAdd)
}

func calcQuantizationError(p benchmarkSetup) float64 {
	quantizationError := misc.GetSampleTimePrecision() * 100.0 / (p.expRuntimePerAdd * float64(p.targetAddsPerRound))
	return quantizationError
}

func predictTotalDuration(p benchmarkSetup) time.Duration {
	numberOfStepsPerSetSize := getNumberOfSteps(p.step, p.toSetSize)
	totalduration := time.Duration(uint32(p.expRuntimePerAdd * float64(p.totalAddsPerConfig)))
	totalduration *= time.Duration(numberOfStepsPerSetSize)
	totalduration *= time.Duration(p.toSetSize - p.fromSetSize + 1)
	totalduration = time.Duration(float64(totalduration) * 1.12)
	return totalduration
}
