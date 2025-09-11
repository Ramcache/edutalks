package models

type SystemStats struct {
	TotalUsers          int `json:"total_users"`
	Admins              int `json:"admins"`
	RegularUsers        int `json:"regular_users"`
	WithSubscription    int `json:"with_subscription"`
	WithoutSubscription int `json:"without_subscription"`

	NewsCount      int `json:"news_count"`
	DocumentsCount int `json:"documents_count"`
	ArticlesCount  int `json:"articles_count"`

	WithSubscriptionPct    int `json:"with_subscription_pct"`
	WithoutSubscriptionPct int `json:"without_subscription_pct"`
}
