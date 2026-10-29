package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Monitor  MonitorConfig
}

type ServerConfig struct {
	Port            int
	Host            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	CORSOrigins     []string
}

type DatabaseConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type AuthConfig struct {
	JWTSecret   string
	TokenExpiry time.Duration
	BCryptCost  int
}

type MonitorConfig struct {
	WorkerCount         int
	SchedulerInterval   time.Duration
	DefaultInterval     time.Duration
	DefaultTimeout      time.Duration
	MaxResponseBodyRead int64
	// AllowPrivateIPs enables checking private/loopback addresses — development only.
	AllowPrivateIPs bool
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{}

	cfg.Server = ServerConfig{
		Port:            envInt("PORT", 8080),
		Host:            envStr("HOST", ""),
		ReadTimeout:     envDuration("SERVER_READ_TIMEOUT", 15*time.Second),
		WriteTimeout:    envDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:     envDuration("SERVER_IDLE_TIMEOUT", 120*time.Second),
		ShutdownTimeout: envDuration("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		CORSOrigins:     envStringSlice("CORS_ORIGINS", []string{"http://localhost:5173"}),
	}

	dbDSN := envStr("DATABASE_URL", "")
	if dbDSN == "" {
		dbDSN = buildDSN()
	}
	cfg.Database = DatabaseConfig{
		DSN:             dbDSN,
		MaxOpenConns:    envInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    envInt("DB_MAX_IDLE_CONNS", 10),
		ConnMaxLifetime: envDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
	}

	jwtSecret := envStr("JWT_SECRET", "")
	if jwtSecret == "" {
		return nil, errors.New("JWT_SECRET must be set")
	}
	if len(jwtSecret) < 32 {
		return nil, errors.New("JWT_SECRET must be at least 32 characters")
	}
	cfg.Auth = AuthConfig{
		JWTSecret:   jwtSecret,
		TokenExpiry: envDuration("JWT_EXPIRY", 24*time.Hour),
		BCryptCost:  envInt("BCRYPT_COST", 12),
	}

	cfg.Monitor = MonitorConfig{
		WorkerCount:         envInt("MONITOR_WORKER_COUNT", 10),
		SchedulerInterval:   envDuration("MONITOR_SCHEDULER_INTERVAL", 30*time.Second),
		DefaultInterval:     envDuration("MONITOR_DEFAULT_INTERVAL", 5*time.Minute),
		DefaultTimeout:      envDuration("MONITOR_DEFAULT_TIMEOUT", 30*time.Second),
		MaxResponseBodyRead: int64(envInt("MONITOR_MAX_RESPONSE_BODY_BYTES", 1<<20)), // 1 MiB
		AllowPrivateIPs:     envBool("MONITOR_ALLOW_PRIVATE_IPS", false),
	}

	return cfg, nil
}

func buildDSN() string {
	host := envStr("DB_HOST", "localhost")
	port := envInt("DB_PORT", 5432)
	user := envStr("DB_USER", "servicewatch")
	pass := envStr("DB_PASSWORD", "")
	name := envStr("DB_NAME", "servicewatch")
	sslmode := envStr("DB_SSLMODE", "disable")
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pass, name, sslmode)
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func envStringSlice(key string, def []string) []string {
	if v := os.Getenv(key); v != "" {
		parts := strings.Split(v, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				result = append(result, s)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return def
}
