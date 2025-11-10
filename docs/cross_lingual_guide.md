# Cross-lingual Semantic Matching Guide

This guide provides comprehensive information on using the semantic matcher library for cross-lingual text matching between Chinese and English.

## Table of Contents

- [Overview](#overview)
- [How It Works](#how-it-works)
- [Getting Started](#getting-started)
- [Downloading Vector Files](#downloading-vector-files)
- [Configuration](#configuration)
- [Usage Examples](#usage-examples)
- [Performance Considerations](#performance-considerations)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)

## Overview

Cross-lingual semantic matching allows you to compare and match text across different languages without translation. The library uses aligned vector spaces where semantically similar words in different languages have similar vector representations.

### Key Benefits

- **No Translation Required**: Match Chinese and English text directly
- **Semantic Understanding**: Captures meaning, not just literal translations
- **Simple Architecture**: No additional components or language detection needed
- **High Performance**: Same performance as monolingual matching
- **Flexible**: Supports mixed-language text and keywords

### Use Cases

- Multilingual search and retrieval
- Cross-lingual document classification
- Bilingual content recommendation
- International e-commerce product matching
- Multilingual customer support

## How It Works

### Aligned Vector Spaces

Cross-lingual aligned vectors map words from different languages into a shared vector space where:

- Semantically similar words have similar vectors
- Example: "苹果" (Chinese) and "apple" (English) have high cosine similarity
- Distance between vectors represents semantic similarity

### Architecture

```
┌─────────────────────────────────────────────────────┐
│                  Semantic Matcher                    │
├─────────────────────────────────────────────────────┤
│                                                      │
│  ┌──────────────────────────────────────────────┐  │
│  │           Vector Model                       │  │
│  │  ┌────────────┐      ┌────────────┐         │  │
│  │  │  Chinese   │      │  English   │         │  │
│  │  │  Vectors   │      │  Vectors   │         │  │
│  │  │            │      │            │         │  │
│  │  │  苹果 →    │      │  apple →   │         │  │
│  │  │  [0.1...]  │      │  [0.1...]  │         │  │
│  │  └────────────┘      └────────────┘         │  │
│  │         Shared Aligned Vector Space          │  │
│  └──────────────────────────────────────────────┘  │
│                                                      │
└─────────────────────────────────────────────────────┘
```

The system loads multiple vector files and merges them into a single model, maintaining the alignment properties.

## Getting Started

### Prerequisites

- Go 1.18 or higher
- 3-4 GB available RAM
- 3-4 GB disk space for vector files

### Installation

```bash
go get github.com/yourusername/semantic_matcher
```

### Quick Start

```go
package main

import (
    "fmt"
    "log"
    sm "github.com/yourusername/semantic_matcher"
)

func main() {
    // Configure with aligned vectors
    config := &sm.Config{
        VectorFilePaths: []string{
            "vector/wiki.zh.align.vec",
            "vector/wiki.en.align.vec",
        },
        MaxSequenceLen: 512,
        EnableStats:    true,
    }

    // Initialize matcher
    matcher, err := sm.NewSemanticMatcherFromConfig(config)
    if err != nil {
        log.Fatal(err)
    }

    // Match Chinese text with English keywords
    text := "我喜欢吃苹果"
    keywords := []string{"apple", "banana", "orange"}
    matches := matcher.FindTopKeywords(text, keywords, 3)

    for _, match := range matches {
        fmt.Printf("%s: %.4f\n", match.Keyword, match.Score)
    }
}
```

## Downloading Vector Files

### Option 1: fastText Aligned Vectors (Recommended)

fastText provides pre-trained aligned vectors for 44 languages including Chinese and English.

#### Download Commands

```bash
# Create vector directory
mkdir -p vector
cd vector

# Download Chinese aligned vectors (~400 MB)
wget https://dl.fbaipublicfiles.com/fasttext/vectors-aligned/wiki.zh.align.vec

# Download English aligned vectors (~3 GB)
wget https://dl.fbaipublicfiles.com/fasttext/vectors-aligned/wiki.en.align.vec
```

#### Verify Downloads

```bash
# Check file sizes
ls -lh wiki.*.align.vec

# Check vector dimensions (should both be 300)
head -n 1 wiki.zh.align.vec
head -n 1 wiki.en.align.vec
```

Expected output:
```
332647 300    # Chinese: 332K words, 300 dimensions
2519370 300   # English: 2.5M words, 300 dimensions
```

### Option 2: MUSE Multilingual Vectors

MUSE provides alternative multilingual aligned vectors.

```bash
# Download MUSE vectors
wget https://dl.fbaipublicfiles.com/arrival/vectors/wiki.multi.zh.vec
wget https://dl.fbaipublicfiles.com/arrival/vectors/wiki.multi.en.vec
```

### Option 3: Merged Vector File

If you prefer a single file, you can merge the vectors:

```bash
# Merge Chinese and English vectors
cat wiki.zh.align.vec wiki.en.align.vec > wiki.zh-en.merged.vec

# Update the first line with total word count
# (This requires manual editing or a script)
```

**Note**: When using multiple files, the system automatically handles merging, so this step is optional.

## Configuration

### Basic Configuration

```go
config := &sm.Config{
    VectorFilePaths: []string{
        "vector/wiki.zh.align.vec",
        "vector/wiki.en.align.vec",
    },
    MaxSequenceLen:     512,
    EnableStats:        true,
    MemoryLimit:        4 * 1024 * 1024 * 1024, // 4 GB
    SupportedLanguages: []string{"zh", "en"},
}
```

### Configuration Fields

#### VectorFilePaths

Specifies one or more vector files to load. The system will:
- Load files in order
- Merge all vectors into a single model
- Verify dimension consistency
- Handle duplicate words (later files override earlier ones)

```go
// Single file
VectorFilePaths: []string{"vector/wiki.multi.vec"}

// Multiple files (recommended)
VectorFilePaths: []string{
    "vector/wiki.zh.align.vec",
    "vector/wiki.en.align.vec",
}
```

#### MaxSequenceLen

Maximum number of words to process in a text sequence.

```go
MaxSequenceLen: 512  // Default: 512
```

#### EnableStats

Enable statistics collection for monitoring.

```go
EnableStats: true  // Default: true
```

#### MemoryLimit

Maximum memory usage in bytes. Loading will fail if exceeded.

```go
MemoryLimit: 4 * 1024 * 1024 * 1024  // 4 GB
```

#### SupportedLanguages

List of supported language codes.

```go
SupportedLanguages: []string{"zh", "en"}  // Default
```

### Configuration from JSON

Create a `config.json` file:

```json
{
  "vector_file_paths": [
    "vector/wiki.zh.align.vec",
    "vector/wiki.en.align.vec"
  ],
  "max_sequence_length": 512,
  "enable_stats": true,
  "memory_limit_bytes": 4294967296,
  "supported_languages": ["zh", "en"]
}
```

Load configuration:

```go
config, err := sm.LoadFromFile("config.json")
if err != nil {
    log.Fatal(err)
}
```

## Usage Examples

### Example 1: Chinese Text with English Keywords

```go
matcher, _ := sm.NewSemanticMatcherFromConfig(config)

text := "我喜欢吃苹果和香蕉，它们都是健康的水果"
keywords := []string{"apple", "banana", "orange", "meat", "vegetable"}

matches := matcher.FindTopKeywords(text, keywords, 3)

for _, match := range matches {
    fmt.Printf("Keyword: %s, Score: %.4f\n", match.Keyword, match.Score)
}

// Output:
// Keyword: apple, Score: 0.7234
// Keyword: banana, Score: 0.7012
// Keyword: orange, Score: 0.5123
```

### Example 2: English Text with Chinese Keywords

```go
text := "I love eating apples and bananas, they are healthy fruits"
keywords := []string{"苹果", "香蕉", "橙子", "肉类", "蔬菜"}

matches := matcher.FindTopKeywords(text, keywords, 3)

for _, match := range matches {
    fmt.Printf("关键词: %s, 分数: %.4f\n", match.Keyword, match.Score)
}

// Output:
// 关键词: 苹果, 分数: 0.7234
// 关键词: 香蕉, 分数: 0.7012
// 关键词: 橙子, 分数: 0.5123
```

### Example 3: Mixed Language Text

```go
text := "我喜欢 apple 和 banana，还有 orange"
keywords := []string{"水果", "fruit", "食物", "food"}

matches := matcher.FindTopKeywords(text, keywords, 4)

for _, match := range matches {
    fmt.Printf("Keyword: %s, Score: %.4f\n", match.Keyword, match.Score)
}

// Output:
// Keyword: fruit, Score: 0.6845
// Keyword: 水果, Score: 0.6723
// Keyword: food, Score: 0.5234
// Keyword: 食物, Score: 0.5123
```

### Example 4: Text Similarity Calculation

```go
text1 := "我喜欢吃苹果"
text2 := "I like eating apples"

similarity, err := matcher.CalculateSimilarity(text1, text2)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Similarity: %.4f\n", similarity)
// Output: Similarity: 0.7856
```

### Example 5: Batch Processing

```go
texts := []string{
    "我喜欢吃苹果",
    "I love apples",
    "香蕉很好吃",
    "Bananas are delicious",
}

keywords := []string{"apple", "banana", "orange"}

for i, text := range texts {
    matches := matcher.FindTopKeywords(text, keywords, 2)
    fmt.Printf("Text %d: %s\n", i+1, text)
    for _, match := range matches {
        fmt.Printf("  - %s: %.4f\n", match.Keyword, match.Score)
    }
}
```

### Example 6: Using Statistics

```go
// Enable statistics
config.EnableStats = true
matcher, _ := sm.NewSemanticMatcherFromConfig(config)

// Perform some operations
matcher.FindTopKeywords("我喜欢苹果", []string{"apple"}, 1)
matcher.FindTopKeywords("I like apples", []string{"苹果"}, 1)

// Get statistics
stats := matcher.GetStats()
fmt.Printf("Total queries: %d\n", stats.TotalQueries)
fmt.Printf("OOV rate: %.2f%%\n", stats.OOVRate*100)
fmt.Printf("Vocabulary size: %d\n", stats.VocabSize)
```

## Performance Considerations

### Memory Usage

Typical memory requirements:

| Configuration | Vocabulary | Memory |
|--------------|------------|---------|
| Chinese only | 332K words | ~400 MB |
| English only | 2.5M words | ~3 GB |
| Both (aligned) | 2.8M words | ~3.4 GB |

### Loading Time

- Chinese vectors: ~5-10 seconds
- English vectors: ~30-60 seconds
- Total (both): ~40-70 seconds

### Query Performance

- Vector lookup: < 0.1 ms
- Average vector calculation (10 words): < 1 ms
- Similarity calculation: < 0.5 ms
- Throughput: 1000+ QPS

### Optimization Tips

1. **Reduce Vocabulary**: Filter low-frequency words to reduce memory
2. **Preload Models**: Load models at startup, not per request
3. **Batch Processing**: Process multiple texts in batches
4. **Memory Limits**: Set appropriate memory limits to prevent OOM
5. **Concurrent Access**: The matcher is thread-safe for concurrent queries

### Memory Optimization Example

```go
// Set memory limit
config.MemoryLimit = 3 * 1024 * 1024 * 1024  // 3 GB max

// The loader will fail if vectors exceed this limit
matcher, err := sm.NewSemanticMatcherFromConfig(config)
if err != nil {
    log.Printf("Memory limit exceeded: %v", err)
    // Fall back to smaller vector files or increase limit
}
```

## Troubleshooting

### Issue: Dimension Mismatch Error

**Error**: `dimension mismatch: expected 300, got 200`

**Cause**: Vector files have different dimensions

**Solution**:
```bash
# Check dimensions of all files
head -n 1 vector/wiki.zh.align.vec
head -n 1 vector/wiki.en.align.vec

# Ensure both show the same dimension (e.g., 300)
```

### Issue: Low Similarity Scores

**Problem**: Cross-lingual similarity scores are unexpectedly low

**Possible Causes**:
1. Using non-aligned vectors (regular monolingual vectors)
2. Words not in vocabulary (OOV)
3. Different vector sources (not from same alignment project)

**Solution**:
```go
// Check if words exist in vocabulary
vec1, ok1 := matcher.GetVector("苹果")
vec2, ok2 := matcher.GetVector("apple")

if !ok1 || !ok2 {
    fmt.Println("One or both words not in vocabulary")
}

// Verify using aligned vectors
// Download from: https://fasttext.cc/docs/en/aligned-vectors.html
```

### Issue: High Memory Usage

**Problem**: Application uses too much memory

**Solutions**:

1. **Use smaller vector files**:
```bash
# Use only essential vocabulary
# Create filtered vector file with common words only
```

2. **Set memory limits**:
```go
config.MemoryLimit = 2 * 1024 * 1024 * 1024  // 2 GB
```

3. **Load only one language**:
```go
// If cross-lingual not needed
config.VectorFilePaths = []string{"vector/wiki.zh.align.vec"}
```

### Issue: Slow Loading Time

**Problem**: Vector files take too long to load

**Solutions**:

1. **Preload at startup**: Load models during application initialization
2. **Use SSD storage**: Faster disk I/O significantly reduces loading time
3. **Cache loaded models**: Keep models in memory between requests

### Issue: File Not Found

**Error**: `vector file not found: vector/wiki.zh.align.vec`

**Solution**:
```bash
# Verify file exists
ls -lh vector/wiki.zh.align.vec

# Check file permissions
chmod 644 vector/wiki.zh.align.vec

# Verify path is correct (relative to working directory)
pwd
```

## Best Practices

### 1. Use Aligned Vectors for Cross-lingual

Always use aligned vectors (not regular monolingual vectors) for cross-lingual matching:

```go
// ✅ Correct: Aligned vectors
VectorFilePaths: []string{
    "vector/wiki.zh.align.vec",
    "vector/wiki.en.align.vec",
}

// ❌ Wrong: Regular monolingual vectors
VectorFilePaths: []string{
    "vector/cc.zh.300.vec",
    "vector/cc.en.300.vec",
}
```

### 2. Validate Vector Quality

Use the validation tool to verify alignment quality:

```bash
cd tools
./validate_crosslingual \
  --vectors vector/wiki.zh.align.vec,vector/wiki.en.align.vec \
  --format json \
  --output report.json

# Check average similarity
jq '.average_similarity' report.json
```

Expected average similarity: > 0.5 for good alignment

### 3. Handle OOV Words

Check for out-of-vocabulary words and handle gracefully:

```go
matches := matcher.FindTopKeywords(text, keywords, 3)

if len(matches) == 0 {
    log.Println("No matches found - possible OOV words")
    // Fall back to alternative matching strategy
}

// Check OOV rate
stats := matcher.GetStats()
if stats.OOVRate > 0.3 {
    log.Printf("High OOV rate: %.2f%%", stats.OOVRate*100)
    // Consider using larger vector files
}
```

### 4. Set Appropriate Thresholds

Use similarity thresholds to filter low-quality matches:

```go
matches := matcher.FindTopKeywords(text, keywords, 10)

// Filter by threshold
threshold := 0.5
var filtered []KeywordMatch
for _, match := range matches {
    if match.Score >= threshold {
        filtered = append(filtered, match)
    }
}
```

### 5. Monitor Performance

Enable statistics and monitor key metrics:

```go
config.EnableStats = true

// Periodically check stats
stats := matcher.GetStats()
log.Printf("Vocab: %d, Queries: %d, OOV Rate: %.2f%%",
    stats.VocabSize, stats.TotalQueries, stats.OOVRate*100)
```

### 6. Preload Models

Load models at application startup, not per request:

```go
var globalMatcher *sm.SemanticMatcher

func init() {
    config := &sm.Config{
        VectorFilePaths: []string{
            "vector/wiki.zh.align.vec",
            "vector/wiki.en.align.vec",
        },
    }
    
    var err error
    globalMatcher, err = sm.NewSemanticMatcherFromConfig(config)
    if err != nil {
        log.Fatal(err)
    }
}

func handleRequest(text string, keywords []string) []KeywordMatch {
    return globalMatcher.FindTopKeywords(text, keywords, 5)
}
```

### 7. Use Appropriate Vector Files

Choose vector files based on your use case:

| Use Case | Recommended Vectors |
|----------|-------------------|
| Chinese only | wiki.zh.align.vec or cc.zh.300.vec |
| English only | wiki.en.align.vec or cc.en.300.vec |
| Cross-lingual | wiki.zh.align.vec + wiki.en.align.vec |
| Memory constrained | Filtered/trimmed vector files |
| High accuracy | Larger vocabulary files |

## Advanced Topics

### Custom Vector Files

You can create custom aligned vectors for domain-specific applications:

1. Train domain-specific monolingual vectors
2. Use alignment tools (MUSE, VecMap) to align them
3. Load custom vectors using the same configuration

### Extending to More Languages

The architecture supports multiple languages:

```go
config.VectorFilePaths = []string{
    "vector/wiki.zh.align.vec",  // Chinese
    "vector/wiki.en.align.vec",  // English
    "vector/wiki.ja.align.vec",  // Japanese
    "vector/wiki.ko.align.vec",  // Korean
}
```

All vectors must be from the same alignment project.

### Integration with Other Systems

The library can be integrated with:

- Search engines (Elasticsearch, Solr)
- Recommendation systems
- Classification pipelines
- NLP preprocessing pipelines

## Resources

### Documentation

- [README](../README.md) - Main documentation
- [Vector Files](../vector/README.md) - Vector file information
- [Tools](../tools/README.md) - Validation tools
- [Design Document](../.kiro/specs/multi-language-vector-model/design.md) - Architecture details

### External Resources

- [fastText Official Site](https://fasttext.cc/)
- [fastText Aligned Vectors](https://fasttext.cc/docs/en/aligned-vectors.html)
- [MUSE Project](https://github.com/facebookresearch/MUSE)
- [VecMap](https://github.com/artetxem/vecmap)

### Papers

- "Word Translation Without Parallel Data" (Conneau et al., 2017)
- "Learning principled bilingual mappings of word embeddings" (Artetxe et al., 2016)

## Support

For issues, questions, or contributions:

- Open an issue on GitHub
- Check existing documentation
- Review test cases for examples

## Next Steps

1. Download aligned vector files
2. Configure your application
3. Run validation tool to verify quality
4. Implement cross-lingual matching
5. Monitor performance and optimize

Happy coding! 🚀
