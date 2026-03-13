package types

import (
	"errors"
	"os"
	"strings"
)

const (
	defaultPort           = "8080"
	defaultAllowedOrigins = "http://localhost:5173,http://127.0.0.1:5173"
	defaultAppAuthValue   = "dev-app-auth"
	defaultProfileStore   = "memory"
)

type Config struct {
	AllowedOrigins []string
	AppBearerToken string
	DatabaseURL    string
	Environment    string
	Port           string
	ProfileStore   string
}

func LoadConfig() Config {
	environment := getEnv("APP_ENV", "development")
	allowedOriginsValue := getEnv("ALLOWED_ORIGINS", defaultAllowedOrigins)
	allowedOrigins := make([]string, 0)
	for _, rawOrigin := range strings.Split(allowedOriginsValue, ",") {
		origin := strings.TrimSpace(rawOrigin)
		if origin != "" {
			allowedOrigins = append(allowedOrigins, origin)
		}
	}

	appBearerToken := strings.TrimSpace(os.Getenv("APP_BEARER_TOKEN"))
	if environment == "development" && appBearerToken == "" {
		appBearerToken = defaultAppAuthValue
	}

	return Config{
		AllowedOrigins: allowedOrigins,
		AppBearerToken: appBearerToken,
		DatabaseURL:    strings.TrimSpace(os.Getenv("DATABASE_URL")),
		Environment:    environment,
		Port:           getEnv("PORT", defaultPort),
		ProfileStore:   getEnv("APP_PROFILE_STORE", defaultProfileStore),
	}
}

func (c Config) AppAuthorizationValue() string {
	return "Bearer " + c.AppBearerToken
}

func (c Config) Validate() error {
	if c.Environment != "development" && strings.TrimSpace(c.AppBearerToken) == "" {
		return errors.New("APP_BEARER_TOKEN is required when APP_ENV is not development")
	}

	return nil
}

func getEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}
