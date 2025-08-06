package handlers

import (
	"net/http"
	"strconv"

	"edutalks/internal/services"
)

type PaymentHandler struct {
	YooKassaService *services.YooKassaService
}

func NewPaymentHandler(yoo *services.YooKassaService) *PaymentHandler {
	return &PaymentHandler{
		YooKassaService: yoo,
	}
}

func (h *PaymentHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	plan := r.URL.Query().Get("plan")
	userIDStr := r.URL.Query().Get("user_id") // можно получать из токена, но сейчас берём из query

	if plan == "" || userIDStr == "" {
		http.Error(w, "missing plan or user_id", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
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
		http.Error(w, "invalid plan", http.StatusBadRequest)
		return
	}

	paymentURL, err := h.YooKassaService.CreatePayment(amount, description, userID)
	if err != nil {
		http.Error(w, "failed to create payment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Перенаправление пользователя на ЮKassa
	http.Redirect(w, r, paymentURL, http.StatusFound)
}
