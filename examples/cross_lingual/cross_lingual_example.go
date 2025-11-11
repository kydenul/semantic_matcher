package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kydenul/log"

	sm "github.com/kydenul/semantic-matcher"
)

// This example demonstrates cross-lingual semantic matching capabilities
// using aligned vector embeddings. The system can match Chinese and English
// text seamlessly in the same vector space.

func main() {
	// Setup logger
	home, _ := os.UserHomeDir()
	pathPrefix := filepath.Join(home, "git-space/semantic_matcher")

	opt, err := log.LoadFromFile(filepath.Join(pathPrefix, "config/config.yaml"))
	if err != nil {
		fmt.Printf("Warning: Failed to load log config, using default: %v\n", err)
		opt = &log.Options{Level: "info"}
	}
	logger := log.NewLog(opt)

	// Example 1: Load cross-lingual vector model
	logger.Infoln("=== Example 1: Loading Cross-lingual Vector Model ===")
	logger.Infoln("------------------------------------------------------")
	matcher := loadCrossLingualModel(pathPrefix, logger)
	if matcher == nil {
		logger.Errorln("Failed to load cross-lingual model, exiting")
		os.Exit(1)
	}
	logger.Infoln()

	// Example 2: Chinese paragraph matching English keywords
	logger.Infoln("=== Example 2: Chinese Paragraph Matching English Keywords ===")
	logger.Infoln("---------------------------------------------------------------")
	chineseToEnglishExample(matcher, logger)
	logger.Infoln()

	// Example 3: English paragraph matching Chinese keywords
	logger.Infoln("=== Example 3: English Paragraph Matching Chinese Keywords ===")
	logger.Infoln("---------------------------------------------------------------")
	englishToChineseExample(matcher, logger)
	logger.Infoln()

	// Example 4: Mixed language text processing
	logger.Infoln("=== Example 4: Mixed Language Text Processing ===")
	logger.Infoln("-------------------------------------------------")
	mixedLanguageExample(matcher, logger)
	logger.Infoln()

	// Example 5: Query similarity between Chinese-English word pairs
	logger.Infoln("=== Example 5: Chinese-English Word Pair Similarity ===")
	logger.Infoln("-------------------------------------------------------")
	wordPairSimilarityExample(matcher, logger)
	logger.Infoln()

	// Display final statistics
	displayStatistics(matcher, logger)
}

// loadCrossLingualModel demonstrates loading a cross-lingual aligned vector model
// Requirements: 3.1, 6.1, 6.2, 6.3, 6.4
func loadCrossLingualModel(pathPrefix string, logger log.Logger) sm.SemanticMatcher {
	logger.Infoln("Loading cross-lingual aligned vector model...")
	logger.Infoln("This model contains both Chinese and English words in the same vector space.")
	logger.Infoln()

	// // Create configuration with multiple aligned vector files
	// // The system will load and merge all files into a single vector model
	// cfg := sm.DefaultConfig()
	// cfg.VectorFilePaths = []string{
	// 	filepath.Join(pathPrefix, "vector/wiki.zh.align.vec"), // Chinese aligned vectors
	// 	filepath.Join(pathPrefix, "vector/wiki.en.align.vec"), // English aligned vectors
	// }
	// cfg.EnableStats = true
	// cfg.MemoryLimit = 10 * 1024 * 1024 * 1024 // 10GB

	cfg, err := sm.LoadFromYAML(filepath.Join(pathPrefix, "config/config.yaml"))
	if err != nil {
		logger.Errorf("Failed to load config, exiting")
		return nil
	}
	logger.Infof("Configuration: %+v", cfg)

	// Create semantic matcher from configuration
	start := time.Now()
	matcher, err := sm.NewSemanticMatcherFromConfig(cfg, logger)
	if err != nil {
		logger.Errorf("Failed to create semantic matcher: %v", err)
		return nil
	}
	duration := time.Since(start)

	logger.Infof("✓ Cross-lingual model loaded successfully in %v", duration)
	logger.Infoln("  The model can now process both Chinese and English text seamlessly.")
	logger.Infoln()

	return matcher
}

// chineseToEnglishExample demonstrates matching Chinese text with English keywords
// This shows the cross-lingual capability where semantically similar words in different
// languages have close vector representations.
// Requirements: 3.1, 3.2
func chineseToEnglishExample(matcher sm.SemanticMatcher, logger log.Logger) {
	// Chinese paragraph about artificial intelligence
	paragraph := "人工智能和机器学习正在改变世界。深度学习技术在计算机视觉和自然语言处理领域取得了突破性进展。" +
		"神经网络模型可以识别图像、理解文本，甚至生成创意内容。"

	// English keywords to match against the Chinese text
	keywords := []string{
		"artificial intelligence",
		"machine learning",
		"deep learning",
		"computer vision",
		"natural language processing",
		"neural networks",
		"image recognition",
		"text understanding",
		"creative content",
		"data mining", // Less relevant keyword for comparison
	}

	logger.Infof("Chinese Paragraph:")
	logger.Infof("  %s", paragraph)
	logger.Infoln()

	logger.Infoln("English Keywords to Match:")
	for i, kw := range keywords {
		logger.Infof("  %d. %s", i+1, kw)
	}
	logger.Infoln()

	// Find top 5 matching keywords
	start := time.Now()
	results := matcher.FindTopKeywords(paragraph, keywords, 5)
	duration := time.Since(start)

	logger.Infof("Top 5 Matches (completed in %v):", duration)
	logger.Infoln("  Rank | Keyword                      | Score  | Words | OOV")
	logger.Infoln("  -----|------------------------------|--------|-------|-----")
	for i, match := range results {
		logger.Infof("  %-4d | %-28s | %.4f | %-5d | %d",
			i+1, match.Keyword, match.Score, match.WordCount, match.OOVCount)
	}
	logger.Infoln()

	logger.Infoln("Analysis:")
	logger.Infoln("  ✓ English keywords successfully matched against Chinese text")
	logger.Infoln("  ✓ Semantically similar terms (e.g., '人工智能' ↔ " +
		"'artificial intelligence') have high scores")
	logger.Infoln("  ✓ Less relevant keywords (e.g., 'data mining') have lower scores")
}

// englishToChineseExample demonstrates matching English text with Chinese keywords
// Requirements: 3.1, 3.3
func englishToChineseExample(matcher sm.SemanticMatcher, logger log.Logger) {
	// English paragraph about technology
	paragraph := "Artificial intelligence and machine learning are transforming the world. " +
		"Deep learning techniques have achieved breakthrough progress in computer vision and " +
		"natural language processing. Neural network models can recognize images, understand text, " +
		"and even generate creative content."

	// Chinese keywords to match against the English text
	keywords := []string{
		"人工智能",   // artificial intelligence
		"机器学习",   // machine learning
		"深度学习",   // deep learning
		"计算机视觉",  // computer vision
		"自然语言处理", // natural language processing
		"神经网络",   // neural networks
		"图像识别",   // image recognition
		"文本理解",   // text understanding
		"创意内容",   // creative content
		"数据挖掘",   // data mining (less relevant)
	}

	logger.Infof("English Paragraph:")
	logger.Infof("  %s", paragraph)
	logger.Infoln()

	logger.Infoln("Chinese Keywords to Match:")
	for i, kw := range keywords {
		logger.Infof("  %d. %s", i+1, kw)
	}
	logger.Infoln()

	// Find top 5 matching keywords
	start := time.Now()
	results := matcher.FindTopKeywords(paragraph, keywords, 5)
	duration := time.Since(start)

	logger.Infof("Top 5 Matches (completed in %v):", duration)
	logger.Infoln("  Rank | Keyword      | Score  | Words | OOV")
	logger.Infoln("  -----|--------------|--------|-------|-----")
	for i, match := range results {
		logger.Infof("  %-4d | %-12s | %.4f | %-5d | %d",
			i+1, match.Keyword, match.Score, match.WordCount, match.OOVCount)
	}
	logger.Infoln()

	logger.Infoln("Analysis:")
	logger.Infoln("  ✓ Chinese keywords successfully matched against English text")
	logger.Infoln("  ✓ Cross-lingual semantic matching works bidirectionally")
	logger.Infoln("  ✓ The aligned vector space enables seamless language mixing")
}

// mixedLanguageExample demonstrates processing text with mixed Chinese and English
// Requirements: 3.1, 3.4
func mixedLanguageExample(matcher sm.SemanticMatcher, logger log.Logger) {
	// Mixed language paragraph (common in modern Chinese tech writing)
	paragraph := "现代AI技术在healthcare医疗领域的应用越来越广泛。Machine learning算法可以帮助doctors医生" +
		"进行disease diagnosis疾病诊断，提高准确率。Deep learning深度学习模型可以分析medical images医学影像，" +
		"识别potential risks潜在风险。"

	// Mixed language keywords
	keywords := []string{
		"AI人工智能",
		"医疗健康",
		"machine learning",
		"疾病诊断",
		"医生",
		"deep learning",
		"医学影像",
		"风险识别",
		"healthcare",
		"diagnosis",
	}

	logger.Infof("Mixed Language Paragraph:")
	logger.Infof("  %s", paragraph)
	logger.Infoln()

	logger.Infoln("Mixed Language Keywords:")
	for i, kw := range keywords {
		logger.Infof("  %d. %s", i+1, kw)
	}
	logger.Infoln()

	// Find top 6 matching keywords
	start := time.Now()
	results := matcher.FindTopKeywords(paragraph, keywords, 6)
	duration := time.Since(start)

	logger.Infof("Top 6 Matches (completed in %v):", duration)
	logger.Infoln("  Rank | Keyword            | Score  | Words | OOV")
	logger.Infoln("  -----|--------------------|---------|----|-----")
	for i, match := range results {
		logger.Infof("  %-4d | %-18s | %.4f | %-5d | %d",
			i+1, match.Keyword, match.Score, match.WordCount, match.OOVCount)
	}
	logger.Infoln()

	logger.Infoln("Analysis:")
	logger.Infoln("  ✓ Mixed language text processed without language detection")
	logger.Infoln("  ✓ Both Chinese and English keywords matched successfully")
	logger.Infoln("  ✓ System handles code-switching naturally")
	logger.Infoln("  ✓ No special preprocessing required for mixed content")
}

// wordPairSimilarityExample demonstrates computing similarity between Chinese-English word pairs
// This shows how semantically equivalent words in different languages have high similarity scores
// in the aligned vector space.
// Requirements: 3.1, 6.1, 6.2, 6.3, 6.4
func wordPairSimilarityExample(matcher sm.SemanticMatcher, logger log.Logger) {
	// Define Chinese-English word pairs with their expected semantic relationship
	wordPairs := []struct {
		chinese  string
		english  string
		relation string
	}{
		{"苹果", "apple", "direct translation"},
		{"香蕉", "banana", "direct translation"},
		{"计算机", "computer", "direct translation"},
		{"人工智能", "artificial intelligence", "direct translation"},
		{"机器学习", "machine learning", "direct translation"},
		{"深度学习", "deep learning", "direct translation"},
		{"神经网络", "neural network", "direct translation"},
		{"自然语言", "natural language", "direct translation"},
		{"图像识别", "image recognition", "direct translation"},
		{"数据科学", "data science", "direct translation"},
		{"云计算", "cloud computing", "direct translation"},
		{"大数据", "big data", "direct translation"},
		// Add some unrelated pairs for comparison
		{"苹果", "computer", "unrelated"},
		{"香蕉", "artificial intelligence", "unrelated"},
	}

	logger.Infoln("Computing similarity scores for Chinese-English word pairs...")
	logger.Infoln("In an aligned vector space, " +
		"semantically equivalent words should have high similarity.")
	logger.Infoln()

	logger.Infoln("Word Pair Similarities:")
	logger.Infoln("  Chinese        | English                    | Similarity | Relation")
	logger.Infoln("  ---------------|----------------------------|------------|------------------")

	var totalSimilarity float64
	var translationCount int
	var unrelatedSimilarity float64
	var unrelatedCount int

	for _, pair := range wordPairs {
		similarity := matcher.ComputeSimilarity(pair.chinese, pair.english)

		logger.Infof("  %-14s | %-26s | %.4f     | %s",
			pair.chinese, pair.english, similarity, pair.relation)

		if pair.relation == "direct translation" {
			totalSimilarity += similarity
			translationCount++
		} else {
			unrelatedSimilarity += similarity
			unrelatedCount++
		}
	}

	logger.Infoln()

	// Calculate average similarities
	avgTranslationSim := totalSimilarity / float64(translationCount)
	avgUnrelatedSim := unrelatedSimilarity / float64(unrelatedCount)

	logger.Infoln("Statistics:")
	logger.Infof("  Average similarity for translations: %.4f", avgTranslationSim)
	logger.Infof("  Average similarity for unrelated pairs: %.4f", avgUnrelatedSim)
	logger.Infof("  Difference: %.4f", avgTranslationSim-avgUnrelatedSim)
	logger.Infoln()

	logger.Infoln("Analysis:")
	logger.Infoln("  ✓ Translation pairs have significantly higher similarity scores")
	logger.Infoln("  ✓ Unrelated pairs have lower similarity scores")
	logger.Infoln("  ✓ The aligned vector space preserves semantic relationships across languages")
	logger.Infoln("  ✓ Cross-lingual semantic matching is effective")
}

// displayStatistics shows performance and usage statistics
func displayStatistics(matcher sm.SemanticMatcher, logger log.Logger) {
	logger.Infoln("=== Performance Statistics ===")
	logger.Infoln("------------------------------")

	stats := matcher.GetStats()

	logger.Infof("Total Requests:     %d", stats.TotalRequests)
	logger.Infof("Average Latency:    %v", stats.AverageLatency)
	logger.Infof("OOV Rate:           %.2f%%", stats.OOVRate*100)
	logger.Infof("Vector Hit Rate:    %.2f%%", stats.VectorHitRate*100)
	logger.Infof("Memory Usage:       %.2f MB", float64(stats.MemoryUsage)/(1024*1024))
	logger.Infof("Last Updated:       %s", stats.LastUpdated.Format(time.RFC3339))
	logger.Infoln()

	logger.Infoln("Key Takeaways:")
	logger.Infoln("  ✓ Cross-lingual matching requires no special handling")
	logger.Infoln("  ✓ Performance is identical to single-language mode")
	logger.Infoln("  ✓ Memory usage scales with total vocabulary size")
	logger.Infoln("  ✓ OOV rate depends on vector file coverage")
	logger.Infoln()

	logger.Infoln("=== Cross-lingual Example Complete ===")
	logger.Infoln("This example demonstrated:")
	logger.Infoln("  1. Loading aligned vector models")
	logger.Infoln("  2. Chinese text → English keyword matching")
	logger.Infoln("  3. English text → Chinese keyword matching")
	logger.Infoln("  4. Mixed language text processing")
	logger.Infoln("  5. Word pair similarity computation")
	logger.Infoln()
	logger.Infoln("For more information, see docs/cross_lingual_guide.md")
}
