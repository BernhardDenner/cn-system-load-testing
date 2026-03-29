//go:build linux

package memory

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

// availableMemoryBytes returns the memory limit visible to this process.
// It checks cgroup v2, then cgroup v1, then falls back to total system RAM.
func availableMemoryBytes() int64 {
	// cgroup v2: /sys/fs/cgroup/memory.max ("max" means unlimited)
	if data, err := os.ReadFile("/sys/fs/cgroup/memory.max"); err == nil {
		s := strings.TrimSpace(string(data))
		if s != "max" {
			if v, err := strconv.ParseInt(s, 10, 64); err == nil && v > 0 {
				return v
			}
		}
	}

	// cgroup v1: /sys/fs/cgroup/memory/memory.limit_in_bytes
	// Reports a very large sentinel (close to int64 max) when unlimited.
	if data, err := os.ReadFile("/sys/fs/cgroup/memory/memory.limit_in_bytes"); err == nil {
		if v, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); err == nil && v > 0 && v < 1<<62 {
			return v
		}
	}

	// Fall back to total physical RAM via sysinfo(2).
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err == nil {
		return int64(info.Totalram) * int64(info.Unit)
	}

	return 1 << 30 // 1 GB last resort
}
