package semanticmatcher

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/go-ego/gse"
)

type Empty struct{}

// textProcessor implements the TextProcessor interface for multilingual text processing
type textProcessor struct {
	seg gse.Segmenter

	chineseStops map[string]Empty
	englishStops map[string]Empty

	englishTokenizer *regexp.Regexp
	mtx              sync.RWMutex
}

// NewTextProcessor creates a new TextProcessor with default configurations
func NewTextProcessor() TextProcessor {
	processor := &textProcessor{
		chineseStops:     defaultChineseStopWords(),
		englishStops:     defaultEnglishStopWords(),
		englishTokenizer: regexp.MustCompile(`\b\w+\b`),
	}

	// Initialize GSE Segmenter
	_ = processor.seg.LoadDict()

	return processor
}

// NewTextProcessorWithConfig creates a TextProcessor with custom stop word dictionaries
func NewTextProcessorWithConfig(chineseStops, englishStops map[string]Empty) TextProcessor {
	processor := &textProcessor{
		chineseStops:     chineseStops,
		englishStops:     englishStops,
		englishTokenizer: regexp.MustCompile(`\b\w+\b`),
	}

	// Initialize GSE segmenter
	_ = processor.seg.LoadDict()

	return processor
}

// NewTextProcessorWithStopWords creates a TextProcessor with stop words loaded from files
func NewTextProcessorWithStopWords(
	chineseStopWordsPath, englishStopWordsPath string,
) (TextProcessor, error) {
	chineseStops := defaultChineseStopWords()
	englishStops := defaultEnglishStopWords()

	// Load Chinese stop words if path is provided
	if chineseStopWordsPath != "" {
		customChineseStops, err := loadStopWordsFromFile(chineseStopWordsPath)
		if err != nil {
			return nil, err
		}
		// Merge with defaults
		for word := range customChineseStops {
			chineseStops[word] = struct{}{}
		}
	}

	// Load English stop words if path is provided
	if englishStopWordsPath != "" {
		customEnglishStops, err := loadStopWordsFromFile(englishStopWordsPath)
		if err != nil {
			return nil, err
		}
		// Merge with defaults
		for word := range customEnglishStops {
			englishStops[word] = struct{}{}
		}
	}

	return NewTextProcessorWithConfig(chineseStops, englishStops), nil
}

// Preprocess segments Chinese text and filters stop words
func (tp *textProcessor) Preprocess(text string) []string {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()

	if strings.TrimSpace(text) == "" {
		return []string{}
	}

	// Detect if text contains Chinese characters
	hasChinese := tp.containsChinese(text)
	hasEnglish := tp.containsEnglish(text)

	var tokens []string

	if hasChinese && hasEnglish {
		// Mixed language processing
		tokens = tp.processMixedText(text)
	} else if hasChinese {
		// Pure Chinese processing
		tokens = tp.processChineseText(text)
	} else {
		// Pure English processing
		tokens = tp.processEnglishText(text)
	}

	return tp.filterTokens(tokens)
}

// PreprocessBatch processes multiple texts efficiently
func (tp *textProcessor) PreprocessBatch(texts []string) [][]string {
	tp.mtx.RLock()
	defer tp.mtx.RUnlock()

	results := make([][]string, len(texts))

	for i, text := range texts {
		results[i] = tp.preprocessInternal(text)
	}

	return results
}

// preprocessInternal is the internal implementation without locking
func (tp *textProcessor) preprocessInternal(text string) []string {
	if strings.TrimSpace(text) == "" {
		return []string{}
	}

	// Detect if text contains Chinese characters
	hasChinese := tp.containsChinese(text)
	hasEnglish := tp.containsEnglish(text)

	var tokens []string

	if hasChinese && hasEnglish {
		// Mixed language processing
		tokens = tp.processMixedText(text)
	} else if hasChinese {
		// Pure Chinese processing
		tokens = tp.processChineseText(text)
	} else {
		// Pure English processing
		tokens = tp.processEnglishText(text)
	}

	return tp.filterTokens(tokens)
}

// containsChinese checks if text contains Chinese characters
func (*textProcessor) containsChinese(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Scripts["Han"], r) {
			return true
		}
	}
	return false
}

// containsEnglish checks if text contains English characters
func (*textProcessor) containsEnglish(text string) bool {
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
}

// processChineseText segments Chinese text using GSE
func (tp *textProcessor) processChineseText(text string) []string {
	segments := tp.seg.Segment([]byte(text))
	tokens := make([]string, 0, len(segments))

	for _, segment := range segments {
		token := strings.TrimSpace(segment.Token().Text())
		if token != "" && !tp.isPunctuation(token) {
			tokens = append(tokens, token)
		}
	}

	return tokens
}

// processEnglishText tokenizes English text using regex
func (tp *textProcessor) processEnglishText(text string) []string {
	matches := tp.englishTokenizer.FindAllString(strings.ToLower(text), -1)
	tokens := make([]string, 0, len(matches))

	for _, match := range matches {
		token := strings.TrimSpace(match)
		if token != "" && !tp.isNumeric(token) {
			tokens = append(tokens, token)
		}
	}

	return tokens
}

// processMixedText handles text with both Chinese and English
func (tp *textProcessor) processMixedText(text string) []string {
	var tokens []string

	// First, segment the entire text using GSE (it can handle mixed text)
	segments := tp.seg.Segment([]byte(text))

	for _, segment := range segments {
		token := strings.TrimSpace(segment.Token().Text())
		if token == "" || tp.isPunctuation(token) {
			continue
		}

		// Check if this token is Chinese or English
		if tp.containsChinese(token) {
			// Chinese token - add directly
			tokens = append(tokens, token)
		} else if tp.containsEnglish(token) {
			// English token - further tokenize with regex and convert to lowercase
			englishTokens := tp.englishTokenizer.FindAllString(strings.ToLower(token), -1)
			for _, englishToken := range englishTokens {
				cleanToken := strings.TrimSpace(englishToken)
				if cleanToken != "" && !tp.isNumeric(cleanToken) {
					tokens = append(tokens, cleanToken)
				}
			}
		}
	}

	return tokens
}

// filterTokens removes stop words and unwanted tokens
func (tp *textProcessor) filterTokens(tokens []string) []string {
	filtered := make([]string, 0, len(tokens))

	for _, token := range tokens {
		// Skip empty tokens
		if token == "" {
			continue
		}

		// Check against stop words
		if tp.containsChinese(token) {
			// Chinese token
			if _, isStop := tp.chineseStops[token]; !isStop {
				filtered = append(filtered, token)
			}
		} else {
			// English token
			if _, isStop := tp.englishStops[strings.ToLower(token)]; !isStop {
				filtered = append(filtered, strings.ToLower(token))
			}
		}
	}

	return filtered
}

// isPunctuation checks if a token is purely punctuation
func (*textProcessor) isPunctuation(token string) bool {
	for _, r := range token {
		if !unicode.IsPunct(r) && !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

// isNumeric checks if a token is purely numeric
func (*textProcessor) isNumeric(token string) bool {
	for _, r := range token {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// defaultChineseStopWords returns a default set of Chinese stop words
func defaultChineseStopWords() map[string]Empty {
	stopWords := []string{
		"的", "了", "在", "是", "我", "有", "和", "就", "不", "人", "都", "一", "一个", "上", "也", "很", "到",
		"说", "要", "去", "会", "着", "没有", "看", "好", "自己", "这", "那", "他", "她", "它", "们", "这个",
		"那个", "什么", "怎么", "为什么", "哪里", "哪个", "多少", "几", "第一", "第二", "可以", "应该", "能够",
		"必须", "需要", "想要", "希望", "觉得", "认为", "知道", "明白", "理解", "记得", "忘记",
	}

	stopWordsMap := make(map[string]Empty)
	for _, word := range stopWords {
		stopWordsMap[word] = Empty{}
	}

	return stopWordsMap
}

// defaultEnglishStopWords returns a default set of English stop words
func defaultEnglishStopWords() map[string]Empty {
	stopWords := []string{
		"a", "an", "and", "are", "as", "at", "be", "by", "for", "from", "has", "he", "in", "is", "it", "its",
		"of", "on", "that", "the", "to", "was", "will", "with", "the", "this", "but", "they", "have", "had",
		"what", "said", "each", "which", "she", "do", "how", "their", "if", "up", "out", "many", "then", "them",
		"these", "so", "some", "her", "would", "make", "like", "into", "him", "time", "two", "more", "go", "no",
		"way", "could", "my", "than", "first", "been", "call", "who", "oil", "sit", "now", "find", "down", "day",
		"did", "get", "come", "made", "may", "part", "i",
	}

	stopWordsMap := make(map[string]Empty)
	for _, word := range stopWords {
		stopWordsMap[word] = Empty{}
	}

	return stopWordsMap
}

// loadStopWordsFromFile loads stop words from a text file (one word per line)
func loadStopWordsFromFile(path string) (map[string]struct{}, error) {
	file, err := os.Open(path) //nolint:gosec
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stopWords := make(map[string]struct{})
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word != "" && !strings.HasPrefix(word, "#") { // Skip empty lines and comments
			stopWords[word] = struct{}{}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return stopWords, nil
}
