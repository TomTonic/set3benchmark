package misc

import (
	"bytes"
	"math"
	"testing"
)

func TestMakeHisto(t *testing.T) {
	h := MakeHisto(0, 10, 5)
	if h == nil {
		t.Fatal("MakeHisto returned nil")
	}
	if len(h.slots) != 6 {
		t.Fatalf("expected 6 slots, got %d", len(h.slots))
	}
	if h.step != 2 {
		t.Fatalf("expected step to be 2, got %f", h.step)
	}
}

func TestAdd(t *testing.T) {
	h := MakeHisto(0, 10, 5)
	h.Add(2)
	h.Add(8)
	if h.min != 2 {
		t.Fatalf("expected min to be 3, got %f", h.min)
	}
	if h.max != 8 {
		t.Fatalf("expected max to be 7, got %f", h.max)
	}
	if h.totalsum != 10 {
		t.Fatalf("expected totalsum to be 10, got %f", h.totalsum)
	}
	if h.elements != 2 {
		t.Fatalf("expected elements to be 2, got %d", h.elements)
	}
	if h.slots[1] != 1 || h.slots[4] != 1 {
		t.Fatal("Add method did not update slots correctly")
	}
}

func TestAdd2(t *testing.T) {
	h := MakeHisto(0, 10, 5)
	h.Add(-1)
	h.Add(11)
	if h.min != -1 {
		t.Fatalf("expected min to be 3, got %f", h.min)
	}
	if h.max != 11 {
		t.Fatalf("expected max to be 7, got %f", h.max)
	}
	if h.totalsum != 10 {
		t.Fatalf("expected totalsum to be 10, got %f", h.totalsum)
	}
	if h.elements != 2 {
		t.Fatalf("expected elements to be 2, got %d", h.elements)
	}
	if h.slots[0] != 1 || h.slots[5] != 1 {
		t.Fatal("Add method did not update slots correctly")
	}
}

func TestAdd3(t *testing.T) {
	h := MakeHisto(1, 11, 5)
	h.Add(3)
	h.Add(9)
	if h.min != 3 {
		t.Fatalf("expected min to be 3, got %f", h.min)
	}
	if h.max != 9 {
		t.Fatalf("expected max to be 7, got %f", h.max)
	}
	if h.totalsum != 12 {
		t.Fatalf("expected totalsum to be 10, got %f", h.totalsum)
	}
	if h.elements != 2 {
		t.Fatalf("expected elements to be 2, got %d", h.elements)
	}
	if h.slots[1] != 1 || h.slots[4] != 1 {
		t.Fatal("Add method did not update slots correctly")
	}
}

func TestAdd4(t *testing.T) {
	// slot   0         1         2         3         4         5
	// <------|----v----|----v----|----v----|----v----|----v----|------>
	// value  0    1    2    3    4    5    6    7    8    9    10
	//       <-0.99
	//             1.0->
	h := MakeHisto(0, 10, 5)
	h.Add(0.99999)
	h.Add(1.0)
	if h.min != 0.99999 {
		t.Fatalf("expected min to be 3, got %f", h.min)
	}
	if h.max != 1.0 {
		t.Fatalf("expected max to be 7, got %f", h.max)
	}
	if h.totalsum != 1.99999 {
		t.Fatalf("expected totalsum to be 10, got %f", h.totalsum)
	}
	if h.elements != 2 {
		t.Fatalf("expected elements to be 2, got %d", h.elements)
	}
	if h.slots[0] != 1 || h.slots[1] != 1 {
		t.Fatal("Add method did not update slots correctly")
	}
}

func TestGetMin(t *testing.T) {
	h := MakeHisto(0, 10, 5)
	h.Add(3)
	if min := h.GetMin(); min != 3 {
		t.Fatalf("expected min to be 0, got %f", min)
	}
}

func TestGetMax(t *testing.T) {
	h := MakeHisto(0, 10, 5)
	h.Add(3)
	if max := h.GetMax(); max != 3 {
		t.Fatalf("expected max to be 3, got %f", max)
	}
}

func TestGetAvg(t *testing.T) {
	h := MakeHisto(0, 10, 5)
	h.Add(3)
	h.Add(7)
	if avg := h.GetAvg(); avg != 5 {
		t.Fatalf("expected avg to be 5, got %f", avg)
	}
}

func TestGetP(t *testing.T) {
	// slot   0         1         2         3         4         5
	// <------|----v----|----v----|----v----|----v----|----v----|------>
	// value  0    1    2    3    4    5    6    7    8    9    10
	//       <-0.99
	//             1.0->
	h := MakeHisto(0, 10, 5)
	h.Add(1)
	h.Add(3)
	h.Add(5)
	h.Add(7)
	if p := h.GetP(-0.1); math.Abs(p) > 1e-9 {
		t.Fatalf("expected P(-0.1) to be 5, got %f", p)
	}
	if p := h.GetP(0.5); math.Abs(p-4) > 1e-9 {
		t.Fatalf("expected P(0.5) to be 5, got %f", p)
	}
	if p := h.GetP25(); math.Abs(p-2) > 1e-9 {
		t.Fatalf("expected P25 to be 2.5, got %f", p)
	}
	if p := h.GetP50(); math.Abs(p-4) > 1e-9 {
		t.Fatalf("expected P50 to be 5, got %f", p)
	}
	if p := h.GetP75(); math.Abs(p-6) > 1e-9 {
		t.Fatalf("expected P75 to be 7.5, got %f", p)
	}
	if p := h.GetP90(); math.Abs(p-8) > 1e-9 {
		t.Fatalf("expected P90 to be 9, got %f", p)
	}
	if p := h.GetP(1.1); math.Abs(p-10) > 1e-9 {
		t.Fatalf("expected P(1.1) to be 10, got %f", p)
	}
}

func TestGetRangeString(t *testing.T) {
	h := MakeHisto(0, 10, 5)
	expectedResults := map[int]string{
		-1:   "<1.000",
		0:    "<1.000",
		1:    "≥1.000,<3.000",
		2:    "≥3.000,<5.000",
		3:    "≥5.000,<7.000",
		4:    "≥7.000,<9.000",
		5:    "≥9.000",
		1000: "≥9.000"}
	for idx, expected := range expectedResults {
		result := h.GetRangeString(idx)
		if result != expected {
			t.Errorf("GetRangeString(%d) = %s; want %s", idx, result, expected)
		}
	}
}

func TestGetAnchorPoint(t *testing.T) {
	h := MakeHisto(0, 10, 5)
	testCases := []struct {
		slotIndex int
		expected  float64
	}{
		{slotIndex: -1, expected: 0},
		{slotIndex: 0, expected: 0},
		{slotIndex: 1, expected: 2},
		{slotIndex: 2, expected: 4},
		{slotIndex: 3, expected: 6},
		{slotIndex: 4, expected: 8},
		{slotIndex: 5, expected: 10},
		{slotIndex: 1000, expected: 10},
	}
	for _, tc := range testCases {
		result := h.GetAnchorPoint(tc.slotIndex)
		if result != tc.expected {
			t.Errorf("GetAnchorPoint(%d) = %f; want %f", tc.slotIndex, result, tc.expected)
		}
	}
}

func TestWriteStatsTo(t *testing.T) {
	h := MakeHisto(0, 10, 5)
	h.Unit = "ns"
	// Add some values to the histogram
	h.Add(1)
	h.Add(2)
	h.Add(3)
	h.Add(4)
	h.Add(5)
	var buf bytes.Buffer
	expected := "  avg 3.000ns;  min 1.000ns;  p25 2.000ns;  p50 4.000ns;  p75 4.000ns;  p90 6.000ns;  max 5.000ns;\n"
	n, err := h.WriteStatsTo(&buf)
	if err != nil {
		t.Errorf("WriteStatsTo returned an error: %v", err)
	}
	if int64(buf.Len()) != n {
		t.Errorf("WriteStatsTo returned incorrect byte count: got %d, want %d", n, buf.Len())
	}
	if buf.String() != expected {
		t.Errorf("WriteStatsTo wrote unexpected output:\n  got %q\n want %q", buf.String(), expected)
	}
}

func TestWriteStatsToEmptyHistogram(t *testing.T) {
	h := MakeHisto(0, 10, 5)
	h.Unit = "ns"
	var buf bytes.Buffer
	expected := "  avg -----ns;  min -----ns;  p25 -----ns;  p50 -----ns;  p75 -----ns;  p90 -----ns;  max -----ns;\n"
	n, err := h.WriteStatsTo(&buf)
	if err != nil {
		t.Errorf("WriteStatsTo returned an error: %v", err)
	}
	if int64(buf.Len()) != n {
		t.Errorf("WriteStatsTo returned incorrect byte count: got %d, want %d", n, buf.Len())
	}
	if buf.String() != expected {
		t.Errorf("WriteStatsTo wrote unexpected output: got %q, want %q", buf.String(), expected)
	}
}

func TestWriteTo(t *testing.T) {
	h := MakeHisto(0, 10, 5)
	h.Unit = "ns"
	h.Width = 2
	// Add some values to the histogram
	h.Add(1.5)
	h.Add(2.0)
	h.Add(4)
	h.Add(9.5)
	h.Add(9.5)
	h.Add(9.5)
	h.Add(9.5)
	var buf bytes.Buffer
	expected := "  avg 6.500ns;  min 1.500ns;  p25 2.000ns;  p50 10.000ns;  p75 10.000ns;  p90 10.000ns;  max 9.500ns;\n    0.000ns-[  0] \n    2.000ns [  2] █\n    4.000ns [  1] ▌\n    6.000ns [  0] \n    8.000ns [  0] \n   10.000ns+[  4] ██\n"
	n, err := h.WriteTo(&buf)
	if err != nil {
		t.Errorf("WriteStatsTo returned an error: %v", err)
	}
	if int64(buf.Len()) != n {
		t.Errorf("WriteStatsTo returned incorrect byte count: got %d, want %d", n, buf.Len())
	}
	if buf.String() != expected {
		t.Errorf("WriteStatsTo wrote unexpected output:\n  got %q\n want %q", buf.String(), expected)
	}
}
