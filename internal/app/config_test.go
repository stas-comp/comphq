package app

import "testing"

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("COMPHQ_DATA_DIR", "")
	t.Setenv("COMPHQ_ADDR", "")
	t.Setenv("COMPHQ_TEST_MODE", "")

	cfg := LoadConfig()

	if cfg.DataDir != "/data" {
		t.Errorf("DataDir = %q, want /data", cfg.DataDir)
	}
	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want :8080", cfg.Addr)
	}
	if cfg.TestMode {
		t.Error("TestMode = true, want false by default")
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("COMPHQ_DATA_DIR", "/tmp/comphq-data")
	t.Setenv("COMPHQ_ADDR", ":9090")
	t.Setenv("COMPHQ_TEST_MODE", "1")

	cfg := LoadConfig()

	if cfg.DataDir != "/tmp/comphq-data" {
		t.Errorf("DataDir = %q, want /tmp/comphq-data", cfg.DataDir)
	}
	if cfg.Addr != ":9090" {
		t.Errorf("Addr = %q, want :9090", cfg.Addr)
	}
	if !cfg.TestMode {
		t.Error("TestMode = false, want true when COMPHQ_TEST_MODE=1")
	}
}
