package handlers

import (
	"context"
	"edutalks/internal/config"
	"edutalks/internal/middleware"
	"edutalks/internal/models"
	"edutalks/internal/services"
	"edutalks/internal/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
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
		http.Error(w, "Невалидный JSON", http.StatusBadRequest)
		return
	}

	user := &models.User{
		Username: req.Username,
		FullName: req.FullName,
		Phone:    req.Phone,
		Email:    req.Email,
		Address:  req.Address,
	}

	err := h.authService.RegisterUser(context.Background(), user, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Пользователь успешно зарегистрирован"))
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
		http.Error(w, "Невалидный JSON", http.StatusBadRequest)
		return
	}

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
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	resp := loginResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		Username:     user.Username,
		FullName:     user.FullName,
		Role:         user.Role,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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

	w.Write([]byte(fmt.Sprintf("Привет, пользователь #%v с ролью %v", userID, role)))
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
		http.Error(w, "Отсутствует refresh token", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	cfg, _ := config.LoadConfig()
	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "Неверный или просроченный refresh token", http.StatusUnauthorized)
		return
	}

	userID, ok1 := claims["user_id"].(float64)
	role, ok2 := claims["role"].(string)
	if !ok1 || !ok2 {
		http.Error(w, "Неверный payload токена", http.StatusUnauthorized)
		return
	}

	isValid, err := h.authService.ValidateRefreshToken(r.Context(), int(userID), tokenString)
	if err != nil || !isValid {
		http.Error(w, "Недействительный refresh token", http.StatusUnauthorized)
		return
	}

	accessTTL, _ := time.ParseDuration(cfg.AccessTokenTTL)
	accessToken, err := utils.GenerateToken(cfg.JWTSecret, int(userID), role, accessTTL)
	if err != nil {
		http.Error(w, "Ошибка генерации токена", http.StatusInternalServerError)
		return
	}

	resp := map[string]string{"access_token": accessToken}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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
		http.Error(w, "Отсутствует refresh token", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	cfg, _ := config.LoadConfig()
	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "Невалидный refresh token", http.StatusUnauthorized)
		return
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		http.Error(w, "Неверный payload", http.StatusUnauthorized)
		return
	}

	err = h.authService.Logout(r.Context(), int(userID), tokenString)
	if err != nil {
		http.Error(w, "Ошибка при удалении токена", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Выход выполнен"))
}

// AdminOnly godoc
// @Summary Доступ только для администратора
// @Tags admin
// @Security ApiKeyAuth
// @Success 200 {string} string "Доступно только администратору"
// @Failure 403 {string} string "Доступ запрещён"
// @Router /api/admin/dashboard [get]
func (h *AuthHandler) AdminOnly(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Доступно только администратору"))
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
		http.Error(w, "Ошибка получения пользователей", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}
