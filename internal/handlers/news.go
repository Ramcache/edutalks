package handlers

import (
	"context"
	"edutalks/internal/models"
	"edutalks/internal/services"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type NewsHandler struct {
	newsService *services.NewsService
}

func NewNewsHandler(newsService *services.NewsService) *NewsHandler {
	return &NewsHandler{newsService: newsService}
}

type createNewsRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// CreateNews godoc
// @Summary Создать новость (только admin)
// @Tags news
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param input body createNewsRequest true "Данные новости"
// @Success 201 {string} string "Новость создана"
// @Failure 400 {string} string "Ошибка запроса"
// @Router /admin/news [post]
func (h *NewsHandler) CreateNews(w http.ResponseWriter, r *http.Request) {
	var req createNewsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Невалидный JSON", http.StatusBadRequest)
		return
	}

	news := &models.News{
		Title:     req.Title,
		Content:   req.Content,
		CreatedAt: time.Now(),
	}

	if err := h.newsService.Create(context.Background(), news); err != nil {
		http.Error(w, "Ошибка создания", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Новость создана"))
}

// ListNews godoc
// @Summary Получить список новостей
// @Tags news
// @Produce json
// @Success 200 {array} models.News
// @Router /news [get]
func (h *NewsHandler) ListNews(w http.ResponseWriter, r *http.Request) {
	newsList, err := h.newsService.List(r.Context())
	if err != nil {
		http.Error(w, "Ошибка получения новостей", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newsList)
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
	news, err := h.newsService.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Новость не найдена", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(news)
}

type updateNewsRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// UpdateNews godoc
// @Summary Обновить новость (только admin)
// @Tags admin
// @Security ApiKeyAuth
// @Param id path int true "ID новости"
// @Param input body updateNewsRequest true "Новое содержимое"
// @Success 200 {string} string "Обновлено"
// @Router /admin/news/{id} [patch]
func (h *NewsHandler) UpdateNews(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	var req updateNewsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Невалидный JSON", http.StatusBadRequest)
		return
	}

	if err := h.newsService.Update(r.Context(), id, req.Title, req.Content); err != nil {
		http.Error(w, "Ошибка обновления", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Обновлено"))
}

// DeleteNews godoc
// @Summary Удалить новость (только admin)
// @Tags admin
// @Security ApiKeyAuth
// @Param id path int true "ID новости"
// @Success 200 {string} string "Удалено"
// @Router /admin/news/{id} [delete]
func (h *NewsHandler) DeleteNews(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	if err := h.newsService.Delete(r.Context(), id); err != nil {
		http.Error(w, "Ошибка удаления", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Удалено"))
}
