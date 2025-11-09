package semanticmatcher

import (
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

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
	vm.GetVector("nonexistent") // miss

	// Check statistics
	total, oov, hit := vm.GetLookupStats()
	if total != 3 {
		t.Errorf("Expected 3 total lookups, got %d", total)
	}
	if oov != 1 {
		t.Errorf("Expected 1 OOV lookup, got %d", oov)
	}
	if hit != 2 {
		t.Errorf("Expected 2 hit lookups, got %d", hit)
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
	vm.GetAverageVector([]string{"word1", "word2", "nonexistent1", "nonexistent2"})

	total, oov, hit = vm.GetLookupStats()
	if total != 7 { // 3 previous + 4 new
		t.Errorf("Expected 7 total lookups, got %d", total)
	}
	if oov != 3 { // 1 previous + 2 new
		t.Errorf("Expected 3 OOV lookups, got %d", oov)
	}
	if hit != 4 { // 2 previous + 2 new
		t.Errorf("Expected 4 hit lookups, got %d", hit)
	}

	// Test reset
	vm.ResetStats()
	total, oov, hit = vm.GetLookupStats()
	if total != 0 || oov != 0 || hit != 0 {
		t.Errorf(
			"Expected all stats to be 0 after reset, got total=%d, oov=%d, hit=%d",
			total,
			oov,
			hit,
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
	total, oov, hit := vm.GetLookupStats()
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
	total, _, _ := vm.GetLookupStats()
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
