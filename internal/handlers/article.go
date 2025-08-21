package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
	"strings"

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
// @Description  Поддерживает JSON и form-data. Поле публикации: `publish` (также принимается `isPublished`).
// @Tags         articles
// @Accept       json
// @Accept       mpfd
// @Accept       x-www-form-urlencoded
// @Produce      json
// @Param        body  body   models.CreateArticleRequest  true  "Данные статьи"
// @Success      201   {object}  models.Article
// @Failure      400   {object}  map[string]string
// @Security     BearerAuth
// @Router       /api/admin/articles [post]
func (h *ArticleHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, err := readCreateArticleRequest(r)
	if err != nil {
		logger.Log.Error("ошибка чтения тела при создании статьи", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "invalid payload")
		return
	}

	authorID := authorIDFromCtx(r.Context())
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
// @Description  Поддерживает JSON и form-data. Поле публикации: `publish` (также принимается `isPublished`).
// @Tags         articles
// @Accept       json
// @Accept       mpfd
// @Accept       x-www-form-urlencoded
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

	req, err := readCreateArticleRequest(r)
	if err != nil {
		logger.Log.Error("ошибка чтения тела при обновлении статьи", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "invalid payload")
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

// readCreateArticleRequest читает JSON или form-data и принимает оба поля publish/isPublished.
func readCreateArticleRequest(r *http.Request) (models.CreateArticleRequest, error) {
	ct := r.Header.Get("Content-Type")
	var req models.CreateArticleRequest

	switch {
	case ct == "" || strings.HasPrefix(ct, "application/json"):
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return req, fmt.Errorf("invalid json: %w", err)
		}

	case strings.HasPrefix(ct, "multipart/form-data"):
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			return req, fmt.Errorf("invalid multipart: %w", err)
		}
		fillFromForm(&req, r)

	case strings.HasPrefix(ct, "application/x-www-form-urlencoded"):
		if err := r.ParseForm(); err != nil {
			return req, fmt.Errorf("invalid form: %w", err)
		}
		fillFromForm(&req, r)

	default:
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return req, fmt.Errorf("unsupported content-type: %s", ct)
		}
	}

	// Алиас: если фронт прислал isPublished, используем его.
	if req.IsPublished != nil {
		req.Publish = *req.IsPublished
	}
	return req, nil
}

func fillFromForm(req *models.CreateArticleRequest, r *http.Request) {
	req.Title = r.FormValue("title")
	req.Summary = r.FormValue("summary")
	req.BodyHTML = r.FormValue("bodyHtml")

	// Теги: tags[]=a&tags[]=b ИЛИ tags="a,b"
	tags := r.Form["tags[]"]
	if len(tags) == 0 {
		if raw := r.FormValue("tags"); raw != "" {
			for _, t := range strings.Split(raw, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
		}
	}
	req.Tags = tags

	// publish / isPublished
	pub := firstNonEmpty(
		r.FormValue("publish"),
		r.FormValue("isPublished"),
	)
	pub = strings.ToLower(strings.TrimSpace(pub))
	req.Publish = pub == "true" || pub == "1" || pub == "on"
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
