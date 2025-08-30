package handlers

import (
	"edutalks/internal/services"
	helpers "edutalks/internal/utils/helpers"
	"net/http"
	"strings"
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
	query := r.URL.Query().Get("query")
	if strings.TrimSpace(query) == "" {
		helpers.Error(w, http.StatusBadRequest, "Пустой запрос")
		return
	}

	ctx := r.Context()

	newsResults, _ := h.newsService.Search(ctx, query)
	documentResults, _ := h.documentService.Search(ctx, query)

	results := map[string]interface{}{
		"news":      newsResults,
		"documents": documentResults,
	}

	helpers.JSON(w, http.StatusOK, results)
}
