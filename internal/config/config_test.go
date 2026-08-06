package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("SIGNAL_JWT_SECRET", "test-secret-for-defaults")
	cfg := Load()
	if cfg.Server.Addr != ":8080" {
		t.Fatalf("expected default addr :8080, got %s", cfg.Server.Addr)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("expected default log level info, got %s", cfg.LogLevel)
	}
	if cfg.RedisAddr != "" {
		t.Fatalf("expected redis disabled by default, got %q", cfg.RedisAddr)
	}
	if len(cfg.STUN) != 1 || cfg.STUN[0] != "stun:stun.l.google.com:19302" {
		t.Fatalf("expected default google STUN, got %v", cfg.STUN)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("SIGNAL_ADDR", ":9090")
	t.Setenv("SIGNAL_LOG_LEVEL", "debug")
	t.Setenv("SIGNAL_JWT_SECRET", "test-secret-for-env")
	t.Setenv("SIGNAL_ALLOWED_ORIGINS", "https://a.example, https://b.example")
	t.Setenv("SIGNAL_REDIS_ADDR", "redis:6379")
	cfg := Load()
	if cfg.Server.Addr != ":9090" {
		t.Fatalf("expected addr :9090, got %s", cfg.Server.Addr)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("expected log level debug, got %s", cfg.LogLevel)
	}
	if len(cfg.Server.AllowedOrigins) != 2 {
		t.Fatalf("expected 2 allowed origins, got %d", len(cfg.Server.AllowedOrigins))
	}
	if cfg.RedisAddr != "redis:6379" {
		t.Fatalf("expected redis addr redis:6379, got %q", cfg.RedisAddr)
	}
}

func TestValidateOK(t *testing.T) {
	t.Setenv("SIGNAL_JWT_SECRET", "0123456789abcdef0123456789abcdef")
	cfg := Load()
	warnings, err := cfg.Validate()
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
}

func TestValidateRequiresJWTSecret(t *testing.T) {
	cfg := Load()
	cfg.Security.JWTSecret = ""
	_, err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error when JWT secret is missing")
	}
}

func TestValidateWarnsOnShortSecret(t *testing.T) {
	cfg := Load()
	cfg.Security.JWTSecret = "short-secret"
	warnings, err := cfg.Validate()
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %v", warnings)
	}
}

func TestValidateJoinTokenTTL(t *testing.T) {
	if got := ValidateJoinTokenTTL(0); got != 900 {
		t.Fatalf("expected default ttl 900, got %d", got)
	}
	if got := ValidateJoinTokenTTL(60); got != 60 {
		t.Fatalf("expected ttl 60, got %d", got)
	}
	if got := ValidateJoinTokenTTL(7200); got != 3600 {
		t.Fatalf("expected ttl clamp 3600, got %d", got)
	}
}

func TestIsOriginAllowed(t *testing.T) {
	allowed := []string{"https://app.example.com", "https://admin.example.com"}
	if !IsOriginAllowed(allowed, "https://app.example.com") {
		t.Fatal("expected exact origin match")
	}
	if !IsOriginAllowed(allowed, " https://admin.example.com ") {
		t.Fatal("expected trimmed origin match")
	}
	if IsOriginAllowed(allowed, "https://evil.example.com") {
		t.Fatal("unexpected origin match")
	}
	if IsOriginAllowed(allowed, "") {
		t.Fatal("empty origin should not be allowed")
	}
}

func TestSplitTrimsEmptyValues(t *testing.T) {
	got := split(" a, ,b ,, c ")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("expected %d values, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("split[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestGetEnv(t *testing.T) {
	t.Setenv("TEST_STRING", "value")
	if got := getEnv("TEST_STRING", "fallback"); got != "value" {
		t.Fatalf("expected existing value, got %q", got)
	}
	if got := getEnv("TEST_MISSING", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}
}
