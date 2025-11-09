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

// LoadMultipleFiles loads vectors from multiple .vec files and merges them into a single model
// All files must have the same vector dimension, otherwise ErrDimensionMismatch is returned
// If duplicate words exist across files, later files will overwrite earlier ones
func (el *embeddingLoader) LoadMultipleFiles(paths []string) (VectorModel, error) {
	if len(paths) == 0 {
		return nil, ErrNoVectorFiles
	}

	el.logger.Infof("Loading multiple vector files, file_count: %d", len(paths))

	var model *vectorModel
	var expectedDimension int

	// Load each file and merge into the model
	for i, path := range paths {
		el.logger.Infof("Loading vector file %d/%d, path: %s", i+1, len(paths), path)

		// Check if file exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, fmt.Errorf("file %s: %w", path, ErrVectorFileNotFound)
		}

		// Open file
		file, err := os.Open(path) //nolint:gosec
		if err != nil {
			return nil, fmt.Errorf("failed to open vector file %s: %w", path, err)
		}

		// Load first file to create the model
		if i == 0 {
			loadedModel, err := el.LoadFromReader(file)
			file.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to load first file %s: %w", path, err)
			}

			model = loadedModel.(*vectorModel)
			expectedDimension = model.Dimension()

			el.logger.Infof("First file loaded, dimension: %d, vocabulary_size: %d, memory_mb: %.2f",
				expectedDimension, model.VocabularySize(), float64(model.MemoryUsage())/(1024*1024))
		} else {
			// Merge subsequent files into the existing model
			err := el.LoadAndMergeIntoModel(model, file)
			file.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to merge file %s: %w", path, err)
			}

			el.logger.Infof("File %d/%d merged, vocabulary_size: %d, memory_mb: %.2f",
				i+1, len(paths), model.VocabularySize(), float64(model.MemoryUsage())/(1024*1024))
		}
	}

	el.logger.Infof("All vector files loaded successfully, total_files: %d, final_vocabulary_size: %d, "+
		"final_dimension: %d, total_memory_mb: %.2f",
		len(paths), model.VocabularySize(), model.Dimension(), float64(model.MemoryUsage())/(1024*1024))

	return model, nil
}

// LoadAndMergeIntoModel loads vectors from a reader and merges them into an existing model
// Returns ErrDimensionMismatch if the vector dimensions don't match
func (el *embeddingLoader) LoadAndMergeIntoModel(model *vectorModel, reader io.Reader) error {
	scanner := bufio.NewScanner(reader)

	// Read first line to get word count and dimension
	if !scanner.Scan() {
		return ErrInvalidVectorFormat
	}

	firstLine := strings.TrimSpace(scanner.Text())
	if firstLine == "" {
		return ErrInvalidVectorFormat
	}

	// Parse first line: "word_count dimension"
	parts := strings.Fields(firstLine)
	if len(parts) != 2 {
		return fmt.Errorf("%w: first line must contain word count and dimension", ErrInvalidVectorFormat)
	}

	wordCount, err := cast.ToIntE(parts[0])
	if err != nil || wordCount <= 0 {
		return fmt.Errorf("%w: invalid word count in first line", ErrInvalidVectorFormat)
	}

	dimension, err := cast.ToIntE(parts[1])
	if err != nil || dimension <= 0 {
		return fmt.Errorf("%w: invalid dimension in first line", ErrInvalidVectorFormat)
	}

	// Verify dimension matches the model
	if dimension != model.Dimension() {
		return fmt.Errorf("%w: expected dimension %d, got %d", ErrDimensionMismatch, model.Dimension(), dimension)
	}

	el.logger.Infof("Merging vector file, word_count: %d, dimension: %d", wordCount, dimension)

	lineNumber := 1
	loadedVectors := 0
	overwrittenVectors := 0
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
			el.logger.Warnf("Skipping invalid line, line_number: %d, expected_parts: %d, actual_parts: %d",
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
				el.logger.Warnf("Skipping line with invalid float value, line_number: %d, word: %s, value: %s",
					lineNumber, word, parts[i])
				parseError = true
				break
			}
			vector[i-1] = float32(val)
		}

		if parseError {
			continue
		}

		// Check if word already exists (will be overwritten)
		if _, exists := model.vectors[word]; exists {
			overwrittenVectors++
		}

		// Add vector to model (will overwrite if exists)
		model.AddVector(word, vector)
		loadedVectors++

		// Report progress at intervals
		if loadedVectors%progressInterval == 0 {
			memUsage := model.MemoryUsage()

			el.logger.Infof("Merge progress, loaded_vectors: %d, target: %d, progress_pct: %.2f, "+
				"overwritten: %d, memory_mb: %.2f",
				loadedVectors, wordCount, float64(loadedVectors)/float64(wordCount)*100,
				overwrittenVectors, float64(memUsage)/(1024*1024))

			// Call progress callback if set
			if el.progressCallback != nil {
				el.progressCallback(loadedVectors, wordCount, memUsage)
			}
		}
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading vector file: %w", err)
	}

	finalMemUsage := model.MemoryUsage()

	el.logger.Infof("Vector merge completed, loaded_vectors: %d, expected_vectors: %d, "+
		"overwritten_vectors: %d, final_vocabulary_size: %d, memory_usage_mb: %.2f",
		loadedVectors, wordCount, overwrittenVectors, model.VocabularySize(), float64(finalMemUsage)/(1024*1024))

	// Final progress callback
	if el.progressCallback != nil {
		el.progressCallback(loadedVectors, wordCount, finalMemUsage)
	}

	// Warn if loaded count doesn't match expected count
	if loadedVectors != wordCount {
		el.logger.Warnf("Loaded vector count differs from header, expected: %d, actual: %d",
			wordCount, loadedVectors)
	}

	if overwrittenVectors > 0 {
		el.logger.Infof("Duplicate words overwritten: %d", overwrittenVectors)
	}

	return nil
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
