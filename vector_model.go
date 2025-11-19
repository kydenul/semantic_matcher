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

	// Fallback statistics
	fallbackAttempts  int64 // Number of character-level fallback attempts
	fallbackSuccesses int64 // Number of successful fallback operations
	fallbackFailures  int64 // Number of failed fallback operations
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
// If the word is not found (OOV), attempts character-level fallback
func (vm *vectorModel) GetVector(word string) ([]float32, bool) {
	vm.mtx.Lock()
	defer vm.mtx.Unlock()

	vm.totalLookups++

	// First, try direct lookup from vocabulary
	vector, exists := vm.vectors[word]
	if exists {
		vm.hitLookups++
		// Return a copy to prevent external modification
		result := make([]float32, len(vector))
		copy(result, vector)
		return result, true
	}

	// Word not found - mark as OOV
	vm.oovLookups++

	// Attempt character-level fallback for OOV words
	vm.fallbackAttempts++
	fallbackVector, success := vm.characterLevelFallback(word)
	if success {
		// Return a copy to prevent external modification
		result := make([]float32, len(fallbackVector))
		copy(result, fallbackVector)
		return result, true
	}

	// Fallback failed
	return nil, false
}

// GetAverageVector computes mean pooling for multiple words
// Returns the averaged vector and a boolean indicating if any words were found
// For OOV words, automatically attempts character-level fallback
func (vm *vectorModel) GetAverageVector(words []string) ([]float32, bool) {
	if len(words) == 0 {
		return nil, false
	}

	vm.mtx.Lock()
	defer vm.mtx.Unlock()

	var sum []float32
	validWords := 0

	for _, word := range words {
		vm.totalLookups++

		// First, try direct lookup from vocabulary
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
			// Word not found - mark as OOV
			vm.oovLookups++

			// Attempt character-level fallback for OOV words
			vm.fallbackAttempts++
			fallbackVector, success := vm.characterLevelFallback(word)
			if success {
				if sum == nil {
					// Initialize sum vector with the dimension
					sum = make([]float32, vm.dimension)
				}

				// Add fallback vector components to sum
				for i, val := range fallbackVector {
					sum[i] += val
				}
				validWords++
			}
			// If fallback fails, the word is simply skipped (no vector to add)
		}
	}

	// Return false if no valid words were found (all OOV and all fallbacks failed)
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

// AddVectorsBatch adds multiple word-vector pairs in a single lock operation
// This is much more efficient than calling AddVector repeatedly
// Returns the number of vectors successfully added
func (vm *vectorModel) AddVectorsBatch(words []string, vectors [][]float32) int {
	if len(words) != len(vectors) {
		return 0
	}

	vm.mtx.Lock()
	defer vm.mtx.Unlock()

	addedCount := 0
	for i := range words {
		// Validate vector dimension
		if len(vectors[i]) != vm.dimension {
			continue // Skip vectors with wrong dimension
		}

		// Use string interning to reduce memory usage
		internedWord := vm.internString(words[i])

		// Store a copy to prevent external modification
		vectorCopy := make([]float32, len(vectors[i]))
		copy(vectorCopy, vectors[i])
		vm.vectors[internedWord] = vectorCopy

		// Update memory usage estimate
		vm.updateMemoryUsage(internedWord, vectorCopy)
		addedCount++
	}

	return addedCount
}

// PreallocateCapacity preallocates map capacity to reduce rehashing during loading
// This should be called before loading large vector files
func (vm *vectorModel) PreallocateCapacity(expectedSize int) {
	vm.mtx.Lock()
	defer vm.mtx.Unlock()

	// Only preallocate if current capacity is smaller
	if len(vm.vectors) < expectedSize {
		newVectors := make(map[string][]float32, expectedSize)
		newStringIntern := make(map[string]string, expectedSize)

		// Copy existing data
		for k, v := range vm.vectors {
			newVectors[k] = v
		}
		for k, v := range vm.stringIntern {
			newStringIntern[k] = v
		}

		vm.vectors = newVectors
		vm.stringIntern = newStringIntern
	}
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
func (vm *vectorModel) GetLookupStats() (totalLookups, oovLookups, hitLookups, fallbackAttempts, fallbackSuccesses, fallbackFailures int64) {
	vm.mtx.RLock()
	defer vm.mtx.RUnlock()

	return vm.totalLookups, vm.oovLookups, vm.hitLookups, vm.fallbackAttempts, vm.fallbackSuccesses, vm.fallbackFailures
}

// GetFallbackSuccessRate returns the success rate of character-level fallback operations
// Returns a value between 0.0 and 1.0, or 0.0 if no fallback attempts have been made
func (vm *vectorModel) GetFallbackSuccessRate() float64 {
	vm.mtx.RLock()
	defer vm.mtx.RUnlock()

	if vm.fallbackAttempts == 0 {
		return 0.0
	}

	return float64(vm.fallbackSuccesses) / float64(vm.fallbackAttempts)
}

// ResetStats resets all statistics counters
func (vm *vectorModel) ResetStats() {
	vm.mtx.Lock()
	defer vm.mtx.Unlock()

	vm.totalLookups = 0
	vm.oovLookups = 0
	vm.hitLookups = 0
	vm.fallbackAttempts = 0
	vm.fallbackSuccesses = 0
	vm.fallbackFailures = 0
}

// characterLevelFallback attempts to generate a vector for an OOV word by splitting it into characters
// and averaging the vectors of characters that exist in the vocabulary.
// This method is called with the lock already held.
func (vm *vectorModel) characterLevelFallback(word string) ([]float32, bool) {
	// Convert string to runes for proper Unicode character handling
	runes := []rune(word)

	// Single character words should not trigger fallback (already failed in main lookup)
	if len(runes) <= 1 {
		vm.fallbackFailures++
		return nil, false
	}

	// Collect character vectors
	var sum []float32
	validChars := 0

	for _, r := range runes {
		char := string(r)
		if charVec, exists := vm.vectors[char]; exists {
			if sum == nil {
				// Initialize sum vector with the correct dimension
				sum = make([]float32, vm.dimension)
			}
			// Add character vector to sum
			for i, val := range charVec {
				sum[i] += val
			}
			validChars++
		}
	}

	// If no characters have vectors, fallback fails
	if validChars == 0 {
		vm.fallbackFailures++
		return nil, false
	}

	// Compute average by dividing sum by number of valid characters
	result := make([]float32, vm.dimension)
	for i := range sum {
		result[i] = sum[i] / float32(validChars)
	}

	vm.fallbackSuccesses++
	return result, true
}
