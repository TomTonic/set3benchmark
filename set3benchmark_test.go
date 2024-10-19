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
		rounds:       uint32(72),
		numOfSets:    uint32(10),
		initSize:     uint32(150),
		finalSetSize: uint32(100),
		seed:         0xabcdef,
	}

	result := addBenchmark(cfg)

	assert.True(t, uint32(len(result)) == cfg.rounds, "Result should return %d measurements. It returned %d measurements.", cfg.rounds, len(result))
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

func TestGetNumberOfSteps(t *testing.T) {
	tests := []struct {
		limit    uint32
		step     Step
		expected uint32
	}{
		{7, Step{true, true, 1.0, 0}, 101},    // expect: 0%, 1%, 2%, ..., 99%, 100%
		{33, Step{true, true, 10.0, 0}, 11},   // expect: 0%, 10%, 20%, ..., 90%, 100%
		{19, Step{true, true, 25.0, 0}, 5},    // expect: 0%, 25%, 50%, 75%, 100%
		{19, Step{true, true, 30.0, 0}, 5},    // expect: 0%, 30%, 60%, 90%, 120%
		{712, Step{true, true, 1.5, 0}, 68},   // expect: 0%, 1.5%, 3.0%, ... 97.5%, 99%, 100.5%
		{234, Step{true, false, 0.0, 1}, 235}, // expect: 0, 1, 2, ..., 233, 234
		{33, Step{true, false, 0.0, 10}, 5},   // expect: 0, 10, 20, 30, 40
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := getNumberOfSteps(tt.step, tt.limit)
			if result != tt.expected {
				t.Errorf("got %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestColumnHeadings(t *testing.T) {
	tests := []struct {
		limit    uint32
		step     Step
		expected []string
	}{
		{33, Step{true, true, 10.0, 0}, []string{"+0.0% ", "+10.0% ", "+20.0% ", "+30.0% ", "+40.0% ", "+50.0% ", "+60.0% ", "+70.0% ", "+80.0% ", "+90.0% ", "+100.0% "}},
		{19, Step{true, true, 25.0, 0}, []string{"+0.0% ", "+25.0% ", "+50.0% ", "+75.0% ", "+100.0% "}},
		{19, Step{true, true, 30.0, 0}, []string{"+0.0% ", "+30.0% ", "+60.0% ", "+90.0% ", "+120.0% "}},
		{4, Step{true, false, 0.0, 1}, []string{"+0 ", "+1 ", "+2 ", "+3 ", "+4 "}},
		{33, Step{true, false, 0.0, 10}, []string{"+0 ", "+10 ", "+20 ", "+30 ", "+40 "}},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := columnHeadings(tt.step, tt.limit)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStepsHeadings(t *testing.T) {
	tenF := 10.0
	twentyfiveF := 25.0
	thirtyF := 30.0
	onehundretF := 100.0
	tenI := uint32(10)
	twentyfiveI := uint32(25)
	thirtyI := uint32(30)
	onehundretI := uint32(100)
	tests := []struct {
		setSize       uint32
		Pstep         *float64
		Istep         *uint32
		RelativeLimit *float64
		AbsoluteLimit *uint32
		err           bool
		expected      []string
	}{
		// Pstep && RelativeLimit
		{5, &tenF, nil, &onehundretF, nil, false, []string{"+0.0% ", "+10.0% ", "+20.0% ", "+30.0% ", "+40.0% ", "+50.0% ", "+60.0% ", "+70.0% ", "+80.0% ", "+90.0% ", "+100.0% "}},
		{5, &twentyfiveF, nil, &onehundretF, nil, false, []string{"+0.0% ", "+25.0% ", "+50.0% ", "+75.0% ", "+100.0% "}},
		{5, &thirtyF, nil, &onehundretF, nil, false, []string{"+0.0% ", "+30.0% ", "+60.0% ", "+90.0% ", "+120.0% "}},

		// Istep && AbsoluteLimit
		{5, nil, &tenI, nil, &onehundretI, false, []string{"+0 ", "+10 ", "+20 ", "+30 ", "+40 ", "+50 ", "+60 ", "+70 ", "+80 ", "+90 ", "+100 "}},
		{5, nil, &twentyfiveI, nil, &onehundretI, false, []string{"+0 ", "+25 ", "+50 ", "+75 ", "+100 "}},
		{5, nil, &thirtyI, nil, &onehundretI, false, []string{"+0 ", "+30 ", "+60 ", "+90 ", "+120 "}},

		// Istep & RelativeLimit
		{50, nil, &tenI, &onehundretF, nil, false, []string{"+0 ", "+10 ", "+20 ", "+30 ", "+40 ", "+50 "}},
		{100, nil, &twentyfiveI, &thirtyF, nil, false, []string{"+0 ", "+25 ", "+50 "}},

		// Pstep & AbsoluteLimit
		{50, &tenF, nil, nil, &onehundretI, false, []string{"+0.0% ", "+10.0% ", "+20.0% ", "+30.0% ", "+40.0% ", "+50.0% ", "+60.0% ", "+70.0% ", "+80.0% ", "+90.0% ", "+100.0% "}},
		{1, &onehundretF, nil, nil, &tenI, false, []string{"+0.0% ", "+100.0% ", "+200.0% ", "+300.0% ", "+400.0% ", "+500.0% ", "+600.0% ", "+700.0% ", "+800.0% ", "+900.0% "}},

		// error cases
		{1, nil, nil, nil, &tenI, true, []string{}},
		{1, &tenF, &tenI, nil, &onehundretI, true, []string{}},
		{1, &tenF, nil, nil, nil, true, []string{}},
		{1, &tenF, nil, &onehundretF, &onehundretI, true, []string{}},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result, err := stepsHeadings(tt.setSize, tt.Pstep, tt.Istep, tt.RelativeLimit, tt.AbsoluteLimit)
			if err != nil != tt.err {
				t.Errorf("returned error: got error '%v', wanted error %v", err, tt.err)
			}
			if !tt.err && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStep_Set(t *testing.T) {
	tests := []struct {
		input    string
		expected Step
		err      bool
	}{
		{"10%", Step{true, true, 10.0, 0}, false},
		{"2.5%", Step{true, true, 2.5, 0}, false},
		{"5", Step{true, false, 0.0, 5}, false},
		{"0", Step{true, false, 0.0, 0}, false},
		{"invalid%", Step{}, true},
		{"invalid", Step{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var step Step
			err := step.Set(tt.input)
			if (err != nil) != tt.err {
				t.Errorf("Set() error = %v, expected error = %v", err, tt.err)
			}
			if err == nil && step != tt.expected {
				t.Errorf("Set() = %v, expected %v", step, tt.expected)
			}
		})
	}
}

func TestStep_String(t *testing.T) {
	tests := []struct {
		step     Step
		expected string
	}{
		{Step{true, true, 10.0, 0}, "10.000000%"},
		{Step{true, true, 25.0, 0}, "25.000000%"},
		{Step{true, false, 0.0, 5}, "5"},
		{Step{true, false, 0.0, 0}, "0"},
		{Step{false, false, 0.0, 0}, "1"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.step.String(); got != tt.expected {
				t.Errorf("String() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestToNSperAdd(t *testing.T) {
	tests := []struct {
		measurements []float64
		addsPerRound uint32
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
		initSize             uint32
		setSize              uint32
		targetAddsPerRound   uint32
		totalAddsPerConfig   uint32
		expectedNumOfSets    uint32
		expectedAddsPerRound uint32
		expectedRounds       uint32
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
		initSize := rand.Uint32() % (1 << 20)
		setSize := rand.Uint32() % (1 << 20)
		targetAddsPerRound := rand.Uint32() % (1 << 20)
		totalAddsPerConfig := rand.Uint32() % (1 << 20)
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
	tests := []struct {
		name             string
		p                programParametrization
		expectedDuration time.Duration
	}{
		{
			name: "Basic case with integer step",
			p: programParametrization{
				step: Step{
					isSet:       true,
					isPercent:   false,
					integerStep: 1,
				},
				targetAddsPerRound: 10,
				toSetSize:          10,
				fromSetSize:        1,
				expRuntimePerAdd:   2.0,
				secondsPerConfig:   1.0,
			},
			expectedDuration: time.Duration(123_200_000_000),
		},
		{
			name: "Large set size with percent step",
			p: programParametrization{
				step: Step{
					isSet:     true,
					isPercent: true,
					percent:   0.1,
				},
				targetAddsPerRound: 20,
				toSetSize:          20,
				fromSetSize:        5,
				expRuntimePerAdd:   8,
				secondsPerConfig:   1.0,
			},
			expectedDuration: time.Duration(17_937_920_000_000),
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

func TestStepType(t *testing.T) {
	step := Step{}
	expected := "string"
	if result := step.Type(); result != expected {
		t.Errorf("Type() = %v, want %v", result, expected)
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
		name     string
		setup    benchmarkSetup
		expected []uint32
	}{
		{
			name: "Normal range",
			setup: benchmarkSetup{
				programParametrization: programParametrization{
					fromSetSize: 1,
					toSetSize:   5,
				},
			},
			expected: []uint32{1, 2, 3, 4, 5},
		},
		{
			name: "Single value range",
			setup: benchmarkSetup{
				programParametrization: programParametrization{
					fromSetSize: 3,
					toSetSize:   3,
				},
			},
			expected: []uint32{3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := 0
			for setSize := range tt.setup.setSizes() {
				assert.True(t, tt.expected[i] == setSize, "%v!=%v", tt.expected[i], setSize)
				i++
			}
		})
	}
}

func TestInitSizes(t *testing.T) {
	tests := []struct {
		name     string
		setup    benchmarkSetup
		setSize  uint32
		expected []uint32
	}{
		{
			name: "From 10, +10%",
			setup: benchmarkSetup{
				programParametrization: programParametrization{
					step: Step{
						isPercent: true,
						percent:   10.0,
					},
					toSetSize: 10,
				},
			},
			setSize:  10,
			expected: []uint32{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		},
		{
			name: "From 100, +25%",
			setup: benchmarkSetup{
				programParametrization: programParametrization{
					step: Step{
						isPercent: true,
						percent:   25.0,
					},
					toSetSize: 100,
				},
			},
			setSize:  100,
			expected: []uint32{100, 125, 150, 175, 200},
		},
		{
			name: "From 10, +30%",
			setup: benchmarkSetup{
				programParametrization: programParametrization{
					step: Step{
						isPercent: true,
						percent:   30.0,
					},
					toSetSize: 10,
				},
			},
			setSize:  10,
			expected: []uint32{10, 13, 16, 19, 22},
		},
		{
			name: "From 2 to 6, +1",
			setup: benchmarkSetup{
				programParametrization: programParametrization{
					step: Step{
						isPercent:   false,
						integerStep: 1,
					},
					toSetSize: 6,
				},
			},
			setSize:  2,
			expected: []uint32{2, 3, 4, 5, 6, 7, 8},
		},
		{
			name: "From 33 to 40, +10",
			setup: benchmarkSetup{
				programParametrization: programParametrization{
					step: Step{
						isPercent:   false,
						integerStep: 10,
					},
					toSetSize: 40,
				},
			},
			setSize:  33,
			expected: []uint32{33, 43, 53, 63, 73},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := 0
			for initSize := range tt.setup.initSizes(tt.setSize) {
				assert.True(t, tt.expected[i] == initSize, "%v!=%v", tt.expected[i], initSize)
				i++
			}
		})
	}
}
