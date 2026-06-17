package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv         string
	HTTPAddr       string
	FrontendOrigin string
	Database       DatabaseConfig
	AuthServiceURL string
	Etsy           EtsyConfig
	Security       SecurityConfig
	Sync           SyncConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
	MaxConns int32
	MinConns int32
}

type EtsyConfig struct {
	APIBaseURL        string
	OAuthAuthorizeURL string
	OAuthTokenURL     string
	ClientID          string
	APIKey            string
	SharedSecret      string
	RedirectURI       string
	Scopes            string
}

type SecurityConfig struct {
	TokenEncryptionKey string
}

type SyncConfig struct {
	MaxConcurrency int
	RequestTimeout time.Duration
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:         getEnv("APP_ENV", "development"),
		HTTPAddr:       getEnv("HTTP_ADDR", ":8082"),
		FrontendOrigin: getEnv("FRONTEND_ORIGIN", "http://localhost:5173"),
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "stok_user"),
			Password: getEnv("DB_PASSWORD", "stok_password"),
			Name:     getEnv("DB_NAME", "stock_etsy_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
			MaxConns: int32(getEnvInt("DB_MAX_CONNS", 10)),
			MinConns: int32(getEnvInt("DB_MIN_CONNS", 1)),
		},
		AuthServiceURL: getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
		Etsy: EtsyConfig{
			APIBaseURL:        getEnv("ETSY_API_BASE_URL", "https://openapi.etsy.com/v3/application"),
			OAuthAuthorizeURL: getEnv("ETSY_OAUTH_AUTHORIZE_URL", "https://www.etsy.com/oauth/connect"),
			OAuthTokenURL:     getEnv("ETSY_OAUTH_TOKEN_URL", "https://api.etsy.com/v3/public/oauth/token"),
			ClientID:          getEnv("ETSY_CLIENT_ID", ""),
			APIKey:            getEnv("ETSY_API_KEY", ""),
			SharedSecret:      getEnv("ETSY_SHARED_SECRET", ""),
			RedirectURI:       getEnv("ETSY_REDIRECT_URI", "http://localhost:8082/etsy/oauth/callback"),
			Scopes:            getEnv("ETSY_SCOPES", "shops_r listings_r"),
		},
		Security: SecurityConfig{
			TokenEncryptionKey: getEnv("TOKEN_ENCRYPTION_KEY", ""),
		},
		Sync: SyncConfig{
			MaxConcurrency: getEnvInt("SYNC_MAX_CONCURRENCY", 2),
			RequestTimeout: getEnvDuration("SYNC_REQUEST_TIMEOUT", 15*time.Second),
		},
	}

	if cfg.Database.Name == "" {
		return nil, fmt.Errorf("DB_NAME is required")
	}

	return cfg, nil
}

func (c DatabaseConfig) URL() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
		Path:   c.Name,
	}

	q := u.Query()
	q.Set("sslmode", c.SSLMode)
	u.RawQuery = q.Encode()

	return u.String()
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}
