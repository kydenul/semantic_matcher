package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kydenul/log"

	sm "github.com/kydenul/semantic-matcher"
)

// This example demonstrates the character-level fallback feature for handling OOV (Out-Of-Vocabulary) words
// The system now automatically uses character-level fallback when encountering OOV words

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

	logger.Infoln("=== OOV Handling with Character-level Fallback ===")
	logger.Infoln("==================================================")
	logger.Infoln("This example demonstrates the automatic character-level fallback feature.")
	logger.Infoln("When a word is not in the vocabulary (OOV), the system automatically:")
	logger.Infoln("  1. Splits the word into individual characters")
	logger.Infoln("  2. Looks up each character's vector")
	logger.Infoln("  3. Computes the average of character vectors")
	logger.Infoln("  4. Uses this average as an approximation of the word's meaning")
	logger.Infoln()

	// Load configuration
	configPath := filepath.Join(pathPrefix, "config/config.yaml")
	cfg, err := sm.LoadFromYAML(configPath)
	if err != nil {
		logger.Errorf("Failed to load config: %v", err)
		return
	}

	// Create semantic matcher
	matcher, err := sm.NewSemanticMatcherFromConfig(cfg, logger)
	if err != nil {
		logger.Errorf("Failed to create semantic matcher: %v", err)
		return
	}

	logger.Infoln("Scenario 1: OOV Word with Character-level Fallback")
	logger.Infoln("---------------------------------------------------")
	testOOVWordWithFallback(matcher, logger)
	logger.Infoln()

	logger.Infoln("Scenario 2: Mixed OOV and Known Words")
	logger.Infoln("--------------------------------------")
	testMixedWords(matcher, logger)
	logger.Infoln()

	logger.Infoln("Scenario 3: Fallback Statistics")
	logger.Infoln("--------------------------------")
	testFallbackStatistics(matcher, logger)
	logger.Infoln()

	logger.Infoln("Scenario 4: Real-world Example with '没事'")
	logger.Infoln("------------------------------------------")
	testRealWorldExample(matcher, logger)
	logger.Infoln()
}

func testOOVWordWithFallback(matcher sm.SemanticMatcher, logger log.Logger) {
	// Test with OOV words that will use character-level fallback
	paragraph := "没事"
	keywords := []string{
		"问题",
		"帮助",
		"查询",
		"正常",
	}

	logger.Infof("Test paragraph: '%s' (likely OOV)", paragraph)
	logger.Infoln("Keywords:")
	for i, kw := range keywords {
		logger.Infof("  %d. %s", i+1, kw)
	}
	logger.Infoln()

	logger.Infoln("Processing with automatic character-level fallback...")
	results := matcher.FindTopKeywords(paragraph, keywords, 3)

	if len(results) == 0 {
		logger.Warnln("❌ No matches found - fallback may have failed")
		logger.Infoln("This could happen if individual characters are also not in vocabulary")
	} else {
		logger.Infoln("✅ Matches found via character-level fallback:")
		for i, match := range results {
			logger.Infof("  %d. %s - Score: %.4f (Words: %d, OOV: %d)",
				i+1, match.Keyword, match.Score, match.WordCount, match.OOVCount)
		}
		logger.Infoln()
		logger.Infoln("Note: Even though '没事' is OOV, the system split it into '没' and '事'")
		logger.Infoln("      and computed an approximate vector from the character vectors.")
	}
}

func testMixedWords(matcher sm.SemanticMatcher, logger log.Logger) {
	// Test with mixed OOV and known words
	paragraph := "没事，我想查询一下业务问题"
	keywords := []string{
		"业务查询",
		"问题咨询",
		"帮助中心",
		"技术支持",
	}

	logger.Infof("Test paragraph: %s", paragraph)
	logger.Infoln("Keywords:")
	for i, kw := range keywords {
		logger.Infof("  %d. %s", i+1, kw)
	}
	logger.Infoln()

	results := matcher.FindTopKeywords(paragraph, keywords, 3)

	if len(results) > 0 {
		logger.Infoln("✅ Matches found (combining known words and fallback):")
		for i, match := range results {
			logger.Infof("  %d. %s - Score: %.4f (Words: %d, OOV: %d)",
				i+1, match.Keyword, match.Score, match.WordCount, match.OOVCount)
		}
		logger.Infoln()
		logger.Infoln("Note: The system used both direct lookups for known words like '查询', '业务', '问题'")
		logger.Infoln("      and character-level fallback for OOV words like '没事'")
	}
}

func testFallbackStatistics(matcher sm.SemanticMatcher, logger log.Logger) {
	// Get the vector model to access statistics
	// Note: In a real application, you would get this from the matcher
	logger.Infoln("Character-level fallback provides detailed statistics:")
	logger.Infoln()

	// Perform some operations to generate statistics
	texts := []string{"没事", "测试", "查询问题"}
	keywords := []string{"问题", "帮助", "支持"}

	for _, text := range texts {
		matcher.FindTopKeywords(text, keywords, 2)
	}

	logger.Infoln("Statistics tracked:")
	logger.Infoln("  • Total lookups: Number of word vector lookups")
	logger.Infoln("  • OOV lookups: Number of words not found in vocabulary")
	logger.Infoln("  • Hit lookups: Number of words found directly")
	logger.Infoln("  • Fallback attempts: Number of times fallback was triggered")
	logger.Infoln("  • Fallback successes: Number of successful fallbacks")
	logger.Infoln("  • Fallback failures: Number of failed fallbacks")
	logger.Infoln()
	logger.Infoln("You can access these statistics via model.GetLookupStats()")
	logger.Infoln("Example:")
	logger.Infoln("  total, oov, hit, attempts, successes, failures := model.GetLookupStats()")
	logger.Infoln("  successRate := float64(successes) / float64(attempts)")
}

func testRealWorldExample(matcher sm.SemanticMatcher, logger log.Logger) {
	logger.Infoln("Real-world scenario: Customer service chatbot")
	logger.Infoln()

	// Simulate customer queries with OOV words
	customerQuery := "没事，我就是想问一下"
	intentKeywords := []string{
		"咨询问题",
		"投诉建议",
		"查询订单",
		"技术支持",
		"闲聊",
	}

	logger.Infof("Customer query: '%s'", customerQuery)
	logger.Infoln("Intent keywords:")
	for i, kw := range intentKeywords {
		logger.Infof("  %d. %s", i+1, kw)
	}
	logger.Infoln()

	results := matcher.FindTopKeywords(customerQuery, intentKeywords, 3)

	if len(results) > 0 {
		logger.Infoln("✅ Intent classification results:")
		for i, match := range results {
			logger.Infof("  %d. %s - Confidence: %.4f",
				i+1, match.Keyword, match.Score)
		}
		logger.Infoln()
		logger.Infof("Detected intent: %s", results[0].Keyword)
		logger.Infoln()
		logger.Infoln("Benefits of character-level fallback:")
		logger.Infoln("  ✓ Handles colloquial expressions like '没事'")
		logger.Infoln("  ✓ Provides approximate semantic matching for OOV words")
		logger.Infoln("  ✓ Improves coverage without expanding vocabulary")
		logger.Infoln("  ✓ Works automatically without configuration")
	}

	logger.Infoln()
	logger.Infoln("Summary:")
	logger.Infoln("  ✅ Character-level fallback is now enabled by default")
	logger.Infoln("  ✅ Automatically handles OOV words like '没事'")
	logger.Infoln("  ✅ Provides detailed statistics for monitoring")
	logger.Infoln("  ✅ No configuration required - works out of the box")
	logger.Infoln()
	logger.Infoln("For more information, see: docs/oov_handling_guide.md")
}
