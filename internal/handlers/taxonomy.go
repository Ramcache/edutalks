package handlers

import (
	"edutalks/internal/models"
	"edutalks/internal/services"
	helpers "edutalks/internal/utils/helpers"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
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
	tree, err := h.svc.PublicTree(r.Context())
	if err != nil {
		helpers.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
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
	var req models.Tab
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.Error(w, 400, "bad json")
		return
	}
	id, err := h.svc.CreateTab(r.Context(), &req)
	if err != nil {
		helpers.Error(w, 500, err.Error())
		return
	}
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
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	var req models.Tab
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.Error(w, 400, "bad json")
		return
	}
	req.ID = id
	if err := h.svc.UpdateTab(r.Context(), &req); err != nil {
		helpers.Error(w, 500, err.Error())
		return
	}
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
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	if err := h.svc.DeleteTab(r.Context(), id); err != nil {
		helpers.Error(w, 500, err.Error())
		return
	}
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
	var req models.Section
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.Error(w, 400, "bad json")
		return
	}
	id, err := h.svc.CreateSection(r.Context(), &req)
	if err != nil {
		helpers.Error(w, 500, err.Error())
		return
	}
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
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	var req models.Section
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		helpers.Error(w, 400, "bad json")
		return
	}
	req.ID = id
	if err := h.svc.UpdateSection(r.Context(), &req); err != nil {
		helpers.Error(w, 500, err.Error())
		return
	}
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
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	if err := h.svc.DeleteSection(r.Context(), id); err != nil {
		helpers.Error(w, 500, err.Error())
		return
	}
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
		}
	}
	if qs := r.URL.Query().Get("slug"); qs != "" {
		tabSlug = &qs
	}

	items, err := h.svc.PublicTreeFiltered(r.Context(), tabID, tabSlug)
	if err != nil {
		helpers.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	helpers.JSON(w, http.StatusOK, map[string]any{"data": items})
}
