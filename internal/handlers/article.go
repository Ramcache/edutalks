package handlers

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"

	"edutalks/internal/logger"
	"edutalks/internal/models"
	"edutalks/internal/services"
	"edutalks/internal/utils/helpers"

	"go.uber.org/zap"
)

type ArticleHandler struct {
	svc services.ArticleService
}

func NewArticleHandler(svc services.ArticleService) *ArticleHandler {
	return &ArticleHandler{svc: svc}
}

// Preview
// @Summary      Предпросмотр статьи
// @Description  Возвращает очищенный HTML (без сохранения в БД)
// @Tags         articles
// @Accept       json
// @Produce      json
// @Param        body  body   map[string]string  true  "Сырый HTML статьи"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Router       /api/admin/articles/preview [post]
func (h *ArticleHandler) Preview(w http.ResponseWriter, r *http.Request) {
	type reqT struct {
		BodyHTML string `json:"bodyHtml"`
	}
	var req reqT
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Error("ошибка декодирования JSON при предпросмотре статьи", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	safe := h.svc.PreviewHTML(req.BodyHTML)

	logger.Log.Info("предпросмотр статьи успешно создан")

	helpers.JSON(w, http.StatusOK, map[string]string{"bodyHtml": safe})
}

// Create
// @Summary      Создать статью
// @Description  Создаёт новую статью (как в Хабре/Вики). Поддерживает до 5 тегов.
// @Tags         articles
// @Accept       json
// @Produce      json
// @Param        body  body   models.CreateArticleRequest  true  "Данные статьи"
// @Success      201   {object}  models.Article
// @Failure      400   {object}  map[string]string
// @Security     BearerAuth
// @Router       /api/admin/articles [post]
func (h *ArticleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Error("ошибка декодирования JSON при создании статьи", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "invalid json")
		return
	}

	authorID := authorIDFromCtx(r.Context()) // берём user_id из JWT (если есть)
	article, err := h.svc.Create(r.Context(), authorID, req)
	if err != nil {
		logger.Log.Error("ошибка создания статьи", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	logger.Log.Info("статья успешно создана",
		zap.Int64("id", article.ID),
		zap.String("title", article.Title),
	)

	helpers.JSON(w, http.StatusCreated, article)
}

// GetAll
// @Summary      Получить список статей
// @Tags         articles
// @Produce      json
// @Param        limit     query int    false "Количество" default(20)
// @Param        offset    query int    false "Смещение"   default(0)
// @Param        tag       query string false "Фильтр по тегу"
// @Param        published query bool   false "Только опубликованные"
// @Success      200 {array} models.Article
// @Failure      500 {object} map[string]string
// @Router       /api/articles [get]
func (h *ArticleHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	limit := parseIntQuery(r, "limit", 20)
	offset := parseIntQuery(r, "offset", 0)
	tag := r.URL.Query().Get("tag")
	onlyPublished := r.URL.Query().Get("published") == "true"

	list, err := h.svc.GetAll(r.Context(), limit, offset, tag, onlyPublished)
	if err != nil {
		logger.Log.Error("ошибка получения списка статей", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	helpers.JSON(w, http.StatusOK, list)
}

// GetByID
// @Summary      Получить статью по ID
// @Tags         articles
// @Produce      json
// @Param        id   path int true "ID статьи"
// @Success      200 {object} models.Article
// @Failure      404 {object} map[string]string
// @Router       /api/articles/{id} [get]
func (h *ArticleHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	aid, _ := strconv.ParseInt(id, 10, 64)

	a, err := h.svc.GetByID(r.Context(), aid)
	if err != nil {
		logger.Log.Error("статья не найдена", zap.Error(err))
		helpers.Error(w, http.StatusNotFound, "not found")
		return
	}
	helpers.JSON(w, http.StatusOK, a)
}

// Update
// @Summary      Обновить статью
// @Tags         articles
// @Accept       json
// @Produce      json
// @Param        id   path int true "ID статьи"
// @Param        body body models.CreateArticleRequest true "Данные статьи"
// @Success      200 {object} models.Article
// @Failure      400 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/admin/articles/{id} [patch]
func (h *ArticleHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	aid, _ := strconv.ParseInt(id, 10, 64)

	var req models.CreateArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Error("ошибка JSON при обновлении статьи", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "invalid json")
		return
	}

	article, err := h.svc.Update(r.Context(), aid, req)
	if err != nil {
		logger.Log.Error("ошибка обновления статьи", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "update failed")
		return
	}
	helpers.JSON(w, http.StatusOK, article)
}

// Delete
// @Summary      Удалить статью
// @Tags         articles
// @Produce      json
// @Param        id   path int true "ID статьи"
// @Success      204 {string} string "no content"
// @Failure      404 {object} map[string]string
// @Security     BearerAuth
// @Router       /api/admin/articles/{id} [delete]
func (h *ArticleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	aid, _ := strconv.ParseInt(id, 10, 64)

	if err := h.svc.Delete(r.Context(), aid); err != nil {
		logger.Log.Error("ошибка удаления статьи", zap.Error(err))
		helpers.Error(w, http.StatusNotFound, "not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---
type ctxKey string

const userIDKey ctxKey = "user_id"

func authorIDFromCtx(ctx interface{ Value(any) any }) *int64 {
	if v := ctx.Value(userIDKey); v != nil {
		switch x := v.(type) {
		case int64:
			return &x
		case int:
			y := int64(x)
			return &y
		}
	}
	return nil
}

// parseIntQuery — хелпер для limit/offset
func parseIntQuery(r *http.Request, key string, def int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return def
	}
	if n, err := strconv.Atoi(val); err == nil {
		return n
	}
	return def
}
