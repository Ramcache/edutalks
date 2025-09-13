// internal/services/notifier.go
package services

import (
	"context"
	"edutalks/internal/logger"
	"edutalks/internal/repository"
	helpers "edutalks/internal/utils/helpers"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Notifier struct {
	subsRepo *repository.SubscriptionRepository
	taxRepo  *repository.TaxonomyRepo
	baseURL  string
	fromName string

	// — батч-уведомления —
	mu     sync.Mutex
	buffer []string
	once   sync.Once
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
	// не завязываемся на HTTP-контекст
	ctx = context.WithoutCancel(ctx)

	emails, err := n.subsRepo.GetAllSubscribedEmails(ctx)
	if err != nil {
		logger.Log.Error("Не удалось получить список подписчиков", zap.Error(err))
		return
	}
	if len(emails) == 0 {
		logger.Log.Debug("Список подписчиков пуст — рассылка пропущена")
		return
	}

	batches := chunkStrings(emails, 50)
	logger.Log.Info("Формирование рассылки",
		zap.Int("total_recipients", len(emails)),
		zap.Int("batches", len(batches)),
		zap.String("subject", subject),
	)

	for i, batch := range batches {
		logger.Log.Debug("Постановка батча писем в очередь",
			zap.Int("batch_index", i),
			zap.Int("batch_size", len(batch)),
		)
		EmailQueue <- EmailJob{
			To:      batch,
			Subject: subject,
			Body:    htmlBody,
			IsHTML:  true,
		}
	}
	logger.Log.Info("Рассылка поставлена в очередь",
		zap.Int("total_recipients", len(emails)),
		zap.Int("batches", len(batches)),
	)
}

// ==== ПИСЬМА ====

func (n *Notifier) NotifyNewDocument(ctx context.Context, title string, tabsID *int) {
	ctx = context.WithoutCancel(ctx)

	base := strings.TrimRight(n.baseURL, "/")
	link := base + "/documents" // fallback

	if tabsID != nil {
		if slug, err := n.taxRepo.GetTabSlugByID(ctx, *tabsID); err == nil && slug != "" {
			link = base + "/" + url.PathEscape(slug) // https://edutalks.ru/<slug>
		} else if err != nil {
			logger.Log.Warn("Не удалось получить slug вкладки для уведомления о документе", zap.Error(err), zap.Intp("tab_id", tabsID))
		}
	}

	logger.Log.Info("Уведомление: новый документ",
		zap.String("title", title),
		zap.String("link", link),
	)

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

	logger.Log.Info("Уведомление: опубликована новость",
		zap.Int("news_id", newsID),
		zap.String("title", title),
		zap.String("link", link),
	)

	subject := "Новая новость на Edutalks"
	html := helpers.BuildNewsHTML(title, "", link) // сюда можно передать краткий контент

	n.sendToAll(context.WithoutCancel(ctx), subject, html)
}

// Статья опубликована
func (n *Notifier) NotifyArticlePublished(ctx context.Context, articleID int, title string) {
	link := fmt.Sprintf("%s/zavuch/%d", n.baseURL, articleID)

	logger.Log.Info("Уведомление: опубликована статья",
		zap.Int("article_id", articleID),
		zap.String("title", title),
		zap.String("link", link),
	)

	body := fmt.Sprintf(`
      <p style="font-size:16px;color:#222;margin:0 0 16px 0;"><strong>%s</strong></p>
      <p><a href="%s" style="display:inline-block;padding:12px 24px;background:#2d74da;color:#fff;text-decoration:none;border-radius:6px;font-weight:600;">Читать статью</a></p>
      <p style="font-size:12px;color:#999;margin-top:16px;">Если кнопка не работает — скопируйте ссылку: %s</p>
    `, title, link, link)
	html := helpers.BuildSimpleHTML("Новая статья", body)

	n.sendToAll(context.WithoutCancel(ctx), "Новая статья на Edutalks", html)
}

// AddDocumentForBatch — добавляем документ в временный буфер для групповой рассылки
func (n *Notifier) AddDocumentForBatch(ctx context.Context, title string, tabsID *int) {
	base := strings.TrimRight(n.baseURL, "/")
	link := base + "/documents"
	if tabsID != nil {
		if slug, err := n.taxRepo.GetTabSlugByID(ctx, *tabsID); err == nil && slug != "" {
			link = base + "/" + url.PathEscape(slug)
		} else if err != nil {
			logger.Log.Warn("Не удалось получить slug вкладки (batch)", zap.Error(err), zap.Intp("tab_id", tabsID))
		}
	}

	item := fmt.Sprintf(`<li><a href="%s">%s</a></li>`, link, title)

	n.mu.Lock()
	n.buffer = append(n.buffer, item)
	size := len(n.buffer)
	n.mu.Unlock()

	logger.Log.Info("Документ добавлен в батч-буфер",
		zap.String("title", title),
		zap.String("link", link),
		zap.Int("buffer_size", size),
	)

	// запускаем воркер только один раз
	n.once.Do(func() {
		logger.Log.Info("Старт батч-воркера уведомлений документов")
		go n.startBatchWorker()
	})
}

func (n *Notifier) startBatchWorker() {
	ticker := time.NewTicker(10 * time.Minute) // период можно вынести в конфиг
	defer ticker.Stop()

	logger.Log.Info("Батч-воркер запущен", zap.String("period", "10m"))

	for range ticker.C {
		n.mu.Lock()
		if len(n.buffer) == 0 {
			n.mu.Unlock()
			logger.Log.Debug("Батч-тик: буфер пуст — рассылка пропущена")
			continue
		}

		items := make([]string, len(n.buffer))
		copy(items, n.buffer)
		n.buffer = nil
		n.mu.Unlock()

		body := "<p>За последние 10 минут добавлены документы:</p><ul>"
		body += strings.Join(items, "")
		body += "</ul>"

		logger.Log.Info("Флаш батча документов",
			zap.Int("items_count", len(items)),
		)

		html := helpers.BuildSimpleHTML("Новые документы на сайте", body)
		n.sendToAll(context.Background(), "Новые документы на Edutalks", html)

		logger.Log.Debug("Буфер батча очищен после отправки")
	}
}
