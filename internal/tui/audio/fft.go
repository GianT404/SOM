package audio

import "math"

type complexN struct{ re, im float64 }

func fft(x []complexN) {
	n := len(x)
	if n <= 1 {
		return
	}

	even := make([]complexN, n/2)
	odd := make([]complexN, n/2)
	for i := 0; i < n/2; i++ {
		even[i] = x[2*i]
		odd[i] = x[2*i+1]
	}
	fft(even)
	fft(odd)

	for k := 0; k < n/2; k++ {
		theta := -2 * math.Pi * float64(k) / float64(n)
		wRe, wIm := math.Cos(theta), math.Sin(theta)
		tRe := wRe*odd[k].re - wIm*odd[k].im
		tIm := wRe*odd[k].im + wIm*odd[k].re

		x[k] = complexN{even[k].re + tRe, even[k].im + tIm}
		x[k+n/2] = complexN{even[k].re - tRe, even[k].im - tIm}
	}
}

func nextPow2(n int) int {
	p := 2
	for p < n {
		p <<= 1
	}
	return p
}

func magnitudeBands(samples []float64, bands int) []float64 {
	n := nextPow2(len(samples))
	buf := make([]complexN, n)

	last := len(samples) - 1
	for i, s := range samples {
		w := 1.0
		if last > 0 {
			w = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(last))
		}
		buf[i] = complexN{s * w, 0}
	}

	fft(buf)

	half := n / 2
	mags := make([]float64, half)
	for i := 0; i < half; i++ {
		mags[i] = math.Hypot(buf[i].re, buf[i].im)
	}

	out := make([]float64, bands)
	if half < 2 {
		return out
	}

	logMin := 0.0 // log(1)
	logMax := math.Log(float64(half))
	for b := 0; b < bands; b++ {
		f0 := math.Exp(logMin + (logMax-logMin)*float64(b)/float64(bands))
		f1 := math.Exp(logMin + (logMax-logMin)*float64(b+1)/float64(bands))
		lo := int(f0)
		hi := int(f1)
		if lo < 1 {
			lo = 1
		}
		if hi <= lo {
			hi = lo + 1
		}
		if hi > half {
			hi = half
		}

		sum, count := 0.0, 0
		for i := lo; i < hi; i++ {
			sum += mags[i]
			count++
		}
		if count > 0 {
			out[b] = sum / float64(count)
		}
	}

	peak := 0.0
	for _, v := range out {
		if v > peak {
			peak = v
		}
	}
	if peak > 0 {
		for i := range out {
			out[i] /= peak
		}
	}

	return out
}
