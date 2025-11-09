package semanticmatcher

import "errors"

// Error types for the semantic search library
var (
	// ErrVectorFileNotFound indicates the vector file could not be found
	ErrVectorFileNotFound = errors.New("vector file not found")

	// ErrInvalidVectorFormat indicates the vector file format is invalid
	ErrInvalidVectorFormat = errors.New("invalid vector file format")

	// ErrDimensionMismatch indicates vector dimensions don't match expected values
	ErrDimensionMismatch = errors.New("vector dimension mismatch")

	// ErrEmptyInput indicates empty or invalid input was provided
	ErrEmptyInput = errors.New("empty input provided")

	// ErrMemoryLimitExceeded indicates memory usage has exceeded configured limits
	ErrMemoryLimitExceeded = errors.New("memory limit exceeded")

	// ErrModelNotInitialized indicates the vector model has not been properly initialized
	ErrModelNotInitialized = errors.New("vector model not initialized")

	// ErrInvalidConfiguration indicates configuration parameters are invalid
	ErrInvalidConfiguration = errors.New("invalid configuration")

	// ErrUnsupportedLanguage indicates the language is not supported
	ErrUnsupportedLanguage = errors.New("unsupported language")

	// ErrNoVectorFiles indicates no vector files were specified in configuration
	ErrNoVectorFiles = errors.New("no vector files specified")
)
