// @title          Edutalks API
// @version        1.0
// @description    Документация API Edutalks (регистрация, логин, токены, статьи, логи и т.д.).
// @contact.name   EduTalks Support
// @contact.url    https://edutalks.ru
// @contact.email  support@edutalks.ru
// @host           edutalks.ru
// @BasePath       /api
// @schemes        https
// @securityDefinitions.apikey ApiKeyAuth
// @in             header
// @name           Authorization
package main

import (
	_ "edutalks/docs"
	"os"

	"edutalks/internal/app"
	"edutalks/internal/config"
	"edutalks/internal/logger"

	"context"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
)

// main
func main() {
	// 1) Загружаем конфиг
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	// 2) Инициализируем логгер
	if err := logger.Init(logger.Options{
		Env:     cfg.Env,      // "prod"/"dev"
		Level:   cfg.LogLevel, // "info", "debug" и т.д.
		Service: "edutalks",
	}); err != nil {
		panic(err)
	}
	defer func() { _ = logger.Log.Sync() }()

	// 3) Инициализируем приложение (роутер, зависимости) и получаем cleanup
	router, cleanup, err := app.InitApp(cfg)
	if err != nil {
		logger.Log.Fatal("Ошибка инициализации приложения", zap.Error(err))
	}

	// 4) Swagger
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// 5) CORS. Для AllowCredentials=true нельзя звездочку в AllowedOrigins.
	corsMiddleware := cors.Handler(cors.Options{
		AllowOriginFunc:  func(r *http.Request, origin string) bool { return true }, // вернёт конкретный Origin
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "Accept", "X-Requested-With"},
		ExposedHeaders:   []string{"Authorization", "Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           86400,
	})

	// 6) HTTP-сервер с таймаутами
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      corsMiddleware(router),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 20 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 7) Запуск + graceful shutdown
	errCh := make(chan error, 1)
	go func() {
		logger.Log.Info("Сервер запущен", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// ловим SIGINT/SIGTERM
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-stopCh:
		logger.Log.Info("Получен сигнал остановки, выключаемся...")
	case err := <-errCh:
		logger.Log.Error("Критическая ошибка сервера", zap.Error(err))
	}

	// корректная остановка
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// сначала останавливаем HTTP
	_ = srv.Shutdown(ctx)

	// затем внутренние фоновые задачи/очереди
	cleanup()

	logger.Log.Info("Сервер остановлен корректно")
}
