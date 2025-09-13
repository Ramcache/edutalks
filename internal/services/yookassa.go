package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"edutalks/internal/logger"
	"go.uber.org/zap"

	"github.com/google/uuid"
)

type YooKassaService struct {
	ShopID     string
	SecretKey  string
	ReturnURL  string
	HTTPClient *http.Client
}

func NewYooKassaService(shopID, secretKey, returnURL string) *YooKassaService {
	client := &http.Client{Timeout: 15 * time.Second}
	return &YooKassaService{
		ShopID:     shopID,
		SecretKey:  secretKey,
		ReturnURL:  returnURL,
		HTTPClient: client,
	}
}

type Amount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type Confirmation struct {
	Type      string `json:"type"`
	ReturnURL string `json:"return_url"`
}

type CreatePaymentRequest struct {
	Amount       Amount            `json:"amount"`
	Confirmation Confirmation      `json:"confirmation"`
	Capture      bool              `json:"capture"`
	Description  string            `json:"description"`
	Metadata     map[string]string `json:"metadata"`
}

type CreatePaymentResponse struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Confirmation struct {
		Type            string `json:"type"`
		ConfirmationURL string `json:"confirmation_url"`
	} `json:"confirmation"`
}

type ykError struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Parameter   string `json:"parameter"`
}

// CreatePayment — создаёт платёж и возвращает URL для подтверждения.
// value — рубли (например 1250.00), plan — один из: monthly | halfyear | yearly.
func (s *YooKassaService) CreatePayment(ctx context.Context, value float64, description string, userID int, plan string) (string, error) {
	if value <= 0 {
		return "", fmt.Errorf("amount must be positive")
	}
	switch plan {
	case "monthly", "halfyear", "yearly":
	default:
		return "", fmt.Errorf("invalid plan")
	}

	reqBody := CreatePaymentRequest{
		Amount: Amount{
			// ЮKassa требует 2 знака после запятой
			Value:    fmt.Sprintf("%.2f", value),
			Currency: "RUB",
		},
		Confirmation: Confirmation{
			Type:      "redirect",
			ReturnURL: s.ReturnURL,
		},
		Capture:     true,
		Description: description,
		Metadata: map[string]string{
			"user_id": fmt.Sprintf("%d", userID),
			"plan":    plan,
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.yookassa.ru/v3/payments", bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Idempotence-Key", "payment-"+uuid.NewString())
	// базовая авторизация: shopId:secretKey
	req.SetBasicAuth(s.ShopID, s.SecretKey)

	logger.Log.Info("YooKassa: создаём платёж",
		zap.Int("user_id", userID),
		zap.String("plan", plan),
		zap.String("amount", reqBody.Amount.Value),
	)

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Успех — любой 2xx (у ЮKassa обычно 200/201)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var res CreatePaymentResponse
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return "", err
		}
		logger.Log.Info("YooKassa: платёж создан",
			zap.String("payment_id", res.ID),
			zap.String("status", res.Status),
		)
		return res.Confirmation.ConfirmationURL, nil
	}

	// Ошибка: попробуем разобрать тело от ЮKassa
	var ek ykError
	_ = json.NewDecoder(resp.Body).Decode(&ek)
	if ek.Code != "" || ek.Description != "" {
		logger.Log.Warn("YooKassa: ошибка создания платежа",
			zap.Int("http_status", resp.StatusCode),
			zap.String("code", ek.Code),
			zap.String("desc", ek.Description),
			zap.String("param", ek.Parameter),
		)
		return "", fmt.Errorf("yookassa error: %s (%s)", ek.Description, ek.Code)
	}

	logger.Log.Warn("YooKassa: неизвестная ошибка создания платежа",
		zap.Int("http_status", resp.StatusCode),
	)
	return "", fmt.Errorf("yookassa http status: %d", resp.StatusCode)
}
