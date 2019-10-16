package strings

import "fmt"

// HumanReadableByteCount converts a size in bytes to a human-readable string decimal
// Taken from here:
// https://yourbasic.org/golang/formatting-byte-size-to-human-readable-format/
func HumanReadableByteCount(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}
