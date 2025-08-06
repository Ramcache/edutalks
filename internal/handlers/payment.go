package handlers

import (
	"edutalks/internal/middleware"
	"edutalks/internal/services"
	"edutalks/internal/utils/helpers"
	"net/http"
)

type PaymentHandler struct {
	YooKassaService *services.YooKassaService
}

func NewPaymentHandler(yoo *services.YooKassaService) *PaymentHandler {
	return &PaymentHandler{
		YooKassaService: yoo,
	}
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
// @Failure 500 {object} helpers.Response
// @Router /api/pay [get]
func (h *PaymentHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	plan := r.URL.Query().Get("plan")
	if plan == "" {
		helpers.Error(w, http.StatusBadRequest, "missing plan")
		return
	}

	userID, ok := r.Context().Value(middleware.ContextUserID).(int)
	if !ok {
		helpers.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var amount float64
	var description string

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
		helpers.Error(w, http.StatusBadRequest, "invalid plan")
		return
	}

	paymentURL, err := h.YooKassaService.CreatePayment(amount, description, userID)
	if err != nil {
		helpers.Error(w, http.StatusInternalServerError, "failed to create payment: "+err.Error())
		return
	}

	helpers.JSON(w, http.StatusOK, PaymentResult{ConfirmationURL: paymentURL})
}
