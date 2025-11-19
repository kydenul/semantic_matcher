package semanticmatcher

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNewTextProcessor(t *testing.T) {
	processor := NewTextProcessor()
	if processor == nil {
		t.Fatal("NewTextProcessor() returned nil")
	}
}

func TestTextProcessor_Preprocess_Chinese(t *testing.T) {
	processor := NewTextProcessor()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Simple Chinese text",
			input:    "我爱中国",
			expected: []string{"爱", "中国"},
		},
		{
			name:     "Chinese with stop words",
			input:    "我是一个学生",
			expected: []string{"学生"}, // "我", "是", "一个" are stop words
		},
		{
			name:     "Chinese with punctuation",
			input:    "你好，世界！",
			expected: []string{"你好", "世界"},
		},
		{
			name:     "Empty Chinese text",
			input:    "",
			expected: []string{},
		},
		{
			name:     "Chinese with spaces",
			input:    "  北京 大学  ",
			expected: []string{"北京", "大学"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.Preprocess(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Preprocess() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestTextProcessor_Preprocess_English(t *testing.T) {
	processor := NewTextProcessor()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Simple English text",
			input:    "Hello world",
			expected: []string{"hello", "world"},
		},
		{
			name:     "English with stop words",
			input:    "The quick brown fox",
			expected: []string{"quick", "brown", "fox"}, // "the" is a stop word
		},
		{
			name:     "English with punctuation",
			input:    "Hello, world!",
			expected: []string{"hello", "world"},
		},
		{
			name:     "English with numbers",
			input:    "I have 123 apples",
			expected: []string{"apples"}, // "I", "have" are stop words, "123" is numeric
		},
		{
			name:     "Mixed case English",
			input:    "Machine Learning",
			expected: []string{"machine", "learning"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.Preprocess(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Preprocess() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestTextProcessor_Preprocess_Mixed(t *testing.T) {
	processor := NewTextProcessor()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:  "Chinese and English mixed",
			input: "我喜欢machine learning",
			expected: []string{
				"machine",
				"learning",
			}, // "我" and "喜欢" might be filtered as stop words
		},
		{
			name:     "English and Chinese mixed",
			input:    "Hello 世界",
			expected: []string{"hello", "世界"},
		},
		{
			name:     "Complex mixed text",
			input:    "人工智能 AI technology 很有趣",
			expected: []string{"人工智能", "ai", "technology", "有趣"}, // "很" is a stop word
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.Preprocess(tt.input)
			// For mixed language tests, we'll check if result contains expected words
			// since segmentation might vary
			for _, expected := range tt.expected {
				found := false
				for _, actual := range result {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected word '%s' not found in result %v", expected, result)
				}
			}
		})
	}
}

func TestTextProcessor_Preprocess_EdgeCases(t *testing.T) {
	processor := NewTextProcessor()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "Only spaces",
			input:    "   ",
			expected: []string{},
		},
		{
			name:     "Only punctuation",
			input:    "!@#$%^&*()",
			expected: []string{},
		},
		{
			name:     "Only numbers",
			input:    "123 456 789",
			expected: []string{},
		},
		{
			name:     "Special characters",
			input:    "测试@#$%test",
			expected: []string{"测试", "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.Preprocess(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Preprocess() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestTextProcessor_PreprocessBatch(t *testing.T) {
	processor := NewTextProcessor()

	inputs := []string{
		"Hello world",
		"你好世界",
		"Machine learning 机器学习",
		"",
		"123 numbers only",
	}

	results := processor.PreprocessBatch(inputs)

	if len(results) != len(inputs) {
		t.Errorf("PreprocessBatch() returned %d results, expected %d", len(results), len(inputs))
	}

	// Test that batch processing gives same results as individual processing
	for i, input := range inputs {
		individual := processor.Preprocess(input)
		batch := results[i]

		if !reflect.DeepEqual(individual, batch) {
			t.Errorf("Batch result[%d] = %v, individual result = %v", i, batch, individual)
		}
	}
}

func TestTextProcessor_StopWordFiltering(t *testing.T) {
	// Test with custom stop words
	chineseStops := map[string]Empty{
		"测试": {},
		"的":  {},
	}
	englishStops := map[string]Empty{
		"test": {},
		"the":  {},
	}

	processor := NewTextProcessorWithConfig(chineseStops, englishStops)

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Custom Chinese stop words",
			input:    "这是测试的文本",
			expected: []string{"这是", "文本"}, // "测试" and "的" should be filtered
		},
		{
			name:     "Custom English stop words",
			input:    "This is a test text",
			expected: []string{"this", "is", "a", "text"}, // "test" and "the" should be filtered
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.Preprocess(tt.input)
			// Check that stop words are not present
			for _, token := range result {
				if token == "测试" || token == "的" || token == "test" || token == "the" {
					t.Errorf("Stop word '%s' found in result %v", token, result)
				}
			}
		})
	}
}

func TestTextProcessor_LanguageDetection(t *testing.T) {
	processor := NewTextProcessor().(*textProcessor)

	tests := []struct {
		name       string
		input      string
		hasChinese bool
		hasEnglish bool
	}{
		{
			name:       "Pure Chinese",
			input:      "你好世界",
			hasChinese: true,
			hasEnglish: false,
		},
		{
			name:       "Pure English",
			input:      "Hello world",
			hasChinese: false,
			hasEnglish: true,
		},
		{
			name:       "Mixed languages",
			input:      "Hello 世界",
			hasChinese: true,
			hasEnglish: true,
		},
		{
			name:       "Numbers only",
			input:      "123456",
			hasChinese: false,
			hasEnglish: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasChinese := processor.containsChinese(tt.input)
			hasEnglish := processor.containsEnglish(tt.input)

			if hasChinese != tt.hasChinese {
				t.Errorf("containsChinese() = %v, expected %v", hasChinese, tt.hasChinese)
			}
			if hasEnglish != tt.hasEnglish {
				t.Errorf("containsEnglish() = %v, expected %v", hasEnglish, tt.hasEnglish)
			}
		})
	}
}

func TestTextProcessor_ConcurrentAccess(t *testing.T) {
	processor := NewTextProcessor()

	// Test concurrent access to ensure thread safety
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Each goroutine processes different text
			texts := []string{
				"Hello world",
				"你好世界",
				"Machine learning",
				"人工智能",
			}

			for j := 0; j < 100; j++ {
				text := texts[j%len(texts)]
				result := processor.Preprocess(text)
				if result == nil {
					t.Errorf("Goroutine %d: Preprocess returned nil", id)
					return
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestNewTextProcessorWithDictPaths(t *testing.T) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Test with valid dictionary paths
	dictPaths := []string{
		filepath.Join(cwd, "vector/dict/zh/t_1.txt"),
		filepath.Join(cwd, "vector/dict/zh/s_1.txt"),
	}

	// Check if dictionary files exist
	for _, path := range dictPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Skipf("Dictionary file not found: %s, skipping test", path)
		}
	}

	processor, err := NewTextProcessorWithDictPaths(dictPaths)
	if err != nil {
		t.Fatalf("Failed to create text processor with dict paths: %v", err)
	}

	if processor == nil {
		t.Fatal("Expected non-nil processor")
	}

	// Test preprocessing with custom dictionary
	text := "人工智能和机器学习正在改变世界"
	tokens := processor.Preprocess(text)

	if len(tokens) == 0 {
		t.Error("Expected non-empty tokens")
	}

	t.Logf("Preprocessed tokens: %v", tokens)
}

func TestNewTextProcessorWithDictPaths_EmptyPaths(t *testing.T) {
	processor, err := NewTextProcessorWithDictPaths([]string{})
	if err == nil {
		t.Error("Expected error for empty dict paths")
	}

	if processor != nil {
		t.Error("Expected nil processor for empty dict paths")
	}
}

func TestNewTextProcessorWithDictPaths_InvalidPath(t *testing.T) {
	dictPaths := []string{
		"/nonexistent/path/dict.txt",
	}

	processor, err := NewTextProcessorWithDictPaths(dictPaths)
	if err == nil {
		t.Error("Expected error for invalid dict path")
	}

	if processor != nil {
		t.Error("Expected nil processor for invalid dict path")
	}
}

func TestConfigWithDictPaths(t *testing.T) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Create config with dict paths
	config := DefaultConfig()
	config.VectorFilePaths = []string{
		filepath.Join(cwd, "vector/wiki.zh.align.reduced.vec"),
	}
	config.DictPaths = []string{
		filepath.Join(cwd, "vector/dict/zh/t_1.txt"),
		filepath.Join(cwd, "vector/dict/zh/s_1.txt"),
	}

	// Check if files exist
	for _, path := range config.VectorFilePaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Skipf("Vector file not found: %s, skipping test", path)
		}
	}
	for _, path := range config.DictPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Skipf("Dictionary file not found: %s, skipping test", path)
		}
	}

	// Validate config
	err = Validate(config)
	if err != nil {
		t.Fatalf("Config validation failed: %v", err)
	}

	t.Log("Config with DictPaths validated successfully")
}

func TestConfigWithInvalidDictPaths(t *testing.T) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Create config with invalid dict paths
	config := DefaultConfig()
	config.VectorFilePaths = []string{
		filepath.Join(cwd, "vector/wiki.zh.align.reduced.vec"),
	}
	config.DictPaths = []string{
		"/nonexistent/path/dict.txt",
	}

	// Validate config - should fail
	err = Validate(config)
	if err == nil {
		t.Error("Expected validation error for invalid dict paths")
	}
}

func TestNewTextProcessorWithDictPathsAndStopWords(t *testing.T) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Test with valid dictionary paths and stop words
	dictPaths := []string{
		filepath.Join(cwd, "vector/dict/zh/t_1.txt"),
		filepath.Join(cwd, "vector/dict/zh/s_1.txt"),
	}

	chineseStopWordsPath := filepath.Join(cwd, "vector/dict/zh/stop_word.txt")
	englishStopWordsPath := ""

	// Check if files exist
	for _, path := range dictPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Skipf("Dictionary file not found: %s, skipping test", path)
		}
	}
	if _, err := os.Stat(chineseStopWordsPath); os.IsNotExist(err) {
		t.Skipf("Stop words file not found: %s, skipping test", chineseStopWordsPath)
	}

	processor, err := NewTextProcessorWithDictPathsAndStopWords(
		dictPaths,
		chineseStopWordsPath,
		englishStopWordsPath,
	)
	if err != nil {
		t.Fatalf("Failed to create text processor with dict paths and stop words: %v", err)
	}

	if processor == nil {
		t.Fatal("Expected non-nil processor")
	}

	// Test preprocessing with custom dictionary and stop words
	text := "人工智能和机器学习正在改变世界"
	tokens := processor.Preprocess(text)

	if len(tokens) == 0 {
		t.Error("Expected non-empty tokens")
	}

	t.Logf("Preprocessed tokens: %v", tokens)

	// Verify that stop words are filtered
	// "和" should be filtered as a stop word
	for _, token := range tokens {
		if token == "和" {
			t.Error("Stop word '和' should have been filtered")
		}
	}
}

func TestNewTextProcessorWithDictPathsAndStopWords_EmptyDictPaths(t *testing.T) {
	processor, err := NewTextProcessorWithDictPathsAndStopWords(
		[]string{},
		"",
		"",
	)
	if err == nil {
		t.Error("Expected error for empty dict paths")
	}

	if processor != nil {
		t.Error("Expected nil processor for empty dict paths")
	}
}

func TestConfigWithDictPathsAndStopWords(t *testing.T) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Create config with both dict paths and stop words
	config := DefaultConfig()
	config.VectorFilePaths = []string{
		filepath.Join(cwd, "vector/wiki.zh.align.reduced.vec"),
	}
	config.DictPaths = []string{
		filepath.Join(cwd, "vector/dict/zh/t_1.txt"),
		filepath.Join(cwd, "vector/dict/zh/s_1.txt"),
	}
	config.ChineseStopWords = filepath.Join(cwd, "vector/dict/zh/stop_word.txt")

	// Check if files exist
	for _, path := range config.VectorFilePaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Skipf("Vector file not found: %s, skipping test", path)
		}
	}
	for _, path := range config.DictPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Skipf("Dictionary file not found: %s, skipping test", path)
		}
	}
	if _, err := os.Stat(config.ChineseStopWords); os.IsNotExist(err) {
		t.Skipf("Stop words file not found: %s, skipping test", config.ChineseStopWords)
	}

	// Validate config
	err = Validate(config)
	if err != nil {
		t.Fatalf("Config validation failed: %v", err)
	}

	t.Log("Config with DictPaths and StopWords validated successfully")
}

func TestNewTextProcessorWithDictPathsAndStopWords_Comparison(t *testing.T) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	dictPaths := []string{
		filepath.Join(cwd, "vector/dict/zh/t_1.txt"),
		filepath.Join(cwd, "vector/dict/zh/s_1.txt"),
	}

	// Check if files exist
	for _, path := range dictPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Skipf("Dictionary file not found: %s, skipping test", path)
		}
	}

	// Create processor with only dict paths
	processorDictOnly, err := NewTextProcessorWithDictPaths(dictPaths)
	if err != nil {
		t.Fatalf("Failed to create processor with dict paths: %v", err)
	}

	// Create processor with dict paths and default stop words
	processorWithStopWords, err := NewTextProcessorWithDictPathsAndStopWords(
		dictPaths,
		"", // Use default Chinese stop words
		"", // Use default English stop words
	)
	if err != nil {
		t.Fatalf("Failed to create processor with dict paths and stop words: %v", err)
	}

	// Test text with stop words
	text := "人工智能和机器学习正在改变世界"

	tokensDictOnly := processorDictOnly.Preprocess(text)
	tokensWithStopWords := processorWithStopWords.Preprocess(text)

	t.Logf("Tokens with dict only: %v", tokensDictOnly)
	t.Logf("Tokens with dict and stop words: %v", tokensWithStopWords)

	// Both should work, but may have different results due to stop word filtering
	if len(tokensDictOnly) == 0 {
		t.Error("Expected non-empty tokens from dict-only processor")
	}
	if len(tokensWithStopWords) == 0 {
		t.Error("Expected non-empty tokens from processor with stop words")
	}
}
