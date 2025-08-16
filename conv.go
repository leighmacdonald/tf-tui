package main

import (
	"strconv"

	"golang.org/x/exp/constraints"
)

func parseInt(s string, def int) int {
	index, errIndex := strconv.ParseInt(s, 10, 32)
	if errIndex != nil {
		return def
	}

	return int(index)
}

func parseBool(s string) bool {
	val, errParse := strconv.ParseBool(s)
	if errParse != nil {
		return false
	}

	return val
}

type Number interface {
	constraints.Integer | constraints.Float
}

func clamp[T Number](v, low, high T) T {
	if high < low {
		low, high = high, low
	}

	return min(high, max(low, v))
}
