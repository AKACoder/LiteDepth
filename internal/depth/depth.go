package depth

import (
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"math"
	"os"
	"sync"

	ort "github.com/yalue/onnxruntime_go"

	"litedepth/internal/i18n"
)

const inputSize = 518

var (
	envOnce sync.Once
	envErr  error

	meanRGB = []float32{0.485, 0.456, 0.406}
	stdRGB  = []float32{0.229, 0.224, 0.225}
)

// Model renders a depth video from one clip.
type Model struct {
	session *ort.DynamicAdvancedSession
	look    *readableLook
	mu      sync.Mutex
}

func New() (*Model, error) {
	envOnce.Do(func() {
		lib, err := ensureORTLibrary()
		if err != nil {
			envErr = err
			return
		}
		ort.SetSharedLibraryPath(lib)
		envErr = ort.InitializeEnvironment()
	})
	if envErr != nil {
		return nil, envErr
	}

	inputs, outputs, err := ort.GetInputOutputInfoWithONNXData(embeddedDepth)
	if err != nil {
		return nil, err
	}
	if len(inputs) != 1 || len(outputs) != 1 {
		return nil, fmt.Errorf("%s", i18n.C().ErrDepthIOCount)
	}

	session, err := ort.NewDynamicAdvancedSessionWithONNXData(
		embeddedDepth,
		[]string{inputs[0].Name},
		[]string{outputs[0].Name},
		nil,
	)
	if err != nil {
		return nil, err
	}
	return &Model{
		session: session,
		look:    newReadableLook(),
	}, nil
}

func (d *Model) Close() error {
	if d == nil {
		return nil
	}
	if d.session == nil {
		return nil
	}
	return d.session.Destroy()
}

// Reset drops the previous frame so the next one is not held against it.
func (d *Model) Reset() {
	d.mu.Lock()
	d.look = newReadableLook()
	d.mu.Unlock()
}

func (d *Model) WritePNG(srcPath, destPath string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	src, err := loadRGB(srcPath)
	if err != nil {
		return err
	}
	pix, err := d.renderFrame(src)
	if err != nil {
		return err
	}
	rgb := grayToRGB(&image.Gray{Pix: pix, Stride: src.W, Rect: image.Rect(0, 0, src.W, src.H)})
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	if err := png.Encode(f, rgb); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func (d *Model) renderFrame(src *rgbImage) ([]uint8, error) {
	depth, dw, dh, err := d.inferDepth(src)
	if err != nil {
		return nil, err
	}
	return d.look.Apply(depth, dw, dh, src), nil
}

func (d *Model) inferDepth(src *rgbImage) ([]float32, int, int, error) {
	if needsDepthTiles(src.W, src.H) {
		return d.inferDepthTiled(src)
	}
	return d.inferRaw(src)
}

// inferRaw runs the fixed 518 model and returns its depth grid before normalization.
func (d *Model) inferRaw(src *rgbImage) ([]float32, int, int, error) {
	tensor, err := preprocess(src)
	if err != nil {
		return nil, 0, 0, err
	}
	defer tensor.Destroy()

	outputs := []ort.Value{nil}
	if err := d.session.Run([]ort.Value{tensor}, outputs); err != nil {
		return nil, 0, 0, err
	}
	outTensor, ok := outputs[0].(*ort.Tensor[float32])
	if !ok {
		if outputs[0] != nil {
			outputs[0].Destroy()
		}
		return nil, 0, 0, fmt.Errorf("%s", i18n.C().ErrDepthType)
	}
	defer outTensor.Destroy()

	outH, outW := depthMapSize(outTensor.GetShape())
	if outW <= 0 || outH <= 0 {
		return nil, 0, 0, fmt.Errorf("%s", i18n.C().ErrDepthSize)
	}
	depth := outTensor.GetData()
	n := outW * outH
	if len(depth) < n {
		return nil, 0, 0, fmt.Errorf("%s", i18n.C().ErrDepthSize)
	}
	raw := make([]float32, n)
	copy(raw, depth[:n])
	return raw, outW, outH, nil
}

type rgbImage struct {
	W, H int
	RGB  []float32 // interleaved RGB 0..1, len W*H*3
}

func loadRGB(path string) (*rgbImage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	out := &rgbImage{W: w, H: h, RGB: make([]float32, w*h*3)}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r16, g16, b16, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			i := (y*w + x) * 3
			out.RGB[i+0] = float32(r16) / 65535
			out.RGB[i+1] = float32(g16) / 65535
			out.RGB[i+2] = float32(b16) / 65535
		}
	}
	return out, nil
}

func preprocess(src *rgbImage) (*ort.Tensor[float32], error) {
	sampled := resizeRGBBilinear(src, inputSize, inputSize)
	data := make([]float32, 1*3*inputSize*inputSize)
	// Depth Anything V2: RGB + ImageNet normalization.
	for y := 0; y < inputSize; y++ {
		for x := 0; x < inputSize; x++ {
			i := (y*inputSize + x) * 3
			for c := 0; c < 3; c++ {
				v := (sampled[i+c] - meanRGB[c]) / stdRGB[c]
				data[c*inputSize*inputSize+y*inputSize+x] = v
			}
		}
	}
	return ort.NewTensor(ort.NewShape(1, 3, int64(inputSize), int64(inputSize)), data)
}

func depthMapSize(shape ort.Shape) (h, w int) {
	dims := shape
	switch len(dims) {
	case 2:
		return int(dims[0]), int(dims[1])
	case 3:
		return int(dims[1]), int(dims[2])
	case 4:
		return int(dims[2]), int(dims[3])
	default:
		return 0, 0
	}
}

func normalizeDepth(depth []float32, w, h int) []uint8 {
	if len(depth) < w*h {
		return nil
	}
	minV, maxV := depth[0], depth[0]
	for _, v := range depth[:w*h] {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	span := maxV - minV
	if span < 1e-6 {
		span = 1
	}
	out := make([]uint8, w*h)
	for i, v := range depth[:w*h] {
		n := (v - minV) / span
		if n < 0 {
			n = 0
		}
		if n > 1 {
			n = 1
		}
		out[i] = uint8(math.Round(float64(n) * 255))
	}
	return out
}

func grayToRGB(src *image.Gray) *image.RGBA {
	b := src.Bounds()
	out := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			v := src.GrayAt(x, y).Y
			i := out.PixOffset(x, y)
			out.Pix[i] = v
			out.Pix[i+1] = v
			out.Pix[i+2] = v
			out.Pix[i+3] = 255
		}
	}
	return out
}
