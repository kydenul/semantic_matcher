package semanticmatcher

import (
	"fmt"
	"math"
	"sync"
	"testing"
)

func TestNewVectorModel(t *testing.T) {
	dimension := 100
	vm := NewVectorModel(dimension)

	if vm.Dimension() != dimension {
		t.Errorf("Expected dimension %d, got %d", dimension, vm.Dimension())
	}

	if vm.VocabularySize() != 0 {
		t.Errorf("Expected empty vocabulary, got size %d", vm.VocabularySize())
	}
}

func TestVectorModel_GetVector(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add test vector
	testVector := []float32{1.0, 2.0, 3.0}
	vm.AddVector("test", testVector)

	// Test existing word
	vector, exists := vm.GetVector("test")
	if !exists {
		t.Error("Expected word 'test' to exist")
	}

	if len(vector) != 3 {
		t.Errorf("Expected vector length 3, got %d", len(vector))
	}

	for i, val := range testVector {
		if vector[i] != val {
			t.Errorf("Expected vector[%d] = %f, got %f", i, val, vector[i])
		}
	}

	// Test non-existing word
	_, exists = vm.GetVector("nonexistent")
	if exists {
		t.Error("Expected word 'nonexistent' to not exist")
	}

	// Test that returned vector is a copy (modification doesn't affect original)
	vector[0] = 999.0
	originalVector, _ := vm.GetVector("test")
	if originalVector[0] == 999.0 {
		t.Error("Vector modification affected original - should return copy")
	}
}

func TestVectorModel_GetAverageVector(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add test vectors
	vm.AddVector("word1", []float32{1.0, 2.0, 3.0})
	vm.AddVector("word2", []float32{4.0, 5.0, 6.0})
	vm.AddVector("word3", []float32{7.0, 8.0, 9.0})

	// Test average of two words
	words := []string{"word1", "word2"}
	avgVector, exists := vm.GetAverageVector(words)
	if !exists {
		t.Error("Expected average vector to exist")
	}

	expected := []float32{2.5, 3.5, 4.5} // (1+4)/2, (2+5)/2, (3+6)/2
	for i, val := range expected {
		if avgVector[i] != val {
			t.Errorf("Expected avgVector[%d] = %f, got %f", i, val, avgVector[i])
		}
	}

	// Test with some OOV words
	wordsWithOOV := []string{"word1", "nonexistent", "word2"}
	avgVector, exists = vm.GetAverageVector(wordsWithOOV)
	if !exists {
		t.Error("Expected average vector to exist even with OOV words")
	}

	// Should still be average of word1 and word2
	for i, val := range expected {
		if avgVector[i] != val {
			t.Errorf("Expected avgVector[%d] = %f, got %f", i, val, avgVector[i])
		}
	}

	// Test with all OOV words
	oovWords := []string{"nonexistent1", "nonexistent2"}
	_, exists = vm.GetAverageVector(oovWords)
	if exists {
		t.Error("Expected no average vector for all OOV words")
	}

	// Test with empty word list
	_, exists = vm.GetAverageVector([]string{})
	if exists {
		t.Error("Expected no average vector for empty word list")
	}
}

func TestVectorModel_Dimension(t *testing.T) {
	dimensions := []int{50, 100, 300}

	for _, dim := range dimensions {
		vm := NewVectorModel(dim)
		if vm.Dimension() != dim {
			t.Errorf("Expected dimension %d, got %d", dim, vm.Dimension())
		}
	}
}

func TestVectorModel_VocabularySize(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	if vm.VocabularySize() != 0 {
		t.Errorf("Expected vocabulary size 0, got %d", vm.VocabularySize())
	}

	// Add vectors and check size
	vm.AddVector("word1", []float32{1.0, 2.0, 3.0})
	if vm.VocabularySize() != 1 {
		t.Errorf("Expected vocabulary size 1, got %d", vm.VocabularySize())
	}

	vm.AddVector("word2", []float32{4.0, 5.0, 6.0})
	if vm.VocabularySize() != 2 {
		t.Errorf("Expected vocabulary size 2, got %d", vm.VocabularySize())
	}
}

func TestVectorModel_AddVector(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Test adding valid vector
	testVector := []float32{1.0, 2.0, 3.0}
	vm.AddVector("test", testVector)

	if vm.VocabularySize() != 1 {
		t.Errorf("Expected vocabulary size 1, got %d", vm.VocabularySize())
	}

	// Test adding vector with wrong dimension (should be ignored)
	wrongDimVector := []float32{1.0, 2.0} // dimension 2 instead of 3
	vm.AddVector("wrong", wrongDimVector)

	if vm.VocabularySize() != 1 {
		t.Errorf("Expected vocabulary size to remain 1, got %d", vm.VocabularySize())
	}

	// Test that stored vector is a copy
	testVector[0] = 999.0
	storedVector, _ := vm.GetVector("test")
	if storedVector[0] == 999.0 {
		t.Error("Stored vector was affected by external modification - should store copy")
	}
}

func TestVectorModel_ConcurrentAccess(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add some initial vectors
	for i := range 100 {
		vm.AddVector(string(rune('a'+i)), []float32{float32(i), float32(i + 1), float32(i + 2)})
	}

	var wg sync.WaitGroup
	numGoroutines := 10

	// Test concurrent reads
	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Perform multiple read operations
			for j := range 100 {
				word := string(rune('a' + (j % 26)))
				vm.GetVector(word)
				vm.GetAverageVector([]string{word, string(rune('b' + (j % 25)))})
				vm.Dimension()
				vm.VocabularySize()
			}
		}(i)
	}

	wg.Wait()

	// Verify data integrity after concurrent access
	if vm.VocabularySize() != 100 {
		t.Errorf(
			"Expected vocabulary size 100 after concurrent access, got %d",
			vm.VocabularySize(),
		)
	}
}

// Benchmark tests for vector operations

func BenchmarkVectorModel_GetVector(b *testing.B) {
	vm := NewVectorModel(100).(*vectorModel)

	// Add test vectors
	for i := range 1000 {
		vector := make([]float32, 100)
		for j := range 100 {
			vector[j] = float32(i*100 + j)
		}
		vm.AddVector(fmt.Sprintf("word%d", i), vector)
	}

	for i := 0; b.Loop(); i++ {
		vm.GetVector(fmt.Sprintf("word%d", i%1000))
	}
}

func BenchmarkVectorModel_GetAverageVector(b *testing.B) {
	vm := NewVectorModel(100).(*vectorModel)

	// Add test vectors
	for i := range 1000 {
		vector := make([]float32, 100)
		for j := range 100 {
			vector[j] = float32(i*100 + j)
		}
		vm.AddVector(fmt.Sprintf("word%d", i), vector)
	}

	// Prepare test word lists
	words := []string{"word0", "word1", "word2", "word3", "word4"}

	for b.Loop() {
		vm.GetAverageVector(words)
	}
}

func BenchmarkVectorModel_AddVector(b *testing.B) {
	vm := NewVectorModel(100).(*vectorModel)
	vector := make([]float32, 100)
	for j := range 100 {
		vector[j] = float32(j)
	}

	for i := 0; b.Loop(); i++ {
		vm.AddVector(fmt.Sprintf("word%d", i), vector)
	}
}

func BenchmarkVectorModel_ConcurrentReads(b *testing.B) {
	vm := NewVectorModel(100).(*vectorModel)

	// Add test vectors
	for i := range 1000 {
		vector := make([]float32, 100)
		for j := range 100 {
			vector[j] = float32(i*100 + j)
		}
		vm.AddVector(fmt.Sprintf("word%d", i), vector)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			vm.GetVector(fmt.Sprintf("word%d", i%1000))
			i++
		}
	})
}

func TestVectorModel_OOVStatistics(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add test vectors
	vm.AddVector("word1", []float32{1.0, 2.0, 3.0})
	vm.AddVector("word2", []float32{4.0, 5.0, 6.0})
	vm.AddVector("word3", []float32{7.0, 8.0, 9.0})

	// Initial state - no lookups yet
	if rate := vm.GetOOVRate(); rate != 0.0 {
		t.Errorf("Expected initial OOV rate 0.0, got %f", rate)
	}

	if rate := vm.GetVectorHitRate(); rate != 0.0 {
		t.Errorf("Expected initial hit rate 0.0, got %f", rate)
	}

	// Perform some lookups
	vm.GetVector("word1")       // hit
	vm.GetVector("word2")       // hit
	vm.GetVector("nonexistent") // miss

	// Check statistics
	total, oov, hit := vm.GetLookupStats()
	if total != 3 {
		t.Errorf("Expected 3 total lookups, got %d", total)
	}
	if oov != 1 {
		t.Errorf("Expected 1 OOV lookup, got %d", oov)
	}
	if hit != 2 {
		t.Errorf("Expected 2 hit lookups, got %d", hit)
	}

	// Check rates
	expectedOOVRate := 1.0 / 3.0
	if rate := vm.GetOOVRate(); math.Abs(rate-expectedOOVRate) > 1e-6 {
		t.Errorf("Expected OOV rate %f, got %f", expectedOOVRate, rate)
	}

	expectedHitRate := 2.0 / 3.0
	if rate := vm.GetVectorHitRate(); math.Abs(rate-expectedHitRate) > 1e-6 {
		t.Errorf("Expected hit rate %f, got %f", expectedHitRate, rate)
	}

	// Test GetAverageVector with mixed hits and misses
	vm.GetAverageVector([]string{"word1", "word2", "nonexistent1", "nonexistent2"})

	total, oov, hit = vm.GetLookupStats()
	if total != 7 { // 3 previous + 4 new
		t.Errorf("Expected 7 total lookups, got %d", total)
	}
	if oov != 3 { // 1 previous + 2 new
		t.Errorf("Expected 3 OOV lookups, got %d", oov)
	}
	if hit != 4 { // 2 previous + 2 new
		t.Errorf("Expected 4 hit lookups, got %d", hit)
	}

	// Test reset
	vm.ResetStats()
	total, oov, hit = vm.GetLookupStats()
	if total != 0 || oov != 0 || hit != 0 {
		t.Errorf(
			"Expected all stats to be 0 after reset, got total=%d, oov=%d, hit=%d",
			total,
			oov,
			hit,
		)
	}
}

func TestVectorModel_AllOOVWords(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add test vectors
	vm.AddVector("word1", []float32{1.0, 2.0, 3.0})

	// Try to get average of all OOV words
	_, exists := vm.GetAverageVector([]string{"nonexistent1", "nonexistent2", "nonexistent3"})
	if exists {
		t.Error("Expected GetAverageVector to return false for all OOV words")
	}

	// Check statistics
	total, oov, hit := vm.GetLookupStats()
	if total != 3 {
		t.Errorf("Expected 3 total lookups, got %d", total)
	}
	if oov != 3 {
		t.Errorf("Expected 3 OOV lookups, got %d", oov)
	}
	if hit != 0 {
		t.Errorf("Expected 0 hit lookups, got %d", hit)
	}

	// OOV rate should be 100%
	if rate := vm.GetOOVRate(); rate != 1.0 {
		t.Errorf("Expected OOV rate 1.0, got %f", rate)
	}
}

func TestVectorModel_EmptyInputValidation(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Test GetAverageVector with empty slice
	_, exists := vm.GetAverageVector([]string{})
	if exists {
		t.Error("Expected GetAverageVector to return false for empty input")
	}

	// Statistics should not be affected by empty input
	total, _, _ := vm.GetLookupStats()
	if total != 0 {
		t.Errorf("Expected 0 total lookups for empty input, got %d", total)
	}
}
