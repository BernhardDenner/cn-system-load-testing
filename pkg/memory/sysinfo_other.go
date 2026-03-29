//go:build !linux

package memory

// availableMemoryBytes returns a conservative default on non-Linux platforms.
func availableMemoryBytes() int64 {
	return 1 << 30 // 1 GB
}
