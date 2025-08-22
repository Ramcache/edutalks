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

	api := router.PathPrefix("/api").Subrouter()

	// --- Публичные маршруты ---
	api.HandleFunc("/register", authHandler.Register).Methods("POST")
	api.HandleFunc("/login", authHandler.Login).Methods("POST")
	api.HandleFunc("/refresh", authHandler.Refresh).Methods("POST")
	api.HandleFunc("/logout", authHandler.Logout).Methods("POST")

	api.HandleFunc("/payments/webhook", webhookHandler.HandleWebhook).Methods("POST")

	api.HandleFunc("/news", newsHandler.ListNews).Methods("GET")
	api.HandleFunc("/news/{id:[0-9]+}", newsHandler.GetNews).Methods("GET")

	api.HandleFunc("/verify-email", emailHandler.VerifyEmail).Methods("GET")
	api.HandleFunc("/resend-verification", authHandler.ResendVerificationEmail).Methods("POST")
	api.HandleFunc("/documents/{id:[0-9]+}/preview", documentHandler.PreviewDocument).Methods("GET")
	api.HandleFunc("/documents/preview", documentHandler.PreviewDocuments).Methods("GET")

	api.HandleFunc("/search", searchHandler.GlobalSearch).Methods("GET")
	api.HandleFunc("/articles/{id:[0-9]+}", articleH.GetByID).Methods("GET")

	api.HandleFunc("/articles", articleH.GetAll).Methods("GET")

	// --- Защищённые JWT ---
	protected := api.PathPrefix("").Subrouter()
	protected.Use(middleware.JWTAuth)

	protected.HandleFunc("/pay", paymentHandler.CreatePayment).Methods("GET")

	protected.HandleFunc("/profile", authHandler.Protected).Methods("GET")
	protected.HandleFunc("/email-subscription", authHandler.EmailSubscribe).Methods("PATCH")

	fileRoutes := protected.PathPrefix("/files").Subrouter()
	fileRoutes.HandleFunc("", documentHandler.ListPublicDocuments).Methods("GET")
	fileRoutes.HandleFunc("/{id:[0-9]+}", documentHandler.DownloadDocument).Methods("GET")

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
	admin.HandleFunc("/users/{id}", authHandler.DeleteUser).Methods("DELETE")
	admin.HandleFunc("/articles/preview", articleH.Preview).Methods("POST")
	admin.HandleFunc("/articles", articleH.Create).Methods("POST")
	admin.HandleFunc("/articles/{id:[0-9]+}", articleH.Update).Methods("PATCH")
	admin.HandleFunc("/articles/{id:[0-9]+}", articleH.Delete).Methods("DELETE")
	admin.HandleFunc("/articles/{id:[0-9]+}/publish", articleH.SetPublish).Methods(http.MethodPatch, http.MethodOptions)

}
