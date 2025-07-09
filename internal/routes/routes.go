package routes

import (
	"edutalks/internal/handlers"

	"edutalks/internal/middleware"

	"github.com/gorilla/mux"
)

func InitRoutes(router *mux.Router, authHandler *handlers.AuthHandler, documentHandler *handlers.DocumentHandler) {
	router.HandleFunc("/register", authHandler.Register).Methods("POST")
	router.HandleFunc("/login", authHandler.Login).Methods("POST")
	router.HandleFunc("/refresh", authHandler.Refresh).Methods("POST")
	router.HandleFunc("/logout", authHandler.Logout).Methods("POST")

	//jwt
	protected := router.PathPrefix("/api").Subrouter()
	protected.Use(middleware.JWTAuth)

	fileRoutes := protected.PathPrefix("/files").Subrouter()
	fileRoutes.HandleFunc("/upload", documentHandler.UploadDocument).Methods("POST")

	//only
	admin := protected.PathPrefix("/admin").Subrouter()
	admin.Use(middleware.OnlyRole("admin"))
	admin.HandleFunc("/dashboard", authHandler.AdminOnly).Methods("GET")
	admin.HandleFunc("/users", authHandler.GetUsers).Methods("GET")
	admin.HandleFunc("/users/{id}", authHandler.GetUserByID).Methods("GET")
	admin.HandleFunc("/users/{id}", authHandler.UpdateUser).Methods("PATCH")

	protected.HandleFunc("/profile", authHandler.Protected).Methods("GET")
}
