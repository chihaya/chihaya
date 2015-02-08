// Copyright 2015 The faststats Authors. All rights reserved.
// Use of this source code is governed by the BSD 2-Clause license,
// which can be found in the LICENSE file.

package faststats

import (
	"math"
	"math/rand"
)

func round(value float64) int64 {
	if value < 0.0 {
		value -= 0.5
	} else {
		value += 0.5
	}

	return int64(value)
}

func uniform(n int, scale float64) []float64 {
	numbers := make([]float64, n)

	for i := 0; i < n; i++ {
		numbers[i] = rand.Float64() * scale
	}

	return numbers
}

func logNorm(n int, scale float64) []float64 {
	numbers := make([]float64, n)

	for i := 0; i < n; i++ {
		numbers[i] = math.Exp(rand.NormFloat64()) * scale
	}

	return numbers
}
