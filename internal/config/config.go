package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ProjectName   string
	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration
	ServerPort    string
	DBUser        string
	DBPassword    string
	DBHost        string
	DBPort        string
	DBName        string
}

func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}

	return &Config{
		ProjectName:   getEnv("PROJECT_NAME", "qulpunoy"),
		JWTSecret:     getEnv("JWT_SECRET", "secret"),
		JWTAccessTTL:  getEnvMinutes("ACCESS_TOKEN_EXPIRE_MINUTES", 15),
		JWTRefreshTTL: getEnvMinutes("REFRESH_TOKEN_EXPIRE_MINUTES", 7*24*60),
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", "postgres"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBName:        getEnv("DB_NAME", "postgres"),
	}, nil
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvMinutes(key string, defaultMinutes int) time.Duration {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if m, err := strconv.Atoi(val); err == nil {
			return time.Duration(m) * time.Minute
		}
		slog.Warn("Invalid minutes in env, using default", "key", key, "value", val)
	}
	return time.Duration(defaultMinutes) * time.Minute
}

func (c *Config) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, "disable",
	)
}

func InitLogger() *slog.Logger {
	var handler slog.Handler

	handler = slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true},
	)

	return slog.New(handler)
}
