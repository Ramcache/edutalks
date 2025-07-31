package routes

import (
	"edutalks/internal/handlers"
	"edutalks/internal/middleware"

	"github.com/gorilla/mux"
)

func InitRoutes(router *mux.Router, authHandler *handlers.AuthHandler, documentHandler *handlers.DocumentHandler, newsHandler *handlers.NewsHandler, emailHandler *handlers.EmailHandler) {
	router.Use(middleware.Logging)

	// --- Весь API теперь начинается с /api ---
	api := router.PathPrefix("/api").Subrouter()

	// --- Публичные маршруты ---
	api.HandleFunc("/register", authHandler.Register).Methods("POST")
	api.HandleFunc("/login", authHandler.Login).Methods("POST")
	api.HandleFunc("/refresh", authHandler.Refresh).Methods("POST")
	api.HandleFunc("/logout", authHandler.Logout).Methods("POST")

	api.HandleFunc("/news", newsHandler.ListNews).Methods("GET")
	api.HandleFunc("/news/{id:[0-9]+}", newsHandler.GetNews).Methods("GET")

	api.HandleFunc("/verify-email", emailHandler.VerifyEmail).Methods("GET")
	api.HandleFunc("/resend-verification", authHandler.ResendVerificationEmail).Methods("POST")

	api.HandleFunc("/documents/{id:[0-9]+}/preview", documentHandler.PreviewDocument).Methods("GET")

	// --- Защищённые JWT ---
	protected := api.PathPrefix("").Subrouter()
	protected.Use(middleware.JWTAuth)

	// Личный кабинет
	protected.HandleFunc("/profile", authHandler.Protected).Methods("GET")
	protected.HandleFunc("/email-subscription", authHandler.EmailSubscribe).Methods("PATCH")

	// Файлы (по подписке)
	fileRoutes := protected.PathPrefix("/files").Subrouter()
	fileRoutes.HandleFunc("", documentHandler.ListPublicDocuments).Methods("GET")
	fileRoutes.HandleFunc("/{id:[0-9]+}", documentHandler.DownloadDocument).Methods("GET")

	// Админка
	admin := protected.PathPrefix("/admin").Subrouter()
	admin.Use(middleware.OnlyRole("admin"))
	admin.HandleFunc("/files", documentHandler.GetAllDocuments).Methods("GET")
	admin.HandleFunc("/files/upload", documentHandler.UploadDocument).Methods("POST")
	admin.HandleFunc("/files/{id:[0-9]+}", documentHandler.DeleteDocument).Methods("DELETE")

	admin.HandleFunc("/dashboard", authHandler.AdminOnly).Methods("GET")
	admin.HandleFunc("/users", authHandler.GetUsers).Methods("GET")
	admin.HandleFunc("/users/{id}", authHandler.GetUserByID).Methods("GET")
	admin.HandleFunc("/users/{id}", authHandler.UpdateUser).Methods("PATCH")
	admin.HandleFunc("/users/{id}/subscription", authHandler.SetSubscription).Methods("PATCH")

	admin.HandleFunc("/news", newsHandler.CreateNews).Methods("POST")
	admin.HandleFunc("/news/{id:[0-9]+}", newsHandler.UpdateNews).Methods("PATCH")
	admin.HandleFunc("/news/{id:[0-9]+}", newsHandler.DeleteNews).Methods("DELETE")
	admin.HandleFunc("/notify", authHandler.NotifySubscribers).Methods("POST")
}
