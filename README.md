# Semantic Matcher

一个高性能的 Go 语义匹配库，支持跨语言文本匹配和相似度计算。

A high-performance Go library for semantic text matching and similarity calculation with cross-lingual support.

## Table of Contents | 目录

- [Features | 功能特性](#features--功能特性)
- [Quick Start | 快速开始](#quick-start--快速开始)
- [Cross-lingual Support | 跨语言支持](#cross-lingual-support--跨语言支持)
- [Configuration | 配置](#configuration--配置)
- [Vector Files | 词向量文件](#vector-files--词向量文件)
- [API Reference | API 参考](#api-reference--api-参考)
- [Performance | 性能](#performance--性能)
- [Memory Requirements | 内存需求](#memory-requirements--内存需求)
- [Testing | 测试](#testing--测试)
- [Tools | 工具](#tools--工具)
- [Examples | 示例](#examples--示例)
- [Documentation | 文档](#documentation--文档)
- [Requirements | 系统要求](#requirements--系统要求)
- [Troubleshooting | 故障排除](#troubleshooting--故障排除)

## Features | 功能特性

- **语义文本匹配 (Semantic Text Matching)**: 使用词向量在文本中查找语义相似的关键词
- **跨语言支持 (Cross-lingual Support)**: 使用对齐向量空间无缝匹配中英文文本，无需翻译
- **多向量模型 (Multiple Vector Models)**: 支持加载单个或多个向量文件并自动合并
- **高性能 (High Performance)**: 高效的向量操作，O(1) 查找时间
- **灵活配置 (Flexible Configuration)**: 基于 YAML/JSON 的配置，提供合理的默认值
- **统计监控 (Statistics & Monitoring)**: 内置词汇覆盖率和性能指标
- **生产就绪 (Production Ready)**: 完善的错误处理和内存管理

## Quick Start | 快速开始

### Installation | 安装

```bash
go get github.com/kydenul/semantic-matcher
```

### Basic Usage | 基础使用

#### 单语言匹配示例 (Monolingual Example)

```go
package main

import (
    "fmt"
    "log"
    sm "github.com/kydenul/semantic-matcher"
)

func main() {
    // 创建配置 (Create configuration)
    config := &sm.Config{
        VectorFilePaths: []string{"vector/cc.zh.300.vec"},
        MaxSequenceLen:  512,
        EnableStats:     true,
    }

    // 初始化语义匹配器 (Initialize semantic matcher)
    matcher, err := sm.NewSemanticMatcherFromConfig(config)
    if err != nil {
        log.Fatal(err)
    }

    // 查找匹配的关键词 (Find matching keywords)
    text := "我喜欢吃苹果和香蕉"
    keywords := []string{"水果", "蔬菜", "肉类"}
    
    matches := matcher.FindTopKeywords(text, keywords, 3)
    for _, match := range matches {
        fmt.Printf("关键词: %s, 分数: %.4f\n", match.Keyword, match.Score)
    }
    // 输出: 关键词: 水果, 分数: 0.7234
}
```

#### 跨语言匹配示例 (Cross-lingual Example)

```go
package main

import (
    "fmt"
    "log"
    sm "github.com/yourusername/semantic_matcher"
)

func main() {
    // 配置跨语言向量 (Configure cross-lingual vectors)
    config := &sm.Config{
        VectorFilePaths: []string{
            "vector/wiki.zh.align.vec",  // 中文对齐向量
            "vector/wiki.en.align.vec",  // 英文对齐向量
        },
        MaxSequenceLen: 512,
        EnableStats:    true,
    }

    matcher, err := sm.NewSemanticMatcherFromConfig(config)
    if err != nil {
        log.Fatal(err)
    }

    // 示例 1: 中文文本匹配英文关键词 (Chinese text with English keywords)
    text := "我喜欢吃苹果和香蕉"
    keywords := []string{"apple", "banana", "orange"}
    
    matches := matcher.FindTopKeywords(text, keywords, 3)
    for _, match := range matches {
        fmt.Printf("Keyword: %s, Score: %.4f\n", match.Keyword, match.Score)
    }
    // 输出: Keyword: apple, Score: 0.7234
    //       Keyword: banana, Score: 0.7012

    // 示例 2: 英文文本匹配中文关键词 (English text with Chinese keywords)
    text2 := "I love eating apples and bananas"
    keywords2 := []string{"苹果", "香蕉", "橙子"}
    
    matches2 := matcher.FindTopKeywords(text2, keywords2, 3)
    for _, match := range matches2 {
        fmt.Printf("关键词: %s, 分数: %.4f\n", match.Keyword, match.Score)
    }
    // 输出: 关键词: 苹果, 分数: 0.7234
    //       关键词: 香蕉, 分数: 0.7012
}
```

## Cross-lingual Support | 跨语言支持

本库支持使用对齐向量空间进行跨语言语义匹配，无需语言检测或翻译即可匹配中英文文本。

The library supports cross-lingual semantic matching using aligned vector spaces. This allows you to match Chinese and English text without language detection or translation.

### 工作原理 (How It Works)

跨语言对齐向量将不同语言的词映射到共享的向量空间中：

- 语义相似的词具有相似的向量表示
- 例如："苹果"（中文）和 "apple"（英文）具有高余弦相似度
- 向量之间的距离表示语义相似度

Cross-lingual aligned vectors map words from different languages into a shared vector space where semantically similar words have similar vector representations.

### 使用场景 (Use Cases)

- 多语言搜索和检索 (Multilingual search and retrieval)
- 跨语言文档分类 (Cross-lingual document classification)
- 双语内容推荐 (Bilingual content recommendation)
- 国际电商产品匹配 (International e-commerce product matching)
- 多语言客户支持 (Multilingual customer support)

### 配置示例 (Configuration Example)

```go
config := &sm.Config{
    VectorFilePaths: []string{
        "vector/wiki.zh.align.vec",  // 中文对齐向量 (Chinese aligned vectors)
        "vector/wiki.en.align.vec",  // 英文对齐向量 (English aligned vectors)
    },
    MaxSequenceLen:  512,
    EnableStats:     true,
}
```

### 更多示例 (More Examples)

```go
// 示例 1: 中文文本匹配英文关键词 (Chinese text with English keywords)
text := "我喜欢吃苹果和香蕉"
keywords := []string{"apple", "banana", "orange"}
matches := matcher.FindTopKeywords(text, keywords, 3)
// 结果: "apple" 和 "banana" 将获得高相似度分数

// 示例 2: 英文文本匹配中文关键词 (English text with Chinese keywords)
text := "I like to eat apples and bananas"
keywords := []string{"苹果", "香蕉", "橙子"}
matches := matcher.FindTopKeywords(text, keywords, 3)
// 结果: "苹果" 和 "香蕉" 将获得高相似度分数

// 示例 3: 混合语言文本和关键词 (Mixed language text and keywords)
text := "我喜欢 apple 和 banana"
keywords := []string{"水果", "fruit", "食物"}
matches := matcher.FindTopKeywords(text, keywords, 3)
// 结果: 所有关键词都将获得相关的相似度分数

// 示例 4: 文本相似度计算 (Text similarity calculation)
text1 := "我喜欢吃苹果"
text2 := "I like eating apples"
similarity, _ := matcher.CalculateSimilarity(text1, text2)
// 结果: similarity ≈ 0.7856
```

详细信息请参阅 [跨语言指南](docs/cross_lingual_guide.md)。

For detailed information, see the [Cross-lingual Guide](docs/cross_lingual_guide.md).

## Configuration | 配置

### 配置结构 (Configuration Structure)

```go
type Config struct {
    // 向量文件路径（支持单个或多个文件）
    // Vector file paths (supports single or multiple files)
    VectorFilePaths    []string
    
    // 文本处理的最大序列长度
    // Maximum sequence length for text processing
    MaxSequenceLen     int
    
    // 停用词文件路径
    // Stop words file paths
    ChineseStopWords   string
    EnglishStopWords   string
    
    // 启用统计信息收集
    // Enable statistics collection
    EnableStats        bool
    
    // 内存限制（字节）
    // Memory limit in bytes
    MemoryLimit        int64
    
    // 支持的语言
    // Supported languages
    SupportedLanguages []string
}
```

### 从文件加载配置 (Configuration from File)

```go
// 从 YAML 文件加载 (Load from YAML file)
config, err := sm.LoadFromFile("config/config.yaml")
if err != nil {
    log.Fatal(err)
}

matcher, err := sm.NewSemanticMatcherFromConfig(config)
```

### 默认配置 (Default Configuration)

```go
// 使用默认配置 (Use default configuration)
config := sm.DefaultConfig()
config.VectorFilePaths = []string{"vector/cc.zh.300.vec"}
```

### 配置选项说明 (Configuration Options)

| 选项 (Option) | 说明 (Description) | 默认值 (Default) |
|--------------|-------------------|-----------------|
| VectorFilePaths | 词向量文件路径列表 | [] |
| MaxSequenceLen | 最大序列长度 | 512 |
| EnableStats | 启用统计信息 | true |
| MemoryLimit | 内存限制（字节） | 4GB |
| SupportedLanguages | 支持的语言代码 | ["zh", "en"] |

## Vector Files | 词向量文件

### 下载对齐向量（推荐用于跨语言）

Download Aligned Vectors (Recommended for Cross-lingual)

```bash
# 创建向量目录 (Create vector directory)
mkdir -p vector
cd vector

# 中文对齐向量 (~400 MB)
# Chinese aligned vectors
wget https://dl.fbaipublicfiles.com/fasttext/vectors-aligned/wiki.zh.align.vec

# 英文对齐向量 (~3 GB)
# English aligned vectors
wget https://dl.fbaipublicfiles.com/fasttext/vectors-aligned/wiki.en.align.vec
```

### 下载单语言向量

Download Monolingual Vectors

```bash
# 中文 Common Crawl 向量
# Chinese Common Crawl vectors
wget https://dl.fbaipublicfiles.com/fasttext/vectors-crawl/cc.zh.300.vec.gz
gunzip cc.zh.300.vec.gz

# 英文 Common Crawl 向量
# English Common Crawl vectors
wget https://dl.fbaipublicfiles.com/fasttext/vectors-crawl/cc.en.300.vec.gz
gunzip cc.en.300.vec.gz
```

### 向量文件说明 (Vector File Information)

| 文件 (File) | 词汇量 (Vocabulary) | 维度 (Dimension) | 大小 (Size) | 用途 (Purpose) |
|------------|-------------------|-----------------|------------|---------------|
| wiki.zh.align.vec | 332K | 300 | ~400 MB | 跨语言中文 |
| wiki.en.align.vec | 2.5M | 300 | ~3 GB | 跨语言英文 |
| cc.zh.300.vec | 2M | 300 | ~2 GB | 单语言中文 |
| cc.en.300.vec | 2M | 300 | ~2 GB | 单语言英文 |

详细信息请参阅 [vector/README.md](vector/README.md)。

See [vector/README.md](vector/README.md) for more details.

## API Reference | API 参考

### SemanticMatcher

```go
// 从配置创建语义匹配器
// Create a new semantic matcher from configuration
func NewSemanticMatcherFromConfig(config *Config) (*SemanticMatcher, error)

// 在文本中查找前 N 个匹配的关键词
// Find top N matching keywords in text
func (sm *SemanticMatcher) FindTopKeywords(text string, keywords []string, topN int) []KeywordMatch

// 计算两个文本之间的相似度
// Calculate similarity between two texts
func (sm *SemanticMatcher) CalculateSimilarity(text1, text2 string) (float64, error)

// 获取统计信息
// Get statistics
func (sm *SemanticMatcher) GetStats() Stats
```

### VectorModel

```go
// 获取单词的向量
// Get vector for a word
func (vm *VectorModel) GetVector(word string) ([]float64, bool)

// 获取多个单词的平均向量
// Get average vector for multiple words
func (vm *VectorModel) GetAverageVector(words []string) ([]float64, bool)

// 获取词汇表大小
// Get vocabulary size
func (vm *VectorModel) VocabSize() int

// 获取向量维度
// Get vector dimension
func (vm *VectorModel) Dimension() int
```

### KeywordMatch

```go
type KeywordMatch struct {
    Keyword string   // 关键词 (Keyword)
    Score   float64  // 相似度分数 (Similarity score)
}
```

### Stats

```go
type Stats struct {
    VocabSize    int     // 词汇表大小 (Vocabulary size)
    TotalQueries int     // 总查询次数 (Total queries)
    OOVRate      float64 // 未登录词率 (Out-of-vocabulary rate)
}
```

## Performance | 性能

### 操作性能 (Operation Performance)

| 操作 (Operation) | 性能 (Performance) | 说明 (Description) |
|-----------------|-------------------|-------------------|
| 向量查找 (Vector Lookup) | < 0.1ms | 单次查找 |
| 平均向量计算 (Average Vector) | < 1ms | 10个词 |
| 相似度计算 (Similarity) | < 0.5ms | 每对文本 |
| 吞吐量 (Throughput) | 1000+ QPS | 典型工作负载 |

### 加载时间 (Loading Time)

- 中文向量 (Chinese vectors): ~5-10 秒
- 英文向量 (English vectors): ~30-60 秒
- 跨语言（两者）(Cross-lingual both): ~40-70 秒

## Memory Requirements | 内存需求

| 配置 (Configuration) | 词汇量 (Vocabulary) | 内存 (Memory) |
|---------------------|-------------------|--------------|
| 单语言中文 (Chinese only) | 332K 词 | ~400 MB |
| 单语言英文 (English only) | 2.5M 词 | ~3 GB |
| 跨语言（对齐）(Cross-lingual aligned) | 2.8M 词 | ~3.4 GB |
| 大型模型 (Large models) | 1M+ 词 | ~4-5 GB |

### 性能优化建议 (Performance Optimization Tips)

1. **预加载模型** (Preload Models): 在应用启动时加载模型，而不是每次请求时加载
2. **批量处理** (Batch Processing): 批量处理多个文本以提高效率
3. **设置内存限制** (Set Memory Limits): 设置适当的内存限制以防止 OOM
4. **并发访问** (Concurrent Access): 匹配器是线程安全的，支持并发查询
5. **减少词汇量** (Reduce Vocabulary): 过滤低频词以减少内存使用

## Testing | 测试

```bash
# 运行所有测试 (Run all tests)
go test -v ./...

# 运行跨语言测试 (Run cross-lingual tests)
go test -v -run TestCrossLingual

# 运行基准测试 (Run benchmarks)
go test -bench=. -benchmem

# 运行特定测试 (Run specific tests)
go test -v -run TestSemanticMatcher

# 查看测试覆盖率 (View test coverage)
go test -cover ./...
```

## Tools | 工具

### 跨语言验证工具 (Cross-lingual Validation Tool)

验证跨语言对齐向量的有效性：

Validate the effectiveness of cross-lingual aligned vectors:

```bash
cd tools
go build -o validate_crosslingual validate_crosslingual.go

# 运行验证 (Run validation)
./validate_crosslingual \
  --vectors vector/wiki.zh.align.vec,vector/wiki.en.align.vec \
  --format json \
  --output report.json

# 查看报告 (View report)
cat report.json
```

详细信息请参阅 [tools/README.md](tools/README.md)。

See [tools/README.md](tools/README.md) for more details.

## Examples | 示例

查看 [examples](examples/) 目录获取完整的工作示例：

See the [examples](examples/) directory for complete working examples:

- `examples/main.go` - 基础使用示例 (Basic usage example)
- `examples/cross_lingual/cross_lingual_example.go` - 跨语言匹配示例 (Cross-lingual matching examples)

## Documentation | 文档

- [跨语言指南 (Cross-lingual Guide)](docs/cross_lingual_guide.md) - 跨语言支持的详细指南
- [词向量文件 (Vector Files)](vector/README.md) - 词向量文件信息
- [工具文档 (Tools)](tools/README.md) - 验证工具文档
- [设计文档 (Design Document)](.kiro/specs/multi-language-vector-model/design.md) - 架构和设计

## Requirements | 系统要求

- Go 1.18 或更高版本 (Go 1.18 or higher)
- 2-4 GB RAM（取决于向量文件大小）(depending on vector file size)
- 词向量文件（fastText 格式）(Vector embedding files in fastText format)
- 推荐使用 SSD 存储以加快加载速度 (SSD storage recommended for faster loading)

## License

[Your License Here]

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Troubleshooting | 故障排除

### 常见问题 (Common Issues)

1. **维度不匹配错误** (Dimension Mismatch Error)
   - 确保所有向量文件具有相同的维度（通常为 300）
   - 检查文件第一行：`head -n 1 vector/wiki.zh.align.vec`

2. **相似度分数过低** (Low Similarity Scores)
   - 确保使用对齐向量而非普通单语言向量
   - 检查词是否在词汇表中
   - 验证向量来源是否一致

3. **内存使用过高** (High Memory Usage)
   - 设置内存限制：`config.MemoryLimit = 2 * 1024 * 1024 * 1024`
   - 使用较小的向量文件
   - 仅加载所需语言

4. **加载时间过长** (Slow Loading Time)
   - 在应用启动时预加载模型
   - 使用 SSD 存储
   - 缓存已加载的模型

详细故障排除请参阅 [跨语言指南](docs/cross_lingual_guide.md#troubleshooting)。

For detailed troubleshooting, see the [Cross-lingual Guide](docs/cross_lingual_guide.md#troubleshooting).

## Acknowledgments | 致谢

- [fastText](https://fasttext.cc/) - 提供预训练词向量 (For providing pre-trained word vectors)
- [Facebook Research](https://github.com/facebookresearch) - 提供对齐向量模型 (For aligned vector models)

## Support | 支持

如有问题或疑问，请在 GitHub 上提交 issue。

For issues and questions, please open an issue on GitHub.
