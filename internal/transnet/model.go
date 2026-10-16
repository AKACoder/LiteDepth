package transnet

import _ "embed"

// TransNet V2 shot detector. The ONNX file is the community export of the
// MIT-licensed weights, and its outputs are already probabilities.
//
//go:embed data/transnetv2.onnx
var model []byte
