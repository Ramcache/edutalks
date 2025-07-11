package routes

import (
	"edutalks/internal/handlers"

	"edutalks/internal/middleware"

	"github.com/gorilla/mux"
)

func InitRoutes(router *mux.Router, authHandler *handlers.AuthHandler, documentHandler *handlers.DocumentHandler) {
	// Public auth
	router.HandleFunc("/register", authHandler.Register).Methods("POST")
	router.HandleFunc("/login", authHandler.Login).Methods("POST")
	router.HandleFunc("/refresh", authHandler.Refresh).Methods("POST")
	router.HandleFunc("/logout", authHandler.Logout).Methods("POST")

	// JWT protected
	protected := router.PathPrefix("/api").Subrouter()
	protected.Use(middleware.JWTAuth)
	{
		// Личный кабинет
		protected.HandleFunc("/profile", authHandler.Protected).Methods("GET")

		// Файлы (по подписке)
		fileRoutes := protected.PathPrefix("/files").Subrouter()
		fileRoutes.HandleFunc("", documentHandler.ListPublicDocuments).Methods("GET")
		fileRoutes.HandleFunc("/{id:[0-9]+}", documentHandler.DownloadDocument).Methods("GET")

		// Админка
		admin := protected.PathPrefix("/admin").Subrouter()
		admin.Use(middleware.OnlyRole("admin"))
		admin.HandleFunc("/files/upload", documentHandler.UploadDocument).Methods("POST")
		admin.HandleFunc("/files/{id:[0-9]+}", documentHandler.DeleteDocument).Methods("DELETE")
		admin.HandleFunc("/dashboard", authHandler.AdminOnly).Methods("GET")
		admin.HandleFunc("/users", authHandler.GetUsers).Methods("GET")
		admin.HandleFunc("/users/{id}", authHandler.GetUserByID).Methods("GET")
		admin.HandleFunc("/users/{id}", authHandler.UpdateUser).Methods("PATCH")
	}
}
