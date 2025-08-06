package handlers

import (
	"encoding/json"
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
	var webhook PaymentWebhook
	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		logger.Log.Error("Ошибка парсинга webhook", zap.Error(err))
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	userID := webhook.Object.Metadata.UserID
	if userID == 0 {
		logger.Log.Error("user_id отсутствует в webhook")
		http.Error(w, "missing user_id", http.StatusBadRequest)
		return
	}

	logger.Log.Info("Webhook получен", zap.String("event", webhook.Event), zap.Int("user_id", userID))

	if webhook.Event == "payment.succeeded" && webhook.Object.Status == "succeeded" {
		if err := h.UserService.SetSubscriptionTrue(userID); err != nil {
			logger.Log.Error("Не удалось обновить подписку", zap.Int("user_id", userID), zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		logger.Log.Info("Подписка активирована", zap.Int("user_id", userID))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
