// Package promptcache provides a semantic prompt-response cache that stores
// LLM responses keyed by prompt similarity. It reduces token consumption and
// latency by reusing responses for identical or semantically similar prompts.
//
// The cache uses two-tier matching:
//  1. Exact SHA-256 hash match (fast, no embedding needed)
//  2. Cosine similarity on embeddings (fuzzy/semantic match)
//
// The caller provides an embedding function, keeping the dep dependency-free.
package promptcache

import "math"

// CosineSimilarity computes the cosine similarity between two vectors.
// Returns a value in [-1, 1], where 1 = identical direction.
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		fa := float64(a[i])
		fb := float64(b[i])
		dot += fa * fb
		normA += fa * fa
		normB += fb * fb
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return float32(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
}

// Float32ToBytes encodes a []float32 to a byte slice for storage.
func Float32ToBytes(v []float32) []byte {
	b := make([]byte, len(v)*4)
	for i, f := range v {
		bits := math.Float32bits(f)
		b[i*4] = byte(bits >> 24)
		b[i*4+1] = byte(bits >> 16)
		b[i*4+2] = byte(bits >> 8)
		b[i*4+3] = byte(bits)
	}
	return b
}

// BytesToFloat32 decodes a byte slice back to []float32.
func BytesToFloat32(b []byte) []float32 {
	if len(b)%4 != 0 {
		return nil
	}
	v := make([]float32, len(b)/4)
	for i := range v {
		bits := uint32(b[i*4])<<24 | uint32(b[i*4+1])<<16 | uint32(b[i*4+2])<<8 | uint32(b[i*4+3])
		v[i] = math.Float32frombits(bits)
	}
	return v
}
