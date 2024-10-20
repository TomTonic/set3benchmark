package main

import (
	"math"
	"math/rand"
	"reflect"
	"regexp"
	"testing"
	"time"

	misc "github.com/TomTonic/set3benchmark/misc"
	"github.com/stretchr/testify/assert"
)

func TestDoBenchmark2(t *testing.T) {
	cfg := singleAddBenchmarkConfig{
		rounds:       uint64(72),
		numOfSets:    uint64(10),
		initSize:     uint64(150),
		finalSetSize: uint64(100),
		seed:         0xabcdef,
	}

	result := addBenchmark(cfg)

	assert.True(t, uint64(len(result)) == cfg.rounds, "Result should return %d measurements. It returned %d measurements.", cfg.rounds, len(result))
	assert.False(t, containsZero(result), "Result should not contain zeros, but it does.")
	assert.False(t, containsNegative(result), "Result should not contain negative numbers, but it does.")

	reportedPrecision := misc.GetSampleTimePrecision()
	assert.True(t, atLeastNtimesPrecision(20.0, reportedPrecision, result),
		"Result should only contain values that exceed %fx the timer precision of %fns, but it does not. The minimum Value is %v.", 20.0, reportedPrecision, minVal(result))
}

func containsZero(measurements []float64) bool {
	for _, d := range measurements {
		if d == 0 {
			return true
		}
	}
	return false
}

func containsNegative(measurements []float64) bool {
	for _, d := range measurements {
		if d < 0 {
			return true
		}
	}
	return false
}

func atLeastNtimesPrecision(nTimes float64, precision float64, measurements []float64) bool {
	for _, d := range measurements {
		if d < precision*nTimes {
			return false
		}
	}
	return true
}

func minVal(measurements []float64) float64 {
	min := math.MaxFloat64
	for _, d := range measurements {
		if d < min {
			min = d
		}
	}
	return min
}

func TestGetNumberOfConfigs(t *testing.T) {
	oneF := 1.0
	tenF := 10.0
	twentyfiveF := 25.0
	//thirtyF := 30.0
	onehundretF := 100.0
	//onepointfiveF := 1.5
	//oneI := uint64(1)
	twoI := uint64(2)
	twentyI := uint64(20)
	//tenI := uint64(10)
	onehundretI := uint64(100)
	tests := []struct {
		setSizeFrom   uint64
		setSizeTo     uint64
		Pstep         *float64
		Istep         *uint64
		RelativeLimit *float64
		AbsoluteLimit *uint64
		expected      uint64
	}{
		{7, 9, &oneF, nil, &onehundretF, nil, 101 * 3},        // expect: 0%, 1%, 2%, ..., 99%, 100% for rows 7, 8 and 9
		{33, 33, &tenF, nil, &onehundretF, nil, 11},           // expect: 0%, 10%, 20%, ..., 90%, 100% for 1 row
		{19, 20, &twentyfiveF, nil, &onehundretF, nil, 5 * 2}, // expect: 0%, 25%, 50%, 75%, 100% for 2 rows

		/*
			Next test:
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
		*/
		{50, 63, &tenF, nil, nil, &onehundretI, 129},

		/*
			Next test:
			setSize	Istep	RelativeLimit	Expected
			5		2		100%			+0	+2	+4	+6
			6		2		100%			+0	+2	+4	+6
			7		2		100%			+0	+2	+4	+6	+8
			8		2		100%			+0	+2	+4	+6	+8
			9		2		100%			+0	+2	+4	+6	+8	+10
		*/
		{5, 9, nil, &twoI, &onehundretF, nil, 24},

		/*
			Next test:
			setSize	Istep	AbsoluteLimit	Expected
			5		2		20				+0	+2	+4	+6	+8	+10	+12	+14	+16
			6		2		20				+0	+2	+4	+6	+8	+10	+12	+14
			7		2		20				+0	+2	+4	+6	+8	+10	+12	+14
			8		2		20				+0	+2	+4	+6	+8	+10	+12
		*/
		{5, 8, nil, &twoI, nil, &twentyI, 32},

		//{19, Step{true, true, 30.0, 0}, 5},                    // expect: 0%, 30%, 60%, 90%, 120%
		//{712, Step{true, true, 1.5, 0}, 68},                   // expect: 0%, 1.5%, 3.0%, ... 97.5%, 99%, 100.5%
		//{234, Step{true, false, 0.0, 1}, 235},                 // expect: 0, 1, 2, ..., 233, 234
		//{33, Step{true, false, 0.0, 10}, 5},                   // expect: 0, 10, 20, 30, 40
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := getNumberOfConfigs(tt.setSizeFrom, tt.setSizeTo, tt.Pstep, tt.Istep, tt.RelativeLimit, tt.AbsoluteLimit)
			if result != tt.expected {
				t.Errorf("got %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestStepsHeadings(t *testing.T) {
	tenF := 10.0
	twentyfiveF := 25.0
	thirtyF := 30.0
	onehundretF := 100.0
	tenI := uint64(10)
	twentyfiveI := uint64(25)
	thirtyI := uint64(30)
	onehundretI := uint64(100)
	tests := []struct {
		setSizeFrom   uint64
		setSizeTo     uint64
		Pstep         *float64
		Istep         *uint64
		RelativeLimit *float64
		AbsoluteLimit *uint64
		err           bool
		expected      []string
	}{
		// Pstep && RelativeLimit, setSizeFrom/setSizeTo are irrelevant
		{5, 10, &tenF, nil, &onehundretF, nil, false, []string{"+0.0% ", "+10.0% ", "+20.0% ", "+30.0% ", "+40.0% ", "+50.0% ", "+60.0% ", "+70.0% ", "+80.0% ", "+90.0% ", "+100.0% "}},
		{5, 10, &twentyfiveF, nil, &onehundretF, nil, false, []string{"+0.0% ", "+25.0% ", "+50.0% ", "+75.0% ", "+100.0% "}},
		{5, 10, &thirtyF, nil, &onehundretF, nil, false, []string{"+0.0% ", "+30.0% ", "+60.0% ", "+90.0% ", "+120.0% "}},

		// Istep && AbsoluteLimit, use setSizeFrom for longest Sequence
		{10, 20, nil, &tenI, nil, &onehundretI, false, []string{"+0 ", "+10 ", "+20 ", "+30 ", "+40 ", "+50 ", "+60 ", "+70 ", "+80 ", "+90 "}},
		{5, 10, nil, &twentyfiveI, nil, &onehundretI, false, []string{"+0 ", "+25 ", "+50 ", "+75 ", "+100 "}},
		{5, 10, nil, &thirtyI, nil, &onehundretI, false, []string{"+0 ", "+30 ", "+60 ", "+90 ", "+120 "}},

		// Istep & RelativeLimit, use setSizeTo for longest sequence
		{10, 50, nil, &tenI, &onehundretF, nil, false, []string{"+0 ", "+10 ", "+20 ", "+30 ", "+40 ", "+50 "}},
		{10, 100, nil, &twentyfiveI, &thirtyF, nil, false, []string{"+0 ", "+25 ", "+50 "}},

		// Pstep & AbsoluteLimit, use setSizeFrom for longest Sequence
		{50, 60, &tenF, nil, nil, &onehundretI, false, []string{"+0.0% ", "+10.0% ", "+20.0% ", "+30.0% ", "+40.0% ", "+50.0% ", "+60.0% ", "+70.0% ", "+80.0% ", "+90.0% ", "+100.0% "}},
		{1, 5, &onehundretF, nil, nil, &tenI, false, []string{"+0.0% ", "+100.0% ", "+200.0% ", "+300.0% ", "+400.0% ", "+500.0% ", "+600.0% ", "+700.0% ", "+800.0% ", "+900.0% "}},

		// error cases
		{1, 2, nil, nil, nil, &tenI, true, []string{}},
		{1, 2, &tenF, &tenI, nil, &onehundretI, true, []string{}},
		{1, 2, &tenF, nil, nil, nil, true, []string{}},
		{1, 2, &tenF, nil, &onehundretF, &onehundretI, true, []string{}},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result, err := stepsHeadings(tt.setSizeFrom, tt.setSizeTo, tt.Pstep, tt.Istep, tt.RelativeLimit, tt.AbsoluteLimit)
			if err != nil != tt.err {
				t.Errorf("returned error: got error '%v', wanted error %v", err, tt.err)
			}
			if !tt.err && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestToNSperAdd(t *testing.T) {
	tests := []struct {
		measurements []float64
		addsPerRound uint64
		expected     []float64
	}{
		{[]float64{10, 20}, 2, []float64{5, 10}},
		{[]float64{100, 200}, 4, []float64{25, 50}},
		{[]float64{0, 50}, 5, []float64{0, 10}},
		{[]float64{1000}, 10, []float64{100}},
		{[]float64{}, 1, []float64{}},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := toNanoSecondsPerAdd(tt.measurements, tt.addsPerRound)
			if len(result) != len(tt.expected) {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("at index %d, got %v, want %v", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestMakeSingleAddBenchmarkConfig(t *testing.T) {
	tests := []struct {
		initSize             uint64
		setSize              uint64
		targetAddsPerRound   uint64
		totalAddsPerConfig   uint64
		expectedNumOfSets    uint64
		expectedAddsPerRound uint64
		expectedRounds       uint64
	}{
		{20, 20, 2_000, 20_000, 100, 2_000, 10},
		{20, 20, 1_999, 20_000, 100, 2_000, 10},
		{20, 20, 2_001, 20_000, 100, 2_000, 10},
		{20, 20, 2_000, 20_010, 100, 2_000, 10},
		{20, 20, 1_999, 20_010, 100, 2_000, 10},
		{20, 20, 2_001, 20_010, 100, 2_000, 10},
		{20, 20, 2_000, 19_990, 100, 2_000, 10},
		{20, 20, 1_999, 19_990, 100, 2_000, 10},
		{20, 20, 2_001, 19_990, 100, 2_000, 10},
		// more/less sets
		{20, 20, 2_020, 20_000, 101, 2_020, 10},
		{20, 20, 1_980, 20_000, 99, 1_980, 10},
		// more/less rounds
		{20, 20, 2_000, 21_001, 100, 2_000, 11},
		{20, 20, 2_000, 18_999, 100, 2_000, 9},
		// more/less sets and rounds
		{20, 20, 1_000, 20_000, 50, 1_000, 20},
		{20, 20, 4_000, 20_000, 200, 4_000, 5},
		// targetAddsPerRound > totalAddsPerConfig
		{20, 20, 2_000, 1_000, 100, 2_000, 1},
		// setSize > targetAddsPerRound
		{20, 2_000, 200, 20_000, 1, 2_000, 10},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			config := makeSingleAddBenchmarkConfig(tt.initSize, tt.setSize, tt.targetAddsPerRound, tt.totalAddsPerConfig, 0)
			if config.numOfSets != tt.expectedNumOfSets {
				t.Errorf("expected numOfSets %v, got %v", tt.expectedNumOfSets, config.numOfSets)
			}
			if config.actualAddsPerRound != tt.expectedAddsPerRound {
				t.Errorf("expected actualAddsPerRound %v, got %v", tt.expectedAddsPerRound, config.actualAddsPerRound)
			}
			if config.rounds != tt.expectedRounds {
				t.Errorf("expected rounds %v, got %v", tt.expectedRounds, config.rounds)
			}
		})
	}
}

func TestMakeSingleAddBenchmarkConfigRandom(t *testing.T) {
	for range 5_000 {
		initSize := rand.Uint64()%(1<<20) + 1
		setSize := rand.Uint64()%(1<<20) + 1
		targetAddsPerRound := rand.Uint64()%(1<<20) + 1
		totalAddsPerConfig := rand.Uint64()%(1<<20) + 1
		t.Run("", func(t *testing.T) {
			config := makeSingleAddBenchmarkConfig(initSize, setSize, targetAddsPerRound, totalAddsPerConfig, 0)
			assert.True(t, config.finalSetSize <= config.targetAddsPerRound, "config.finalSetSize(%d) > config.targetAddsPerRound(%d)", config.finalSetSize, config.targetAddsPerRound)
			assert.True(t, config.targetAddsPerRound <= config.totalAddsPerConfig, "config.targetAddsPerRound(%d) > config.totalAddsPerConfig(%d)", config.targetAddsPerRound, config.totalAddsPerConfig)
			actualAddsPerConfig := config.rounds * config.numOfSets * config.finalSetSize
			assert.True(t, config.totalAddsPerConfig-config.targetAddsPerRound <= actualAddsPerConfig, "config.totalAddsPerConfig-config.targetAddsPerRound(%d-%d=%d) > actualAddsPerConfig(%d)",
				config.totalAddsPerConfig, config.targetAddsPerRound, config.totalAddsPerConfig-config.targetAddsPerRound, actualAddsPerConfig)
			assert.True(t, config.totalAddsPerConfig+config.targetAddsPerRound >= actualAddsPerConfig, "config.totalAddsPerConfig+config.targetAddsPerRound(%d+%d=%d) < actualAddsPerConfig(%d)",
				config.totalAddsPerConfig, config.targetAddsPerRound, config.totalAddsPerConfig+config.targetAddsPerRound, actualAddsPerConfig)
		})
	}
}

func TestPredictTotalDuration(t *testing.T) {
	one := uint64(1)
	pointone := float64(0.1)
	onehundret := float64(100.0)
	tests := []struct {
		name             string
		p                programParametrization
		expectedDuration time.Duration
	}{
		{
			name: "Basic case with integer step",
			p: programParametrization{
				Istep:              &one,
				RelativeLimit:      &onehundret,
				targetAddsPerRound: 10,
				toSetSize:          10,
				fromSetSize:        1,
				expRuntimePerAdd:   2.0,
				secondsPerConfig:   1.0,
			},
			expectedDuration: time.Duration(72_800_000_000),
		},
		{
			name: "Large set size with percent step",
			p: programParametrization{
				Pstep:              &pointone,
				RelativeLimit:      &onehundret,
				targetAddsPerRound: 20,
				toSetSize:          20,
				fromSetSize:        5,
				expRuntimePerAdd:   8,
				secondsPerConfig:   1.0,
			},
			expectedDuration: time.Duration(17_955_840_000_000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup, err := benchmarkSetupFrom(tt.p)
			if err != nil {
				t.Errorf("error converting %v: %v", tt.p, err)
			}
			got := predictTotalDuration(setup)
			if got != tt.expectedDuration {
				t.Errorf("predictTotalDuration() = %v, want %v", got, tt.expectedDuration)
			}
		})
	}
}

func TestCalcQuantizationError(t *testing.T) {
	tests := []struct {
		name             string
		p                programParametrization
		expectedErrorWin float64
		expectedErrorLin float64
	}{
		{
			name: "Basic case",
			p: programParametrization{
				expRuntimePerAdd:   8.0,
				targetAddsPerRound: 50_000,
			},
			expectedErrorWin: 100.0 * 100.0 / (8.0 * 50_000),
			expectedErrorLin: 30.0 * 100.0 / (8.0 * 50_000),
		},
		{
			name: "Small number of adds per round",
			p: programParametrization{
				expRuntimePerAdd:   1.5,
				targetAddsPerRound: 100,
			},
			expectedErrorWin: 100.0 * 100.0 / (1.5 * 100),
			expectedErrorLin: 30.0 * 100.0 / (1.5 * 100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup, _ := benchmarkSetupFrom(tt.p)
			got := calcQuantizationError(setup)
			if got > tt.expectedErrorWin || got < tt.expectedErrorLin {
				t.Errorf("calcQuantizationError() = %v, want something between %v and %v", got, tt.expectedErrorLin, tt.expectedErrorWin)
			}
		})
	}
}

func TestPrintTotalRuntime(t *testing.T) {

	start := time.Now().Add(-time.Second) // Simulate a start time 1 second ago
	s := printTotalRuntime(start)

	// Define the regular expression to match the output
	re := regexp.MustCompile(`\nTotal runtime of benchmark: \d+(\.\d+)?s\n`)

	if !re.MatchString(s) {
		t.Errorf("Output did not match expected format: %v", s)
	}
}

func TestBenchmarkSetupFrom(t *testing.T) {
	tests := []struct {
		name    string
		input   programParametrization
		wantErr bool
	}{
		{
			name: "Valid parameters",
			input: programParametrization{
				fromSetSize:        10,
				toSetSize:          20,
				targetAddsPerRound: 20,
				expRuntimePerAdd:   1.0,
				secondsPerConfig:   10.0,
			},
			wantErr: false,
		},
		{
			name: "toSetSize less than fromSetSize",
			input: programParametrization{
				fromSetSize:        20,
				toSetSize:          10,
				targetAddsPerRound: 5,
				expRuntimePerAdd:   1.0,
				secondsPerConfig:   10.0,
			},
			wantErr: true,
		},
		{
			name: "toSetSize too big",
			input: programParametrization{
				fromSetSize:        10,
				toSetSize:          1 << 29,
				targetAddsPerRound: 5,
				expRuntimePerAdd:   1.0,
				secondsPerConfig:   10.0,
			},
			wantErr: true,
		},
		{
			name: "secondsPerConfig too low",
			input: programParametrization{
				fromSetSize:        10,
				toSetSize:          20,
				targetAddsPerRound: 5,
				expRuntimePerAdd:   1.0,
				secondsPerConfig:   0.0,
			},
			wantErr: true,
		},
		{
			name: "targetAddsPerRound too big",
			input: programParametrization{
				fromSetSize:        10,
				toSetSize:          20,
				targetAddsPerRound: 1000000000,
				expRuntimePerAdd:   8.0,
				secondsPerConfig:   0.1,
			},
			wantErr: true,
		},
		{
			name: "targetAddsPerRound too small for 'to'",
			input: programParametrization{
				fromSetSize:        10,
				toSetSize:          20,
				targetAddsPerRound: 5,
				expRuntimePerAdd:   1.0,
				secondsPerConfig:   10.0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := benchmarkSetupFrom(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("benchmarkSetupFrom() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSetSizes(t *testing.T) {
	tests := []struct {
		name        string
		fromSetSize uint64
		toSetSize   uint64
		expected    []uint64
	}{
		{
			name:        "Normal range",
			fromSetSize: 1,
			toSetSize:   5,
			expected:    []uint64{1, 2, 3, 4, 5},
		},
		{
			name:        "Single value range",
			fromSetSize: 3,
			toSetSize:   3,
			expected:    []uint64{3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := 0
			for setSize := range setSizes(tt.fromSetSize, tt.toSetSize) {
				assert.True(t, tt.expected[i] == setSize, "%v!=%v", tt.expected[i], setSize)
				i++
			}
		})
	}
}

func TestInitSizes(t *testing.T) {
	tenF := 10.0
	twentifiveF := 25.0
	onehundretF := 100.0
	twoI := uint64(2)
	twentyI := uint64(20)
	onehundretI := uint64(100)
	tests := []struct {
		name          string
		setSize       uint64
		Pstep         *float64
		Istep         *uint64
		RelativeLimit *float64
		AbsoluteLimit *uint64
		expected      []uint64
	}{
		{"SetSize 10, +10%, up to +100%", 10, &tenF, nil, &onehundretF, nil, []uint64{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}},
		{"SetSize 100, +25%, up to +100%", 100, &twentifiveF, nil, &onehundretF, nil, []uint64{100, 125, 150, 175, 200}},
		{"SetSize 53, +10%, up to 100", 53, &tenF, nil, nil, &onehundretI, []uint64{53, 58, 64, 69, 74, 80, 85, 90, 95, 101}},
		{"SetSize 7, +2, up to +100%", 7, nil, &twoI, &onehundretF, nil, []uint64{7, 9, 11, 13, 15}},
		{"SetSize 6, +2, up to 20", 6, nil, &twoI, nil, &twentyI, []uint64{6, 8, 10, 12, 14, 16, 18, 20}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := make([]uint64, 0, len(tt.expected))
			for initSize := range initSizes2(tt.setSize, tt.Pstep, tt.Istep, tt.RelativeLimit, tt.AbsoluteLimit) {
				result = append(result, initSize)
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}
