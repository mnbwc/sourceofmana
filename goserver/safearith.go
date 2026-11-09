package main

import "math"

// safeAdd returns a+b with int64 overflow saturation.
func safeAdd(a, b int64) int64 {
	if b > 0 && a > math.MaxInt64-b {
		return math.MaxInt64
	}
	if b < 0 && a < math.MinInt64-b {
		return math.MinInt64
	}
	return a + b
}

// safeSub returns a-b with int64 overflow saturation.
func safeSub(a, b int64) int64 {
	if b < 0 && a > math.MaxInt64+b {
		return math.MaxInt64
	}
	if b > 0 && a < math.MinInt64+b {
		return math.MinInt64
	}
	return a - b
}

// safeMul returns a*b with int64 overflow saturation.
func safeMul(a, b int64) int64 {
	if a == 0 || b == 0 {
		return 0
	}
	result := a * b
	if result/b != a {
		// Overflow occurred; return saturated value.
		if (a > 0 && b > 0) || (a < 0 && b < 0) {
			return math.MaxInt64
		}
		return math.MinInt64
	}
	return result
}
