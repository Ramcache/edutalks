package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"edutalks/internal/logger"
	"edutalks/internal/models"
	"edutalks/internal/services"
	"edutalks/internal/utils/helpers"
)

type ArticleHandler struct {
	svc      services.ArticleService
	notifier *services.Notifier
}

func NewArticleHandler(svc services.ArticleService, notifier *services.Notifier) *ArticleHandler {
	return &ArticleHandler{svc: svc, notifier: notifier}
}

// Preview
// @Summary     Предпросмотр статьи
// @Tags        articles
// @Accept      json
// @Produce     json
// @Param       body body map[string]string true "Сырый HTML статьи"
// @Success     200 {object} map[string]string
// @Failure     400 {object} map[string]string
// @Router      /api/admin/articles/preview [post]
func (h *ArticleHandler) Preview(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	var req struct {
		BodyHTML string `json:"bodyHtml"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("Невалидный JSON при предпросмотре статьи", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "invalid json")
		return
	}

	safe := h.svc.PreviewHTML(req.BodyHTML)
	log.Info("Предпросмотр статьи создан")

	helpers.JSON(w, http.StatusOK, map[string]string{"bodyHtml": safe})
}

// Create
// @Summary     Создать статью
// @Tags        articles
// @Accept      json,mpfd,x-www-form-urlencoded
// @Produce     json
// @Param       body body models.CreateArticleRequest true "Данные статьи"
// @Success     201 {object} models.Article
// @Failure     400 {object} map[string]string
// @Security    BearerAuth
// @Router      /api/admin/articles [post]
func (h *ArticleHandler) Create(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	req, err := readCreateArticleRequest(r)
	if err != nil {
		log.Warn("Невалидный payload при создании статьи", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "invalid payload")
		return
	}

	authorID := authorIDFromCtx(r.Context())
	log.Info("Запрос на создание статьи",
		zap.String("title", req.Title),
		zap.Bool("publish", req.Publish),
	)

	article, err := h.svc.Create(r.Context(), authorID, req)
	if err != nil {
		log.Error("Ошибка создания статьи", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	log.Info("Статья создана",
		zap.Int64("id", article.ID),
		zap.Bool("published", article.IsPublished),
	)

	ctx := context.WithoutCancel(r.Context())
	go h.notifier.NotifyArticlePublished(ctx, int(article.ID), article.Title)

	helpers.JSON(w, http.StatusCreated, article)
}

// GetAll
// @Summary     Список статей
// @Tags        articles
// @Produce     json
// @Param       page query int false "Номер страницы"
// @Param       page_size query int false "Размер страницы"
// @Success     200 {array} models.Article
// @Failure     500 {object} map[string]string
// @Router      /api/articles [get]
func (h *ArticleHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	limit := parseIntQuery(r, "limit", 20)
	offset := parseIntQuery(r, "offset", 0)
	tag := r.URL.Query().Get("tag")
	onlyPublished := r.URL.Query().Get("published") == "true"

	log.Info("Запрос списка статей",
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.String("tag", tag),
		zap.Bool("only_published", onlyPublished),
	)

	list, err := h.svc.GetAll(r.Context(), limit, offset, tag, onlyPublished)
	if err != nil {
		log.Error("Ошибка получения статей", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "internal error")
		return
	}

	log.Info("Список статей получен", zap.Int("count", len(list)))
	helpers.JSON(w, http.StatusOK, list)
}

// GetByID
// @Summary     Получить статью по ID
// @Tags        articles
// @Produce     json
// @Param       id path int true "ID статьи"
// @Success     200 {object} models.Article
// @Failure     404 {object} map[string]string
// @Router      /api/articles/{id} [get]
func (h *ArticleHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	id := mux.Vars(r)["id"]
	aid, _ := strconv.ParseInt(id, 10, 64)

	log.Info("Запрос статьи по ID", zap.Int64("id", aid))

	a, err := h.svc.GetByID(r.Context(), aid)
	if err != nil {
		log.Warn("Статья не найдена", zap.Int64("id", aid))
		helpers.Error(w, http.StatusNotFound, "not found")
		return
	}

	log.Info("Статья получена", zap.Int64("id", aid))
	helpers.JSON(w, http.StatusOK, a)
}

// Update
// @Summary     Обновить статью
// @Tags        articles
// @Accept      json,mpfd,x-www-form-urlencoded
// @Produce     json
// @Param       id path int true "ID статьи"
// @Param       body body models.CreateArticleRequest true "Данные статьи"
// @Success     200 {object} models.Article
// @Failure     400 {object} map[string]string
// @Router      /api/admin/articles/{id} [patch]
func (h *ArticleHandler) Update(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	id := mux.Vars(r)["id"]
	aid, _ := strconv.ParseInt(id, 10, 64)

	req, err := readCreateArticleRequest(r)
	if err != nil {
		log.Warn("Невалидный payload при обновлении статьи", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "invalid payload")
		return
	}

	log.Info("Запрос на обновление статьи", zap.Int64("id", aid), zap.String("title", req.Title))

	article, err := h.svc.Update(r.Context(), aid, req)
	if err != nil {
		log.Error("Ошибка обновления статьи", zap.Int64("id", aid), zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "update failed")
		return
	}

	log.Info("Статья обновлена", zap.Int64("id", aid))
	helpers.JSON(w, http.StatusOK, article)
}

// Delete
// @Summary     Удалить статью
// @Tags        articles
// @Produce     json
// @Param       id path int true "ID статьи"
// @Success     204 {string} string "no content"
// @Failure     404 {object} map[string]string
// @Router      /api/admin/articles/{id} [delete]
func (h *ArticleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	id := mux.Vars(r)["id"]
	aid, _ := strconv.ParseInt(id, 10, 64)

	log.Info("Запрос на удаление статьи", zap.Int64("id", aid))

	if err := h.svc.Delete(r.Context(), aid); err != nil {
		log.Error("Ошибка удаления статьи", zap.Int64("id", aid), zap.Error(err))
		helpers.Error(w, http.StatusNotFound, "not found")
		return
	}

	log.Info("Статья удалена", zap.Int64("id", aid))
	w.WriteHeader(http.StatusNoContent)
}

// SetPublish
// @Summary     Установить публикацию статьи
// @Tags        articles
// @Accept      json
// @Produce     json
// @Param       id path int true "ID статьи"
// @Param       body body SetPublishBody true "Флаг публикации"
// @Success     200 {object} models.Article
// @Failure     400 {object} map[string]string
// @Failure     404 {object} map[string]string
// @Router      /api/admin/articles/{id}/publish [patch]
func (h *ArticleHandler) SetPublish(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	aid, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil || aid <= 0 {
		log.Warn("Невалидный ID при SetPublish", zap.String("raw", mux.Vars(r)["id"]))
		helpers.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body SetPublishBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Publish == nil {
		log.Warn("Невалидный payload при SetPublish", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "invalid payload")
		return
	}

	log.Info("Запрос на изменение публикации", zap.Int64("id", aid), zap.Bool("publish", *body.Publish))

	article, err := h.svc.SetPublish(r.Context(), aid, *body.Publish)
	if err != nil {
		log.Error("Ошибка при SetPublish", zap.Int64("id", aid), zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	log.Info("Публикация изменена", zap.Int64("id", aid), zap.Bool("publish", *body.Publish))
	helpers.JSON(w, http.StatusOK, article)
}

type SetPublishBody struct {
	Publish *bool `json:"publish"`
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

	if req.IsPublished != nil {
		req.Publish = *req.IsPublished
	}
	return req, nil
}

func fillFromForm(req *models.CreateArticleRequest, r *http.Request) {
	req.Title = r.FormValue("title")
	req.Summary = r.FormValue("summary")
	req.BodyHTML = r.FormValue("bodyHtml")

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

	pub := firstNonEmpty(r.FormValue("publish"), r.FormValue("isPublished"))
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
