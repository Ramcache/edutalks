package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"edutalks/internal/logger"
	"edutalks/internal/middleware"
	"edutalks/internal/models"
	"edutalks/internal/services"
	helpers "edutalks/internal/utils/helpers"

	"go.uber.org/zap"
)

// userReader — рядом с PasswordHandler
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
	log := logger.WithCtx(r.Context())

	var req forgotReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Email) == "" {
		log.Warn("Невалидный payload в Forgot")
		helpers.Error(w, http.StatusBadRequest, "invalid payload")
		return
	}

	// Не раскрываем, существует ли email — всегда возвращаем 200
	if err := h.svc.RequestReset(r.Context(), req.Email); err != nil {
		// Ошибку логируем, но клиенту отвечаем одинаково
		log.Error("Сбой при запросе восстановления пароля", zap.String("email_masked", maskEmail(req.Email)), zap.Error(err))
	} else {
		log.Info("Запрошено восстановление пароля", zap.String("email_masked", maskEmail(req.Email)))
	}

	helpers.JSON(w, http.StatusOK, map[string]any{"message": "If the email exists, a reset link has been sent."})
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
	log := logger.WithCtx(r.Context())

	var req resetReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Token) == "" || strings.TrimSpace(req.NewPassword) == "" {
		log.Warn("Невалидный payload в Reset")
		helpers.Error(w, http.StatusBadRequest, "invalid payload")
		return
	}

	if err := h.svc.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		// Ошибки токена/валидации — это 400
		log.Warn("Не удалось сбросить пароль по токену", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "invalid token or password")
		return
	}

	log.Info("Пароль успешно сброшен")
	helpers.JSON(w, http.StatusOK, map[string]string{"message": "Password has been reset."})
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
	log := logger.WithCtx(r.Context())

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok || userID == 0 {
		log.Warn("Нет доступа для Change: отсутствует user_id")
		helpers.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req changeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.OldPassword) == "" || strings.TrimSpace(req.NewPassword) == "" {
		log.Warn("Невалидный payload в Change", zap.Int("user_id", userID))
		helpers.Error(w, http.StatusBadRequest, "invalid payload")
		return
	}

	u, err := h.userRepo.GetUserByID(r.Context(), userID)
	if err != nil {
		log.Warn("Пользователь не найден при смене пароля", zap.Int("user_id", userID))
		helpers.Error(w, http.StatusUnauthorized, "user not found")
		return
	}

	if _, err := h.svc.ChangePassword(r.Context(), int64(userID), req.OldPassword, req.NewPassword, u.PasswordHash); err != nil {
		// Ошибки валидации/несовпадения старого пароля — 400
		log.Warn("Не удалось сменить пароль", zap.Int("user_id", userID), zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	log.Info("Пароль изменён", zap.Int("user_id", userID))
	helpers.JSON(w, http.StatusOK, map[string]string{"message": "Password changed."})
}
