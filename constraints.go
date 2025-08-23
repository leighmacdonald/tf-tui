package main

import (
	"golang.org/x/exp/constraints"
)

type Number interface {
	constraints.Integer | constraints.Float
}

func clamp[T Number](v, low, high T) T {
	if high < low {
		low, high = high, low
	}

	return min(high, max(low, v))
}
