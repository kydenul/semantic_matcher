package semanticmatcher

import (
	"strings"
	"testing"
	"time"
)

// createTestVectorModel creates a simple vector model for testing
func createTestVectorModel() VectorModel {
	model := NewVectorModel(3).(*vectorModel)

	// Add some test vectors
	model.AddVector("这", []float32{0.1, 0.2, 0.3})
	model.AddVector("是", []float32{0.2, 0.3, 0.4})
	model.AddVector("一个", []float32{0.3, 0.4, 0.5})
	model.AddVector("测试", []float32{0.4, 0.5, 0.6})
	model.AddVector("段落", []float32{0.5, 0.6, 0.7})
	model.AddVector("文本", []float32{0.6, 0.7, 0.8})
	model.AddVector("关键词", []float32{0.7, 0.8, 0.9})
	model.AddVector("第一个", []float32{0.8, 0.9, 1.0})
	model.AddVector("第二个", []float32{0.9, 1.0, 0.8})

	return model
}

func TestSemanticMatcherStats(t *testing.T) {
	// Create test components
	processor := NewTextProcessor()
	model := createTestVectorModel()
	calculator := NewSimilarityCalculator()
	logger := &mockLogger{messages: make([]string, 0)}

	// Create matcher with logger
	matcher := NewSemanticMatcherWithLogger(processor, model, calculator, logger, 0.5)

	// Test FindTopKeywords
	paragraph := "这是一个测试段落"
	keywords := []string{"测试", "段落", "关键词"}
	results := matcher.FindTopKeywords(paragraph, keywords, 2)

	// Verify results
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Verify stats were updated
	stats := matcher.GetStats()
	if stats.TotalRequests != 1 {
		t.Errorf("Expected 1 request, got %d", stats.TotalRequests)
	}

	if stats.AverageLatency == 0 {
		t.Error("Expected non-zero average latency")
	}

	// Verify logging occurred
	if len(logger.messages) == 0 {
		t.Error("Expected log messages")
	}

	// Check for specific log messages
	hasCompletedLog := false
	for _, msg := range logger.messages {
		if strings.Contains(msg, "completed") {
			hasCompletedLog = true
			break
		}
	}
	if !hasCompletedLog {
		t.Error("Expected 'completed' log message")
	}
}

func TestSemanticMatcherOOVWarning(t *testing.T) {
	// Create test components
	processor := NewTextProcessor()
	model := createTestVectorModel()
	calculator := NewSimilarityCalculator()
	logger := &mockLogger{messages: make([]string, 0)}

	// Create matcher with low OOV threshold
	matcher := NewSemanticMatcherWithLogger(processor, model, calculator, logger, 0.1)

	// Test with text containing many OOV words
	paragraph := "未知词汇1 未知词汇2 未知词汇3"
	keywords := []string{"未知词汇4", "未知词汇5"}
	matcher.FindTopKeywords(paragraph, keywords, 1)

	// Verify OOV warning was logged
	hasWarning := false
	for _, msg := range logger.messages {
		if strings.Contains(msg, "WARN") {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("Expected OOV warning to be logged")
	}
}

func TestComputeSimilarityStats(t *testing.T) {
	// Create test components
	processor := NewTextProcessor()
	model := createTestVectorModel()
	calculator := NewSimilarityCalculator()
	logger := &mockLogger{messages: make([]string, 0)}

	// Create matcher with logger
	matcher := NewSemanticMatcherWithLogger(processor, model, calculator, logger, 0.5)

	// Test ComputeSimilarity
	text1 := "这是第一个文本"
	text2 := "这是第二个文本"
	similarity := matcher.ComputeSimilarity(text1, text2)

	// Verify similarity is computed
	if similarity < 0 || similarity > 1 {
		t.Errorf("Expected similarity in [0, 1], got %f", similarity)
	}

	// Verify stats were updated
	stats := matcher.GetStats()
	if stats.TotalRequests != 1 {
		t.Errorf("Expected 1 request, got %d", stats.TotalRequests)
	}

	// Verify logging occurred
	if len(logger.messages) == 0 {
		t.Error("Expected log messages")
	}
}

func TestGetStatsUpdatesModelMetrics(t *testing.T) {
	// Create test components
	processor := NewTextProcessor()
	model := createTestVectorModel()
	calculator := NewSimilarityCalculator()

	// Create matcher without logger
	matcher := NewSemanticMatcher(processor, model, calculator)

	// Perform some operations to generate stats
	paragraph := "测试文本"
	keywords := []string{"测试", "文本"}
	matcher.FindTopKeywords(paragraph, keywords, 1)

	// Get stats
	stats := matcher.GetStats()

	// Verify stats are populated
	if stats.TotalRequests != 1 {
		t.Errorf("Expected 1 request, got %d", stats.TotalRequests)
	}

	if stats.MemoryUsage == 0 {
		t.Error("Expected non-zero memory usage")
	}

	if stats.LastUpdated.IsZero() {
		t.Error("Expected LastUpdated to be set")
	}

	// Verify OOV rate is calculated
	if stats.OOVRate < 0 || stats.OOVRate > 1 {
		t.Errorf("Expected OOV rate in [0, 1], got %f", stats.OOVRate)
	}

	// Verify vector hit rate is calculated
	if stats.VectorHitRate < 0 || stats.VectorHitRate > 1 {
		t.Errorf("Expected vector hit rate in [0, 1], got %f", stats.VectorHitRate)
	}
}

func TestAverageLatencyCalculation(t *testing.T) {
	// Create test components
	processor := NewTextProcessor()
	model := createTestVectorModel()
	calculator := NewSimilarityCalculator()

	// Create matcher
	matcher := NewSemanticMatcher(processor, model, calculator)

	// Perform multiple operations
	paragraph := "测试段落"
	keywords := []string{"测试"}

	for i := 0; i < 5; i++ {
		matcher.FindTopKeywords(paragraph, keywords, 1)
		time.Sleep(1 * time.Millisecond) // Small delay to vary latency
	}

	// Get stats
	stats := matcher.GetStats()

	// Verify average latency is calculated
	if stats.TotalRequests != 5 {
		t.Errorf("Expected 5 requests, got %d", stats.TotalRequests)
	}

	if stats.AverageLatency == 0 {
		t.Error("Expected non-zero average latency")
	}
}
