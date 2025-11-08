package semanticmatcher

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cast"
)

// embeddingLoader implements the EmbeddingLoader interface
type embeddingLoader struct {
	logger           Logger
	progressCallback ProgressCallback
}

// NewEmbeddingLoader creates a new EmbeddingLoader instance
func NewEmbeddingLoader(logger Logger) EmbeddingLoader {
	return &embeddingLoader{
		logger:           logger,
		progressCallback: nil,
	}
}

// SetProgressCallback sets a callback for progress reporting during loading
func (el *embeddingLoader) SetProgressCallback(callback ProgressCallback) {
	el.progressCallback = callback
}

// LoadFromFile loads vectors from .vec text format file
func (el *embeddingLoader) LoadFromFile(path string) (VectorModel, error) {
	el.logger.Infof("Loading vector file, path: %s", path)

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, ErrVectorFileNotFound
	}

	// Open file
	file, err := os.Open(path) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("failed to open vector file: %w", err)
	}
	defer file.Close()

	return el.LoadFromReader(file)
}

// LoadFromReader loads vectors from any io.Reader
//
//nolint:cyclop
func (el *embeddingLoader) LoadFromReader(reader io.Reader) (VectorModel, error) {
	scanner := bufio.NewScanner(reader)

	// Read first line to get word count and dimension
	if !scanner.Scan() {
		return nil, ErrInvalidVectorFormat
	}

	firstLine := strings.TrimSpace(scanner.Text())
	if firstLine == "" {
		return nil, ErrInvalidVectorFormat
	}

	// Parse first line: "word_count dimension"
	parts := strings.Fields(firstLine)
	if len(parts) != 2 {
		return nil, fmt.Errorf(
			"%w: first line must contain word count and dimension",
			ErrInvalidVectorFormat,
		)
	}

	wordCount, err := cast.ToIntE(parts[0])
	if err != nil || wordCount <= 0 {
		return nil, fmt.Errorf("%w: invalid word count in first line", ErrInvalidVectorFormat)
	}

	dimension, err := cast.ToIntE(parts[1])
	if err != nil || dimension <= 0 {
		return nil, fmt.Errorf("%w: invalid dimension in first line", ErrInvalidVectorFormat)
	}

	el.logger.Infof("Vector file header parsed, word_count: %d, dimension: %d",
		wordCount, dimension)

	// Create vector model
	model, ok := NewVectorModel(dimension).(*vectorModel)
	if !ok {
		return nil, fmt.Errorf("%w: failed to create vector model", ErrInvalidVectorFormat)
	}

	lineNumber := 1
	loadedVectors := 0
	progressInterval := 10000 // Report progress every 10k vectors

	// Adjust progress interval for smaller files
	if wordCount < 50000 {
		progressInterval = 5000
	}
	if wordCount < 10000 {
		progressInterval = 1000
	}

	// Process each line
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse vector line: "word value1 value2 ... valueN"
		parts := strings.Fields(line)
		if len(parts) != dimension+1 {
			el.logger.Warnf(
				"Skipping invalid line, line_number: %d, expected_parts: %d, actual_parts: %d",
				lineNumber, dimension+1, len(parts))
			continue
		}

		word := parts[0]
		vector := make([]float32, dimension)

		// Parse vector values
		parseError := false
		for i := 1; i <= dimension; i++ {
			val, err := cast.ToFloat64E(parts[i])
			if err != nil {
				el.logger.Warnf(
					"Skipping line with invalid float value, line_number: %d, word: %s, value: %s",
					lineNumber, word, parts[i])
				parseError = true
				break
			}
			vector[i-1] = float32(val)
		}

		if parseError {
			continue
		}

		// Add vector to model
		model.AddVector(word, vector)
		loadedVectors++

		// Report progress at intervals
		if loadedVectors%progressInterval == 0 {
			memUsage := model.MemoryUsage()

			// Log progress
			el.logger.Infof(
				"Loading progress, loaded_vectors: %d, target: %d, progress_pct: %.2f, memory_mb: %.2f",
				loadedVectors,
				wordCount,
				float64(loadedVectors)/float64(wordCount)*100,
				float64(memUsage)/(1024*1024),
			)

			// Call progress callback if set
			if el.progressCallback != nil {
				el.progressCallback(loadedVectors, wordCount, memUsage)
			}
		}
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading vector file: %w", err)
	}

	// Get final memory usage
	finalMemUsage := model.MemoryUsage()

	el.logger.Infof(
		"Vector loading completed, loaded_vectors: %d, expected_vectors: %d, dimension: %d, "+
			"vocabulary_size: %d, memory_usage_mb: %.2f, avg_bytes_per_vector: %.2f",
		loadedVectors,
		wordCount,
		dimension,
		model.VocabularySize(),
		float64(finalMemUsage)/(1024*1024),
		finalMemUsage/int64(loadedVectors),
	)

	// Final progress callback
	if el.progressCallback != nil {
		el.progressCallback(loadedVectors, wordCount, finalMemUsage)
	}

	// Warn if loaded count doesn't match expected count
	if loadedVectors != wordCount {
		el.logger.Warnf(
			"Loaded vector count differs from header, expected: %d, actual: %d",
			wordCount, loadedVectors)
	}

	return model, nil
}
