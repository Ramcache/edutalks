package handlers

import (
	"edutalks/internal/logger"
	"edutalks/internal/middleware"
	"edutalks/internal/models"
	"edutalks/internal/services"
	helpers "edutalks/internal/utils/helpers"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type DocumentHandler struct {
	service     *services.DocumentService
	userService *services.AuthService
}

func NewDocumentHandler(docService *services.DocumentService, userService *services.AuthService) *DocumentHandler {
	return &DocumentHandler{
		service:     docService,
		userService: userService,
	}
}

// UploadDocument
// @Summary      Загрузить документ
// @Description  Админ может загрузить документ и привязать его к разделу
// @Tags         documents
// @Accept       multipart/form-data
// @Produce      json
// @Param        file        formData  file    true   "Файл"
// @Param        description formData  string  false  "Описание"
// @Param        is_public   formData  bool    true   "Публичный документ?"
// @Param        category    formData  string  false  "Категория"
// @Param        section_id  formData  int     false  "ID раздела"
// @Success      201 {object} map[string]int
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /api/admin/files/upload [post]
func (h *DocumentHandler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	logger.Log.Info("Запрос на загрузку документа")
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		logger.Log.Warn("Ошибка разбора формы при загрузке документа", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "Ошибка разбора формы")
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		logger.Log.Warn("Файл не найден при загрузке", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "Файл не найден")
		return
	}
	defer file.Close()

	description := r.FormValue("description")
	isPublic := strings.ToLower(r.FormValue("is_public")) == "true"
	category := r.FormValue("category")

	var sectionIDPtr *int
	if s := r.FormValue("section_id"); s != "" {
		if sid, err := strconv.Atoi(s); err == nil {
			sectionIDPtr = &sid
		}
	}

	userID := r.Context().Value(middleware.ContextUserID).(int)

	uploadDir := "uploaded"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		logger.Log.Error("Ошибка создания директории загрузки", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка сохранения файла")
		return
	}

	filename := fmt.Sprintf("%d_%s", time.Now().Unix(), handler.Filename)
	fullPath := filepath.Join(uploadDir, filename)

	dst, err := os.Create(fullPath)
	if err != nil {
		logger.Log.Error("Ошибка при сохранении файла", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка при сохранении файла")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		logger.Log.Error("Ошибка записи файла", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка при сохранении файла")
		return
	}

	doc := &models.Document{
		UserID:      userID,
		Filename:    handler.Filename,
		Filepath:    fullPath,
		Description: description,
		IsPublic:    isPublic,
		Category:    category,
		SectionID:   sectionIDPtr,
		UploadedAt:  time.Now(),
	}

	logger.Log.Info("Сохраняем информацию о документе", zap.String("filename", handler.Filename), zap.Int("user_id", userID))
	id, err := h.service.Upload(r.Context(), doc)
	if err != nil {
		logger.Log.Error("Ошибка при сохранении документа в базе", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка при сохранении документа")
		return
	}

	helpers.JSON(w, http.StatusCreated, map[string]any{
		"id": id,
		"item": map[string]any{
			"id":          id,
			"filename":    doc.Filename,
			"description": doc.Description,
			"category":    doc.Category,
			"section_id":  doc.SectionID,
			"is_public":   doc.IsPublic,
			"uploaded_at": doc.UploadedAt,
		},
	})
}

// ListPublicDocuments
// @Summary      Получить список публичных документов
// @Description  Поддерживает фильтры: section_id и category
// @Tags         documents
// @Produce      json
// @Param        page        query  int     false  "Номер страницы (по умолчанию 1)"
// @Param        page_size   query  int     false  "Размер страницы (по умолчанию 10)"
// @Param        section_id  query  int     false  "ID раздела"
// @Param        category    query  string  false  "Категория документа"
// @Success      200 {object} map[string]interface{} "items, page, page_size, total"
// @Failure      500 {object} map[string]string
// @Router       /api/files [get]
func (h *DocumentHandler) ListPublicDocuments(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	category := r.URL.Query().Get("category")
	var sectionIDPtr *int
	if s := r.URL.Query().Get("section_id"); s != "" {
		if sid, err := strconv.Atoi(s); err == nil {
			sectionIDPtr = &sid
		}
	}

	var (
		docs  []*models.Document
		total int
		err   error
	)

	if sectionIDPtr != nil || category != "" {
		docs, total, err = h.service.GetPublicDocumentsByFilterPaginated(r.Context(), pageSize, offset, sectionIDPtr, category)
	} else {
		docs, total, err = h.service.GetPublicDocumentsPaginated(r.Context(), pageSize, offset, "")
	}
	if err != nil {
		logger.Log.Error("Ошибка при получении документов", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка при получении документов")
		return
	}

	helpers.JSON(w, http.StatusOK, map[string]any{
		"items":      docs,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"category":   category,
		"section_id": sectionIDPtr,
	})
}

// DownloadDocument godoc
// @Summary Скачать документ по ID
// @Tags files
// @Security ApiKeyAuth
// @Produce application/octet-stream
// @Param id path int true "ID документа"
// @Success 200 {file} file
// @Failure 403 {string} string "Нет доступа"
// @Failure 404 {string} string "Документ не найден"
// @Router /api/files/{id} [get]
func (h *DocumentHandler) DownloadDocument(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.ContextUserID).(int)
	logger.Log.Info("Запрос на скачивание документа", zap.Int("user_id", userID))

	user, err := h.userService.GetUserByID(r.Context(), userID)
	if err != nil {
		logger.Log.Warn("Пользователь не найден при скачивании документа", zap.Int("user_id", userID))
		helpers.Error(w, http.StatusUnauthorized, "Пользователь не найден")
		return
	}

	idStr := mux.Vars(r)["id"]
	id, _ := strconv.Atoi(idStr)

	doc, err := h.service.GetDocumentByID(r.Context(), id)
	if err != nil {
		logger.Log.Warn("Документ не найден", zap.Int("doc_id", id))
		helpers.Error(w, http.StatusNotFound, "Документ не найден")
		return
	}

	// --- доступ как у тебя сейчас ---
	now := time.Now().UTC()
	if !(user.HasSubscription && user.SubscriptionExpiresAt != nil && user.SubscriptionExpiresAt.After(now)) {
		logger.Log.Warn("Подписка неактивна", zap.Int("user_id", userID), zap.Timep("expires_at", user.SubscriptionExpiresAt))
		helpers.Error(w, http.StatusForbidden, "Нет доступа — купите подписку")
		return
	}
	if !doc.IsPublic {
		logger.Log.Warn("Попытка доступа к закрытому документу", zap.Int("user_id", userID), zap.Int("doc_id", id))
		helpers.Error(w, http.StatusForbidden, "Этот документ закрыт")
		return
	}
	// ---------------------------------

	// Открываем файл и определяем корректный Content-Type
	f, err := os.Open(doc.Filepath)
	if err != nil {
		logger.Log.Error("Файл не найден на диске", zap.String("filepath", doc.Filepath), zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Файл не найден")
		return
	}
	defer f.Close()

	// 1) по расширению
	ctype := mime.TypeByExtension(strings.ToLower(filepath.Ext(doc.Filename)))
	if ctype == "" {
		// 2) по содержимому (первые 512 байт)
		buf := make([]byte, 512)
		n, _ := f.Read(buf)
		ctype = http.DetectContentType(buf[:n])
		_, _ = f.Seek(0, io.SeekStart)
	}
	if ctype == "" {
		ctype = "application/octet-stream"
	}

	// Безопасное имя файла (UTF-8, пробелы/кириллица ок)
	encoded := url.PathEscape(doc.Filename)
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", encoded))

	// (необязательно) длина файла
	if fi, err := f.Stat(); err == nil {
		w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	}

	// Эффективная отдача с поддержкой Range/кэша
	http.ServeContent(w, r, doc.Filename, doc.UploadedAt, f)

	logger.Log.Info("Документ успешно скачан", zap.String("filename", doc.Filename), zap.Int("user_id", userID))
}

// DeleteDocument godoc
// @Summary Удаление документа (только для админа)
// @Tags admin-files
// @Security ApiKeyAuth
// @Param id path int true "ID документа"
// @Success 200 {string} string "Документ удалён"
// @Failure 404 {string} string "Документ не найден"
// @Router /api/admin/files/{id} [delete]
func (h *DocumentHandler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])
	logger.Log.Info("Запрос на удаление документа", zap.Int("doc_id", id))

	doc, err := h.service.GetDocumentByID(r.Context(), id)
	if err != nil {
		logger.Log.Warn("Документ не найден для удаления", zap.Int("doc_id", id))
		helpers.Error(w, http.StatusNotFound, "Документ не найден")
		return
	}

	err = h.service.Delete(r.Context(), id)
	if err != nil {
		logger.Log.Error("Ошибка при удалении документа из базы", zap.Error(err), zap.Int("doc_id", id))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка при удалении")
		return
	}

	if err := os.Remove(doc.Filepath); err != nil && !os.IsNotExist(err) {
		logger.Log.Error("Ошибка при удалении файла с диска", zap.String("filepath", doc.Filepath), zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Файл не удалось удалить")
		return
	}

	logger.Log.Info("Документ успешно удалён", zap.Int("doc_id", id))
	helpers.JSON(w, http.StatusOK, "Документ удалён")
}

// GetAllDocuments godoc
// @Summary Получить все документы (только для админа)
// @Tags admin-files
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {array} models.Document
// @Failure 500 {string} string "Ошибка сервера"
// @Router /api/admin/files [get]
func (h *DocumentHandler) GetAllDocuments(w http.ResponseWriter, r *http.Request) {
	docs, err := h.service.GetAllDocuments(r.Context())
	if err != nil {
		helpers.Error(w, http.StatusInternalServerError, "Ошибка получения документов")
		return
	}
	helpers.JSON(w, http.StatusOK, map[string]any{"items": docs})
}

// PreviewDocument godoc
// @Summary Превью публичного документа (только метаданные)
// @Description Показывает название, описание и категорию документа. Файл не отдаётся.
// @Tags public-documents
// @Param id path int true "ID документа"
// @Produce json
// @Success 200 {object} models.DocumentPreviewResponse
// @Failure 404 {object} string "Документ не найден"
// @Failure 403 {object} string "Документ не публичный"
// @Router /documents/{id}/preview [get]
func (h *DocumentHandler) PreviewDocument(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, _ := strconv.Atoi(idStr)

	doc, err := h.service.GetDocumentByID(r.Context(), id)
	if err != nil {
		helpers.Error(w, http.StatusNotFound, "Документ не найден")
		return
	}

	if !doc.IsPublic {
		helpers.Error(w, http.StatusForbidden, "Документ недоступен для просмотра")
		return
	}

	resp := models.DocumentPreviewResponse{
		ID:          doc.ID,
		Title:       doc.Filename,
		Description: doc.Description,
		Category:    doc.Category,
		UploadedAt:  doc.UploadedAt.Format("2006-01-02"),
		Message:     "Документ доступен только по подписке",
	}
	helpers.JSON(w, http.StatusOK, map[string]any{"item": resp})

}

// PreviewDocuments godoc
// @Summary Превью публичных документов (список, метаданные)
// @Description Возвращает список превью документов c пагинацией и фильтром категории.
// @Tags public-documents
// @Produce json
// @Param page query int false "Номер страницы (>=1)"
// @Param page_size query int false "Размер страницы (1..100)"
// @Param category query string false "Категория (например, 'приказ', 'шаблон')"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {string} string "Ошибка сервера"
// @Router /documents/preview [get]
func (h *DocumentHandler) PreviewDocuments(w http.ResponseWriter, r *http.Request) {
	// пагинация
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	// фильтр по категории
	category := r.URL.Query().Get("category")

	// берём из сервиса только публичные документы
	docs, total, err := h.service.GetPublicDocumentsPaginated(r.Context(), pageSize, offset, category)
	if err != nil {
		helpers.Error(w, http.StatusInternalServerError, "Ошибка получения документов")
		return
	}

	// собираем превью (метаданные)
	previews := make([]models.DocumentPreviewResponse, 0, len(docs))
	for _, d := range docs {
		if !d.IsPublic {
			continue
		}
		previews = append(previews, models.DocumentPreviewResponse{
			ID:          d.ID,
			Title:       d.Filename,
			Description: d.Description,
			Category:    d.Category,
			UploadedAt:  d.UploadedAt.Format("2006-01-02"),
			Message:     "Документ доступен только по подписке",
		})
	}

	helpers.JSON(w, http.StatusOK, map[string]any{
		"items":     previews,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"category":  category,
	})
}
