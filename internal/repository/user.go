package repository

import (
	"context"
	"edutalks/internal/logger"
	"edutalks/internal/models"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

type UserRepo interface {
	IsUsernameTaken(ctx context.Context, username string) (bool, error)
	IsEmailTaken(ctx context.Context, email string) (bool, error)
	CreateUser(ctx context.Context, user *models.User) error
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	SaveRefreshToken(ctx context.Context, userID int, token string) error
	IsRefreshTokenValid(ctx context.Context, userID int, token string) (bool, error)
	DeleteRefreshToken(ctx context.Context, userID int, token string) error
	GetAllUsersPaginated(ctx context.Context, limit, offset int) ([]*models.User, int, error)
	GetUserByID(ctx context.Context, id int) (*models.User, error)
	UpdateUserFields(ctx context.Context, id int, input *models.UpdateUserRequest) error
	UpdateSubscriptionStatus(ctx context.Context, userID int, status bool) error
	GetSubscribedEmails(ctx context.Context) ([]string, error)
	UpdateEmailSubscription(ctx context.Context, userID int, subscribe bool) error
	SetEmailVerified(ctx context.Context, userID int, verified bool) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	DeleteUserByID(ctx context.Context, userID int) error
	SetSubscriptionWithExpiry(ctx context.Context, userID int, duration time.Duration) error
	ExpireSubscriptions(ctx context.Context) error
	ExtendSubscription(ctx context.Context, userID int, duration time.Duration) error
	GetUserByPhone(ctx context.Context, phoneDigits string) (*models.User, error)
	GetSystemStats(ctx context.Context) (*models.SystemStats, error)
	GetUsersFiltered(
		ctx context.Context,
		limit, offset int,
		q string,
		role *string,
		hasSubscription *bool,
	) ([]*models.User, int, error)
}

func (r *UserRepository) CreateUser(ctx context.Context, user *models.User) error {
	log := logger.WithCtx(ctx)

	const q = `
		INSERT INTO users (username, full_name, phone, email, address, password_hash, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	if err := r.db.QueryRow(ctx, q,
		user.Username,
		user.FullName,
		user.Phone,
		user.Email,
		user.Address,
		user.PasswordHash,
		user.Role,
	).Scan(&user.ID); err != nil {
		log.Error("user repo: create user failed", zap.Error(err), zap.String("username", user.Username), zap.String("email", user.Email))
		return err
	}

	log.Info("user repo: user created", zap.Int("id", user.ID), zap.String("username", user.Username))
	return nil
}

func (r *UserRepository) IsUsernameTaken(ctx context.Context, username string) (bool, error) {
	log := logger.WithCtx(ctx)

	const q = `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`
	var exists bool
	if err := r.db.QueryRow(ctx, q, username).Scan(&exists); err != nil {
		log.Error("user repo: username check failed", zap.Error(err), zap.String("username", username))
		return false, err
	}
	log.Debug("user repo: username exists check", zap.String("username", username), zap.Bool("exists", exists))
	return exists, nil
}

func (r *UserRepository) IsEmailTaken(ctx context.Context, email string) (bool, error) {
	log := logger.WithCtx(ctx)

	const q = `SELECT EXISTS(SELECT 1 FROM users WHERE lower(email) = lower($1))`
	var exists bool
	if err := r.db.QueryRow(ctx, q, email).Scan(&exists); err != nil {
		log.Error("user repo: email check failed", zap.Error(err), zap.String("email", email))
		return false, err
	}
	log.Debug("user repo: email exists check", zap.String("email", email), zap.Bool("exists", exists))
	return exists, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	log := logger.WithCtx(ctx)

	const q = `
		SELECT id, username, full_name, phone, email, address, password_hash, role,
		       created_at, updated_at, has_subscription, subscription_expires_at,
		       email_subscription, email_verified
		FROM users
		WHERE username = $1
	`

	var user models.User
	if err := r.db.QueryRow(ctx, q, username).Scan(
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
		&user.SubscriptionExpiresAt,
		&user.EmailSubscription,
		&user.EmailVerified,
	); err != nil {
		log.Error("user repo: get by username failed", zap.Error(err), zap.String("username", username))
		return nil, err
	}

	log.Debug("user repo: got user by username", zap.String("username", username), zap.Int("id", user.ID))
	return &user, nil
}

func (r *UserRepository) SaveRefreshToken(ctx context.Context, userID int, token string) error {
	log := logger.WithCtx(ctx)

	const q = `INSERT INTO refresh_tokens (user_id, token) VALUES ($1, $2)`
	if _, err := r.db.Exec(ctx, q, userID, token); err != nil {
		log.Error("user repo: save refresh token failed", zap.Error(err), zap.Int("user_id", userID))
		return err
	}
	log.Debug("user repo: refresh token saved", zap.Int("user_id", userID))
	return nil
}

func (r *UserRepository) IsRefreshTokenValid(ctx context.Context, userID int, token string) (bool, error) {
	log := logger.WithCtx(ctx)

	const q = `SELECT EXISTS(SELECT 1 FROM refresh_tokens WHERE user_id = $1 AND token = $2)`
	var exists bool
	if err := r.db.QueryRow(ctx, q, userID, token).Scan(&exists); err != nil {
		log.Error("user repo: check refresh token failed", zap.Error(err), zap.Int("user_id", userID))
		return false, err
	}
	log.Debug("user repo: refresh token valid check", zap.Int("user_id", userID), zap.Bool("valid", exists))
	return exists, nil
}

func (r *UserRepository) DeleteRefreshToken(ctx context.Context, userID int, token string) error {
	log := logger.WithCtx(ctx)

	const q = `DELETE FROM refresh_tokens WHERE user_id = $1 AND token = $2`
	if _, err := r.db.Exec(ctx, q, userID, token); err != nil {
		log.Error("user repo: delete refresh token failed", zap.Error(err), zap.Int("user_id", userID))
		return err
	}
	log.Debug("user repo: refresh token deleted", zap.Int("user_id", userID))
	return nil
}

func (r *UserRepository) GetAllUsersPaginated(ctx context.Context, limit, offset int) ([]*models.User, int, error) {
	log := logger.WithCtx(ctx)

	const q = `
		SELECT id, username, full_name, phone, email, address, role,
		       created_at, updated_at, has_subscription, subscription_expires_at,
		       email_subscription, email_verified
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, q, limit, offset)
	if err != nil {
		log.Error("user repo: list users failed", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(
			&u.ID, &u.Username, &u.FullName, &u.Phone, &u.Email, &u.Address,
			&u.Role, &u.CreatedAt, &u.UpdatedAt, &u.HasSubscription, &u.SubscriptionExpiresAt,
			&u.EmailSubscription, &u.EmailVerified,
		); err != nil {
			log.Error("user repo: scan user failed", zap.Error(err))
			return nil, 0, err
		}
		users = append(users, &u)
	}
	if err := rows.Err(); err != nil {
		log.Error("user repo: rows error list users", zap.Error(err))
		return nil, 0, err
	}

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&total); err != nil {
		log.Error("user repo: count users failed", zap.Error(err))
		return nil, 0, err
	}

	log.Debug("user repo: list users done", zap.Int("count", len(users)), zap.Int("total", total))
	return users, total, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	log := logger.WithCtx(ctx)

	const q = `
		SELECT id, username, full_name, phone, email, address,
		       password_hash, role, created_at, updated_at,
		       has_subscription, subscription_expires_at,
		       email_subscription, email_verified
		FROM users
		WHERE id = $1
	`

	var u models.User
	if err := r.db.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.Username, &u.FullName, &u.Phone, &u.Email, &u.Address,
		&u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt,
		&u.HasSubscription, &u.SubscriptionExpiresAt,
		&u.EmailSubscription, &u.EmailVerified,
	); err != nil {
		log.Error("user repo: get by id failed", zap.Error(err), zap.Int("user_id", id))
		return nil, err
	}

	log.Debug("user repo: got user by id", zap.Int("user_id", id))
	return &u, nil
}

func (r *UserRepository) UpdateUserFields(ctx context.Context, id int, input *models.UpdateUserRequest) error {
	log := logger.WithCtx(ctx)

	q := `UPDATE users SET`
	var args []any
	argNum := 1

	if input.FullName != nil {
		q += fmt.Sprintf(" full_name = $%d,", argNum)
		args = append(args, *input.FullName)
		argNum++
	}
	if input.Email != nil {
		q += fmt.Sprintf(" email = $%d,", argNum)
		args = append(args, *input.Email)
		argNum++
	}
	if input.Phone != nil {
		q += fmt.Sprintf(" phone = $%d,", argNum)
		args = append(args, *input.Phone)
		argNum++
	}
	if input.Address != nil {
		q += fmt.Sprintf(" address = $%d,", argNum)
		args = append(args, *input.Address)
		argNum++
	}
	if input.Role != nil {
		q += fmt.Sprintf(" role = $%d,", argNum)
		args = append(args, *input.Role)
		argNum++
	}

	if len(args) == 0 {
		log.Warn("user repo: no fields to update", zap.Int("user_id", id))
		return nil
	}

	q = strings.TrimSuffix(q, ",") + fmt.Sprintf(" WHERE id = $%d", argNum)
	args = append(args, id)

	if _, err := r.db.Exec(ctx, q, args...); err != nil {
		log.Error("user repo: update user failed", zap.Error(err), zap.Int("user_id", id))
		return err
	}

	log.Info("user repo: user updated", zap.Int("user_id", id))
	return nil
}

func (r *UserRepository) UpdateSubscriptionStatus(ctx context.Context, userID int, status bool) error {
	log := logger.WithCtx(ctx)

	const q = `
		UPDATE users
		SET has_subscription = $1,
		    subscription_expires_at = CASE WHEN $1 THEN subscription_expires_at ELSE NULL END
		WHERE id = $2
	`
	if _, err := r.db.Exec(ctx, q, status, userID); err != nil {
		log.Error("user repo: update subscription status failed", zap.Error(err), zap.Int("user_id", userID))
		return err
	}

	log.Info("user repo: subscription status updated", zap.Int("user_id", userID), zap.Bool("status", status))
	return nil
}

func (r *UserRepository) GetSubscribedEmails(ctx context.Context) ([]string, error) {
	log := logger.WithCtx(ctx)

	rows, err := r.db.Query(ctx, `SELECT email FROM users WHERE email_subscription = true`)
	if err != nil {
		log.Error("user repo: get subscribed emails failed", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var emails []string
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			log.Error("user repo: scan subscribed email failed", zap.Error(err))
			return nil, err
		}
		emails = append(emails, email)
	}
	if err := rows.Err(); err != nil {
		log.Error("user repo: rows error subscribed emails", zap.Error(err))
		return nil, err
	}

	log.Debug("user repo: got subscribed emails", zap.Int("count", len(emails)))
	return emails, nil
}

func (r *UserRepository) UpdateEmailSubscription(ctx context.Context, userID int, subscribe bool) error {
	log := logger.WithCtx(ctx)

	const q = `UPDATE users SET email_subscription = $1 WHERE id = $2`
	if _, err := r.db.Exec(ctx, q, subscribe, userID); err != nil {
		log.Error("user repo: update email subscription failed", zap.Error(err), zap.Int("user_id", userID))
		return err
	}
	log.Info("user repo: email subscription updated", zap.Int("user_id", userID), zap.Bool("subscribe", subscribe))
	return nil
}

func (r *UserRepository) SetEmailVerified(ctx context.Context, userID int, verified bool) error {
	log := logger.WithCtx(ctx)

	const q = `UPDATE users SET email_verified = $1 WHERE id = $2`
	if _, err := r.db.Exec(ctx, q, verified, userID); err != nil {
		log.Error("user repo: set email verified failed", zap.Error(err), zap.Int("user_id", userID))
		return err
	}
	log.Info("user repo: email verification updated", zap.Int("user_id", userID), zap.Bool("verified", verified))
	return nil
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	log := logger.WithCtx(ctx)

	const q = `
		SELECT id, username, full_name, phone, email, address, password_hash, role,
		       created_at, updated_at, has_subscription, subscription_expires_at,
		       email_subscription, email_verified
		FROM users
		WHERE lower(email) = lower($1)
	`

	var user models.User
	if err := r.db.QueryRow(ctx, q, email).Scan(
		&user.ID, &user.Username, &user.FullName, &user.Phone, &user.Email, &user.Address,
		&user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt,
		&user.HasSubscription, &user.SubscriptionExpiresAt,
		&user.EmailSubscription, &user.EmailVerified,
	); err != nil {
		log.Error("user repo: get by email failed", zap.Error(err), zap.String("email", email))
		return nil, err
	}

	log.Debug("user repo: got user by email", zap.Int("id", user.ID))
	return &user, nil
}

func (r *UserRepository) DeleteUserByID(ctx context.Context, userID int) error {
	log := logger.WithCtx(ctx)

	const q = `DELETE FROM users WHERE id = $1`
	if _, err := r.db.Exec(ctx, q, userID); err != nil {
		log.Error("user repo: delete user failed", zap.Error(err), zap.Int("user_id", userID))
		return err
	}

	log.Info("user repo: user deleted", zap.Int("user_id", userID))
	return nil
}

func (r *UserRepository) SetSubscriptionWithExpiry(ctx context.Context, userID int, duration time.Duration) error {
	log := logger.WithCtx(ctx)

	const q = `
		UPDATE users
		SET has_subscription = true,
		    subscription_expires_at = NOW() + $1 * interval '1 second'
		WHERE id = $2
	`
	if _, err := r.db.Exec(ctx, q, int64(duration.Seconds()), userID); err != nil {
		log.Error("user repo: set subscription with expiry failed", zap.Error(err), zap.Int("user_id", userID))
		return err
	}

	log.Info("user repo: subscription set with expiry", zap.Int("user_id", userID), zap.Int64("seconds", int64(duration.Seconds())))
	return nil
}

func (r *UserRepository) ExpireSubscriptions(ctx context.Context) error {
	log := logger.WithCtx(ctx)

	const q = `
		UPDATE users
		SET has_subscription = false
		WHERE has_subscription = true
		  AND subscription_expires_at IS NOT NULL
		  AND subscription_expires_at <= NOW()
	`
	if _, err := r.db.Exec(ctx, q); err != nil {
		log.Error("user repo: expire subscriptions failed", zap.Error(err))
		return err
	}

	log.Info("user repo: subscriptions expired where due")
	return nil
}

func (r *UserRepository) ExtendSubscription(ctx context.Context, userID int, duration time.Duration) error {
	log := logger.WithCtx(ctx)

	const q = `
		UPDATE users
		SET has_subscription = true,
		    subscription_expires_at = COALESCE(subscription_expires_at, NOW()) + $1 * interval '1 second'
		WHERE id = $2
	`
	if _, err := r.db.Exec(ctx, q, int64(duration.Seconds()), userID); err != nil {
		log.Error("user repo: extend subscription failed", zap.Error(err), zap.Int("user_id", userID))
		return err
	}

	log.Info("user repo: subscription extended", zap.Int("user_id", userID), zap.Int64("seconds", int64(duration.Seconds())))
	return nil
}

func (r *UserRepository) GetUserByPhone(ctx context.Context, phoneDigits string) (*models.User, error) {
	log := logger.WithCtx(ctx)

	const q = `
		SELECT id, username, full_name, phone, email, address, password_hash, role,
		       created_at, updated_at, has_subscription, subscription_expires_at,
		       email_subscription, email_verified
		FROM users
		WHERE right(regexp_replace(phone, '\D', '', 'g'), 10) = right($1, 10)
		LIMIT 1
	`

	var user models.User
	if err := r.db.QueryRow(ctx, q, phoneDigits).Scan(
		&user.ID, &user.Username, &user.FullName, &user.Phone, &user.Email, &user.Address,
		&user.PasswordHash, &user.Role, &user.CreatedAt, &user.UpdatedAt,
		&user.HasSubscription, &user.SubscriptionExpiresAt,
		&user.EmailSubscription, &user.EmailVerified,
	); err != nil {
		log.Error("user repo: get by phone failed", zap.Error(err))
		return nil, err
	}

	log.Debug("user repo: got user by phone", zap.Int("id", user.ID))
	return &user, nil
}

func (r *UserRepository) GetSystemStats(ctx context.Context) (*models.SystemStats, error) {
	log := logger.WithCtx(ctx)

	const q = `
SELECT
  (SELECT COUNT(*) FROM users)                                                   AS total_users,
  (SELECT COUNT(*) FROM users WHERE role = 'admin')                              AS admins,
  (SELECT COUNT(*) FROM users WHERE role <> 'admin' OR role IS NULL)             AS regular_users,
  (SELECT COUNT(*) FROM users
     WHERE has_subscription = true
       AND (subscription_expires_at IS NULL OR subscription_expires_at > NOW())
  )                                                                              AS with_subscription,
  (SELECT COUNT(*) FROM users
     WHERE has_subscription = false
        OR (subscription_expires_at IS NOT NULL AND subscription_expires_at <= NOW())
  )                                                                              AS without_subscription,
  (SELECT COUNT(*) FROM news)                                                    AS news_count,
  (SELECT COUNT(*) FROM documents)                                               AS documents_count,
  (SELECT COUNT(*) FROM articles)                                                AS articles_count
`
	var s models.SystemStats
	if err := r.db.QueryRow(ctx, q).Scan(
		&s.TotalUsers,
		&s.Admins,
		&s.RegularUsers,
		&s.WithSubscription,
		&s.WithoutSubscription,
		&s.NewsCount,
		&s.DocumentsCount,
		&s.ArticlesCount,
	); err != nil {
		log.Error("user repo: get system stats failed", zap.Error(err))
		return nil, err
	}

	if s.TotalUsers > 0 {
		s.WithSubscriptionPct = int(float64(s.WithSubscription)*100.0/float64(s.TotalUsers) + 0.5)
		s.WithoutSubscriptionPct = 100 - s.WithSubscriptionPct
	}

	log.Debug("user repo: system stats", zap.Any("stats", s))
	return &s, nil
}

func (r *UserRepository) GetUsersFiltered(
	ctx context.Context,
	limit, offset int,
	q string,
	role *string,
	hasSubscription *bool,
) ([]*models.User, int, error) {
	log := logger.WithCtx(ctx)

	base := `
		SELECT id, username, full_name, phone, email, address, role,
		       created_at, updated_at, has_subscription, subscription_expires_at,
		       email_subscription, email_verified
		FROM users
	`
	where := " WHERE 1=1"
	whereArgs := []any{}
	argn := 1

	q = strings.TrimSpace(q)
	if q != "" {
		where += fmt.Sprintf(" AND (full_name ILIKE $%d OR lower(email) ILIKE $%d)", argn, argn+1)
		whereArgs = append(whereArgs, "%"+q+"%", "%"+strings.ToLower(q)+"%")
		argn += 2
	}
	if role != nil && strings.TrimSpace(*role) != "" {
		where += fmt.Sprintf(" AND role = $%d", argn)
		whereArgs = append(whereArgs, strings.TrimSpace(*role))
		argn++
	}
	if hasSubscription != nil {
		where += fmt.Sprintf(" AND has_subscription = $%d", argn)
		whereArgs = append(whereArgs, *hasSubscription)
		argn++
	}

	orderPage := fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argn, argn+1)
	args := append(append([]any{}, whereArgs...), limit, offset)

	rows, err := r.db.Query(ctx, base+where+orderPage, args...)
	if err != nil {
		log.Error("user repo: filtered list users failed", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(
			&u.ID, &u.Username, &u.FullName, &u.Phone, &u.Email, &u.Address, &u.Role,
			&u.CreatedAt, &u.UpdatedAt, &u.HasSubscription, &u.SubscriptionExpiresAt,
			&u.EmailSubscription, &u.EmailVerified,
		); err != nil {
			log.Error("user repo: scan filtered user failed", zap.Error(err))
			return nil, 0, err
		}
		users = append(users, &u)
	}
	if err := rows.Err(); err != nil {
		log.Error("user repo: rows error filtered users", zap.Error(err))
		return nil, 0, err
	}

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM users"+where, whereArgs...).Scan(&total); err != nil {
		log.Error("user repo: count filtered users failed", zap.Error(err))
		return nil, 0, err
	}

	log.Debug("user repo: filtered users done",
		zap.Int("count", len(users)),
		zap.Int("total", total),
		zap.String("q", q),
		zap.Any("role", role),
		zap.Any("has_subscription", hasSubscription),
	)
	return users, total, nil
}
