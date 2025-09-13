package handlers

import (
	"net/http"

	"edutalks/internal/logger"
	"edutalks/internal/middleware"
	"edutalks/internal/services"
	"edutalks/internal/utils/helpers"

	"go.uber.org/zap"
)

type PaymentHandler struct {
	YooKassaService *services.YooKassaService
}

func NewPaymentHandler(yoo *services.YooKassaService) *PaymentHandler {
	return &PaymentHandler{YooKassaService: yoo}
}

type PaymentResult struct {
	ConfirmationURL string `json:"confirmation_url"`
}

// CreatePayment godoc
// @Summary Инициализировать оплату подписки
// @Tags Оплата
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param plan query string true "Тип подписки: monthly, halfyear, yearly"
// @Success 200 {object} helpers.Response{data=handlers.PaymentResult}
// @Failure 400 {object} helpers.Response
// @Failure 401 {object} helpers.Response
// @Router /api/pay [get]
func (h *PaymentHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	plan := r.URL.Query().Get("plan")
	if plan == "" {
		log.Warn("create payment: отсутствует параметр plan")
		helpers.Error(w, http.StatusBadRequest, "missing plan")
		return
	}

	userID, ok := r.Context().Value(middleware.ContextUserID).(int)
	if !ok || userID == 0 {
		log.Warn("create payment: отсутствует user_id в контексте")
		helpers.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var (
		amount      float64
		description string
	)

	switch plan {
	case "monthly":
		amount = 1250
		description = "Месячная подписка"
	case "halfyear":
		amount = 7500
		description = "Подписка на 6 месяцев"
	case "yearly":
		amount = 15000
		description = "Годовая подписка"
	default:
		log.Warn("create payment: неверный план", zap.String("plan", plan))
		helpers.Error(w, http.StatusBadRequest, "invalid plan")
		return
	}

	log.Info("create payment: параметры",
		zap.Int("user_id", userID),
		zap.String("plan", plan),
		zap.Float64("amount", amount),
		zap.String("description", description),
	)

	paymentURL, err := h.YooKassaService.CreatePayment(r.Context(), amount, description, userID, plan)
	if err != nil {
		log.Error("create payment: ошибка сервиса YooKassa", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "failed to create payment: "+err.Error())
		return
	}

	log.Info("create payment: ссылка получена", zap.String("confirmation_url", paymentURL))
	helpers.JSON(w, http.StatusOK, PaymentResult{ConfirmationURL: paymentURL})
}
