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

	userRepo := repository.NewUserRepository(conn)
	authService := services.NewAuthService(userRepo)
	authHandler := handlers.NewAuthHandler(authService)

	docRepo := repository.NewDocumentRepository(conn)
	docService := services.NewDocumentService(docRepo)
	docHandler := handlers.NewDocumentHandler(docService, authService)

	router := mux.NewRouter()
	routes.InitRoutes(router, authHandler, docHandler)

	return router, nil
}
