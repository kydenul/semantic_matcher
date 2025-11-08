package semanticmatcher

import (
	"encoding/json"
	"os"
)

const (
	DefaultVectorFilePath   = ""
	DefaultMaxSequenceLen   = 512
	DefaultChineseStopWords = ""
	DefaultEnglishStopWords = ""
	DefaultEnableStats      = true
	DefaultMemoryLimit      = 10 * 1024 * 1024 * 1024 // 1GB default
)

var DefaultSupportedLanguages = []string{"zh", "en"}

// Config holds configuration parameters for the semantic matcher
type Config struct {
	VectorFilePath     string   `json:"vector_file_path"`
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
		VectorFilePath:     DefaultVectorFilePath,
		MaxSequenceLen:     DefaultMaxSequenceLen,
		ChineseStopWords:   DefaultChineseStopWords,
		EnglishStopWords:   DefaultEnglishStopWords,
		EnableStats:        DefaultEnableStats,
		MemoryLimit:        DefaultMemoryLimit,
		SupportedLanguages: DefaultSupportedLanguages,
	}
}

// LoadFromFile loads configuration from a JSON file
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return nil, err
	}

	config := DefaultConfig()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

// SaveToFile saves configuration to a JSON file
func SaveToFile(config *Config, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644) //nolint:gosec
}

// Validate checks if the configuration is valid
func Validate(config *Config) error {
	if config == nil {
		return ErrInvalidConfiguration
	}

	if config.VectorFilePath == "" {
		return ErrInvalidConfiguration
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
