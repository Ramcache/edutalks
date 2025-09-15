package handlers

import (
	"context"
	"edutalks/internal/config"
	"edutalks/internal/logger"
	"edutalks/internal/middleware"
	"edutalks/internal/models"
	"edutalks/internal/services"
	helpers "edutalks/internal/utils/helpers"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type AuthHandler struct {
	authService       *services.AuthService
	emailService      *services.EmailService
	emailTokenService *services.EmailTokenService
}

func NewAuthHandler(authService *services.AuthService, emailService *services.EmailService, emailTokenService *services.EmailTokenService) *AuthHandler {
	return &AuthHandler{
		authService:       authService,
		emailService:      emailService,
		emailTokenService: emailTokenService,
	}
}

type registerRequest struct {
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Address  string `json:"address"`
	Password string `json:"password"`
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`

	Username string `json:"username,omitempty"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
	Username    string `json:"username"`
	FullName    string `json:"full_name"`
	Role        string `json:"role"`
}

type subscriptionRequest struct {
	Active bool `json:"active"`
}

type notifyRequest struct {
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type emailSubscriptionRequest struct {
	Subscribe bool `json:"subscribe"`
}

// Register godoc
// @Summary Регистрация нового пользователя
// @Tags auth
// @Accept json
// @Produce json
// @Param input body registerRequest true "Данные регистрации"
// @Success 201 {string} string "Пользователь успешно зарегистрирован"
// @Failure 400 {string} string "Ошибка валидации"
// @Router /api/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("Невалидный JSON в Register", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "Невалидный JSON")
		return
	}

	log.Info("Регистрация пользователя",
		zap.String("username", strings.TrimSpace(req.Username)),
		zap.String("email_masked", maskEmail(req.Email)),
		zap.String("phone_masked", maskPhone(req.Phone)),
	)

	user := &models.User{
		Username: req.Username,
		FullName: req.FullName,
		Phone:    req.Phone,
		Email:    req.Email,
		Address:  req.Address,
	}

	if err := h.authService.RegisterUser(r.Context(), user, req.Password); err != nil {
		log.Error("Ошибка регистрации пользователя", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// Антиспам по письмам с подтверждением
	if lastToken, err := h.emailTokenService.GetLastTokenByUserID(r.Context(), user.ID); err == nil && time.Since(lastToken.CreatedAt) < 5*time.Minute {
		log.Warn("Слишком частая отправка письма подтверждения",
			zap.Int("user_id", user.ID),
			zap.Duration("remaining", 5*time.Minute-time.Since(lastToken.CreatedAt)),
		)
		helpers.Error(w, http.StatusTooManyRequests, "Повторная отправка письма возможна через 5 минут")
		return
	}

	// Генерация токена
	emailToken, err := h.emailTokenService.GenerateToken(r.Context(), user.ID)
	if err != nil {
		log.Error("Ошибка генерации email-токена", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка генерации токена")
		return
	}

	// Отправка письма с токеном
	if err := h.SendVerificationEmail(r.Context(), user, emailToken.Token); err != nil {
		log.Error("Ошибка отправки письма подтверждения", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка при отправке письма")
		return
	}

	log.Info("Пользователь зарегистрирован, письмо подтверждения отправлено", zap.Int("user_id", user.ID))
	helpers.JSON(w, http.StatusCreated, "Пользователь успешно зарегистрирован. Проверьте вашу почту для подтверждения.")
}

// Login godoc
// @Summary Авторизация пользователя
// @Tags auth
// @Accept json
// @Produce json
// @Param input body loginRequest true "Данные для входа"
// @Success 200 {object} loginResponse
// @Failure 401 {string} string "Неверный логин или пароль"
// @Router /api/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	//log := logger.WithCtx(r.Context())

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.Error(w, http.StatusBadRequest, "Невалидный JSON")
		return
	}

	identifier := strings.TrimSpace(req.Login)
	if identifier == "" {
		identifier = strings.TrimSpace(req.Username)
	}
	if identifier == "" || req.Password == "" {
		helpers.Error(w, http.StatusBadRequest, "Требуются login/username и password")
		return
	}

	cfg, _ := config.LoadConfig()
	accessTTL, _ := time.ParseDuration(cfg.AccessTokenTTL)

	access, user, err := h.authService.LoginUserByIdentifier(
		r.Context(), identifier, req.Password, cfg.JWTSecret, accessTTL,
	)
	if err != nil {
		helpers.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	resp := loginResponse{
		AccessToken: access,
		Username:    user.Username,
		FullName:    user.FullName,
		Role:        user.Role,
	}
	helpers.JSON(w, http.StatusOK, resp)
}

// Protected godoc
// @Summary Получить данные профиля
// @Tags profile
// @Security ApiKeyAuth
// @Success 200 {object} models.UserProfileResponse "Профиль пользователя"
// @Failure 401 {string} string "Нет доступа"
// @Failure 404 {string} string "Пользователь не найден"
// @Router /api/profile [get]
func (h *AuthHandler) Protected(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	userID, ok := r.Context().Value(middleware.ContextUserID).(int)
	if !ok || userID == 0 {
		log.Warn("Нет доступа в /profile: user_id отсутствует")
		helpers.Error(w, http.StatusUnauthorized, "Нет доступа")
		return
	}

	user, err := h.authService.GetUserByID(r.Context(), userID)
	if err != nil {
		log.Warn("Пользователь не найден (profile)", zap.Int("user_id", userID))
		helpers.Error(w, http.StatusNotFound, "Пользователь не найден")
		return
	}

	now := time.Now().UTC()
	isActive := user.HasSubscription && user.SubscriptionExpiresAt != nil && user.SubscriptionExpiresAt.After(now)

	resp := models.UserProfileResponse{
		ID:                    user.ID,
		Username:              user.Username,
		FullName:              user.FullName,
		Phone:                 user.Phone,
		Email:                 user.Email,
		Address:               user.Address,
		Role:                  user.Role,
		CreatedAt:             user.CreatedAt,
		UpdatedAt:             user.UpdatedAt,
		HasSubscription:       user.HasSubscription,
		SubscriptionExpiresAt: user.SubscriptionExpiresAt,
		IsSubscriptionActive:  isActive,
		EmailSubscription:     user.EmailSubscription,
		EmailVerified:         user.EmailVerified,
	}

	log.Info("Профиль отдан", zap.Int("user_id", userID))
	helpers.JSON(w, http.StatusOK, resp)
}

// Logout godoc
// @Summary Выход (удаление refresh токена)
// @Tags auth
// @Security ApiKeyAuth
// @Success 200 {string} string "Выход выполнен"
// @Failure 401 {string} string "Невалидный токен"
// @Router /api/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		helpers.Error(w, http.StatusUnauthorized, "Отсутствует токен")
		return
	}

	cfg, _ := config.LoadConfig()
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		helpers.Error(w, http.StatusUnauthorized, "Невалидный токен")
		return
	}

	expUnix, _ := claims["exp"].(float64)
	exp := time.Unix(int64(expUnix), 0)

	if err := h.authService.Logout(r.Context(), tokenString, exp); err != nil {
		log.Error("Ошибка при logout", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка при выходе")
		return
	}

	log.Info("Пользователь вышел, токен в блоклисте")
	helpers.JSON(w, http.StatusOK, "Выход выполнен")
}

// AdminOnly godoc
// @Summary Доступ только для администратора
// @Tags admin-users
// @Security ApiKeyAuth
// @Success 200 {string} string "Доступно только администратору"
// @Failure 403 {string} string "Доступ запрещён"
// @Router /api/admin/dashboard [get]
func (h *AuthHandler) AdminOnly(w http.ResponseWriter, r *http.Request) {
	logger.WithCtx(r.Context()).Info("AdminOnly доступ")
	helpers.JSON(w, http.StatusOK, "Доступно только администратору")
}

// GetUsers godoc
// @Summary Получить пользователей (с фильтрами)
// @Tags admin-users
// @Security ApiKeyAuth
// @Produce json
// @Param page query int false "Номер страницы (начиная с 1)"
// @Param page_size query int false "Размер страницы"
// @Param q query string false "Поиск по ФИО или email"
// @Param role query string false "Фильтр по роли (admin/user/...)"
// @Param has_subscription query string false "true|false — фильтр по подписке"
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/users [get]
func (h *AuthHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	q := r.URL.Query().Get("q")

	var rolePtr *string
	if role := strings.TrimSpace(r.URL.Query().Get("role")); role != "" {
		rolePtr = &role
	}

	var hasSubPtr *bool
	if hs := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("has_subscription"))); hs != "" {
		switch hs {
		case "true", "1", "t", "yes", "y":
			v := true
			hasSubPtr = &v
		case "false", "0", "f", "no", "n":
			v := false
			hasSubPtr = &v
		default:
			log.Warn("Невалидное значение has_subscription", zap.String("value", hs))
			helpers.Error(w, http.StatusBadRequest, "has_subscription должен быть true|false")
			return
		}
	}

	log.Info("Запрос списка пользователей",
		zap.Int("page", page), zap.Int("page_size", pageSize),
		zap.Int("offset", offset), zap.String("q", q),
		zap.Any("role", rolePtr), zap.Any("has_subscription", hasSubPtr),
	)

	users, total, err := h.authService.GetUsersFiltered(r.Context(), pageSize, offset, q, rolePtr, hasSubPtr)
	if err != nil {
		log.Error("Ошибка получения пользователей (handler)", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка получения пользователей")
		return
	}

	log.Info("Список пользователей получен", zap.Int("count", len(users)), zap.Int("total", total))
	helpers.JSON(w, http.StatusOK, map[string]interface{}{
		"data":             users,
		"total":            total,
		"page":             page,
		"page_size":        pageSize,
		"q":                q,
		"role":             rolePtr,
		"has_subscription": func() *bool { return hasSubPtr }(),
	})
}

// GetUserByID godoc
// @Summary Получить пользователя по ID
// @Tags admin-users
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID пользователя"
// @Success 200 {object} models.User
// @Failure 400 {string} string "Невалидный ID"
// @Failure 404 {string} string "Пользователь не найден"
// @Router /api/admin/users/{id} [get]
func (h *AuthHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Warn("Невалидный ID при получении пользователя", zap.String("id", idStr))
		helpers.Error(w, http.StatusBadRequest, "Невалидный ID")
		return
	}

	user, err := h.authService.GetUserByID(r.Context(), id)
	if err != nil {
		log.Warn("Пользователь не найден", zap.Int("user_id", id))
		helpers.Error(w, http.StatusNotFound, "Пользователь не найден")
		return
	}

	log.Info("Получен пользователь по ID", zap.Int("user_id", id))
	helpers.JSON(w, http.StatusOK, user)
}

// UpdateUser godoc
// @Summary Частичное обновление пользователя
// @Tags admin-users
// @Security ApiKeyAuth
// @Param id path int true "ID пользователя"
// @Accept json
// @Produce json
// @Param input body models.UpdateUserRequest true "Что обновить"
// @Success 200 {string} string "Пользователь обновлён"
// @Failure 400 {string} string "Ошибка валидации"
// @Failure 404 {string} string "Пользователь не найден"
// @Router /api/admin/users/{id} [patch]
func (h *AuthHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Warn("Невалидный ID при обновлении пользователя", zap.Any("vars", vars))
		helpers.Error(w, http.StatusBadRequest, "Невалидный ID")
		return
	}

	var input models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Warn("Невалидный JSON при обновлении пользователя", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "Невалидный JSON")
		return
	}

	if err := h.authService.UpdateUser(r.Context(), id, &input); err != nil {
		log.Error("Ошибка при обновлении пользователя", zap.Error(err), zap.Int("user_id", id))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка при обновлении")
		return
	}

	log.Info("Пользователь обновлён", zap.Int("user_id", id))
	helpers.JSON(w, http.StatusOK, "Пользователь обновлён")
}

// SetSubscription godoc
// @Summary Управление подпиской пользователя (выдать/продлить/отключить)
// @Tags admin-users
// @Security ApiKeyAuth
// @Param id path int true "ID пользователя"
// @Accept json
// @Produce json
// @Param input body setSubscriptionRequest true "Действие над подпиской"
// @Success 200 {object} map[string]interface{} "Текущее состояние подписки"
// @Failure 400 {string} string "Ошибка запроса"
// @Router /api/admin/users/{id}/subscription [patch]
func (h *AuthHandler) SetSubscription(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	idStr := mux.Vars(r)["id"]
	userID, err := strconv.Atoi(idStr)
	if err != nil {
		log.Warn("Неверный ID при обновлении подписки", zap.String("id", idStr))
		helpers.Error(w, http.StatusBadRequest, "Неверный ID")
		return
	}

	var req setSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("Невалидный JSON при обновлении подписки", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "Невалидный JSON")
		return
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "" {
		action = "grant"
	}

	switch action {
	case "revoke":
		log.Info("Отключение подписки", zap.Int("user_id", userID))
		if err := h.authService.SetSubscription(r.Context(), userID, false); err != nil {
			log.Error("Ошибка отключения подписки", zap.Error(err), zap.Int("user_id", userID))
			helpers.Error(w, http.StatusInternalServerError, "Ошибка отключения подписки")
			return
		}
	case "grant", "extend":
		dur, err := parseHumanDuration(req.Duration)
		if err != nil {
			log.Warn("Невалидный duration при подписке", zap.String("duration", req.Duration))
			helpers.Error(w, http.StatusBadRequest, "Неверный формат duration")
			return
		}
		if action == "grant" {
			log.Info("Выдача подписки", zap.Int("user_id", userID), zap.String("duration", req.Duration), zap.Duration("parsed", dur))
			if err := h.authService.SetSubscriptionWithExpiry(r.Context(), userID, dur); err != nil {
				log.Error("Ошибка выдачи подписки", zap.Error(err), zap.Int("user_id", userID))
				helpers.Error(w, http.StatusInternalServerError, "Ошибка выдачи подписки")
				return
			}
		} else {
			log.Info("Продление подписки", zap.Int("user_id", userID), zap.String("duration", req.Duration), zap.Duration("parsed", dur))
			if err := h.authService.ExtendSubscription(r.Context(), userID, dur); err != nil {
				log.Error("Ошибка продления подписки", zap.Error(err), zap.Int("user_id", userID))
				helpers.Error(w, http.StatusInternalServerError, "Ошибка продления подписки")
				return
			}
		}
	default:
		log.Warn("Невалидное действие для подписки", zap.String("action", action))
		helpers.Error(w, http.StatusBadRequest, "action должен быть grant|extend|revoke")
		return
	}

	u, err := h.authService.GetUserByID(r.Context(), userID)
	if err != nil {
		log.Error("Не удалось получить пользователя после изменения подписки", zap.Error(err), zap.Int("user_id", userID))
		helpers.Error(w, http.StatusInternalServerError, "Не удалось получить пользователя")
		return
	}
	now := time.Now().UTC()
	isActive := u.HasSubscription && u.SubscriptionExpiresAt != nil && u.SubscriptionExpiresAt.After(now)

	log.Info("Подписка обновлена", zap.Int("user_id", userID), zap.Bool("has_subscription", u.HasSubscription))
	helpers.JSON(w, http.StatusOK, map[string]interface{}{
		"user_id":                 u.ID,
		"has_subscription":        u.HasSubscription,
		"subscription_expires_at": u.SubscriptionExpiresAt,
		"is_active":               isActive,
	})
}

type setSubscriptionRequest struct {
	Action   string `json:"action"`             // grant | extend | revoke
	Duration string `json:"duration,omitempty"` // monthly | halfyear | yearly | "30d" | "72h" | ...
}

// NotifySubscribers godoc
// @Summary Отправить письмо всем подписанным
// @Tags admin-notify
// @Security ApiKeyAuth
// @Accept json
// @Param input body notifyRequest true "Сообщение"
// @Success 200 {string} string "Письма отправлены"
// @Failure 400 {string} string "Ошибка запроса"
// @Failure 500 {string} string "Ошибка отправки"
// @Router /api/admin/notify [post]
func (h *AuthHandler) NotifySubscribers(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	var req notifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("Невалидный JSON в NotifySubscribers", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "Невалидный JSON")
		return
	}

	emails, err := h.authService.GetSubscribedEmails(r.Context())
	if err != nil {
		log.Error("Не удалось получить список подписчиков", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Не удалось получить список подписчиков")
		return
	}

	if len(emails) == 0 {
		log.Info("Нет подписчиков для рассылки")
		helpers.JSON(w, http.StatusOK, map[string]string{"message": "Нет подписчиков"})
		return
	}

	for _, email := range emails {
		html := helpers.BuildSimpleHTML(req.Subject, req.Message)
		services.EmailQueue <- services.EmailJob{
			To:      []string{email},
			Subject: req.Subject,
			Body:    html,
			IsHTML:  true,
		}
	}
	log.Info("Письма поставлены в очередь", zap.Int("count", len(emails)))
	helpers.JSON(w, http.StatusOK, "Письма отправлены")
}

// EmailSubscribe godoc
// @Summary Подписка или отписка от email-уведомлений
// @Tags auth
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param input body emailSubscriptionRequest true "Подписка на email"
// @Success 200 {string} string "Статус подписки обновлён"
// @Failure 400 {string} string "Невалидный запрос"
// @Router /api/email-subscription [patch]
func (h *AuthHandler) EmailSubscribe(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	var req emailSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("Невалидный JSON в EmailSubscribe", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "Невалидный JSON")
		return
	}

	userID, ok := r.Context().Value(middleware.ContextUserID).(int)
	if !ok || userID == 0 {
		log.Warn("Нет доступа для EmailSubscribe: user_id отсутствует")
		helpers.Error(w, http.StatusUnauthorized, "Нет доступа")
		return
	}

	if err := h.authService.UpdateEmailSubscription(r.Context(), userID, req.Subscribe); err != nil {
		log.Error("Не удалось обновить статус email-подписки", zap.Error(err), zap.Int("user_id", userID))
		helpers.Error(w, http.StatusInternalServerError, "Не удалось обновить статус подписки")
		return
	}

	log.Info("Статус email-подписки обновлён", zap.Int("user_id", userID), zap.Bool("subscribe", req.Subscribe))
	helpers.JSON(w, http.StatusOK, map[string]string{"message": "Статус подписки обновлён"})
}

func (h *AuthHandler) SendVerificationEmail(ctx context.Context, user *models.User, token string) error {
	cfg, _ := config.LoadConfig()
	verifyLink := fmt.Sprintf("%s/verify-email?token=%s", cfg.SiteURL, token)
	htmlBody := helpers.BuildVerificationHTML(user.FullName, verifyLink)

	services.EmailQueue <- services.EmailJob{
		To:      []string{user.Email},
		Subject: "Подтверждение регистрации",
		Body:    htmlBody,
		IsHTML:  true,
	}
	logger.WithCtx(ctx).Info("Письмо подтверждения поставлено в очередь", zap.String("email_masked", maskEmail(user.Email)))

	return nil
}

// DeleteUser
// @Summary Удалить пользователя
// @Description Удаляет пользователя по его ID
// @Tags Users
// @Param id path int true "ID пользователя"
// @Success 200 {object} map[string]string "Пользователь успешно удалён"
// @Failure 400 {object} string "Некорректный id пользователя"
// @Failure 404 {object} string "Пользователь не найден"
// @Failure 500 {object} string "Ошибка при удалении пользователя"
// @Security ApiKeyAuth
// @Router /api/admin/users/{id} [delete]
func (h *AuthHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Warn("Некорректный id пользователя в DeleteUser", zap.String("id", idStr))
		helpers.Error(w, http.StatusBadRequest, "Некорректный id пользователя")
		return
	}

	log.Info("Запрос на удаление пользователя", zap.Int("user_id", id))

	if _, err := h.authService.GetUserByID(r.Context(), id); err != nil {
		log.Warn("Пользователь не найден для удаления", zap.Int("user_id", id))
		helpers.Error(w, http.StatusNotFound, "Пользователь не найден")
		return
	}

	if err := h.authService.DeleteUserByID(r.Context(), id); err != nil {
		log.Error("Ошибка при удалении пользователя из БД", zap.Error(err), zap.Int("user_id", id))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка при удалении пользователя")
		return
	}

	log.Info("Пользователь успешно удалён", zap.Int("user_id", id))
	helpers.JSON(w, http.StatusOK, map[string]string{"message": "Пользователь удалён"})
}

// GetSystemStats godoc
// @Summary Системная статистика для админ-дашборда
// @Tags admin-users
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} models.SystemStats
// @Router /api/admin/stats [get]
func (h *AuthHandler) GetSystemStats(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	stats, err := h.authService.GetSystemStats(r.Context())
	if err != nil {
		log.Error("Ошибка получения системной статистики (handler)", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Не удалось получить статистику")
		return
	}

	log.Info("Системная статистика отдана")
	helpers.JSON(w, http.StatusOK, stats)
}

// --- helpers ---

// parseHumanDuration:
// "monthly"=30d, "halfyear"=182d, "yearly"=365d, "Nd" — дни, также поддерживаются "72h", "90m", "3600s".
func parseHumanDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "monthly":
		return 30 * 24 * time.Hour, nil
	case "halfyear":
		return 182 * 24 * time.Hour, nil
	case "yearly":
		return 365 * 24 * time.Hour, nil
	}
	if strings.HasSuffix(s, "d") {
		num := strings.TrimSuffix(s, "d")
		n, err := strconv.Atoi(num)
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("bad days")
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

func keys(m map[string]interface{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func maskEmail(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	at := strings.IndexByte(s, '@')
	if at <= 1 {
		return "***"
	}
	name := s[:at]
	domain := s[at:]
	if len(name) <= 2 {
		return name[:1] + "*" + domain
	}
	return name[:1] + strings.Repeat("*", len(name)-2) + name[len(name)-1:] + domain
}

func maskPhone(s string) string {
	digits := []rune{}
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	if len(digits) < 4 {
		return "***"
	}
	return "***" + string(digits[len(digits)-4:])
}

func maskLogin(s string) string {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "@") {
		return maskEmail(s)
	}
	onlyDigits := true
	for _, r := range s {
		if r < '0' || r > '9' {
			onlyDigits = false
			break
		}
	}
	if onlyDigits {
		return maskPhone(s)
	}
	if len(s) <= 2 {
		return s[:1] + "*"
	}
	return s[:1] + strings.Repeat("*", len(s)-2) + s[len(s)-1:]
}
