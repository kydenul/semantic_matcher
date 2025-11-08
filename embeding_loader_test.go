package semanticmatcher

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// mockLogger implements Logger interface for testing
type mockLogger struct {
	messages []string
}

func (ml *mockLogger) Debug(fields ...any) {
	ml.messages = append(ml.messages, "DEBUG: "+fmt.Sprint(fields...))
}

func (ml *mockLogger) Info(fields ...any) {
	ml.messages = append(ml.messages, "INFO: "+fmt.Sprint(fields...))
}

func (ml *mockLogger) Warn(fields ...any) {
	ml.messages = append(ml.messages, "WARN: "+fmt.Sprint(fields...))
}

func (ml *mockLogger) Error(fields ...any) {
	ml.messages = append(ml.messages, "ERROR: "+fmt.Sprint(fields...))
}

func (ml *mockLogger) Debugf(template string, args ...any) {
	ml.messages = append(ml.messages, "DEBUG: "+fmt.Sprintf(template, args...))
}

func (ml *mockLogger) Infof(template string, args ...any) {
	ml.messages = append(ml.messages, "INFO: "+fmt.Sprintf(template, args...))
}

func (ml *mockLogger) Warnf(template string, args ...any) {
	ml.messages = append(ml.messages, "WARN: "+fmt.Sprintf(template, args...))
}

func (ml *mockLogger) Errorf(template string, args ...any) {
	ml.messages = append(ml.messages, "ERROR: "+fmt.Sprintf(template, args...))
}

func TestNewEmbeddingLoader(t *testing.T) {
	logger := &mockLogger{}
	loader := NewEmbeddingLoader(logger)

	if loader == nil {
		t.Error("Expected non-nil EmbeddingLoader")
	}
}

func TestEmbeddingLoader_LoadFromReader_ValidFormat(t *testing.T) {
	logger := &mockLogger{}
	loader := NewEmbeddingLoader(logger)

	// Create valid .vec format content
	vecContent := `3 2
word1 0.1 0.2
word2 0.3 0.4
word3 0.5 0.6`

	reader := strings.NewReader(vecContent)
	model, err := loader.LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if model == nil {
		t.Fatal("Expected non-nil VectorModel")
	}

	// Check model properties
	if model.Dimension() != 2 {
		t.Errorf("Expected dimension 2, got %d", model.Dimension())
	}

	if model.VocabularySize() != 3 {
		t.Errorf("Expected vocabulary size 3, got %d", model.VocabularySize())
	}

	// Check specific vectors
	vector1, exists := model.GetVector("word1")
	if !exists {
		t.Error("Expected word1 to exist")
	}
	if len(vector1) != 2 || vector1[0] != 0.1 || vector1[1] != 0.2 {
		t.Errorf("Expected word1 vector [0.1, 0.2], got %v", vector1)
	}

	vector2, exists := model.GetVector("word2")
	if !exists {
		t.Error("Expected word2 to exist")
	}
	if len(vector2) != 2 || vector2[0] != 0.3 || vector2[1] != 0.4 {
		t.Errorf("Expected word2 vector [0.3, 0.4], got %v", vector2)
	}
}

func TestEmbeddingLoader_LoadFromReader_InvalidFirstLine(t *testing.T) {
	logger := &mockLogger{}
	loader := NewEmbeddingLoader(logger)

	testCases := []struct {
		name    string
		content string
	}{
		{"empty file", ""},
		{"missing dimension", "100"},
		{"invalid word count", "abc 100"},
		{"invalid dimension", "100 abc"},
		{"negative word count", "-1 100"},
		{"zero dimension", "100 0"},
		{"extra fields", "100 200 300"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader(tc.content)
			model, err := loader.LoadFromReader(reader)

			if err == nil {
				t.Errorf("Expected error for %s, got nil", tc.name)
			}

			if model != nil {
				t.Errorf("Expected nil model for %s, got non-nil", tc.name)
			}
		})
	}
}

func TestEmbeddingLoader_LoadFromReader_InvalidVectorLines(t *testing.T) {
	logger := &mockLogger{}
	loader := NewEmbeddingLoader(logger)

	// Content with some invalid lines that should be skipped
	vecContent := `3 2
word1 0.1 0.2
invalid_line_missing_values 0.3
word2 0.4 abc
word3 0.5 0.6`

	reader := strings.NewReader(vecContent)
	model, err := loader.LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should only load word1 and word3 (word2 has invalid float)
	if model.VocabularySize() != 2 {
		t.Errorf("Expected vocabulary size 2, got %d", model.VocabularySize())
	}

	// Check that valid vectors were loaded
	_, exists1 := model.GetVector("word1")
	if !exists1 {
		t.Error("Expected word1 to exist")
	}

	_, exists3 := model.GetVector("word3")
	if !exists3 {
		t.Error("Expected word3 to exist")
	}

	// Check that invalid vectors were not loaded
	_, exists2 := model.GetVector("word2")
	if exists2 {
		t.Error("Expected word2 to not exist due to invalid float")
	}
}

func TestEmbeddingLoader_LoadFromReader_EmptyLines(t *testing.T) {
	logger := &mockLogger{}
	loader := NewEmbeddingLoader(logger)

	// Content with empty lines that should be skipped
	vecContent := `2 2
word1 0.1 0.2

word2 0.3 0.4

`

	reader := strings.NewReader(vecContent)
	model, err := loader.LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if model.VocabularySize() != 2 {
		t.Errorf("Expected vocabulary size 2, got %d", model.VocabularySize())
	}
}

func TestEmbeddingLoader_LoadFromFile_FileNotFound(t *testing.T) {
	logger := &mockLogger{}
	loader := NewEmbeddingLoader(logger)

	model, err := loader.LoadFromFile("nonexistent_file.vec")

	if err != ErrVectorFileNotFound {
		t.Errorf("Expected ErrVectorFileNotFound, got: %v", err)
	}

	if model != nil {
		t.Error("Expected nil model for nonexistent file")
	}
}

func TestEmbeddingLoader_LoadFromFile_ValidFile(t *testing.T) {
	logger := &mockLogger{}
	loader := NewEmbeddingLoader(logger)

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test_vectors_*.vec")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test content
	vecContent := `2 3
hello 0.1 0.2 0.3
world 0.4 0.5 0.6`

	if _, err := tmpFile.WriteString(vecContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Load from file
	model, err := loader.LoadFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if model == nil {
		t.Fatal("Expected non-nil VectorModel")
	}

	if model.Dimension() != 3 {
		t.Errorf("Expected dimension 3, got %d", model.Dimension())
	}

	if model.VocabularySize() != 2 {
		t.Errorf("Expected vocabulary size 2, got %d", model.VocabularySize())
	}
}

func TestEmbeddingLoader_LoadFromReader_LargeFile(t *testing.T) {
	logger := &mockLogger{}
	loader := NewEmbeddingLoader(logger)

	// Create content with many vectors to test progress logging
	var builder strings.Builder
	builder.WriteString("100 2\n")

	for i := range 100 {
		builder.WriteString("word")
		builder.WriteString(fmt.Sprintf("%d", i))
		builder.WriteString(" 0.1 0.2\n")
	}

	reader := strings.NewReader(builder.String())
	model, err := loader.LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if model.VocabularySize() != 100 {
		t.Errorf("Expected vocabulary size 100, got %d", model.VocabularySize())
	}
}

func TestVectorModel_MemoryUsage(t *testing.T) {
	model := NewVectorModel(3).(*vectorModel)

	// Initially should have minimal memory usage
	initialMemory := model.MemoryUsage()
	if initialMemory != 0 {
		t.Errorf("Expected initial memory usage to be 0, got %d", initialMemory)
	}

	// Add some vectors
	model.AddVector("word1", []float32{0.1, 0.2, 0.3})
	model.AddVector("word2", []float32{0.4, 0.5, 0.6})
	model.AddVector("word3", []float32{0.7, 0.8, 0.9})

	// Memory usage should increase
	memoryAfterAdd := model.MemoryUsage()
	if memoryAfterAdd <= 0 {
		t.Errorf("Expected positive memory usage after adding vectors, got %d", memoryAfterAdd)
	}

	// Memory should be proportional to number of vectors
	// Each vector has 3 float32s (12 bytes) + string overhead + map overhead
	expectedMinMemory := int64(3 * 12) // At least the vector data
	if memoryAfterAdd < expectedMinMemory {
		t.Errorf("Expected memory usage >= %d, got %d", expectedMinMemory, memoryAfterAdd)
	}
}

func TestVectorModel_StringInterning(t *testing.T) {
	model := NewVectorModel(2).(*vectorModel)

	// Add the same word multiple times (shouldn't happen in practice, but tests interning)
	model.AddVector("duplicate", []float32{0.1, 0.2})
	model.AddVector("duplicate", []float32{0.3, 0.4}) // Should overwrite

	// Check that only one entry exists
	if model.VocabularySize() != 1 {
		t.Errorf("Expected vocabulary size 1, got %d", model.VocabularySize())
	}

	// Check that the interned string pool has the word
	if len(model.stringIntern) != 1 {
		t.Errorf("Expected string intern pool size 1, got %d", len(model.stringIntern))
	}
}

func TestEmbeddingLoader_ProgressCallback(t *testing.T) {
	logger := &mockLogger{}
	loader := NewEmbeddingLoader(logger)

	// Track progress callback invocations
	var progressCalls []struct {
		loaded int
		total  int
		memory int64
	}

	// Set progress callback
	loader.SetProgressCallback(func(loaded, total int, memoryUsage int64) {
		progressCalls = append(progressCalls, struct {
			loaded int
			total  int
			memory int64
		}{loaded, total, memoryUsage})
	})

	// Create content with enough vectors to trigger progress reporting
	var builder strings.Builder
	builder.WriteString("15000 2\n")

	for i := range 15000 {
		builder.WriteString(fmt.Sprintf("word%d 0.1 0.2\n", i))
	}

	reader := strings.NewReader(builder.String())
	model, err := loader.LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if model.VocabularySize() != 15000 {
		t.Errorf("Expected vocabulary size 15000, got %d", model.VocabularySize())
	}

	// Should have received progress callbacks
	if len(progressCalls) == 0 {
		t.Error("Expected at least one progress callback")
	}

	// Verify progress callbacks have increasing or equal loaded counts (final callback may repeat)
	for i := 1; i < len(progressCalls); i++ {
		if progressCalls[i].loaded < progressCalls[i-1].loaded {
			t.Errorf("Expected non-decreasing loaded count, got %d after %d",
				progressCalls[i].loaded, progressCalls[i-1].loaded)
		}
	}

	// Verify final callback has correct total
	finalCall := progressCalls[len(progressCalls)-1]
	if finalCall.total != 15000 {
		t.Errorf("Expected final total 15000, got %d", finalCall.total)
	}

	// Verify memory usage is reported and increasing
	for _, call := range progressCalls {
		if call.memory <= 0 {
			t.Errorf("Expected positive memory usage in callback, got %d", call.memory)
		}
	}
}

func TestEmbeddingLoader_MemoryReporting(t *testing.T) {
	logger := &mockLogger{}
	loader := NewEmbeddingLoader(logger)

	// Create content
	vecContent := `3 100
word1 ` + strings.Repeat("0.1 ", 100) + `
word2 ` + strings.Repeat("0.2 ", 100) + `
word3 ` + strings.Repeat("0.3 ", 100)

	reader := strings.NewReader(vecContent)
	model, err := loader.LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that memory usage is tracked
	memUsage := model.MemoryUsage()
	if memUsage <= 0 {
		t.Errorf("Expected positive memory usage, got %d", memUsage)
	}

	// Memory should be at least the size of the vectors
	// 3 words * 100 dimensions * 4 bytes per float32 = 1200 bytes minimum
	minExpectedMemory := int64(3 * 100 * 4)
	if memUsage < minExpectedMemory {
		t.Errorf("Expected memory usage >= %d, got %d", minExpectedMemory, memUsage)
	}

	// Check that logger received completion information
	foundCompletionLog := false
	for _, msg := range logger.messages {
		if strings.Contains(msg, "completed") || strings.Contains(msg, "Loading progress") {
			foundCompletionLog = true
			break
		}
	}
	if !foundCompletionLog {
		t.Error("Expected loading completion to be logged")
	}
}

// Benchmark tests for embedding loading performance

func BenchmarkEmbeddingLoader_LoadSmallFile(b *testing.B) {
	logger := &mockLogger{}

	// Create small test content (100 vectors, dimension 50)
	var builder strings.Builder
	builder.WriteString("100 50\n")
	for i := range 100 {
		builder.WriteString(fmt.Sprintf("word%d", i))
		for j := range 50 {
			builder.WriteString(fmt.Sprintf(" %f", float32(i*50+j)*0.01))
		}
		builder.WriteString("\n")
	}
	content := builder.String()

	for b.Loop() {
		loader := NewEmbeddingLoader(logger)
		reader := strings.NewReader(content)
		_, err := loader.LoadFromReader(reader)
		if err != nil {
			b.Fatalf("Failed to load: %v", err)
		}
	}
}

func BenchmarkEmbeddingLoader_LoadMediumFile(b *testing.B) {
	logger := &mockLogger{}

	// Create medium test content (1000 vectors, dimension 100)
	var builder strings.Builder
	builder.WriteString("1000 100\n")
	for i := range 1000 {
		builder.WriteString(fmt.Sprintf("word%d", i))
		for j := range 100 {
			builder.WriteString(fmt.Sprintf(" %f", float32(i*100+j)*0.01))
		}
		builder.WriteString("\n")
	}
	content := builder.String()

	for b.Loop() {
		loader := NewEmbeddingLoader(logger)
		reader := strings.NewReader(content)
		_, err := loader.LoadFromReader(reader)
		if err != nil {
			b.Fatalf("Failed to load: %v", err)
		}
	}
}

func BenchmarkEmbeddingLoader_ParseVectorLine(b *testing.B) {
	logger := &mockLogger{}
	loader := NewEmbeddingLoader(logger)

	// Create content with single vector repeated
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%d 100\n", b.N))
	for i := 0; b.Loop(); i++ {
		builder.WriteString(fmt.Sprintf("word%d", i))
		for range 100 {
			builder.WriteString(" 0.123456")
		}
		builder.WriteString("\n")
	}

	reader := strings.NewReader(builder.String())
	b.ResetTimer()
	_, err := loader.LoadFromReader(reader)
	if err != nil {
		b.Fatalf("Failed to load: %v", err)
	}
}
