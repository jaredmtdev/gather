// Package op - used for math operations
package op

// Integer - integer types.
type Integer interface {
	~int | ~int32 | ~int64 | ~uint | ~uint32 | ~uint64
}

// PosMod - modulus operator that always returns positive number.
func PosMod[T Integer](x, m T) T {
	return (x%m + m) % m
}
