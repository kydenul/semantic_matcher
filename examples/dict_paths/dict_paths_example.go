package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kydenul/log"
	sm "github.com/kydenul/semantic-matcher"
)

// This example demonstrates how to use custom dictionary paths
// for improved Chinese word segmentation

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

	logger.Infoln("=== Custom Dictionary Paths Example ===")
	logger.Infoln("========================================")
	logger.Infoln()

	// Example 1: Using default embedded dictionary
	logger.Infoln("Example 1: Default Embedded Dictionary")
	logger.Infoln("---------------------------------------")
	defaultExample(logger)
	logger.Infoln()

	// Example 2: Using custom dictionary paths
	logger.Infoln("Example 2: Custom Dictionary Paths")
	logger.Infoln("-----------------------------------")
	customDictExample(pathPrefix, logger)
	logger.Infoln()

	// Example 3: Loading from YAML config with DictPaths
	logger.Infoln("Example 3: Loading from YAML Config")
	logger.Infoln("------------------------------------")
	yamlConfigExample(pathPrefix, logger)
	logger.Infoln()
}

func defaultExample(logger log.Logger) {
	// Create text processor with default embedded dictionary
	processor := sm.NewTextProcessor()

	testText := "人工智能和机器学习正在改变世界"
	tokens := processor.Preprocess(testText)

	logger.Infof("Input text: %s", testText)
	logger.Infof("Tokens: %v", tokens)
	logger.Infof("Token count: %d", len(tokens))
}

func customDictExample(pathPrefix string, logger log.Logger) {
	// Define custom dictionary paths
	dictPaths := []string{
		filepath.Join(pathPrefix, "vector/dict/zh/t_1.txt"),
		filepath.Join(pathPrefix, "vector/dict/zh/s_1.txt"),
	}

	logger.Infof("Loading custom dictionaries:")
	for i, path := range dictPaths {
		logger.Infof("  %d. %s", i+1, path)
	}
	logger.Infoln()

	// Create text processor with custom dictionaries only
	processor, err := sm.NewTextProcessorWithDictPaths(dictPaths)
	if err != nil {
		logger.Errorf("Failed to create processor with custom dictionaries: %v", err)
		return
	}

	testText := "人工智能和机器学习正在改变世界"
	tokens := processor.Preprocess(testText)

	logger.Infof("Input text: %s", testText)
	logger.Infof("Tokens (dict only): %v", tokens)
	logger.Infof("Token count: %d", len(tokens))
	logger.Infoln()

	// Now create processor with both custom dictionaries and stop words
	chineseStopWordsPath := filepath.Join(pathPrefix, "vector/dict/zh/stop_word.txt")
	logger.Infof("Loading custom dictionaries with stop words:")
	logger.Infof("  Stop words file: %s", chineseStopWordsPath)
	logger.Infoln()

	processorWithStopWords, err := sm.NewTextProcessorWithDictPathsAndStopWords(
		dictPaths,
		chineseStopWordsPath,
		"",
	)
	if err != nil {
		logger.Errorf("Failed to create processor with dictionaries and stop words: %v", err)
		return
	}

	tokensWithStopWords := processorWithStopWords.Preprocess(testText)
	logger.Infof("Tokens (dict + stop words): %v", tokensWithStopWords)
	logger.Infof("Token count: %d", len(tokensWithStopWords))
	logger.Infoln()

	logger.Infoln("Benefits of custom dictionaries:")
	logger.Infoln("  ✓ Better domain-specific word segmentation")
	logger.Infoln("  ✓ Support for specialized terminology")
	logger.Infoln("  ✓ Improved accuracy for specific use cases")
	logger.Infoln()
	logger.Infoln("Benefits of combining with stop words:")
	logger.Infoln("  ✓ DictPaths controls word segmentation")
	logger.Infoln("  ✓ StopWords controls filtering")
	logger.Infoln("  ✓ Both can be used together for maximum control")
}

func yamlConfigExample(pathPrefix string, logger log.Logger) {
	// Load configuration from YAML file
	// The config file includes dict_paths configuration
	configPath := filepath.Join(pathPrefix, "config/config.yaml")
	logger.Infof("Loading configuration from: %s", configPath)
	logger.Infoln()

	cfg, err := sm.LoadFromYAML(configPath)
	if err != nil {
		logger.Errorf("Failed to load config: %v", err)
		return
	}

	logger.Infof("Configuration loaded successfully:")
	logger.Infof("  Vector files: %d", len(cfg.VectorFilePaths))
	logger.Infof("  Dictionary paths: %d", len(cfg.DictPaths))
	logger.Infof("  Supported languages: %v", cfg.SupportedLanguages)
	logger.Infoln()

	if len(cfg.DictPaths) > 0 {
		logger.Infoln("Custom dictionaries configured:")
		for i, path := range cfg.DictPaths {
			logger.Infof("  %d. %s", i+1, path)
		}
		logger.Infoln()
	}

	// Create semantic matcher from config
	matcher, err := sm.NewSemanticMatcherFromConfig(cfg, logger)
	if err != nil {
		logger.Errorf("Failed to create semantic matcher: %v", err)
		return
	}

	// Test with sample text
	paragraph := "深度学习技术在计算机视觉领域取得了突破性进展"
	keywords := []string{
		"人工智能",
		"机器学习",
		"深度学习",
		"计算机视觉",
		"自然语言处理",
	}

	logger.Infof("Test paragraph: %s", paragraph)
	logger.Infoln()
	logger.Infoln("Keywords to match:")
	for i, kw := range keywords {
		logger.Infof("  %d. %s", i+1, kw)
	}
	logger.Infoln()

	results := matcher.FindTopKeywords(paragraph, keywords, 3)

	logger.Infoln("Top 3 matches:")
	logger.Infoln("  Rank | Keyword      | Score  | Words | OOV")
	logger.Infoln("  -----|--------------|--------|-------|-----")
	for i, match := range results {
		logger.Infof("  %-4d | %-12s | %.4f | %-5d | %d",
			i+1, match.Keyword, match.Score, match.WordCount, match.OOVCount)
	}
	logger.Infoln()

	logger.Infoln("Summary:")
	logger.Infoln("  ✓ Configuration loaded from YAML")
	logger.Infoln("  ✓ Custom dictionaries applied automatically")
	logger.Infoln("  ✓ Semantic matching works with custom segmentation")
}
