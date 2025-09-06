// internal/services/notifier.go
package services

import (
	"context"
	"edutalks/internal/logger"
	"edutalks/internal/repository"
	helpers "edutalks/internal/utils/helpers"
	"fmt"
	"go.uber.org/zap"
	"net/url"
	"strings"
)

type Notifier struct {
	subsRepo *repository.SubscriptionRepository
	taxRepo  *repository.TaxonomyRepo
	baseURL  string
	fromName string
}

func NewNotifier(
	subsRepo *repository.SubscriptionRepository,
	taxRepo *repository.TaxonomyRepo,
	baseURL, fromName string,
) *Notifier {
	return &Notifier{
		subsRepo: subsRepo,
		taxRepo:  taxRepo,
		baseURL:  strings.TrimRight(baseURL, "/"),
		fromName: fromName,
	}
}

func chunkStrings(all []string, n int) [][]string {
	if n <= 0 {
		n = 50
	}
	var out [][]string
	for i := 0; i < len(all); i += n {
		j := i + n
		if j > len(all) {
			j = len(all)
		}
		out = append(out, all[i:j])
	}
	return out
}

func (n *Notifier) sendToAll(ctx context.Context, subject, htmlBody string) {
	// важное: не завязываемся на HTTP-контекст
	ctx = context.WithoutCancel(ctx)

	emails, err := n.subsRepo.GetAllSubscribedEmails(ctx)
	if err != nil {
		logger.Log.Error("Не удалось получить список подписчиков", zap.Error(err))
		return
	}
	for _, batch := range chunkStrings(emails, 50) {
		EmailQueue <- EmailJob{
			To:      batch,
			Subject: subject,
			Body:    htmlBody,
			IsHTML:  true,
		}
	}
}

// ==== ПИСЬМА ====

func (n *Notifier) NotifyNewDocument(ctx context.Context, title string, tabsID *int) {
	ctx = context.WithoutCancel(ctx)

	base := strings.TrimRight(n.baseURL, "/")
	link := base + "/documents"

	if tabsID != nil {
		if slug, err := n.taxRepo.GetTabSlugByID(ctx, *tabsID); err == nil && slug != "" {
			link = base + "/" + url.PathEscape(slug)
		}
	}

	subject := "Новый документ на Edutalks"
	body := fmt.Sprintf(`
      <p style="font-size:16px;color:#222;margin:0 0 16px 0;"><strong>%s</strong></p>
      <p><a href="%s" style="display:inline-block;padding:12px 24px;background:#2d74da;color:#fff;text-decoration:none;border-radius:6px;font-weight:600;">Открыть документ</a></p>
      <p style="font-size:12px;color:#999;margin-top:16px;">Если кнопка не работает — скопируйте ссылку: %s</p>
    `, title, link, link)
	html := helpers.BuildSimpleHTML("Добавлен новый документ", body)

	n.sendToAll(ctx, subject, html)
}

// Новость опубликована
func (n *Notifier) NotifyNewsPublished(ctx context.Context, newsID int, title string) {
	link := fmt.Sprintf("%s/recomm/%d", n.baseURL, newsID)
	subject := "Новая новость на Edutalks"

	// Можно передать краткий контент вместо "" если он у тебя есть
	html := helpers.BuildNewsHTML(title, "", link)

	n.sendToAll(ctx, subject, html)
}

// Статья опубликована
func (n *Notifier) NotifyArticlePublished(ctx context.Context, articleID int, title string) {
	link := fmt.Sprintf("%s/zavuch/%d", n.baseURL, articleID)
	subject := "Новая статья на Edutalks"

	body := fmt.Sprintf(`
      <p style="font-size:16px;color:#222;margin:0 0 16px 0;"><strong>%s</strong></p>
      <p><a href="%s" style="display:inline-block;padding:12px 24px;background:#2d74da;color:#fff;text-decoration:none;border-radius:6px;font-weight:600;">Читать статью</a></p>
      <p style="font-size:12px;color:#999;margin-top:16px;">Если кнопка не работает — скопируйте ссылку: %s</p>
    `, title, link, link)
	html := helpers.BuildSimpleHTML("Новая статья", body)

	n.sendToAll(ctx, subject, html)
}
