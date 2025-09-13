package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"edutalks/internal/logger"
	"edutalks/internal/models"
	"edutalks/internal/repository"
	"edutalks/internal/utils"
	"edutalks/internal/utils/helpers"

	"go.uber.org/zap"
)

type AuthService struct {
	repo repository.UserRepo
}

func NewAuthService(repo repository.UserRepo) *AuthService {
	return &AuthService{repo: repo}
}

func (s *AuthService) RegisterUser(ctx context.Context, input *models.User, plainPassword string) error {
	log := logger.WithCtx(ctx)
	log.Info("Регистрация пользователя",
		zap.String("username", input.Username),
		zap.String("email", strings.ToLower(strings.TrimSpace(input.Email))),
	)

	if exists, err := s.repo.IsUsernameTaken(ctx, input.Username); exists || err != nil {
		if err != nil {
			log.Error("Ошибка проверки уникальности username", zap.Error(err))
		}
		return errors.New("имя пользователя уже занято")
	}
	if exists, err := s.repo.IsEmailTaken(ctx, input.Email); exists || err != nil {
		if err != nil {
			log.Error("Ошибка проверки уникальности email", zap.Error(err))
		}
		return errors.New("адрес электронной почты уже зарегистрирован")
	}

	hashed, err := utils.HashPassword(plainPassword)
	if err != nil {
		log.Error("Ошибка хеширования пароля", zap.Error(err))
		return err
	}

	input.PasswordHash = hashed
	input.Role = "user"

	if err := s.repo.CreateUser(ctx, input); err != nil {
		log.Error("Ошибка создания пользователя", zap.Error(err))
		return err
	}

	log.Info("Пользователь зарегистрирован",
		zap.String("username", input.Username),
		zap.Int("user_id", input.ID),
	)
	return nil
}

func (s *AuthService) LoginUser(
	ctx context.Context,
	username, password, jwtSecret string,
	accessTTL, refreshTTL time.Duration,
) (string, string, error) {
	log := logger.WithCtx(ctx)
	log.Info("Попытка входа", zap.String("username", username))

	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		log.Warn("Пользователь не найден", zap.String("username", username), zap.Error(err))
		return "", "", errors.New("пользователь не найден")
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		log.Warn("Неверный пароль", zap.Int("user_id", user.ID))
		return "", "", errors.New("неверный пароль")
	}

	accessToken, err := utils.GenerateToken(jwtSecret, user.ID, user.Role, accessTTL, "access")
	if err != nil {
		log.Error("Ошибка генерации access-токена", zap.Error(err))
		return "", "", err
	}

	refreshToken, err := utils.GenerateToken(jwtSecret, user.ID, user.Role, refreshTTL, "refresh")
	if err != nil {
		log.Error("Ошибка генерации refresh-токена", zap.Error(err))
		return "", "", err
	}

	if err := s.repo.SaveRefreshToken(ctx, user.ID, refreshToken); err != nil {
		log.Error("Ошибка сохранения refresh-токена", zap.Error(err))
		return "", "", err
	}

	log.Info("Вход выполнен", zap.Int("user_id", user.ID), zap.String("role", user.Role))
	return accessToken, refreshToken, nil
}

func (s *AuthService) ValidateRefreshToken(ctx context.Context, userID int, token string) (bool, error) {
	log := logger.WithCtx(ctx)
	log.Debug("Проверка refresh-токена", zap.Int("user_id", userID))
	return s.repo.IsRefreshTokenValid(ctx, userID, token)
}

func (s *AuthService) Logout(ctx context.Context, userID int, token string) error {
	log := logger.WithCtx(ctx)
	log.Info("Выход пользователя", zap.Int("user_id", userID))
	return s.repo.DeleteRefreshToken(ctx, userID, token)
}

func (s *AuthService) LoginUserWithUser(
	ctx context.Context,
	username, password, jwtSecret string,
	accessTTL, refreshTTL time.Duration,
) (string, string, *models.User, error) {
	log := logger.WithCtx(ctx)
	log.Info("Попытка входа (с возвратом пользователя)", zap.String("username", username))

	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		log.Warn("Пользователь не найден", zap.String("username", username), zap.Error(err))
		return "", "", nil, errors.New("пользователь не найден")
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		log.Warn("Неверный пароль", zap.Int("user_id", user.ID))
		return "", "", nil, errors.New("неверный пароль")
	}

	accessToken, err := utils.GenerateToken(jwtSecret, user.ID, user.Role, accessTTL, "access")
	if err != nil {
		log.Error("Ошибка генерации access-токена", zap.Error(err))
		return "", "", nil, err
	}

	refreshToken, err := utils.GenerateToken(jwtSecret, user.ID, user.Role, refreshTTL, "refresh")
	if err != nil {
		log.Error("Ошибка генерации refresh-токена", zap.Error(err))
		return "", "", nil, err
	}

	if err := s.repo.SaveRefreshToken(ctx, user.ID, refreshToken); err != nil {
		log.Error("Ошибка сохранения refresh-токена", zap.Error(err))
		return "", "", nil, err
	}

	log.Info("Вход выполнен", zap.Int("user_id", user.ID))
	return accessToken, refreshToken, user, nil
}

func (s *AuthService) GetUsersPaginated(ctx context.Context, limit, offset int) ([]*models.User, int, error) {
	return s.repo.GetAllUsersPaginated(ctx, limit, offset)
}

func (s *AuthService) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	log := logger.WithCtx(ctx)
	log.Info("Получение пользователя по ID", zap.Int("user_id", id))

	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		log.Warn("Пользователь не найден по ID", zap.Int("user_id", id), zap.Error(err))
	}
	return user, err
}

func (s *AuthService) UpdateUser(ctx context.Context, id int, input *models.UpdateUserRequest) error {
	log := logger.WithCtx(ctx)
	log.Info("Обновление пользователя", zap.Int("user_id", id))

	if err := s.repo.UpdateUserFields(ctx, id, input); err != nil {
		log.Error("Ошибка при обновлении пользователя", zap.Error(err), zap.Int("user_id", id))
		return err
	}

	log.Info("Пользователь обновлён", zap.Int("user_id", id))
	return nil
}

func (s *AuthService) SetSubscription(ctx context.Context, userID int, status bool) error {
	log := logger.WithCtx(ctx)
	log.Info("Изменение статуса подписки", zap.Int("user_id", userID), zap.Bool("status", status))

	// Снимем прежнюю дату окончания (для письма)
	uBefore, _ := s.repo.GetUserByID(ctx, userID)
	var prevExpiresAt *time.Time
	if uBefore != nil && uBefore.SubscriptionExpiresAt != nil {
		prevExpiresAt = uBefore.SubscriptionExpiresAt
	}

	if err := s.repo.UpdateSubscriptionStatus(ctx, userID, status); err != nil {
		log.Error("Ошибка изменения статуса подписки", zap.Error(err))
		return err
	}

	// При отключении подписки отправим письмо (не блокируя запрос)
	if !status {
		u, err := s.repo.GetUserByID(ctx, userID)
		if err != nil {
			log.Warn("Не удалось получить пользователя после отключения подписки", zap.Error(err), zap.Int("user_id", userID))
			return nil
		}
		if u != nil && u.Email != "" {
			html := helpers.BuildSubscriptionRevokedHTML(u.FullName, time.Now().UTC(), prevExpiresAt)
			EmailQueue <- EmailJob{
				To:      []string{u.Email},
				Subject: "Подписка отключена",
				Body:    html,
				IsHTML:  true,
			}
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
	log := logger.WithCtx(ctx)
	log.Info("Получение пользователя по email", zap.String("email", strings.ToLower(strings.TrimSpace(email))))

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		log.Warn("Пользователь не найден по email", zap.String("email", strings.ToLower(strings.TrimSpace(email))), zap.Error(err))
	}
	return user, err
}

func (s *AuthService) DeleteUserByID(ctx context.Context, id int) error {
	log := logger.WithCtx(ctx)
	log.Info("Удаление пользователя", zap.Int("user_id", id))

	err := s.repo.DeleteUserByID(ctx, id)
	if err != nil {
		log.Error("Ошибка удаления пользователя", zap.Int("user_id", id), zap.Error(err))
	}
	return err
}

func (s *AuthService) SetSubscriptionTrue(userID int) error {
	// Нет контекста извне — логгер без контекста.
	logger.Log.Info("Принудительное включение подписки", zap.Int("user_id", userID))
	ctx := context.Background()
	return s.repo.UpdateSubscriptionStatus(ctx, userID, true)
}

func (s *AuthService) SetSubscriptionWithExpiry(ctx context.Context, userID int, duration time.Duration) error {
	log := logger.WithCtx(ctx)
	log.Info("Выдача подписки с истечением", zap.Int("user_id", userID), zap.Duration("duration", duration))

	if err := s.repo.SetSubscriptionWithExpiry(ctx, userID, duration); err != nil {
		log.Error("Ошибка выдачи подписки с истечением", zap.Error(err))
		return err
	}

	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		log.Error("Не удалось получить пользователя после выдачи подписки", zap.Error(err), zap.Int("user_id", userID))
		return nil // подписка уже установлена — письмо необязательно
	}

	if u != nil && u.Email != "" && u.SubscriptionExpiresAt != nil {
		plan := humanizeDuration(duration)
		html := helpers.BuildSubscriptionGrantedHTML(u.FullName, plan, u.SubscriptionExpiresAt.Format("02.01.2006 15:04"))

		EmailQueue <- EmailJob{
			To:      []string{u.Email},
			Subject: "Подписка активирована",
			Body:    html,
			IsHTML:  true,
		}
	}

	log.Info("Подписка с истечением успешно установлена", zap.Int("user_id", userID))
	return nil
}

func (s *AuthService) ExtendSubscription(ctx context.Context, userID int, duration time.Duration) error {
	log := logger.WithCtx(ctx)
	log.Info("Продление подписки", zap.Int("user_id", userID), zap.Duration("duration", duration))

	if err := s.repo.ExtendSubscription(ctx, userID, duration); err != nil {
		log.Error("Ошибка продления подписки", zap.Error(err))
		return err
	}

	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		log.Error("Не удалось получить пользователя после продления", zap.Error(err), zap.Int("user_id", userID))
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
	}

	log.Info("Подписка продлена", zap.Int("user_id", userID))
	return nil
}

func (s *AuthService) findUserByIdentifier(ctx context.Context, identifier string) (*models.User, error) {
	log := logger.WithCtx(ctx)
	id := strings.TrimSpace(identifier)
	if id == "" {
		return nil, errors.New("пустой логин")
	}

	// email
	if strings.Contains(id, "@") {
		log.Debug("Поиск пользователя по email")
		return s.repo.GetUserByEmail(ctx, id)
	}

	// телефон — по последним 10 цифрам
	digits := normalizePhoneDigits(id)
	if len(digits) >= 10 {
		log.Debug("Поиск пользователя по телефону")
		return s.repo.GetUserByPhone(ctx, digits)
	}

	// username
	log.Debug("Поиск пользователя по username")
	return s.repo.GetByUsername(ctx, id)
}

func (s *AuthService) LoginUserByIdentifier(
	ctx context.Context,
	identifier, password, jwtSecret string,
	accessTTL, refreshTTL time.Duration,
) (string, string, *models.User, error) {
	log := logger.WithCtx(ctx)
	log.Info("Попытка входа (универсальный идентификатор)")

	user, err := s.findUserByIdentifier(ctx, identifier)
	if err != nil {
		log.Warn("Пользователь не найден по идентификатору", zap.String("identifier", identifier), zap.Error(err))
		return "", "", nil, errors.New("пользователь не найден")
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		log.Warn("Неверный пароль", zap.Int("user_id", user.ID))
		return "", "", nil, errors.New("неверный пароль")
	}

	accessToken, err := utils.GenerateToken(jwtSecret, user.ID, user.Role, accessTTL, "access")
	if err != nil {
		log.Error("Ошибка генерации access-токена", zap.Error(err))
		return "", "", nil, err
	}
	refreshToken, err := utils.GenerateToken(jwtSecret, user.ID, user.Role, refreshTTL, "refresh")
	if err != nil {
		log.Error("Ошибка генерации refresh-токена", zap.Error(err))
		return "", "", nil, err
	}

	if err := s.repo.SaveRefreshToken(ctx, user.ID, refreshToken); err != nil {
		log.Error("Ошибка сохранения refresh-токена", zap.Error(err))
		return "", "", nil, err
	}

	log.Info("Вход выполнен", zap.Int("user_id", user.ID), zap.String("role", user.Role))
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
	var b []rune
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

func (s *AuthService) GetUsersFiltered(ctx context.Context, limit, offset int, q string, role *string, hasSubscription *bool) ([]*models.User, int, error) {
	return s.repo.GetUsersFiltered(ctx, limit, offset, q, role, hasSubscription)
}
