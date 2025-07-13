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
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
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

	logger.Log.Info("Пользователь успешно зарегистрирован", zap.String("username", req.Username))
	helpers.JSON(w, http.StatusCreated, "Пользователь успешно зарегистрирован")
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
// @Summary Защищённый маршрут (тест)
// @Tags protected
// @Security ApiKeyAuth
// @Success 200 {string} string "Привет, пользователь с ролью"
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
// @Tags admin
// @Security ApiKeyAuth
// @Success 200 {string} string "Доступно только администратору"
// @Failure 403 {string} string "Доступ запрещён"
// @Router /api/admin/dashboard [get]
func (h *AuthHandler) AdminOnly(w http.ResponseWriter, r *http.Request) {
	helpers.JSON(w, http.StatusOK, "Доступно только администратору")
}

// GetUsers godoc
// @Summary Получить всех пользователей с ролью user
// @Tags admin
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {array} models.User
// @Failure 403 {string} string "Доступ запрещён"
// @Router /api/admin/users [get]
func (h *AuthHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.authService.GetUsers(r.Context())
	if err != nil {
		logger.Log.Error("Ошибка получения пользователей", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка получения пользователей")
		return
	}
	logger.Log.Info("Получены пользователи", zap.Int("count", len(users)))
	helpers.JSON(w, http.StatusOK, users)
}

// GetUserByID godoc
// @Summary Получить пользователя по ID
// @Tags admin
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
// @Tags admin
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
// @Summary Включение или отключение подписки у пользователя (только admin)
// @Tags admin
// @Security ApiKeyAuth
// @Param id path int true "ID пользователя"
// @Param input body subscriptionRequest true "Статус подписки"
// @Success 200 {string} string "Статус обновлён"
// @Failure 400 {string} string "Ошибка запроса"
// @Router /admin/users/{id}/subscription [patch]
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
