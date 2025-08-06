package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type YooKassaService struct {
	ShopID     string
	SecretKey  string
	ReturnURL  string
	HTTPClient *http.Client
}

func NewYooKassaService(shopID, secretKey, returnURL string) *YooKassaService {
	return &YooKassaService{
		ShopID:     shopID,
		SecretKey:  secretKey,
		ReturnURL:  returnURL,
		HTTPClient: &http.Client{},
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
	Amount       Amount         `json:"amount"`
	Confirmation Confirmation   `json:"confirmation"`
	Capture      bool           `json:"capture"`
	Description  string         `json:"description"`
	Metadata     map[string]any `json:"metadata"`
}

type CreatePaymentResponse struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Confirmation struct {
		Type            string `json:"type"`
		ConfirmationURL string `json:"confirmation_url"`
	} `json:"confirmation"`
}

func (s *YooKassaService) CreatePayment(value float64, description string, userID int) (string, error) {
	reqBody := CreatePaymentRequest{
		Amount: Amount{
			Value:    fmt.Sprintf("%.2f", value),
			Currency: "RUB",
		},
		Confirmation: Confirmation{
			Type:      "redirect",
			ReturnURL: s.ReturnURL,
		},
		Capture:     true,
		Description: description,
		Metadata: map[string]any{
			"user_id": userID,
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.yookassa.ru/v3/payments", bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotence-Key", fmt.Sprintf("payment-%d", time.Now().UnixNano()))
	auth := base64.StdEncoding.EncodeToString([]byte(s.ShopID + ":" + s.SecretKey))
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res CreatePaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	return res.Confirmation.ConfirmationURL, nil
}
