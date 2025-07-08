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
	DbSSLMode string // 👈 добавили sslmode

	JWTSecret       string
	AccessTokenTTL  string
	RefreshTokenTTL string
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
