package helpers

import "slices"

// IsUserDeveloper reports whether userID is in the injected developers list.
// The list is loaded once from config and passed in — no per-call disk I/O.
func IsUserDeveloper(userID int64, developers []int64) bool {
	return slices.Contains(developers, userID)
}
