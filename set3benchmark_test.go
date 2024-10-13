package main

import (
	"math"
	"reflect"
	"testing"

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
		setSizeTo uint32
		step      Step
		expected  uint32
	}{
		{33, Step{true, true, 10.0, 0}, 11},   // expect: 0%, 10%, 20%, ..., 90%, 100%
		{19, Step{true, true, 25.0, 0}, 5},    // expect: 0%, 25%, 50%, 75%, 100%
		{19, Step{true, true, 30.0, 0}, 5},    // expect: 0%, 30%, 60%, 90%, 120%
		{234, Step{true, false, 0.0, 1}, 235}, // expect: 0, 1, 2, ..., 233, 234
		{33, Step{true, false, 0.0, 10}, 5},   // expect: 0, 10, 20, 30, 40
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := getNumberOfSteps(tt.setSizeTo, tt.step)
			if result != tt.expected {
				t.Errorf("got %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestColumnHeadings(t *testing.T) {
	tests := []struct {
		setSizeTo uint32
		step      Step
		expected  []string
	}{
		{33, Step{true, true, 10.0, 0}, []string{"+0.00%% ", "+10.00%% ", "+20.00%% ", "+30.00%% ", "+40.00%% ", "+50.00%% ", "+60.00%% ", "+70.00%% ", "+80.00%% ", "+90.00%% ", "+100.00%% "}},
		{19, Step{true, true, 25.0, 0}, []string{"+0.00%% ", "+25.00%% ", "+50.00%% ", "+75.00%% ", "+100.00%% "}},
		{19, Step{true, true, 30.0, 0}, []string{"+0.00%% ", "+30.00%% ", "+60.00%% ", "+90.00%% ", "+120.00%% "}},
		{4, Step{true, false, 0.0, 1}, []string{"+0 ", "+1 ", "+2 ", "+3 ", "+4 "}},
		{33, Step{true, false, 0.0, 10}, []string{"+0 ", "+10 ", "+20 ", "+30 ", "+40 "}},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := columnHeadings(tt.setSizeTo, tt.step)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestInitSizeValues(t *testing.T) {
	tests := []struct {
		currentSetSize uint32
		setSizeTo      uint32
		step           Step
		expected       []uint32
	}{
		{10, 11, Step{true, true, 10.0, 0}, []uint32{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}},
		{100, 1000, Step{true, true, 25.0, 0}, []uint32{100, 125, 150, 175, 200}},
		{10, 1, Step{true, true, 30.0, 0}, []uint32{10, 13, 16, 19, 22}},
		{2, 6, Step{true, false, 0.0, 1}, []uint32{2, 3, 4, 5, 6, 7, 8}},
		{33, 40, Step{true, false, 0.0, 10}, []uint32{33, 43, 53, 63, 73}},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := initSizeValues(tt.currentSetSize, tt.setSizeTo, tt.step)
			if !reflect.DeepEqual(result, tt.expected) {
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
