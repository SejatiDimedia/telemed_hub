package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application.
// Loaded once at startup from environment variables (12-factor style).
// Passed via dependency injection — no global config lookups.
type Config struct {
	App                              AppConfig
	DB                               DBConfig
	Redis                            RedisConfig
	MinIO                            MinIOConfig
	JWT                              JWTConfig
	Server                           ServerConfig
	LLM                              LLMConfig
	Midtrans                         MidtransConfig
	AppointmentCancelCutoffMinutes int
}

type MidtransConfig struct {
	ServerKey   string
	ClientKey   string
	Environment string
}

type AppConfig struct {
	Env      string // "development", "staging", "production"
	Name     string
	LogLevel string
}

type ServerConfig struct {
	Port            int
	ShutdownTimeout time.Duration
}

type DBConfig struct {
	URL string
}

type RedisConfig struct {
	URL string
}

type MinIOConfig struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	UseSSL     bool
	BucketName string
}

type JWTConfig struct {
	Secret         string
	AccessTTL      time.Duration
	RefreshTTLDays int
}

type LLMConfig struct {
	APIKey string
	APIURL string
}

// Load reads all configuration from environment variables.
// Returns an error if any required variable is missing.
func Load() (*Config, error) {
	cfg := &Config{}

	// App
	cfg.App.Env = getEnvOrDefault("APP_ENV", "development")
	cfg.App.Name = getEnvOrDefault("APP_NAME", "telemedhub")
	cfg.App.LogLevel = getEnvOrDefault("LOG_LEVEL", "debug")

	// Server
	port, err := getEnvAsInt("HTTP_PORT", 8080)
	if err != nil {
		return nil, fmt.Errorf("invalid HTTP_PORT: %w", err)
	}
	cfg.Server.Port = port

	shutdownSec, err := getEnvAsInt("SHUTDOWN_TIMEOUT_SECONDS", 15)
	if err != nil {
		return nil, fmt.Errorf("invalid SHUTDOWN_TIMEOUT_SECONDS: %w", err)
	}
	cfg.Server.ShutdownTimeout = time.Duration(shutdownSec) * time.Second

	// Database (required)
	cfg.DB.URL, err = getEnvRequired("DATABASE_URL")
	if err != nil {
		return nil, err
	}

	// Redis (required)
	cfg.Redis.URL, err = getEnvRequired("REDIS_URL")
	if err != nil {
		return nil, err
	}

	// MinIO
	cfg.MinIO.Endpoint = getEnvOrDefault("MINIO_ENDPOINT", "minio:9000")
	cfg.MinIO.AccessKey = getEnvOrDefault("MINIO_ACCESS_KEY", "minioadmin")
	cfg.MinIO.SecretKey = getEnvOrDefault("MINIO_SECRET_KEY", "minioadmin_secret")
	cfg.MinIO.UseSSL = getEnvOrDefault("MINIO_USE_SSL", "false") == "true"
	cfg.MinIO.BucketName = getEnvOrDefault("MINIO_BUCKET_NAME", "telemedhub")

	// JWT
	cfg.JWT.Secret = getEnvOrDefault("JWT_SECRET", "")
	accessTTL, err := getEnvAsInt("JWT_ACCESS_TTL_SECONDS", 900)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TTL_SECONDS: %w", err)
	}
	cfg.JWT.AccessTTL = time.Duration(accessTTL) * time.Second

	refreshDays, err := getEnvAsInt("JWT_REFRESH_TTL_DAYS", 30)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_TTL_DAYS: %w", err)
	}
	cfg.JWT.RefreshTTLDays = refreshDays

	cutoffMins, err := getEnvAsInt("APPOINTMENT_CANCEL_CUTOFF_MINUTES", 60)
	if err != nil {
		return nil, fmt.Errorf("invalid APPOINTMENT_CANCEL_CUTOFF_MINUTES: %w", err)
	}
	cfg.AppointmentCancelCutoffMinutes = cutoffMins

	// LLM
	cfg.LLM.APIKey = getEnvOrDefault("LLM_API_KEY", "")
	cfg.LLM.APIURL = getEnvOrDefault("LLM_API_URL", "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent")

	// Midtrans
	cfg.Midtrans.ServerKey = getEnvOrDefault("MIDTRANS_SERVER_KEY", "")
	cfg.Midtrans.ClientKey = getEnvOrDefault("MIDTRANS_CLIENT_KEY", "")
	cfg.Midtrans.Environment = getEnvOrDefault("MIDTRANS_ENVIRONMENT", "sandbox")

	return cfg, nil
}

// IsDevelopment returns true if the app is running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

// IsProduction returns true if the app is running in production mode.
func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}

// --- helpers ---

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvRequired(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return val, nil
}

func getEnvAsInt(key string, defaultVal int) (int, error) {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal, nil
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0, fmt.Errorf("env %s=%q is not a valid integer: %w", key, valStr, err)
	}
	return val, nil
}
