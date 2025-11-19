package semanticmatcher

import (
	"io"
	"time"
)

// TextProcessor handles multilingual text processing including Chinese word segmentation
// and English tokenization with stop word filtering
type TextProcessor interface {
	// Preprocess segments Chinese text and filters stop words
	Preprocess(text string) []string

	// PreprocessBatch processes multiple texts efficiently
	PreprocessBatch(texts []string) [][]string
}

// VectorModel provides in-memory storage and retrieval of word vectors
type VectorModel interface {
	// GetVector retrieves vector for a single word
	GetVector(word string) ([]float32, bool)

	// GetAverageVector computes mean pooling for multiple words
	GetAverageVector(words []string) ([]float32, bool)

	// Dimension returns the vector dimension
	Dimension() int

	// VocabularySize returns total number of words in model
	VocabularySize() int

	// MemoryUsage returns estimated memory usage in bytes
	MemoryUsage() int64

	// GetOOVRate returns the rate of out-of-vocabulary lookups (0.0 to 1.0)
	GetOOVRate() float64

	// GetVectorHitRate returns the rate of successful vector lookups (0.0 to 1.0)
	GetVectorHitRate() float64

	// GetLookupStats returns detailed lookup statistics (total, oov, hit, fallback attempts, successes, failures)
	GetLookupStats() (totalLookups, oovLookups, hitLookups, fallbackAttempts, fallbackSuccesses, fallbackFailures int64)

	// GetFallbackSuccessRate returns the success rate of character-level fallback operations (0.0 to 1.0)
	GetFallbackSuccessRate() float64

	// ResetStats resets all statistics counters
	ResetStats()
}

// SimilarityCalculator computes similarity scores between vectors
type SimilarityCalculator interface {
	// CosineSimilarity computes cosine similarity between two vectors
	// Returns 0.0 for invalid inputs (empty, nil, mismatched dimensions, or zero vectors)
	CosineSimilarity(v1, v2 []float32) float64

	// BatchSimilarity computes similarities between one vector and many
	// Returns empty slice for invalid query, and 0.0 for invalid candidates
	BatchSimilarity(query []float32, candidates [][]float32) []float64
}

// SemanticMatcher orchestrates the complete semantic matching pipeline
type SemanticMatcher interface {
	// FindTopKeywords finds most similar keywords to paragraph
	FindTopKeywords(paragraph string, keywords []string, k int) []KeywordMatch

	// ComputeSimilarity computes similarity between two texts
	ComputeSimilarity(text1, text2 string) float64

	// GetStats returns performance and usage statistics
	GetStats() MatcherStats
}

// KeywordMatch represents a keyword with its similarity score
type KeywordMatch struct {
	Keyword   string  `json:"keyword"`
	Score     float64 `json:"score"`
	WordCount int     `json:"word_count"` // Number of words in keyword
	OOVCount  int     `json:"oov_count"`  // Number of OOV words
}

// MatcherStats provides performance and usage statistics
type MatcherStats struct {
	TotalRequests  int64         `json:"total_requests"`
	AverageLatency time.Duration `json:"average_latency"`
	OOVRate        float64       `json:"oov_rate"`
	VectorHitRate  float64       `json:"vector_hit_rate"`
	MemoryUsage    int64         `json:"memory_usage_bytes"`
	LastUpdated    time.Time     `json:"last_updated"`
}

// EmbeddingLoader handles loading and parsing of pre-trained word vector files
type EmbeddingLoader interface {
	// LoadFromFile loads vectors from .vec text format
	LoadFromFile(path string) (VectorModel, error)

	// LoadFromReader loads vectors from any io.Reader
	LoadFromReader(reader io.Reader) (VectorModel, error)

	// LoadMultipleFiles loads vectors from multiple .vec files and merges them into a single model
	// All files must have the same vector dimension, otherwise ErrDimensionMismatch is returned
	// If duplicate words exist across files, later files will overwrite earlier ones
	LoadMultipleFiles(paths []string) (VectorModel, error)

	// SetProgressCallback sets a callback for progress reporting during loading
	SetProgressCallback(callback ProgressCallback)
}

// ProgressCallback is called during vector loading to report progress
type ProgressCallback func(loaded, total int, memoryUsage int64)

// Logger interface for configurable logging
type Logger interface {
	Debug(args ...any)
	Info(args ...any)
	Warn(args ...any)
	Error(args ...any)

	Debugf(template string, args ...any)
	Infof(template string, args ...any)
	Warnf(template string, args ...any)
	Errorf(template string, args ...any)
}
