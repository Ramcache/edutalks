package handlers

import (
	"encoding/json"
	"net/http"

	"edutalks/internal/logger"
	"edutalks/internal/models"
	"edutalks/internal/services"

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
// @Router       /api/v1/articles/preview [post]
func (h *ArticleHandler) Preview(w http.ResponseWriter, r *http.Request) {
	type reqT struct {
		BodyHTML string `json:"bodyHtml"`
	}
	var req reqT
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Error("ошибка декодирования JSON при предпросмотре статьи", zap.Error(err))
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	safe := h.svc.PreviewHTML(req.BodyHTML)

	logger.Log.Info("предпросмотр статьи успешно создан")

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"bodyHtml": safe})
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
// @Router       /api/v1/articles [post]
func (h *ArticleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateArticleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.Error("ошибка декодирования JSON при создании статьи", zap.Error(err))
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	authorID := authorIDFromCtx(r.Context()) // берём user_id из JWT (если есть)
	article, err := h.svc.Create(r.Context(), authorID, req)
	if err != nil {
		logger.Log.Error("ошибка создания статьи", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	logger.Log.Info("статья успешно создана",
		zap.Int64("id", article.ID),
		zap.String("title", article.Title),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(article)
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
