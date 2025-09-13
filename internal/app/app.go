package app

import (
	"context"
	"edutalks/internal/config"
	"edutalks/internal/db"
	"edutalks/internal/handlers"
	"edutalks/internal/logger"
	"edutalks/internal/repository"
	"edutalks/internal/routes"
	"edutalks/internal/services"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// InitApp возвращает router, cleanup-функцию и ошибку.
func InitApp(cfg *config.Config) (*mux.Router, func(), error) {
	// DB
	conn, err := db.NewPostgresConnection(cfg)
	if err != nil {
		logger.Log.Error("Не удалось подключиться к Postgres", zap.Error(err))
		return nil, nil, err
	}
	logger.Log.Info("Подключение к Postgres успешно")

	// Репозитории
	userRepo := repository.NewUserRepository(conn)
	docRepo := repository.NewDocumentRepository(conn)
	newsRepo := repository.NewNewsRepository(conn)
	emailTokenRepo := repository.NewEmailTokenRepository(conn)
	articleRepo := repository.NewArticleRepo(conn)
	taxonomyRepo := repository.NewTaxonomyRepo(conn)
	subsRepo := repository.NewSubscriptionRepository(conn)
	pwdResetRepo := repository.NewPasswordResetRepository(conn)

	// Сервисы
	emailService := services.NewEmailService(cfg) // <-- единственный экземпляр
	authService := services.NewAuthService(userRepo)
	docService := services.NewDocumentService(docRepo)
	newsService := services.NewNewsService(newsRepo, userRepo, emailService, cfg)
	emailTokenService := services.NewEmailTokenService(emailTokenRepo, userRepo)
	articleSvc := services.NewArticleService(articleRepo)
	taxonomySvc := services.NewTaxonomyService(taxonomyRepo)
	notifier := services.NewNotifier(subsRepo, taxonomyRepo, cfg.SiteURLNews, "Edutalks")
	passwordSvc := services.NewPasswordService(pwdResetRepo, emailService, cfg.FrontendURL)
	yookassaService := services.NewYooKassaService(
		cfg.YooKassaShopID,
		cfg.YooKassaSecret,
		cfg.YooKassaReturnURL,
	)

	// Хендлеры
	authHandler := handlers.NewAuthHandler(authService, emailService, emailTokenService)
	docHandler := handlers.NewDocumentHandler(docService, authService, notifier, taxonomyRepo)
	newsHandler := handlers.NewNewsHandler(newsService, notifier)
	emailHandler := handlers.NewEmailHandler(emailTokenService)
	searchHandler := handlers.NewSearchHandler(newsService, docService)
	articleH := handlers.NewArticleHandler(articleSvc, notifier)
	taxonomyH := handlers.NewTaxonomyHandler(taxonomySvc)
	paymentHandler := handlers.NewPaymentHandler(yookassaService)
	webhookHandler := handlers.NewWebhookHandler(authService)
	passwordHandler := handlers.NewPasswordHandler(passwordSvc, userRepo)
	logsAdminH := handlers.NewAdminLogsHandler()

	// Применяем параметры воркера из .env (интервалы, ретраи, размер батча)
	services.ConfigureEmailWorkerFromEnv(cfg)

	// Запуск почтовых воркеров — начни с одного (дозированная отправка)
	services.StartEmailWorker(1, emailService)

	// Чистка подписок при старте
	if err := userRepo.ExpireSubscriptions(context.Background()); err != nil {
		logger.Log.Warn("Не удалось выполнить ExpireSubscriptions при старте", zap.Error(err))
	} else {
		logger.Log.Info("ExpireSubscriptions при старте выполнен")
	}
	stopCleaner := startSubscriptionCleaner(userRepo)

	// Маршруты
	router := mux.NewRouter()
	routes.InitRoutes(
		router,
		authHandler, docHandler, newsHandler, emailHandler,
		searchHandler, paymentHandler, webhookHandler,
		articleH, taxonomyH,
		passwordHandler,
		logsAdminH,
	)

	logger.Log.Info("Приложение инициализировано")

	// cleanup: закрываем email-очередь и останавливаем планировщик
	cleanup := func() {
		services.StopEmailWorkers() // закрывает канал и завершает горутины-воркеры
		stopCleaner()
	}

	return router, cleanup, nil
}

func startSubscriptionCleaner(repo *repository.UserRepository) func() {
	ticker := time.NewTicker(1 * time.Hour)
	done := make(chan struct{})

	go func() {
		logger.Log.Info("SubscriptionCleaner запущен")
		for {
			select {
			case <-ticker.C:
				if err := repo.ExpireSubscriptions(context.Background()); err != nil {
					logger.Log.Error("Ошибка в ExpireSubscriptions", zap.Error(err))
				} else {
					logger.Log.Debug("ExpireSubscriptions выполнен по расписанию")
				}
			case <-done:
				ticker.Stop()
				logger.Log.Info("SubscriptionCleaner остановлен")
				return
			}
		}
	}()

	return func() { close(done) }
}
