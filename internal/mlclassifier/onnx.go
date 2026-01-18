package mlclassifier

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// ONNXRuntime provides an interface to ONNX model inference
// This is a wrapper that can work with or without the actual ONNX runtime
type ONNXRuntime struct {
	mu        sync.RWMutex
	models    map[string]*ONNXModel
	logger    *slog.Logger
	available bool
}

// ONNXModel represents a loaded ONNX model
type ONNXModel struct {
	Name        string
	Path        string
	InputNames  []string
	OutputNames []string
	InputShapes map[string][]int64
	Loaded      bool
}

// ONNXConfig contains configuration for the ONNX runtime
type ONNXConfig struct {
	ModelsDir      string
	UseGPU         bool
	NumThreads     int
	LogLevel       int
	EnableProfiling bool
}

// InferenceInput represents input to an ONNX model
type InferenceInput struct {
	Name  string
	Data  interface{} // []float32 or []int32 or []int64
	Shape []int64
}

// InferenceOutput represents output from an ONNX model
type InferenceOutput struct {
	Name  string
	Data  []float32
	Shape []int64
}

// NewONNXRuntime creates a new ONNX runtime instance
func NewONNXRuntime(config ONNXConfig, logger *slog.Logger) (*ONNXRuntime, error) {
	runtime := &ONNXRuntime{
		models:    make(map[string]*ONNXModel),
		logger:    logger,
		available: false, // Set to true when actual ONNX runtime is available
	}

	// Note: In a production implementation, this would initialize the ONNX runtime
	// using github.com/yalue/onnxruntime_go or similar
	//
	// Example initialization (requires CGO and onnxruntime shared library):
	// err := ort.SetSharedLibraryPath("/path/to/onnxruntime.so")
	// err = ort.InitializeEnvironment()

	logger.Info("ONNX runtime initialized",
		"available", runtime.available,
		"models_dir", config.ModelsDir)

	return runtime, nil
}

// LoadModel loads an ONNX model from disk
func (r *ONNXRuntime) LoadModel(name, path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.models[name]; exists {
		return fmt.Errorf("model %s already loaded", name)
	}

	model := &ONNXModel{
		Name:        name,
		Path:        path,
		InputNames:  []string{"input_ids", "attention_mask"},
		OutputNames: []string{"logits"},
		InputShapes: map[string][]int64{
			"input_ids":      {1, 512},
			"attention_mask": {1, 512},
		},
		Loaded: true,
	}

	// Note: In production, this would actually load the ONNX model
	// session, err := ort.NewAdvancedSession(path, inputNames, outputNames, options)

	r.models[name] = model

	r.logger.Info("loaded ONNX model",
		"name", name,
		"path", path)

	return nil
}

// UnloadModel unloads an ONNX model
func (r *ONNXRuntime) UnloadModel(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.models[name]; !exists {
		return fmt.Errorf("model %s not loaded", name)
	}

	delete(r.models, name)

	r.logger.Info("unloaded ONNX model", "name", name)

	return nil
}

// RunInference runs inference on a loaded model
func (r *ONNXRuntime) RunInference(ctx context.Context, modelName string, inputs []InferenceInput) ([]InferenceOutput, error) {
	r.mu.RLock()
	model, exists := r.models[modelName]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("model %s not loaded", modelName)
	}

	if !model.Loaded {
		return nil, fmt.Errorf("model %s not properly loaded", modelName)
	}

	// Note: In production, this would run actual ONNX inference
	// outputs, err := session.Run(inputs)

	// For now, return simulated outputs
	outputs := make([]InferenceOutput, len(model.OutputNames))
	for i, name := range model.OutputNames {
		// Simulate output shape based on model type
		var shape []int64
		var data []float32

		switch modelName {
		case "ner":
			// NER model: [batch, seq_len, num_labels]
			shape = []int64{1, 512, 9} // 9 entity types
			data = make([]float32, 512*9)
		case "doc_classifier":
			// Document classifier: [batch, num_classes]
			shape = []int64{1, 6} // 6 document types
			data = make([]float32, 6)
		default:
			shape = []int64{1, 512}
			data = make([]float32, 512)
		}

		outputs[i] = InferenceOutput{
			Name:  name,
			Data:  data,
			Shape: shape,
		}
	}

	return outputs, nil
}

// IsAvailable returns whether the ONNX runtime is available
func (r *ONNXRuntime) IsAvailable() bool {
	return r.available
}

// ListLoadedModels returns the names of all loaded models
func (r *ONNXRuntime) ListLoadedModels() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.models))
	for name := range r.models {
		names = append(names, name)
	}
	return names
}

// GetModelInfo returns information about a loaded model
func (r *ONNXRuntime) GetModelInfo(name string) (*ONNXModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	model, exists := r.models[name]
	if !exists {
		return nil, fmt.Errorf("model %s not loaded", name)
	}

	return model, nil
}

// Close shuts down the ONNX runtime
func (r *ONNXRuntime) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.models = make(map[string]*ONNXModel)
	r.available = false

	r.logger.Info("ONNX runtime closed")

	return nil
}

// Softmax applies softmax to logits
func Softmax(logits []float32) []float32 {
	if len(logits) == 0 {
		return logits
	}

	// Find max for numerical stability
	max := logits[0]
	for _, v := range logits[1:] {
		if v > max {
			max = v
		}
	}

	// Compute exp and sum
	expSum := float32(0)
	result := make([]float32, len(logits))
	for i, v := range logits {
		result[i] = float32exp(v - max)
		expSum += result[i]
	}

	// Normalize
	for i := range result {
		result[i] /= expSum
	}

	return result
}

// Argmax returns the index of the maximum value
func Argmax(values []float32) int {
	if len(values) == 0 {
		return -1
	}

	maxIdx := 0
	maxVal := values[0]
	for i, v := range values[1:] {
		if v > maxVal {
			maxVal = v
			maxIdx = i + 1
		}
	}
	return maxIdx
}

// TopK returns the indices of the top k values
func TopK(values []float32, k int) []int {
	if k > len(values) {
		k = len(values)
	}

	type indexedValue struct {
		index int
		value float32
	}

	indexed := make([]indexedValue, len(values))
	for i, v := range values {
		indexed[i] = indexedValue{i, v}
	}

	// Partial sort to find top k
	for i := 0; i < k; i++ {
		maxIdx := i
		for j := i + 1; j < len(indexed); j++ {
			if indexed[j].value > indexed[maxIdx].value {
				maxIdx = j
			}
		}
		indexed[i], indexed[maxIdx] = indexed[maxIdx], indexed[i]
	}

	result := make([]int, k)
	for i := 0; i < k; i++ {
		result[i] = indexed[i].index
	}
	return result
}

// float32exp is a fast approximation of exp for float32
func float32exp(x float32) float32 {
	// Clamp to avoid overflow
	if x < -87 {
		return 0
	}
	if x > 88 {
		return 3.402823e+38 // Max float32
	}

	// Use Taylor series approximation for small x
	if x >= -0.5 && x <= 0.5 {
		// exp(x) ≈ 1 + x + x²/2 + x³/6 + x⁴/24
		x2 := x * x
		x3 := x2 * x
		x4 := x3 * x
		return 1 + x + x2/2 + x3/6 + x4/24
	}

	// For larger x, use range reduction
	// exp(x) = exp(n*ln(2) + r) = 2^n * exp(r)
	ln2 := float32(0.693147180559945)
	n := int(x/ln2 + 0.5)
	r := x - float32(n)*ln2

	// exp(r) using Taylor series
	r2 := r * r
	r3 := r2 * r
	r4 := r3 * r
	expr := 1 + r + r2/2 + r3/6 + r4/24

	// 2^n * exp(r)
	return ldexp(expr, n)
}

// ldexp computes x * 2^n
func ldexp(x float32, n int) float32 {
	if n == 0 {
		return x
	}
	if n > 0 {
		for i := 0; i < n; i++ {
			x *= 2
		}
	} else {
		for i := 0; i < -n; i++ {
			x /= 2
		}
	}
	return x
}

// Int32ToFloat32 converts int32 slice to float32 slice
func Int32ToFloat32(input []int32) []float32 {
	output := make([]float32, len(input))
	for i, v := range input {
		output[i] = float32(v)
	}
	return output
}

// Int64ToFloat32 converts int64 slice to float32 slice
func Int64ToFloat32(input []int64) []float32 {
	output := make([]float32, len(input))
	for i, v := range input {
		output[i] = float32(v)
	}
	return output
}

// Float32ToInt64 converts float32 slice to int64 slice
func Float32ToInt64(input []float32) []int64 {
	output := make([]int64, len(input))
	for i, v := range input {
		output[i] = int64(v)
	}
	return output
}
