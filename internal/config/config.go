package config

import (
	"fmt"
	"os"
	"strings"

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
	Env      string // dev|prod

	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string

	SiteURL           string
	SiteURLNews       string
	YooKassaShopID    string
	YooKassaSecret    string
	YooKassaReturnURL string

	FrontendURL         string
	PasswordResetTTLMin string
}

// LoadConfig загружает .env, читает переменные окружения и выставляет дефолты.
// Ничего не логирует — чтобы не создавать зависимость от logger.
func LoadConfig() (*Config, error) {
	_ = godotenv.Load(".env")

	def := func(v, d string) string {
		v = strings.TrimSpace(v)
		if v == "" {
			return d
		}
		return v
	}

	cfg := &Config{
		Port:      def(os.Getenv("PORT"), "8080"),
		DbHost:    os.Getenv("DB_HOST"),
		DbPort:    def(os.Getenv("DB_PORT"), "5432"),
		DbUser:    os.Getenv("DB_USER"),
		DbPass:    os.Getenv("DB_PASSWORD"),
		DbName:    os.Getenv("DB_NAME"),
		DbSSLMode: def(os.Getenv("DB_SSLMODE"), "disable"),

		JWTSecret:       os.Getenv("JWT_SECRET"),
		AccessTokenTTL:  def(os.Getenv("ACCESS_TOKEN_EXPIRY"), "15m"),
		RefreshTokenTTL: def(os.Getenv("REFRESH_TOKEN_EXPIRY"), "720h"),

		Log:      os.Getenv("LOG"),
		LogLevel: strings.ToLower(def(os.Getenv("LOGLEVEL"), "info")),
		Env:      strings.ToLower(def(os.Getenv("ENV"), "prod")),

		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     def(os.Getenv("SMTP_PORT"), "587"),
		SMTPUser:     os.Getenv("SMTP_USER"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),

		SiteURL:             os.Getenv("SITEURL"),
		SiteURLNews:         os.Getenv("SITEURLNEWS"),
		YooKassaReturnURL:   os.Getenv("YOOKASSA_RETURN_URL"),
		YooKassaSecret:      os.Getenv("YOOKASSA_SECRET"),
		YooKassaShopID:      os.Getenv("YOOKASSA_SHOP_ID"),
		FrontendURL:         os.Getenv("FRONTEND_URL"),
		PasswordResetTTLMin: def(os.Getenv("PASSWORD_RESET_TTL_MIN"), "30"),
	}

	return cfg, nil
}

// Validate возвращает предупреждения и фатальную ошибку (если критично).
func (c *Config) Validate() (warnings []string, err error) {
	// Критичные: БД
	if c.DbHost == "" || c.DbUser == "" || c.DbName == "" {
		return nil, fmt.Errorf("incomplete DB config (DB_HOST/DB_USER/DB_NAME)")
	}

	// JWT — предупреждение (можешь сделать ошибкой, если нужно)
	if strings.TrimSpace(c.JWTSecret) == "" {
		warnings = append(warnings, "JWT_SECRET is empty")
	}

	// YooKassa — предупреждение, если проект может работать и без оплат
	if c.YooKassaShopID == "" || c.YooKassaSecret == "" {
		warnings = append(warnings, "YooKassa credentials are not set")
	}

	// SMTP — предупреждение
	if c.SMTPHost == "" || c.SMTPUser == "" {
		warnings = append(warnings, "SMTP is not fully configured")
	}

	// PORT
	if c.Port == "" {
		warnings = append(warnings, "PORT is empty, using default 8080")
	}

	return warnings, nil
}

// GetDSN — полная DSN (с паролем)
func (c *Config) GetDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DbUser, c.DbPass, c.DbHost, c.DbPort, c.DbName, c.DbSSLMode,
	)
}

// GetDSNSafe — DSN без пароля (для логов)
func (c *Config) GetDSNSafe() string {
	return fmt.Sprintf(
		"postgres://%s:***@%s:%s/%s?sslmode=%s",
		c.DbUser, c.DbHost, c.DbPort, c.DbName, c.DbSSLMode,
	)
}
