package handlers

import (
	"edutalks/internal/logger"
	"edutalks/internal/models"
	"edutalks/internal/services"
	helpers "edutalks/internal/utils/helpres"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type NewsHandler struct {
	newsService *services.NewsService
}

func NewNewsHandler(newsService *services.NewsService) *NewsHandler {
	return &NewsHandler{newsService: newsService}
}

type createNewsRequest struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	ImageURL string `json:"image_url"`
}

type updateNewsRequest struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	ImageURL string `json:"image_url"`
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
		CreatedAt: time.Now(),
	}

	if err := h.newsService.Create(r.Context(), news); err != nil {
		logger.Log.Error("Ошибка создания новости", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка создания")
		return
	}

	logger.Log.Info("Новость успешно создана", zap.String("title", news.Title))
	helpers.JSON(w, http.StatusCreated, "Новость создана")
}

// ListNews godoc
// @Summary Получить список новостей
// @Tags news
// @Produce json
// @Success 200 {array} models.News
// @Router /news [get]
func (h *NewsHandler) ListNews(w http.ResponseWriter, r *http.Request) {
	logger.Log.Info("Запрос на получение списка новостей")
	newsList, err := h.newsService.List(r.Context())
	if err != nil {
		logger.Log.Error("Ошибка получения новостей", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка получения новостей")
		return
	}

	logger.Log.Info("Новости получены", zap.Int("count", len(newsList)))
	helpers.JSON(w, http.StatusOK, newsList)
}

// GetNews godoc
// @Summary Получить новость по ID
// @Tags news
// @Produce json
// @Param id path int true "ID новости"
// @Success 200 {object} models.News
// @Failure 404 {string} string "Не найдено"
// @Router /news/{id} [get]
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

	if err := h.newsService.Update(r.Context(), id, req.Title, req.Content, req.ImageURL); err != nil {
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
