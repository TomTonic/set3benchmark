package misc

import (
	"fmt"
	"io"
	"math"
	"strings"
)

type Histo struct {
	from     float64
	to       float64
	min      float64
	max      float64
	totalsum float64
	step     float64
	elements uint64
	Width    int
	Unit     string
	slots    []uint64
}

func MakeHisto(from, to float64, steps int) *Histo {
	result := &Histo{
		from:     from,
		to:       to,
		slots:    make([]uint64, steps+1),
		min:      math.MaxFloat64,
		max:      -math.MaxFloat64,
		totalsum: 0,
		step:     (to - from) / float64(steps),
		elements: 0,
		Width:    40,
		Unit:     "",
	}
	return result
}

func (h *Histo) Add(value float64) {
	if value < h.min {
		h.min = value
	}
	if value > h.max {
		h.max = value
	}
	h.totalsum += value
	h.elements++

	// slot   0         1         2         3         4         5
	// <------|----v----|----v----|----v----|----v----|----v----|------>
	// value  0    1    2    3    4    5    6    7    8    9    10
	//       <-0.99
	//             1.0->

	relVal := value - h.from
	slot := int(math.Round(relVal / h.step))

	if slot <= 0 {
		h.slots[0]++
		return
	}
	if slot >= len(h.slots)-1 {
		h.slots[len(h.slots)-1]++
		return
	}
	h.slots[slot]++
}

func (h *Histo) GetMin() float64 {
	return h.min
}

func (h *Histo) GetMax() float64 {
	return h.max
}

func (h *Histo) GetAvg() float64 {
	return h.totalsum / float64(h.elements)
}

func (h *Histo) GetP(x float64) float64 {
	if x <= 0.0 {
		return h.from
	}
	if x >= 1.0 {
		return h.to
	}
	idx := 0
	count := uint64(h.slots[idx])
	searchlimit := uint64(math.Round(x * float64(h.elements)))
	for count < searchlimit {
		idx++
		count += uint64(h.slots[idx])
	}
	result := h.from + (float64(idx) * h.step)
	return result
}

func (h *Histo) GetP25() float64 {
	return h.GetP(0.25)
}

func (h *Histo) GetP50() float64 {
	return h.GetP(0.50)
}

func (h *Histo) GetP75() float64 {
	return h.GetP(0.75)
}

func (h *Histo) GetP90() float64 {
	return h.GetP(0.90)
}

func (h *Histo) GetNumberOfSlots() int {
	return len(h.slots)
}

func (h *Histo) GetCount(slotIndex int) uint64 {
	if slotIndex < 0 {
		slotIndex = 0
	}
	if slotIndex > len(h.slots)-1 {
		slotIndex = len(h.slots) - 1
	}
	return h.slots[slotIndex]
}

func (h *Histo) GetRangeString(slotIndex int) string {
	if slotIndex < 0 {
		slotIndex = 0
	}
	if slotIndex > len(h.slots)-1 {
		slotIndex = len(h.slots) - 1
	}
	if slotIndex == 0 {
		val := h.from
		val += h.step / 2
		s := fmt.Sprintf("<%.3f", val)
		return s
	}
	if slotIndex == len(h.slots)-1 {
		val := h.to
		val -= h.step / 2
		s := fmt.Sprintf("≥%.3f", val)
		return s
	}
	val := h.from
	val += float64(slotIndex) * h.step
	v1 := val - h.step/2
	v2 := val + h.step/2
	s := fmt.Sprintf("≥%.3f,<%.3f", v1, v2)
	return s
}

func (h *Histo) GetAnchorPoint(slotIndex int) float64 {
	if slotIndex <= 0 {
		return h.from
	}
	if slotIndex >= len(h.slots)-1 {
		return h.to
	}
	val := h.from
	val += float64(slotIndex) * h.step
	return val
}

func (h *Histo) WriteStatsTo(w io.Writer) (int64, error) {
	if h.elements == 0 {
		// no values yet
		n, err := fmt.Fprintf(w, "  avg -----%v;  min -----%v;  p25 -----%v;  p50 -----%v;  p75 -----%v;  p90 -----%v;  max -----%v;\n", h.Unit, h.Unit, h.Unit, h.Unit, h.Unit, h.Unit, h.Unit)
		return int64(n), err
	}
	n, err := fmt.Fprintf(w, "  avg %.3f%v;  min %.3f%v;  p25 %.3f%v;  p50 %.3f%v;  p75 %.3f%v;  p90 %.3f%v;  max %.3f%v;\n",
		h.GetAvg(), h.Unit,
		h.GetMin(), h.Unit,
		h.GetP25(), h.Unit,
		h.GetP50(), h.Unit,
		h.GetP75(), h.Unit,
		h.GetP90(), h.Unit,
		h.GetMax(), h.Unit,
	)
	return int64(n), err
}

// WriteTo writes formatted statistics and histogram to w.
func (hist *Histo) WriteTo(w io.Writer) (int64, error) {
	written, err := hist.WriteStatsTo(w)
	if err != nil {
		return written, err
	}

	maxCountLength := 3
	for _, count := range hist.slots {
		x := (int)(math.Ceil(math.Log10(float64(count + 1))))
		if x > maxCountLength {
			maxCountLength = x
		}
	}

	maxCount := uint64(0)
	for _, count := range hist.slots {
		if count > maxCount {
			maxCount = count
		}
	}

	relWidth := make([]float64, len(hist.slots))
	for i, count := range hist.slots {
		relWidth[i] = float64(count) / float64(maxCount)
	}

	var n int
	for idx, count := range hist.slots {
		anchor := hist.GetAnchorPoint(idx)
		if idx == 0 {
			n, err = fmt.Fprintf(w, " %8.3f%v-[%[3]*[4]v] ", anchor, hist.Unit, maxCountLength, count)
		} else if idx == len(hist.slots)-1 {
			n, err = fmt.Fprintf(w, " %8.3f%v+[%[3]*[4]v] ", anchor, hist.Unit, maxCountLength, count)
		} else {
			n, err = fmt.Fprintf(w, " %8.3f%v [%[3]*[4]v] ", anchor, hist.Unit, maxCountLength, count)
		}
		written += int64(n)
		if err != nil {
			return written, err
		}

		width := float64(hist.Width) * relWidth[idx]
		frac := width - math.Trunc(width)

		n, err = io.WriteString(w, strings.Repeat("█", int(width)))
		written += int64(n)
		if err != nil {
			return written, err
		}

		if frac >= 0.5 {
			n, err = io.WriteString(w, `▌`)
			written += int64(n)
			if err != nil {
				return written, err
			}
		}

		n, err = fmt.Fprintf(w, "\n")
		written += int64(n)
		if err != nil {
			return written, err
		}
	}
	return written, nil
}

// String returns a string representation of the histogram.
func (hist *Histo) String() string {
	var buffer strings.Builder
	_, _ = hist.WriteTo(&buffer)
	return buffer.String()
}
