package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"edutalks/internal/logger"
	"edutalks/internal/services"
	helpers "edutalks/internal/utils/helpers"

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
	log := logger.WithCtx(r.Context())
	start := time.Now()

	// ограничим размер тела, чтобы не словить OOM
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB

	var webhook PaymentWebhook
	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		log.Warn("webhook: не удалось распарсить JSON", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "invalid json")
		return
	}

	userIDStr := webhook.Object.Metadata.UserID
	plan := webhook.Object.Metadata.Plan
	if userIDStr == "" || plan == "" {
		log.Warn("webhook: отсутствуют обязательные поля metadata",
			zap.String("user_id", userIDStr), zap.String("plan", plan))
		helpers.Error(w, http.StatusBadRequest, "missing metadata.user_id or metadata.plan")
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil || userID <= 0 {
		log.Warn("webhook: некорректный user_id", zap.String("raw_user_id", userIDStr), zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	log.Info("webhook: получено событие",
		zap.String("event", webhook.Event),
		zap.String("payment_id", webhook.Object.ID),
		zap.String("status", webhook.Object.Status),
		zap.Int("user_id", userID),
		zap.String("plan", plan),
	)

	// Нормируем длительности как в остальном коде (halfyear = 182d)
	planDurations := map[string]time.Duration{
		"monthly":  30 * 24 * time.Hour,
		"halfyear": 182 * 24 * time.Hour,
		"yearly":   365 * 24 * time.Hour,
	}
	duration, ok := planDurations[plan]
	if !ok {
		log.Warn("webhook: неизвестный план", zap.String("plan", plan))
		helpers.Error(w, http.StatusBadRequest, "invalid plan")
		return
	}

	if webhook.Event == "payment.succeeded" && webhook.Object.Status == "succeeded" {
		if err := h.UserService.SetSubscriptionWithExpiry(r.Context(), userID, duration); err != nil {
			log.Error("webhook: не удалось активировать подписку",
				zap.Int("user_id", userID),
				zap.String("plan", plan),
				zap.Duration("duration", duration),
				zap.Error(err),
			)
			helpers.Error(w, http.StatusInternalServerError, "internal error")
			return
		}
		log.Info("webhook: подписка активирована",
			zap.Int("user_id", userID),
			zap.String("plan", plan),
			zap.Duration("duration", duration),
		)
	} else {
		// Идемпотентно подтверждаем другие события
		log.Info("webhook: событие проигнорировано (не succeeded)",
			zap.String("event", webhook.Event),
			zap.String("status", webhook.Object.Status))
	}

	log.Info("webhook: обработано", zap.Duration("elapsed", time.Since(start)))
	helpers.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
