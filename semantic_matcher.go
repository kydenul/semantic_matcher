package semanticmatcher

import (
	"slices"
	"sort"
	"sync"
	"time"
)

type semanticMatcher struct {
	processor    TextProcessor
	model        VectorModel
	calculator   SimilarityCalculator
	stats        *MatcherStats
	logger       Logger
	oovThreshold float64 // Threshold for logging OOV warnings (e.g., 0.5 = 50%)
	mtx          sync.RWMutex
}

// NewSemanticMatcher creates a new SemanticMatcher instance
func NewSemanticMatcher(
	processor TextProcessor,
	model VectorModel,
	calculator SimilarityCalculator,
) SemanticMatcher {
	return &semanticMatcher{
		processor:    processor,
		model:        model,
		calculator:   calculator,
		logger:       DiscardLogger{},
		oovThreshold: 0.5, // Default: warn if 50% or more words are OOV
		stats: &MatcherStats{
			LastUpdated: time.Now(),
		},
	}
}

// NewSemanticMatcherWithLogger creates a new SemanticMatcher instance with custom logger
func NewSemanticMatcherWithLogger(
	processor TextProcessor,
	model VectorModel,
	calculator SimilarityCalculator,
	logger Logger,
	oovThreshold float64,
) SemanticMatcher {
	return &semanticMatcher{
		processor:    processor,
		model:        model,
		calculator:   calculator,
		logger:       logger,
		oovThreshold: oovThreshold,
		stats: &MatcherStats{
			LastUpdated: time.Now(),
		},
	}
}

// NewSemanticMatcherFromConfig creates a new SemanticMatcher from a configuration
// This is the recommended way to initialize a SemanticMatcher with all components
func NewSemanticMatcherFromConfig(config *Config, logger Logger) (SemanticMatcher, error) {
	if config == nil {
		return nil, ErrInvalidConfiguration
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	// Initialize text processor with stop words and custom dictionaries
	var processor TextProcessor
	var err error

	// Determine which processor to create based on configuration
	// DictPaths and StopWords can be used together
	hasCustomDict := len(config.DictPaths) > 0
	hasCustomStopWords := config.ChineseStopWords != "" || config.EnglishStopWords != ""

	if hasCustomDict && hasCustomStopWords {
		// Use both custom dictionaries and custom stop words
		logger.Infof(
			"Loading text processor with custom dictionaries and stop words, "+
				"dict_count: %d, dict_paths: %v, chinese_stopwords: %s, english_stopwords: %s",
			len(
				config.DictPaths,
			),
			config.DictPaths,
			config.ChineseStopWords,
			config.EnglishStopWords,
		)
		processor, err = NewTextProcessorWithDictPathsAndStopWords(
			config.DictPaths,
			config.ChineseStopWords,
			config.EnglishStopWords,
		)
		if err != nil {
			logger.Errorf(
				"Failed to load custom dictionaries and stop words, using default processor, error: %v",
				err,
			)
			processor = NewTextProcessor()
		} else {
			logger.Infof("Custom dictionaries and stop words loaded successfully")
		}
	} else if hasCustomDict {
		// Use only custom dictionary paths
		logger.Infof("Loading text processor with custom dictionaries, dict_count: %d, paths: %v",
			len(config.DictPaths), config.DictPaths)
		processor, err = NewTextProcessorWithDictPaths(config.DictPaths)
		if err != nil {
			logger.Errorf("Failed to load custom dictionaries, using default processor, error: %v", err)
			processor = NewTextProcessor()
		} else {
			logger.Infof("Custom dictionaries loaded successfully")
		}
	} else if hasCustomStopWords {
		// Use only custom stop words
		logger.Infof("Loading text processor with custom stop words, "+
			"chinese_stopwords: %s, english_stopwords: %s",
			config.ChineseStopWords, config.EnglishStopWords)
		processor, err = NewTextProcessorWithStopWords(
			config.ChineseStopWords,
			config.EnglishStopWords,
		)
		if err != nil {
			logger.Errorf("Failed to load stop words, using default processor, error: %v", err)
			processor = NewTextProcessor()
		} else {
			logger.Infof("Custom stop words loaded successfully")
		}
	} else {
		// Use default processor
		logger.Infof("Loading text processor with default configuration")
		processor = NewTextProcessor()
	}

	// Initialize embedding loader
	loader := NewEmbeddingLoader(logger)

	// Load vector model from file(s)
	logger.Infof("Loading vector model, file_count: %d, paths: %v",
		len(config.VectorFilePaths), config.VectorFilePaths)

	// Load vector model using multi-file loading (supports single or multiple files)
	model, err := loader.LoadMultipleFiles(config.VectorFilePaths)
	if err != nil {
		logger.Errorf(
			"Failed to load vector model, error: %v, file_count: %d, paths: %v",
			err, len(config.VectorFilePaths), config.VectorFilePaths)
		return nil, err
	}

	// Log detailed information about loaded model
	logger.Infof("Vector model loaded successfully, file_count: %d, "+
		"vocabulary_size: %d, dimension: %d, memory_mb: %.2f",
		len(config.VectorFilePaths), model.VocabularySize(),
		model.Dimension(), float64(model.MemoryUsage())/(1024*1024))

	// Check memory limit
	if config.MemoryLimit > 0 {
		memUsage := model.MemoryUsage()
		if memUsage > config.MemoryLimit {
			logger.Warnf(
				"Memory usage exceeds limit, usage_bytes: %d, limit_bytes: %d, usage_mb: %.2f, limit_mb: %.2f",
				memUsage,
				config.MemoryLimit,
				float64(memUsage)/(1024*1024),
				float64(config.MemoryLimit)/(1024*1024),
			)
			return nil, ErrMemoryLimitExceeded
		}
		logger.Infof("Memory usage within limit, usage_mb: %.2f, limit_mb: %.2f",
			float64(memUsage)/(1024*1024), float64(config.MemoryLimit)/(1024*1024))
	}

	// Initialize similarity calculator
	calculator := NewSimilarityCalculator()

	// Determine OOV threshold (use default if not specified)
	oovThreshold := 0.5
	if config.EnableStats {
		// Use a more aggressive threshold when stats are enabled
		oovThreshold = 0.3
	}

	// Create semantic matcher
	matcher := &semanticMatcher{
		processor:    processor,
		model:        model,
		calculator:   calculator,
		logger:       logger,
		oovThreshold: oovThreshold,
		stats: &MatcherStats{
			LastUpdated: time.Now(),
		},
	}

	logger.Infof(
		"SemanticMatcher initialized successfully, file_count: %d, vector_dimension: %d, vocabulary_size: %d, "+
			"memory_usage_mb: %.2f, supported_languages: %v",
		len(config.VectorFilePaths),
		model.Dimension(),
		model.VocabularySize(),
		float64(model.MemoryUsage())/(1024*1024),
		config.SupportedLanguages,
	)

	return matcher, nil
}

// validateConfig validates the configuration parameters
func validateConfig(config *Config) error {
	if len(config.VectorFilePaths) == 0 {
		return ErrNoVectorFiles
	}

	// Verify all vector files are non-empty strings
	if slices.Contains(config.VectorFilePaths, "") {
		return ErrInvalidConfiguration
	}

	if config.MaxSequenceLen <= 0 {
		return ErrInvalidConfiguration
	}

	if config.MemoryLimit < 0 {
		return ErrInvalidConfiguration
	}

	if len(config.SupportedLanguages) == 0 {
		return ErrInvalidConfiguration
	}

	// Validate supported languages
	for _, lang := range config.SupportedLanguages {
		if lang != "zh" && lang != "en" {
			return ErrUnsupportedLanguage
		}
	}

	return nil
}

// FindTopKeywords finds most similar keywords to paragraph
// Returns at most k results sorted by similarity score in descending order
// If k <= 0, returns all results
func (sm *semanticMatcher) FindTopKeywords(
	paragraph string,
	keywords []string,
	k int,
) []KeywordMatch {
	sm.logger.Debugf("FindTopKeywords called, paragraph_length: %d, keywords_count: %d, k: %d",
		len(paragraph), len(keywords), k)

	// Handle empty inputs
	startTime := time.Now()
	if paragraph == "" || len(keywords) == 0 {
		sm.updateStats(time.Since(startTime), 0, 0)
		sm.logger.Warnf("Empty input provided, paragraph_empty: %v, keywords_empty: %v",
			paragraph == "", len(keywords) == 0)
		return []KeywordMatch{}
	}

	// Preprocess paragraph
	preprocessStart := time.Now()
	paragraphTokens := sm.processor.Preprocess(paragraph)
	preprocessDuration := time.Since(preprocessStart)
	sm.logger.Debugf("Paragraph preprocessing completed, tokens_count: %d, duration_ms: %d",
		len(paragraphTokens), preprocessDuration.Milliseconds())
	if len(paragraphTokens) == 0 {
		sm.updateStats(time.Since(startTime), 0, 0)
		sm.logger.Warnf("No valid tokens after preprocessing paragraph")
		return []KeywordMatch{}
	}

	// Get paragraph vector using mean pooling
	vectorizeStart := time.Now()
	paragraphVector, ok := sm.model.GetAverageVector(paragraphTokens)
	vectorizeDuration := time.Since(vectorizeStart)
	if !ok {
		// All words are OOV
		sm.updateStats(time.Since(startTime), len(paragraphTokens), len(paragraphTokens))
		sm.logger.Warnf("All paragraph words are OOV, token_count: %d", len(paragraphTokens))
		return make([]KeywordMatch, 0)
	}

	// Count OOV words in paragraph
	paragraphOOVCount := 0
	for _, token := range paragraphTokens {
		if _, exists := sm.model.GetVector(token); !exists {
			paragraphOOVCount++
		}
	}

	paragraphOOVRate := float64(paragraphOOVCount) / float64(len(paragraphTokens))
	sm.logger.Debugf(
		"Paragraph vectorization completed, duration_ms: %d, oov_count: %d, oov_rate: %.4f",
		vectorizeDuration.Milliseconds(),
		paragraphOOVCount,
		paragraphOOVRate,
	)

	// Warn if OOV rate exceeds threshold
	if paragraphOOVRate >= sm.oovThreshold {
		sm.logger.Warnf(
			"High OOV rate in paragraph, oov_rate: %.4f, oov_count: %d, total_tokens: %d",
			paragraphOOVRate,
			paragraphOOVCount,
			len(paragraphTokens),
		)
	}

	// Process each keyword and compute similarity
	matches := make([]KeywordMatch, 0, len(keywords))
	totalKeywordTokens := 0
	totalKeywordOOV := 0

	similarityStart := time.Now()
	for _, keyword := range keywords {
		// Preprocess keyword
		keywordTokens := sm.processor.Preprocess(keyword)
		if len(keywordTokens) == 0 {
			// Empty keyword after preprocessing
			matches = append(matches, KeywordMatch{
				Keyword:   keyword,
				Score:     0.0,
				WordCount: 0,
				OOVCount:  0,
			})
			continue
		}

		totalKeywordTokens += len(keywordTokens)

		// Get keyword vector using mean pooling
		var score float64
		var oovCount int
		keywordVector, ok := sm.model.GetAverageVector(keywordTokens)
		if !ok {
			// All words are OOV
			score = 0.0
			oovCount = len(keywordTokens)
			totalKeywordOOV += oovCount
		} else {
			// Compute cosine similarity
			score = sm.calculator.CosineSimilarity(paragraphVector, keywordVector)

			// Count OOV words
			for _, token := range keywordTokens {
				if _, exists := sm.model.GetVector(token); !exists {
					oovCount++
				}
			}
			totalKeywordOOV += oovCount
		}

		matches = append(matches, KeywordMatch{
			Keyword:   keyword,
			Score:     score,
			WordCount: len(keywordTokens),
			OOVCount:  oovCount,
		})
	}
	similarityDuration := time.Since(similarityStart)

	// Sort by similarity score in descending order
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	// Limit results if k > 0
	if k > 0 && k < len(matches) {
		matches = matches[:k]
	}

	totalDuration := time.Since(startTime)
	totalTokens := len(paragraphTokens) + totalKeywordTokens
	totalOOV := paragraphOOVCount + totalKeywordOOV

	sm.updateStats(totalDuration, totalTokens, totalOOV)

	overallOOVRate := 0.0
	if totalTokens > 0 {
		overallOOVRate = float64(totalOOV) / float64(totalTokens)
	}

	// Get fallback statistics
	_, _, _, fallbackAttempts, fallbackSuccesses, fallbackFailures := sm.model.GetLookupStats()
	fallbackSuccessRate := 0.0
	if fallbackAttempts > 0 {
		fallbackSuccessRate = float64(fallbackSuccesses) / float64(fallbackAttempts)
	}

	// Log detailed fallback information at Debug level
	if fallbackAttempts > 0 {
		sm.logger.Debugf(
			"Character-level fallback used in FindTopKeywords, fallback_attempts: %d, "+
				"fallback_successes: %d, fallback_failures: %d, fallback_success_rate: %.4f",
			fallbackAttempts,
			fallbackSuccesses,
			fallbackFailures,
			fallbackSuccessRate,
		)
	}

	sm.logger.Infof(
		"FindTopKeywords completed, total_duration_ms: %d, preprocess_duration_ms: %d, "+
			"vectorize_duration_ms: %d, similarity_duration_ms: %d, keywords_processed: %d, "+
			"results_returned: %d, total_tokens: %d, total_oov: %d, oov_rate: %.4f, "+
			"fallback_attempts: %d, fallback_success_rate: %.4f",
		totalDuration.Milliseconds(),
		preprocessDuration.Milliseconds(),
		vectorizeDuration.Milliseconds(),
		similarityDuration.Milliseconds(),
		len(keywords),
		len(matches),
		totalTokens,
		totalOOV,
		overallOOVRate,
		fallbackAttempts,
		fallbackSuccessRate,
	)

	// Warn if overall OOV rate is high
	if overallOOVRate >= sm.oovThreshold {
		sm.logger.Warnf(
			"High overall OOV rate detected, oov_rate: %.4f, total_oov: %d, total_tokens: %d",
			overallOOVRate, totalOOV, totalTokens)
	}

	return matches
}

// ComputeSimilarity computes similarity between two texts
// Returns a value between 0.0 and 1.0 (cosine similarity is normalized to [0, 1])
func (sm *semanticMatcher) ComputeSimilarity(text1, text2 string) float64 {
	startTime := time.Now()

	sm.logger.Debugf("ComputeSimilarity called, text1_length: %d, text2_length: %d",
		len(text1), len(text2))

	// Handle empty inputs
	if text1 == "" || text2 == "" {
		sm.updateStats(time.Since(startTime), 0, 0)
		sm.logger.Debugf("Empty input provided, text1_empty: %v, text2_empty: %v",
			text1 == "", text2 == "")
		return 0.0
	}

	// Preprocess both texts
	preprocessStart := time.Now()
	tokens1 := sm.processor.Preprocess(text1)
	tokens2 := sm.processor.Preprocess(text2)
	preprocessDuration := time.Since(preprocessStart)

	if len(tokens1) == 0 || len(tokens2) == 0 {
		sm.updateStats(time.Since(startTime), 0, 0)
		sm.logger.Warnf("No valid tokens after preprocessing, tokens1_count: %d, tokens2_count: %d",
			len(tokens1), len(tokens2))
		return 0.0
	}

	// Get vectors using mean pooling
	vectorizeStart := time.Now()
	vector1, ok1 := sm.model.GetAverageVector(tokens1)
	vector2, ok2 := sm.model.GetAverageVector(tokens2)
	vectorizeDuration := time.Since(vectorizeStart)

	// Count OOV words
	oov1, oov2 := 0, 0
	for _, token := range tokens1 {
		if _, exists := sm.model.GetVector(token); !exists {
			oov1++
		}
	}
	for _, token := range tokens2 {
		if _, exists := sm.model.GetVector(token); !exists {
			oov2++
		}
	}

	totalTokens := len(tokens1) + len(tokens2)
	totalOOV := oov1 + oov2

	if !ok1 || !ok2 {
		// One or both texts have all OOV words
		sm.updateStats(time.Since(startTime), totalTokens, totalOOV)
		sm.logger.Warnf(
			"All words are OOV in one or both texts, text1_all_oov: %v, text2_all_oov: %v",
			!ok1,
			!ok2,
		)
		return 0.0
	}

	// Compute cosine similarity
	similarityStart := time.Now()
	similarity := sm.calculator.CosineSimilarity(vector1, vector2)
	similarityDuration := time.Since(similarityStart)

	// Normalize to [0, 1] range (cosine similarity is in [-1, 1])
	// For semantic matching, we typically care about positive similarity
	if similarity < 0 {
		similarity = 0
	}

	totalDuration := time.Since(startTime)
	sm.updateStats(totalDuration, totalTokens, totalOOV)

	oovRate := float64(totalOOV) / float64(totalTokens)

	// Get fallback statistics
	_, _, _, fallbackAttempts, fallbackSuccesses, fallbackFailures := sm.model.GetLookupStats()
	fallbackSuccessRate := 0.0
	if fallbackAttempts > 0 {
		fallbackSuccessRate = float64(fallbackSuccesses) / float64(fallbackAttempts)
	}

	// Log detailed fallback information at Debug level
	if fallbackAttempts > 0 {
		sm.logger.Debugf(
			"Character-level fallback used in ComputeSimilarity, fallback_attempts: %d, "+
				"fallback_successes: %d, fallback_failures: %d, fallback_success_rate: %.4f",
			fallbackAttempts,
			fallbackSuccesses,
			fallbackFailures,
			fallbackSuccessRate,
		)
	}

	sm.logger.Infof(
		"ComputeSimilarity completed, total_duration_ms: %d, preprocess_duration_ms: %d, "+
			"vectorize_duration_ms: %d, similarity_duration_ms: %d, tokens1_count: %d, tokens2_count: %d, "+
			"oov1_count: %d, oov2_count: %d, oov_rate: %.4f, similarity_score: %.4f, "+
			"fallback_attempts: %d, fallback_success_rate: %.4f",
		totalDuration.Milliseconds(),
		preprocessDuration.Milliseconds(),
		vectorizeDuration.Milliseconds(),
		similarityDuration.Milliseconds(),
		len(tokens1),
		len(tokens2),
		oov1,
		oov2,
		oovRate,
		similarity,
		fallbackAttempts,
		fallbackSuccessRate,
	)

	// Warn if OOV rate is high
	if oovRate >= sm.oovThreshold {
		sm.logger.Warnf("High OOV rate detected, oov_rate: %.4f, total_oov: %d, total_tokens: %d",
			oovRate, totalOOV, totalTokens)
	}

	return similarity
}

// GetStats returns performance and usage statistics
func (sm *semanticMatcher) GetStats() MatcherStats {
	sm.mtx.RLock()
	defer sm.mtx.RUnlock()

	// Update stats with current model statistics
	sm.stats.OOVRate = sm.model.GetOOVRate()
	sm.stats.VectorHitRate = sm.model.GetVectorHitRate()
	sm.stats.MemoryUsage = sm.model.MemoryUsage()
	sm.stats.LastUpdated = time.Now()

	sm.logger.Debugf("Statistics retrieved, total_requests: %d, average_latency_ms: %d, "+
		"oov_rate: %.4f, vector_hit_rate: %.4f, memory_usage_mb: %.2f",
		sm.stats.TotalRequests, sm.stats.AverageLatency.Milliseconds(),
		sm.stats.OOVRate, sm.stats.VectorHitRate, float64(sm.stats.MemoryUsage)/(1024*1024))

	// Return a copy to prevent external modification
	return MatcherStats{
		TotalRequests:  sm.stats.TotalRequests,
		AverageLatency: sm.stats.AverageLatency,
		OOVRate:        sm.stats.OOVRate,
		VectorHitRate:  sm.stats.VectorHitRate,
		MemoryUsage:    sm.stats.MemoryUsage,
		LastUpdated:    sm.stats.LastUpdated,
	}
}

// updateStats updates the internal statistics
func (sm *semanticMatcher) updateStats(latency time.Duration, totalTokens, oovTokens int) {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()

	sm.logger.Debugf("Total tokens, total_tokens: %d, oov_tokens: %d", totalTokens, oovTokens)

	sm.stats.TotalRequests++

	// Update average latency using incremental average formula
	// new_avg = old_avg + (new_value - old_avg) / count
	if sm.stats.TotalRequests == 1 {
		sm.stats.AverageLatency = latency
	} else {
		delta := latency - sm.stats.AverageLatency
		sm.stats.AverageLatency += delta / time.Duration(sm.stats.TotalRequests)
	}

	// Log performance metrics if logger is available
	if sm.stats.TotalRequests%100 == 0 {
		// Log every 100 requests for monitoring
		sm.logger.Infof("Performance statistics, total_requests: %d, average_latency_ms: %d, "+
			"current_oov_rate: %.4f, current_hit_rate: %.4f, memory_usage_mb: %.2f",
			sm.stats.TotalRequests, sm.stats.AverageLatency.Milliseconds(),
			sm.model.GetOOVRate(), sm.model.GetVectorHitRate(),
			float64(sm.model.MemoryUsage())/(1024*1024))
	}
}
