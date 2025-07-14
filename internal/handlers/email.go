package handlers

import (
	"edutalks/internal/logger"
	"edutalks/internal/services"
	helpers "edutalks/internal/utils/helpres"
	"net/http"

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
// @Router /verify-email [get]
func (h *EmailHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		helpers.Error(w, http.StatusBadRequest, "Токен отсутствует")
		return
	}

	err := h.emailTokenService.ConfirmToken(r.Context(), token)
	if err != nil {
		logger.Log.Warn("Ошибка подтверждения email", zap.Error(err))
		switch err {
		case services.ErrTokenInvalid:
			helpers.Error(w, http.StatusBadRequest, "Неверный токен")
		case services.ErrTokenExpired:
			helpers.Error(w, http.StatusBadRequest, "Токен истёк")
		default:
			helpers.Error(w, http.StatusInternalServerError, "Ошибка подтверждения")
		}
		return
	}

	helpers.JSON(w, http.StatusOK, map[string]string{"message": "Почта успешно подтверждена"})
}
