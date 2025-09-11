package handlers

import (
	"edutalks/internal/logger"
	"edutalks/internal/services"
	helpers "edutalks/internal/utils/helpers"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type EmailHandler struct {
	emailTokenService *services.EmailTokenService
}

func NewEmailHandler(emailTokenService *services.EmailTokenService) *EmailHandler {
	return &EmailHandler{emailTokenService: emailTokenService}
}

// VerifyEmail godoc
// @Summary Подтвердить email
// @Description Подтверждает email по токену из письма
// @Tags email
// @Accept json
// @Produce json
// @Param token query string true "Токен подтверждения"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /api/verify-email [get]
func (h *EmailHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		helpers.JSON(w, http.StatusBadRequest, map[string]string{
			"message": "Токен отсутствует",
		})
		return
	}

	err := h.emailTokenService.ConfirmToken(r.Context(), token)
	if err != nil {
		logger.Log.Warn("Ошибка подтверждения email", zap.Error(err))
		var msg string
		switch err {
		case services.ErrTokenInvalid:
			msg = "Неверный или уже использованный токен."
		case services.ErrTokenExpired:
			msg = "Срок действия токена истёк."
		default:
			msg = "Внутренняя ошибка сервиса."
		}
		helpers.JSON(w, http.StatusBadRequest, map[string]string{
			"message": msg,
		})
		return
	}

	http.Redirect(w, r, "https://edutalks.ru/verify-email?status=success", http.StatusFound)

}

// ResendVerificationEmail godoc
// @Summary Повторная отправка письма для подтверждения e-mail
// @Tags email
// @Accept json
// @Produce json
// @Param input body map[string]string true "Email пользователя"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/resend-verification [post]
func (h *AuthHandler) ResendVerificationEmail(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Email string `json:"email"`
	}

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		helpers.Error(w, http.StatusBadRequest, "Неверный формат запроса или пустой email")
		return
	}

	user, err := h.authService.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		logger.Log.Warn("Пользователь не найден при ResendVerificationEmail", zap.String("email", req.Email))
		helpers.Error(w, http.StatusNotFound, "Пользователь не найден")
		return
	}

	// Проверяем лимит по последнему токену
	lastToken, err := h.emailTokenService.GetLastTokenByUserID(r.Context(), user.ID)
	if err == nil && time.Since(lastToken.CreatedAt) < 5*time.Minute {
		remaining := int((5*time.Minute - time.Since(lastToken.CreatedAt)).Seconds())
		helpers.Error(w, http.StatusTooManyRequests,
			fmt.Sprintf("Вы можете повторно запросить письмо через %d секунд", remaining))
		return
	}

	// Создаём новый токен
	emailToken, err := h.emailTokenService.GenerateToken(r.Context(), user.ID)
	if err != nil {
		logger.Log.Error("Ошибка генерации токена при ResendVerificationEmail", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка генерации токена")
		return
	}
	logger.Log.Info("DEBUG: LastToken",
		zap.Time("created_at", lastToken.CreatedAt),
		zap.Duration("since", time.Since(lastToken.CreatedAt)))

	// Отправляем письмо
	if err := h.SendVerificationEmail(r.Context(), user, emailToken.Token); err != nil {
		logger.Log.Error("Ошибка при отправке письма в ResendVerificationEmail", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка при отправке письма")
		return
	}

	helpers.JSON(w, http.StatusOK, map[string]string{
		"message": "Письмо с подтверждением отправлено повторно",
	})

}
