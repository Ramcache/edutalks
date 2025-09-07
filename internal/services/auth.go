package services

import (
	"context"
	"edutalks/internal/logger"
	"edutalks/internal/models"
	"edutalks/internal/utils"
	"edutalks/internal/utils/helpers"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
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
}

func (s *AuthService) RegisterUser(ctx context.Context, input *models.User, plainPassword string) error {
	logger.Log.Info("Регистрация пользователя (service)", zap.String("username", input.Username), zap.String("email", input.Email))
	if exists, err := s.repo.IsUsernameTaken(ctx, input.Username); exists || err != nil {
		if err != nil {
			logger.Log.Error("Ошибка проверки username", zap.Error(err))
		}
		return errors.New("имя пользователя уже занято")
	}
	if exists, err := s.repo.IsEmailTaken(ctx, input.Email); exists || err != nil {
		if err != nil {
			logger.Log.Error("Ошибка проверки email", zap.Error(err))
		}
		return errors.New("адрес электронной почты уже зарегистрирован")
	}

	hashed, err := utils.HashPassword(plainPassword)
	if err != nil {
		logger.Log.Error("Ошибка хеширования пароля", zap.Error(err))
		return err
	}

	input.PasswordHash = hashed
	input.Role = "user"

	if err := s.repo.CreateUser(ctx, input); err != nil {
		logger.Log.Error("Ошибка создания пользователя", zap.Error(err))
		return err
	}
	logger.Log.Info("Пользователь зарегистрирован (service)", zap.String("username", input.Username))
	return nil
}

func (s *AuthService) LoginUser(
	ctx context.Context,
	username, password, jwtSecret string,
	accessTTL, refreshTTL time.Duration,
) (string, string, error) {
	logger.Log.Info("Попытка входа (service)", zap.String("username", username))
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		logger.Log.Warn("Пользователь не найден (service)", zap.String("username", username), zap.Error(err))
		return "", "", errors.New("пользователь не найден")
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		logger.Log.Warn("Неверный пароль (service)", zap.String("username", username))
		return "", "", errors.New("неверный пароль")
	}

	accessToken, err := utils.GenerateToken(jwtSecret, user.ID, user.Role, accessTTL, "access")
	if err != nil {
		logger.Log.Error("Ошибка генерации access-токена", zap.Error(err))
		return "", "", err
	}

	refreshToken, err := utils.GenerateToken(jwtSecret, user.ID, user.Role, refreshTTL, "refresh")
	if err != nil {
		logger.Log.Error("Ошибка генерации refresh-токена", zap.Error(err))
		return "", "", err
	}

	if err := s.repo.SaveRefreshToken(ctx, user.ID, refreshToken); err != nil {
		logger.Log.Error("Ошибка сохранения refresh-токена", zap.Error(err))
		return "", "", err
	}

	logger.Log.Info("Вход выполнен (service)", zap.String("username", username))
	return accessToken, refreshToken, nil
}

func (s *AuthService) ValidateRefreshToken(ctx context.Context, userID int, token string) (bool, error) {
	logger.Log.Debug("Проверка refresh токена (service)", zap.Int("user_id", userID))
	return s.repo.IsRefreshTokenValid(ctx, userID, token)
}

func (s *AuthService) Logout(ctx context.Context, userID int, token string) error {
	logger.Log.Info("Выход пользователя (service)", zap.Int("user_id", userID))
	return s.repo.DeleteRefreshToken(ctx, userID, token)
}

func (s *AuthService) LoginUserWithUser(
	ctx context.Context,
	username, password, jwtSecret string,
	accessTTL, refreshTTL time.Duration,
) (string, string, *models.User, error) {
	logger.Log.Info("Попытка входа с возвратом user (service)", zap.String("username", username))
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		logger.Log.Warn("Пользователь не найден (service)", zap.String("username", username), zap.Error(err))
		return "", "", nil, errors.New("пользователь не найден")
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		logger.Log.Warn("Неверный пароль (service)", zap.String("username", username))
		return "", "", nil, errors.New("неверный пароль")
	}

	accessToken, err := utils.GenerateToken(jwtSecret, user.ID, user.Role, accessTTL, "access")
	if err != nil {
		logger.Log.Error("Ошибка генерации access-токена", zap.Error(err))
		return "", "", nil, err
	}

	refreshToken, err := utils.GenerateToken(jwtSecret, user.ID, user.Role, refreshTTL, "refresh")
	if err != nil {
		logger.Log.Error("Ошибка генерации refresh-токена", zap.Error(err))
		return "", "", nil, err
	}

	if err := s.repo.SaveRefreshToken(ctx, user.ID, refreshToken); err != nil {
		logger.Log.Error("Ошибка сохранения refresh-токена", zap.Error(err))
		return "", "", nil, err
	}

	logger.Log.Info("Вход выполнен (service)", zap.String("username", username))
	return accessToken, refreshToken, user, nil
}

func (s *AuthService) GetUsersPaginated(ctx context.Context, limit, offset int) ([]*models.User, int, error) {
	return s.repo.GetAllUsersPaginated(ctx, limit, offset)
}

func (s *AuthService) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	logger.Log.Info("Получение пользователя по ID (service)", zap.Int("user_id", id))
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		logger.Log.Warn("Пользователь не найден по ID (service)", zap.Int("user_id", id), zap.Error(err))
	}
	return user, err
}

func (s *AuthService) UpdateUser(ctx context.Context, id int, input *models.UpdateUserRequest) error {
	logger.Log.Info("Обновление пользователя (service)", zap.Int("user_id", id))
	if err := s.repo.UpdateUserFields(ctx, id, input); err != nil {
		logger.Log.Error("Ошибка при обновлении пользователя (service)", zap.Error(err), zap.Int("user_id", id))
		return err
	}
	logger.Log.Info("Пользователь обновлён (service)", zap.Int("user_id", id))
	return nil
}

func (s *AuthService) SetSubscription(ctx context.Context, userID int, status bool) error {
	logger.Log.Info("Изменение подписки (service)", zap.Int("user_id", userID), zap.Bool("status", status))

	// Получим пользователя ДО изменения — чтобы знать прежнюю дату окончания
	uBefore, _ := s.repo.GetUserByID(ctx, userID)
	var prevExpiresAt *time.Time
	if uBefore != nil && uBefore.SubscriptionExpiresAt != nil {
		prevExpiresAt = uBefore.SubscriptionExpiresAt
	}

	if err := s.repo.UpdateSubscriptionStatus(ctx, userID, status); err != nil {
		logger.Log.Error("Ошибка изменения подписки (service)", zap.Error(err))
		return err
	}

	// Если отключили подписку — отправим письмо
	if !status {
		u, err := s.repo.GetUserByID(ctx, userID)
		if err != nil {
			logger.Log.Warn("Не удалось получить пользователя после revoke", zap.Error(err), zap.Int("user_id", userID))
			return nil
		}
		if u != nil && u.Email != "" {
			// A) асинхронно через очередь (не блокируем запрос):
			html := helpers.BuildSubscriptionRevokedHTML(u.FullName, time.Now().UTC(), prevExpiresAt)
			EmailQueue <- EmailJob{
				To:      []string{u.Email},
				Subject: "Подписка отключена",
				Body:    html,
				IsHTML:  true,
			}

			// B) либо синхронно, если инжектируете EmailService в AuthService:
			// _ = s.emailService.SendSubscriptionRevoked(ctx, u.Email, u.FullName, time.Now().UTC(), prevExpiresAt)
		}
	}

	return nil
}

func (s *AuthService) GetSubscribedEmails(ctx context.Context) ([]string, error) {
	return s.repo.GetSubscribedEmails(ctx)
}

func (s *AuthService) UpdateEmailSubscription(ctx context.Context, userID int, subscribe bool) error {
	return s.repo.UpdateEmailSubscription(ctx, userID, subscribe)
}

func (s *AuthService) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	logger.Log.Info("Получение пользователя по email (service)", zap.String("email", email))
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		logger.Log.Warn("Пользователь не найден по email (service)", zap.String("email", email), zap.Error(err))
	}
	return user, err
}

func (s *AuthService) DeleteUserByID(ctx context.Context, id int) error {
	logger.Log.Info("Сервис: удаление user", zap.Int("user_id", id))
	err := s.repo.DeleteUserByID(ctx, id)
	if err != nil {
		logger.Log.Error("Ошибка удаления users (service)", zap.Int("user_id", id), zap.Error(err))
	}
	return err
}

func (s *AuthService) SetSubscriptionTrue(userID int) error {
	ctx := context.Background()
	return s.repo.UpdateSubscriptionStatus(ctx, userID, true)
}

func (s *AuthService) SetSubscriptionWithExpiry(ctx context.Context, userID int, duration time.Duration) error {
	logger.Log.Info("Выдача подписки с истечением (service)", zap.Int("user_id", userID), zap.Duration("duration", duration))

	if err := s.repo.SetSubscriptionWithExpiry(ctx, userID, duration); err != nil {
		logger.Log.Error("Ошибка выдачи подписки (service)", zap.Error(err))
		return err
	}

	// Получим пользователя, чтобы взять email/ФИО/дату истечения
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		logger.Log.Error("Не удалось получить пользователя после выдачи подписки", zap.Error(err), zap.Int("user_id", userID))
		return nil // подписку уже выдали — письмо не критично
	}

	// Если email есть — отправим уведомление
	if u != nil && u.Email != "" && u.SubscriptionExpiresAt != nil {
		plan := humanizeDuration(duration)

		// Вариант A: через очередь (не блокируем запрос)
		html := helpers.BuildSubscriptionGrantedHTML(u.FullName, plan, u.SubscriptionExpiresAt.Format("02.01.2006 15:04"))
		EmailQueue <- EmailJob{
			To:      []string{u.Email},
			Subject: "Подписка активирована",
			Body:    html,
			IsHTML:  true,
		}

		// Вариант B (синхронно): если вы инжектируете EmailService в AuthService
		// _ = s.emailService.SendSubscriptionGranted(ctx, u.Email, u.FullName, plan, *u.SubscriptionExpiresAt)
	}

	logger.Log.Info("Подписка с истечением успешно установлена", zap.Int("user_id", userID))
	return nil
}

func (s *AuthService) ExtendSubscription(ctx context.Context, userID int, duration time.Duration) error {
	logger.Log.Info("Продление подписки (service)", zap.Int("user_id", userID), zap.Duration("duration", duration))

	if err := s.repo.ExtendSubscription(ctx, userID, duration); err != nil {
		logger.Log.Error("Ошибка продления подписки (service)", zap.Error(err))
		return err
	}

	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		logger.Log.Error("Не удалось получить пользователя после продления", zap.Error(err), zap.Int("user_id", userID))
		return nil
	}

	if u != nil && u.Email != "" && u.SubscriptionExpiresAt != nil {
		plan := humanizeDuration(duration)
		html := helpers.BuildSubscriptionGrantedHTML(u.FullName, plan, u.SubscriptionExpiresAt.Format("02.01.2006 15:04"))
		EmailQueue <- EmailJob{
			To:      []string{u.Email},
			Subject: "Подписка продлена",
			Body:    html,
			IsHTML:  true,
		}
		// или синхронно через emailService, как выше
	}

	return nil
}

func (s *AuthService) findUserByIdentifier(ctx context.Context, identifier string) (*models.User, error) {
	id := strings.TrimSpace(identifier)
	if id == "" {
		return nil, errors.New("пустой логин")
	}

	// очень простая эвристика:
	// 1) email: есть '@'
	if strings.Contains(id, "@") {
		return s.repo.GetUserByEmail(ctx, id)
	}

	// 2) телефон: если после нормализации есть 10+ цифр
	digits := normalizePhoneDigits(id)
	if len(digits) >= 10 {
		return s.repo.GetUserByPhone(ctx, digits)
	}

	// 3) иначе считаем username
	return s.repo.GetByUsername(ctx, id)
}

func (s *AuthService) LoginUserByIdentifier(
	ctx context.Context,
	identifier, password, jwtSecret string,
	accessTTL, refreshTTL time.Duration,
) (string, string, *models.User, error) {
	user, err := s.findUserByIdentifier(ctx, identifier)
	if err != nil {
		logger.Log.Warn("Пользователь не найден (service)", zap.String("identifier", identifier), zap.Error(err))
		return "", "", nil, errors.New("пользователь не найден")
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		logger.Log.Warn("Неверный пароль (service)", zap.Int("user_id", user.ID))
		return "", "", nil, errors.New("неверный пароль")
	}

	accessToken, err := utils.GenerateToken(jwtSecret, user.ID, user.Role, accessTTL, "access")
	if err != nil {
		return "", "", nil, err
	}
	refreshToken, err := utils.GenerateToken(jwtSecret, user.ID, user.Role, refreshTTL, "refresh")
	if err != nil {
		return "", "", nil, err
	}

	if err := s.repo.SaveRefreshToken(ctx, user.ID, refreshToken); err != nil {
		return "", "", nil, err
	}
	return accessToken, refreshToken, user, nil
}
func humanizeDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	switch {
	case days%365 == 0 && days >= 365:
		return fmt.Sprintf("%d год(а)", days/365)
	case days%30 == 0 && days >= 30:
		return fmt.Sprintf("%d мес.", days/30)
	default:
		return fmt.Sprintf("%d дней", days)
	}
}

func normalizePhoneDigits(s string) string {
	// оставить только цифры, без какой-либо нормализации
	b := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b = append(b, r)
		}
	}
	return string(b)
}

func (s *AuthService) GetSystemStats(ctx context.Context) (*models.SystemStats, error) {
	return s.repo.GetSystemStats(ctx)
}
