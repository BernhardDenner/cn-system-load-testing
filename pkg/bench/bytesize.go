package bench

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var byteSizeRe = regexp.MustCompile(`(?i)^\s*(\d+)\s*(b|kb|k|mb|m|gb|g)?\s*$`)

// ParseByteSize parses a human-readable byte size string into a number of bytes.
// Accepted suffixes (case-insensitive): b, k/kb, m/mb, g/gb.
// A plain number without suffix is interpreted as bytes.
func ParseByteSize(s string) (int64, error) {
	m := byteSizeRe.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("invalid byte size %q: use a number with optional unit (b, kb, mb, gb)", s)
	}

	n, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid byte size %q: %w", s, err)
	}

	switch strings.ToLower(m[2]) {
	case "", "b":
		return n, nil
	case "k", "kb":
		return n * 1024, nil
	case "m", "mb":
		return n * 1024 * 1024, nil
	case "g", "gb":
		return n * 1024 * 1024 * 1024, nil
	default:
		return 0, fmt.Errorf("invalid byte size unit %q", m[2])
	}
}
