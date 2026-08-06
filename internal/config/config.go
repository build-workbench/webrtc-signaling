package config

import (
	"fmt"
	"os"
	"strings"
)

type ServerCfg struct {
	Addr           string   `json:"addr"`
	AllowedOrigins []string `json:"allowedOrigins"`
}

type SecurityCfg struct {
	JWTSecret string `json:"jwtSecret"`
	AdminKey  string `json:"adminKey"`
}

type Config struct {
	LogLevel  string      `json:"logLevel"`
	Server    ServerCfg   `json:"server"`
	Security  SecurityCfg `json:"security"`
	RedisAddr string      `json:"redisAddr"` // non-empty enables Redis Pub/Sub scale-out
	STUN      []string    `json:"stun"`
}

func Load() *Config {
	return &Config{
		LogLevel: getEnv("SIGNAL_LOG_LEVEL", "info"),
		Server: ServerCfg{
			Addr:           getEnv("SIGNAL_ADDR", ":8080"),
			AllowedOrigins: split(getEnv("SIGNAL_ALLOWED_ORIGINS", "")),
		},
		Security: SecurityCfg{
			JWTSecret: getEnv("SIGNAL_JWT_SECRET", ""),
			AdminKey:  getEnv("SIGNAL_ADMIN_KEY", ""),
		},
		RedisAddr: getEnv("SIGNAL_REDIS_ADDR", ""),
		STUN:      split(getEnv("SIGNAL_STUN", "stun:stun.l.google.com:19302")),
	}
}

func (c *Config) Validate() (warnings []string, err error) {
	if strings.TrimSpace(c.Security.JWTSecret) == "" {
		return warnings, fmt.Errorf("SIGNAL_JWT_SECRET is required")
	}
	if len(c.Security.JWTSecret) < 16 {
		warnings = append(warnings, "JWT secret is shorter than 16 characters, consider using a stronger secret")
	}
	return warnings, nil
}

func ValidateJoinTokenTTL(ttlSeconds int) int {
	const (
		defaultTTLSeconds = 900
		maxTTLSeconds     = 3600
	)
	if ttlSeconds <= 0 {
		return defaultTTLSeconds
	}
	if ttlSeconds > maxTTLSeconds {
		return maxTTLSeconds
	}
	return ttlSeconds
}

func IsOriginAllowed(allowed []string, origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return false
	}
	for _, candidate := range allowed {
		if strings.EqualFold(strings.TrimSpace(candidate), origin) {
			return true
		}
	}
	return false
}

func split(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			res = append(res, p)
		}
	}
	return res
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
