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
	//log := logger.WithCtx(ctx)

	if exists, _ := s.repo.IsUsernameTaken(ctx, input.Username); exists {
		return errors.New("имя пользователя уже занято")
	}
	if exists, _ := s.repo.IsEmailTaken(ctx, input.Email); exists {
		return errors.New("адрес электронной почты уже зарегистрирован")
	}

	hashed, err := utils.HashPassword(plainPassword)
	if err != nil {
		return err
	}

	input.PasswordHash = hashed
	input.Role = "user"

	return s.repo.CreateUser(ctx, input)
}

func (s *AuthService) Logout(ctx context.Context, token string, exp time.Time) error {
	return s.repo.AddAccessTokenToBlacklist(ctx, token, exp)
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
	id := strings.TrimSpace(identifier)
	if id == "" {
		return nil, errors.New("пустой логин")
	}
	if strings.Contains(id, "@") {
		return s.repo.GetUserByEmail(ctx, id)
	}
	digits := normalizePhoneDigits(id)
	if len(digits) >= 10 {
		return s.repo.GetUserByPhone(ctx, digits)
	}
	return s.repo.GetByUsername(ctx, id)
}

func (s *AuthService) LoginUserByIdentifier(
	ctx context.Context,
	identifier, password, jwtSecret string,
	accessTTL time.Duration,
) (string, *models.User, error) {
	log := logger.WithCtx(ctx)
	log.Info("Попытка входа (только access)")

	user, err := s.findUserByIdentifier(ctx, identifier)
	if err != nil {
		return "", nil, errors.New("пользователь не найден")
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return "", nil, errors.New("неверный пароль")
	}

	accessToken, err := utils.GenerateToken(jwtSecret, user.ID, user.Role, accessTTL, "access")
	if err != nil {
		log.Error("Ошибка генерации access-токена", zap.Error(err))
		return "", nil, err
	}

	log.Info("Вход выполнен", zap.Int("user_id", user.ID))
	return accessToken, user, nil
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
