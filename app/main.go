package main

import (
	_ "edutalks/docs"
	"edutalks/internal/app"
	"edutalks/internal/config"
	"edutalks/internal/logger"
	"net/http"

	"github.com/rs/cors"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
)

// @title Edutalks API
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @version 1.0
// @description Документация API Edutalks (регистрация, логин, токены и т.д.).
// @host http://85.143.175.100:8080
// @BasePath /
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

	logger.Log.Info("Сервер запущен", zap.String("port", cfg.Port))

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
	})

	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	if err := http.ListenAndServe(":"+cfg.Port, corsMiddleware.Handler(router)); err != nil {
		logger.Log.Fatal("Ошибка запуска сервера", zap.Error(err))
	}
}
