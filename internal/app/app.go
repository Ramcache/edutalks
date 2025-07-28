package app

import (
	"edutalks/internal/config"
	"edutalks/internal/db"
	"edutalks/internal/handlers"
	"edutalks/internal/repository"
	"edutalks/internal/routes"
	"edutalks/internal/services"

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

	// Сервисы
	authService := services.NewAuthService(userRepo)
	docService := services.NewDocumentService(docRepo)
	emailService := services.NewEmailService(cfg)
	newsService := services.NewNewsService(newsRepo, userRepo, emailService, cfg)
	emailTokenService := services.NewEmailTokenService(emailTokenRepo, userRepo)
	emaService := services.NewEmailService(cfg)
	// Хендлеры
	authHandler := handlers.NewAuthHandler(authService, emailService, emailTokenService)
	docHandler := handlers.NewDocumentHandler(docService, authService)
	newsHandler := handlers.NewNewsHandler(newsService)
	emailHandler := handlers.NewEmailHandler(emailTokenService)

	for i := 0; i < 3; i++ {
		go services.StartEmailWorker(emaService)
	}

	// Маршруты
	router := mux.NewRouter()
	routes.InitRoutes(router, authHandler, docHandler, newsHandler, emailHandler)

	return router, nil
}
