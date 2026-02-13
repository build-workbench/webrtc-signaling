package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg := Load()
	if cfg.Server.Addr != ":8080" {
		t.Fatalf("expected default addr :8080, got %s", cfg.Server.Addr)
	}
	if cfg.Server.MaxMsgBytes != 65536 {
		t.Fatalf("expected default maxMsgBytes 65536, got %d", cfg.Server.MaxMsgBytes)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("expected default log level info, got %s", cfg.LogLevel)
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("SIGNAL_ADDR", ":9090")
	os.Setenv("SIGNAL_LOG_LEVEL", "debug")
	defer os.Unsetenv("SIGNAL_ADDR")
	defer os.Unsetenv("SIGNAL_LOG_LEVEL")

	cfg := Load()
	if cfg.Server.Addr != ":9090" {
		t.Fatalf("expected addr :9090, got %s", cfg.Server.Addr)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("expected log level debug, got %s", cfg.LogLevel)
	}
}

func TestValidateOK(t *testing.T) {
	cfg := Load()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestValidatePongLessThanPing(t *testing.T) {
	cfg := Load()
	cfg.Server.PongWaitSec = 5
	cfg.Server.PingIntervalSec = 10
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error when pong <= ping")
	}
}

func TestValidateMaxMsgBytesZero(t *testing.T) {
	cfg := Load()
	cfg.Server.MaxMsgBytes = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error when maxMsgBytes <= 0")
	}
}

func TestGetEnvBool(t *testing.T) {
	cases := []struct {
		val  string
		want bool
	}{
		{"true", true},
		{"1", true},
		{"yes", true},
		{"false", false},
		{"0", false},
		{"no", false},
	}
	for _, tc := range cases {
		os.Setenv("TEST_BOOL", tc.val)
		got := getEnvBool("TEST_BOOL", false)
		if got != tc.want {
			t.Errorf("getEnvBool(%q) = %v, want %v", tc.val, got, tc.want)
		}
	}
	os.Unsetenv("TEST_BOOL")
}
