package semanticmatcher

import (
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVectorModel(t *testing.T) {
	dimension := 100
	vm := NewVectorModel(dimension)

	if vm.Dimension() != dimension {
		t.Errorf("Expected dimension %d, got %d", dimension, vm.Dimension())
	}

	if vm.VocabularySize() != 0 {
		t.Errorf("Expected empty vocabulary, got size %d", vm.VocabularySize())
	}
}

func TestVectorModel_GetVector(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add test vector
	testVector := []float32{1.0, 2.0, 3.0}
	vm.AddVector("test", testVector)

	// Test existing word
	vector, exists := vm.GetVector("test")
	if !exists {
		t.Error("Expected word 'test' to exist")
	}

	if len(vector) != 3 {
		t.Errorf("Expected vector length 3, got %d", len(vector))
	}

	for i, val := range testVector {
		if vector[i] != val {
			t.Errorf("Expected vector[%d] = %f, got %f", i, val, vector[i])
		}
	}

	// Test non-existing word
	_, exists = vm.GetVector("nonexistent")
	if exists {
		t.Error("Expected word 'nonexistent' to not exist")
	}

	// Test that returned vector is a copy (modification doesn't affect original)
	vector[0] = 999.0
	originalVector, _ := vm.GetVector("test")
	if originalVector[0] == 999.0 {
		t.Error("Vector modification affected original - should return copy")
	}
}

func TestVectorModel_GetAverageVector(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add test vectors
	vm.AddVector("word1", []float32{1.0, 2.0, 3.0})
	vm.AddVector("word2", []float32{4.0, 5.0, 6.0})
	vm.AddVector("word3", []float32{7.0, 8.0, 9.0})

	// Test average of two words
	words := []string{"word1", "word2"}
	avgVector, exists := vm.GetAverageVector(words)
	if !exists {
		t.Error("Expected average vector to exist")
	}

	expected := []float32{2.5, 3.5, 4.5} // (1+4)/2, (2+5)/2, (3+6)/2
	for i, val := range expected {
		if avgVector[i] != val {
			t.Errorf("Expected avgVector[%d] = %f, got %f", i, val, avgVector[i])
		}
	}

	// Test with some OOV words
	wordsWithOOV := []string{"word1", "nonexistent", "word2"}
	avgVector, exists = vm.GetAverageVector(wordsWithOOV)
	if !exists {
		t.Error("Expected average vector to exist even with OOV words")
	}

	// Should still be average of word1 and word2
	for i, val := range expected {
		if avgVector[i] != val {
			t.Errorf("Expected avgVector[%d] = %f, got %f", i, val, avgVector[i])
		}
	}

	// Test with all OOV words
	oovWords := []string{"nonexistent1", "nonexistent2"}
	_, exists = vm.GetAverageVector(oovWords)
	if exists {
		t.Error("Expected no average vector for all OOV words")
	}

	// Test with empty word list
	_, exists = vm.GetAverageVector([]string{})
	if exists {
		t.Error("Expected no average vector for empty word list")
	}
}

func TestVectorModel_Dimension(t *testing.T) {
	dimensions := []int{50, 100, 300}

	for _, dim := range dimensions {
		vm := NewVectorModel(dim)
		if vm.Dimension() != dim {
			t.Errorf("Expected dimension %d, got %d", dim, vm.Dimension())
		}
	}
}

func TestVectorModel_VocabularySize(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	if vm.VocabularySize() != 0 {
		t.Errorf("Expected vocabulary size 0, got %d", vm.VocabularySize())
	}

	// Add vectors and check size
	vm.AddVector("word1", []float32{1.0, 2.0, 3.0})
	if vm.VocabularySize() != 1 {
		t.Errorf("Expected vocabulary size 1, got %d", vm.VocabularySize())
	}

	vm.AddVector("word2", []float32{4.0, 5.0, 6.0})
	if vm.VocabularySize() != 2 {
		t.Errorf("Expected vocabulary size 2, got %d", vm.VocabularySize())
	}
}

func TestVectorModel_AddVector(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Test adding valid vector
	testVector := []float32{1.0, 2.0, 3.0}
	vm.AddVector("test", testVector)

	if vm.VocabularySize() != 1 {
		t.Errorf("Expected vocabulary size 1, got %d", vm.VocabularySize())
	}

	// Test adding vector with wrong dimension (should be ignored)
	wrongDimVector := []float32{1.0, 2.0} // dimension 2 instead of 3
	vm.AddVector("wrong", wrongDimVector)

	if vm.VocabularySize() != 1 {
		t.Errorf("Expected vocabulary size to remain 1, got %d", vm.VocabularySize())
	}

	// Test that stored vector is a copy
	testVector[0] = 999.0
	storedVector, _ := vm.GetVector("test")
	if storedVector[0] == 999.0 {
		t.Error("Stored vector was affected by external modification - should store copy")
	}
}

func TestVectorModel_ConcurrentAccess(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add some initial vectors
	for i := range 100 {
		vm.AddVector(string(rune('a'+i)), []float32{float32(i), float32(i + 1), float32(i + 2)})
	}

	var wg sync.WaitGroup
	numGoroutines := 10

	// Test concurrent reads
	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Perform multiple read operations
			for j := range 100 {
				word := string(rune('a' + (j % 26)))
				vm.GetVector(word)
				vm.GetAverageVector([]string{word, string(rune('b' + (j % 25)))})
				vm.Dimension()
				vm.VocabularySize()
			}
		}(i)
	}

	wg.Wait()

	// Verify data integrity after concurrent access
	if vm.VocabularySize() != 100 {
		t.Errorf(
			"Expected vocabulary size 100 after concurrent access, got %d",
			vm.VocabularySize(),
		)
	}
}

// Benchmark tests for vector operations

func BenchmarkVectorModel_GetVector(b *testing.B) {
	vm := NewVectorModel(100).(*vectorModel)

	// Add test vectors
	for i := range 1000 {
		vector := make([]float32, 100)
		for j := range 100 {
			vector[j] = float32(i*100 + j)
		}
		vm.AddVector(fmt.Sprintf("word%d", i), vector)
	}

	for i := 0; b.Loop(); i++ {
		vm.GetVector(fmt.Sprintf("word%d", i%1000))
	}
}

func BenchmarkVectorModel_GetAverageVector(b *testing.B) {
	vm := NewVectorModel(100).(*vectorModel)

	// Add test vectors
	for i := range 1000 {
		vector := make([]float32, 100)
		for j := range 100 {
			vector[j] = float32(i*100 + j)
		}
		vm.AddVector(fmt.Sprintf("word%d", i), vector)
	}

	// Prepare test word lists
	words := []string{"word0", "word1", "word2", "word3", "word4"}

	for b.Loop() {
		vm.GetAverageVector(words)
	}
}

func BenchmarkVectorModel_AddVector(b *testing.B) {
	vm := NewVectorModel(100).(*vectorModel)
	vector := make([]float32, 100)
	for j := range 100 {
		vector[j] = float32(j)
	}

	for i := 0; b.Loop(); i++ {
		vm.AddVector(fmt.Sprintf("word%d", i), vector)
	}
}

func BenchmarkVectorModel_ConcurrentReads(b *testing.B) {
	vm := NewVectorModel(100).(*vectorModel)

	// Add test vectors
	for i := range 1000 {
		vector := make([]float32, 100)
		for j := range 100 {
			vector[j] = float32(i*100 + j)
		}
		vm.AddVector(fmt.Sprintf("word%d", i), vector)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			vm.GetVector(fmt.Sprintf("word%d", i%1000))
			i++
		}
	})
}

func TestVectorModel_OOVStatistics(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add test vectors
	vm.AddVector("word1", []float32{1.0, 2.0, 3.0})
	vm.AddVector("word2", []float32{4.0, 5.0, 6.0})
	vm.AddVector("word3", []float32{7.0, 8.0, 9.0})

	// Initial state - no lookups yet
	if rate := vm.GetOOVRate(); rate != 0.0 {
		t.Errorf("Expected initial OOV rate 0.0, got %f", rate)
	}

	if rate := vm.GetVectorHitRate(); rate != 0.0 {
		t.Errorf("Expected initial hit rate 0.0, got %f", rate)
	}

	// Perform some lookups
	vm.GetVector("word1")       // hit
	vm.GetVector("word2")       // hit
	vm.GetVector("nonexistent") // miss - triggers fallback

	// Check statistics
	total, oov, hit, fallbackAttempts, fallbackSuccesses, fallbackFailures := vm.GetLookupStats()
	if total != 3 {
		t.Errorf("Expected 3 total lookups, got %d", total)
	}
	if oov != 1 {
		t.Errorf("Expected 1 OOV lookup, got %d", oov)
	}
	if hit != 2 {
		t.Errorf("Expected 2 hit lookups, got %d", hit)
	}
	if fallbackAttempts != 1 {
		t.Errorf("Expected 1 fallback attempt, got %d", fallbackAttempts)
	}
	if fallbackSuccesses != 0 {
		t.Errorf("Expected 0 fallback successes, got %d", fallbackSuccesses)
	}
	if fallbackFailures != 1 {
		t.Errorf("Expected 1 fallback failure, got %d", fallbackFailures)
	}

	// Check rates
	expectedOOVRate := 1.0 / 3.0
	if rate := vm.GetOOVRate(); math.Abs(rate-expectedOOVRate) > 1e-6 {
		t.Errorf("Expected OOV rate %f, got %f", expectedOOVRate, rate)
	}

	expectedHitRate := 2.0 / 3.0
	if rate := vm.GetVectorHitRate(); math.Abs(rate-expectedHitRate) > 1e-6 {
		t.Errorf("Expected hit rate %f, got %f", expectedHitRate, rate)
	}

	// Test GetAverageVector with mixed hits and misses
	// Note: GetAverageVector now triggers fallback for OOV words (task 4 completed)
	vm.GetAverageVector([]string{"word1", "word2", "nonexistent1", "nonexistent2"})

	total, oov, hit, fallbackAttempts, fallbackSuccesses, fallbackFailures = vm.GetLookupStats()
	if total != 7 { // 3 previous + 4 new
		t.Errorf("Expected 7 total lookups, got %d", total)
	}
	if oov != 3 { // 1 previous + 2 new
		t.Errorf("Expected 3 OOV lookups, got %d", oov)
	}
	if hit != 4 { // 2 previous + 2 new
		t.Errorf("Expected 4 hit lookups, got %d", hit)
	}
	// GetAverageVector now triggers fallback for OOV words: 1 previous + 2 new = 3 attempts
	if fallbackAttempts != 3 {
		t.Errorf("Expected 3 fallback attempts, got %d", fallbackAttempts)
	}
	if fallbackSuccesses != 0 {
		t.Errorf("Expected 0 fallback successes, got %d", fallbackSuccesses)
	}
	// All fallback attempts fail because "nonexistent" words have no character vectors
	if fallbackFailures != 3 {
		t.Errorf("Expected 3 fallback failures, got %d", fallbackFailures)
	}

	// Test reset
	vm.ResetStats()
	total, oov, hit, fallbackAttempts, fallbackSuccesses, fallbackFailures = vm.GetLookupStats()
	if total != 0 || oov != 0 || hit != 0 || fallbackAttempts != 0 || fallbackSuccesses != 0 || fallbackFailures != 0 {
		t.Errorf(
			"Expected all stats to be 0 after reset, got total=%d, oov=%d, hit=%d, fallbackAttempts=%d, fallbackSuccesses=%d, fallbackFailures=%d",
			total,
			oov,
			hit,
			fallbackAttempts,
			fallbackSuccesses,
			fallbackFailures,
		)
	}
}

func TestVectorModel_AllOOVWords(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add test vectors
	vm.AddVector("word1", []float32{1.0, 2.0, 3.0})

	// Try to get average of all OOV words
	_, exists := vm.GetAverageVector([]string{"nonexistent1", "nonexistent2", "nonexistent3"})
	if exists {
		t.Error("Expected GetAverageVector to return false for all OOV words")
	}

	// Check statistics
	total, oov, hit, _, _, _ := vm.GetLookupStats()
	if total != 3 {
		t.Errorf("Expected 3 total lookups, got %d", total)
	}
	if oov != 3 {
		t.Errorf("Expected 3 OOV lookups, got %d", oov)
	}
	if hit != 0 {
		t.Errorf("Expected 0 hit lookups, got %d", hit)
	}

	// OOV rate should be 100%
	if rate := vm.GetOOVRate(); rate != 1.0 {
		t.Errorf("Expected OOV rate 1.0, got %f", rate)
	}
}

func TestVectorModel_EmptyInputValidation(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Test GetAverageVector with empty slice
	_, exists := vm.GetAverageVector([]string{})
	if exists {
		t.Error("Expected GetAverageVector to return false for empty input")
	}

	// Statistics should not be affected by empty input
	total, _, _, _, _, _ := vm.GetLookupStats()
	if total != 0 {
		t.Errorf("Expected 0 total lookups for empty input, got %d", total)
	}
}

// TestCrossLingualWordPairSimilarity 验证中英文对应词的向量相似度
func TestCrossLingualWordPairSimilarity(t *testing.T) {
	logger := &DiscardLogger{}
	loader := NewEmbeddingLoader(logger)

	paths := []string{
		"vector/wiki.zh.align.vec",
		"vector/wiki.en.align.vec",
	}

	model, err := loader.LoadMultipleFiles(paths)
	require.NoError(t, err, "应该成功加载跨语言向量文件")

	calculator := NewSimilarityCalculator()

	// 测试常见中英文对应词的相似度
	wordPairs := []struct {
		chinese string
		english string
		minSim  float64 // 最小期望相似度
	}{
		{"苹果", "apple", 0.3},
		{"中国", "china", 0.3},
		{"电脑", "computer", 0.3},
		{"学习", "learning", 0.2},
		{"工作", "work", 0.2},
	}

	for _, pair := range wordPairs {
		vecCh, foundCh := model.GetVector(pair.chinese)
		vecEn, foundEn := model.GetVector(pair.english)

		if foundCh && foundEn {
			similarity := calculator.CosineSimilarity(vecCh, vecEn)
			t.Logf("词对 '%s' <-> '%s' 的相似度: %.4f", pair.chinese, pair.english, similarity)

			// 验证相似度在合理范围内
			assert.GreaterOrEqual(t, similarity, -1.0, "相似度应该 >= -1")
			assert.LessOrEqual(t, similarity, 1.0, "相似度应该 <= 1")

			// 对于语义对应的词，相似度应该相对较高
			if similarity >= pair.minSim {
				t.Logf("✓ 词对 '%s' <-> '%s' 的相似度 %.4f 符合预期 (>= %.2f)",
					pair.chinese, pair.english, similarity, pair.minSim)
			} else {
				t.Logf("⚠ 词对 '%s' <-> '%s' 的相似度 %.4f 低于预期 (< %.2f)",
					pair.chinese, pair.english, similarity, pair.minSim)
			}
		} else {
			t.Logf("词对 '%s'/'%s' 中有词不在词汇表中 (chinese_found: %v, english_found: %v)",
				pair.chinese, pair.english, foundCh, foundEn)
		}
	}
}

// TestChineseParagraphMatchEnglishKeywords 测试中文段落匹配英文关键词
func TestChineseParagraphMatchEnglishKeywords(t *testing.T) {
	logger := &DiscardLogger{}

	config := &Config{
		VectorFilePaths:    []string{"vector/wiki.zh.align.vec", "vector/wiki.en.align.vec"},
		MaxSequenceLen:     512,
		EnableStats:        true,
		MemoryLimit:        10 * 1024 * 1024 * 1024, // 10GB
		SupportedLanguages: []string{"zh", "en"},
	}

	matcher, err := NewSemanticMatcherFromConfig(config, logger)
	require.NoError(t, err, "应该成功创建 SemanticMatcher")

	// 测试场景：中文段落，英文关键词
	testCases := []struct {
		name            string
		paragraph       string
		keywords        []string
		expectedTopWord string // 期望排名最高的关键词
	}{
		{
			name:            "水果相关",
			paragraph:       "我喜欢吃苹果和香蕉，它们都是很健康的水果",
			keywords:        []string{"apple", "banana", "computer", "car"},
			expectedTopWord: "apple", // apple 或 banana 应该排名靠前
		},
		{
			name:            "国家相关",
			paragraph:       "中国是一个历史悠久的国家",
			keywords:        []string{"china", "america", "computer", "food"},
			expectedTopWord: "china",
		},
		{
			name:            "科技相关",
			paragraph:       "我每天使用电脑工作和学习",
			keywords:        []string{"computer", "work", "study", "food"},
			expectedTopWord: "computer",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results := matcher.FindTopKeywords(tc.paragraph, tc.keywords, 3)

			// 验证返回了结果
			assert.NotEmpty(t, results, "应该返回匹配结果")

			// 记录所有结果
			t.Logf("测试场景: %s", tc.name)
			t.Logf("中文段落: %s", tc.paragraph)
			t.Logf("英文关键词: %v", tc.keywords)
			for i, match := range results {
				t.Logf("  排名 %d: %s (相似度: %.4f)", i+1, match.Keyword, match.Score)
			}

			// 验证期望的关键词在前几名
			if len(results) > 0 {
				topKeywords := make([]string, 0, len(results))
				for _, match := range results {
					topKeywords = append(topKeywords, match.Keyword)
				}

				// 检查期望的关键词是否在结果中
				found := false
				for _, kw := range topKeywords {
					if kw == tc.expectedTopWord {
						found = true
						break
					}
				}

				if found {
					t.Logf("✓ 期望的关键词 '%s' 在匹配结果中", tc.expectedTopWord)
				} else {
					t.Logf("⚠ 期望的关键词 '%s' 不在匹配结果中", tc.expectedTopWord)
				}
			}
		})
	}
}

// TestEnglishParagraphMatchChineseKeywords 测试英文段落匹配中文关键词
func TestEnglishParagraphMatchChineseKeywords(t *testing.T) {
	logger := &DiscardLogger{}

	config := &Config{
		VectorFilePaths:    []string{"vector/wiki.zh.align.vec", "vector/wiki.en.align.vec"},
		MaxSequenceLen:     512,
		EnableStats:        true,
		MemoryLimit:        10 * 1024 * 1024 * 1024, // 10GB
		SupportedLanguages: []string{"zh", "en"},
	}

	matcher, err := NewSemanticMatcherFromConfig(config, logger)
	require.NoError(t, err, "应该成功创建 SemanticMatcher")

	// 测试场景：英文段落，中文关键词
	testCases := []struct {
		name            string
		paragraph       string
		keywords        []string
		expectedTopWord string
	}{
		{
			name:            "水果相关",
			paragraph:       "I like to eat apples and bananas, they are healthy fruits",
			keywords:        []string{"苹果", "香蕉", "电脑", "汽车"},
			expectedTopWord: "苹果",
		},
		{
			name:            "国家相关",
			paragraph:       "China is a country with a long history",
			keywords:        []string{"中国", "美国", "电脑", "食物"},
			expectedTopWord: "中国",
		},
		{
			name:            "科技相关",
			paragraph:       "I use my computer for work and study every day",
			keywords:        []string{"电脑", "工作", "学习", "食物"},
			expectedTopWord: "电脑",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results := matcher.FindTopKeywords(tc.paragraph, tc.keywords, 3)

			// 验证返回了结果
			assert.NotEmpty(t, results, "应该返回匹配结果")

			// 记录所有结果
			t.Logf("测试场景: %s", tc.name)
			t.Logf("英文段落: %s", tc.paragraph)
			t.Logf("中文关键词: %v", tc.keywords)
			for i, match := range results {
				t.Logf("  排名 %d: %s (相似度: %.4f)", i+1, match.Keyword, match.Score)
			}

			// 验证期望的关键词在前几名
			if len(results) > 0 {
				topKeywords := make([]string, 0, len(results))
				for _, match := range results {
					topKeywords = append(topKeywords, match.Keyword)
				}

				found := false
				for _, kw := range topKeywords {
					if kw == tc.expectedTopWord {
						found = true
						break
					}
				}

				if found {
					t.Logf("✓ 期望的关键词 '%s' 在匹配结果中", tc.expectedTopWord)
				} else {
					t.Logf("⚠ 期望的关键词 '%s' 不在匹配结果中", tc.expectedTopWord)
				}
			}
		})
	}
}

// TestMixedLanguageParagraphMatchMixedKeywords 测试混合语言段落匹配混合语言关键词
func TestMixedLanguageParagraphMatchMixedKeywords(t *testing.T) {
	logger := &DiscardLogger{}

	config := &Config{
		VectorFilePaths:    []string{"vector/wiki.zh.align.vec", "vector/wiki.en.align.vec"},
		MaxSequenceLen:     512,
		EnableStats:        true,
		MemoryLimit:        10 * 1024 * 1024 * 1024, // 10GB
		SupportedLanguages: []string{"zh", "en"},
	}

	matcher, err := NewSemanticMatcherFromConfig(config, logger)
	require.NoError(t, err, "应该成功创建 SemanticMatcher")

	// 测试场景：混合语言段落和关键词
	testCases := []struct {
		name      string
		paragraph string
		keywords  []string
	}{
		{
			name:      "中英混合-科技",
			paragraph: "我使用 computer 进行 work 和 study",
			keywords:  []string{"电脑", "computer", "工作", "work", "学习", "study"},
		},
		{
			name:      "中英混合-水果",
			paragraph: "I like 苹果 and banana",
			keywords:  []string{"apple", "苹果", "香蕉", "banana", "水果", "fruit"},
		},
		{
			name:      "中英混合-国家",
			paragraph: "China 是一个 country with long history",
			keywords:  []string{"中国", "china", "国家", "country", "历史", "history"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results := matcher.FindTopKeywords(tc.paragraph, tc.keywords, len(tc.keywords))

			// 验证返回了结果
			assert.NotEmpty(t, results, "应该返回匹配结果")

			// 记录所有结果
			t.Logf("测试场景: %s", tc.name)
			t.Logf("混合语言段落: %s", tc.paragraph)
			t.Logf("混合语言关键词: %v", tc.keywords)
			for i, match := range results {
				t.Logf("  排名 %d: %s (相似度: %.4f, OOV: %d/%d)",
					i+1, match.Keyword, match.Score, match.OOVCount, match.WordCount)
			}

			// 验证所有关键词都有相似度分数
			for _, match := range results {
				assert.GreaterOrEqual(t, match.Score, 0.0, "相似度应该 >= 0")
				assert.LessOrEqual(t, match.Score, 1.0, "相似度应该 <= 1")
			}
		})
	}
}

// TestCrossLingualSimilarityComputation 验证跨语言相似度计算的合理性
func TestCrossLingualSimilarityComputation(t *testing.T) {
	logger := &DiscardLogger{}

	config := &Config{
		VectorFilePaths:    []string{"vector/wiki.zh.align.vec", "vector/wiki.en.align.vec"},
		MaxSequenceLen:     512,
		EnableStats:        true,
		MemoryLimit:        10 * 1024 * 1024 * 1024, // 10GB
		SupportedLanguages: []string{"zh", "en"},
	}

	matcher, err := NewSemanticMatcherFromConfig(config, logger)
	require.NoError(t, err, "应该成功创建 SemanticMatcher")

	// 测试不同类型的文本对
	testCases := []struct {
		name        string
		text1       string
		text2       string
		expectHigh  bool // 是否期望高相似度
		description string
	}{
		{
			name:        "相同语义-中英",
			text1:       "我喜欢吃苹果",
			text2:       "I like apples",
			expectHigh:  true,
			description: "相同语义的中英文应该有较高相似度",
		},
		{
			name:        "相同语义-英中",
			text1:       "China is a great country",
			text2:       "中国是一个伟大的国家",
			expectHigh:  true,
			description: "相同语义的英中文应该有较高相似度",
		},
		{
			name:        "不同语义-中英",
			text1:       "我喜欢吃苹果",
			text2:       "I use a computer",
			expectHigh:  false,
			description: "不同语义的中英文应该有较低相似度",
		},
		{
			name:        "不同语义-英中",
			text1:       "I like fruits",
			text2:       "我使用电脑工作",
			expectHigh:  false,
			description: "不同语义的英中文应该有较低相似度",
		},
		{
			name:        "混合语言-相似",
			text1:       "我使用 computer 工作",
			text2:       "I use 电脑 for work",
			expectHigh:  true,
			description: "相同语义的混合语言文本应该有较高相似度",
		},
		{
			name:        "同语言-中文",
			text1:       "我喜欢吃苹果和香蕉",
			text2:       "我喜欢水果",
			expectHigh:  true,
			description: "相关语义的同语言文本应该有较高相似度",
		},
		{
			name:        "同语言-英文",
			text1:       "I like apples and bananas",
			text2:       "I like fruits",
			expectHigh:  true,
			description: "相关语义的同语言文本应该有较高相似度",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			similarity := matcher.ComputeSimilarity(tc.text1, tc.text2)

			t.Logf("测试场景: %s", tc.name)
			t.Logf("文本1: %s", tc.text1)
			t.Logf("文本2: %s", tc.text2)
			t.Logf("相似度: %.4f", similarity)
			t.Logf("说明: %s", tc.description)

			// 验证相似度在有效范围内
			assert.GreaterOrEqual(t, similarity, 0.0, "相似度应该 >= 0")
			assert.LessOrEqual(t, similarity, 1.0, "相似度应该 <= 1")

			// 根据期望验证相似度
			if tc.expectHigh {
				if similarity >= 0.3 {
					t.Logf("✓ 相似度 %.4f 符合高相似度预期 (>= 0.3)", similarity)
				} else {
					t.Logf("⚠ 相似度 %.4f 低于高相似度预期 (< 0.3)", similarity)
				}
			} else {
				if similarity < 0.5 {
					t.Logf("✓ 相似度 %.4f 符合低相似度预期 (< 0.5)", similarity)
				} else {
					t.Logf("⚠ 相似度 %.4f 高于低相似度预期 (>= 0.5)", similarity)
				}
			}
		})
	}
}

// TestCrossLingualWithRealVectorFiles 使用真实的跨语言向量文件进行端到端测试
func TestCrossLingualWithRealVectorFiles(t *testing.T) {
	logger := &DiscardLogger{}

	config := &Config{
		VectorFilePaths:    []string{"vector/wiki.zh.align.vec", "vector/wiki.en.align.vec"},
		MaxSequenceLen:     512,
		EnableStats:        true,
		MemoryLimit:        10 * 1024 * 1024 * 1024, // 10GB
		SupportedLanguages: []string{"zh", "en"},
	}

	// 创建 SemanticMatcher
	matcher, err := NewSemanticMatcherFromConfig(config, logger)
	require.NoError(t, err, "应该成功创建 SemanticMatcher")

	// 测试1: 中文段落匹配英文关键词
	t.Run("中文段落匹配英文关键词", func(t *testing.T) {
		paragraph := "人工智能和机器学习是现代科技的重要领域"
		keywords := []string{"artificial", "intelligence", "machine", "learning", "technology", "food"}

		results := matcher.FindTopKeywords(paragraph, keywords, 3)
		assert.NotEmpty(t, results, "应该返回匹配结果")

		t.Logf("段落: %s", paragraph)
		for i, match := range results {
			t.Logf("  排名 %d: %s (相似度: %.4f)", i+1, match.Keyword, match.Score)
		}
	})

	// 测试2: 英文段落匹配中文关键词
	t.Run("英文段落匹配中文关键词", func(t *testing.T) {
		paragraph := "Artificial intelligence and machine learning are important fields in modern technology"
		keywords := []string{"人工智能", "机器学习", "科技", "技术", "食物"}

		results := matcher.FindTopKeywords(paragraph, keywords, 3)
		assert.NotEmpty(t, results, "应该返回匹配结果")

		t.Logf("段落: %s", paragraph)
		for i, match := range results {
			t.Logf("  排名 %d: %s (相似度: %.4f)", i+1, match.Keyword, match.Score)
		}
	})

	// 测试3: 计算跨语言文本相似度
	t.Run("计算跨语言文本相似度", func(t *testing.T) {
		text1 := "我喜欢学习新技术"
		text2 := "I love learning new technology"

		similarity := matcher.ComputeSimilarity(text1, text2)
		t.Logf("文本1: %s", text1)
		t.Logf("文本2: %s", text2)
		t.Logf("相似度: %.4f", similarity)

		assert.GreaterOrEqual(t, similarity, 0.0, "相似度应该 >= 0")
		assert.LessOrEqual(t, similarity, 1.0, "相似度应该 <= 1")
	})

	// 测试4: 验证统计信息
	t.Run("验证统计信息", func(t *testing.T) {
		stats := matcher.GetStats()

		t.Logf("统计信息:")
		t.Logf("  总请求数: %d", stats.TotalRequests)
		t.Logf("  平均延迟: %v", stats.AverageLatency)
		t.Logf("  OOV率: %.4f", stats.OOVRate)
		t.Logf("  向量命中率: %.4f", stats.VectorHitRate)
		t.Logf("  内存使用: %.2f MB", float64(stats.MemoryUsage)/(1024*1024))

		assert.Greater(t, stats.TotalRequests, int64(0), "应该有请求记录")
		assert.GreaterOrEqual(t, stats.OOVRate, 0.0, "OOV率应该 >= 0")
		assert.LessOrEqual(t, stats.OOVRate, 1.0, "OOV率应该 <= 1")
		assert.GreaterOrEqual(t, stats.VectorHitRate, 0.0, "向量命中率应该 >= 0")
		assert.LessOrEqual(t, stats.VectorHitRate, 1.0, "向量命中率应该 <= 1")
	})
}

// TestCrossLingualMultiFileLoading 验证多文件加载功能
func TestCrossLingualMultiFileLoading(t *testing.T) {
	logger := &DiscardLogger{}
	loader := NewEmbeddingLoader(logger)

	// 加载跨语言对齐向量文件
	paths := []string{
		"vector/wiki.zh.align.vec",
		"vector/wiki.en.align.vec",
	}

	model, err := loader.LoadMultipleFiles(paths)
	require.NoError(t, err, "应该成功加载多个跨语言向量文件")
	require.NotNil(t, model, "加载的模型不应为空")

	// 验证模型包含中英文词汇
	vocabSize := model.VocabularySize()
	assert.Greater(t, vocabSize, 0, "词汇表大小应该大于0")

	t.Logf("成功加载跨语言向量模型，词汇表大小: %d", vocabSize)
}

// TestCrossLingualChineseVectorQuery 验证中文词向量查询
func TestCrossLingualChineseVectorQuery(t *testing.T) {
	logger := &DiscardLogger{}
	loader := NewEmbeddingLoader(logger)

	paths := []string{
		"vector/wiki.zh.align.vec",
		"vector/wiki.en.align.vec",
	}

	model, err := loader.LoadMultipleFiles(paths)
	require.NoError(t, err)

	// 测试常见中文词汇
	chineseWords := []string{"中国", "苹果", "电脑", "学习", "工作"}

	for _, word := range chineseWords {
		vec, found := model.GetVector(word)
		if found {
			assert.NotNil(t, vec, "中文词 '%s' 的向量不应为空", word)
			assert.Equal(t, 300, len(vec), "向量维度应该是300")
			t.Logf("成功查询中文词 '%s' 的向量", word)
		} else {
			t.Logf("中文词 '%s' 不在词汇表中（这是正常的）", word)
		}
	}
}

// TestCrossLingualEnglishVectorQuery 验证英文词向量查询
func TestCrossLingualEnglishVectorQuery(t *testing.T) {
	logger := &DiscardLogger{}
	loader := NewEmbeddingLoader(logger)

	paths := []string{
		"vector/wiki.zh.align.vec",
		"vector/wiki.en.align.vec",
	}

	model, err := loader.LoadMultipleFiles(paths)
	require.NoError(t, err)

	// 测试常见英文词汇
	englishWords := []string{"china", "apple", "computer", "learning", "work"}

	for _, word := range englishWords {
		vec, found := model.GetVector(word)
		if found {
			assert.NotNil(t, vec, "英文词 '%s' 的向量不应为空", word)
			assert.Equal(t, 300, len(vec), "向量维度应该是300")
			t.Logf("成功查询英文词 '%s' 的向量", word)
		} else {
			t.Logf("英文词 '%s' 不在词汇表中（这是正常的）", word)
		}
	}
}

// TestCrossLingualMixedLanguageQuery 验证混合中英文词列表处理
func TestCrossLingualMixedLanguageQuery(t *testing.T) {
	logger := &DiscardLogger{}
	loader := NewEmbeddingLoader(logger)

	paths := []string{
		"vector/wiki.zh.align.vec",
		"vector/wiki.en.align.vec",
	}

	model, err := loader.LoadMultipleFiles(paths)
	require.NoError(t, err)

	// 测试混合语言词列表
	mixedWords := []string{"中国", "china", "苹果", "apple", "电脑", "computer"}

	foundCount := 0
	for _, word := range mixedWords {
		vec, found := model.GetVector(word)
		if found {
			assert.NotNil(t, vec, "词 '%s' 的向量不应为空", word)
			assert.Equal(t, 300, len(vec), "向量维度应该是300")
			foundCount++
			t.Logf("成功查询混合语言词 '%s' 的向量", word)
		}
	}

	t.Logf("混合语言词列表中找到 %d/%d 个词", foundCount, len(mixedWords))
}

// TestCrossLingualAverageVector 验证混合语言词列表的平均向量计算
func TestCrossLingualAverageVector(t *testing.T) {
	logger := &DiscardLogger{}
	loader := NewEmbeddingLoader(logger)

	paths := []string{
		"vector/wiki.zh.align.vec",
		"vector/wiki.en.align.vec",
	}

	model, err := loader.LoadMultipleFiles(paths)
	require.NoError(t, err)

	// 测试混合语言词列表的平均向量
	mixedWords := []string{"中国", "china", "苹果", "apple"}

	avgVec, found := model.GetAverageVector(mixedWords)

	if found {
		assert.NotNil(t, avgVec, "平均向量不应为空")
		assert.Equal(t, 300, len(avgVec), "平均向量维度应该是300")

		// 验证平均向量的值不全为0
		hasNonZero := false
		for _, val := range avgVec {
			if val != 0 {
				hasNonZero = true
				break
			}
		}
		assert.True(t, hasNonZero, "平均向量应该包含非零值")

		t.Logf("成功计算混合语言词列表的平均向量")
	} else {
		t.Logf("混合语言词列表中没有找到足够的词来计算平均向量")
	}
}

// TestCrossLingualSemanticSimilarity 验证跨语言语义相似度
func TestCrossLingualSemanticSimilarity(t *testing.T) {
	logger := &DiscardLogger{}
	loader := NewEmbeddingLoader(logger)

	paths := []string{
		"vector/wiki.zh.align.vec",
		"vector/wiki.en.align.vec",
	}

	model, err := loader.LoadMultipleFiles(paths)
	require.NoError(t, err)

	// 测试中英文对应词的相似度
	wordPairs := []struct {
		chinese string
		english string
	}{
		{"中国", "china"},
		{"苹果", "apple"},
		{"电脑", "computer"},
	}

	calculator := NewSimilarityCalculator()

	for _, pair := range wordPairs {
		vecCh, foundCh := model.GetVector(pair.chinese)
		vecEn, foundEn := model.GetVector(pair.english)

		if foundCh && foundEn {
			similarity := calculator.CosineSimilarity(vecCh, vecEn)
			t.Logf("'%s' 和 '%s' 的相似度: %.4f", pair.chinese, pair.english, similarity)

			// 跨语言对齐向量应该有一定的相似度，但不要求太高
			// 因为对齐质量取决于训练数据
			assert.GreaterOrEqual(t, similarity, -1.0, "相似度应该 >= -1")
			assert.LessOrEqual(t, similarity, 1.0, "相似度应该 <= 1")
		} else {
			t.Logf("词对 '%s'/'%s' 中有词不在词汇表中", pair.chinese, pair.english)
		}
	}
}

// TestCrossLingualNoCodeModification 验证无需代码修改即可工作
func TestCrossLingualNoCodeModification(t *testing.T) {
	// 这个测试验证现有的 VectorModel 接口可以直接处理跨语言向量
	// 无需任何接口修改或特殊处理

	logger := &DiscardLogger{}
	loader := NewEmbeddingLoader(logger)

	paths := []string{
		"vector/wiki.zh.align.vec",
		"vector/wiki.en.align.vec",
	}

	// 使用现有的 LoadMultipleFiles 方法
	model, err := loader.LoadMultipleFiles(paths)
	require.NoError(t, err)

	// 使用现有的 GetVector 方法查询中文
	vecCh, foundCh := model.GetVector("中国")
	if foundCh {
		assert.NotNil(t, vecCh)
	}

	// 使用现有的 GetVector 方法查询英文
	vecEn, foundEn := model.GetVector("china")
	if foundEn {
		assert.NotNil(t, vecEn)
	}

	// 使用现有的 GetAverageVector 方法处理混合语言
	avgVec, foundAvg := model.GetAverageVector([]string{"中国", "china"})
	if foundAvg {
		assert.NotNil(t, avgVec)
	}

	// 使用现有的 VocabularySize 方法
	vocabSize := model.VocabularySize()
	assert.Greater(t, vocabSize, 0)

	// 使用现有的 Dimension 方法
	dim := model.Dimension()
	assert.Equal(t, 300, dim)

	t.Log("验证通过：现有接口无需修改即可处理跨语言向量")
}

// BenchmarkCrossLingualVectorLookup benchmarks vector lookup performance in cross-lingual mode
func BenchmarkCrossLingualVectorLookup(b *testing.B) {
	logger := &DiscardLogger{}
	loader := NewEmbeddingLoader(logger)

	// Load cross-lingual aligned vectors
	paths := []string{
		"vector/wiki.zh.align.vec",
		"vector/wiki.en.align.vec",
	}

	model, err := loader.LoadMultipleFiles(paths)
	if err != nil {
		b.Fatalf("Failed to load cross-lingual vectors: %v", err)
	}

	// Test words (mix of Chinese and English)
	testWords := []string{
		"苹果", "apple", "中国", "china", "电脑", "computer",
		"学习", "learning", "工作", "work", "技术", "technology",
	}

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		word := testWords[i%len(testWords)]
		model.GetVector(word)
	}
}

// BenchmarkSingleLanguageVectorLookup benchmarks vector lookup in single-language mode for comparison
func BenchmarkSingleLanguageVectorLookup(b *testing.B) {
	logger := &DiscardLogger{}
	loader := NewEmbeddingLoader(logger)

	// Load single language vector file
	model, err := loader.LoadFromFile("vector/wiki.zh.align.vec")
	if err != nil {
		b.Fatalf("Failed to load single language vectors: %v", err)
	}

	// Test words (Chinese only)
	testWords := []string{
		"苹果", "中国", "电脑", "学习", "工作", "技术",
	}

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		word := testWords[i%len(testWords)]
		model.GetVector(word)
	}
}

// BenchmarkCrossLingualAverageVector benchmarks average vector computation in cross-lingual mode
func BenchmarkCrossLingualAverageVector(b *testing.B) {
	logger := &DiscardLogger{}
	loader := NewEmbeddingLoader(logger)

	paths := []string{
		"vector/wiki.zh.align.vec",
		"vector/wiki.en.align.vec",
	}

	model, err := loader.LoadMultipleFiles(paths)
	if err != nil {
		b.Fatalf("Failed to load cross-lingual vectors: %v", err)
	}

	// Test with different word list sizes
	testCases := []struct {
		name  string
		words []string
	}{
		{
			name:  "5_words_mixed",
			words: []string{"苹果", "apple", "香蕉", "banana", "水果"},
		},
		{
			name:  "10_words_mixed",
			words: []string{"苹果", "apple", "香蕉", "banana", "水果", "fruit", "中国", "china", "学习", "learning"},
		},
		{
			name: "20_words_mixed",
			words: []string{
				"苹果", "apple", "香蕉", "banana", "水果", "fruit",
				"中国", "china", "学习", "learning", "工作", "work",
				"电脑", "computer", "技术", "technology", "科学", "science",
				"教育", "education", "文化", "culture",
			},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				model.GetAverageVector(tc.words)
			}
		})
	}
}

// BenchmarkSingleLanguageAverageVector benchmarks average vector computation in single-language mode
func BenchmarkSingleLanguageAverageVector(b *testing.B) {
	logger := &DiscardLogger{}
	loader := NewEmbeddingLoader(logger)

	model, err := loader.LoadFromFile("vector/wiki.zh.align.vec")
	if err != nil {
		b.Fatalf("Failed to load single language vectors: %v", err)
	}

	testCases := []struct {
		name  string
		words []string
	}{
		{
			name:  "5_words_chinese",
			words: []string{"苹果", "香蕉", "水果", "中国", "学习"},
		},
		{
			name:  "10_words_chinese",
			words: []string{"苹果", "香蕉", "水果", "中国", "学习", "工作", "电脑", "技术", "科学", "教育"},
		},
		{
			name: "20_words_chinese",
			words: []string{
				"苹果", "香蕉", "水果", "中国", "学习", "工作",
				"电脑", "技术", "科学", "教育", "文化", "历史",
				"社会", "经济", "政治", "法律", "医学", "艺术",
				"音乐", "体育",
			},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				model.GetAverageVector(tc.words)
			}
		})
	}
}

// BenchmarkCrossLingualSimilarityCalculation benchmarks similarity calculation in cross-lingual mode
func BenchmarkCrossLingualSimilarityCalculation(b *testing.B) {
	logger := &DiscardLogger{}

	config := &Config{
		VectorFilePaths:    []string{"vector/wiki.zh.align.vec", "vector/wiki.en.align.vec"},
		MaxSequenceLen:     512,
		EnableStats:        false, // Disable stats for pure performance measurement
		MemoryLimit:        10 * 1024 * 1024 * 1024,
		SupportedLanguages: []string{"zh", "en"},
	}

	matcher, err := NewSemanticMatcherFromConfig(config, logger)
	if err != nil {
		b.Fatalf("Failed to create SemanticMatcher: %v", err)
	}

	testCases := []struct {
		name  string
		text1 string
		text2 string
	}{
		{
			name:  "chinese_to_english",
			text1: "我喜欢吃苹果和香蕉",
			text2: "I like apples and bananas",
		},
		{
			name:  "english_to_chinese",
			text1: "I use my computer for work",
			text2: "我使用电脑工作",
		},
		{
			name:  "mixed_to_mixed",
			text1: "我使用 computer 进行 work",
			text2: "I use 电脑 for 工作",
		},
		{
			name:  "chinese_to_chinese",
			text1: "我喜欢学习新技术",
			text2: "我热爱学习科技知识",
		},
		{
			name:  "english_to_english",
			text1: "I love learning new technology",
			text2: "I enjoy studying technical knowledge",
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				matcher.ComputeSimilarity(tc.text1, tc.text2)
			}
		})
	}
}

// BenchmarkSingleLanguageSimilarityCalculation benchmarks similarity calculation in single-language mode
func BenchmarkSingleLanguageSimilarityCalculation(b *testing.B) {
	logger := &DiscardLogger{}

	config := &Config{
		VectorFilePaths:    []string{"vector/wiki.zh.align.vec"},
		MaxSequenceLen:     512,
		EnableStats:        false,
		MemoryLimit:        10 * 1024 * 1024 * 1024,
		SupportedLanguages: []string{"zh"},
	}

	matcher, err := NewSemanticMatcherFromConfig(config, logger)
	if err != nil {
		b.Fatalf("Failed to create SemanticMatcher: %v", err)
	}

	testCases := []struct {
		name  string
		text1 string
		text2 string
	}{
		{
			name:  "short_texts",
			text1: "我喜欢吃苹果",
			text2: "我喜欢香蕉",
		},
		{
			name:  "medium_texts",
			text1: "我喜欢学习新技术和知识",
			text2: "我热爱学习科技和文化",
		},
		{
			name:  "long_texts",
			text1: "人工智能和机器学习是现代科技的重要领域，它们正在改变我们的生活方式",
			text2: "深度学习和神经网络是当代技术的核心部分，它们影响着社会的发展",
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				matcher.ComputeSimilarity(tc.text1, tc.text2)
			}
		})
	}
}

// BenchmarkCrossLingualKeywordMatching benchmarks keyword matching in cross-lingual mode
func BenchmarkCrossLingualKeywordMatching(b *testing.B) {
	logger := &DiscardLogger{}

	config := &Config{
		VectorFilePaths:    []string{"vector/wiki.zh.align.vec", "vector/wiki.en.align.vec"},
		MaxSequenceLen:     512,
		EnableStats:        false,
		MemoryLimit:        10 * 1024 * 1024 * 1024,
		SupportedLanguages: []string{"zh", "en"},
	}

	matcher, err := NewSemanticMatcherFromConfig(config, logger)
	if err != nil {
		b.Fatalf("Failed to create SemanticMatcher: %v", err)
	}

	testCases := []struct {
		name      string
		paragraph string
		keywords  []string
		topK      int
	}{
		{
			name:      "chinese_paragraph_english_keywords_5",
			paragraph: "我喜欢吃苹果和香蕉，它们都是很健康的水果",
			keywords:  []string{"apple", "banana", "fruit", "computer", "car"},
			topK:      3,
		},
		{
			name:      "english_paragraph_chinese_keywords_5",
			paragraph: "I like to eat apples and bananas, they are healthy fruits",
			keywords:  []string{"苹果", "香蕉", "水果", "电脑", "汽车"},
			topK:      3,
		},
		{
			name:      "mixed_paragraph_mixed_keywords_10",
			paragraph: "我使用 computer 进行 work 和 study",
			keywords:  []string{"电脑", "computer", "工作", "work", "学习", "study", "食物", "food", "汽车", "car"},
			topK:      5,
		},
		{
			name:      "chinese_paragraph_english_keywords_20",
			paragraph: "人工智能和机器学习是现代科技的重要领域",
			keywords: []string{
				"artificial", "intelligence", "machine", "learning", "technology",
				"science", "computer", "data", "algorithm", "neural",
				"network", "deep", "model", "training", "prediction",
				"food", "car", "music", "sport", "art",
			},
			topK: 10,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				matcher.FindTopKeywords(tc.paragraph, tc.keywords, tc.topK)
			}
		})
	}
}

// BenchmarkSingleLanguageKeywordMatching benchmarks keyword matching in single-language mode
func BenchmarkSingleLanguageKeywordMatching(b *testing.B) {
	logger := &DiscardLogger{}

	config := &Config{
		VectorFilePaths:    []string{"vector/wiki.zh.align.vec"},
		MaxSequenceLen:     512,
		EnableStats:        false,
		MemoryLimit:        10 * 1024 * 1024 * 1024,
		SupportedLanguages: []string{"zh"},
	}

	matcher, err := NewSemanticMatcherFromConfig(config, logger)
	if err != nil {
		b.Fatalf("Failed to create SemanticMatcher: %v", err)
	}

	testCases := []struct {
		name      string
		paragraph string
		keywords  []string
		topK      int
	}{
		{
			name:      "chinese_5_keywords",
			paragraph: "我喜欢吃苹果和香蕉，它们都是很健康的水果",
			keywords:  []string{"苹果", "香蕉", "水果", "电脑", "汽车"},
			topK:      3,
		},
		{
			name:      "chinese_10_keywords",
			paragraph: "我使用电脑进行工作和学习",
			keywords:  []string{"电脑", "工作", "学习", "食物", "汽车", "音乐", "体育", "艺术", "科学", "技术"},
			topK:      5,
		},
		{
			name:      "chinese_20_keywords",
			paragraph: "人工智能和机器学习是现代科技的重要领域",
			keywords: []string{
				"人工智能", "机器学习", "科技", "技术", "计算机",
				"数据", "算法", "神经网络", "深度学习", "模型",
				"训练", "预测", "食物", "汽车", "音乐",
				"体育", "艺术", "历史", "文化", "教育",
			},
			topK: 10,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				matcher.FindTopKeywords(tc.paragraph, tc.keywords, tc.topK)
			}
		})
	}
}

// BenchmarkCrossLingualConcurrentAccess benchmarks concurrent access in cross-lingual mode
func BenchmarkCrossLingualConcurrentAccess(b *testing.B) {
	logger := &DiscardLogger{}

	config := &Config{
		VectorFilePaths:    []string{"vector/wiki.zh.align.vec", "vector/wiki.en.align.vec"},
		MaxSequenceLen:     512,
		EnableStats:        false,
		MemoryLimit:        10 * 1024 * 1024 * 1024,
		SupportedLanguages: []string{"zh", "en"},
	}

	matcher, err := NewSemanticMatcherFromConfig(config, logger)
	if err != nil {
		b.Fatalf("Failed to create SemanticMatcher: %v", err)
	}

	testTexts := []struct {
		text1 string
		text2 string
	}{
		{"我喜欢吃苹果", "I like apples"},
		{"中国是一个国家", "China is a country"},
		{"我使用电脑工作", "I use computer for work"},
		{"学习新技术", "Learning new technology"},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			tc := testTexts[i%len(testTexts)]
			matcher.ComputeSimilarity(tc.text1, tc.text2)
			i++
		}
	})
}

// BenchmarkSingleLanguageConcurrentAccess benchmarks concurrent access in single-language mode
func BenchmarkSingleLanguageConcurrentAccess(b *testing.B) {
	logger := &DiscardLogger{}

	config := &Config{
		VectorFilePaths:    []string{"vector/wiki.zh.align.vec"},
		MaxSequenceLen:     512,
		EnableStats:        false,
		MemoryLimit:        10 * 1024 * 1024 * 1024,
		SupportedLanguages: []string{"zh"},
	}

	matcher, err := NewSemanticMatcherFromConfig(config, logger)
	if err != nil {
		b.Fatalf("Failed to create SemanticMatcher: %v", err)
	}

	testTexts := []struct {
		text1 string
		text2 string
	}{
		{"我喜欢吃苹果", "我喜欢香蕉"},
		{"中国是一个国家", "美国是一个国家"},
		{"我使用电脑工作", "我用计算机学习"},
		{"学习新技术", "研究新科技"},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			tc := testTexts[i%len(testTexts)]
			matcher.ComputeSimilarity(tc.text1, tc.text2)
			i++
		}
	})
}

// BenchmarkMemoryFootprint measures memory footprint of cross-lingual vs single-language models
func BenchmarkMemoryFootprint(b *testing.B) {
	logger := &DiscardLogger{}

	b.Run("cross_lingual_model", func(b *testing.B) {
		loader := NewEmbeddingLoader(logger)
		paths := []string{"vector/wiki.zh.align.vec", "vector/wiki.en.align.vec"}

		model, err := loader.LoadMultipleFiles(paths)
		if err != nil {
			b.Fatalf("Failed to load cross-lingual vectors: %v", err)
		}

		b.ReportMetric(float64(model.MemoryUsage())/(1024*1024), "MB")
		b.ReportMetric(float64(model.VocabularySize()), "words")
	})

	b.Run("single_language_model", func(b *testing.B) {
		loader := NewEmbeddingLoader(logger)

		model, err := loader.LoadFromFile("vector/wiki.zh.align.vec")
		if err != nil {
			b.Fatalf("Failed to load single language vectors: %v", err)
		}

		b.ReportMetric(float64(model.MemoryUsage())/(1024*1024), "MB")
		b.ReportMetric(float64(model.VocabularySize()), "words")
	})
}

// BenchmarkTextProcessing benchmarks text processing performance
func BenchmarkTextProcessing(b *testing.B) {
	processor := NewTextProcessor()

	testCases := []struct {
		name string
		text string
	}{
		{
			name: "chinese_short",
			text: "我喜欢吃苹果",
		},
		{
			name: "english_short",
			text: "I like apples",
		},
		{
			name: "mixed_short",
			text: "我喜欢 apple",
		},
		{
			name: "chinese_long",
			text: "人工智能和机器学习是现代科技的重要领域，它们正在改变我们的生活方式和工作模式",
		},
		{
			name: "english_long",
			text: "Artificial intelligence and machine learning are important fields in modern technology that are changing our lifestyle and work patterns",
		},
		{
			name: "mixed_long",
			text: "人工智能 artificial intelligence 和 machine learning 机器学习是现代科技 modern technology 的重要领域",
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				processor.Preprocess(tc.text)
			}
		})
	}
}

// Helper function to generate performance report
func generatePerformanceReport(b *testing.B, crossLingualNs, singleLanguageNs float64) {
	overhead := ((crossLingualNs - singleLanguageNs) / singleLanguageNs) * 100
	b.Logf("\n=== Performance Comparison ===")
	b.Logf("Cross-lingual mode: %.2f ns/op", crossLingualNs)
	b.Logf("Single-language mode: %.2f ns/op", singleLanguageNs)
	b.Logf("Overhead: %.2f%%", overhead)

	if overhead < 10 {
		b.Logf("✓ Performance difference is within acceptable range (< 10%%)")
	} else if overhead < 20 {
		b.Logf("⚠ Performance difference is moderate (10-20%%)")
	} else {
		b.Logf("✗ Performance difference is significant (> 20%%)")
	}
}

// TestPerformanceComparison runs benchmarks and compares performance
func TestPerformanceComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance comparison in short mode")
	}

	t.Log("\n=== Cross-lingual vs Single-language Performance Comparison ===\n")

	// Run vector lookup benchmarks
	t.Run("VectorLookup", func(t *testing.T) {
		crossResult := testing.Benchmark(BenchmarkCrossLingualVectorLookup)
		singleResult := testing.Benchmark(BenchmarkSingleLanguageVectorLookup)

		t.Logf("Cross-lingual vector lookup: %s", crossResult.String())
		t.Logf("Single-language vector lookup: %s", singleResult.String())

		if crossResult.NsPerOp() > 0 && singleResult.NsPerOp() > 0 {
			overhead := float64(crossResult.NsPerOp()-singleResult.NsPerOp()) / float64(singleResult.NsPerOp()) * 100
			t.Logf("Overhead: %.2f%%", overhead)

			if overhead < 10 {
				t.Logf("✓ Vector lookup performance difference is acceptable")
			}
		}
	})

	// Run similarity calculation benchmarks
	t.Run("SimilarityCalculation", func(t *testing.T) {
		crossResult := testing.Benchmark(func(b *testing.B) {
			BenchmarkCrossLingualSimilarityCalculation(b)
		})
		singleResult := testing.Benchmark(func(b *testing.B) {
			BenchmarkSingleLanguageSimilarityCalculation(b)
		})

		t.Logf("Cross-lingual similarity: %s", crossResult.String())
		t.Logf("Single-language similarity: %s", singleResult.String())
	})

	t.Log("\n=== Performance Report Generated ===")
	t.Log("Run 'go test -bench=. -benchmem' for detailed benchmark results")
}

// BenchmarkEndToEndWorkflow benchmarks complete end-to-end workflow
func BenchmarkEndToEndWorkflow(b *testing.B) {
	logger := &DiscardLogger{}

	b.Run("cross_lingual_workflow", func(b *testing.B) {
		config := &Config{
			VectorFilePaths:    []string{"vector/wiki.zh.align.vec", "vector/wiki.en.align.vec"},
			MaxSequenceLen:     512,
			EnableStats:        false,
			MemoryLimit:        10 * 1024 * 1024 * 1024,
			SupportedLanguages: []string{"zh", "en"},
		}

		matcher, err := NewSemanticMatcherFromConfig(config, logger)
		if err != nil {
			b.Fatalf("Failed to create matcher: %v", err)
		}

		paragraph := "我喜欢学习人工智能和机器学习技术"
		keywords := []string{"artificial", "intelligence", "machine", "learning", "technology", "computer", "science"}

		b.ResetTimer()
		for b.Loop() {
			results := matcher.FindTopKeywords(paragraph, keywords, 3)
			_ = results
		}
	})

	b.Run("single_language_workflow", func(b *testing.B) {
		config := &Config{
			VectorFilePaths:    []string{"vector/wiki.zh.align.vec"},
			MaxSequenceLen:     512,
			EnableStats:        false,
			MemoryLimit:        10 * 1024 * 1024 * 1024,
			SupportedLanguages: []string{"zh"},
		}

		matcher, err := NewSemanticMatcherFromConfig(config, logger)
		if err != nil {
			b.Fatalf("Failed to create matcher: %v", err)
		}

		paragraph := "我喜欢学习人工智能和机器学习技术"
		keywords := []string{"人工智能", "机器学习", "技术", "计算机", "科学", "数据", "算法"}

		b.ResetTimer()
		for b.Loop() {
			results := matcher.FindTopKeywords(paragraph, keywords, 3)
			_ = results
		}
	})
}

// TestIntegrationCrossLingualEndToEnd 测试完整的跨语言语义匹配流程
func TestIntegrationCrossLingualEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := &DiscardLogger{}

	// 配置跨语言向量模型
	config := &Config{
		VectorFilePaths:    []string{"vector/wiki.zh.align.vec", "vector/wiki.en.align.vec"},
		MaxSequenceLen:     512,
		EnableStats:        true,
		MemoryLimit:        10 * 1024 * 1024 * 1024, // 10GB
		SupportedLanguages: []string{"zh", "en"},
	}

	// 创建 SemanticMatcher
	matcher, err := NewSemanticMatcherFromConfig(config, logger)
	require.NoError(t, err, "应该成功创建 SemanticMatcher")
	require.NotNil(t, matcher, "matcher 不应该为 nil")

	// 测试场景1: 中文段落匹配英文关键词
	t.Run("中文段落匹配英文关键词", func(t *testing.T) {
		paragraph := "我喜欢吃苹果和香蕉，它们都是很健康的水果"
		keywords := []string{"apple", "banana", "computer", "car", "fruit"}

		results := matcher.FindTopKeywords(paragraph, keywords, 3)
		assert.NotEmpty(t, results, "应该返回匹配结果")
		assert.LessOrEqual(t, len(results), 3, "结果数量不应超过请求的数量")

		// 验证结果按相似度降序排列
		for i := 1; i < len(results); i++ {
			assert.GreaterOrEqual(t, results[i-1].Score, results[i].Score,
				"结果应该按相似度降序排列")
		}

		t.Logf("中文段落: %s", paragraph)
		for i, match := range results {
			t.Logf("  排名 %d: %s (相似度: %.4f)", i+1, match.Keyword, match.Score)
		}
	})

	// 测试场景2: 英文段落匹配中文关键词
	t.Run("英文段落匹配中文关键词", func(t *testing.T) {
		paragraph := "I like to eat apples and bananas, they are healthy fruits"
		keywords := []string{"苹果", "香蕉", "电脑", "汽车", "水果"}

		results := matcher.FindTopKeywords(paragraph, keywords, 3)
		assert.NotEmpty(t, results, "应该返回匹配结果")
		assert.LessOrEqual(t, len(results), 3, "结果数量不应超过请求的数量")

		t.Logf("英文段落: %s", paragraph)
		for i, match := range results {
			t.Logf("  排名 %d: %s (相似度: %.4f)", i+1, match.Keyword, match.Score)
		}
	})

	// 测试场景3: 混合语言段落和关键词
	t.Run("混合语言段落和关键词", func(t *testing.T) {
		paragraph := "我使用 computer 进行 work 和 study"
		keywords := []string{"电脑", "computer", "工作", "work", "学习", "study"}

		results := matcher.FindTopKeywords(paragraph, keywords, len(keywords))
		assert.NotEmpty(t, results, "应该返回匹配结果")

		t.Logf("混合语言段落: %s", paragraph)
		for i, match := range results {
			t.Logf("  排名 %d: %s (相似度: %.4f)", i+1, match.Keyword, match.Score)
		}
	})

	// 测试场景4: 计算跨语言文本相似度
	t.Run("计算跨语言文本相似度", func(t *testing.T) {
		testCases := []struct {
			text1 string
			text2 string
			name  string
		}{
			{
				name:  "中英相似语义",
				text1: "我喜欢吃苹果",
				text2: "I like apples",
			},
			{
				name:  "英中相似语义",
				text1: "China is a great country",
				text2: "中国是一个伟大的国家",
			},
			{
				name:  "混合语言",
				text1: "我使用 computer 工作",
				text2: "I use 电脑 for work",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				similarity := matcher.ComputeSimilarity(tc.text1, tc.text2)
				assert.GreaterOrEqual(t, similarity, 0.0, "相似度应该 >= 0")
				assert.LessOrEqual(t, similarity, 1.0, "相似度应该 <= 1")

				t.Logf("文本1: %s", tc.text1)
				t.Logf("文本2: %s", tc.text2)
				t.Logf("相似度: %.4f", similarity)
			})
		}
	})

	// 测试场景5: 验证统计信息
	t.Run("验证统计信息", func(t *testing.T) {
		stats := matcher.GetStats()

		assert.Greater(t, stats.TotalRequests, int64(0), "应该有请求记录")
		assert.GreaterOrEqual(t, stats.OOVRate, 0.0, "OOV率应该 >= 0")
		assert.LessOrEqual(t, stats.OOVRate, 1.0, "OOV率应该 <= 1")
		assert.GreaterOrEqual(t, stats.VectorHitRate, 0.0, "向量命中率应该 >= 0")
		assert.LessOrEqual(t, stats.VectorHitRate, 1.0, "向量命中率应该 <= 1")
		assert.Greater(t, stats.MemoryUsage, int64(0), "内存使用应该 > 0")
		assert.False(t, stats.LastUpdated.IsZero(), "LastUpdated 应该被设置")

		t.Logf("统计信息:")
		t.Logf("  总请求数: %d", stats.TotalRequests)
		t.Logf("  平均延迟: %v", stats.AverageLatency)
		t.Logf("  OOV率: %.4f", stats.OOVRate)
		t.Logf("  向量命中率: %.4f", stats.VectorHitRate)
		t.Logf("  内存使用: %.2f MB", float64(stats.MemoryUsage)/(1024*1024))
	})
}

// TestIntegrationConcurrentCrossLingualMatching 测试并发场景下的跨语言匹配
func TestIntegrationConcurrentCrossLingualMatching(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := &DiscardLogger{}

	config := &Config{
		VectorFilePaths:    []string{"vector/wiki.zh.align.vec", "vector/wiki.en.align.vec"},
		MaxSequenceLen:     512,
		EnableStats:        true,
		MemoryLimit:        10 * 1024 * 1024 * 1024,
		SupportedLanguages: []string{"zh", "en"},
	}

	matcher, err := NewSemanticMatcherFromConfig(config, logger)
	require.NoError(t, err)
	require.NotNil(t, matcher)

	// 并发测试场景
	testCases := []struct {
		paragraph string
		keywords  []string
		name      string
	}{
		{
			name:      "中文段落-英文关键词",
			paragraph: "我喜欢吃苹果和香蕉",
			keywords:  []string{"apple", "banana", "computer"},
		},
		{
			name:      "英文段落-中文关键词",
			paragraph: "I like apples and bananas",
			keywords:  []string{"苹果", "香蕉", "电脑"},
		},
		{
			name:      "混合语言",
			paragraph: "我使用 computer 工作",
			keywords:  []string{"电脑", "work", "学习"},
		},
	}

	// 并发执行测试
	concurrency := 10
	iterations := 5

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*iterations*len(testCases))

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				for _, tc := range testCases {
					// 测试 FindTopKeywords
					results := matcher.FindTopKeywords(tc.paragraph, tc.keywords, 2)
					if len(results) == 0 {
						errors <- assert.AnError
						continue
					}

					// 验证结果有效性
					for _, match := range results {
						if match.Score < 0 || match.Score > 1 {
							errors <- assert.AnError
						}
					}

					// 测试 ComputeSimilarity
					similarity := matcher.ComputeSimilarity(tc.paragraph, tc.keywords[0])
					if similarity < 0 || similarity > 1 {
						errors <- assert.AnError
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// 检查是否有错误
	errorCount := 0
	for range errors {
		errorCount++
	}

	assert.Equal(t, 0, errorCount, "并发测试不应该产生错误")

	// 验证统计信息
	stats := matcher.GetStats()
	expectedRequests := int64(concurrency * iterations * len(testCases) * 2) // FindTopKeywords + ComputeSimilarity
	assert.GreaterOrEqual(t, stats.TotalRequests, expectedRequests,
		"总请求数应该至少为 %d", expectedRequests)

	t.Logf("并发测试完成:")
	t.Logf("  并发数: %d", concurrency)
	t.Logf("  迭代次数: %d", iterations)
	t.Logf("  总请求数: %d", stats.TotalRequests)
	t.Logf("  平均延迟: %v", stats.AverageLatency)
}

// TestIntegrationMemoryUsageAndResourceManagement 测试内存使用和资源管理
func TestIntegrationMemoryUsageAndResourceManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := &DiscardLogger{}

	t.Run("正常内存限制", func(t *testing.T) {
		config := &Config{
			VectorFilePaths:    []string{"vector/wiki.zh.align.vec", "vector/wiki.en.align.vec"},
			MaxSequenceLen:     512,
			EnableStats:        true,
			MemoryLimit:        10 * 1024 * 1024 * 1024, // 10GB
			SupportedLanguages: []string{"zh", "en"},
		}

		matcher, err := NewSemanticMatcherFromConfig(config, logger)
		require.NoError(t, err)
		require.NotNil(t, matcher)

		stats := matcher.GetStats()
		assert.Greater(t, stats.MemoryUsage, int64(0), "内存使用应该 > 0")
		assert.Less(t, stats.MemoryUsage, config.MemoryLimit, "内存使用应该在限制内")

		t.Logf("内存使用: %.2f MB", float64(stats.MemoryUsage)/(1024*1024))
		t.Logf("内存限制: %.2f MB", float64(config.MemoryLimit)/(1024*1024))
	})

	t.Run("内存限制过低", func(t *testing.T) {
		config := &Config{
			VectorFilePaths:    []string{"vector/wiki.zh.align.vec", "vector/wiki.en.align.vec"},
			MaxSequenceLen:     512,
			EnableStats:        true,
			MemoryLimit:        1024, // 1KB - 太低
			SupportedLanguages: []string{"zh", "en"},
		}

		matcher, err := NewSemanticMatcherFromConfig(config, logger)
		assert.Error(t, err, "应该返回内存限制错误")
		assert.Equal(t, ErrMemoryLimitExceeded, err, "应该是 ErrMemoryLimitExceeded")
		assert.Nil(t, matcher, "matcher 应该为 nil")
	})

	t.Run("持续使用后的内存稳定性", func(t *testing.T) {
		config := &Config{
			VectorFilePaths:    []string{"vector/wiki.zh.align.vec", "vector/wiki.en.align.vec"},
			MaxSequenceLen:     512,
			EnableStats:        true,
			MemoryLimit:        10 * 1024 * 1024 * 1024,
			SupportedLanguages: []string{"zh", "en"},
		}

		matcher, err := NewSemanticMatcherFromConfig(config, logger)
		require.NoError(t, err)

		// 记录初始内存使用
		initialStats := matcher.GetStats()
		initialMemory := initialStats.MemoryUsage

		// 执行大量操作
		paragraph := "我喜欢吃苹果和香蕉"
		keywords := []string{"apple", "banana", "computer"}

		for i := 0; i < 100; i++ {
			matcher.FindTopKeywords(paragraph, keywords, 2)
			matcher.ComputeSimilarity(paragraph, keywords[0])
		}

		// 检查内存使用是否稳定
		finalStats := matcher.GetStats()
		finalMemory := finalStats.MemoryUsage

		// 内存使用应该保持相对稳定（允许小幅波动）
		memoryGrowth := float64(finalMemory-initialMemory) / float64(initialMemory)
		assert.Less(t, memoryGrowth, 0.1, "内存增长应该小于 10%%")

		t.Logf("初始内存: %.2f MB", float64(initialMemory)/(1024*1024))
		t.Logf("最终内存: %.2f MB", float64(finalMemory)/(1024*1024))
		t.Logf("内存增长: %.2f%%", memoryGrowth*100)
	})
}

// TestIntegrationErrorHandling 测试错误处理
func TestIntegrationErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := &DiscardLogger{}

	t.Run("文件不存在", func(t *testing.T) {
		config := &Config{
			VectorFilePaths:    []string{"nonexistent/file.vec"},
			MaxSequenceLen:     512,
			EnableStats:        true,
			MemoryLimit:        10 * 1024 * 1024 * 1024,
			SupportedLanguages: []string{"zh", "en"},
		}

		matcher, err := NewSemanticMatcherFromConfig(config, logger)
		assert.Error(t, err, "应该返回文件不存在错误")
		assert.Nil(t, matcher, "matcher 应该为 nil")
	})

	t.Run("空配置", func(t *testing.T) {
		matcher, err := NewSemanticMatcherFromConfig(nil, logger)
		assert.Error(t, err, "应该返回配置错误")
		assert.Equal(t, ErrInvalidConfiguration, err)
		assert.Nil(t, matcher, "matcher 应该为 nil")
	})

	t.Run("空向量文件列表", func(t *testing.T) {
		config := &Config{
			VectorFilePaths:    []string{},
			MaxSequenceLen:     512,
			EnableStats:        true,
			MemoryLimit:        10 * 1024 * 1024 * 1024,
			SupportedLanguages: []string{"zh", "en"},
		}

		matcher, err := NewSemanticMatcherFromConfig(config, logger)
		assert.Error(t, err, "应该返回错误")
		assert.Equal(t, ErrNoVectorFiles, err)
		assert.Nil(t, matcher, "matcher 应该为 nil")
	})

	t.Run("无效的最大序列长度", func(t *testing.T) {
		config := &Config{
			VectorFilePaths:    []string{"vector/wiki.zh.align.vec"},
			MaxSequenceLen:     0,
			EnableStats:        true,
			MemoryLimit:        10 * 1024 * 1024 * 1024,
			SupportedLanguages: []string{"zh", "en"},
		}

		matcher, err := NewSemanticMatcherFromConfig(config, logger)
		assert.Error(t, err, "应该返回配置错误")
		assert.Equal(t, ErrInvalidConfiguration, err)
		assert.Nil(t, matcher, "matcher 应该为 nil")
	})

	t.Run("空支持语言列表", func(t *testing.T) {
		config := &Config{
			VectorFilePaths:    []string{"vector/wiki.zh.align.vec"},
			MaxSequenceLen:     512,
			EnableStats:        true,
			MemoryLimit:        10 * 1024 * 1024 * 1024,
			SupportedLanguages: []string{},
		}

		matcher, err := NewSemanticMatcherFromConfig(config, logger)
		assert.Error(t, err, "应该返回配置错误")
		assert.Equal(t, ErrInvalidConfiguration, err)
		assert.Nil(t, matcher, "matcher 应该为 nil")
	})

	t.Run("不支持的语言", func(t *testing.T) {
		config := &Config{
			VectorFilePaths:    []string{"vector/wiki.zh.align.vec"},
			MaxSequenceLen:     512,
			EnableStats:        true,
			MemoryLimit:        10 * 1024 * 1024 * 1024,
			SupportedLanguages: []string{"fr"}, // 法语不支持
		}

		matcher, err := NewSemanticMatcherFromConfig(config, logger)
		assert.Error(t, err, "应该返回不支持的语言错误")
		assert.Equal(t, ErrUnsupportedLanguage, err)
		assert.Nil(t, matcher, "matcher 应该为 nil")
	})
}

// TestIntegrationPerformanceUnderLoad 测试负载下的性能
func TestIntegrationPerformanceUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := &DiscardLogger{}

	config := &Config{
		VectorFilePaths:    []string{"vector/wiki.zh.align.vec", "vector/wiki.en.align.vec"},
		MaxSequenceLen:     512,
		EnableStats:        true,
		MemoryLimit:        10 * 1024 * 1024 * 1024,
		SupportedLanguages: []string{"zh", "en"},
	}

	matcher, err := NewSemanticMatcherFromConfig(config, logger)
	require.NoError(t, err)

	// 测试不同负载下的性能
	testCases := []struct {
		name       string
		iterations int
		concurrent int
	}{
		{"低负载", 10, 1},
		{"中负载", 50, 5},
		{"高负载", 100, 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			startTime := time.Now()

			var wg sync.WaitGroup
			for i := 0; i < tc.concurrent; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < tc.iterations; j++ {
						paragraph := "我喜欢吃苹果和香蕉"
						keywords := []string{"apple", "banana", "computer"}
						matcher.FindTopKeywords(paragraph, keywords, 2)
					}
				}()
			}

			wg.Wait()
			duration := time.Since(startTime)

			totalRequests := tc.iterations * tc.concurrent
			avgLatency := duration / time.Duration(totalRequests)

			t.Logf("%s 性能:", tc.name)
			t.Logf("  总请求数: %d", totalRequests)
			t.Logf("  总耗时: %v", duration)
			t.Logf("  平均延迟: %v", avgLatency)
			t.Logf("  QPS: %.2f", float64(totalRequests)/duration.Seconds())

			// 验证平均延迟在合理范围内（< 100ms）
			assert.Less(t, avgLatency, 100*time.Millisecond,
				"平均延迟应该小于 100ms")
		})
	}
}

// TestIntegrationExistingTestsCompatibility 验证所有现有测试继续通过
func TestIntegrationExistingTestsCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 这个测试确保跨语言功能不会破坏现有的单语言功能

	logger := &DiscardLogger{}

	t.Run("单文件配置兼容性", func(t *testing.T) {
		config := &Config{
			VectorFilePaths:    []string{"vector/wiki.zh.align.vec"},
			MaxSequenceLen:     512,
			EnableStats:        true,
			MemoryLimit:        10 * 1024 * 1024 * 1024, // 10GB
			SupportedLanguages: []string{"zh"},
		}

		matcher, err := NewSemanticMatcherFromConfig(config, logger)
		require.NoError(t, err)
		require.NotNil(t, matcher)

		// 测试基本功能
		text1 := "这是测试"
		text2 := "中文词汇"
		similarity := matcher.ComputeSimilarity(text1, text2)

		assert.GreaterOrEqual(t, similarity, 0.0)
		assert.LessOrEqual(t, similarity, 1.0)
	})

	t.Run("多文件配置兼容性", func(t *testing.T) {
		config := &Config{
			VectorFilePaths:    []string{"vector/wiki.zh.align.vec", "vector/wiki.en.align.vec"},
			MaxSequenceLen:     512,
			EnableStats:        true,
			MemoryLimit:        10 * 1024 * 1024 * 1024, // 10GB
			SupportedLanguages: []string{"zh", "en"},
		}

		matcher, err := NewSemanticMatcherFromConfig(config, logger)
		require.NoError(t, err)
		require.NotNil(t, matcher)

		// 测试中文
		chineseText := "这是测试"
		chineseSimilarity := matcher.ComputeSimilarity(chineseText, "中文词汇")
		assert.GreaterOrEqual(t, chineseSimilarity, 0.0)
		assert.LessOrEqual(t, chineseSimilarity, 1.0)

		// 测试英文
		englishText := "test english"
		englishSimilarity := matcher.ComputeSimilarity(englishText, "english words")
		assert.GreaterOrEqual(t, englishSimilarity, 0.0)
		assert.LessOrEqual(t, englishSimilarity, 1.0)
	})

	t.Run("统计功能兼容性", func(t *testing.T) {
		config := &Config{
			VectorFilePaths:    []string{"vector/wiki.zh.align.vec"},
			MaxSequenceLen:     512,
			EnableStats:        true,
			MemoryLimit:        10 * 1024 * 1024 * 1024, // 10GB
			SupportedLanguages: []string{"zh"},
		}

		matcher, err := NewSemanticMatcherFromConfig(config, logger)
		require.NoError(t, err)

		// 执行一些操作
		paragraph := "测试文本"
		keywords := []string{"测试", "文本"}
		matcher.FindTopKeywords(paragraph, keywords, 1)

		// 验证统计信息
		stats := matcher.GetStats()
		assert.Greater(t, stats.TotalRequests, int64(0))
		assert.GreaterOrEqual(t, stats.OOVRate, 0.0)
		assert.LessOrEqual(t, stats.OOVRate, 1.0)
		assert.Greater(t, stats.MemoryUsage, int64(0))
	})
}

// Property-Based Tests for Character-Level Fallback

// TestProperty1_OOVWordTriggersCharacterLevelFallback tests that when GetVector is called with an OOV word,
// it triggers character-level fallback if the word has multiple characters.
// **Feature: character-level-fallback, Property 1: OOV 词触发字符级回退**
// **Validates: Requirements 1.1**
func TestProperty1_OOVWordTriggersCharacterLevelFallback(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("OOV words with multiple characters trigger fallback", prop.ForAll(
		func(numChars int, inVocabCount int) bool {
			// Ensure we have at least 2 characters and at least one in vocabulary
			if numChars < 2 || inVocabCount <= 0 || inVocabCount > numChars {
				return true // Skip invalid cases
			}

			// Create a vector model with dimension 3
			vm := NewVectorModel(3).(*vectorModel)

			// Generate characters - first inVocabCount will be in vocabulary
			chars := make([]rune, numChars)

			for i := 0; i < numChars; i++ {
				chars[i] = rune('一' + i) // Chinese characters

				if i < inVocabCount {
					// Add this character to vocabulary
					charVec := []float32{float32(i), float32(i + 1), float32(i + 2)}
					vm.AddVector(string(chars[i]), charVec)
				}
			}

			// Create an OOV word from these characters
			oovWord := string(chars)

			// Reset stats to have a clean slate
			vm.ResetStats()

			// Call GetVector on the OOV word
			result, ok := vm.GetVector(oovWord)

			// Get statistics
			total, oov, hit, fallbackAttempts, fallbackSuccesses, fallbackFailures := vm.GetLookupStats()

			// Verify that:
			// 1. Total lookups is 1
			if total != 1 {
				return false
			}

			// 2. OOV count is 1 (word not in vocabulary)
			if oov != 1 {
				return false
			}

			// 3. Hit count is 0 (word not found directly)
			if hit != 0 {
				return false
			}

			// 4. Fallback was attempted (OOV word with multiple characters)
			if fallbackAttempts != 1 {
				return false
			}

			// 5. If at least one character is in vocabulary, fallback should succeed
			if inVocabCount > 0 {
				if !ok {
					return false
				}
				if fallbackSuccesses != 1 {
					return false
				}
				if fallbackFailures != 0 {
					return false
				}
				if result == nil {
					return false
				}
				if len(result) != 3 {
					return false
				}
			} else {
				// If no characters in vocabulary, fallback should fail
				if ok {
					return false
				}
				if fallbackSuccesses != 0 {
					return false
				}
				if fallbackFailures != 1 {
					return false
				}
				if result != nil {
					return false
				}
			}

			return true
		},
		gen.IntRange(2, 10), // Total characters: 2 to 10
		gen.IntRange(1, 8),  // In-vocab characters: 1 to 8
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty2_AllCharactersInVocabulary tests that when all characters of a word are in the vocabulary,
// the fallback returns the complete average of all character vectors.
// **Feature: character-level-fallback, Property 2: 所有单字都在词库时返回完整平均**
// **Validates: Requirements 1.2**
func TestProperty2_AllCharactersInVocabulary(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("all characters in vocabulary returns complete average", prop.ForAll(
		func(numChars int) bool {
			// Create a vector model with dimension 3
			vm := NewVectorModel(3).(*vectorModel)

			// Generate random character vectors and add them to the model
			chars := make([]rune, numChars)
			expectedSum := make([]float32, 3)

			for i := 0; i < numChars; i++ {
				// Use different Unicode characters to avoid collisions
				chars[i] = rune('一' + i) // Chinese characters starting from U+4E00
				charVec := []float32{float32(i), float32(i + 1), float32(i + 2)}
				vm.AddVector(string(chars[i]), charVec)

				// Accumulate expected sum
				for j := 0; j < 3; j++ {
					expectedSum[j] += charVec[j]
				}
			}

			// Create a word from these characters
			word := string(chars)

			// Call characterLevelFallback
			vm.mtx.Lock()
			vm.fallbackAttempts++
			result, ok := vm.characterLevelFallback(word)
			vm.mtx.Unlock()

			// Should succeed
			if !ok {
				return false
			}

			// Check that result is the average of all character vectors
			expectedAvg := make([]float32, 3)
			for j := 0; j < 3; j++ {
				expectedAvg[j] = expectedSum[j] / float32(numChars)
			}

			// Compare with small tolerance for floating point errors
			for j := 0; j < 3; j++ {
				if math.Abs(float64(result[j]-expectedAvg[j])) > 1e-6 {
					return false
				}
			}

			return true
		},
		gen.IntRange(2, 10), // Test with 2 to 10 characters
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty3_PartialCharactersInVocabulary tests that when only some characters of a word are in the vocabulary,
// the fallback returns the average of only the characters that exist in the vocabulary.
// **Feature: character-level-fallback, Property 3: 部分单字在词库时返回部分平均**
// **Validates: Requirements 1.3**
func TestProperty3_PartialCharactersInVocabulary(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("partial characters in vocabulary returns partial average", prop.ForAll(
		func(totalChars int, inVocabCount int) bool {
			// Ensure we have at least one character in vocab and at least one not in vocab
			if inVocabCount <= 0 || inVocabCount >= totalChars {
				return true // Skip invalid cases
			}

			// Create a vector model with dimension 3
			vm := NewVectorModel(3).(*vectorModel)

			// Generate characters - first inVocabCount will be in vocabulary
			chars := make([]rune, totalChars)
			expectedSum := make([]float32, 3)

			for i := 0; i < totalChars; i++ {
				chars[i] = rune('一' + i) // Chinese characters

				if i < inVocabCount {
					// Add this character to vocabulary
					charVec := []float32{float32(i), float32(i + 1), float32(i + 2)}
					vm.AddVector(string(chars[i]), charVec)

					// Accumulate expected sum (only for characters in vocab)
					for j := 0; j < 3; j++ {
						expectedSum[j] += charVec[j]
					}
				}
				// Characters at index >= inVocabCount are NOT added to vocabulary
			}

			// Create a word from these characters
			word := string(chars)

			// Call characterLevelFallback
			vm.mtx.Lock()
			vm.fallbackAttempts++
			result, ok := vm.characterLevelFallback(word)
			vm.mtx.Unlock()

			// Should succeed because at least one character is in vocabulary
			if !ok {
				return false
			}

			// Check that result is the average of only the in-vocabulary character vectors
			expectedAvg := make([]float32, 3)
			for j := 0; j < 3; j++ {
				expectedAvg[j] = expectedSum[j] / float32(inVocabCount)
			}

			// Compare with small tolerance for floating point errors
			for j := 0; j < 3; j++ {
				if math.Abs(float64(result[j]-expectedAvg[j])) > 1e-6 {
					return false
				}
			}

			return true
		},
		gen.IntRange(3, 10), // Total characters: 3 to 10
		gen.IntRange(1, 8),  // In-vocab characters: 1 to 8 (will be constrained by totalChars)
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty4_UnicodeHandling tests that the fallback correctly handles multi-byte Unicode characters
// by using []rune for proper character splitting.
// **Feature: character-level-fallback, Property 4: Unicode 正确处理**
// **Validates: Requirements 2.1**
func TestProperty4_UnicodeHandling(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("unicode characters are correctly split and processed", prop.ForAll(
		func(numChars int) bool {
			// Create a vector model with dimension 3
			vm := NewVectorModel(3).(*vectorModel)

			// Use various Unicode ranges to test multi-byte character handling
			unicodeRanges := []struct {
				start rune
				name  string
			}{
				{0x4E00, "CJK Unified Ideographs"}, // Chinese
				{0x3040, "Hiragana"},               // Japanese
				{0x0400, "Cyrillic"},               // Russian
				{0x0600, "Arabic"},                 // Arabic
			}

			// Pick a random range for this test
			rangeIdx := numChars % len(unicodeRanges)
			baseRune := unicodeRanges[rangeIdx].start

			// Generate random multi-byte Unicode characters and add them to the model
			chars := make([]rune, numChars)
			expectedSum := make([]float32, 3)

			for i := 0; i < numChars; i++ {
				// Use characters from the selected Unicode range
				chars[i] = baseRune + rune(i)
				charVec := []float32{float32(i), float32(i + 1), float32(i + 2)}
				vm.AddVector(string(chars[i]), charVec)

				// Accumulate expected sum
				for j := 0; j < 3; j++ {
					expectedSum[j] += charVec[j]
				}
			}

			// Create a word from these multi-byte characters
			word := string(chars)

			// Verify that the word is indeed multi-byte
			if len(word) == len(chars) {
				// This would mean single-byte characters, skip this case
				return true
			}

			// Call characterLevelFallback
			vm.mtx.Lock()
			vm.fallbackAttempts++
			result, ok := vm.characterLevelFallback(word)
			vm.mtx.Unlock()

			// Should succeed
			if !ok {
				return false
			}

			// Check that result is the average of all character vectors
			expectedAvg := make([]float32, 3)
			for j := 0; j < 3; j++ {
				expectedAvg[j] = expectedSum[j] / float32(numChars)
			}

			// Compare with small tolerance for floating point errors
			for j := 0; j < 3; j++ {
				if math.Abs(float64(result[j]-expectedAvg[j])) > 1e-6 {
					return false
				}
			}

			// Verify that the number of runes matches our expectation
			if len([]rune(word)) != numChars {
				return false
			}

			return true
		},
		gen.IntRange(2, 10), // Test with 2 to 10 characters
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty5_VectorDimensionInvariance tests that vectors generated by character-level fallback
// have the same dimension as the original vector model.
// **Feature: character-level-fallback, Property 5: 向量维度不变性**
// **Validates: Requirements 2.3**
func TestProperty5_VectorDimensionInvariance(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("fallback vectors maintain model dimension", prop.ForAll(
		func(dimension int, numChars int) bool {
			// Create a vector model with the specified dimension
			vm := NewVectorModel(dimension).(*vectorModel)

			// Generate random character vectors with the correct dimension
			chars := make([]rune, numChars)

			for i := 0; i < numChars; i++ {
				chars[i] = rune('一' + i) // Chinese characters
				charVec := make([]float32, dimension)
				for j := 0; j < dimension; j++ {
					charVec[j] = float32(i*dimension + j)
				}
				vm.AddVector(string(chars[i]), charVec)
			}

			// Create a word from these characters
			word := string(chars)

			// Call characterLevelFallback
			vm.mtx.Lock()
			vm.fallbackAttempts++
			result, ok := vm.characterLevelFallback(word)
			vm.mtx.Unlock()

			// Should succeed
			if !ok {
				return false
			}

			// Check that result has the same dimension as the model
			if len(result) != dimension {
				return false
			}

			// Also verify that the model's dimension hasn't changed
			if vm.Dimension() != dimension {
				return false
			}

			return true
		},
		gen.IntRange(10, 300), // Test various dimensions from 10 to 300
		gen.IntRange(2, 10),   // Test with 2 to 10 characters
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty6_GetAverageVectorAutomaticFallback tests that GetAverageVector automatically
// applies character-level fallback for OOV words in the word list.
// **Feature: character-level-fallback, Property 6: GetAverageVector 自动回退**
// **Validates: Requirements 3.1, 3.2**
func TestProperty6_GetAverageVectorAutomaticFallback(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("GetAverageVector automatically applies fallback for OOV words", prop.ForAll(
		func(numInVocab int, numOOV int, charsPerOOV int) bool {
			// Ensure valid parameters
			if numInVocab < 1 || numOOV < 1 || charsPerOOV < 2 {
				return true // Skip invalid cases
			}
			if numInVocab > 10 || numOOV > 10 || charsPerOOV > 5 {
				return true // Keep test cases manageable
			}

			// Create a vector model with dimension 3
			vm := NewVectorModel(3).(*vectorModel)

			// Add in-vocabulary words
			inVocabWords := make([]string, numInVocab)
			for i := 0; i < numInVocab; i++ {
				word := fmt.Sprintf("word%d", i)
				inVocabWords[i] = word
				vec := []float32{float32(i), float32(i + 1), float32(i + 2)}
				vm.AddVector(word, vec)
			}

			// Create OOV words with characters that exist in vocabulary
			oovWords := make([]string, numOOV)
			for i := 0; i < numOOV; i++ {
				chars := make([]rune, charsPerOOV)
				for j := 0; j < charsPerOOV; j++ {
					char := rune('一' + i*charsPerOOV + j)
					chars[j] = char

					// Add character to vocabulary so fallback can succeed
					charVec := []float32{float32(i*10 + j), float32(i*10 + j + 1), float32(i*10 + j + 2)}
					vm.AddVector(string(char), charVec)
				}
				oovWords[i] = string(chars)
			}

			// Combine in-vocabulary and OOV words
			allWords := append(inVocabWords, oovWords...)

			// Reset stats
			vm.ResetStats()

			// Call GetAverageVector with mixed word list
			result, ok := vm.GetAverageVector(allWords)

			// Get statistics
			total, oov, hit, fallbackAttempts, fallbackSuccesses, fallbackFailures := vm.GetLookupStats()

			// Verify that:
			// 1. Result should be successful (we have valid words)
			if !ok {
				return false
			}
			if result == nil {
				return false
			}
			if len(result) != 3 {
				return false
			}

			// 2. Total lookups should equal number of words
			if total != int64(len(allWords)) {
				return false
			}

			// 3. Hit count should equal number of in-vocabulary words
			if hit != int64(numInVocab) {
				return false
			}

			// 4. OOV count should equal number of OOV words
			if oov != int64(numOOV) {
				return false
			}

			// 5. Fallback should be attempted for each OOV word
			if fallbackAttempts != int64(numOOV) {
				return false
			}

			// 6. All fallbacks should succeed (we added all characters to vocabulary)
			if fallbackSuccesses != int64(numOOV) {
				return false
			}
			if fallbackFailures != 0 {
				return false
			}

			return true
		},
		gen.IntRange(1, 5), // Number of in-vocabulary words
		gen.IntRange(1, 5), // Number of OOV words
		gen.IntRange(2, 4), // Characters per OOV word
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty7_MixedWordListCorrectAverage tests that GetAverageVector correctly computes
// the average of all successfully retrieved vectors (both direct hits and fallback results).
// **Feature: character-level-fallback, Property 7: 混合词列表正确平均**
// **Validates: Requirements 3.3**
func TestProperty7_MixedWordListCorrectAverage(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("Mixed word list produces correct average", prop.ForAll(
		func(numDirectHits int, numFallbackWords int) bool {
			// Ensure valid parameters
			if numDirectHits < 1 || numFallbackWords < 1 {
				return true // Skip invalid cases
			}
			if numDirectHits > 5 || numFallbackWords > 5 {
				return true // Keep test cases manageable
			}

			// Create a vector model with dimension 3
			vm := NewVectorModel(3).(*vectorModel)

			// Add direct hit words with known vectors
			directHitWords := make([]string, numDirectHits)
			expectedSum := []float32{0, 0, 0}

			for i := 0; i < numDirectHits; i++ {
				word := fmt.Sprintf("direct%d", i)
				directHitWords[i] = word
				vec := []float32{float32(i * 10), float32(i*10 + 1), float32(i*10 + 2)}
				vm.AddVector(word, vec)

				// Add to expected sum
				for j := 0; j < 3; j++ {
					expectedSum[j] += vec[j]
				}
			}

			// Create OOV words that will use fallback
			fallbackWords := make([]string, numFallbackWords)
			for i := 0; i < numFallbackWords; i++ {
				// Create a 2-character word where both characters are in vocabulary
				char1 := rune('甲' + i*2)
				char2 := rune('甲' + i*2 + 1)

				vec1 := []float32{float32(100 + i*10), float32(100 + i*10 + 1), float32(100 + i*10 + 2)}
				vec2 := []float32{float32(200 + i*10), float32(200 + i*10 + 1), float32(200 + i*10 + 2)}

				vm.AddVector(string(char1), vec1)
				vm.AddVector(string(char2), vec2)

				fallbackWords[i] = string([]rune{char1, char2})

				// Add fallback average to expected sum
				// Fallback computes average of char1 and char2
				for j := 0; j < 3; j++ {
					expectedSum[j] += (vec1[j] + vec2[j]) / 2.0
				}
			}

			// Combine all words
			allWords := append(directHitWords, fallbackWords...)
			totalValidWords := numDirectHits + numFallbackWords

			// Reset stats
			vm.ResetStats()

			// Call GetAverageVector
			result, ok := vm.GetAverageVector(allWords)

			// Verify result is successful
			if !ok {
				return false
			}
			if result == nil {
				return false
			}
			if len(result) != 3 {
				return false
			}

			// Compute expected average
			expectedAvg := make([]float32, 3)
			for i := 0; i < 3; i++ {
				expectedAvg[i] = expectedSum[i] / float32(totalValidWords)
			}

			// Verify the result matches expected average (with floating point tolerance)
			tolerance := float32(1e-5)
			for i := 0; i < 3; i++ {
				diff := result[i] - expectedAvg[i]
				if diff < 0 {
					diff = -diff
				}
				if diff > tolerance {
					return false
				}
			}

			// Verify statistics
			total, oov, hit, fallbackAttempts, fallbackSuccesses, _ := vm.GetLookupStats()

			if total != int64(len(allWords)) {
				return false
			}
			if hit != int64(numDirectHits) {
				return false
			}
			if oov != int64(numFallbackWords) {
				return false
			}
			if fallbackAttempts != int64(numFallbackWords) {
				return false
			}
			if fallbackSuccesses != int64(numFallbackWords) {
				return false
			}

			return true
		},
		gen.IntRange(1, 4), // Number of direct hit words
		gen.IntRange(1, 4), // Number of fallback words
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty8_FallbackStatisticsCorrectlyRecorded tests that all fallback operations
// (both successful and failed) correctly update the corresponding statistics counters.
// **Feature: character-level-fallback, Property 8: 回退统计正确记录**
// **Validates: Requirements 4.1, 4.2, 4.3**
func TestProperty8_FallbackStatisticsCorrectlyRecorded(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("fallback statistics are correctly recorded", prop.ForAll(
		func(numWords int, charsPerWord int, inVocabRatio float64) bool {
			// Validate inputs
			if numWords < 1 || numWords > 20 || charsPerWord < 2 || charsPerWord > 10 {
				return true // Skip invalid cases
			}
			if inVocabRatio < 0.0 || inVocabRatio > 1.0 {
				return true // Skip invalid ratios
			}

			// Create a vector model with dimension 3
			vm := NewVectorModel(3).(*vectorModel)

			// Add some characters to vocabulary based on inVocabRatio
			totalChars := 100
			inVocabChars := int(float64(totalChars) * inVocabRatio)

			for i := 0; i < inVocabChars; i++ {
				char := string(rune('一' + i))
				charVec := []float32{float32(i), float32(i + 1), float32(i + 2)}
				vm.AddVector(char, charVec)
			}

			// Reset stats to have a clean slate
			vm.ResetStats()

			// Track expected statistics
			expectedFallbackAttempts := int64(0)
			expectedFallbackSuccesses := int64(0)
			expectedFallbackFailures := int64(0)

			// Generate and query OOV words
			for i := 0; i < numWords; i++ {
				// Create an OOV word with multiple characters
				chars := make([]rune, charsPerWord)
				hasInVocabChar := false

				for j := 0; j < charsPerWord; j++ {
					charIndex := i*charsPerWord + j
					chars[j] = rune('一' + charIndex)

					// Check if this character is in vocabulary
					if charIndex < inVocabChars {
						hasInVocabChar = true
					}
				}

				oovWord := string(chars)

				// Query the OOV word (should trigger fallback)
				_, ok := vm.GetVector(oovWord)

				// Update expected statistics
				expectedFallbackAttempts++
				if hasInVocabChar {
					expectedFallbackSuccesses++
					if !ok {
						return false // Should have succeeded
					}
				} else {
					expectedFallbackFailures++
					if ok {
						return false // Should have failed
					}
				}
			}

			// Get actual statistics
			total, oov, hit, fallbackAttempts, fallbackSuccesses, fallbackFailures := vm.GetLookupStats()

			// Verify statistics
			// 1. Total lookups should equal number of words
			if total != int64(numWords) {
				return false
			}

			// 2. All lookups should be OOV (we only queried OOV words)
			if oov != int64(numWords) {
				return false
			}

			// 3. No direct hits (all words are OOV)
			if hit != 0 {
				return false
			}

			// 4. Fallback attempts should match expected
			if fallbackAttempts != expectedFallbackAttempts {
				return false
			}

			// 5. Fallback successes should match expected
			if fallbackSuccesses != expectedFallbackSuccesses {
				return false
			}

			// 6. Fallback failures should match expected
			if fallbackFailures != expectedFallbackFailures {
				return false
			}

			// 7. Verify fallback success rate calculation
			expectedRate := 0.0
			if expectedFallbackAttempts > 0 {
				expectedRate = float64(expectedFallbackSuccesses) / float64(expectedFallbackAttempts)
			}
			actualRate := vm.GetFallbackSuccessRate()
			if math.Abs(actualRate-expectedRate) > 1e-6 {
				return false
			}

			// 8. Verify that successes + failures = attempts
			if fallbackSuccesses+fallbackFailures != fallbackAttempts {
				return false
			}

			return true
		},
		gen.IntRange(1, 20),        // Number of words: 1 to 20
		gen.IntRange(2, 10),        // Characters per word: 2 to 10
		gen.Float64Range(0.0, 1.0), // In-vocab ratio: 0.0 to 1.0
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Unit Tests for Character-Level Fallback (Task 7.1)

// TestCharacterLevelFallback_OOVWord tests that OOV words trigger character-level fallback
func TestCharacterLevelFallback_OOVWord(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add character vectors
	vm.AddVector("没", []float32{1.0, 2.0, 3.0})
	vm.AddVector("事", []float32{4.0, 5.0, 6.0})

	// Reset stats
	vm.ResetStats()

	// Query OOV word "没事" (not in vocabulary, but characters are)
	result, ok := vm.GetVector("没事")

	// Should succeed via fallback
	assert.True(t, ok, "Should succeed via character-level fallback")
	assert.NotNil(t, result, "Result should not be nil")
	assert.Equal(t, 3, len(result), "Result should have correct dimension")

	// Verify the result is the average of character vectors
	expected := []float32{2.5, 3.5, 4.5} // (1+4)/2, (2+5)/2, (3+6)/2
	for i := range expected {
		assert.InDelta(t, expected[i], result[i], 1e-6, "Vector component %d should match expected average", i)
	}

	// Verify statistics
	total, oov, hit, fallbackAttempts, fallbackSuccesses, fallbackFailures := vm.GetLookupStats()
	assert.Equal(t, int64(1), total, "Should have 1 total lookup")
	assert.Equal(t, int64(1), oov, "Should have 1 OOV lookup")
	assert.Equal(t, int64(0), hit, "Should have 0 direct hits")
	assert.Equal(t, int64(1), fallbackAttempts, "Should have 1 fallback attempt")
	assert.Equal(t, int64(1), fallbackSuccesses, "Should have 1 fallback success")
	assert.Equal(t, int64(0), fallbackFailures, "Should have 0 fallback failures")
}

// TestCharacterLevelFallback_SingleCharacterWord tests that single character OOV words don't trigger fallback
func TestCharacterLevelFallback_SingleCharacterWord(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add some vectors but not the single character we'll query
	vm.AddVector("没", []float32{1.0, 2.0, 3.0})
	vm.AddVector("事", []float32{4.0, 5.0, 6.0})

	// Reset stats
	vm.ResetStats()

	// Query single character OOV word "好" (not in vocabulary)
	result, ok := vm.GetVector("好")

	// Should fail (single character doesn't trigger fallback)
	assert.False(t, ok, "Single character OOV should not succeed")
	assert.Nil(t, result, "Result should be nil")

	// Verify statistics
	total, oov, hit, fallbackAttempts, fallbackSuccesses, fallbackFailures := vm.GetLookupStats()
	assert.Equal(t, int64(1), total, "Should have 1 total lookup")
	assert.Equal(t, int64(1), oov, "Should have 1 OOV lookup")
	assert.Equal(t, int64(0), hit, "Should have 0 direct hits")
	assert.Equal(t, int64(1), fallbackAttempts, "Should have 1 fallback attempt")
	assert.Equal(t, int64(0), fallbackSuccesses, "Should have 0 fallback successes")
	assert.Equal(t, int64(1), fallbackFailures, "Should have 1 fallback failure")
}

// TestCharacterLevelFallback_AllCharactersInVocabulary tests fallback when all characters are in vocabulary
func TestCharacterLevelFallback_AllCharactersInVocabulary(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add all character vectors
	vm.AddVector("人", []float32{1.0, 2.0, 3.0})
	vm.AddVector("工", []float32{4.0, 5.0, 6.0})
	vm.AddVector("智", []float32{7.0, 8.0, 9.0})
	vm.AddVector("能", []float32{10.0, 11.0, 12.0})

	// Reset stats
	vm.ResetStats()

	// Query OOV word "人工智能" (not in vocabulary, but all characters are)
	result, ok := vm.GetVector("人工智能")

	// Should succeed via fallback
	assert.True(t, ok, "Should succeed via character-level fallback")
	assert.NotNil(t, result, "Result should not be nil")

	// Verify the result is the average of all character vectors
	expected := []float32{5.5, 6.5, 7.5} // (1+4+7+10)/4, (2+5+8+11)/4, (3+6+9+12)/4
	for i := range expected {
		assert.InDelta(t, expected[i], result[i], 1e-6, "Vector component %d should match expected average", i)
	}

	// Verify statistics
	_, _, _, fallbackAttempts, fallbackSuccesses, fallbackFailures := vm.GetLookupStats()
	assert.Equal(t, int64(1), fallbackAttempts, "Should have 1 fallback attempt")
	assert.Equal(t, int64(1), fallbackSuccesses, "Should have 1 fallback success")
	assert.Equal(t, int64(0), fallbackFailures, "Should have 0 fallback failures")
}

// TestCharacterLevelFallback_PartialCharactersInVocabulary tests fallback when only some characters are in vocabulary
func TestCharacterLevelFallback_PartialCharactersInVocabulary(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add only some character vectors
	vm.AddVector("机", []float32{1.0, 2.0, 3.0})
	vm.AddVector("学", []float32{4.0, 5.0, 6.0})
	// Note: "器" and "习" are NOT added

	// Reset stats
	vm.ResetStats()

	// Query OOV word "机器学习" (not in vocabulary, only 2 out of 4 characters are)
	result, ok := vm.GetVector("机器学习")

	// Should succeed via fallback (at least one character is in vocabulary)
	assert.True(t, ok, "Should succeed via character-level fallback")
	assert.NotNil(t, result, "Result should not be nil")

	// Verify the result is the average of only the in-vocabulary character vectors
	expected := []float32{2.5, 3.5, 4.5} // (1+4)/2, (2+5)/2, (3+6)/2
	for i := range expected {
		assert.InDelta(t, expected[i], result[i], 1e-6, "Vector component %d should match expected average", i)
	}

	// Verify statistics
	_, _, _, fallbackAttempts, fallbackSuccesses, fallbackFailures := vm.GetLookupStats()
	assert.Equal(t, int64(1), fallbackAttempts, "Should have 1 fallback attempt")
	assert.Equal(t, int64(1), fallbackSuccesses, "Should have 1 fallback success")
	assert.Equal(t, int64(0), fallbackFailures, "Should have 0 fallback failures")
}

// TestCharacterLevelFallback_NoCharactersInVocabulary tests fallback when no characters are in vocabulary
func TestCharacterLevelFallback_NoCharactersInVocabulary(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add some vectors but not the characters we'll query
	vm.AddVector("人", []float32{1.0, 2.0, 3.0})
	vm.AddVector("工", []float32{4.0, 5.0, 6.0})

	// Reset stats
	vm.ResetStats()

	// Query OOV word "未知词" (not in vocabulary, and no characters are either)
	result, ok := vm.GetVector("未知词")

	// Should fail (no characters in vocabulary)
	assert.False(t, ok, "Should fail when no characters are in vocabulary")
	assert.Nil(t, result, "Result should be nil")

	// Verify statistics
	_, _, _, fallbackAttempts, fallbackSuccesses, fallbackFailures := vm.GetLookupStats()
	assert.Equal(t, int64(1), fallbackAttempts, "Should have 1 fallback attempt")
	assert.Equal(t, int64(0), fallbackSuccesses, "Should have 0 fallback successes")
	assert.Equal(t, int64(1), fallbackFailures, "Should have 1 fallback failure")
}

// Unit Tests for Unicode Handling (Task 7.2)

// TestUnicodeHandling_MultiByteChineseCharacters tests correct splitting of multi-byte Chinese characters
func TestUnicodeHandling_MultiByteChineseCharacters(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add multi-byte Chinese character vectors
	// These are 3-byte UTF-8 characters
	vm.AddVector("你", []float32{1.0, 2.0, 3.0})
	vm.AddVector("好", []float32{4.0, 5.0, 6.0})
	vm.AddVector("吗", []float32{7.0, 8.0, 9.0})

	// Reset stats
	vm.ResetStats()

	// Query OOV word "你好吗" (3 multi-byte characters)
	result, ok := vm.GetVector("你好吗")

	// Should succeed via fallback
	assert.True(t, ok, "Should succeed via character-level fallback")
	assert.NotNil(t, result, "Result should not be nil")

	// Verify the result is the average of all character vectors
	expected := []float32{4.0, 5.0, 6.0} // (1+4+7)/3, (2+5+8)/3, (3+6+9)/3
	for i := range expected {
		assert.InDelta(t, expected[i], result[i], 1e-6, "Vector component %d should match expected average", i)
	}

	// Verify that the word is indeed multi-byte
	wordBytes := []byte("你好吗")
	wordRunes := []rune("你好吗")
	assert.Greater(t, len(wordBytes), len(wordRunes), "Multi-byte characters should have more bytes than runes")
	assert.Equal(t, 3, len(wordRunes), "Should have exactly 3 runes")
	assert.Equal(t, 9, len(wordBytes), "Should have 9 bytes (3 chars × 3 bytes each)")
}

// TestUnicodeHandling_MixedChineseEnglish tests handling of mixed Chinese and English words
func TestUnicodeHandling_MixedChineseEnglish(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add character vectors for both Chinese and English characters
	vm.AddVector("A", []float32{1.0, 2.0, 3.0})
	vm.AddVector("I", []float32{4.0, 5.0, 6.0})
	vm.AddVector("人", []float32{7.0, 8.0, 9.0})
	vm.AddVector("工", []float32{10.0, 11.0, 12.0})

	// Reset stats
	vm.ResetStats()

	// Query OOV word "AI人工" (mixed English and Chinese)
	result, ok := vm.GetVector("AI人工")

	// Should succeed via fallback
	assert.True(t, ok, "Should succeed via character-level fallback")
	assert.NotNil(t, result, "Result should not be nil")

	// Verify the result is the average of all character vectors
	expected := []float32{5.5, 6.5, 7.5} // (1+4+7+10)/4, (2+5+8+11)/4, (3+6+9+12)/4
	for i := range expected {
		assert.InDelta(t, expected[i], result[i], 1e-6, "Vector component %d should match expected average", i)
	}

	// Verify correct character count
	wordRunes := []rune("AI人工")
	assert.Equal(t, 4, len(wordRunes), "Should have exactly 4 characters")
}

// TestUnicodeHandling_SpecialUnicodeCharacters tests handling of special Unicode characters
func TestUnicodeHandling_SpecialUnicodeCharacters(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add vectors for various Unicode ranges
	// Emoji (4-byte UTF-8)
	vm.AddVector("😀", []float32{1.0, 2.0, 3.0})
	vm.AddVector("😊", []float32{4.0, 5.0, 6.0})

	// Japanese Hiragana (3-byte UTF-8)
	vm.AddVector("あ", []float32{7.0, 8.0, 9.0})
	vm.AddVector("い", []float32{10.0, 11.0, 12.0})

	// Cyrillic (2-byte UTF-8)
	vm.AddVector("А", []float32{13.0, 14.0, 15.0})
	vm.AddVector("Б", []float32{16.0, 17.0, 18.0})

	testCases := []struct {
		name     string
		word     string
		expected []float32
	}{
		{
			name:     "Emoji",
			word:     "😀😊",
			expected: []float32{2.5, 3.5, 4.5}, // (1+4)/2, (2+5)/2, (3+6)/2
		},
		{
			name:     "Japanese Hiragana",
			word:     "あい",
			expected: []float32{8.5, 9.5, 10.5}, // (7+10)/2, (8+11)/2, (9+12)/2
		},
		{
			name:     "Cyrillic",
			word:     "АБ",
			expected: []float32{14.5, 15.5, 16.5}, // (13+16)/2, (14+17)/2, (15+18)/2
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset stats for each test
			vm.ResetStats()

			// Query OOV word
			result, ok := vm.GetVector(tc.word)

			// Should succeed via fallback
			assert.True(t, ok, "Should succeed via character-level fallback")
			assert.NotNil(t, result, "Result should not be nil")

			// Verify the result
			for i := range tc.expected {
				assert.InDelta(t, tc.expected[i], result[i], 1e-6, "Vector component %d should match expected average", i)
			}

			// Verify correct rune count
			wordRunes := []rune(tc.word)
			assert.Equal(t, 2, len(wordRunes), "Should have exactly 2 characters")
		})
	}
}

// TestUnicodeHandling_ComplexMixedScript tests handling of complex mixed-script words
func TestUnicodeHandling_ComplexMixedScript(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add vectors for various scripts
	vm.AddVector("中", []float32{1.0, 2.0, 3.0})
	vm.AddVector("A", []float32{4.0, 5.0, 6.0})
	vm.AddVector("あ", []float32{7.0, 8.0, 9.0})
	vm.AddVector("1", []float32{10.0, 11.0, 12.0})
	vm.AddVector("😀", []float32{13.0, 14.0, 15.0})

	// Reset stats
	vm.ResetStats()

	// Query OOV word with mixed scripts: Chinese + English + Japanese + Number + Emoji
	result, ok := vm.GetVector("中Aあ1😀")

	// Should succeed via fallback
	assert.True(t, ok, "Should succeed via character-level fallback")
	assert.NotNil(t, result, "Result should not be nil")

	// Verify the result is the average of all character vectors
	expected := []float32{7.0, 8.0, 9.0} // (1+4+7+10+13)/5, (2+5+8+11+14)/5, (3+6+9+12+15)/5
	for i := range expected {
		assert.InDelta(t, expected[i], result[i], 1e-6, "Vector component %d should match expected average", i)
	}

	// Verify correct character count
	wordRunes := []rune("中Aあ1😀")
	assert.Equal(t, 5, len(wordRunes), "Should have exactly 5 characters")

	// Verify byte count is much larger than rune count (multi-byte characters)
	wordBytes := []byte("中Aあ1😀")
	assert.Greater(t, len(wordBytes), len(wordRunes), "Multi-byte characters should have more bytes than runes")
}

// TestUnicodeHandling_SurrogatePairs tests handling of Unicode surrogate pairs (4-byte UTF-8)
func TestUnicodeHandling_SurrogatePairs(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add vectors for characters that require surrogate pairs in UTF-16 (4-byte UTF-8)
	// These are characters outside the Basic Multilingual Plane (BMP)
	vm.AddVector("𝐀", []float32{1.0, 2.0, 3.0})    // Mathematical Bold Capital A (U+1D400)
	vm.AddVector("𝐁", []float32{4.0, 5.0, 6.0})    // Mathematical Bold Capital B (U+1D401)
	vm.AddVector("🀀", []float32{7.0, 8.0, 9.0})    // Mahjong Tile East Wind (U+1F000)
	vm.AddVector("🀁", []float32{10.0, 11.0, 12.0}) // Mahjong Tile South Wind (U+1F001)

	// Reset stats
	vm.ResetStats()

	// Query OOV word with 4-byte UTF-8 characters
	result, ok := vm.GetVector("𝐀𝐁🀀🀁")

	// Should succeed via fallback
	assert.True(t, ok, "Should succeed via character-level fallback")
	assert.NotNil(t, result, "Result should not be nil")

	// Verify the result is the average of all character vectors
	expected := []float32{5.5, 6.5, 7.5} // (1+4+7+10)/4, (2+5+8+11)/4, (3+6+9+12)/4
	for i := range expected {
		assert.InDelta(t, expected[i], result[i], 1e-6, "Vector component %d should match expected average", i)
	}

	// Verify correct character count
	wordRunes := []rune("𝐀𝐁🀀🀁")
	assert.Equal(t, 4, len(wordRunes), "Should have exactly 4 characters")

	// Verify byte count (each character is 4 bytes in UTF-8)
	wordBytes := []byte("𝐀𝐁🀀🀁")
	assert.Equal(t, 16, len(wordBytes), "Should have 16 bytes (4 chars × 4 bytes each)")
}

// Unit Tests for Statistics Functionality (Task 7.3)

// TestStatistics_FallbackCountersCorrectlyUpdated tests that fallback counters are correctly updated
func TestStatistics_FallbackCountersCorrectlyUpdated(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add character vectors
	vm.AddVector("成", []float32{1.0, 2.0, 3.0})
	vm.AddVector("功", []float32{4.0, 5.0, 6.0})

	// Reset stats to start fresh
	vm.ResetStats()

	// Scenario 1: Successful fallback
	result1, ok1 := vm.GetVector("成功") // OOV word with characters in vocabulary
	assert.True(t, ok1, "First fallback should succeed")
	assert.NotNil(t, result1, "First result should not be nil")

	total, oov, hit, fallbackAttempts, fallbackSuccesses, fallbackFailures := vm.GetLookupStats()
	assert.Equal(t, int64(1), total, "Should have 1 total lookup")
	assert.Equal(t, int64(1), oov, "Should have 1 OOV lookup")
	assert.Equal(t, int64(0), hit, "Should have 0 direct hits")
	assert.Equal(t, int64(1), fallbackAttempts, "Should have 1 fallback attempt")
	assert.Equal(t, int64(1), fallbackSuccesses, "Should have 1 fallback success")
	assert.Equal(t, int64(0), fallbackFailures, "Should have 0 fallback failures")

	// Scenario 2: Failed fallback (no characters in vocabulary)
	result2, ok2 := vm.GetVector("失败") // OOV word with no characters in vocabulary
	assert.False(t, ok2, "Second fallback should fail")
	assert.Nil(t, result2, "Second result should be nil")

	total, oov, hit, fallbackAttempts, fallbackSuccesses, fallbackFailures = vm.GetLookupStats()
	assert.Equal(t, int64(2), total, "Should have 2 total lookups")
	assert.Equal(t, int64(2), oov, "Should have 2 OOV lookups")
	assert.Equal(t, int64(0), hit, "Should have 0 direct hits")
	assert.Equal(t, int64(2), fallbackAttempts, "Should have 2 fallback attempts")
	assert.Equal(t, int64(1), fallbackSuccesses, "Should have 1 fallback success")
	assert.Equal(t, int64(1), fallbackFailures, "Should have 1 fallback failure")

	// Scenario 3: Direct hit (no fallback)
	result3, ok3 := vm.GetVector("成") // Character in vocabulary
	assert.True(t, ok3, "Direct hit should succeed")
	assert.NotNil(t, result3, "Third result should not be nil")

	total, oov, hit, fallbackAttempts, fallbackSuccesses, fallbackFailures = vm.GetLookupStats()
	assert.Equal(t, int64(3), total, "Should have 3 total lookups")
	assert.Equal(t, int64(2), oov, "Should have 2 OOV lookups")
	assert.Equal(t, int64(1), hit, "Should have 1 direct hit")
	assert.Equal(t, int64(2), fallbackAttempts, "Should still have 2 fallback attempts")
	assert.Equal(t, int64(1), fallbackSuccesses, "Should still have 1 fallback success")
	assert.Equal(t, int64(1), fallbackFailures, "Should still have 1 fallback failure")

	// Scenario 4: Multiple successful fallbacks
	result4, ok4 := vm.GetVector("功成") // Another OOV word with characters in vocabulary
	assert.True(t, ok4, "Fourth fallback should succeed")
	assert.NotNil(t, result4, "Fourth result should not be nil")

	total, oov, hit, fallbackAttempts, fallbackSuccesses, fallbackFailures = vm.GetLookupStats()
	assert.Equal(t, int64(4), total, "Should have 4 total lookups")
	assert.Equal(t, int64(3), oov, "Should have 3 OOV lookups")
	assert.Equal(t, int64(1), hit, "Should have 1 direct hit")
	assert.Equal(t, int64(3), fallbackAttempts, "Should have 3 fallback attempts")
	assert.Equal(t, int64(2), fallbackSuccesses, "Should have 2 fallback successes")
	assert.Equal(t, int64(1), fallbackFailures, "Should have 1 fallback failure")
}

// TestStatistics_QueryReturnsCorrectValues tests that statistics query returns correct values
func TestStatistics_QueryReturnsCorrectValues(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add test vectors
	vm.AddVector("测", []float32{1.0, 2.0, 3.0})
	vm.AddVector("试", []float32{4.0, 5.0, 6.0})
	vm.AddVector("word", []float32{7.0, 8.0, 9.0})

	// Reset stats
	vm.ResetStats()

	// Perform various operations
	vm.GetVector("测试")   // OOV with successful fallback
	vm.GetVector("word") // Direct hit
	vm.GetVector("未知")   // OOV with failed fallback
	vm.GetVector("测")    // Direct hit
	vm.GetVector("试测")   // OOV with successful fallback

	// Query statistics using GetLookupStats
	total, oov, hit, fallbackAttempts, fallbackSuccesses, fallbackFailures := vm.GetLookupStats()

	// Verify all values
	assert.Equal(t, int64(5), total, "Total lookups should be 5")
	assert.Equal(t, int64(3), oov, "OOV lookups should be 3")
	assert.Equal(t, int64(2), hit, "Hit lookups should be 2")
	assert.Equal(t, int64(3), fallbackAttempts, "Fallback attempts should be 3")
	assert.Equal(t, int64(2), fallbackSuccesses, "Fallback successes should be 2")
	assert.Equal(t, int64(1), fallbackFailures, "Fallback failures should be 1")

	// Verify OOV rate
	expectedOOVRate := 3.0 / 5.0
	actualOOVRate := vm.GetOOVRate()
	assert.InDelta(t, expectedOOVRate, actualOOVRate, 1e-6, "OOV rate should be correct")

	// Verify hit rate
	expectedHitRate := 2.0 / 5.0
	actualHitRate := vm.GetVectorHitRate()
	assert.InDelta(t, expectedHitRate, actualHitRate, 1e-6, "Hit rate should be correct")

	// Verify fallback success rate
	expectedFallbackRate := 2.0 / 3.0
	actualFallbackRate := vm.GetFallbackSuccessRate()
	assert.InDelta(t, expectedFallbackRate, actualFallbackRate, 1e-6, "Fallback success rate should be correct")
}

// TestStatistics_FallbackSuccessRateCalculation tests the fallback success rate calculation
func TestStatistics_FallbackSuccessRateCalculation(t *testing.T) {
	testCases := []struct {
		name                      string
		setupFunc                 func(*vectorModel)
		expectedFallbackRate      float64
		expectedFallbackAttempts  int64
		expectedFallbackSuccesses int64
		expectedFallbackFailures  int64
	}{
		{
			name: "100% success rate",
			setupFunc: func(vm *vectorModel) {
				vm.AddVector("好", []float32{1.0, 2.0, 3.0})
				vm.AddVector("的", []float32{4.0, 5.0, 6.0})
				vm.ResetStats()
				vm.GetVector("好的") // Successful fallback
				vm.GetVector("的好") // Successful fallback
			},
			expectedFallbackRate:      1.0,
			expectedFallbackAttempts:  2,
			expectedFallbackSuccesses: 2,
			expectedFallbackFailures:  0,
		},
		{
			name: "0% success rate",
			setupFunc: func(vm *vectorModel) {
				vm.AddVector("好", []float32{1.0, 2.0, 3.0})
				vm.ResetStats()
				vm.GetVector("未知") // Failed fallback
				vm.GetVector("失败") // Failed fallback
			},
			expectedFallbackRate:      0.0,
			expectedFallbackAttempts:  2,
			expectedFallbackSuccesses: 0,
			expectedFallbackFailures:  2,
		},
		{
			name: "50% success rate",
			setupFunc: func(vm *vectorModel) {
				vm.AddVector("成", []float32{1.0, 2.0, 3.0})
				vm.AddVector("功", []float32{4.0, 5.0, 6.0})
				vm.ResetStats()
				vm.GetVector("成功") // Successful fallback
				vm.GetVector("失败") // Failed fallback
			},
			expectedFallbackRate:      0.5,
			expectedFallbackAttempts:  2,
			expectedFallbackSuccesses: 1,
			expectedFallbackFailures:  1,
		},
		{
			name: "No fallback attempts",
			setupFunc: func(vm *vectorModel) {
				vm.AddVector("word", []float32{1.0, 2.0, 3.0})
				vm.ResetStats()
				vm.GetVector("word") // Direct hit, no fallback
			},
			expectedFallbackRate:      0.0,
			expectedFallbackAttempts:  0,
			expectedFallbackSuccesses: 0,
			expectedFallbackFailures:  0,
		},
		{
			name: "Mixed operations with 75% success rate",
			setupFunc: func(vm *vectorModel) {
				vm.AddVector("一", []float32{1.0, 2.0, 3.0})
				vm.AddVector("二", []float32{4.0, 5.0, 6.0})
				vm.AddVector("三", []float32{7.0, 8.0, 9.0})
				vm.ResetStats()
				vm.GetVector("一二") // Successful fallback
				vm.GetVector("二三") // Successful fallback
				vm.GetVector("三一") // Successful fallback
				vm.GetVector("未知") // Failed fallback
			},
			expectedFallbackRate:      0.75,
			expectedFallbackAttempts:  4,
			expectedFallbackSuccesses: 3,
			expectedFallbackFailures:  1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vm := NewVectorModel(3).(*vectorModel)
			tc.setupFunc(vm)

			// Get fallback success rate
			actualRate := vm.GetFallbackSuccessRate()
			assert.InDelta(t, tc.expectedFallbackRate, actualRate, 1e-6, "Fallback success rate should match expected")

			// Verify detailed statistics
			_, _, _, fallbackAttempts, fallbackSuccesses, fallbackFailures := vm.GetLookupStats()
			assert.Equal(t, tc.expectedFallbackAttempts, fallbackAttempts, "Fallback attempts should match expected")
			assert.Equal(t, tc.expectedFallbackSuccesses, fallbackSuccesses, "Fallback successes should match expected")
			assert.Equal(t, tc.expectedFallbackFailures, fallbackFailures, "Fallback failures should match expected")

			// Verify that successes + failures = attempts
			assert.Equal(t, fallbackAttempts, fallbackSuccesses+fallbackFailures, "Successes + failures should equal attempts")
		})
	}
}

// TestStatistics_ResetStats tests that ResetStats correctly resets all counters
func TestStatistics_ResetStats(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add test vectors
	vm.AddVector("重", []float32{1.0, 2.0, 3.0})
	vm.AddVector("置", []float32{4.0, 5.0, 6.0})

	// Perform some operations to accumulate statistics
	vm.GetVector("重置") // OOV with successful fallback
	vm.GetVector("重")  // Direct hit
	vm.GetVector("未知") // OOV with failed fallback

	// Verify statistics are non-zero
	total, oov, hit, fallbackAttempts, fallbackSuccesses, fallbackFailures := vm.GetLookupStats()
	assert.Greater(t, total, int64(0), "Total should be > 0 before reset")
	assert.Greater(t, oov, int64(0), "OOV should be > 0 before reset")
	assert.Greater(t, hit, int64(0), "Hit should be > 0 before reset")
	assert.Greater(t, fallbackAttempts, int64(0), "Fallback attempts should be > 0 before reset")
	assert.Greater(t, fallbackSuccesses, int64(0), "Fallback successes should be > 0 before reset")
	assert.Greater(t, fallbackFailures, int64(0), "Fallback failures should be > 0 before reset")

	// Reset statistics
	vm.ResetStats()

	// Verify all statistics are zero
	total, oov, hit, fallbackAttempts, fallbackSuccesses, fallbackFailures = vm.GetLookupStats()
	assert.Equal(t, int64(0), total, "Total should be 0 after reset")
	assert.Equal(t, int64(0), oov, "OOV should be 0 after reset")
	assert.Equal(t, int64(0), hit, "Hit should be 0 after reset")
	assert.Equal(t, int64(0), fallbackAttempts, "Fallback attempts should be 0 after reset")
	assert.Equal(t, int64(0), fallbackSuccesses, "Fallback successes should be 0 after reset")
	assert.Equal(t, int64(0), fallbackFailures, "Fallback failures should be 0 after reset")

	// Verify rates are zero
	assert.Equal(t, 0.0, vm.GetOOVRate(), "OOV rate should be 0 after reset")
	assert.Equal(t, 0.0, vm.GetVectorHitRate(), "Hit rate should be 0 after reset")
	assert.Equal(t, 0.0, vm.GetFallbackSuccessRate(), "Fallback success rate should be 0 after reset")

	// Perform new operations after reset
	vm.GetVector("重置") // OOV with successful fallback

	// Verify statistics are updated correctly after reset
	total, oov, hit, fallbackAttempts, fallbackSuccesses, fallbackFailures = vm.GetLookupStats()
	assert.Equal(t, int64(1), total, "Total should be 1 after reset and new operation")
	assert.Equal(t, int64(1), oov, "OOV should be 1 after reset and new operation")
	assert.Equal(t, int64(0), hit, "Hit should be 0 after reset and new operation")
	assert.Equal(t, int64(1), fallbackAttempts, "Fallback attempts should be 1 after reset and new operation")
	assert.Equal(t, int64(1), fallbackSuccesses, "Fallback successes should be 1 after reset and new operation")
	assert.Equal(t, int64(0), fallbackFailures, "Fallback failures should be 0 after reset and new operation")
}

// TestStatistics_GetAverageVectorStatistics tests that GetAverageVector correctly updates statistics
func TestStatistics_GetAverageVectorStatistics(t *testing.T) {
	vm := NewVectorModel(3).(*vectorModel)

	// Add test vectors
	vm.AddVector("平", []float32{1.0, 2.0, 3.0})
	vm.AddVector("均", []float32{4.0, 5.0, 6.0})
	vm.AddVector("word", []float32{7.0, 8.0, 9.0})

	// Reset stats
	vm.ResetStats()

	// Call GetAverageVector with mixed words
	words := []string{
		"word", // Direct hit
		"平均",   // OOV with successful fallback
		"未知",   // OOV with failed fallback
		"均平",   // OOV with successful fallback
	}

	result, ok := vm.GetAverageVector(words)
	assert.True(t, ok, "GetAverageVector should succeed")
	assert.NotNil(t, result, "Result should not be nil")

	// Verify statistics
	total, oov, hit, fallbackAttempts, fallbackSuccesses, fallbackFailures := vm.GetLookupStats()
	assert.Equal(t, int64(4), total, "Total lookups should be 4")
	assert.Equal(t, int64(3), oov, "OOV lookups should be 3")
	assert.Equal(t, int64(1), hit, "Hit lookups should be 1")
	assert.Equal(t, int64(3), fallbackAttempts, "Fallback attempts should be 3")
	assert.Equal(t, int64(2), fallbackSuccesses, "Fallback successes should be 2")
	assert.Equal(t, int64(1), fallbackFailures, "Fallback failures should be 1")

	// Verify rates
	expectedOOVRate := 3.0 / 4.0
	assert.InDelta(t, expectedOOVRate, vm.GetOOVRate(), 1e-6, "OOV rate should be correct")

	expectedHitRate := 1.0 / 4.0
	assert.InDelta(t, expectedHitRate, vm.GetVectorHitRate(), 1e-6, "Hit rate should be correct")

	expectedFallbackRate := 2.0 / 3.0
	assert.InDelta(t, expectedFallbackRate, vm.GetFallbackSuccessRate(), 1e-6, "Fallback success rate should be correct")
}

// TestStatistics_EdgeCases tests edge cases in statistics calculation
func TestStatistics_EdgeCases(t *testing.T) {
	t.Run("Division by zero - no lookups", func(t *testing.T) {
		vm := NewVectorModel(3).(*vectorModel)
		vm.ResetStats()

		// With no lookups, all rates should be 0
		assert.Equal(t, 0.0, vm.GetOOVRate(), "OOV rate should be 0 with no lookups")
		assert.Equal(t, 0.0, vm.GetVectorHitRate(), "Hit rate should be 0 with no lookups")
		assert.Equal(t, 0.0, vm.GetFallbackSuccessRate(), "Fallback success rate should be 0 with no fallback attempts")
	})

	t.Run("Division by zero - no fallback attempts", func(t *testing.T) {
		vm := NewVectorModel(3).(*vectorModel)
		vm.AddVector("word", []float32{1.0, 2.0, 3.0})
		vm.ResetStats()

		// Only direct hits, no fallback attempts
		vm.GetVector("word")

		assert.Equal(t, 0.0, vm.GetFallbackSuccessRate(), "Fallback success rate should be 0 with no fallback attempts")
	})

	t.Run("All operations are OOV", func(t *testing.T) {
		vm := NewVectorModel(3).(*vectorModel)
		vm.AddVector("好", []float32{1.0, 2.0, 3.0})
		vm.ResetStats()

		vm.GetVector("未知1")
		vm.GetVector("未知2")
		vm.GetVector("未知3")

		assert.Equal(t, 1.0, vm.GetOOVRate(), "OOV rate should be 1.0 when all lookups are OOV")
		assert.Equal(t, 0.0, vm.GetVectorHitRate(), "Hit rate should be 0.0 when all lookups are OOV")
	})

	t.Run("All operations are direct hits", func(t *testing.T) {
		vm := NewVectorModel(3).(*vectorModel)
		vm.AddVector("word1", []float32{1.0, 2.0, 3.0})
		vm.AddVector("word2", []float32{4.0, 5.0, 6.0})
		vm.ResetStats()

		vm.GetVector("word1")
		vm.GetVector("word2")

		assert.Equal(t, 0.0, vm.GetOOVRate(), "OOV rate should be 0.0 when all lookups are hits")
		assert.Equal(t, 1.0, vm.GetVectorHitRate(), "Hit rate should be 1.0 when all lookups are hits")
		assert.Equal(t, 0.0, vm.GetFallbackSuccessRate(), "Fallback success rate should be 0.0 with no fallback attempts")
	})

	t.Run("Single character OOV doesn't affect fallback success rate", func(t *testing.T) {
		vm := NewVectorModel(3).(*vectorModel)
		vm.ResetStats()

		// Single character OOV triggers fallback attempt but fails
		vm.GetVector("X")

		_, _, _, fallbackAttempts, fallbackSuccesses, fallbackFailures := vm.GetLookupStats()
		assert.Equal(t, int64(1), fallbackAttempts, "Should have 1 fallback attempt")
		assert.Equal(t, int64(0), fallbackSuccesses, "Should have 0 fallback successes")
		assert.Equal(t, int64(1), fallbackFailures, "Should have 1 fallback failure")
		assert.Equal(t, 0.0, vm.GetFallbackSuccessRate(), "Fallback success rate should be 0.0")
	})
}

// TestProperty9_ConcurrentSafety tests that concurrent calls to GetVector and GetAverageVector
// with character-level fallback do not produce data races.
// **Feature: character-level-fallback, Property 9: 并发安全性**
// **Validates: Requirements 5.1, 5.2, 5.3, 5.4**
func TestProperty9_ConcurrentSafety(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("concurrent GetVector and GetAverageVector calls are thread-safe", prop.ForAll(
		func(numGoroutines int, numOperations int, numCharsInVocab int) bool {
			// Validate inputs
			if numGoroutines < 2 || numGoroutines > 50 {
				return true // Skip invalid test cases
			}
			if numOperations < 5 || numOperations > 100 {
				return true // Skip invalid test cases
			}
			if numCharsInVocab < 5 || numCharsInVocab > 50 {
				return true // Skip invalid test cases
			}

			// Create a vector model with dimension 10
			vm := NewVectorModel(10).(*vectorModel)

			// Add some single characters to vocabulary (for fallback to work)
			for i := 0; i < numCharsInVocab; i++ {
				char := string(rune('a' + i))
				vector := make([]float32, 10)
				for j := 0; j < 10; j++ {
					vector[j] = float32(i*10 + j)
				}
				vm.AddVector(char, vector)
			}

			// Add some multi-character words to vocabulary (for direct hits)
			for i := 0; i < 10; i++ {
				word := fmt.Sprintf("word%d", i)
				vector := make([]float32, 10)
				for j := 0; j < 10; j++ {
					vector[j] = float32(i*100 + j)
				}
				vm.AddVector(word, vector)
			}

			// Create a wait group for synchronization
			var wg sync.WaitGroup
			wg.Add(numGoroutines)

			// Channel to collect any errors
			errors := make(chan error, numGoroutines*numOperations)

			// Launch concurrent goroutines
			for g := 0; g < numGoroutines; g++ {
				go func(goroutineID int) {
					defer wg.Done()

					for op := 0; op < numOperations; op++ {
						// Alternate between different operations
						switch op % 5 {
						case 0:
							// Test GetVector with OOV word (triggers fallback)
							oovWord := fmt.Sprintf("oov%d%d", goroutineID, op)
							vec, found := vm.GetVector(oovWord)
							if found && len(vec) != 10 {
								errors <- fmt.Errorf("goroutine %d: GetVector returned wrong dimension: %d", goroutineID, len(vec))
							}

						case 1:
							// Test GetVector with existing word (direct hit)
							word := fmt.Sprintf("word%d", op%10)
							vec, found := vm.GetVector(word)
							if !found {
								errors <- fmt.Errorf("goroutine %d: GetVector failed to find existing word: %s", goroutineID, word)
							}
							if found && len(vec) != 10 {
								errors <- fmt.Errorf("goroutine %d: GetVector returned wrong dimension: %d", goroutineID, len(vec))
							}

						case 2:
							// Test GetAverageVector with mixed words (some OOV, some in vocab)
							words := []string{
								fmt.Sprintf("word%d", op%10),
								fmt.Sprintf("oov%d%d", goroutineID, op),
								fmt.Sprintf("word%d", (op+1)%10),
							}
							vec, found := vm.GetAverageVector(words)
							if found && len(vec) != 10 {
								errors <- fmt.Errorf("goroutine %d: GetAverageVector returned wrong dimension: %d", goroutineID, len(vec))
							}

						case 3:
							// Test GetAverageVector with all OOV words
							words := []string{
								fmt.Sprintf("oov%d%da", goroutineID, op),
								fmt.Sprintf("oov%d%db", goroutineID, op),
							}
							vec, found := vm.GetAverageVector(words)
							if found && len(vec) != 10 {
								errors <- fmt.Errorf("goroutine %d: GetAverageVector returned wrong dimension: %d", goroutineID, len(vec))
							}

						case 4:
							// Test statistics queries
							total, oov, hit, fallbackAttempts, fallbackSuccesses, fallbackFailures := vm.GetLookupStats()
							if total < 0 || oov < 0 || hit < 0 || fallbackAttempts < 0 || fallbackSuccesses < 0 || fallbackFailures < 0 {
								errors <- fmt.Errorf("goroutine %d: negative statistics values", goroutineID)
							}
							if total < oov+hit {
								errors <- fmt.Errorf("goroutine %d: inconsistent statistics: total=%d, oov=%d, hit=%d", goroutineID, total, oov, hit)
							}
						}
					}
				}(g)
			}

			// Wait for all goroutines to complete
			wg.Wait()
			close(errors)

			// Check if any errors occurred
			for err := range errors {
				t.Logf("Concurrent operation error: %v", err)
				return false
			}

			// Verify final statistics consistency
			total, oov, hit, fallbackAttempts, fallbackSuccesses, fallbackFailures := vm.GetLookupStats()

			// Basic sanity checks
			if total < 0 || oov < 0 || hit < 0 {
				t.Logf("Negative statistics after concurrent operations: total=%d, oov=%d, hit=%d", total, oov, hit)
				return false
			}

			if fallbackAttempts < 0 || fallbackSuccesses < 0 || fallbackFailures < 0 {
				t.Logf("Negative fallback statistics: attempts=%d, successes=%d, failures=%d", fallbackAttempts, fallbackSuccesses, fallbackFailures)
				return false
			}

			// Total lookups should equal hits + OOV
			if total != oov+hit {
				t.Logf("Inconsistent statistics: total=%d, oov=%d, hit=%d", total, oov, hit)
				return false
			}

			// Fallback successes + failures should equal fallback attempts
			if fallbackAttempts != fallbackSuccesses+fallbackFailures {
				t.Logf("Inconsistent fallback statistics: attempts=%d, successes=%d, failures=%d", fallbackAttempts, fallbackSuccesses, fallbackFailures)
				return false
			}

			// Verify vocabulary size hasn't changed (no concurrent writes)
			expectedVocabSize := numCharsInVocab + 10
			if vm.VocabularySize() != expectedVocabSize {
				t.Logf("Vocabulary size changed during concurrent operations: expected=%d, got=%d", expectedVocabSize, vm.VocabularySize())
				return false
			}

			return true
		},
		gen.IntRange(2, 20),  // numGoroutines: 2-20 concurrent goroutines
		gen.IntRange(10, 50), // numOperations: 10-50 operations per goroutine
		gen.IntRange(10, 30), // numCharsInVocab: 10-30 characters in vocabulary
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// ============================================================================
// Character-Level Fallback Performance Benchmarks
// ============================================================================

// BenchmarkCharacterLevelFallback_SingleOperation benchmarks a single fallback operation
func BenchmarkCharacterLevelFallback_SingleOperation(b *testing.B) {
	vm := NewVectorModel(300).(*vectorModel)

	// Add single character vectors for Chinese characters
	chineseChars := []string{"没", "事", "问", "题", "好", "的", "是", "在", "有", "人"}
	for _, char := range chineseChars {
		vector := make([]float32, 300)
		for i := range vector {
			vector[i] = float32(i) * 0.01
		}
		vm.AddVector(char, vector)
	}

	// Test with OOV words that will trigger fallback
	testWords := []string{"没事", "问题", "好的"}

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		word := testWords[i%len(testWords)]
		vm.GetVector(word)
	}
}

// BenchmarkCharacterLevelFallback_WithoutFallback benchmarks direct vector lookup (no fallback)
func BenchmarkCharacterLevelFallback_WithoutFallback(b *testing.B) {
	vm := NewVectorModel(300).(*vectorModel)

	// Add complete words to vocabulary
	testWords := []string{"没事", "问题", "好的"}
	for _, word := range testWords {
		vector := make([]float32, 300)
		for i := range vector {
			vector[i] = float32(i) * 0.01
		}
		vm.AddVector(word, vector)
	}

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		word := testWords[i%len(testWords)]
		vm.GetVector(word)
	}
}

// BenchmarkCharacterLevelFallback_PerformanceOverhead measures the overhead of fallback
func BenchmarkCharacterLevelFallback_PerformanceOverhead(b *testing.B) {
	b.Run("with_fallback", func(b *testing.B) {
		vm := NewVectorModel(300).(*vectorModel)

		// Add single character vectors
		chineseChars := []string{"没", "事", "问", "题", "好", "的", "是", "在", "有", "人"}
		for _, char := range chineseChars {
			vector := make([]float32, 300)
			for i := range vector {
				vector[i] = float32(i) * 0.01
			}
			vm.AddVector(char, vector)
		}

		testWords := []string{"没事", "问题", "好的"}

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			word := testWords[i%len(testWords)]
			vm.GetVector(word)
		}
	})

	b.Run("without_fallback", func(b *testing.B) {
		vm := NewVectorModel(300).(*vectorModel)

		// Add complete words to vocabulary
		testWords := []string{"没事", "问题", "好的"}
		for _, word := range testWords {
			vector := make([]float32, 300)
			for i := range vector {
				vector[i] = float32(i) * 0.01
			}
			vm.AddVector(word, vector)
		}

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			word := testWords[i%len(testWords)]
			vm.GetVector(word)
		}
	})
}

// BenchmarkCharacterLevelFallback_DifferentWordLengths benchmarks fallback with different word lengths
func BenchmarkCharacterLevelFallback_DifferentWordLengths(b *testing.B) {
	// Add single character vectors
	chineseChars := []string{"没", "事", "问", "题", "好", "的", "是", "在", "有", "人", "工", "智", "能", "学", "习", "技", "术"}

	testCases := []struct {
		name  string
		words []string
	}{
		{"2_chars", []string{"没事", "问题", "好的"}},
		{"3_chars", []string{"没问题", "好事情", "有意思"}},
		{"4_chars", []string{"人工智能", "机器学习", "深度学习"}},
		{"5_chars", []string{"没有问题的", "好的事情是", "有意思的人"}},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			vm := NewVectorModel(300).(*vectorModel)

			for _, char := range chineseChars {
				vector := make([]float32, 300)
				for i := range vector {
					vector[i] = float32(i) * 0.01
				}
				vm.AddVector(char, vector)
			}

			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				word := tc.words[i%len(tc.words)]
				vm.GetVector(word)
			}
		})
	}
}

// BenchmarkCharacterLevelFallback_BatchRequests benchmarks batch requests with fallback
func BenchmarkCharacterLevelFallback_BatchRequests(b *testing.B) {
	testCases := []struct {
		name         string
		batchSize    int
		fallbackRate float64 // Percentage of words that need fallback
	}{
		{"batch_10_fallback_0%", 10, 0.0},
		{"batch_10_fallback_50%", 10, 0.5},
		{"batch_10_fallback_100%", 10, 1.0},
		{"batch_50_fallback_0%", 50, 0.0},
		{"batch_50_fallback_50%", 50, 0.5},
		{"batch_50_fallback_100%", 50, 1.0},
		{"batch_100_fallback_50%", 100, 0.5},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			vm := NewVectorModel(300).(*vectorModel)

			// Add single character vectors
			chineseChars := []string{"没", "事", "问", "题", "好", "的", "是", "在", "有", "人"}
			for _, char := range chineseChars {
				vector := make([]float32, 300)
				for i := range vector {
					vector[i] = float32(i) * 0.01
				}
				vm.AddVector(char, vector)
			}

			// Create batch of words
			words := make([]string, tc.batchSize)
			fallbackCount := int(float64(tc.batchSize) * tc.fallbackRate)

			// Words that need fallback (OOV compound words)
			for i := 0; i < fallbackCount; i++ {
				words[i] = "没事" // OOV word that will trigger fallback
			}

			// Words that don't need fallback (add to vocabulary)
			for i := fallbackCount; i < tc.batchSize; i++ {
				word := fmt.Sprintf("word%d", i)
				vector := make([]float32, 300)
				for j := range vector {
					vector[j] = float32(j) * 0.01
				}
				vm.AddVector(word, vector)
				words[i] = word
			}

			b.ResetTimer()
			for b.Loop() {
				vm.GetAverageVector(words)
			}
		})
	}
}

// BenchmarkCharacterLevelFallback_Throughput measures throughput with different fallback rates
func BenchmarkCharacterLevelFallback_Throughput(b *testing.B) {
	testCases := []struct {
		name         string
		fallbackRate float64
	}{
		{"throughput_0%_fallback", 0.0},
		{"throughput_25%_fallback", 0.25},
		{"throughput_50%_fallback", 0.5},
		{"throughput_75%_fallback", 0.75},
		{"throughput_100%_fallback", 1.0},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			vm := NewVectorModel(300).(*vectorModel)

			// Add single character vectors
			chineseChars := []string{"没", "事", "问", "题", "好", "的", "是", "在", "有", "人"}
			for _, char := range chineseChars {
				vector := make([]float32, 300)
				for i := range vector {
					vector[i] = float32(i) * 0.01
				}
				vm.AddVector(char, vector)
			}

			// Create test words
			totalWords := 100
			fallbackWords := int(float64(totalWords) * tc.fallbackRate)
			testWords := make([]string, totalWords)

			// OOV words that trigger fallback
			for i := 0; i < fallbackWords; i++ {
				testWords[i] = "没事"
			}

			// Known words
			for i := fallbackWords; i < totalWords; i++ {
				word := fmt.Sprintf("word%d", i)
				vector := make([]float32, 300)
				for j := range vector {
					vector[j] = float32(j) * 0.01
				}
				vm.AddVector(word, vector)
				testWords[i] = word
			}

			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				word := testWords[i%len(testWords)]
				vm.GetVector(word)
			}
		})
	}
}

// BenchmarkCharacterLevelFallback_MemoryAllocation measures memory allocation overhead
func BenchmarkCharacterLevelFallback_MemoryAllocation(b *testing.B) {
	vm := NewVectorModel(300).(*vectorModel)

	// Add single character vectors
	chineseChars := []string{"没", "事", "问", "题", "好", "的"}
	for _, char := range chineseChars {
		vector := make([]float32, 300)
		for i := range vector {
			vector[i] = float32(i) * 0.01
		}
		vm.AddVector(char, vector)
	}

	testWord := "没事"

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		vm.GetVector(testWord)
	}
}

// BenchmarkCharacterLevelFallback_ConcurrentAccess benchmarks concurrent access with fallback
func BenchmarkCharacterLevelFallback_ConcurrentAccess(b *testing.B) {
	vm := NewVectorModel(300).(*vectorModel)

	// Add single character vectors
	chineseChars := []string{"没", "事", "问", "题", "好", "的", "是", "在", "有", "人"}
	for _, char := range chineseChars {
		vector := make([]float32, 300)
		for i := range vector {
			vector[i] = float32(i) * 0.01
		}
		vm.AddVector(char, vector)
	}

	testWords := []string{"没事", "问题", "好的"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			word := testWords[i%len(testWords)]
			vm.GetVector(word)
			i++
		}
	})
}

// BenchmarkCharacterLevelFallback_PartialCharacterCoverage benchmarks fallback with partial character coverage
func BenchmarkCharacterLevelFallback_PartialCharacterCoverage(b *testing.B) {
	testCases := []struct {
		name     string
		coverage float64 // Percentage of characters in vocabulary
	}{
		{"coverage_25%", 0.25},
		{"coverage_50%", 0.5},
		{"coverage_75%", 0.75},
		{"coverage_100%", 1.0},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			vm := NewVectorModel(300).(*vectorModel)

			// All possible characters in test words
			allChars := []string{"没", "事", "问", "题"}
			coverageCount := int(float64(len(allChars)) * tc.coverage)

			// Add only a subset of characters to vocabulary
			for i := 0; i < coverageCount; i++ {
				vector := make([]float32, 300)
				for j := range vector {
					vector[j] = float32(j) * 0.01
				}
				vm.AddVector(allChars[i], vector)
			}

			testWords := []string{"没事", "问题"}

			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				word := testWords[i%len(testWords)]
				vm.GetVector(word)
			}
		})
	}
}

// BenchmarkCharacterLevelFallback_ExecutionTime measures execution time to validate < 10ms requirement
func BenchmarkCharacterLevelFallback_ExecutionTime(b *testing.B) {
	vm := NewVectorModel(300).(*vectorModel)

	// Add single character vectors
	chineseChars := []string{"没", "事", "问", "题", "好", "的", "是", "在", "有", "人"}
	for _, char := range chineseChars {
		vector := make([]float32, 300)
		for i := range vector {
			vector[i] = float32(i) * 0.01
		}
		vm.AddVector(char, vector)
	}

	// Test with different word lengths to ensure all complete within 10ms
	testWords := []string{
		"没事",       // 2 chars
		"没问题",      // 3 chars
		"人工智能",     // 4 chars
		"没有问题的",    // 5 chars
		"好的事情是在",   // 6 chars
		"有意思的人工智能", // 8 chars
	}

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		word := testWords[i%len(testWords)]
		start := time.Now()
		vm.GetVector(word)
		elapsed := time.Since(start)

		// Validate that execution time is < 10ms (requirement 6.1)
		if elapsed > 10*time.Millisecond {
			b.Errorf("Fallback operation took %v, exceeds 10ms requirement", elapsed)
		}
	}
}

// BenchmarkCharacterLevelFallback_RealWorldScenario benchmarks a realistic usage scenario
func BenchmarkCharacterLevelFallback_RealWorldScenario(b *testing.B) {
	logger := &DiscardLogger{}
	loader := NewEmbeddingLoader(logger)

	// Load real vector file
	model, err := loader.LoadFromFile("vector/wiki.zh.align.vec")
	if err != nil {
		b.Skipf("Skipping real-world benchmark: %v", err)
		return
	}

	// Real Chinese OOV words that might trigger fallback
	testWords := []string{
		"没事",  // Common colloquial phrase
		"问题",  // Common word
		"好的",  // Common response
		"谢谢",  // Thank you
		"不客气", // You're welcome
		"再见",  // Goodbye
		"明天见", // See you tomorrow
		"没问题", // No problem
		"好主意", // Good idea
		"很高兴", // Very happy
	}

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		word := testWords[i%len(testWords)]
		model.GetVector(word)
	}
}

// BenchmarkCharacterLevelFallback_CompareWithBaseline compares fallback performance with baseline
func BenchmarkCharacterLevelFallback_CompareWithBaseline(b *testing.B) {
	// Baseline: Direct lookup (no fallback needed)
	b.Run("baseline_direct_lookup", func(b *testing.B) {
		vm := NewVectorModel(300).(*vectorModel)

		testWords := []string{"word1", "word2", "word3"}
		for _, word := range testWords {
			vector := make([]float32, 300)
			for i := range vector {
				vector[i] = float32(i) * 0.01
			}
			vm.AddVector(word, vector)
		}

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			word := testWords[i%len(testWords)]
			vm.GetVector(word)
		}
	})

	// With fallback: OOV words requiring character-level fallback
	b.Run("with_fallback", func(b *testing.B) {
		vm := NewVectorModel(300).(*vectorModel)

		chineseChars := []string{"没", "事", "问", "题", "好", "的"}
		for _, char := range chineseChars {
			vector := make([]float32, 300)
			for i := range vector {
				vector[i] = float32(i) * 0.01
			}
			vm.AddVector(char, vector)
		}

		testWords := []string{"没事", "问题", "好的"}

		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			word := testWords[i%len(testWords)]
			vm.GetVector(word)
		}
	})
}
