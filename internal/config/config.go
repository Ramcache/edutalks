package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port      string
	DbHost    string
	DbPort    string
	DbUser    string
	DbPass    string
	DbName    string
	DbSSLMode string

	JWTSecret       string
	AccessTokenTTL  string
	RefreshTokenTTL string

	Log      string
	LogLevel string

	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
}

// LoadConfig загружает переменные из .env
func LoadConfig() (*Config, error) {
	_ = godotenv.Load(".env")

	cfg := &Config{
		Port:            os.Getenv("PORT"),
		DbHost:          os.Getenv("DB_HOST"),
		DbPort:          os.Getenv("DB_PORT"),
		DbUser:          os.Getenv("DB_USER"),
		DbPass:          os.Getenv("DB_PASSWORD"),
		DbName:          os.Getenv("DB_NAME"),
		DbSSLMode:       os.Getenv("DB_SSLMODE"), // 👈
		JWTSecret:       os.Getenv("JWT_SECRET"),
		AccessTokenTTL:  os.Getenv("ACCESS_TOKEN_EXPIRY"),
		RefreshTokenTTL: os.Getenv("REFRESH_TOKEN_EXPIRY"),
		Log:             os.Getenv("Log"),
		LogLevel:        os.Getenv("LogLevel"),
		SMTPHost:        os.Getenv("SMTP_HOST"),
		SMTPPort:        os.Getenv("SMTP_PORT"),
		SMTPUser:        os.Getenv("SMTP_USER"),
		SMTPPassword:    os.Getenv("SMTP_PASSWORD"),
	}
	return cfg, nil
}

// GetDSN собирает строку подключения для pgx/pq
func (c *Config) GetDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DbUser, c.DbPass, c.DbHost, c.DbPort, c.DbName, c.DbSSLMode,
	)
}
