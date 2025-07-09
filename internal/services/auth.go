package services

import (
	"context"
	"edutalks/internal/models"
	"edutalks/internal/utils"
	"errors"
	"time"
)

type AuthService struct {
	repo UserRepo
}

func NewAuthService(repo UserRepo) *AuthService {
	return &AuthService{repo: repo}
}

type UserRepo interface {
	IsUsernameTaken(ctx context.Context, username string) (bool, error)
	IsEmailTaken(ctx context.Context, email string) (bool, error)
	CreateUser(ctx context.Context, user *models.User) error
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	SaveRefreshToken(ctx context.Context, userID int, token string) error
	IsRefreshTokenValid(ctx context.Context, userID int, token string) (bool, error)
	DeleteRefreshToken(ctx context.Context, userID int, token string) error
	GetAllUsers(ctx context.Context) ([]*models.User, error)
	GetUserByID(ctx context.Context, id int) (*models.User, error)
	UpdateUserFields(ctx context.Context, id int, input *models.UpdateUserRequest) error
}

func (s *AuthService) RegisterUser(ctx context.Context, input *models.User, plainPassword string) error {
	if exists, _ := s.repo.IsUsernameTaken(ctx, input.Username); exists {
		return errors.New("username already taken")
	}
	if exists, _ := s.repo.IsEmailTaken(ctx, input.Email); exists {
		return errors.New("email already registered")
	}

	hashed, err := utils.HashPassword(plainPassword)
	if err != nil {
		return err
	}

	input.PasswordHash = hashed
	input.Role = "user"

	return s.repo.CreateUser(ctx, input)
}

func (s *AuthService) LoginUser(
	ctx context.Context,
	username, password, jwtSecret string,
	accessTTL, refreshTTL time.Duration,
) (string, string, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return "", "", errors.New("пользователь не найден")
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return "", "", errors.New("неверный пароль")
	}

	accessToken, err := utils.GenerateToken(jwtSecret, user.ID, user.Role, accessTTL)
	if err != nil {
		return "", "", err
	}

	refreshToken, err := utils.GenerateToken(jwtSecret, user.ID, user.Role, refreshTTL)
	if err != nil {
		return "", "", err
	}

	if err := s.repo.SaveRefreshToken(ctx, user.ID, refreshToken); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) ValidateRefreshToken(ctx context.Context, userID int, token string) (bool, error) {
	return s.repo.IsRefreshTokenValid(ctx, userID, token)
}

func (s *AuthService) Logout(ctx context.Context, userID int, token string) error {
	return s.repo.DeleteRefreshToken(ctx, userID, token)
}

func (s *AuthService) LoginUserWithUser(
	ctx context.Context,
	username, password, jwtSecret string,
	accessTTL, refreshTTL time.Duration,
) (string, string, *models.User, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return "", "", nil, errors.New("пользователь не найден")
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return "", "", nil, errors.New("неверный пароль")
	}

	accessToken, err := utils.GenerateToken(jwtSecret, user.ID, user.Role, accessTTL)
	if err != nil {
		return "", "", nil, err
	}

	refreshToken, err := utils.GenerateToken(jwtSecret, user.ID, user.Role, refreshTTL)
	if err != nil {
		return "", "", nil, err
	}

	if err := s.repo.SaveRefreshToken(ctx, user.ID, refreshToken); err != nil {
		return "", "", nil, err
	}

	return accessToken, refreshToken, user, nil
}

func (s *AuthService) GetUsers(ctx context.Context) ([]*models.User, error) {
	return s.repo.GetAllUsers(ctx)
}

func (s *AuthService) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	return s.repo.GetUserByID(ctx, id)
}

func (s *AuthService) UpdateUser(ctx context.Context, id int, input *models.UpdateUserRequest) error {
	return s.repo.UpdateUserFields(ctx, id, input)
}
