package routes

import (
	"edutalks/internal/handlers"
	"edutalks/internal/middleware"
	"github.com/gorilla/mux"
	"net/http"
)

func InitRoutes(
	router *mux.Router,
	authHandler *handlers.AuthHandler,
	documentHandler *handlers.DocumentHandler,
	newsHandler *handlers.NewsHandler,
	emailHandler *handlers.EmailHandler,
	searchHandler *handlers.SearchHandler,
	paymentHandler *handlers.PaymentHandler,
	webhookHandler *handlers.WebhookHandler,
	articleH *handlers.ArticleHandler,
) {
	router.Use(middleware.Logging)

	// --- Глобальный обработчик preflight (на всякий случай) ---
	router.Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	api := router.PathPrefix("/api").Subrouter()

	// Тоже полезно иметь preflight на уровне подсеток
	api.Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// --- Публичные маршруты ---
	api.HandleFunc("/register", authHandler.Register).Methods("POST", http.MethodOptions)
	api.HandleFunc("/login", authHandler.Login).Methods("POST", http.MethodOptions)
	api.HandleFunc("/refresh", authHandler.Refresh).Methods("POST", http.MethodOptions)
	api.HandleFunc("/logout", authHandler.Logout).Methods("POST", http.MethodOptions)

	api.HandleFunc("/payments/webhook", webhookHandler.HandleWebhook).Methods("POST", http.MethodOptions)

	api.HandleFunc("/news", newsHandler.ListNews).Methods("GET", http.MethodOptions)
	api.HandleFunc("/news/{id:[0-9]+}", newsHandler.GetNews).Methods("GET", http.MethodOptions)

	api.HandleFunc("/verify-email", emailHandler.VerifyEmail).Methods("GET", http.MethodOptions)
	api.HandleFunc("/resend-verification", authHandler.ResendVerificationEmail).Methods("POST", http.MethodOptions)
	api.HandleFunc("/documents/{id:[0-9]+}/preview", documentHandler.PreviewDocument).Methods("GET", http.MethodOptions)
	api.HandleFunc("/documents/preview", documentHandler.PreviewDocuments).Methods("GET", http.MethodOptions)

	api.HandleFunc("/search", searchHandler.GlobalSearch).Methods("GET", http.MethodOptions)
	api.HandleFunc("/articles/{id:[0-9]+}", articleH.GetByID).Methods("GET", http.MethodOptions)
	api.HandleFunc("/articles", articleH.GetAll).Methods("GET", http.MethodOptions)

	// --- Защищённые JWT ---
	protected := api.PathPrefix("").Subrouter()
	protected.Use(middleware.JWTAuth)
	protected.Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	protected.HandleFunc("/pay", paymentHandler.CreatePayment).Methods("GET", http.MethodOptions)
	protected.HandleFunc("/profile", authHandler.Protected).Methods("GET", http.MethodOptions)
	protected.HandleFunc("/email-subscription", authHandler.EmailSubscribe).Methods("PATCH", http.MethodOptions)

	fileRoutes := protected.PathPrefix("/files").Subrouter()
	fileRoutes.Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	fileRoutes.HandleFunc("", documentHandler.ListPublicDocuments).Methods("GET", http.MethodOptions)
	fileRoutes.HandleFunc("/{id:[0-9]+}", documentHandler.DownloadDocument).Methods("GET", http.MethodOptions)

	// --- Админ (требует роль admin) ---
	admin := protected.PathPrefix("/admin").Subrouter()
	admin.Use(middleware.OnlyRole("admin"))
	admin.Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	admin.HandleFunc("/files", documentHandler.GetAllDocuments).Methods("GET", http.MethodOptions)
	admin.HandleFunc("/files/upload", documentHandler.UploadDocument).Methods("POST", http.MethodOptions)
	admin.HandleFunc("/files/{id:[0-9]+}", documentHandler.DeleteDocument).Methods("DELETE", http.MethodOptions)

	admin.HandleFunc("/dashboard", authHandler.AdminOnly).Methods("GET", http.MethodOptions)
	admin.HandleFunc("/users", authHandler.GetUsers).Methods("GET", http.MethodOptions)
	admin.HandleFunc("/users/{id}", authHandler.GetUserByID).Methods("GET", http.MethodOptions)
	admin.HandleFunc("/users/{id}", authHandler.UpdateUser).Methods("PATCH", http.MethodOptions)
	admin.HandleFunc("/users/{id}/subscription", authHandler.SetSubscription).Methods("PATCH", http.MethodOptions)
	admin.HandleFunc("/users/{id}", authHandler.DeleteUser).Methods("DELETE", http.MethodOptions)

	admin.HandleFunc("/news", newsHandler.CreateNews).Methods("POST", http.MethodOptions)
	admin.HandleFunc("/news/{id:[0-9]+}", newsHandler.UpdateNews).Methods("PATCH", http.MethodOptions)
	admin.HandleFunc("/news/{id:[0-9]+}", newsHandler.DeleteNews).Methods("DELETE", http.MethodOptions)
	admin.HandleFunc("/news/upload", newsHandler.UploadNewsImage).Methods("POST", http.MethodOptions)

	admin.HandleFunc("/notify", authHandler.NotifySubscribers).Methods("POST", http.MethodOptions)

	admin.HandleFunc("/articles/preview", articleH.Preview).Methods("POST", http.MethodOptions)
	admin.HandleFunc("/articles", articleH.Create).Methods("POST", http.MethodOptions)
	admin.HandleFunc("/articles/{id:[0-9]+}", articleH.Update).Methods("PATCH", http.MethodOptions)
	admin.HandleFunc("/articles/{id:[0-9]+}", articleH.Delete).Methods("DELETE", http.MethodOptions)
	admin.HandleFunc("/articles/{id:[0-9]+}/publish", articleH.SetPublish).Methods(http.MethodPatch, http.MethodOptions)
}
