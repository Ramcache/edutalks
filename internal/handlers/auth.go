package handlers

import (
	"context"
	"edutalks/internal/config"
	"edutalks/internal/logger"
	"edutalks/internal/middleware"
	"edutalks/internal/models"
	"edutalks/internal/services"
	"edutalks/internal/utils"
	helpers "edutalks/internal/utils/helpres"
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
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Username     string `json:"username"`
	FullName     string `json:"full_name"`
	Role         string `json:"role"`
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
// @Router /register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Warn("Ошибка декодирования JSON в Register", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "Невалидный JSON")
		return
	}
	logger.Log.Info("Регистрация пользователя", zap.String("username", req.Username), zap.String("email", req.Email))

	user := &models.User{
		Username: req.Username,
		FullName: req.FullName,
		Phone:    req.Phone,
		Email:    req.Email,
		Address:  req.Address,
	}

	err := h.authService.RegisterUser(context.Background(), user, req.Password)
	if err != nil {
		logger.Log.Error("Ошибка регистрации пользователя", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	_ = h.SendVerificationEmail(context.Background(), user)

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
// @Router /login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Warn("Ошибка декодирования JSON в Login", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "Невалидный JSON")
		return
	}
	logger.Log.Info("Попытка входа", zap.String("username", req.Username))

	cfg, _ := config.LoadConfig()
	accessTTL, _ := time.ParseDuration(cfg.AccessTokenTTL)
	refreshTTL, _ := time.ParseDuration(cfg.RefreshTokenTTL)

	access, refresh, user, err := h.authService.LoginUserWithUser(
		context.Background(),
		req.Username,
		req.Password,
		cfg.JWTSecret,
		accessTTL,
		refreshTTL,
	)
	if err != nil {
		logger.Log.Warn("Ошибка входа пользователя", zap.String("username", req.Username), zap.Error(err))
		helpers.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	logger.Log.Info("Вход выполнен", zap.String("username", req.Username), zap.String("role", user.Role))
	resp := loginResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		Username:     user.Username,
		FullName:     user.FullName,
		Role:         user.Role,
	}

	helpers.JSON(w, http.StatusOK, resp)
}

// Protected godoc
// @Summary Получить данные профиля
// @Tags profile
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{} "Профиль пользователя"
// @Failure 401 {string} string "Нет доступа"
// @Router /api/profile [get]
func (h *AuthHandler) Protected(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.ContextUserID)
	role := r.Context().Value(middleware.ContextRole)
	logger.Log.Debug("Доступ к защищённому маршруту", zap.Any("userID", userID), zap.Any("role", role))
	helpers.JSON(w, http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("Привет, пользователь #%v с ролью %v", userID, role),
	})
}

// Refresh godoc
// @Summary Обновление access-токена
// @Tags auth
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]string
// @Failure 401 {string} string "Недействительный refresh токен"
// @Router /refresh [post]
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		logger.Log.Warn("Отсутствует refresh token в Refresh")
		helpers.Error(w, http.StatusUnauthorized, "Отсутствует refresh token")
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	logger.Log.Debug("Попытка обновления токена", zap.String("token", tokenString))

	cfg, _ := config.LoadConfig()
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		logger.Log.Warn("Неверный или просроченный refresh token", zap.Error(err))
		helpers.Error(w, http.StatusUnauthorized, "Неверный или просроченный refresh token")
		return
	}

	userID, ok1 := claims["user_id"].(float64)
	role, ok2 := claims["role"].(string)
	if !ok1 || !ok2 {
		logger.Log.Error("Неверный payload токена", zap.Any("claims", claims))
		helpers.Error(w, http.StatusUnauthorized, "Неверный payload токена")
		return
	}

	isValid, err := h.authService.ValidateRefreshToken(r.Context(), int(userID), tokenString)
	if err != nil || !isValid {
		logger.Log.Warn("Недействительный refresh token", zap.Error(err))
		helpers.Error(w, http.StatusUnauthorized, "Недействительный refresh token")
		return
	}

	accessTTL, _ := time.ParseDuration(cfg.AccessTokenTTL)
	accessToken, err := utils.GenerateToken(cfg.JWTSecret, int(userID), role, accessTTL)
	if err != nil {
		logger.Log.Error("Ошибка генерации токена", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка генерации токена")
		return
	}

	logger.Log.Info("Токен обновлён", zap.Float64("user_id", userID))
	helpers.JSON(w, http.StatusOK, map[string]string{"access_token": accessToken})
}

// Logout godoc
// @Summary Выход (удаление refresh токена)
// @Tags auth
// @Security ApiKeyAuth
// @Success 200 {string} string "Выход выполнен"
// @Failure 401 {string} string "Невалидный токен"
// @Router /logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		logger.Log.Warn("Отсутствует refresh token в Logout")
		helpers.Error(w, http.StatusUnauthorized, "Отсутствует refresh token")
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	cfg, _ := config.LoadConfig()
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		logger.Log.Warn("Невалидный refresh token при выходе", zap.Error(err))
		helpers.Error(w, http.StatusUnauthorized, "Невалидный refresh token")
		return
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		logger.Log.Error("Неверный payload при выходе", zap.Any("claims", claims))
		helpers.Error(w, http.StatusUnauthorized, "Неверный payload")
		return
	}

	err = h.authService.Logout(r.Context(), int(userID), tokenString)
	if err != nil {
		logger.Log.Error("Ошибка при удалении токена", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка при удалении токена")
		return
	}

	logger.Log.Info("Пользователь вышел", zap.Float64("user_id", userID))
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
	helpers.JSON(w, http.StatusOK, "Доступно только администратору")
}

// GetUsers godoc
// @Summary Получить всех пользователей
// @Tags admin-users
// @Security ApiKeyAuth
// @Produce json
// @Param page query int false "Номер страницы (начиная с 1)"
// @Param page_size query int false "Размер страницы"
// @Success 200 {array} models.User
// @Failure 403 {string} string "Доступ запрещён"
// @Router /api/admin/users [get]
func (h *AuthHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	users, total, err := h.authService.GetUsersPaginated(r.Context(), pageSize, offset)
	if err != nil {
		logger.Log.Error("Ошибка получения пользователей", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка получения пользователей")
		return
	}
	helpers.JSON(w, http.StatusOK, map[string]interface{}{
		"data":      users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
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
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logger.Log.Warn("Невалидный ID при получении пользователя", zap.String("id", idStr))
		helpers.Error(w, http.StatusBadRequest, "Невалидный ID")
		return
	}

	user, err := h.authService.GetUserByID(r.Context(), id)
	if err != nil {
		logger.Log.Warn("Пользователь не найден", zap.Int("user_id", id))
		helpers.Error(w, http.StatusNotFound, "Пользователь не найден")
		return
	}
	logger.Log.Info("Получен пользователь по ID", zap.Int("user_id", id))
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
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		logger.Log.Warn("Невалидный ID при обновлении пользователя", zap.Any("vars", vars))
		helpers.Error(w, http.StatusBadRequest, "Невалидный ID")
		return
	}

	var input models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		logger.Log.Warn("Невалидный JSON при обновлении пользователя", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "Невалидный JSON")
		return
	}

	err = h.authService.UpdateUser(r.Context(), id, &input)
	if err != nil {
		logger.Log.Error("Ошибка при обновлении пользователя", zap.Error(err), zap.Int("user_id", id))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка при обновлении")
		return
	}

	logger.Log.Info("Пользователь обновлён", zap.Int("user_id", id))
	helpers.JSON(w, http.StatusOK, "Пользователь обновлён")
}

// SetSubscription godoc
// @Summary Включение или отключение подписки у пользователя
// @Tags admin-users
// @Security ApiKeyAuth
// @Param id path int true "ID пользователя"
// @Param input body subscriptionRequest true "Статус подписки"
// @Success 200 {string} string "Статус обновлён"
// @Failure 400 {string} string "Ошибка запроса"
// @Router /api/admin/users/{id}/subscription [patch]
func (h *AuthHandler) SetSubscription(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	userID, err := strconv.Atoi(idStr)
	if err != nil {
		logger.Log.Warn("Неверный ID при обновлении подписки", zap.String("id", idStr))
		helpers.Error(w, http.StatusBadRequest, "Неверный ID")
		return
	}

	var req subscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Warn("Невалидный JSON при обновлении подписки", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "Невалидный JSON")
		return
	}

	if err := h.authService.SetSubscription(r.Context(), userID, req.Active); err != nil {
		logger.Log.Error("Ошибка обновления подписки", zap.Error(err), zap.Int("user_id", userID))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка обновления подписки")
		return
	}

	logger.Log.Info("Подписка пользователя изменена", zap.Int("user_id", userID), zap.Bool("active", req.Active))
	helpers.JSON(w, http.StatusOK, "Статус подписки обновлён")
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
	var req notifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.Error(w, http.StatusBadRequest, "Невалидный JSON")
		return
	}

	emails, err := h.authService.GetSubscribedEmails(r.Context())
	if err != nil {
		helpers.Error(w, http.StatusInternalServerError, "Не удалось получить список подписчиков")
		return
	}

	if len(emails) == 0 {
		helpers.JSON(w, http.StatusOK, map[string]string{"message": "Нет подписчиков"})
		return
	}

	type result struct {
		Email string `json:"email"`
		Error string `json:"error,omitempty"`
	}
	results := make([]result, len(emails))

	for i, email := range emails {
		html := helpers.BuildSimpleHTML(req.Subject, req.Message)
		services.EmailQueue <- services.EmailJob{
			To:      []string{email},
			Subject: req.Subject,
			Body:    html,
			IsHTML:  true,
		}
		results[i] = result{Email: email}
	}

	helpers.JSON(w, http.StatusOK, map[string]interface{}{
		"sent":    len(emails),
		"failed":  0,
		"results": results,
	})
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
	var req emailSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.Error(w, http.StatusBadRequest, "Невалидный JSON")
		return
	}

	userID := r.Context().Value(middleware.ContextUserID).(int)

	err := h.authService.UpdateEmailSubscription(r.Context(), userID, req.Subscribe)
	if err != nil {
		helpers.Error(w, http.StatusInternalServerError, "Не удалось обновить статус подписки")
		return
	}

	helpers.JSON(w, http.StatusOK, map[string]string{"message": "Статус подписки обновлён"})
}

func (h *AuthHandler) SendVerificationEmail(ctx context.Context, user *models.User) error {
	emailToken, err := h.emailTokenService.GenerateToken(ctx, user.ID)
	if err != nil {
		logger.Log.Error("Ошибка генерации email токена", zap.Error(err))
		return err
	}

	cfg, _ := config.LoadConfig()
	verifyLink := fmt.Sprintf("%s/verify-email?token=%s", cfg.SiteURL, emailToken.Token)
	htmlBody := helpers.BuildVerificationHTML(user.FullName, verifyLink)

	services.EmailQueue <- services.EmailJob{
		To:      []string{user.Email},
		Subject: "Подтверждение регистрации",
		Body:    htmlBody,
		IsHTML:  true,
	}

	return nil
}
