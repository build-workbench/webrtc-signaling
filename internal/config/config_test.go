package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("SIGNAL_JWT_SECRET", "test-secret-for-defaults")
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
	if cfg.Observability.MetricsAddr != ":9090" {
		t.Fatalf("expected default metrics addr :9090, got %s", cfg.Observability.MetricsAddr)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("SIGNAL_ADDR", ":9090")
	t.Setenv("SIGNAL_LOG_LEVEL", "debug")
	t.Setenv("SIGNAL_JWT_SECRET", "test-secret-for-env")
	t.Setenv("SIGNAL_ALLOWED_ORIGINS", "https://a.example, https://b.example")
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

func TestValidatePongLessThanPing(t *testing.T) {
	t.Setenv("SIGNAL_JWT_SECRET", "test-secret-for-validate")
	cfg := Load()
	cfg.Server.PongWaitSec = 5
	cfg.Server.PingIntervalSec = 10
	_, err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error when pong <= ping")
	}
}

func TestValidateMaxMsgBytesZero(t *testing.T) {
	t.Setenv("SIGNAL_JWT_SECRET", "test-secret-for-validate")
	cfg := Load()
	cfg.Server.MaxMsgBytes = 0
	_, err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error when maxMsgBytes <= 0")
	}
}

func TestValidateRedisEnabledRequiresAddr(t *testing.T) {
	t.Setenv("SIGNAL_JWT_SECRET", "test-secret-for-validate")
	cfg := Load()
	cfg.Redis.Enabled = true
	cfg.Redis.Addr = ""
	_, err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error when redis enabled without addr")
	}
}

func TestNormalizeRole(t *testing.T) {
	cases := map[string]string{
		"":           "speaker",
		"speaker":    "speaker",
		"Speaker":    "speaker",
		" viewer ":   "viewer",
		"moderator":  "moderator",
		"not-a-role": "",
	}
	for input, want := range cases {
		if got := NormalizeRole(input); got != want {
			t.Fatalf("NormalizeRole(%q) = %q, want %q", input, got, want)
		}
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

func TestGetEnvHelpers(t *testing.T) {
	t.Setenv("TEST_STRING", "value")
	if got := getEnv("TEST_STRING", "fallback"); got != "value" {
		t.Fatalf("expected existing value, got %q", got)
	}
	if got := getEnv("TEST_MISSING", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}

	t.Setenv("TEST_INT", "123")
	if got := getEnvInt("TEST_INT", 42); got != 123 {
		t.Fatalf("expected int 123, got %d", got)
	}
	t.Setenv("TEST_INT", "invalid")
	if got := getEnvInt("TEST_INT", 42); got != 42 {
		t.Fatalf("expected fallback int 42, got %d", got)
	}

	boolCases := []struct {
		value string
		want  bool
	}{
		{"true", true},
		{"1", true},
		{"yes", true},
		{"false", false},
		{"0", false},
		{"no", false},
	}
	for _, tc := range boolCases {
		t.Setenv("TEST_BOOL", tc.value)
		if got := getEnvBool("TEST_BOOL", false); got != tc.want {
			t.Fatalf("getEnvBool(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}
