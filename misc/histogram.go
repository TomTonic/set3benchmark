/*
Initial code from https://github.com/loov/hrtime

The MIT License (MIT)

Copyright (c) 2018 Egon Elbre <egonelbre@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package misc

import (
	"fmt"
	"io"
	"math"
	"slices"
	"strings"
	"time"
)

// HistogramOptions is configuration.
type HistogramOptions struct {
	BinCount int
	// NiceRange will try to round the bucket sizes to have a nicer output.
	NiceRange bool
	// Clamp values to either percentile or to a specific ns value.
	ClampMinimum    float64
	ClampMaximum    float64
	ClampPercentile float64
}

var DefaultOptions = HistogramOptions{
	BinCount:        10,
	NiceRange:       true,
	ClampMinimum:    0,
	ClampMaximum:    0,
	ClampPercentile: 0.999,
}

// Histogram is a binned historgram with different statistics.
type Histogram struct {
	Minimum float64
	Average float64
	Maximum float64

	P25, P50, P75, P90, P99, P999, P9999 float64

	Bins []HistogramBin

	// for pretty printing
	Width int
}

// HistogramBin is a single bin in histogram
type HistogramBin struct {
	Start    float64
	Count    int
	Width    float64
	andAbove bool
}

// NewDurationHistogram creates a histogram from time.Duration-s.
func NewDurationHistogram(durations []time.Duration, opts *HistogramOptions) *Histogram {
	nanos := make([]float64, len(durations))
	for i, d := range durations {
		nanos[i] = float64(d.Nanoseconds())
	}
	return NewHistogram(nanos, opts)
}

// NewHistogram creates a new histogram from the specified nanosecond values.
func NewHistogram(nanoseconds []float64, opts *HistogramOptions) *Histogram {
	if opts.BinCount <= 0 {
		panic("binCount must be larger than 0")
	}

	hist := &Histogram{}
	hist.Width = 40
	hist.Bins = make([]HistogramBin, opts.BinCount)
	if len(nanoseconds) == 0 {
		return hist
	}

	//nanoseconds = append(nanoseconds[:0:0], nanoseconds...)
	slices.Sort(nanoseconds)
	AssertPositive(nanoseconds, "C")

	hist.Minimum = nanoseconds[0]
	hist.Maximum = nanoseconds[len(nanoseconds)-1]

	if hist.Minimum < 0 {
		panic("lwg3zp39qbhap9")
	}

	hist.Average = float64(0)
	for _, x := range nanoseconds {
		hist.Average += x
	}
	hist.Average /= float64(len(nanoseconds))

	p := func(p float64) float64 {
		i := int(math.Round(p * float64(len(nanoseconds))))
		if i < 0 {
			i = 0
		}
		if i >= len(nanoseconds) {
			i = len(nanoseconds) - 1
		}
		return nanoseconds[i]
	}

	hist.P25, hist.P50, hist.P75, hist.P90, hist.P99, hist.P999, hist.P9999 = p(0.25), p(0.50), p(0.75), p(0.90), p(0.99), p(0.999), p(0.9999)

	clampMinimum := hist.Minimum
	if opts.ClampMinimum > 0 {
		clampMinimum = opts.ClampMinimum
	}

	clampMaximum := hist.Maximum
	if opts.ClampPercentile > 0 {
		clampMaximum = p(opts.ClampPercentile)
	}
	if opts.ClampMaximum > 0 {
		clampMaximum = opts.ClampMaximum
	}

	if clampMaximum < hist.Minimum {
		panic("oAWEGNLAGHq34g")
	}

	var minimum, spacing float64

	if opts.NiceRange {
		minimum, spacing = calculateNiceSteps(clampMinimum, clampMaximum, opts.BinCount)
	} else {
		minimum, spacing = calculateSteps(clampMinimum, clampMaximum, opts.BinCount)
	}

	if minimum < 0.0 {
		fmt.Printf("min: %f; max: %f; clampMinimum: %f; clampMaximum: %f", hist.Minimum, hist.Maximum, clampMinimum, clampMaximum)
		panic("phfgap4zpa2t89")
	}

	if spacing < 0.0 {
		panic("jhap9w3t8p92t2")
	}

	for i := range hist.Bins {
		hist.Bins[i].Start = spacing*float64(i) + clampMinimum
		if hist.Bins[i].Start < 0.0 {
			panic("aph4pahgh")
		}
	}
	hist.Bins[0].Start = clampMinimum

	for _, x := range nanoseconds {
		k := int(float64(x-clampMinimum) / spacing)
		if k < 0 {
			k = 0
		}
		if k >= opts.BinCount {
			k = opts.BinCount - 1
			hist.Bins[k].andAbove = true
		}
		hist.Bins[k].Count++
	}

	maxBin := 0
	for _, bin := range hist.Bins {
		if bin.Count > maxBin {
			maxBin = bin.Count
		}
	}

	for k := range hist.Bins {
		bin := &hist.Bins[k]
		bin.Width = float64(bin.Count) / float64(maxBin)
	}

	return hist
}

// Divide divides histogram by number of repetitions for the tests.
func (hist *Histogram) Divide(n int) {
	hist.Minimum /= float64(n)
	hist.Average /= float64(n)
	hist.Maximum /= float64(n)

	hist.P25 /= float64(n)
	hist.P50 /= float64(n)
	hist.P75 /= float64(n)
	hist.P90 /= float64(n)
	hist.P99 /= float64(n)
	hist.P999 /= float64(n)
	hist.P9999 /= float64(n)

	for i := range hist.Bins {
		hist.Bins[i].Start /= float64(n)
	}
}

// WriteStatsTo writes formatted statistics to w.
func (hist *Histogram) WriteStatsTo(w io.Writer) (int64, error) {
	n, err := fmt.Fprintf(w, "  min %8.3fns;  p25 %8.3fns;  p50 %8.3fns;  p75  %8.3fns;  max   %8.3fns;\n  avg %8.3fns;  p90 %8.3fns;  p99 %8.3fns;  p999 %8.3fns;  p9999 %8.3fns;\n",
		hist.Minimum,
		hist.P25,
		hist.P50,
		hist.P75,
		hist.Maximum,

		hist.Average,
		hist.P90,
		hist.P99,
		hist.P999,
		hist.P9999,
	)
	return int64(n), err
}

// WriteTo writes formatted statistics and histogram to w.
func (hist *Histogram) WriteTo(w io.Writer) (int64, error) {
	written, err := hist.WriteStatsTo(w)
	if err != nil {
		return written, err
	}

	// TODO: use consistently single unit instead of multiple
	maxCountLength := 3
	for i := range hist.Bins {
		x := (int)(math.Ceil(math.Log10(float64(hist.Bins[i].Count + 1))))
		if x > maxCountLength {
			maxCountLength = x
		}
	}

	var n int
	for _, bin := range hist.Bins {
		if bin.andAbove {
			n, err = fmt.Fprintf(w, " %8.3fns+[%[2]*[3]v] ", bin.Start, maxCountLength, bin.Count)
		} else {
			n, err = fmt.Fprintf(w, " %8.3fns [%[2]*[3]v] ", bin.Start, maxCountLength, bin.Count)
		}
		written += int64(n)
		if err != nil {
			return written, err
		}

		width := float64(hist.Width) * bin.Width
		frac := width - math.Trunc(width)

		n, err = io.WriteString(w, strings.Repeat("█", int(width)))
		written += int64(n)
		if err != nil {
			return written, err
		}

		if frac > 0.5 {
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

// StringStats returns a string representation of the histogram stats.
func (hist *Histogram) StringStats() string {
	var buffer strings.Builder
	_, _ = hist.WriteStatsTo(&buffer)
	return buffer.String()
}

// String returns a string representation of the histogram.
func (hist *Histogram) String() string {
	var buffer strings.Builder
	_, _ = hist.WriteTo(&buffer)
	return buffer.String()
}

func calculateSteps(min, max float64, bincount int) (minimum, spacing float64) {
	minimum = min
	spacing = (max - min) / float64(bincount)
	return minimum, spacing
}

func calculateNiceSteps(min, max float64, bincount int) (minimum, spacing float64) {
	span := niceNumber(max-min, false)
	spacing = niceNumber(span/float64(bincount-1), true)
	minimum = math.Floor(min/spacing) * spacing
	return minimum, spacing
}

func niceNumber(span float64, round bool) float64 {
	exp := math.Floor(math.Log10(span))
	frac := span / math.Pow(10, exp)

	var nice float64
	if round {
		switch {
		case frac < 1.5:
			nice = 1
		case frac < 3:
			nice = 2
		case frac < 7:
			nice = 5
		default:
			nice = 10
		}
	} else {
		switch {
		case frac <= 1:
			nice = 1
		case frac <= 2:
			nice = 2
		case frac <= 5:
			nice = 5
		default:
			nice = 10
		}
	}

	return nice * math.Pow(10, exp)
}

/*
func truncate(v float64, digits int) float64 {
	if digits == 0 || v == 0 {
		return 0
	}

	scale := math.Pow(10, math.Floor(math.Log10(v))+1-float64(digits))
	return scale * math.Trunc(v/scale)
}

func round(v float64, digits int) float64 {
	if digits == 0 || v == 0 {
		return 0
	}

	scale := math.Pow(10, math.Floor(math.Log10(v))+1-float64(digits))
	return scale * math.Round(v/scale)
}
*/
