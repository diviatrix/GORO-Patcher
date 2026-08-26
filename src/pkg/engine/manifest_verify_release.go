//go:build release

package engine

func MissingKeyAllowsUnsigned() bool {
	return false
}