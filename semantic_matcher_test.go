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

// TestNewSemanticMatcherFromConfigSingleFile tests initialization with a single vector file
func TestNewSemanticMatcherFromConfigSingleFile(t *testing.T) {
	logger := &mockLogger{messages: make([]string, 0)}

	config := &Config{
		VectorFilePaths:    []string{"vector/wiki.zh.align.vec"},
		MaxSequenceLen:     512,
		EnableStats:        true,
		MemoryLimit:        10 * 1024 * 1024 * 1024, // 10GB
		SupportedLanguages: []string{"zh"},
	}

	matcher, err := NewSemanticMatcherFromConfig(config, logger)
	if err != nil {
		t.Fatalf("Failed to create SemanticMatcher: %v", err)
	}

	if matcher == nil {
		t.Fatal("Expected non-nil matcher")
	}

	// Verify logging occurred
	hasLoadingLog := false
	for _, msg := range logger.messages {
		if strings.Contains(msg, "Loading vector model") && strings.Contains(msg, "file_count: 1") {
			hasLoadingLog = true
			break
		}
	}
	if !hasLoadingLog {
		t.Error("Expected loading log with file_count: 1")
	}

	// Test that matcher works with Chinese text
	text1 := "这是测试"
	text2 := "中文词汇"
	similarity := matcher.ComputeSimilarity(text1, text2)

	if similarity < 0 || similarity > 1 {
		t.Errorf("Expected similarity in [0, 1], got %f", similarity)
	}
}

// TestNewSemanticMatcherFromConfigMultipleFiles tests initialization with multiple vector files
func TestNewSemanticMatcherFromConfigMultipleFiles(t *testing.T) {
	logger := &mockLogger{messages: make([]string, 0)}

	config := &Config{
		VectorFilePaths:    []string{"vector/wiki.zh.align.vec", "vector/wiki.en.align.vec"},
		MaxSequenceLen:     512,
		EnableStats:        true,
		MemoryLimit:        10 * 1024 * 1024 * 1024, // 10GB
		SupportedLanguages: []string{"zh", "en"},
	}

	matcher, err := NewSemanticMatcherFromConfig(config, logger)
	if err != nil {
		t.Fatalf("Failed to create SemanticMatcher: %v", err)
	}

	if matcher == nil {
		t.Fatal("Expected non-nil matcher")
	}

	// Verify logging occurred with correct file count
	hasLoadingLog := false
	for _, msg := range logger.messages {
		if strings.Contains(msg, "Loading vector model") && strings.Contains(msg, "file_count: 2") {
			hasLoadingLog = true
			break
		}
	}
	if !hasLoadingLog {
		t.Error("Expected loading log with file_count: 2")
	}

	// Verify model loaded successfully log
	hasSuccessLog := false
	for _, msg := range logger.messages {
		if strings.Contains(msg, "Vector model loaded successfully") {
			hasSuccessLog = true
			break
		}
	}
	if !hasSuccessLog {
		t.Error("Expected success log")
	}

	// Test that matcher works with both Chinese and English text
	chineseText := "这是测试"
	englishText := "test english"
	similarity := matcher.ComputeSimilarity(chineseText, englishText)

	if similarity < 0 || similarity > 1 {
		t.Errorf("Expected similarity in [0, 1], got %f", similarity)
	}
}

// TestNewSemanticMatcherFromConfigMemoryLimit tests memory limit checking
func TestNewSemanticMatcherFromConfigMemoryLimit(t *testing.T) {
	logger := &mockLogger{messages: make([]string, 0)}

	config := &Config{
		VectorFilePaths:    []string{"vector/wiki.zh.align.vec"},
		MaxSequenceLen:     512,
		EnableStats:        true,
		MemoryLimit:        1, // Very low limit to trigger error
		SupportedLanguages: []string{"zh"},
	}

	matcher, err := NewSemanticMatcherFromConfig(config, logger)
	if err != ErrMemoryLimitExceeded {
		t.Errorf("Expected ErrMemoryLimitExceeded, got %v", err)
	}

	if matcher != nil {
		t.Error("Expected nil matcher when memory limit exceeded")
	}

	// Verify warning was logged
	hasWarning := false
	for _, msg := range logger.messages {
		if strings.Contains(msg, "WARN") && strings.Contains(msg, "Memory usage exceeds limit") {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("Expected memory limit warning")
	}
}

// TestNewSemanticMatcherFromConfigInvalidConfig tests configuration validation
func TestNewSemanticMatcherFromConfigInvalidConfig(t *testing.T) {
	logger := &mockLogger{messages: make([]string, 0)}

	tests := []struct {
		name   string
		config *Config
		errMsg string
	}{
		{
			name:   "nil config",
			config: nil,
			errMsg: "invalid configuration",
		},
		{
			name: "empty vector files",
			config: &Config{
				VectorFilePaths:    []string{},
				MaxSequenceLen:     512,
				SupportedLanguages: []string{"zh"},
			},
			errMsg: "no vector files",
		},
		{
			name: "invalid max sequence length",
			config: &Config{
				VectorFilePaths:    []string{"testdata/test_zh.vec"},
				MaxSequenceLen:     0,
				SupportedLanguages: []string{"zh"},
			},
			errMsg: "invalid configuration",
		},
		{
			name: "empty supported languages",
			config: &Config{
				VectorFilePaths:    []string{"testdata/test_zh.vec"},
				MaxSequenceLen:     512,
				SupportedLanguages: []string{},
			},
			errMsg: "invalid configuration",
		},
		{
			name: "unsupported language",
			config: &Config{
				VectorFilePaths:    []string{"testdata/test_zh.vec"},
				MaxSequenceLen:     512,
				SupportedLanguages: []string{"fr"},
			},
			errMsg: "unsupported language",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := NewSemanticMatcherFromConfig(tt.config, logger)
			if err == nil {
				t.Errorf("Expected error containing '%s', got nil", tt.errMsg)
			}
			if matcher != nil {
				t.Error("Expected nil matcher on error")
			}
		})
	}
}

// TestNewSemanticMatcherFromConfigWithStopWords tests initialization with stop words
func TestNewSemanticMatcherFromConfigWithStopWords(t *testing.T) {
	logger := &mockLogger{messages: make([]string, 0)}

	config := &Config{
		VectorFilePaths:    []string{"vector/wiki.zh.align.vec"},
		MaxSequenceLen:     512,
		ChineseStopWords:   "testdata/chinese_stopwords.txt",
		EnglishStopWords:   "testdata/english_stopwords.txt",
		EnableStats:        true,
		MemoryLimit:        10 * 1024 * 1024 * 1024, // 10GB
		SupportedLanguages: []string{"zh"},
	}

	// This should not fail even if stop word files don't exist
	// It should fall back to default processor
	matcher, err := NewSemanticMatcherFromConfig(config, logger)
	if err != nil {
		t.Fatalf("Failed to create SemanticMatcher: %v", err)
	}

	if matcher == nil {
		t.Fatal("Expected non-nil matcher")
	}
}

// TestNewSemanticMatcherFromConfigLogsMemoryUsage tests that memory usage is logged
func TestNewSemanticMatcherFromConfigLogsMemoryUsage(t *testing.T) {
	logger := &mockLogger{messages: make([]string, 0)}

	config := &Config{
		VectorFilePaths:    []string{"vector/wiki.zh.align.vec"},
		MaxSequenceLen:     512,
		EnableStats:        true,
		MemoryLimit:        10 * 1024 * 1024 * 1024, // 10GB
		SupportedLanguages: []string{"zh"},
	}

	matcher, err := NewSemanticMatcherFromConfig(config, logger)
	if err != nil {
		t.Fatalf("Failed to create SemanticMatcher: %v", err)
	}

	if matcher == nil {
		t.Fatal("Expected non-nil matcher")
	}

	// Verify memory usage was logged
	hasMemoryLog := false
	for _, msg := range logger.messages {
		if strings.Contains(msg, "memory_mb") || strings.Contains(msg, "Memory usage within limit") {
			hasMemoryLog = true
			break
		}
	}
	if !hasMemoryLog {
		t.Error("Expected memory usage to be logged")
	}

	// Verify initialization success log includes all details
	hasDetailedLog := false
	for _, msg := range logger.messages {
		if strings.Contains(msg, "SemanticMatcher initialized successfully") &&
			strings.Contains(msg, "file_count") &&
			strings.Contains(msg, "vocabulary_size") &&
			strings.Contains(msg, "memory_usage_mb") {
			hasDetailedLog = true
			break
		}
	}
	if !hasDetailedLog {
		t.Error("Expected detailed initialization log")
	}
}
