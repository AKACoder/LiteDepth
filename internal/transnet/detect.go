package transnet

import (
	"fmt"

	ort "github.com/yalue/onnxruntime_go"

	"litedepth/internal/depth"
	"litedepth/internal/ffmpegx"
	"litedepth/internal/i18n"
	"litedepth/internal/job"
)

const (
	frameW    = 48
	frameH    = 27
	window    = 100
	stride    = 50
	edge      = 25
	Threshold = 0.5
	// borderline is the lowest single-frame score that can still be a cut.
	// It stays below Threshold. A score in between becomes a cut only when the
	// luma histogram also jumps, so a lower global threshold is not required.
	borderline = 0.3
	histBins   = 32
	// histCut is the chi-square distance between 32-bin luma histograms.
	// On 48x27 frames, shot changes in the borderline band land near 0.28,
	// while motion inside a shot stays under 0.05 on the peak's other side.
	histCut = 0.15
)

// Detect finds hard cuts in one source window. Frames are read at 48x27 with
// the same seek the depth extract uses, so a later depth slice lines up.
// Every score above Threshold starts a new shot, including a short one.
func Detect(video string, startS, lengthS float64) ([]job.FrameSpan, error) {
	raw, n, err := ffmpegx.ExtractRaw(video, startS, lengthS, frameW, frameH)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, fmt.Errorf("%s", i18n.C().ErrNoShots)
	}
	scores, err := predict(raw, n)
	if err != nil {
		return nil, err
	}
	return SpansFromScores(applyBorderline(raw, scores)), nil
}

// applyBorderline keeps every score above Threshold. A local peak between
// borderline and Threshold becomes a cut only when the picture's luma
// histogram changes. Scores below borderline are left alone.
func applyBorderline(raw []byte, scores []float32) []float32 {
	out := append([]float32(nil), scores...)
	n := len(out)
	for i, s := range out {
		if i == 0 || s > Threshold || s < borderline || !localPeak(out, i) {
			continue
		}
		if histJump(raw, n, i) < histCut {
			continue
		}
		out[i] = 1
	}
	return out
}

func localPeak(scores []float32, i int) bool {
	s := scores[i]
	if i > 0 && s <= scores[i-1] {
		return false
	}
	if i+1 < len(scores) && s <= scores[i+1] {
		return false
	}
	return true
}

func histJump(raw []byte, n, i int) float64 {
	jump := 0.0
	if i > 0 {
		jump = chi2(lumaHist(raw, i-1), lumaHist(raw, i))
	}
	if i+1 < n {
		if next := chi2(lumaHist(raw, i), lumaHist(raw, i+1)); next > jump {
			jump = next
		}
	}
	return jump
}

func lumaHist(raw []byte, index int) []float64 {
	bins := make([]float64, histBins)
	base := index * frameW * frameH * 3
	for p := 0; p < frameW*frameH; p++ {
		o := base + p*3
		y := 0.299*float64(raw[o]) + 0.587*float64(raw[o+1]) + 0.114*float64(raw[o+2])
		b := int(y) / 8
		if b >= histBins {
			b = histBins - 1
		}
		if b < 0 {
			b = 0
		}
		bins[b]++
	}
	var total float64
	for _, v := range bins {
		total += v
	}
	if total == 0 {
		return bins
	}
	for i := range bins {
		bins[i] /= total
	}
	return bins
}

func chi2(a, b []float64) float64 {
	var sum float64
	for i := range a {
		if a[i]+b[i] == 0 {
			continue
		}
		d := a[i] - b[i]
		sum += d * d / (a[i] + b[i])
	}
	return sum
}

// SpansFromScores turns per-frame cut probabilities into half-open frame spans.
// A run of frames above Threshold stays in the shot before the following frame.
func SpansFromScores(scores []float32) []job.FrameSpan {
	n := len(scores)
	if n == 0 {
		return nil
	}
	flag := make([]bool, n)
	for i, s := range scores {
		flag[i] = s > Threshold
	}
	type pair struct{ start, end int }
	var scenes []pair
	prev := false
	start := 0
	lastOn := false
	for i, on := range flag {
		if prev && !on {
			start = i
		}
		if !prev && on && i != 0 {
			scenes = append(scenes, pair{start, i})
		}
		prev = on
		lastOn = on
	}
	if !lastOn {
		scenes = append(scenes, pair{start, n - 1})
	}
	if len(scenes) == 0 {
		return []job.FrameSpan{{Start: 0, End: n}}
	}
	spans := make([]job.FrameSpan, len(scenes))
	for i, sc := range scenes {
		end := n
		if i+1 < len(scenes) {
			end = scenes[i+1].start
		}
		spans[i] = job.FrameSpan{Start: sc.start, End: end}
	}
	return spans
}

func predict(raw []byte, n int) ([]float32, error) {
	if err := depth.EnsureRuntime(); err != nil {
		return nil, err
	}
	inputs, outputs, err := ort.GetInputOutputInfoWithONNXData(model)
	if err != nil {
		return nil, err
	}
	if len(inputs) != 1 || len(outputs) < 1 {
		return nil, fmt.Errorf("%s", i18n.C().ErrDepthIOCount)
	}
	outNames := []string{outputs[0].Name}
	session, err := ort.NewDynamicAdvancedSessionWithONNXData(model, []string{inputs[0].Name}, outNames, nil)
	if err != nil {
		return nil, err
	}
	defer session.Destroy()

	padStart := edge
	rem := n % stride
	if rem == 0 {
		rem = stride
	}
	padEnd := edge + stride - rem
	padded := make([]int, 0, padStart+n+padEnd)
	for i := 0; i < padStart; i++ {
		padded = append(padded, 0)
	}
	for i := 0; i < n; i++ {
		padded = append(padded, i)
	}
	for i := 0; i < padEnd; i++ {
		padded = append(padded, n-1)
	}

	frameBytes := frameW * frameH * 3
	var scores []float32
	for ptr := 0; ptr+window <= len(padded); ptr += stride {
		data := make([]float32, window*frameBytes)
		for f := 0; f < window; f++ {
			src := padded[ptr+f] * frameBytes
			dst := f * frameBytes
			for b := 0; b < frameBytes; b++ {
				data[dst+b] = float32(raw[src+b])
			}
		}
		tensor, err := ort.NewTensor(ort.NewShape(1, window, frameH, frameW, 3), data)
		if err != nil {
			return nil, err
		}
		got := []ort.Value{nil}
		err = session.Run([]ort.Value{tensor}, got)
		tensor.Destroy()
		if err != nil {
			return nil, err
		}
		out, ok := got[0].(*ort.Tensor[float32])
		if !ok {
			if got[0] != nil {
				got[0].Destroy()
			}
			return nil, fmt.Errorf("%s", i18n.C().ErrDepthType)
		}
		pred := out.GetData()
		if len(pred) < window {
			out.Destroy()
			return nil, fmt.Errorf("%s", i18n.C().ErrDepthSize)
		}
		scores = append(scores, pred[edge:edge+stride]...)
		out.Destroy()
	}
	if len(scores) < n {
		return nil, fmt.Errorf("%s", i18n.C().ErrNoShots)
	}
	return scores[:n], nil
}
