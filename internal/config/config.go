package config

import (
	"os"
	"strconv"
)

type Config struct {
	Environment               string
	DatabaseURL               string
	JWTSecret                 string
	FastAPIURL                string
	FastAPIKey                string
	SMTPHost                  string
	SMTPPort                  int
	SMTPUsername              string
	SMTPPassword              string
	FromEmail                 string
	RateLimitRPS              int
	RateLimitBurst            int
	AbstractEmailAPIKey       string
	AbstractPhoneNumberAPIKey string
	BaseURL                   string // Base URL for the application, used in email links
}

func Load() *Config {
	smtpPort, _ := strconv.Atoi(getEnv("SMTP_PORT", "587"))
	rateLimitRPS, _ := strconv.Atoi(getEnv("RATE_LIMIT_RPS", "100"))
	rateLimitBurst, _ := strconv.Atoi(getEnv("RATE_LIMIT_BURST", "200"))

	return &Config{
		Environment:               getEnv("ENVIRONMENT", "development"),
		DatabaseURL:               getEnv("DATABASE_URL", "postgres://user:password@localhost/ecommerce?sslmode=disable"),
		JWTSecret:                 getEnv("JWT_SECRET", "your-super-secret-jwt-key"),
		FastAPIURL:                getEnv("FASTAPI_URL", "http://localhost:8000"),
		FastAPIKey:                getEnv("FASTAPI_INTERNAL_KEY", "your-internal-api-key"),
		SMTPHost:                  getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:                  smtpPort,
		SMTPUsername:              getEnv("SMTP_USERNAME", ""),
		SMTPPassword:              getEnv("SMTP_PASSWORD", ""),
		FromEmail:                 getEnv("FROM_EMAIL", "noreply@yourapp.com"),
		RateLimitRPS:              rateLimitRPS,
		RateLimitBurst:            rateLimitBurst,
		AbstractEmailAPIKey:       getEnv("ABSTRACT_EMAIL_API_KEY", ""),
		AbstractPhoneNumberAPIKey: getEnv("ABSTRACT_PHONE_NUMBER_API_KEY", ""),
		BaseURL:                   getEnv("BASE_URL", "http://localhost:8080"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
