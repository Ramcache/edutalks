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
	taxonomyH *handlers.TaxonomyHandler,
) {
	// Логирование всех запросов
	router.Use(middleware.Logging)

	// Корневой /api
	api := router.PathPrefix("/api").Subrouter()

	// ---------- ПУБЛИЧНЫЕ ----------
	api.HandleFunc("/register", authHandler.Register).Methods(http.MethodPost)
	api.HandleFunc("/login", authHandler.Login).Methods(http.MethodPost)
	api.HandleFunc("/refresh", authHandler.Refresh).Methods(http.MethodPost)
	api.HandleFunc("/logout", authHandler.Logout).Methods(http.MethodPost)

	// платежный вебхук (публичная точка приёмки от ЮKassa)
	api.HandleFunc("/payments/webhook", webhookHandler.HandleWebhook).Methods(http.MethodPost)

	// контент, доступный без авторизации
	api.HandleFunc("/news", newsHandler.ListNews).Methods(http.MethodGet)
	api.HandleFunc("/news/{id:[0-9]+}", newsHandler.GetNews).Methods(http.MethodGet)

	// --- ДОБАВЛЕНО: публичные статьи ---
	api.HandleFunc("/articles", articleH.GetAll).Methods(http.MethodGet)
	api.HandleFunc("/articles/{id:[0-9]+}", articleH.GetByID).Methods(http.MethodGet)

	api.HandleFunc("/verify-email", emailHandler.VerifyEmail).Methods(http.MethodGet)
	api.HandleFunc("/resend-verification", authHandler.ResendVerificationEmail).Methods(http.MethodPost)

	// превью документов (метаданные)
	api.HandleFunc("/documents/{id:[0-9]+}/preview", documentHandler.PreviewDocument).Methods(http.MethodGet)
	api.HandleFunc("/documents/preview", documentHandler.PreviewDocuments).Methods(http.MethodGet)

	// публичный таксономический лес
	api.HandleFunc("/taxonomy/tree", taxonomyH.PublicTree).Methods(http.MethodGet)
	api.HandleFunc("/taxonomy/tree/{tab}", taxonomyH.PublicTreeByTab).Methods(http.MethodGet)

	// публичный список файлов (без скачивания)
	api.HandleFunc("/files", documentHandler.ListPublicDocuments).Methods(http.MethodGet)

	// глобальный поиск
	api.HandleFunc("/search", searchHandler.GlobalSearch).Methods(http.MethodGet)

	// ---------- ПРОТЕКТИРОВАННЫЕ (JWT) ----------
	protected := api.PathPrefix("").Subrouter()
	protected.Use(middleware.JWTAuth)

	// профиль, платеж и пр.
	protected.HandleFunc("/pay", paymentHandler.CreatePayment).Methods(http.MethodGet)
	protected.HandleFunc("/profile", authHandler.Protected).Methods(http.MethodGet)
	protected.HandleFunc("/email-subscription", authHandler.EmailSubscribe).Methods(http.MethodPatch)

	// скачивание файла (нужен user из контекста и проверка подписки)
	protected.HandleFunc("/files/{id:[0-9]+}", documentHandler.DownloadDocument).Methods(http.MethodGet)

	// ---------- АДМИН ----------
	admin := protected.PathPrefix("/admin").Subrouter()
	admin.Use(middleware.OnlyRole("admin"))

	// файлы (админ)
	admin.HandleFunc("/files", documentHandler.GetAllDocuments).Methods(http.MethodGet)
	admin.HandleFunc("/files/upload", documentHandler.UploadDocument).Methods(http.MethodPost)
	admin.HandleFunc("/files/{id:[0-9]+}", documentHandler.DeleteDocument).Methods(http.MethodDelete)

	// админ-панель / пользователи
	admin.HandleFunc("/dashboard", authHandler.AdminOnly).Methods(http.MethodGet)
	admin.HandleFunc("/users", authHandler.GetUsers).Methods(http.MethodGet)
	admin.HandleFunc("/users/{id}", authHandler.GetUserByID).Methods(http.MethodGet)
	admin.HandleFunc("/users/{id}", authHandler.UpdateUser).Methods(http.MethodPatch)
	admin.HandleFunc("/users/{id}/subscription", authHandler.SetSubscription).Methods(http.MethodPatch)
	admin.HandleFunc("/users/{id}", authHandler.DeleteUser).Methods(http.MethodDelete)

	// новости (админ)
	admin.HandleFunc("/news", newsHandler.CreateNews).Methods(http.MethodPost)
	admin.HandleFunc("/news/{id:[0-9]+}", newsHandler.UpdateNews).Methods(http.MethodPatch)
	admin.HandleFunc("/news/{id:[0-9]+}", newsHandler.DeleteNews).Methods(http.MethodDelete)
	admin.HandleFunc("/news/upload", newsHandler.UploadNewsImage).Methods(http.MethodPost)

	// рассылка
	admin.HandleFunc("/notify", authHandler.NotifySubscribers).Methods(http.MethodPost)

	// статьи (админ)
	admin.HandleFunc("/articles/preview", articleH.Preview).Methods(http.MethodPost)
	admin.HandleFunc("/articles", articleH.Create).Methods(http.MethodPost)
	admin.HandleFunc("/articles/{id:[0-9]+}", articleH.Update).Methods(http.MethodPatch)
	admin.HandleFunc("/articles/{id:[0-9]+}", articleH.Delete).Methods(http.MethodDelete)
	admin.HandleFunc("/articles/{id:[0-9]+}/publish", articleH.SetPublish).Methods(http.MethodPatch)

	// таксономия (админ)
	admin.HandleFunc("/tabs", taxonomyH.CreateTab).Methods(http.MethodPost)
	admin.HandleFunc("/tabs/{id:[0-9]+}", taxonomyH.UpdateTab).Methods(http.MethodPatch)
	admin.HandleFunc("/tabs/{id:[0-9]+}", taxonomyH.DeleteTab).Methods(http.MethodDelete)
	admin.HandleFunc("/sections", taxonomyH.CreateSection).Methods(http.MethodPost)
	admin.HandleFunc("/sections/{id:[0-9]+}", taxonomyH.UpdateSection).Methods(http.MethodPatch)
	admin.HandleFunc("/sections/{id:[0-9]+}", taxonomyH.DeleteSection).Methods(http.MethodDelete)
}
