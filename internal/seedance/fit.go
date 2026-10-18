package seedance

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Reference-video limits for Seedance 2.5. Edit tasks are not handled here;
// those require each clip to be at least 4 seconds.
const (
	MinDuration = 2.0
	MaxDuration = 30.0
	MaxClips    = 10
	MinAspect   = 0.4
	MaxAspect   = 2.5
	MinEdge     = 300
	MaxEdge     = 6000
	MinPixels   = 407696
	MaxPixels   = 8295044
	MinFPS      = 24.0
	MaxFPS      = 60.0
	NearFPS     = 23.5
	MaxBytes    = 200 << 20
	CRFStart    = 18
	CRFMax      = 28
)

// Shot is one hard cut on the source clock. Start and End are a half-open
// interval in seconds.
type Shot struct {
	Start float64
	End   float64
}

// Input is the clip that would be written without --seedance.
type Input struct {
	WindowStart float64
	WindowEnd   float64
	SourceEnd   float64
	FPS         float64
	Width       int
	Height      int
	LongEdge    int
	// HasShots keeps a window longer than 30 seconds intact. The upload
	// packing then brings the shot list inside the reference limits.
	HasShots bool
	Shots    []Shot
}

// Upload is one reference clip after short shots have been joined.
// From lists the 1-based hard-cut indexes it absorbed.
type Upload struct {
	Start float64
	End   float64
	From  []int
}

// Plan is the extract and the upload list. A field that matches the input
// is left alone.
type Plan struct {
	WindowStart float64
	WindowEnd   float64
	CropX       int
	CropY       int
	CropW       int
	CropH       int
	Width       int
	Height      int
	LongEdge    int
	FPS         float64
	FrameStep   int
	FrameRepeat int
	Retimestamp bool
	Slow        float64
	Upload      []Upload
	Dropped     []int
	Changes     []string
	inFPS       float64
}

// Playback is how long the fitted window plays, in seconds.
func (p Plan) Playback() float64 {
	return p.play(p.WindowEnd - p.WindowStart)
}

func (p Plan) play(source float64) float64 {
	if p.FPS <= 0 || source <= 0 {
		return 0
	}
	step := p.FrameStep
	if step < 1 {
		step = 1
	}
	repeat := p.FrameRepeat
	if repeat < 1 {
		repeat = 1
	}
	src := p.inFPS
	if src <= 0 {
		src = p.FPS
	}
	return source * src * float64(repeat) / float64(step) / p.FPS
}

// Error is a fit failure the pipeline maps to a translated message.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrLimits = Error("limits")
	ErrShots  = Error("shots")
)

// Fit plans one reference export. It does not read a file or run a model.
func Fit(in Input) (Plan, error) {
	if in.FPS <= 0 || in.Width < 2 || in.Height < 2 || in.LongEdge < 2 {
		return Plan{}, ErrLimits
	}
	winStart, winEnd := in.WindowStart, in.WindowEnd
	if winEnd < winStart {
		winEnd = winStart
	}
	var changes []string
	length := winEnd - winStart
	if length < MinDuration {
		nextStart, nextEnd := extend(winStart, winEnd, in.SourceEnd, MinDuration)
		if nextEnd-nextStart > length+1e-9 {
			changes = append(changes, fmt.Sprintf("window %s -> %s (extended)", sec(length), sec(nextEnd-nextStart)))
			winStart, winEnd = nextStart, nextEnd
			length = winEnd - winStart
		}
	}
	slow := 1.0
	if length < MinDuration {
		slow = MinDuration / length
		changes = append(changes, fmt.Sprintf("speed x%s", trimFloat(slow)))
	}
	if !in.HasShots && length > MaxDuration {
		winEnd = winStart + MaxDuration
		changes = append(changes, fmt.Sprintf("window %s -> %s (trimmed)", sec(length), sec(MaxDuration)))
		length = MaxDuration
	}

	cropX, cropY, cropW, cropH := cropToAspect(in.Width, in.Height)
	if cropW != in.Width || cropH != in.Height {
		changes = append(changes, fmt.Sprintf("crop %.3f -> %.3f", float64(in.Width)/float64(in.Height), float64(cropW)/float64(cropH)))
	}
	outW, outH, edge, err := fitEdge(cropW, cropH, in.LongEdge)
	if err != nil {
		return Plan{}, err
	}
	if edge != in.LongEdge {
		changes = append(changes, fmt.Sprintf("long edge %d -> %d", in.LongEdge, edge))
	}

	rate := in.FPS / slow
	outFPS, step, repeat, stamp := fitFPS(rate)
	if math.Abs(outFPS-in.FPS) > 1e-6 || step != 1 || repeat != 1 {
		changes = append(changes, fmt.Sprintf("fps %s -> %s", fmtFPS(in.FPS), fmtFPS(outFPS)))
	}

	plan := Plan{
		WindowStart: winStart,
		WindowEnd:   winEnd,
		CropX:       cropX,
		CropY:       cropY,
		CropW:       cropW,
		CropH:       cropH,
		Width:       outW,
		Height:      outH,
		LongEdge:    edge,
		FPS:         outFPS,
		FrameStep:   step,
		FrameRepeat: repeat,
		Retimestamp: stamp,
		Slow:        slow,
		inFPS:       in.FPS,
	}
	if len(in.Shots) > 0 {
		upload, dropped, shotChanges, err := packShots(in.Shots, plan)
		if err != nil {
			return Plan{}, err
		}
		plan.Upload = upload
		plan.Dropped = dropped
		changes = append(changes, shotChanges...)
	}
	plan.Changes = changes
	return plan, nil
}

type piece struct {
	start float64
	end   float64
	ids   []int
}

func packShots(shots []Shot, plan Plan) ([]Upload, []int, []string, error) {
	pieces := make([]piece, 0, len(shots))
	for i, shot := range shots {
		if shot.End <= shot.Start {
			continue
		}
		pieces = append(pieces, piece{start: shot.Start, end: shot.End, ids: []int{i + 1}})
	}
	if len(pieces) == 0 {
		return nil, nil, nil, ErrShots
	}
	before := len(pieces)
	pieces = mergeShort(pieces, plan)
	var notes []string
	if len(pieces) != before {
		notes = append(notes, fmt.Sprintf("shots %d -> %d", before, len(pieces)))
	}
	capped := false
	for i := range pieces {
		play := plan.play(pieces[i].end - pieces[i].start)
		if play > MaxDuration+1e-6 {
			pieces[i].end = pieces[i].start + plan.sourceFor(MaxDuration)
			capped = true
		}
	}
	if capped {
		notes = append(notes, "shot capped at 30s")
	}
	var dropped []int
	if len(pieces) > MaxClips {
		for _, p := range pieces[MaxClips:] {
			dropped = append(dropped, p.ids...)
		}
		pieces = pieces[:MaxClips]
		notes = append(notes, "dropped shots "+joinInts(dropped))
	}
	total := 0.0
	for _, p := range pieces {
		total += plan.play(p.end - p.start)
	}
	if err := trimTotal(pieces, plan, total); err != nil {
		return nil, nil, nil, err
	}
	after := 0.0
	for _, p := range pieces {
		after += plan.play(p.end - p.start)
	}
	if after+1e-6 < total {
		notes = append(notes, fmt.Sprintf("shots trimmed to %s", sec(after)))
	}
	upload := make([]Upload, len(pieces))
	for i, p := range pieces {
		upload[i] = Upload{Start: p.start, End: p.end, From: append([]int(nil), p.ids...)}
	}
	return upload, dropped, notes, nil
}

func mergeShort(pieces []piece, plan Plan) []piece {
	for {
		idx := -1
		for i, p := range pieces {
			if plan.play(p.end-p.start) < MinDuration-1e-9 {
				idx = i
				break
			}
		}
		if idx < 0 || len(pieces) < 2 {
			return pieces
		}
		if idx == len(pieces)-1 {
			pieces[idx-1].end = pieces[idx].end
			pieces[idx-1].ids = append(pieces[idx-1].ids, pieces[idx].ids...)
			pieces = pieces[:idx]
			continue
		}
		pieces[idx+1].start = pieces[idx].start
		pieces[idx+1].ids = append(append([]int{}, pieces[idx].ids...), pieces[idx+1].ids...)
		pieces = append(pieces[:idx], pieces[idx+1:]...)
	}
}

func trimTotal(pieces []piece, plan Plan, total float64) error {
	const eps = 1e-6
	for n := 0; total > MaxDuration+eps && n < 1000; n++ {
		excess := total - MaxDuration
		maxPlay := 0.0
		for _, p := range pieces {
			play := plan.play(p.end - p.start)
			if play > maxPlay {
				maxPlay = play
			}
		}
		if maxPlay <= MinDuration+eps {
			return ErrShots
		}
		floor := MinDuration
		for _, p := range pieces {
			play := plan.play(p.end - p.start)
			if play < maxPlay-eps && play > floor {
				floor = play
			}
		}
		var peers []int
		can := 0.0
		for i, p := range pieces {
			play := plan.play(p.end - p.start)
			if play >= maxPlay-eps {
				peers = append(peers, i)
				can += play - floor
			}
		}
		if len(peers) == 0 || can <= eps {
			return ErrShots
		}
		give := excess
		if give > can {
			give = can
		}
		each := give / float64(len(peers))
		for _, i := range peers {
			pieces[i].end -= plan.sourceFor(each)
		}
		total -= give
	}
	if total > MaxDuration+1e-4 {
		return ErrShots
	}
	return nil
}

func (p Plan) sourceFor(play float64) float64 {
	step := p.FrameStep
	if step < 1 {
		step = 1
	}
	repeat := p.FrameRepeat
	if repeat < 1 {
		repeat = 1
	}
	src := p.inFPS
	if src <= 0 || p.FPS <= 0 {
		return play
	}
	return play * p.FPS * float64(step) / float64(repeat) / src
}

func extend(start, end, source, want float64) (float64, float64) {
	need := want - (end - start)
	if need <= 0 {
		return start, end
	}
	if source > end {
		add := source - end
		if add > need {
			add = need
		}
		end += add
		need -= add
	}
	if need > 0 && start > 0 {
		sub := start
		if sub > need {
			sub = need
		}
		start -= sub
	}
	return start, end
}

func cropToAspect(w, h int) (x, y, cw, ch int) {
	if h <= 0 || w <= 0 {
		return 0, 0, w, h
	}
	aspect := float64(w) / float64(h)
	cw, ch = w, h
	switch {
	case aspect > MaxAspect:
		cw = int(math.Round(float64(h) * MaxAspect))
	case aspect < MinAspect:
		ch = int(math.Round(float64(w) / MinAspect))
	}
	if cw > w {
		cw = w
	}
	if ch > h {
		ch = h
	}
	if cw < 2 {
		cw = 2
	}
	if ch < 2 {
		ch = 2
	}
	return (w - cw) / 2, (h - ch) / 2, cw, ch
}

func fitEdge(cropW, cropH, requested int) (w, h, edge int, err error) {
	if requested%2 != 0 {
		requested++
	}
	w, h = sizeAt(cropW, cropH, requested)
	if legal(w, h) {
		return w, h, requested, nil
	}
	if tooBig(w, h) && !tooSmall(w, h) {
		for e := requested - 2; e >= 2; e -= 2 {
			w, h = sizeAt(cropW, cropH, e)
			if legal(w, h) {
				return w, h, e, nil
			}
		}
		return 0, 0, 0, ErrLimits
	}
	for e := requested; e <= MaxEdge; e += 2 {
		w, h = sizeAt(cropW, cropH, e)
		if legal(w, h) {
			return w, h, e, nil
		}
		if tooBig(w, h) && !tooSmall(w, h) {
			break
		}
	}
	return 0, 0, 0, ErrLimits
}

// SmallerEdge is the next shorter even long edge that still meets the limits.
// ok is false when none exists.
func SmallerEdge(cropW, cropH, edge int) (w, h, next int, ok bool) {
	for e := edge - 2; e >= 2; e -= 2 {
		w, h = sizeAt(cropW, cropH, e)
		if legal(w, h) {
			return w, h, e, true
		}
	}
	return 0, 0, 0, false
}

func sizeAt(cropW, cropH, edge int) (int, int) {
	if cropW >= cropH {
		h := nearestEven(float64(edge) * float64(cropH) / float64(cropW))
		if h < 2 {
			h = 2
		}
		return edge, h
	}
	w := nearestEven(float64(edge) * float64(cropW) / float64(cropH))
	if w < 2 {
		w = 2
	}
	return w, edge
}

func nearestEven(x float64) int {
	down := int(math.Floor(x))
	if down%2 != 0 {
		down--
	}
	if down < 2 {
		down = 2
	}
	up := down + 2
	if math.Abs(x-float64(down)) <= math.Abs(float64(up)-x) {
		return down
	}
	return up
}

func legal(w, h int) bool {
	if w < MinEdge || h < MinEdge || w > MaxEdge || h > MaxEdge {
		return false
	}
	pixels := w * h
	if pixels < MinPixels || pixels > MaxPixels {
		return false
	}
	aspect := float64(w) / float64(h)
	return aspect >= MinAspect-1e-9 && aspect <= MaxAspect+1e-9
}

func tooSmall(w, h int) bool {
	return w < MinEdge || h < MinEdge || w*h < MinPixels
}

func tooBig(w, h int) bool {
	return w > MaxEdge || h > MaxEdge || w*h > MaxPixels
}

func fitFPS(fps float64) (out float64, step, repeat int, stamp bool) {
	if fps >= MinFPS && fps <= MaxFPS {
		return fps, 1, 1, false
	}
	if fps >= NearFPS && fps < MinFPS {
		return MinFPS, 1, 1, true
	}
	if fps > MaxFPS {
		step = int(math.Ceil(fps / MaxFPS))
		if step < 2 {
			step = 2
		}
		return fps / float64(step), step, 1, false
	}
	repeat = int(math.Ceil(MinFPS / fps))
	if repeat < 2 {
		repeat = 2
	}
	return fps * float64(repeat), 1, repeat, false
}

// EncodeAction is the next attempt after a finished file is measured.
type EncodeAction int

const (
	EncodeKeep EncodeAction = iota
	EncodeCRF
	EncodeDropFPS
	EncodeShrink
	EncodeFail
)

// EncodeStep is one file-size decision. Inference is not involved until Shrink.
type EncodeStep struct {
	Action EncodeAction
	CRF    int
	FPS    float64
	Step   int
}

// NextEncode decides how to react to a file size. CRF rises before the frame
// rate drops, and the frame rate drops before the picture is made smaller.
func NextEncode(size int64, crf int, fps float64, dropped, shrunk bool) EncodeStep {
	if size <= MaxBytes {
		return EncodeStep{Action: EncodeKeep, CRF: crf, FPS: fps}
	}
	if crf < CRFMax {
		next := crf + 2
		if next > CRFMax {
			next = CRFMax
		}
		return EncodeStep{Action: EncodeCRF, CRF: next, FPS: fps}
	}
	if !dropped {
		if out, step, ok := dropFPS(fps); ok {
			return EncodeStep{Action: EncodeDropFPS, CRF: crf, FPS: out, Step: step}
		}
	}
	if !shrunk {
		return EncodeStep{Action: EncodeShrink, CRF: crf, FPS: fps}
	}
	return EncodeStep{Action: EncodeFail, CRF: crf, FPS: fps}
}

func dropFPS(fps float64) (float64, int, bool) {
	if fps <= MinFPS {
		return 0, 0, false
	}
	for step := 2; step < 1000; step++ {
		out := fps / float64(step)
		if out < MinFPS {
			return 0, 0, false
		}
		if out <= MaxFPS {
			return out, step, true
		}
	}
	return 0, 0, false
}

func sec(v float64) string {
	s := strconv.FormatFloat(v, 'f', 1, 64)
	return s + "s"
}

func fmtFPS(v float64) string {
	if math.Abs(v-math.Round(v)) < 1e-3 {
		return strconv.FormatFloat(math.Round(v), 'f', 0, 64)
	}
	return strconv.FormatFloat(v, 'f', 3, 64)
}

func trimFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func joinInts(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return strings.Join(parts, ",")
}
