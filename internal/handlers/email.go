package handlers

import (
	"edutalks/internal/config"
	"edutalks/internal/logger"
	"edutalks/internal/services"
	helpers "edutalks/internal/utils/helpers"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
// @Description Подверждает email по токену из письма
// @Tags email
// @Accept json
// @Produce json
// @Param token query string true "Токен подтверждения"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /api/verify-email [get]
func (h *EmailHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	token := r.URL.Query().Get("token")
	if strings.TrimSpace(token) == "" {
		log.Warn("VerifyEmail: отсутствует токен")
		helpers.Error(w, http.StatusBadRequest, "Токен отсутствует")
		return
	}

	if err := h.emailTokenService.ConfirmToken(r.Context(), token); err != nil {
		log.Warn("VerifyEmail: ошибка подтверждения email", zap.Error(err))

		var msg string
		switch err {
		case services.ErrTokenInvalid:
			msg = "Неверный или уже использованный токен."
		case services.ErrTokenExpired:
			msg = "Срок действия токена истёк."
		default:
			msg = "Внутренняя ошибка сервиса."
		}
		helpers.Error(w, http.StatusBadRequest, msg)
		return
	}

	cfg, _ := config.LoadConfig()
	base := strings.TrimRight(strings.TrimSpace(cfg.FrontendURL), "/")
	if base == "" {
		base = "https://edutalks.ru"
	}
	redirectURL := base + "/verify-email?status=success"

	log.Info("VerifyEmail: email подтверждён, редирект на фронт", zap.String("redirect_to", redirectURL))
	http.Redirect(w, r, redirectURL, http.StatusFound)
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
// @Failure 429 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/resend-verification [post]
func (h *AuthHandler) ResendVerificationEmail(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	type request struct {
		Email string `json:"email"`
	}
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Email) == "" {
		log.Warn("ResendVerificationEmail: невалидный payload")
		helpers.Error(w, http.StatusBadRequest, "Неверный формат запроса или пустой email")
		return
	}

	user, err := h.authService.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		log.Warn("ResendVerificationEmail: пользователь не найден", zap.String("email_masked", maskEmail(req.Email)))
		helpers.Error(w, http.StatusNotFound, "Пользователь не найден")
		return
	}

	// Лимит повторной отправки
	if lastToken, err := h.emailTokenService.GetLastTokenByUserID(r.Context(), user.ID); err == nil {
		nextAllowed := lastToken.CreatedAt.Add(5 * time.Minute)
		if nextAllowed.After(time.Now()) {
			remaining := int(time.Until(nextAllowed).Seconds())
			log.Info("ResendVerificationEmail: превышен лимит, слишком рано",
				zap.Time("created_at", lastToken.CreatedAt),
				zap.Time("next_allowed", nextAllowed),
				zap.Int("remaining_sec", remaining),
				zap.Int("user_id", user.ID),
			)
			helpers.Error(w, http.StatusTooManyRequests,
				fmt.Sprintf("Вы можете повторно запросить письмо через %d секунд", remaining))
			return
		}
	}

	// Создание нового токена
	emailToken, err := h.emailTokenService.GenerateToken(r.Context(), user.ID)
	if err != nil {
		log.Error("ResendVerificationEmail: ошибка генерации токена", zap.Error(err), zap.Int("user_id", user.ID))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка генерации токена")
		return
	}

	// Отправка письма
	if err := h.SendVerificationEmail(r.Context(), user, emailToken.Token); err != nil {
		log.Error("ResendVerificationEmail: ошибка при отправке письма", zap.Error(err), zap.Int("user_id", user.ID))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка при отправке письма")
		return
	}

	log.Info("ResendVerificationEmail: письмо отправлено повторно", zap.Int("user_id", user.ID))
	helpers.JSON(w, http.StatusOK, map[string]string{
		"message": "Письмо с подтверждением отправлено повторно",
	})
}
