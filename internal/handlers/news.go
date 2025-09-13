package handlers

import (
	"context"
	"crypto/rand"
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

	"edutalks/internal/logger"
	"edutalks/internal/models"
	"edutalks/internal/services"
	helpers "edutalks/internal/utils/helpers"

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
	log := logger.WithCtx(r.Context())
	var req createNewsRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("create news: невалидный JSON", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "Невалидный JSON")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	req.ImageURL = strings.TrimSpace(req.ImageURL)
	req.Color = strings.TrimSpace(req.Color)
	req.Sticker = strings.TrimSpace(req.Sticker)

	log.Info("create news: входные данные",
		zap.String("title", req.Title),
		zap.String("image_url", req.ImageURL),
		zap.String("color", req.Color),
		zap.String("sticker", req.Sticker),
	)

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
		log.Error("create news: ошибка сервиса", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Не удалось создать новость")
		return
	}

	ctx := context.WithoutCancel(r.Context())
	go h.notifier.NotifyNewsPublished(ctx, id, news.Title)

	log.Info("create news: новость создана", zap.Int("news_id", id))
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
	log := logger.WithCtx(r.Context())

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	log.Info("list news: параметры", zap.Int("page", page), zap.Int("page_size", pageSize), zap.Int("offset", offset))

	newsList, total, err := h.newsService.ListPaginated(r.Context(), pageSize, offset)
	if err != nil {
		log.Error("list news: ошибка сервиса", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка получения новостей")
		return
	}

	log.Info("list news: успех", zap.Int("returned", len(newsList)), zap.Int("total", total))
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
	log := logger.WithCtx(r.Context())

	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	log.Info("get news: вход", zap.Int("news_id", id))

	news, err := h.newsService.GetByID(r.Context(), id)
	if err != nil {
		log.Warn("get news: новость не найдена", zap.Int("news_id", id))
		helpers.Error(w, http.StatusNotFound, "Новость не найдена")
		return
	}

	log.Info("get news: успех", zap.Int("news_id", id))
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
	log := logger.WithCtx(r.Context())

	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	var req updateNewsRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("update news: невалидный JSON", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "Невалидный JSON")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	req.ImageURL = strings.TrimSpace(req.ImageURL)
	req.Color = strings.TrimSpace(req.Color)
	req.Sticker = strings.TrimSpace(req.Sticker)

	log.Info("update news: входные данные",
		zap.Int("news_id", id),
		zap.String("title", req.Title),
		zap.String("image_url", req.ImageURL),
		zap.String("color", req.Color),
		zap.String("sticker", req.Sticker),
	)

	if err := h.newsService.Update(r.Context(), id, req.Title, req.Content, req.ImageURL, req.Color, req.Sticker); err != nil {
		log.Error("update news: ошибка сервиса", zap.Error(err), zap.Int("news_id", id))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка обновления")
		return
	}

	log.Info("update news: успех", zap.Int("news_id", id))
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
	log := logger.WithCtx(r.Context())

	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	log.Info("delete news: вход", zap.Int("news_id", id))

	if err := h.newsService.Delete(r.Context(), id); err != nil {
		log.Error("delete news: ошибка сервиса", zap.Error(err), zap.Int("news_id", id))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка удаления")
		return
	}

	log.Info("delete news: успех", zap.Int("news_id", id))
	helpers.JSON(w, http.StatusOK, "Удалено")
}

func uploadsRoot() string {
	if v := os.Getenv("UPLOADS_DIR"); strings.TrimSpace(v) != "" {
		return v // например: /edu-talks/uploads
	}
	return "/edutalks/uploads"
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
	log := logger.WithCtx(r.Context())

	const maxUpload = 10 << 20 // 10 MiB
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)

	if err := r.ParseMultipartForm(maxUpload); err != nil {
		log.Warn("upload news image: multipart parse error", zap.Error(err))
		helpers.Error(w, http.StatusRequestEntityTooLarge, "файл слишком большой (макс 10 МБ)")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		log.Warn("upload news image: отсутствует поле file", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "поле file обязательно")
		return
	}
	defer file.Close()

	// определить content-type по содержимому
	sniff := make([]byte, 512)
	n, _ := file.Read(sniff)
	contentType := http.DetectContentType(sniff[:n])

	// допустимые типы -> расширения
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
			log.Warn("upload news image: недопустимый тип", zap.String("ctype", contentType), zap.String("filename", header.Filename))
			helpers.Error(w, http.StatusBadRequest, "допустимы только изображения: jpg, png, webp, gif")
			return
		}
	}

	root := uploadsRoot()              // абсолютный путь на диске
	dir := filepath.Join(root, "news") // /edutalks/uploads/news
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Error("upload news image: mkdir error", zap.Error(err), zap.String("dir", dir))
		helpers.Error(w, http.StatusInternalServerError, "не удалось создать директорию")
		return
	}

	name := fmt.Sprintf("%d_%s%s", time.Now().Unix(), randHex(6), ext)
	fullPath := filepath.Join(dir, name)

	dst, err := os.Create(fullPath)
	if err != nil {
		log.Error("upload news image: create dst error", zap.Error(err), zap.String("path", fullPath))
		helpers.Error(w, http.StatusInternalServerError, "ошибка сохранения файла")
		return
	}
	defer dst.Close()

	// дописываем уже прочитанные байты и остаток
	if n > 0 {
		if _, err := dst.Write(sniff[:n]); err != nil {
			log.Error("upload news image: запись первых байт не удалась", zap.Error(err))
			helpers.Error(w, http.StatusInternalServerError, "ошибка записи файла")
			return
		}
	}
	if _, err := io.Copy(dst, file); err != nil {
		log.Error("upload news image: запись остатка не удалась", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "ошибка записи файла")
		return
	}

	publicURL := "/uploads/news/" + name

	log.Info("upload news image: успех",
		zap.String("filename", header.Filename),
		zap.String("stored_name", name),
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
