package main

import (
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

func printTotalRuntime(start time.Time) {
	end := time.Now()
	fmt.Printf("\nTotal runtime of benchmark: %v\n", end.Sub(start))
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
			result = append(result, fmt.Sprintf("+%.2f%%%% ", f)) // caution: strings are used in fmt.Printf() so encode %% twice
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

func initSizeValues(start uint32, step Step, limit uint32) []uint32 {
	result := make([]uint32, 0, 100)
	if step.isPercent {
		pval := step.percent
		for f := 0.0; f < 100.0+pval; f += pval {
			retval := start + uint32(math.Round(f*float64(start)/100.0))
			result = append(result, retval)
		}
	} else {
		numberOfSteps := limit
		ival := step.integerStep
		for i := start; i <= start+numberOfSteps; i += ival {
			result = append(result, i)
		}
	}
	return result
}

type programParametrization struct {
	fromSetSize, toSetSize, targetAddsPerRound uint32
	expRuntimePerAdd, secondsPerConfig         float64
	step                                       Step
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
	var p programParametrization

	flag.Uint32Var(&p.fromSetSize, "from", 100, "First set size to benchmark (inclusive)")
	flag.Uint32Var(&p.toSetSize, "to", 200, "Last set size to benchmark (inclusive)")
	// 50_000 x ~8ns = ~400_000ns; Timer precision 100ns (Windows) => 0,025% error, i.e. 0,02ns per Add()
	flag.Uint32Var(&p.targetAddsPerRound, "apr", 50_000, "AddsPerRound - instructions between two measurements. Balance between memory consumption (cache!) and timer precision (Windows: 100ns)")
	flag.Float64Var(&p.secondsPerConfig, "spc", 1.0, "SecondsPerConfig - estimated benchmark time per configuration in seconds")
	flag.Float64Var(&p.expRuntimePerAdd, "erpa", 8.0, "Expected Runtime Per Add - in nanoseconds per instruction. Used to predcict runtimes")
	flag.Var(&p.step, "step", "Step to increment headroom of pre-allocated sets. Either percent of set size (e.g. \"2.5%\") or absolut value (e.g. \"2\") (default: 1)")

	flag.Parse()

	if !p.step.isSet {
		p.step.isSet = true
		p.step.isPercent = false
		p.step.integerStep = 1
	}

	if p.toSetSize < p.fromSetSize {
		panic("to < from")
	}

	if p.toSetSize > 1<<28 {
		panic("to too big")
	}

	totalAddsPerConfig := uint32(p.secondsPerConfig * (1_000_000_000.0 / p.expRuntimePerAdd))

	printSetup(p, totalAddsPerConfig)

	start := time.Now()
	defer printTotalRuntime(start)

	fmt.Printf("setSize ")
	for _, columnH := range columnHeadings(p.step, p.toSetSize) {
		fmt.Print(columnH)
	}
	fmt.Print("\n")
	for currentSetSize := p.fromSetSize; currentSetSize <= p.toSetSize; currentSetSize++ {
		fmt.Printf("%d ", currentSetSize)
		for _, initSize := range initSizeValues(currentSetSize, p.step, p.toSetSize) {
			cfg := makeSingleAddBenchmarkConfig(initSize, currentSetSize, p.targetAddsPerRound, totalAddsPerConfig, 0xABCDEF0123456789)
			measurements := addBenchmark(cfg)
			nsValues := toNanoSecondsPerAdd(measurements, cfg.actualAddsPerRound)
			median := misc.Median(nsValues)
			fmt.Printf("%.3f ", median)
		}
		fmt.Printf("\n")
	}
}

func printSetup(p programParametrization, totalAddsPerConfig uint32) {
	fmt.Printf("Architecture:\t\t\t%s\n", runtime.GOARCH)
	fmt.Printf("OS:\t\t\t\t%s\n", runtime.GOOS)
	fmt.Printf("Max timer precision:\t\t%.2fns\n", misc.GetSampleTimePrecision())
	fmt.Printf("SampleTime() runtime:\t\t%.2fns (informative, already subtracted from below measurement values)\n", misc.GetSampleTimeRuntime())
	fmt.Printf("prng.Uint64() runtime:\t\t%.2fns (informative, already subtracted from below measurement values)\n", rngOverhead)
	fmt.Printf("Exp. Add(prng.Uint64()) rt:\t%.2fns\n", p.expRuntimePerAdd)
	quantizationError := calcQuantizationError(p)
	fmt.Printf("Add()'s per round:\t\t%d (expect a quantization error of %.3f%%, i.e. %.3fns per Add)\n", p.targetAddsPerRound, quantizationError, quantizationError*p.expRuntimePerAdd)
	fmt.Printf("Add()'s per config:\t\t%d (should result in a benchmarking time of %.2fs per config)\n", totalAddsPerConfig, p.secondsPerConfig)
	fmt.Printf("Set3 sizes:\t\t\tfrom %d to %d, stepsize %v\n", p.fromSetSize, p.toSetSize, p.step.String())
	numberOfStepsPerSetSize := getNumberOfSteps(p.step, p.toSetSize)
	fmt.Printf("Number of configs:\t\t%d\n", numberOfStepsPerSetSize*(p.toSetSize-p.fromSetSize+1))
	totalduration := predictTotalDuration(p, totalAddsPerConfig)
	fmt.Printf("Expected total runtime:\t\t%v (assumption: %fns per Add(prng.Uint64()) and 12%% overhead for housekeeping)\n\n", totalduration, p.expRuntimePerAdd)
}

func calcQuantizationError(p programParametrization) float64 {
	quantizationError := misc.GetSampleTimePrecision() * 100.0 / (p.expRuntimePerAdd * float64(p.targetAddsPerRound))
	return quantizationError
}

func predictTotalDuration(p programParametrization, totalAddsPerConfig uint32) time.Duration {
	numberOfStepsPerSetSize := getNumberOfSteps(p.step, p.toSetSize)
	totalduration := time.Duration(uint32(p.expRuntimePerAdd * float64(totalAddsPerConfig)))
	totalduration *= time.Duration(numberOfStepsPerSetSize)
	totalduration *= time.Duration(p.toSetSize - p.fromSetSize + 1)
	totalduration = time.Duration(float64(totalduration) * 1.12)
	return totalduration
}
