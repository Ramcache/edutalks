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

	// Сервисы
	authService := services.NewAuthService(userRepo)
	docService := services.NewDocumentService(docRepo)
	emailService := services.NewEmailService(cfg)
	newsService := services.NewNewsService(newsRepo, userRepo, emailService)

	// Хендлеры
	authHandler := handlers.NewAuthHandler(authService)
	docHandler := handlers.NewDocumentHandler(docService, authService)
	newsHandler := handlers.NewNewsHandler(newsService)

	// Маршруты
	router := mux.NewRouter()
	routes.InitRoutes(router, authHandler, docHandler, newsHandler)

	return router, nil
}
