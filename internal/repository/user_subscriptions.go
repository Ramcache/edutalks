package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
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

// Вариант 1 (простой): один флаг "email_subscribed" в таблице users
func (r *SubscriptionRepository) GetAllSubscribedEmails(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT email FROM users WHERE email_verified = TRUE AND email_subscription = TRUE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []string
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			return nil, err
		}
		emails = append(emails, e)
	}
	return emails, rows.Err()
}

/*
Вариант 2 (тонкие темы): отдельная таблица user_email_topics(user_id, topic)
и метод выборки по теме. Добавишь позже при желании.
*/
