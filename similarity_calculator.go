package semanticmatcher

import "math"

// similarityCalculator implements the SimilarityCalculator interface
type similarityCalculator struct{}

// NewSimilarityCalculator creates a new SimilarityCalculator instance
func NewSimilarityCalculator() SimilarityCalculator {
	return &similarityCalculator{}
}

// isZeroVector checks if a vector is a zero vector (all elements are zero)
func isZeroVector(v []float32) bool {
	for _, val := range v {
		if val != 0.0 {
			return false
		}
	}
	return true
}

// isValidVector checks if a vector is valid (not nil, not empty, not zero)
func isValidVector(v []float32) bool {
	if len(v) == 0 {
		return false
	}
	return !isZeroVector(v)
}

// CosineSimilarity computes cosine similarity between two vectors
// Returns a value between -1 and 1, where 1 means identical direction
// Returns 0.0 for invalid inputs (empty, nil, mismatched dimensions, or zero vectors)
// Formula: cos(θ) = (v1 · v2) / (||v1|| * ||v2||)
func (*similarityCalculator) CosineSimilarity(v1, v2 []float32) float64 {
	// Check Validate input vectors and dimension mismatch
	if !isValidVector(v1) || !isValidVector(v2) || len(v1) != len(v2) {
		return 0.0
	}

	var dotProduct, norm1, norm2 float64

	// Compute dot product and norms in a single pass for efficiency
	for i := range v1 {
		dotProduct += float64(v1[i]) * float64(v2[i])
		norm1 += float64(v1[i]) * float64(v1[i])
		norm2 += float64(v2[i]) * float64(v2[i])
	}

	// Handle zero vectors - return 0.0 for undefined similarity
	if norm1 == 0.0 || norm2 == 0.0 {
		return 0.0
	}

	// Compute cosine similarity
	similarity := dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))

	// Clamp result to [-1, 1] to handle floating point precision issues
	if similarity > 1.0 {
		similarity = 1.0
	} else if similarity < -1.0 {
		similarity = -1.0
	}

	return similarity
}

// BatchSimilarity computes similarities between one query vector and multiple candidate vectors
// This is optimized for computing multiple similarities at once
// Returns empty slice for invalid query, and 0.0 for invalid candidates
func (*similarityCalculator) BatchSimilarity(query []float32, candidates [][]float32) []float64 {
	// Validate query vector
	if len(query) == 0 {
		return []float64{}
	}

	// Handle empty candidates
	if len(candidates) == 0 {
		return []float64{}
	}

	results := make([]float64, len(candidates))

	// Pre-compute query norm once for all comparisons
	var queryNorm float64
	for i := range query {
		queryNorm += float64(query[i]) * float64(query[i])
	}

	// Handle zero query vector - return all zeros
	if queryNorm == 0.0 {
		return results // All zeros
	}

	querySqrtNorm := math.Sqrt(queryNorm)

	// Compute similarity for each candidate
	for idx, candidate := range candidates {
		// Validate candidate vector
		if len(candidate) == 0 || len(candidate) != len(query) {
			results[idx] = 0.0
			continue
		}

		var dotProduct, candidateNorm float64

		// Compute dot product and candidate norm
		for i := range query {
			dotProduct += float64(query[i]) * float64(candidate[i])
			candidateNorm += float64(candidate[i]) * float64(candidate[i])
		}

		// Handle zero candidate vector
		if candidateNorm == 0.0 {
			results[idx] = 0.0
			continue
		}

		// Compute cosine similarity
		similarity := dotProduct / (querySqrtNorm * math.Sqrt(candidateNorm))

		// Clamp result to [-1, 1] to handle floating point precision issues
		if similarity > 1.0 {
			similarity = 1.0
		} else if similarity < -1.0 {
			similarity = -1.0
		}

		results[idx] = similarity
	}

	return results
}
