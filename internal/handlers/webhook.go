package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

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
			UserID string `json:"user_id"`
			Plan   string `json:"plan"`
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

	userIDStr := webhook.Object.Metadata.UserID
	plan := webhook.Object.Metadata.Plan

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		logger.Log.Error("Некорректный user_id в webhook", zap.String("raw_user_id", userIDStr), zap.Error(err))
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	logger.Log.Info("Webhook получен",
		zap.String("event", webhook.Event),
		zap.Int("user_id", userID),
		zap.String("plan", plan),
	)

	if webhook.Event == "payment.succeeded" && webhook.Object.Status == "succeeded" {
		var duration time.Duration
		switch plan {
		case "monthly":
			duration = 30 * 24 * time.Hour
		case "halfyear":
			duration = 180 * 24 * time.Hour
		case "yearly":
			duration = 365 * 24 * time.Hour
		default:
			logger.Log.Warn("Неизвестный тип подписки", zap.String("plan", plan))
			http.Error(w, "invalid plan", http.StatusBadRequest)
			return
		}

		err := h.UserService.SetSubscriptionWithExpiry(r.Context(), userID, duration)
		if err != nil {
			logger.Log.Error("Не удалось установить подписку с истечением", zap.Int("user_id", userID), zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		logger.Log.Info("Подписка успешно активирована", zap.Int("user_id", userID), zap.String("plan", plan))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
