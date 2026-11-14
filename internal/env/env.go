// Package env holds small environment helpers shared by the module's
// runnable demos (example and cmd/health-hub).
package env

import "os"

// GetOrDefault returns the environment value for key, falling back when the
// variable is unset or empty.
func GetOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
