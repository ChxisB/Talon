package frugal

import "fmt"

// sprintf is a convenience alias for fmt.Sprintf
func sprintf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}
