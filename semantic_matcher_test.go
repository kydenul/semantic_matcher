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

// TestFallbackLoggingInFindTopKeywords tests that fallback statistics are logged in FindTopKeywords
func TestFallbackLoggingInFindTopKeywords(t *testing.T) {
	// Create test components with a model that has individual characters but not full words
	model := NewVectorModel(3).(*vectorModel)

	// Add individual character vectors (for fallback)
	model.AddVector("没", []float32{0.1, 0.2, 0.3})
	model.AddVector("事", []float32{0.2, 0.3, 0.4})
	model.AddVector("测", []float32{0.3, 0.4, 0.5})
	model.AddVector("试", []float32{0.4, 0.5, 0.6})

	// Add some known words
	model.AddVector("这是", []float32{0.5, 0.6, 0.7})
	model.AddVector("一个", []float32{0.6, 0.7, 0.8})

	processor := NewTextProcessor()
	calculator := NewSimilarityCalculator()
	logger := &mockLogger{messages: make([]string, 0)}

	// Create matcher with logger
	matcher := NewSemanticMatcherWithLogger(processor, model, calculator, logger, 0.5)

	// Test with OOV words that will trigger fallback
	paragraph := "这是 一个 测试"
	keywords := []string{"没事", "测试"}
	matcher.FindTopKeywords(paragraph, keywords, 2)

	// Verify fallback logging occurred
	hasFallbackDebugLog := false
	hasFallbackInfoLog := false

	for _, msg := range logger.messages {
		if strings.Contains(msg, "DEBUG") && strings.Contains(msg, "Character-level fallback used in FindTopKeywords") {
			hasFallbackDebugLog = true
		}
		if strings.Contains(msg, "INFO") && strings.Contains(msg, "FindTopKeywords completed") &&
			strings.Contains(msg, "fallback_attempts") && strings.Contains(msg, "fallback_success_rate") {
			hasFallbackInfoLog = true
		}
	}

	if !hasFallbackDebugLog {
		t.Error("Expected DEBUG log for character-level fallback in FindTopKeywords")
	}

	if !hasFallbackInfoLog {
		t.Error("Expected INFO log with fallback statistics in FindTopKeywords")
	}
}

// TestFallbackLoggingInComputeSimilarity tests that fallback statistics are logged in ComputeSimilarity
func TestFallbackLoggingInComputeSimilarity(t *testing.T) {
	// Create test components with a model that has individual characters but not full words
	model := NewVectorModel(3).(*vectorModel)

	// Add individual character vectors (for fallback)
	model.AddVector("没", []float32{0.1, 0.2, 0.3})
	model.AddVector("事", []float32{0.2, 0.3, 0.4})
	model.AddVector("测", []float32{0.3, 0.4, 0.5})
	model.AddVector("试", []float32{0.4, 0.5, 0.6})

	processor := NewTextProcessor()
	calculator := NewSimilarityCalculator()
	logger := &mockLogger{messages: make([]string, 0)}

	// Create matcher with logger
	matcher := NewSemanticMatcherWithLogger(processor, model, calculator, logger, 0.5)

	// Test with OOV words that will trigger fallback
	text1 := "没事"
	text2 := "测试"
	matcher.ComputeSimilarity(text1, text2)

	// Verify fallback logging occurred
	hasFallbackDebugLog := false
	hasFallbackInfoLog := false

	for _, msg := range logger.messages {
		if strings.Contains(msg, "DEBUG") && strings.Contains(msg, "Character-level fallback used in ComputeSimilarity") {
			hasFallbackDebugLog = true
		}
		if strings.Contains(msg, "INFO") && strings.Contains(msg, "ComputeSimilarity completed") &&
			strings.Contains(msg, "fallback_attempts") && strings.Contains(msg, "fallback_success_rate") {
			hasFallbackInfoLog = true
		}
	}

	if !hasFallbackDebugLog {
		t.Error("Expected DEBUG log for character-level fallback in ComputeSimilarity")
	}

	if !hasFallbackInfoLog {
		t.Error("Expected INFO log with fallback statistics in ComputeSimilarity")
	}
}

// TestNoFallbackLoggingWhenNotUsed tests that fallback logs are not generated when fallback is not used
func TestNoFallbackLoggingWhenNotUsed(t *testing.T) {
	// Create test components with all words in vocabulary
	model := NewVectorModel(3).(*vectorModel)

	// Add complete words (no fallback needed)
	model.AddVector("这是", []float32{0.1, 0.2, 0.3})
	model.AddVector("测试", []float32{0.2, 0.3, 0.4})
	model.AddVector("文本", []float32{0.3, 0.4, 0.5})

	processor := NewTextProcessor()
	calculator := NewSimilarityCalculator()
	logger := &mockLogger{messages: make([]string, 0)}

	// Create matcher with logger
	matcher := NewSemanticMatcherWithLogger(processor, model, calculator, logger, 0.5)

	// Test with known words (no fallback)
	paragraph := "这是 测试"
	keywords := []string{"测试", "文本"}
	matcher.FindTopKeywords(paragraph, keywords, 2)

	// Verify no fallback debug logging occurred (since no fallback was used)
	hasFallbackDebugLog := false

	for _, msg := range logger.messages {
		if strings.Contains(msg, "DEBUG") && strings.Contains(msg, "Character-level fallback used") {
			hasFallbackDebugLog = true
		}
	}

	if hasFallbackDebugLog {
		t.Error("Did not expect DEBUG log for character-level fallback when no fallback occurred")
	}

	// But INFO log should still contain fallback stats (just with 0 attempts)
	hasFallbackInfoLog := false
	for _, msg := range logger.messages {
		if strings.Contains(msg, "INFO") && strings.Contains(msg, "completed") &&
			strings.Contains(msg, "fallback_attempts") {
			hasFallbackInfoLog = true
		}
	}

	if !hasFallbackInfoLog {
		t.Error("Expected INFO log with fallback statistics even when no fallback occurred")
	}
}

// TestFindTopKeywordsWithFallback tests FindTopKeywords with character-level fallback
// This integration test validates Requirements 1.1 and 3.1
func TestFindTopKeywordsWithFallback(t *testing.T) {
	// Create test model with individual characters but not full words
	model := NewVectorModel(3).(*vectorModel)

	// Add individual character vectors (for fallback)
	model.AddVector("没", []float32{0.1, 0.2, 0.3})
	model.AddVector("事", []float32{0.2, 0.3, 0.4})
	model.AddVector("测", []float32{0.3, 0.4, 0.5})
	model.AddVector("试", []float32{0.4, 0.5, 0.6})
	model.AddVector("问", []float32{0.5, 0.6, 0.7})
	model.AddVector("题", []float32{0.6, 0.7, 0.8})

	// Add some known words
	model.AddVector("这是", []float32{0.7, 0.8, 0.9})
	model.AddVector("一个", []float32{0.8, 0.9, 1.0})

	processor := NewTextProcessor()
	calculator := NewSimilarityCalculator()
	logger := &mockLogger{messages: make([]string, 0)}

	// Create matcher with logger
	matcher := NewSemanticMatcherWithLogger(processor, model, calculator, logger, 0.5)

	// Test with OOV words that will trigger fallback
	paragraph := "这是 一个 测试"
	keywords := []string{"没事", "测试", "问题"}

	results := matcher.FindTopKeywords(paragraph, keywords, 3)

	// Verify results
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Verify that OOV words with character fallback got non-zero scores
	foundFallbackKeyword := false
	for _, result := range results {
		if result.Keyword == "没事" || result.Keyword == "测试" || result.Keyword == "问题" {
			if result.Score > 0 {
				foundFallbackKeyword = true
			}
		}
	}

	if !foundFallbackKeyword {
		t.Error("Expected at least one OOV keyword to have non-zero score via fallback")
	}

	// Verify fallback statistics were recorded
	_, _, _, fallbackAttempts, fallbackSuccesses, _ := model.GetLookupStats()

	if fallbackAttempts == 0 {
		t.Error("Expected fallback attempts to be recorded")
	}

	if fallbackSuccesses == 0 {
		t.Error("Expected at least one successful fallback")
	}

	// Verify logging occurred
	hasFallbackLog := false
	for _, msg := range logger.messages {
		if strings.Contains(msg, "fallback") {
			hasFallbackLog = true
			break
		}
	}

	if !hasFallbackLog {
		t.Error("Expected fallback information to be logged")
	}
}

// TestComputeSimilarityWithFallback tests ComputeSimilarity with character-level fallback
// This integration test validates Requirements 1.1 and 3.1
func TestComputeSimilarityWithFallback(t *testing.T) {
	// Create test model with individual characters but not full words
	model := NewVectorModel(3).(*vectorModel)

	// Add individual character vectors (for fallback)
	model.AddVector("没", []float32{0.1, 0.2, 0.3})
	model.AddVector("事", []float32{0.2, 0.3, 0.4})
	model.AddVector("测", []float32{0.3, 0.4, 0.5})
	model.AddVector("试", []float32{0.4, 0.5, 0.6})

	processor := NewTextProcessor()
	calculator := NewSimilarityCalculator()
	logger := &mockLogger{messages: make([]string, 0)}

	// Create matcher with logger
	matcher := NewSemanticMatcherWithLogger(processor, model, calculator, logger, 0.5)

	// Test with OOV words that will trigger fallback
	text1 := "没事"
	text2 := "测试"

	similarity := matcher.ComputeSimilarity(text1, text2)

	// Verify similarity is computed (should be non-zero due to fallback)
	if similarity <= 0 {
		t.Errorf("Expected non-zero similarity via fallback, got %f", similarity)
	}

	if similarity < 0 || similarity > 1 {
		t.Errorf("Expected similarity in [0, 1], got %f", similarity)
	}

	// Verify fallback statistics were recorded
	_, _, _, fallbackAttempts, fallbackSuccesses, _ := model.GetLookupStats()

	if fallbackAttempts == 0 {
		t.Error("Expected fallback attempts to be recorded")
	}

	if fallbackSuccesses == 0 {
		t.Error("Expected at least one successful fallback")
	}

	// Verify logging occurred
	hasFallbackLog := false
	for _, msg := range logger.messages {
		if strings.Contains(msg, "fallback") {
			hasFallbackLog = true
			break
		}
	}

	if !hasFallbackLog {
		t.Error("Expected fallback information to be logged")
	}
}

// TestRealChineseOOVWord tests with the real Chinese OOV word "没事"
// This integration test validates Requirements 1.1 and 3.1
func TestRealChineseOOVWord(t *testing.T) {
	// Create test model with individual characters but not the full word "没事"
	model := NewVectorModel(3).(*vectorModel)

	// Add individual character vectors
	model.AddVector("没", []float32{0.1, 0.2, 0.3})
	model.AddVector("事", []float32{0.2, 0.3, 0.4})

	// Add some related words for comparison
	model.AddVector("问题", []float32{0.15, 0.25, 0.35})
	model.AddVector("正常", []float32{0.12, 0.22, 0.32})
	model.AddVector("好的", []float32{0.11, 0.21, 0.31})

	processor := NewTextProcessor()
	calculator := NewSimilarityCalculator()
	logger := &mockLogger{messages: make([]string, 0)}

	// Create matcher
	matcher := NewSemanticMatcherWithLogger(processor, model, calculator, logger, 0.5)

	// Test 1: FindTopKeywords with "没事" as paragraph
	t.Run("FindTopKeywords_OOV_Paragraph", func(t *testing.T) {
		paragraph := "没事"
		keywords := []string{"问题", "正常", "好的"}

		results := matcher.FindTopKeywords(paragraph, keywords, 3)

		// Should get results via character-level fallback
		if len(results) == 0 {
			t.Error("Expected results via character-level fallback")
		}

		// At least one result should have non-zero score
		hasNonZeroScore := false
		for _, result := range results {
			if result.Score > 0 {
				hasNonZeroScore = true
				break
			}
		}

		if !hasNonZeroScore {
			t.Error("Expected at least one keyword to have non-zero score")
		}
	})

	// Test 2: FindTopKeywords with "没事" as keyword
	t.Run("FindTopKeywords_OOV_Keyword", func(t *testing.T) {
		paragraph := "问题"
		keywords := []string{"没事", "正常", "好的"}

		results := matcher.FindTopKeywords(paragraph, keywords, 3)

		// Should get results, including for "没事" via fallback
		if len(results) == 0 {
			t.Error("Expected results via character-level fallback")
		}

		// Find the "没事" result
		foundOOV := false
		for _, result := range results {
			if result.Keyword == "没事" {
				foundOOV = true
				if result.Score <= 0 {
					t.Errorf("Expected non-zero score for OOV keyword '没事', got %f", result.Score)
				}
			}
		}

		if !foundOOV {
			t.Error("Expected to find '没事' in results")
		}
	})

	// Test 3: ComputeSimilarity with "没事"
	t.Run("ComputeSimilarity_OOV", func(t *testing.T) {
		text1 := "没事"
		text2 := "问题"

		similarity := matcher.ComputeSimilarity(text1, text2)

		// Should get non-zero similarity via fallback
		if similarity <= 0 {
			t.Errorf("Expected non-zero similarity via fallback, got %f", similarity)
		}

		if similarity < 0 || similarity > 1 {
			t.Errorf("Expected similarity in [0, 1], got %f", similarity)
		}
	})

	// Verify fallback was used
	_, _, _, fallbackAttempts, fallbackSuccesses, _ := model.GetLookupStats()

	if fallbackAttempts == 0 {
		t.Error("Expected fallback attempts for OOV word '没事'")
	}

	if fallbackSuccesses == 0 {
		t.Error("Expected successful fallback for OOV word '没事'")
	}
}

// TestMixedKnownAndOOVWords tests scenarios with both known and OOV words
// This integration test validates Requirements 1.1, 3.1, 3.2, and 3.3
func TestMixedKnownAndOOVWords(t *testing.T) {
	// Create test model with some complete words and some individual characters
	model := NewVectorModel(3).(*vectorModel)

	// Add complete words
	model.AddVector("这是", []float32{0.1, 0.2, 0.3})
	model.AddVector("一个", []float32{0.2, 0.3, 0.4})
	model.AddVector("测试", []float32{0.3, 0.4, 0.5})

	// Add individual characters for fallback
	model.AddVector("没", []float32{0.4, 0.5, 0.6})
	model.AddVector("事", []float32{0.5, 0.6, 0.7})
	model.AddVector("问", []float32{0.6, 0.7, 0.8})
	model.AddVector("题", []float32{0.7, 0.8, 0.9})

	processor := NewTextProcessor()
	calculator := NewSimilarityCalculator()
	logger := &mockLogger{messages: make([]string, 0)}

	// Create matcher
	matcher := NewSemanticMatcherWithLogger(processor, model, calculator, logger, 0.5)

	// Test with mixed known and OOV words
	paragraph := "这是 一个 测试 没事"
	keywords := []string{"测试", "没事", "问题"}

	results := matcher.FindTopKeywords(paragraph, keywords, 3)

	// Should get results for all keywords
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// All results should have non-zero scores
	for _, result := range results {
		if result.Score <= 0 {
			t.Errorf("Expected non-zero score for keyword %s, got %f", result.Keyword, result.Score)
		}
	}

	// Verify both direct lookups and fallbacks occurred
	totalLookups, oovLookups, hitLookups, fallbackAttempts, fallbackSuccesses, _ := model.GetLookupStats()

	if totalLookups == 0 {
		t.Error("Expected total lookups to be recorded")
	}

	if hitLookups == 0 {
		t.Error("Expected some direct hits for known words")
	}

	if oovLookups == 0 {
		t.Error("Expected some OOV lookups")
	}

	if fallbackAttempts == 0 {
		t.Error("Expected fallback attempts for OOV words")
	}

	if fallbackSuccesses == 0 {
		t.Error("Expected successful fallbacks")
	}

	// Verify logging includes both known word hits and fallback usage
	hasHitLog := false
	hasFallbackLog := false

	for _, msg := range logger.messages {
		if strings.Contains(msg, "completed") {
			hasHitLog = true
		}
		if strings.Contains(msg, "fallback") {
			hasFallbackLog = true
		}
	}

	if !hasHitLog {
		t.Error("Expected completion log")
	}

	if !hasFallbackLog {
		t.Error("Expected fallback log")
	}
}

// TestFallbackStatisticsAccuracy tests that fallback statistics are accurately reported
// This integration test validates Requirements 4.1, 4.2, 4.3, and 4.4
func TestFallbackStatisticsAccuracy(t *testing.T) {
	// Create test model
	model := NewVectorModel(3).(*vectorModel)

	// Add individual characters
	model.AddVector("测", []float32{0.1, 0.2, 0.3})
	model.AddVector("试", []float32{0.2, 0.3, 0.4})
	model.AddVector("没", []float32{0.3, 0.4, 0.5})
	// Note: "事" is intentionally missing to test fallback failure

	processor := NewTextProcessor()
	calculator := NewSimilarityCalculator()
	logger := &mockLogger{messages: make([]string, 0)}

	// Create matcher
	matcher := NewSemanticMatcherWithLogger(processor, model, calculator, logger, 0.5)

	// Reset statistics
	model.totalLookups = 0
	model.oovLookups = 0
	model.hitLookups = 0
	model.fallbackAttempts = 0
	model.fallbackSuccesses = 0
	model.fallbackFailures = 0

	// Test 1: Successful fallback
	paragraph1 := "测试"
	keywords1 := []string{"问题"}
	matcher.FindTopKeywords(paragraph1, keywords1, 1)

	_, _, _, attempts1, successes1, _ := model.GetLookupStats()

	if attempts1 == 0 {
		t.Error("Expected fallback attempt for '测试'")
	}

	if successes1 == 0 {
		t.Error("Expected successful fallback for '测试'")
	}

	// Test 2: Failed fallback (missing character)
	paragraph2 := "没事"
	keywords2 := []string{"问题"}
	matcher.FindTopKeywords(paragraph2, keywords2, 1)

	_, _, _, attempts2, successes2, failures2 := model.GetLookupStats()

	if attempts2 <= attempts1 {
		t.Error("Expected additional fallback attempt for '没事'")
	}

	if failures2 == 0 {
		t.Error("Expected fallback failure for '没事' (missing '事' character)")
	}

	// Verify statistics are cumulative
	if attempts2 != successes2+failures2 {
		t.Errorf("Fallback statistics inconsistent: attempts=%d, successes=%d, failures=%d",
			attempts2, successes2, failures2)
	}

	// Verify logging includes statistics
	hasStatsLog := false
	for _, msg := range logger.messages {
		if strings.Contains(msg, "fallback_attempts") &&
			strings.Contains(msg, "fallback_success_rate") {
			hasStatsLog = true
			break
		}
	}

	if !hasStatsLog {
		t.Error("Expected fallback statistics in logs")
	}
}

// TestFallbackWithEmptyResults tests fallback behavior when no vectors can be generated
// This integration test validates Requirements 1.4
func TestFallbackWithEmptyResults(t *testing.T) {
	// Create test model with no vectors
	model := NewVectorModel(3).(*vectorModel)

	processor := NewTextProcessor()
	calculator := NewSimilarityCalculator()
	logger := &mockLogger{messages: make([]string, 0)}

	// Create matcher
	matcher := NewSemanticMatcherWithLogger(processor, model, calculator, logger, 0.5)

	// Test with OOV words that cannot be resolved via fallback
	paragraph := "完全未知的词"
	keywords := []string{"另一个未知词"}

	results := matcher.FindTopKeywords(paragraph, keywords, 1)

	// Should get empty results
	if len(results) != 0 {
		t.Errorf("Expected empty results when all words are OOV and fallback fails, got %d results", len(results))
	}

	// Verify fallback was attempted but failed
	_, _, _, fallbackAttempts, _, fallbackFailures := model.GetLookupStats()

	if fallbackAttempts == 0 {
		t.Error("Expected fallback attempts")
	}

	if fallbackFailures == 0 {
		t.Error("Expected fallback failures when no character vectors exist")
	}

	// Verify appropriate warning was logged
	hasWarning := false
	for _, msg := range logger.messages {
		if strings.Contains(msg, "WARN") && strings.Contains(msg, "OOV") {
			hasWarning = true
			break
		}
	}

	if !hasWarning {
		t.Error("Expected OOV warning to be logged")
	}
}
