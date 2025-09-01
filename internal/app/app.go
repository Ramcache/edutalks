package app

import (
	"context"
	"edutalks/internal/config"
	"edutalks/internal/db"
	"edutalks/internal/handlers"
	"edutalks/internal/repository"
	"edutalks/internal/routes"
	"edutalks/internal/services"
	"time"

	"github.com/gorilla/mux"
)

func InitApp(cfg *config.Config) (*mux.Router, error) {
	conn, err := db.NewPostgresConnection(cfg)
	if err != nil {
		return nil, err
	}
	// Репозитории
	userRepo := repository.NewUserRepository(conn)
	docRepo := repository.NewDocumentRepository(conn)
	newsRepo := repository.NewNewsRepository(conn)
	emailTokenRepo := repository.NewEmailTokenRepository(conn)
	articleRepo := repository.NewArticleRepo(conn)
	taxonomyRepo := repository.NewTaxonomyRepo(conn)
	subsRepo := repository.NewSubscriptionRepository(conn)

	// Сервисы
	authService := services.NewAuthService(userRepo)
	docService := services.NewDocumentService(docRepo)
	emailService := services.NewEmailService(cfg)
	newsService := services.NewNewsService(newsRepo, userRepo, emailService, cfg)
	emailTokenService := services.NewEmailTokenService(emailTokenRepo, userRepo)
	emaService := services.NewEmailService(cfg)
	articleSvc := services.NewArticleService(articleRepo)
	taxonomySvc := services.NewTaxonomyService(taxonomyRepo)
	notifier := services.NewNotifier(subsRepo, cfg.SiteURL, "Edutalks")

	yookassaService := services.NewYooKassaService(
		cfg.YooKassaShopID,
		cfg.YooKassaSecret,
		cfg.YooKassaReturnURL,
	)
	// Хендлеры
	authHandler := handlers.NewAuthHandler(authService, emailService, emailTokenService)
	docHandler := handlers.NewDocumentHandler(docService, authService, notifier)
	newsHandler := handlers.NewNewsHandler(newsService, notifier)
	emailHandler := handlers.NewEmailHandler(emailTokenService)
	searchHandler := handlers.NewSearchHandler(newsService, docService)
	articleH := handlers.NewArticleHandler(articleSvc, notifier)
	taxonomyH := handlers.NewTaxonomyHandler(taxonomySvc)

	paymentHandler := handlers.NewPaymentHandler(yookassaService)
	webhookHandler := handlers.NewWebhookHandler(authService)

	_ = userRepo.ExpireSubscriptions(context.Background())

	StartSubscriptionCleaner(userRepo)

	for i := 0; i < 3; i++ {
		go services.StartEmailWorker(emaService)
	}

	// Маршруты
	router := mux.NewRouter()
	routes.InitRoutes(router, authHandler, docHandler, newsHandler, emailHandler, searchHandler, paymentHandler, webhookHandler, articleH, taxonomyH)

	return router, nil
}

func StartSubscriptionCleaner(repo *repository.UserRepository) {
	t := time.NewTicker(1 * time.Hour)
	go func() {
		for range t.C {
			_ = repo.ExpireSubscriptions(context.Background())
		}
	}()
}
