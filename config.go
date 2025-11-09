package semanticmatcher

import (
	"os"
)

const (
	DefaultMaxSequenceLen = 512
	DefaultEnableStats    = true
	DefaultMemoryLimit    = 10 * 1024 * 1024 * 1024 // 10GB default
)

var DefaultSupportedLanguages = []string{"zh", "en"}

// Config holds configuration parameters for the semantic matcher
type Config struct {
	// VectorFilePaths specifies one or more vector embedding files to load.
	// Supports both single-language and cross-lingual scenarios:
	// 	- Single file: []string{"vector/cc.zh.300.vec"}
	// 	- Multiple aligned files: []string{"vector/wiki.zh.align.vec", "vector/wiki.en.align.vec"}
	// All files must have the same vector dimension. If duplicate words exist across files,
	// later files will override earlier ones.
	VectorFilePaths    []string `json:"vector_file_paths"`
	MaxSequenceLen     int      `json:"max_sequence_length"`
	ChineseStopWords   string   `json:"chinese_stop_words_path"`
	EnglishStopWords   string   `json:"english_stop_words_path"`
	EnableStats        bool     `json:"enable_stats"`
	MemoryLimit        int64    `json:"memory_limit_bytes"`
	SupportedLanguages []string `json:"supported_languages"` // ["zh", "en"]
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		VectorFilePaths:    []string{},
		MaxSequenceLen:     DefaultMaxSequenceLen,
		ChineseStopWords:   "",
		EnglishStopWords:   "",
		EnableStats:        DefaultEnableStats,
		MemoryLimit:        DefaultMemoryLimit,
		SupportedLanguages: DefaultSupportedLanguages,
	}
}

// Validate checks if the configuration is valid
func Validate(config *Config) error {
	if config == nil {
		return ErrInvalidConfiguration
	}

	if len(config.VectorFilePaths) == 0 {
		return ErrNoVectorFiles
	}

	// Verify all vector files exist and are readable
	for _, path := range config.VectorFilePaths {
		if path == "" {
			return ErrInvalidConfiguration
		}
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return ErrInvalidConfiguration
			}
			return err
		}
	}

	if config.MaxSequenceLen <= 0 {
		return ErrInvalidConfiguration
	}

	if config.MemoryLimit <= 0 {
		return ErrInvalidConfiguration
	}

	if len(config.SupportedLanguages) == 0 {
		return ErrInvalidConfiguration
	}

	return nil
}
