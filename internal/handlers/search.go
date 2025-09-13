package handlers

import (
	"net/http"
	"strings"
	"time"

	"edutalks/internal/logger"
	"edutalks/internal/services"
	helpers "edutalks/internal/utils/helpers"

	"go.uber.org/zap"
)

type SearchHandler struct {
	newsService     *services.NewsService
	documentService *services.DocumentService
}

func NewSearchHandler(
	newsSvc *services.NewsService,
	documentSvc *services.DocumentService,
) *SearchHandler {
	return &SearchHandler{
		newsService:     newsSvc,
		documentService: documentSvc,
	}
}

// GlobalSearch godoc
// @Summary Глобальный поиск по материалам
// @Tags search
// @Produce json
// @Param query query string true "Поисковый запрос"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {string} string "Пустой запрос"
// @Router /api/search [get]
func (h *SearchHandler) GlobalSearch(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		log.Warn("search: пустой запрос")
		helpers.Error(w, http.StatusBadRequest, "Пустой запрос")
		return
	}

	start := time.Now()
	log.Info("search: старт", zap.String("query", query))

	newsResults, errNews := h.newsService.Search(r.Context(), query)
	if errNews != nil {
		log.Error("search: ошибка поиска по новостям", zap.Error(errNews))
	}

	documentResults, errDocs := h.documentService.Search(r.Context(), query)
	if errDocs != nil {
		log.Error("search: ошибка поиска по документам", zap.Error(errDocs))
	}

	elapsed := time.Since(start)
	log.Info("search: готово",
		zap.String("query", query),
		zap.Int("news_count", len(newsResults)),
		zap.Int("documents_count", len(documentResults)),
		zap.Duration("elapsed", elapsed),
	)

	results := map[string]interface{}{
		"news":      newsResults,
		"documents": documentResults,
	}

	helpers.JSON(w, http.StatusOK, results)
}
