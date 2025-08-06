package handlers

import (
	"edutalks/internal/middleware"
	"edutalks/internal/services"
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

// CreatePayment godoc
// @Summary Инициализировать оплату подписки
// @Tags Оплата
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param plan query string true "Тип подписки: monthly, halfyear, yearly"
// @Success 302 {string} string "Redirect to YooKassa"
// @Failure 400 {string} string "Некорректный запрос"
// @Failure 401 {string} string "Неавторизован"
// @Failure 500 {string} string "Ошибка сервера"
// @Router /api/pay [get]
func (h *PaymentHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	plan := r.URL.Query().Get("plan")
	if plan == "" {
		http.Error(w, "missing plan", http.StatusBadRequest)
		return
	}

	// ✅ userID теперь берём из JWT
	userID := r.Context().Value(middleware.ContextUserID).(int)

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
		http.Error(w, "invalid plan", http.StatusBadRequest)
		return
	}

	paymentURL, err := h.YooKassaService.CreatePayment(amount, description, userID)
	if err != nil {
		http.Error(w, "failed to create payment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, paymentURL, http.StatusFound)
}
