package semanticmatcher

import (
	"math"
	"testing"
)

func TestCosineSimilarity(t *testing.T) {
	calc := NewSimilarityCalculator()

	tests := []struct {
		name     string
		v1       []float32
		v2       []float32
		expected float64
		epsilon  float64
	}{
		{
			name:     "identical vectors",
			v1:       []float32{1.0, 2.0, 3.0},
			v2:       []float32{1.0, 2.0, 3.0},
			expected: 1.0,
			epsilon:  1e-6,
		},
		{
			name:     "orthogonal vectors",
			v1:       []float32{1.0, 0.0, 0.0},
			v2:       []float32{0.0, 1.0, 0.0},
			expected: 0.0,
			epsilon:  1e-6,
		},
		{
			name:     "opposite vectors",
			v1:       []float32{1.0, 2.0, 3.0},
			v2:       []float32{-1.0, -2.0, -3.0},
			expected: -1.0,
			epsilon:  1e-6,
		},
		{
			name:     "similar vectors",
			v1:       []float32{1.0, 2.0, 3.0},
			v2:       []float32{2.0, 4.0, 6.0},
			expected: 1.0,
			epsilon:  1e-6,
		},
		{
			name:     "partially similar vectors",
			v1:       []float32{1.0, 0.0, 0.0},
			v2:       []float32{1.0, 1.0, 0.0},
			expected: 0.7071067811865475, // 1/sqrt(2)
			epsilon:  1e-6,
		},
		{
			name:     "empty vectors",
			v1:       []float32{},
			v2:       []float32{},
			expected: 0.0,
			epsilon:  1e-6,
		},
		{
			name:     "zero vector v1",
			v1:       []float32{0.0, 0.0, 0.0},
			v2:       []float32{1.0, 2.0, 3.0},
			expected: 0.0,
			epsilon:  1e-6,
		},
		{
			name:     "zero vector v2",
			v1:       []float32{1.0, 2.0, 3.0},
			v2:       []float32{0.0, 0.0, 0.0},
			expected: 0.0,
			epsilon:  1e-6,
		},
		{
			name:     "mismatched dimensions",
			v1:       []float32{1.0, 2.0},
			v2:       []float32{1.0, 2.0, 3.0},
			expected: 0.0,
			epsilon:  1e-6,
		},
		{
			name:     "high dimensional vectors",
			v1:       []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
			v2:       []float32{0.15, 0.25, 0.35, 0.45, 0.55, 0.65, 0.75, 0.85, 0.95, 1.05},
			expected: 0.9999,
			epsilon:  1e-3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calc.CosineSimilarity(tt.v1, tt.v2)
			if math.Abs(result-tt.expected) > tt.epsilon {
				t.Errorf(
					"CosineSimilarity() = %v, expected %v (epsilon %v)",
					result,
					tt.expected,
					tt.epsilon,
				)
			}
		})
	}
}

func TestBatchSimilarity(t *testing.T) {
	calc := NewSimilarityCalculator()

	tests := []struct {
		name       string
		query      []float32
		candidates [][]float32
		expected   []float64
		epsilon    float64
	}{
		{
			name:  "multiple candidates",
			query: []float32{1.0, 2.0, 3.0},
			candidates: [][]float32{
				{1.0, 2.0, 3.0},    // identical
				{2.0, 4.0, 6.0},    // same direction
				{-1.0, -2.0, -3.0}, // opposite
				{0.0, 1.0, 0.0},    // orthogonal-ish
			},
			expected: []float64{1.0, 1.0, -1.0, 0.5345224838248488},
			epsilon:  1e-6,
		},
		{
			name:       "empty candidates",
			query:      []float32{1.0, 2.0, 3.0},
			candidates: [][]float32{},
			expected:   []float64{},
			epsilon:    1e-6,
		},
		{
			name:       "empty query",
			query:      []float32{},
			candidates: [][]float32{{1.0, 2.0, 3.0}},
			expected:   []float64{},
			epsilon:    1e-6,
		},
		{
			name:  "zero query vector",
			query: []float32{0.0, 0.0, 0.0},
			candidates: [][]float32{
				{1.0, 2.0, 3.0},
				{4.0, 5.0, 6.0},
			},
			expected: []float64{0.0, 0.0},
			epsilon:  1e-6,
		},
		{
			name:  "zero candidate vector",
			query: []float32{1.0, 2.0, 3.0},
			candidates: [][]float32{
				{1.0, 2.0, 3.0},
				{0.0, 0.0, 0.0},
				{2.0, 4.0, 6.0},
			},
			expected: []float64{1.0, 0.0, 1.0},
			epsilon:  1e-6,
		},
		{
			name:  "mismatched dimensions",
			query: []float32{1.0, 2.0, 3.0},
			candidates: [][]float32{
				{1.0, 2.0, 3.0},
				{1.0, 2.0}, // wrong dimension
				{2.0, 4.0, 6.0},
			},
			expected: []float64{1.0, 0.0, 1.0},
			epsilon:  1e-6,
		},
		{
			name:  "single candidate",
			query: []float32{1.0, 0.0, 0.0},
			candidates: [][]float32{
				{1.0, 1.0, 0.0},
			},
			expected: []float64{0.7071067811865475},
			epsilon:  1e-6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := calc.BatchSimilarity(tt.query, tt.candidates)

			if len(results) != len(tt.expected) {
				t.Fatalf(
					"BatchSimilarity() returned %d results, expected %d",
					len(results),
					len(tt.expected),
				)
			}

			for i, result := range results {
				if math.Abs(result-tt.expected[i]) > tt.epsilon {
					t.Errorf(
						"BatchSimilarity()[%d] = %v, expected %v (epsilon %v)",
						i,
						result,
						tt.expected[i],
						tt.epsilon,
					)
				}
			}
		})
	}
}

func BenchmarkCosineSimilarity(b *testing.B) {
	calc := NewSimilarityCalculator()
	v1 := make([]float32, 300)
	v2 := make([]float32, 300)

	for i := range v1 {
		v1[i] = float32(i) * 0.01
		v2[i] = float32(i) * 0.02
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.CosineSimilarity(v1, v2)
	}
}

func BenchmarkBatchSimilarity(b *testing.B) {
	calc := NewSimilarityCalculator()
	query := make([]float32, 300)
	candidates := make([][]float32, 100)

	for i := range query {
		query[i] = float32(i) * 0.01
	}

	for i := range candidates {
		candidates[i] = make([]float32, 300)
		for j := range candidates[i] {
			candidates[i][j] = float32(j) * 0.01 * float32(i+1)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.BatchSimilarity(query, candidates)
	}
}
