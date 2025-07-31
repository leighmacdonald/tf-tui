package main

import "strconv"

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
