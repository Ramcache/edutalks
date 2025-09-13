package repository

import (
	"context"

	"edutalks/internal/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type SubscriptionTopic string

const (
	TopicAll       SubscriptionTopic = "all"
	TopicNews      SubscriptionTopic = "news"
	TopicArticles  SubscriptionTopic = "articles"
	TopicDocuments SubscriptionTopic = "documents"
)

type SubscriptionRepository struct {
	db *pgxpool.Pool
}

func NewSubscriptionRepository(db *pgxpool.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// GetAllSubscribedEmails — простой вариант: один флаг в users.email_subscription (+ email_verified)
// выдержан в общем стиле логирования с ReqID/UserID из контекста.
func (r *SubscriptionRepository) GetAllSubscribedEmails(ctx context.Context) ([]string, error) {
	log := logger.WithCtx(ctx)

	const q = `SELECT email FROM users WHERE email_verified = TRUE AND email_subscription = TRUE`

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		log.Error("subscription repo: query subscribed emails failed", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	emails := make([]string, 0, 128)
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			log.Error("subscription repo: scan email failed", zap.Error(err))
			return nil, err
		}
		emails = append(emails, e)
	}
	if err := rows.Err(); err != nil {
		log.Error("subscription repo: rows error subscribed emails", zap.Error(err))
		return nil, err
	}

	log.Debug("subscription repo: got subscribed emails", zap.Int("count", len(emails)))
	return emails, nil
}

/*
Вариант 2 (тонкие темы): отдельная таблица user_email_topics(user_id, topic)
и метод выборки по теме. Добавишь позже при желании.
*/
