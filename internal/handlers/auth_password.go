package handlers

import (
	"context"
	"edutalks/internal/logger"
	"edutalks/internal/middleware"
	"edutalks/internal/models"
	"edutalks/internal/services"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// рядом с PasswordHandler
type userReader interface {
	GetUserByID(ctx context.Context, id int) (*models.User, error)
}

type PasswordHandler struct {
	svc      *services.PasswordService
	userRepo userReader
}

func NewPasswordHandler(svc *services.PasswordService, userRepo userReader) *PasswordHandler {
	return &PasswordHandler{svc: svc, userRepo: userRepo}
}

type forgotReq struct {
	Email string `json:"email"`
}

// Forgot godoc
// @Summary Запрос восстановления пароля
// @Description Отправляет письмо со ссылкой для сброса пароля. Ответ всегда одинаковый, даже если e-mail не найден.
// @Tags password
// @Accept json
// @Produce json
// @Param input body forgotReq true "Email пользователя"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /api/password/forgot [post]
func (h *PasswordHandler) Forgot(w http.ResponseWriter, r *http.Request) {
	var req forgotReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if err := h.svc.RequestReset(r.Context(), req.Email); err != nil {
		logger.Log.Error("password forgot failed", zap.Error(err))
	}
	// одинаковый ответ всегда
	writeJSON(w, http.StatusOK, map[string]any{"message": "If the email exists, a reset link has been sent."})
}

type resetReq struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// Reset godoc
// @Summary Сброс пароля по токену
// @Description Устанавливает новый пароль по токену из письма.
// @Tags password
// @Accept json
// @Produce json
// @Param input body resetReq true "Токен и новый пароль"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /api/password/reset [post]
func (h *PasswordHandler) Reset(w http.ResponseWriter, r *http.Request) {
	var req resetReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" || req.NewPassword == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if err := h.svc.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		http.Error(w, "invalid token or password", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Password has been reset."})
}

type changeReq struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// Change godoc
// @Summary Смена пароля (авторизованный пользователь)
// @Description Смена пароля по старому паролю. Требуется JWT-токен.
// @Tags password
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param input body changeReq true "Старый и новый пароль"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/password/change [post]
func (h *PasswordHandler) Change(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req changeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OldPassword == "" || req.NewPassword == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	// достаём текущий hash пользователя
	u, err := h.userRepo.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}
	newHash, err := h.svc.ChangePassword(r.Context(), int64(userID), req.OldPassword, req.NewPassword, u.PasswordHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// (опционально) инвалидация refresh-токенов и т.д.
	_ = newHash

	writeJSON(w, http.StatusOK, map[string]string{"message": "Password changed."})
}

// локальный helper
func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}
