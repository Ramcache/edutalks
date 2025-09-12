package main

import (
	_ "edutalks/docs"
	"edutalks/internal/app"
	"edutalks/internal/config"
	"edutalks/internal/logger"
	"net/http"

	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
)

// @title          Edutalks API
// @version        1.0
// @description    Документация API Edutalks (регистрация, логин, токены, статьи, логи и т.д.).

// @contact.name   EduTalks Support
// @contact.url    https://edutalks.ru
// @contact.email  support@edutalks.ru

// @host      edutalks.ru
// @BasePath  /
// @schemes   https

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
func main() {
	cfg, err := config.LoadConfig()
	logger.InitLogger()
	defer logger.Log.Sync()

	if err != nil {
		logger.Log.Fatal("Ошибка загрузки конфига", zap.Error(err))
	}

	router, err := app.InitApp(cfg)
	if err != nil {
		logger.Log.Fatal("Ошибка инициализации приложения", zap.Error(err))
	}

	// Swagger по префиксу /swagger/
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Максимально permissive CORS:
	// - любой Origin/порт (через AllowOriginFunc)
	// - любые методы
	// - любые заголовки
	// - credentials включены (cookies/Authorization)
	corsMiddleware := cors.Handler(cors.Options{
		// Если нужен прямой wildcard, можно оставить AllowedOrigins: []string{"*"},
		// но для credentials это запрещено спецификацией.
		// AllowOriginFunc вернёт true для любого Origin и библиотека подставит его в заголовок.
		AllowOriginFunc: func(r *http.Request, origin string) bool { return true },

		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"*"},
		AllowCredentials: true,
		MaxAge:           86400, // кэш preflight на сутки
	})

	logger.Log.Info("Сервер запущен", zap.String("port", cfg.Port))
	if err := http.ListenAndServe(":"+cfg.Port, corsMiddleware(router)); err != nil {
		logger.Log.Fatal("Ошибка запуска сервера", zap.Error(err))
	}
}
