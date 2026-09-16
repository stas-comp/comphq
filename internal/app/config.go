// Package app holds the router, frame layout, and config shared by every
// section (SPEC B2).
package app

import "os"

// Config is read from the environment (SPEC P1-03 behaviour).
type Config struct {
	DataDir string
	Addr    string
}

// LoadConfig reads Config from the environment, applying defaults.
func LoadConfig() Config {
	return Config{
		DataDir: getenv("COMPHQ_DATA_DIR", "/data"),
		Addr:    getenv("COMPHQ_ADDR", ":8080"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
