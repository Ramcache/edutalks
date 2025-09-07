package handlers

import (
	"context"
	"crypto/rand"
	"edutalks/internal/logger"
	"edutalks/internal/models"
	"edutalks/internal/services"
	helpers "edutalks/internal/utils/helpers"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type NewsHandler struct {
	newsService *services.NewsService
	notifier    *services.Notifier
}

func NewNewsHandler(newsService *services.NewsService, notifier *services.Notifier) *NewsHandler {
	return &NewsHandler{newsService: newsService, notifier: notifier}
}

type createNewsRequest struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	ImageURL string `json:"image_url"`
	Color    string `json:"color"`
	Sticker  string `json:"sticker"`
}

type updateNewsRequest struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	ImageURL string `json:"image_url"`
	Color    string `json:"color"`
	Sticker  string `json:"sticker"`
}

// CreateNews godoc
// @Summary Создать новость (только admin)
// @Tags admin-news
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param input body createNewsRequest true "Данные новости"
// @Success 201 {string} string "Новость создана"
// @Failure 400 {string} string "Ошибка запроса"
// @Router /api/admin/news [post]
func (h *NewsHandler) CreateNews(w http.ResponseWriter, r *http.Request) {
	logger.Log.Info("Запрос на создание новости")
	var req createNewsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Warn("Невалидный JSON при создании новости", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "Невалидный JSON")
		return
	}

	news := &models.News{
		Title:     req.Title,
		Content:   req.Content,
		ImageURL:  req.ImageURL,
		Color:     req.Color,
		Sticker:   req.Sticker,
		CreatedAt: time.Now(),
	}

	id, err := h.newsService.Create(r.Context(), news)
	if err != nil {
		helpers.Error(w, http.StatusInternalServerError, "Не удалось создать новость")
		return
	}

	ctx := context.WithoutCancel(r.Context())
	go h.notifier.NotifyNewsPublished(ctx, id, news.Title)

	helpers.JSON(w, http.StatusCreated, map[string]any{
		"message": "Новость создана",
		"id":      id,
	})

}

// ListNews godoc
// @Summary Получить список новостей
// @Tags news
// @Produce json
// @Param page query int false "Номер страницы (начиная с 1)"
// @Param page_size query int false "Размер страницы"
// @Success 200 {array} models.News
// @Router /api/news [get]
func (h *NewsHandler) ListNews(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	newsList, total, err := h.newsService.ListPaginated(r.Context(), pageSize, offset)
	if err != nil {
		logger.Log.Error("Ошибка получения новостей", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка получения новостей")
		return
	}

	helpers.JSON(w, http.StatusOK, map[string]interface{}{
		"data":      newsList,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetNews godoc
// @Summary Получить новость по ID
// @Tags news
// @Produce json
// @Param id path int true "ID новости"
// @Success 200 {object} models.News
// @Failure 404 {string} string "Не найдено"
// @Router /api/news/{id} [get]
func (h *NewsHandler) GetNews(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	logger.Log.Info("Запрос на получение новости по ID", zap.Int("news_id", id))
	news, err := h.newsService.GetByID(r.Context(), id)
	if err != nil {
		logger.Log.Warn("Новость не найдена", zap.Int("news_id", id))
		helpers.Error(w, http.StatusNotFound, "Новость не найдена")
		return
	}

	logger.Log.Info("Новость получена", zap.Int("news_id", id))
	helpers.JSON(w, http.StatusOK, news)
}

// UpdateNews godoc
// @Summary Обновить новость (только admin)
// @Tags admin-news
// @Security ApiKeyAuth
// @Param id path int true "ID новости"
// @Param input body updateNewsRequest true "Новое содержимое"
// @Success 200 {string} string "Обновлено"
// @Router /api/admin/news/{id} [patch]
func (h *NewsHandler) UpdateNews(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	logger.Log.Info("Запрос на обновление новости", zap.Int("news_id", id))
	var req updateNewsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Warn("Невалидный JSON при обновлении новости", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "Невалидный JSON")
		return
	}

	if err := h.newsService.Update(r.Context(), id, req.Title, req.Content, req.ImageURL, req.Color, req.Sticker); err != nil {
		logger.Log.Error("Ошибка обновления новости", zap.Error(err), zap.Int("news_id", id))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка обновления")
		return
	}

	logger.Log.Info("Новость успешно обновлена", zap.Int("news_id", id))
	helpers.JSON(w, http.StatusOK, "Обновлено")
}

// DeleteNews godoc
// @Summary Удалить новость (только admin)
// @Tags admin-news
// @Security ApiKeyAuth
// @Param id path int true "ID новости"
// @Success 200 {string} string "Удалено"
// @Router /api/admin/news/{id} [delete]
func (h *NewsHandler) DeleteNews(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	logger.Log.Info("Запрос на удаление новости", zap.Int("news_id", id))
	if err := h.newsService.Delete(r.Context(), id); err != nil {
		logger.Log.Error("Ошибка удаления новости", zap.Error(err), zap.Int("news_id", id))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка удаления")
		return
	}

	logger.Log.Info("Новость успешно удалена", zap.Int("news_id", id))
	helpers.JSON(w, http.StatusOK, "Удалено")
}

func uploadsRoot() string {
	if v := os.Getenv("UPLOADS_DIR"); strings.TrimSpace(v) != "" {
		return v // например: /edu-talks/uploads
	}
	return "/edutalks/uploads"
}

// helper для мапы allowed
func allowed(ct string) (string, bool) {
	switch ct {
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/webp":
		return ".webp", true
	case "image/gif":
		return ".gif", true
	default:
		return "", false
	}
}

// UploadNewsImage godoc
// @Summary Загрузка изображения для новости
// @Tags news
// @Accept mpfd
// @Produce json
// @Param file formData file true "Файл изображения (jpeg/png/webp/gif)"
// @Success 201 {object} map[string]string "url: публичная ссылка"
// @Failure 400 {object} map[string]string
// @Failure 413 {object} map[string]string
// @Security ApiKeyAuth
// @Router /api/admin/news/upload [post]
func (h *NewsHandler) UploadNewsImage(w http.ResponseWriter, r *http.Request) {
	const maxUpload = 10 << 20 // 10 MiB
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)

	if err := r.ParseMultipartForm(maxUpload); err != nil {
		logger.Log.Warn("multipart parse error", zap.Error(err))
		helpers.Error(w, http.StatusRequestEntityTooLarge, "файл слишком большой (макс 10 МБ)")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		helpers.Error(w, http.StatusBadRequest, "поле file обязательно")
		return
	}
	defer file.Close()

	// определить content-type по содержимому
	sniff := make([]byte, 512)
	n, _ := file.Read(sniff)
	contentType := http.DetectContentType(sniff[:n])

	// допустимые типы -> расширения (ПЕРЕИМЕНОВАЛ!)
	allowedTypes := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/webp": ".webp",
		"image/gif":  ".gif",
	}

	ext, ok := allowedTypes[contentType]
	if !ok {
		// fallback по имени файла
		ext = strings.ToLower(filepath.Ext(header.Filename))
		if ext == ".jpeg" {
			ext = ".jpg"
		}
		if _, ok := map[string]struct{}{".jpg": {}, ".png": {}, ".webp": {}, ".gif": {}}[ext]; !ok {
			helpers.Error(w, http.StatusBadRequest, "допустимы только изображения: jpg, png, webp, gif")
			return
		}
	}

	// абсолютный путь на диске
	root := uploadsRoot()              // /edu-talks/uploads или из ENV
	dir := filepath.Join(root, "news") // /edu-talks/uploads/news
	if err := os.MkdirAll(dir, 0o755); err != nil {
		logger.Log.Error("mkdir uploads/news", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "не удалось создать директорию")
		return
	}

	name := fmt.Sprintf("%d_%s%s", time.Now().Unix(), randHex(6), ext)
	fullPath := filepath.Join(dir, name)

	dst, err := os.Create(fullPath)
	if err != nil {
		logger.Log.Error("create dst", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "ошибка сохранения файла")
		return
	}
	defer dst.Close()

	// дописываем уже прочитанные байты и остаток
	if n > 0 {
		if _, err := dst.Write(sniff[:n]); err != nil {
			helpers.Error(w, http.StatusInternalServerError, "ошибка записи файла")
			return
		}
	}
	if _, err := io.Copy(dst, file); err != nil {
		helpers.Error(w, http.StatusInternalServerError, "ошибка записи файла")
		return
	}

	publicURL := "/uploads/news/" + name

	logger.Log.Info("news image uploaded",
		zap.String("name", name),
		zap.String("ctype", contentType),
		zap.String("path", fullPath),
		zap.String("url", publicURL),
	)

	helpers.JSON(w, http.StatusCreated, map[string]string{"url": publicURL})
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
