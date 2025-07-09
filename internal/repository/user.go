package repository

import (
	"context"
	"edutalks/internal/models"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(ctx context.Context, user *models.User) error {
	query := `
	INSERT INTO users (username, full_name, phone, email, address, password_hash, role)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.Exec(ctx, query,
		user.Username,
		user.FullName,
		user.Phone,
		user.Email,
		user.Address,
		user.PasswordHash,
		user.Role,
	)
	return err
}

func (r *UserRepository) IsUsernameTaken(ctx context.Context, username string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, username).Scan(&exists)
	return exists, err
}

func (r *UserRepository) IsEmailTaken(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, email).Scan(&exists)
	return exists, err
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `SELECT id, username, full_name, phone, email, address, password_hash, role, created_at, updated_at FROM users WHERE username = $1`

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
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) SaveRefreshToken(ctx context.Context, userID int, token string) error {
	query := `INSERT INTO refresh_tokens (user_id, token) VALUES ($1, $2)`
	_, err := r.db.Exec(ctx, query, userID, token)
	return err
}

func (r *UserRepository) IsRefreshTokenValid(ctx context.Context, userID int, token string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM refresh_tokens WHERE user_id = $1 AND token = $2)`
	var exists bool
	err := r.db.QueryRow(ctx, query, userID, token).Scan(&exists)
	return exists, err
}

func (r *UserRepository) DeleteRefreshToken(ctx context.Context, userID int, token string) error {
	query := `DELETE FROM refresh_tokens WHERE user_id = $1 AND token = $2`
	_, err := r.db.Exec(ctx, query, userID, token)
	return err
}

func (r *UserRepository) GetAllUsers(ctx context.Context) ([]*models.User, error) {
	query := `
		SELECT id, username, full_name, phone, email, address, role, created_at, updated_at
		FROM users
		WHERE role = 'user'
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
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
		)
		if err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	query := `
		SELECT id, username, full_name, phone, email, address, role, created_at, updated_at
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
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) UpdateUserFields(ctx context.Context, id int, input *models.UpdateUserRequest) error {
	query := `UPDATE users SET`
	args := []interface{}{}
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
		return nil // ничего не обновляем
	}

	query = strings.TrimSuffix(query, ",") + fmt.Sprintf(" WHERE id = $%d", argNum)
	args = append(args, id)

	_, err := r.db.Exec(ctx, query, args...)
	return err
}
