package semanticmatcher

import (
	"sync"
	"unsafe"
)

// vectorModel implements the VectorModel interface with hash map-based vector storage
type vectorModel struct {
	vectors      map[string][]float32 // Hash map for O(1) vector lookup
	dimension    int                  // Vector dimension
	mtx          sync.RWMutex         // Read-write mutex for thread-safe concurrent access
	stringIntern map[string]string    // String interning for memory optimization
	memoryUsage  int64                // Cached memory usage in bytes

	// Statistics tracking
	totalLookups int64 // Total number of vector lookups
	oovLookups   int64 // Number of OOV (out-of-vocabulary) lookups
	hitLookups   int64 // Number of successful lookups
}

// NewVectorModel creates a new VectorModel instance
func NewVectorModel(dimension int) VectorModel {
	return &vectorModel{
		vectors:      make(map[string][]float32),
		dimension:    dimension,
		stringIntern: make(map[string]string),
		memoryUsage:  0,
	}
}

// GetVector retrieves vector for a single word
// Returns the vector and a boolean indicating if the word was found
func (vm *vectorModel) GetVector(word string) ([]float32, bool) {
	vm.mtx.Lock()
	defer vm.mtx.Unlock()

	vm.totalLookups++

	vector, exists := vm.vectors[word]
	if !exists {
		vm.oovLookups++
		return nil, false
	}

	vm.hitLookups++

	// Return a copy to prevent external modification
	result := make([]float32, len(vector))
	copy(result, vector)
	return result, true
}

// GetAverageVector computes mean pooling for multiple words
// Returns the averaged vector and a boolean indicating if any words were found
func (vm *vectorModel) GetAverageVector(words []string) ([]float32, bool) {
	if len(words) == 0 {
		return nil, false
	}

	vm.mtx.Lock()
	defer vm.mtx.Unlock()

	var sum []float32
	validWords := 0
	oovCount := 0

	for _, word := range words {
		vm.totalLookups++
		if vector, exists := vm.vectors[word]; exists {
			vm.hitLookups++
			if sum == nil {
				// Initialize sum vector with the dimension
				sum = make([]float32, vm.dimension)
			}

			// Add vector components to sum
			for i, val := range vector {
				sum[i] += val
			}
			validWords++
		} else {
			vm.oovLookups++
			oovCount++
		}
	}

	// Return false if no valid words were found (all OOV)
	if validWords == 0 {
		return nil, false
	}

	// Compute mean by dividing by number of valid words
	result := make([]float32, vm.dimension)
	for i := range sum {
		result[i] = sum[i] / float32(validWords)
	}

	return result, true
}

// Dimension returns the vector dimension
func (vm *vectorModel) Dimension() int {
	vm.mtx.RLock()
	defer vm.mtx.RUnlock()
	return vm.dimension
}

// VocabularySize returns total number of words in model
func (vm *vectorModel) VocabularySize() int {
	vm.mtx.RLock()
	defer vm.mtx.RUnlock()
	return len(vm.vectors)
}

// AddVector adds a word-vector pair to the model (used by EmbeddingLoader)
// This method is not part of the public interface but needed for loading
func (vm *vectorModel) AddVector(word string, vector []float32) {
	vm.mtx.Lock()
	defer vm.mtx.Unlock()

	// Validate vector dimension
	if len(vector) != vm.dimension {
		return // Silently ignore vectors with wrong dimension
	}

	// Use string interning to reduce memory usage for duplicate strings
	internedWord := vm.internString(word)

	// Store a copy to prevent external modification
	vectorCopy := make([]float32, len(vector))
	copy(vectorCopy, vector)
	vm.vectors[internedWord] = vectorCopy

	// Update memory usage estimate
	vm.updateMemoryUsage(internedWord, vectorCopy)
}

// internString implements string interning to reduce memory usage
// Returns the interned string from the pool or adds it if new
func (vm *vectorModel) internString(s string) string {
	if interned, exists := vm.stringIntern[s]; exists {
		return interned
	}
	vm.stringIntern[s] = s
	return s
}

// updateMemoryUsage updates the cached memory usage estimate
// This is called with the lock already held
func (vm *vectorModel) updateMemoryUsage(word string, vector []float32) {
	// Calculate memory for this entry:
	// - string header: 16 bytes (pointer + length)
	// - string data: len(word) bytes
	// - slice header: 24 bytes (pointer + len + cap)
	// - slice data: len(vector) * 4 bytes (float32)
	// - map overhead: approximately 48 bytes per entry

	stringSize := int64(unsafe.Sizeof(word)) + int64(len(word))
	vectorSize := int64(unsafe.Sizeof(vector)) + int64(len(vector)*4)
	mapOverhead := int64(48)

	vm.memoryUsage += stringSize + vectorSize + mapOverhead
}

// MemoryUsage returns estimated memory usage in bytes
func (vm *vectorModel) MemoryUsage() int64 {
	vm.mtx.RLock()
	defer vm.mtx.RUnlock()
	return vm.memoryUsage
}

// GetOOVRate returns the rate of out-of-vocabulary lookups
// Returns a value between 0.0 and 1.0
func (vm *vectorModel) GetOOVRate() float64 {
	vm.mtx.RLock()
	defer vm.mtx.RUnlock()

	if vm.totalLookups == 0 {
		return 0.0
	}

	return float64(vm.oovLookups) / float64(vm.totalLookups)
}

// GetVectorHitRate returns the rate of successful vector lookups
// Returns a value between 0.0 and 1.0
func (vm *vectorModel) GetVectorHitRate() float64 {
	vm.mtx.RLock()
	defer vm.mtx.RUnlock()

	if vm.totalLookups == 0 {
		return 0.0
	}

	return float64(vm.hitLookups) / float64(vm.totalLookups)
}

// GetLookupStats returns detailed lookup statistics
func (vm *vectorModel) GetLookupStats() (totalLookups, oovLookups, hitLookups int64) {
	vm.mtx.RLock()
	defer vm.mtx.RUnlock()

	return vm.totalLookups, vm.oovLookups, vm.hitLookups
}

// ResetStats resets all statistics counters
func (vm *vectorModel) ResetStats() {
	vm.mtx.Lock()
	defer vm.mtx.Unlock()

	vm.totalLookups = 0
	vm.oovLookups = 0
	vm.hitLookups = 0
}
