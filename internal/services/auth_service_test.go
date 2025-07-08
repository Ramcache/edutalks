package services

import (
	"context"
	"edutalks/internal/models"
	"edutalks/internal/utils"
	"errors"
	"testing"
	"time"
)

// Мок-репозиторий (заглушка)
type mockUserRepo struct {
	users    map[string]*models.User
	lastUser *models.User
}

func (m *mockUserRepo) IsUsernameTaken(_ context.Context, username string) (bool, error) {
	_, exists := m.users[username]
	return exists, nil
}

func (m *mockUserRepo) IsEmailTaken(_ context.Context, email string) (bool, error) {
	for _, u := range m.users {
		if u.Email == email {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockUserRepo) CreateUser(_ context.Context, user *models.User) error {
	m.users[user.Username] = user
	m.lastUser = user
	return nil
}

func (m *mockUserRepo) GetByUsername(_ context.Context, username string) (*models.User, error) {
	u, ok := m.users[username]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

func (m *mockUserRepo) SaveRefreshToken(_ context.Context, userID int, token string) error {
	return nil
}
func (m *mockUserRepo) IsRefreshTokenValid(_ context.Context, userID int, token string) (bool, error) {
	return true, nil
}
func (m *mockUserRepo) DeleteRefreshToken(_ context.Context, userID int, token string) error {
	return nil
}

func TestRegisterUser(t *testing.T) {
	repo := &mockUserRepo{users: make(map[string]*models.User)}
	service := NewAuthService(repo)

	user := &models.User{
		Username: "testuser",
		Email:    "test@example.com",
		FullName: "Тестовый Пользователь",
	}

	err := service.RegisterUser(context.Background(), user, "secret")
	if err != nil {
		t.Fatalf("ошибка регистрации: %v", err)
	}

	if repo.lastUser == nil || repo.lastUser.PasswordHash == "" {
		t.Fatal("пароль не захеширован или пользователь не сохранён")
	}
}

func TestLoginUser_Success(t *testing.T) {
	repo := &mockUserRepo{users: make(map[string]*models.User)}
	service := NewAuthService(repo)

	// создаём пользователя вручную
	hashed, _ := utils.HashPassword("secret")
	repo.users["testuser"] = &models.User{
		ID:           1,
		Username:     "testuser",
		PasswordHash: hashed,
		Role:         "user",
	}

	access, refresh, err := service.LoginUser(context.Background(), "testuser", "secret", "mysecret", 15*time.Minute, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("ошибка логина: %v", err)
	}

	if access == "" || refresh == "" {
		t.Fatal("токены не сгенерированы")
	}
}

func TestLoginUser_Fail(t *testing.T) {
	repo := &mockUserRepo{users: make(map[string]*models.User)}
	service := NewAuthService(repo)

	_, _, err := service.LoginUser(context.Background(), "unknown", "pass", "secret", time.Minute, time.Hour)
	if err == nil {
		t.Fatal("ожидалась ошибка при логине несуществующего пользователя")
	}
}
