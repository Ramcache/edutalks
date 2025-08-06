package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"edutalks/internal/logger"
	"edutalks/internal/services"
	"go.uber.org/zap"
)

type WebhookHandler struct {
	UserService *services.AuthService
}

func NewWebhookHandler(userService *services.AuthService) *WebhookHandler {
	return &WebhookHandler{
		UserService: userService,
	}
}

type PaymentWebhook struct {
	Event  string `json:"event"`
	Object struct {
		ID       string `json:"id"`
		Status   string `json:"status"`
		Metadata struct {
			UserID int `json:"user_id"`
		} `json:"metadata"`
	} `json:"object"`
}

// HandleWebhook godoc
// @Summary Обработка webhook от YooKassa
// @Tags Оплата
// @Accept json
// @Produce json
// @Success 200 {string} string "OK"
// @Failure 400 {string} string "Ошибка парсинга запроса"
// @Failure 500 {string} string "Ошибка обновления подписки"
// @Router /api/payments/webhook [post]
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Log.Error("Ошибка чтения тела webhook", zap.Error(err))
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var webhook PaymentWebhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		logger.Log.Error("Ошибка парсинга webhook", zap.Error(err))
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	logger.Log.Info("Webhook получен", zap.String("event", webhook.Event), zap.Int("user_id", webhook.Object.Metadata.UserID))

	if webhook.Event == "payment.succeeded" && webhook.Object.Status == "succeeded" {
		userID := webhook.Object.Metadata.UserID
		if err := h.UserService.SetSubscriptionTrue(userID); err != nil {
			logger.Log.Error("Не удалось обновить подписку", zap.Int("user_id", userID), zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		logger.Log.Info("Подписка активирована", zap.Int("user_id", userID))
	}

	w.WriteHeader(http.StatusOK)
}
