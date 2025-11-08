package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kydenul/log"

	sm "github.com/kydenul/semantic-matcher"
)

var Logger log.Logger

func main() {
	home, _ := os.UserHomeDir()

	pathPrefix := filepath.Join(home + "/git-space/semantic_matcher")
	log.Debugf("pathPrefix: %s", pathPrefix)

	opt, err := log.LoadFromFile(filepath.Join(pathPrefix, "config/config.yaml"))
	if err != nil {
		panic(fmt.Sprintf("Failed to load log config from file: %v", err))
	}
	Logger = log.NewLog(opt)

	cfg := sm.DefaultConfig()
	cfg.VectorFilePath = filepath.Join(pathPrefix, "vector/cc.zh.300.vec")

	smer, err := sm.NewSemanticMatcherFromConfig(cfg, Logger)
	if err != nil {
		log.Error("Failed to create semantic matcher: %v", err)
		os.Exit(1)
	}

	runDemo(smer, 5)
}

func runDemo(matcher sm.SemanticMatcher, topK int) {
	log.Info("=== Demo Mode ===")

	// Example 1: Chinese text matching
	log.Infoln("Example 1: Chinese Semantic Matching")
	log.Infoln("-------------------------------------")

	chineseParagraph := "人工智能和机器学习正在改变世界。深度学习技术在图像识别和自然语言处理领域取得了突破性进展。"
	chineseKeywords := []string{
		"人工智能",
		"深度学习",
		"计算机视觉",
		"自然语言处理",
		"机器学习",
		"数据挖掘",
		"神经网络",
		"大数据",
	}

	log.Infof("Paragraph: %s\n\n", chineseParagraph)
	log.Infoln("Keywords to match:")
	for _, kw := range chineseKeywords {
		log.Infof("  - %s\n", kw)
	}

	start := time.Now()
	results := matcher.FindTopKeywords(chineseParagraph, chineseKeywords, topK)
	duration := time.Since(start)

	log.Infof("Top %d matches (completed in %v):\n", topK, duration)
	for i, match := range results {
		log.Infof("  %d. %s (score: %.4f, words: %d, OOV: %d)\n",
			i+1, match.Keyword, match.Score, match.WordCount, match.OOVCount)
	}

	// Example 2: English text matching
	log.Infoln("Example 2: English Semantic Matching")
	log.Infoln("-------------------------------------")

	englishParagraph := "Artificial intelligence and machine learning are transforming the world. " +
		"Deep learning techniques have achieved breakthrough progress in computer vision and natural language processing."
	englishKeywords := []string{
		"artificial intelligence",
		"deep learning",
		"computer vision",
		"natural language processing",
		"machine learning",
		"data mining",
		"neural networks",
		"big data",
	}

	log.Infof("Paragraph: %s\n\n", englishParagraph)
	log.Infoln("Keywords to match:")
	for _, kw := range englishKeywords {
		log.Infof("  - %s\n", kw)
	}
	log.Infoln()

	start = time.Now()
	results = matcher.FindTopKeywords(englishParagraph, englishKeywords, topK)
	duration = time.Since(start)

	log.Infof("Top %d matches (completed in %v):\n", topK, duration)
	for i, match := range results {
		log.Infof("  %d. %s (score: %.4f, words: %d, OOV: %d)\n",
			i+1, match.Keyword, match.Score, match.WordCount, match.OOVCount)
	}
	log.Infoln()

	// Example 3: Mixed Chinese-English text
	log.Infoln("Example 3: Mixed Chinese-English Text")
	log.Infoln("--------------------------------------")

	mixedParagraph := "AI人工智能技术在healthcare医疗领域的应用越来越广泛。Machine learning机器学习算法可以帮助doctors医生进行disease diagnosis疾病诊断。"
	mixedKeywords := []string{
		"人工智能AI",
		"医疗健康",
		"machine learning",
		"疾病诊断",
		"医生",
		"算法",
	}

	log.Infof("Paragraph: %s\n\n", mixedParagraph)
	log.Infoln("Keywords to match:")
	for _, kw := range mixedKeywords {
		log.Infof("  - %s\n", kw)
	}
	log.Infoln()

	start = time.Now()
	results = matcher.FindTopKeywords(mixedParagraph, mixedKeywords, topK)
	duration = time.Since(start)

	log.Infof("Top %d matches (completed in %v):\n", topK, duration)
	for i, match := range results {
		log.Infof("  %d. %s (score: %.4f, words: %d, OOV: %d)\n",
			i+1, match.Keyword, match.Score, match.WordCount, match.OOVCount)
	}
	log.Infoln()

	// Example 4: Text similarity comparison
	log.Infoln("Example 4: Text Similarity Comparison")
	log.Infoln("--------------------------------------")

	text1 := "机器学习是人工智能的一个重要分支"
	text2 := "深度学习是机器学习的一个子领域"
	text3 := "今天天气很好，适合出去散步"

	log.Infof("Text 1: %s\n", text1)
	log.Infof("Text 2: %s\n", text2)
	log.Infof("Text 3: %s\n\n", text3)

	sim12 := matcher.ComputeSimilarity(text1, text2)
	sim13 := matcher.ComputeSimilarity(text1, text3)
	sim23 := matcher.ComputeSimilarity(text2, text3)

	log.Infof("Similarity(Text1, Text2): %.4f\n", sim12)
	log.Infof("Similarity(Text1, Text3): %.4f\n", sim13)
	log.Infof("Similarity(Text2, Text3): %.4f\n", sim23)
	log.Infoln()

	// Display statistics
	log.Infoln("=== Performance Statistics ===")
	stats := matcher.GetStats()
	log.Infof("Total Requests:    %d\n", stats.TotalRequests)
	log.Infof("Average Latency:   %v\n", stats.AverageLatency)
	log.Infof("OOV Rate:          %.2f%%\n", stats.OOVRate*100)
	log.Infof("Vector Hit Rate:   %.2f%%\n", stats.VectorHitRate*100)
	log.Infof("Memory Usage:      %.2f MB\n", float64(stats.MemoryUsage)/(1024*1024))
	log.Infof("Last Updated:      %s\n", stats.LastUpdated.Format(time.RFC3339))
	log.Infoln()
}
