//go:build !darwin

package nve

// SaveFileVersion is a no-op on non-macOS platforms.
func SaveFileVersion(path string) {}
