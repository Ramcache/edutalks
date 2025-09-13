package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"edutalks/internal/logger"
	"edutalks/internal/models"
	"edutalks/internal/services"
	helpers "edutalks/internal/utils/helpers"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type TaxonomyHandler struct{ svc *services.TaxonomyService }

func NewTaxonomyHandler(s *services.TaxonomyService) *TaxonomyHandler {
	return &TaxonomyHandler{svc: s}
}

// PublicTree
// @Summary      Получить дерево вкладок и разделов
// @Description  Возвращает список вкладок с разделами и количеством документов в каждом разделе
// @Tags         taxonomy
// @Produce      json
// @Success      200 {object} map[string][]models.TabTree
// @Failure      500 {object} map[string]string
// @Router       /api/taxonomy/tree [get]
func (h *TaxonomyHandler) PublicTree(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())
	log.Info("taxonomy: запрос дерева вкладок и разделов")

	tree, err := h.svc.PublicTree(r.Context())
	if err != nil {
		log.Error("taxonomy: ошибка получения дерева", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Info("taxonomy: дерево получено", zap.Int("tabs_count", len(tree)))
	helpers.JSON(w, http.StatusOK, map[string]any{"data": tree})
}

// CreateTab
// @Summary      Создать вкладку
// @Description  Доступно только администратору
// @Tags         taxonomy
// @Accept       json
// @Produce      json
// @Param        body  body  models.Tab  true  "Данные вкладки"
// @Success      201   {object} map[string]int
// @Failure      400   {object} map[string]string
// @Failure      500   {object} map[string]string
// @Router       /api/admin/tabs [post]
func (h *TaxonomyHandler) CreateTab(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	var req models.Tab
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("taxonomy: невалидный JSON при создании вкладки", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "bad json")
		return
	}

	log.Info("taxonomy: создание вкладки", zap.String("title", req.Title), zap.String("slug", req.Slug))

	id, err := h.svc.CreateTab(r.Context(), &req)
	if err != nil {
		log.Error("taxonomy: ошибка создания вкладки", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Info("taxonomy: вкладка создана", zap.Int("id", id))
	helpers.JSON(w, http.StatusCreated, map[string]int{"id": id})
}

// UpdateTab
// @Summary      Обновить вкладку
// @Description  Доступно только администратору
// @Tags         taxonomy
// @Accept       json
// @Produce      json
// @Param        id    path  int        true  "ID вкладки"
// @Param        body  body  models.Tab true  "Обновлённые данные"
// @Success      204   {string} string  "No Content"
// @Failure      400   {object} map[string]string
// @Failure      500   {object} map[string]string
// @Router       /api/admin/tabs/{id} [patch]
func (h *TaxonomyHandler) UpdateTab(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		log.Warn("taxonomy: неверный id вкладки при обновлении", zap.String("raw", idStr))
		helpers.Error(w, http.StatusBadRequest, "bad id")
		return
	}

	var req models.Tab
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("taxonomy: невалидный JSON при обновлении вкладки", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "bad json")
		return
	}
	req.ID = id

	log.Info("taxonomy: обновление вкладки", zap.Int("id", id), zap.String("title", req.Title), zap.String("slug", req.Slug))

	if err := h.svc.UpdateTab(r.Context(), &req); err != nil {
		log.Error("taxonomy: ошибка обновления вкладки", zap.Error(err), zap.Int("id", id))
		helpers.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Info("taxonomy: вкладка обновлена", zap.Int("id", id))
	w.WriteHeader(http.StatusNoContent)
}

// DeleteTab
// @Summary      Удалить вкладку
// @Description  Доступно только администратору
// @Tags         taxonomy
// @Param        id  path  int  true  "ID вкладки"
// @Success      204 {string} string "No Content"
// @Failure      500 {object} map[string]string
// @Router       /api/admin/tabs/{id} [delete]
func (h *TaxonomyHandler) DeleteTab(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		log.Warn("taxonomy: неверный id вкладки при удалении", zap.String("raw", idStr))
		helpers.Error(w, http.StatusBadRequest, "bad id")
		return
	}

	log.Info("taxonomy: удаление вкладки", zap.Int("id", id))
	if err := h.svc.DeleteTab(r.Context(), id); err != nil {
		log.Error("taxonomy: ошибка удаления вкладки", zap.Error(err), zap.Int("id", id))
		helpers.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Info("taxonomy: вкладка удалена", zap.Int("id", id))
	w.WriteHeader(http.StatusNoContent)
}

// CreateSection
// @Summary      Создать раздел во вкладке
// @Description  Доступно только администратору
// @Tags         taxonomy
// @Accept       json
// @Produce      json
// @Param        body  body  models.Section  true  "Данные раздела"
// @Success      201   {object} map[string]int
// @Failure      400   {object} map[string]string
// @Failure      500   {object} map[string]string
// @Router       /api/admin/sections [post]
func (h *TaxonomyHandler) CreateSection(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	var req models.Section
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("taxonomy: невалидный JSON при создании раздела", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "bad json")
		return
	}

	log.Info("taxonomy: создание раздела", zap.String("title", req.Title), zap.Int("tab_id", req.TabID))

	id, err := h.svc.CreateSection(r.Context(), &req)
	if err != nil {
		log.Error("taxonomy: ошибка создания раздела", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Info("taxonomy: раздел создан", zap.Int("id", id))
	helpers.JSON(w, http.StatusCreated, map[string]int{"id": id})
}

// UpdateSection
// @Summary      Обновить раздел
// @Description  Доступно только администратору
// @Tags         taxonomy
// @Accept       json
// @Produce      json
// @Param        id    path  int            true  "ID раздела"
// @Param        body  body  models.Section true  "Обновлённые данные"
// @Success      204   {string} string      "No Content"
// @Failure      400   {object} map[string]string
// @Failure      500   {object} map[string]string
// @Router       /api/admin/sections/{id} [patch]
func (h *TaxonomyHandler) UpdateSection(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		log.Warn("taxonomy: неверный id раздела при обновлении", zap.String("raw", idStr))
		helpers.Error(w, http.StatusBadRequest, "bad id")
		return
	}

	var req models.Section
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("taxonomy: невалидный JSON при обновлении раздела", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "bad json")
		return
	}
	req.ID = id

	log.Info("taxonomy: обновление раздела", zap.Int("id", id), zap.String("title", req.Title), zap.Int("tab_id", req.TabID))

	if err := h.svc.UpdateSection(r.Context(), &req); err != nil {
		log.Error("taxonomy: ошибка обновления раздела", zap.Error(err), zap.Int("id", id))
		helpers.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Info("taxonomy: раздел обновлён", zap.Int("id", id))
	w.WriteHeader(http.StatusNoContent)
}

// DeleteSection
// @Summary      Удалить раздел
// @Description  Доступно только администратору
// @Tags         taxonomy
// @Param        id  path  int  true  "ID раздела"
// @Success      204 {string} string "No Content"
// @Failure      500 {object} map[string]string
// @Router       /api/admin/sections/{id} [delete]
func (h *TaxonomyHandler) DeleteSection(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		log.Warn("taxonomy: неверный id раздела при удалении", zap.String("raw", idStr))
		helpers.Error(w, http.StatusBadRequest, "bad id")
		return
	}

	log.Info("taxonomy: удаление раздела", zap.Int("id", id))
	if err := h.svc.DeleteSection(r.Context(), id); err != nil {
		log.Error("taxonomy: ошибка удаления раздела", zap.Error(err), zap.Int("id", id))
		helpers.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Info("taxonomy: раздел удалён", zap.Int("id", id))
	w.WriteHeader(http.StatusNoContent)
}

// PublicTreeByTab
// @Summary      Получить дерево по конкретной вкладке
// @Description  {tab} может быть slug или числовой ID. Параметры ?id= и ?slug= также поддерживаются и необязательны.
// @Tags         taxonomy
// @Produce      json
// @Param        tab   path   string  true   "Slug или ID вкладки"
// @Param        id    query  int     false  "ID вкладки (необязателен)"
// @Param        slug  query  string  false  "Slug вкладки (необязателен)"
// @Success      200 {object} map[string][]models.TabTree
// @Failure      500 {object} map[string]string
// @Router       /api/taxonomy/tree/{tab} [get]
func (h *TaxonomyHandler) PublicTreeByTab(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	vars := mux.Vars(r)
	pathVal := vars["tab"]

	var (
		tabID   *int
		tabSlug *string
	)

	// path: попытка как id
	if id, err := strconv.Atoi(pathVal); err == nil {
		tabID = &id
	} else {
		s := pathVal
		tabSlug = &s
	}

	if qid := r.URL.Query().Get("id"); qid != "" {
		if v, err := strconv.Atoi(qid); err == nil {
			tabID = &v
		} else {
			log.Warn("taxonomy: неверный query id", zap.String("raw", qid))
		}
	}
	if qs := r.URL.Query().Get("slug"); qs != "" {
		tabSlug = &qs
	}

	log.Info("taxonomy: запрос дерева по вкладке",
		zap.Any("tab_id", tabID),
		zap.Any("tab_slug", tabSlug),
		zap.String("path_val", pathVal),
	)

	items, err := h.svc.PublicTreeFiltered(r.Context(), tabID, tabSlug)
	if err != nil {
		log.Error("taxonomy: ошибка получения фильтрованного дерева", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Info("taxonomy: дерево по вкладке получено", zap.Int("tabs_count", len(items)))
	helpers.JSON(w, http.StatusOK, map[string]any{"data": items})
}
