package main

import (
	"math"
	"math/rand"
	"time"
)

func NewRandFromSource(src rand.Source) *rand.Rand {
	if src != nil {
		return rand.New(src)
	}

	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

func RandIntInRange(min, max int) int {
	diff := max - min
	dist := math.Round(rand.Float64() * float64(diff))
	return min + int(dist)
}

// Returns a slice of random bytes of the specified length, between the specified ranges.
// The range from which to insert the random byte is also chosen randomly, once per iteration.
// Panics if a range outside a byte is provided.
func RandByteSliceFromRanges(length int, ranges [][]int) []byte {
	intSlice := []byte{}

	/* 	if asciiLimit < 0 {
	   		asciiLimit = 0
	   	}

	   	if asciiLimit > math.MaxInt8 {
	   		asciiLimit = math.MaxInt8
	   	} */

	// 48-57
	// 65-90
	// 97-122
	for i := 0; i < length; i++ {
		currRange := ranges[rand.Intn(len(ranges))]
		intSlice = append(intSlice, byte(RandIntInRange(currRange[0], currRange[1])))
	}

	return intSlice
}
