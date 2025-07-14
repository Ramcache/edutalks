package repository

import (
	"context"
	"edutalks/internal/logger"
	"edutalks/internal/models"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(ctx context.Context, user *models.User) error {
	logger.Log.Info("Создание пользователя (repo)", zap.String("username", user.Username), zap.String("email", user.Email))
	query := `
	INSERT INTO users (username, full_name, phone, email, address, password_hash, role)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	RETURNING id`
	return r.db.QueryRow(ctx, query,
		user.Username,
		user.FullName,
		user.Phone,
		user.Email,
		user.Address,
		user.PasswordHash,
		user.Role,
	).Scan(&user.ID)
}

func (r *UserRepository) IsUsernameTaken(ctx context.Context, username string) (bool, error) {
	logger.Log.Debug("Проверка username на уникальность (repo)", zap.String("username", username))
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, username).Scan(&exists)
	if err != nil {
		logger.Log.Error("Ошибка проверки username (repo)", zap.Error(err))
	}
	return exists, err
}

func (r *UserRepository) IsEmailTaken(ctx context.Context, email string) (bool, error) {
	logger.Log.Debug("Проверка email на уникальность (repo)", zap.String("email", email))
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		logger.Log.Error("Ошибка проверки email (repo)", zap.Error(err))
	}
	return exists, err
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	logger.Log.Debug("Получение пользователя по username (repo)", zap.String("username", username))
	query := `SELECT id, username, full_name, phone, email, address, password_hash, role, created_at, updated_at, has_subscription, email_subscription, email_verified
	FROM users 
	WHERE username = $1`

	var user models.User
	err := r.db.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.FullName,
		&user.Phone,
		&user.Email,
		&user.Address,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.HasSubscription,
	)

	if err != nil {
		logger.Log.Error("Ошибка получения пользователя по username (repo)", zap.String("username", username), zap.Error(err))
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) SaveRefreshToken(ctx context.Context, userID int, token string) error {
	logger.Log.Debug("Сохранение refresh токена (repo)", zap.Int("user_id", userID))
	query := `INSERT INTO refresh_tokens (user_id, token) VALUES ($1, $2)`
	_, err := r.db.Exec(ctx, query, userID, token)
	if err != nil {
		logger.Log.Error("Ошибка сохранения refresh токена (repo)", zap.Error(err))
	}
	return err
}

func (r *UserRepository) IsRefreshTokenValid(ctx context.Context, userID int, token string) (bool, error) {
	logger.Log.Debug("Проверка refresh токена (repo)", zap.Int("user_id", userID))
	query := `SELECT EXISTS(SELECT 1 FROM refresh_tokens WHERE user_id = $1 AND token = $2)`
	var exists bool
	err := r.db.QueryRow(ctx, query, userID, token).Scan(&exists)
	if err != nil {
		logger.Log.Error("Ошибка проверки refresh токена (repo)", zap.Error(err))
	}
	return exists, err
}

func (r *UserRepository) DeleteRefreshToken(ctx context.Context, userID int, token string) error {
	logger.Log.Debug("Удаление refresh токена (repo)", zap.Int("user_id", userID))
	query := `DELETE FROM refresh_tokens WHERE user_id = $1 AND token = $2`
	_, err := r.db.Exec(ctx, query, userID, token)
	if err != nil {
		logger.Log.Error("Ошибка удаления refresh токена (repo)", zap.Error(err))
	}
	return err
}

func (r *UserRepository) GetAllUsers(ctx context.Context) ([]*models.User, error) {
	logger.Log.Info("Получение всех пользователей (repo)")
	query := `SELECT id, username, full_name, phone, email, address, role, created_at, updated_at, has_subscription, email_subscription, email_verified FROM users`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		logger.Log.Error("Ошибка получения пользователей (repo)", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var u models.User
		err := rows.Scan(
			&u.ID,
			&u.Username,
			&u.FullName,
			&u.Phone,
			&u.Email,
			&u.Address,
			&u.Role,
			&u.CreatedAt,
			&u.UpdatedAt,
			&u.HasSubscription,
			&u.EmailSubscription,
			&u.Email_verified,
		)
		if err != nil {
			logger.Log.Error("Ошибка сканирования пользователя (repo)", zap.Error(err))
			return nil, err
		}
		users = append(users, &u)
	}
	return users, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	logger.Log.Debug("Получение пользователя по ID (repo)", zap.Int("user_id", id))
	query := `
		SELECT id, username, full_name, phone, email, address, role, created_at, updated_at, has_subscription, email_subscription, email_verified
		FROM users
		WHERE id = $1
	`

	var u models.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Username,
		&u.FullName,
		&u.Phone,
		&u.Email,
		&u.Address,
		&u.Role,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.HasSubscription,
		&u.EmailSubscription,
		&u.Email_verified,
	)
	if err != nil {
		logger.Log.Error("Ошибка получения пользователя по ID (repo)", zap.Int("user_id", id), zap.Error(err))
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) UpdateUserFields(ctx context.Context, id int, input *models.UpdateUserRequest) error {
	logger.Log.Info("Обновление пользователя (repo)", zap.Int("user_id", id))
	query := `UPDATE users SET`
	var args []interface{}
	argNum := 1

	if input.FullName != nil {
		query += fmt.Sprintf(" full_name = $%d,", argNum)
		args = append(args, *input.FullName)
		argNum++
	}
	if input.Email != nil {
		query += fmt.Sprintf(" email = $%d,", argNum)
		args = append(args, *input.Email)
		argNum++
	}
	if input.Phone != nil {
		query += fmt.Sprintf(" phone = $%d,", argNum)
		args = append(args, *input.Phone)
		argNum++
	}
	if input.Address != nil {
		query += fmt.Sprintf(" address = $%d,", argNum)
		args = append(args, *input.Address)
		argNum++
	}
	if input.Role != nil {
		query += fmt.Sprintf(" role = $%d,", argNum)
		args = append(args, *input.Role)
		argNum++
	}

	if len(args) == 0 {
		logger.Log.Warn("Нет полей для обновления пользователя (repo)", zap.Int("user_id", id))
		return nil // ничего не обновляем
	}

	query = strings.TrimSuffix(query, ",") + fmt.Sprintf(" WHERE id = $%d", argNum)
	args = append(args, id)

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		logger.Log.Error("Ошибка обновления пользователя (repo)", zap.Error(err), zap.Int("user_id", id))
	}
	return err
}

func (r *UserRepository) UpdateSubscriptionStatus(ctx context.Context, userID int, status bool) error {
	logger.Log.Info("Изменение статуса подписки (repo)", zap.Int("user_id", userID), zap.Bool("status", status))
	query := `UPDATE users SET has_subscription = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, status, userID)
	if err != nil {
		logger.Log.Error("Ошибка обновления подписки (repo)", zap.Error(err), zap.Int("user_id", userID))
	}
	return err
}

func (r *UserRepository) GetSubscribedEmails(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT email FROM users WHERE email_subscription = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err == nil {
			emails = append(emails, email)
		}
	}
	return emails, nil
}

// internal/repository/user.go

func (r *UserRepository) UpdateEmailSubscription(ctx context.Context, userID int, subscribe bool) error {
	query := `UPDATE users SET email_subscription = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, subscribe, userID)
	return err
}
