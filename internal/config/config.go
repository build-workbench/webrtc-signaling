package config

import (
	"os"
	"strconv"
	"strings"
)

type ICEServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
	TTL        int      `json:"ttl,omitempty"`
}

type ServerCfg struct {
	Addr            string   `json:"addr"`
	AllowedOrigins  []string `json:"allowedOrigins"`
	ReadTimeoutSec  int      `json:"readTimeoutSec"`
	WriteTimeoutSec int      `json:"writeTimeoutSec"`
	MaxMsgBytes     int      `json:"maxMsgBytes"`
	PingIntervalSec int      `json:"pingIntervalSec"`
	PongWaitSec     int      `json:"pongWaitSec"`
}

type SecurityCfg struct {
	JWTSecret string `json:"jwtSecret"`
	AdminKey  string `json:"adminKey"`
	RateLimit RateLimitCfg `json:"rateLimit"`
}

type RateLimitCfg struct {
	WSPerConnRPS int `json:"wsPerConnRps"`
	WSBurst      int `json:"wsBurst"`
}

type RedisCfg struct {
	Enabled  bool   `json:"enabled"`
	Addr     string `json:"addr"`
	DB       int    `json:"db"`
	Password string `json:"password"`
}

type ObservabilityCfg struct {
	PrometheusEnabled bool   `json:"prometheusEnabled"`
	MetricsAddr       string `json:"metricsAddr"`
}

type TurnCfg struct {
	STUN []string    `json:"stun"`
	TURN []ICEServer `json:"turn"`
}

type Config struct {
	Server        ServerCfg        `json:"server"`
	Security      SecurityCfg      `json:"security"`
	Redis         RedisCfg         `json:"redis"`
	Turn          TurnCfg          `json:"turn"`
	Observability ObservabilityCfg `json:"observability"`
}

func Load() *Config {
	cfg := &Config{
		Server: ServerCfg{
			Addr:            getEnv("SIGNAL_ADDR", ":8080"),
			AllowedOrigins:  split(getEnv("SIGNAL_ALLOWED_ORIGINS", "")),
			ReadTimeoutSec:  getEnvInt("SIGNAL_READ_TIMEOUT", 10),
			WriteTimeoutSec: getEnvInt("SIGNAL_WRITE_TIMEOUT", 10),
			MaxMsgBytes:     getEnvInt("SIGNAL_MAX_MSG_BYTES", 65536),
			PingIntervalSec: getEnvInt("SIGNAL_WS_PING_INTERVAL", 10),
			PongWaitSec:     getEnvInt("SIGNAL_WS_PONG_WAIT", 25),
		},
		Security: SecurityCfg{
			JWTSecret: getEnv("SIGNAL_JWT_SECRET", "dev-secret-change"),
			AdminKey:  getEnv("SIGNAL_ADMIN_KEY", ""),
			RateLimit: RateLimitCfg{
				WSPerConnRPS: getEnvInt("SIGNAL_WS_RPS", 20),
				WSBurst:      getEnvInt("SIGNAL_WS_BURST", 40),
			},
		},
		Redis: RedisCfg{
			Enabled:  getEnvBool("SIGNAL_REDIS_ENABLED", false),
			Addr:     getEnv("SIGNAL_REDIS_ADDR", "redis:6379"),
			DB:       getEnvInt("SIGNAL_REDIS_DB", 0),
			Password: getEnv("SIGNAL_REDIS_PASSWORD", ""),
		},
		Turn: TurnCfg{
			STUN: split(getEnv("SIGNAL_STUN", "stun:stun.l.google.com:19302")),
			TURN: []ICEServer{},
		},
		Observability: ObservabilityCfg{
			PrometheusEnabled: getEnvBool("SIGNAL_PROM_ENABLED", true),
			MetricsAddr:       getEnv("SIGNAL_METRICS_ADDR", ":9090"),
		},
	}

	// Optional TURN via env
	if tu := strings.TrimSpace(os.Getenv("SIGNAL_TURN_URLS")); tu != "" {
		cfg.Turn.TURN = append(cfg.Turn.TURN, ICEServer{
			URLs:       split(tu),
			Username:   getEnv("SIGNAL_TURN_USERNAME", ""),
			Credential: getEnv("SIGNAL_TURN_CREDENTIAL", ""),
			TTL:        getEnvInt("SIGNAL_TURN_TTL", 600),
		})
	}
	return cfg
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

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		v = strings.ToLower(v)
		return v == "1" || v == "true" || v == "yes"
	}
	return def
}
