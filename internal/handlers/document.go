package handlers

import (
	"context"
	"encoding/json"
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

	"edutalks/internal/logger"
	"edutalks/internal/middleware"
	"edutalks/internal/models"
	"edutalks/internal/repository"
	"edutalks/internal/services"
	helpers "edutalks/internal/utils/helpers"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type DocumentHandler struct {
	service      *services.DocumentService
	userService  *services.AuthService
	notifier     *services.Notifier
	taxonomyRepo *repository.TaxonomyRepo
}

func NewDocumentHandler(docService *services.DocumentService, userService *services.AuthService, notifier *services.Notifier, taxonomyRepo *repository.TaxonomyRepo) *DocumentHandler {
	return &DocumentHandler{
		service:      docService,
		userService:  userService,
		notifier:     notifier,
		taxonomyRepo: taxonomyRepo,
	}
}

// UploadDocument
// @Summary      Загрузить документ
// @Description  Админ может загрузить документ и привязать его к разделу
// @Tags         documents
// @Accept       multipart/form-data
// @Produce      json
// @Param        title       formData  string  false  "Название документа"
// @Param        file        formData  file    true   "Файл"
// @Param        description formData  string  false  "Описание"
// @Param        is_public   formData  bool    true   "Публичный документ?"
// @Param        category    formData  string  false  "Категория"
// @Param        section_id  formData  int     false  "ID раздела"
// @Param        allow_free_download formData bool false "Можно скачивать без подписки?"
// @Success      201 {object} map[string]int
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /api/admin/files/upload [post]
func (h *DocumentHandler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())
	log.Info("Запрос на загрузку документа")

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		log.Warn("Ошибка разбора формы при загрузке документа", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "Ошибка разбора формы")
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		log.Warn("Файл не найден в форме", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "Файл не найден")
		return
	}
	defer file.Close()

	description := r.FormValue("description")
	isPublic := strings.ToLower(r.FormValue("is_public")) == "true"
	category := r.FormValue("category")
	title := r.FormValue("title")
	allowFreeDownload := strings.ToLower(r.FormValue("allow_free_download")) == "true"

	var sectionIDPtr *int
	if s := r.FormValue("section_id"); s != "" {
		if sid, convErr := strconv.Atoi(s); convErr == nil {
			sectionIDPtr = &sid
		} else {
			log.Warn("Невалидный section_id", zap.String("raw", s))
		}
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok || userID == 0 {
		log.Warn("Нет user_id в контексте при загрузке документа")
		helpers.Error(w, http.StatusUnauthorized, "Нет доступа")
		return
	}

	log.Info("Параметры загрузки документа",
		zap.String("original_filename", handler.Filename),
		zap.Int64("upload_size_hint", handler.Size),
		zap.String("title", title),
		zap.String("category", category),
		zap.Bool("is_public", isPublic),
		zap.Bool("allow_free_download", allowFreeDownload),
		zap.Any("section_id", sectionIDPtr),
		zap.Int("user_id", userID),
	)

	uploadDir := "uploaded"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		log.Error("Не удалось создать директорию загрузки", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка сохранения файла")
		return
	}

	filename := fmt.Sprintf("%d_%s", time.Now().Unix(), handler.Filename)
	fullPath := filepath.Join(uploadDir, filename)

	dst, err := os.Create(fullPath)
	if err != nil {
		log.Error("Не удалось создать файл на диске", zap.String("path", fullPath), zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка при сохранении файла")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		log.Error("Ошибка записи файла на диск", zap.String("path", fullPath), zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка при сохранении файла")
		return
	}

	doc := &models.Document{
		UserID:            userID,
		Title:             title,
		Filename:          handler.Filename,
		Filepath:          fullPath,
		Description:       description,
		IsPublic:          isPublic,
		Category:          category,
		SectionID:         sectionIDPtr,
		UploadedAt:        time.Now(),
		AllowFreeDownload: allowFreeDownload,
	}

	log.Info("Сохраняем метаданные документа в БД",
		zap.String("stored_filename", filename),
		zap.String("original_filename", handler.Filename),
		zap.Int("user_id", userID),
	)

	id, err := h.service.Upload(r.Context(), doc)
	if err != nil {
		log.Error("Ошибка сохранения документа в БД", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка при сохранении документа")
		return
	}

	ctx := context.WithoutCancel(r.Context())
	var tabsID *int
	if doc.SectionID != nil {
		if tid, e := h.taxonomyRepo.GetTabIDBySectionID(ctx, *doc.SectionID); e == nil {
			tabsID = &tid
		} else {
			log.Warn("Не удалось получить tab_id по section_id", zap.Any("section_id", *doc.SectionID), zap.Error(e))
		}
	}
	h.notifier.AddDocumentForBatch(ctx, doc.Title, tabsID)
	log.Info("Документ добавлен в batched-уведомления", zap.Int("doc_id", id), zap.Any("tab_id", tabsID))

	helpers.JSON(w, http.StatusCreated, map[string]any{
		"id": id,
		"data": map[string]any{
			"id":                  id,
			"title":               doc.Title,
			"filename":            doc.Filename,
			"description":         doc.Description,
			"category":            doc.Category,
			"section_id":          doc.SectionID,
			"is_public":           doc.IsPublic,
			"uploaded_at":         doc.UploadedAt,
			"allow_free_download": doc.AllowFreeDownload,
		},
	})
}

// ListPublicDocuments
// @Summary      Получить список публичных документов (без пагинации)
// @Description  Поддерживает фильтры: section_id и category. Возвращает все подходящие документы.
// @Tags         documents
// @Produce      json
// @Param        section_id  query  int     false  "ID раздела"
// @Param        category    query  string  false  "Категория документа"
// @Success      200 {object} map[string]interface{} "data, total, category, section_id"
// @Failure      500 {object} map[string]string
// @Router       /api/files [get]
func (h *DocumentHandler) ListPublicDocuments(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	category := r.URL.Query().Get("category")

	var sectionIDPtr *int
	if s := r.URL.Query().Get("section_id"); s != "" {
		if sid, err := strconv.Atoi(s); err == nil {
			sectionIDPtr = &sid
		} else {
			log.Warn("Невалидный section_id", zap.String("raw", s))
		}
	}

	log.Info("Запрос публичных документов", zap.Any("section_id", sectionIDPtr), zap.String("category", category))

	docs, err := h.service.GetPublicDocuments(r.Context(), sectionIDPtr, category)
	if err != nil {
		log.Error("Ошибка получения публичных документов", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка при получении документов")
		return
	}

	log.Info("Публичные документы получены", zap.Int("count", len(docs)))
	helpers.JSON(w, http.StatusOK, map[string]any{
		"data":       docs,
		"total":      len(docs),
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
	log := logger.WithCtx(r.Context())

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok || userID == 0 {
		log.Warn("Нет доступа при скачивании документа: отсутствует user_id")
		helpers.Error(w, http.StatusUnauthorized, "Пользователь не найден")
		return
	}

	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		log.Warn("Невалидный идентификатор документа", zap.String("raw", idStr))
		helpers.Error(w, http.StatusBadRequest, "Некорректный идентификатор документа")
		return
	}

	log.Info("Запрос на скачивание документа", zap.Int("user_id", userID), zap.Int("doc_id", id))

	user, err := h.userService.GetUserByID(r.Context(), userID)
	if err != nil {
		log.Warn("Пользователь не найден при скачивании документа", zap.Int("user_id", userID))
		helpers.Error(w, http.StatusUnauthorized, "Пользователь не найден")
		return
	}

	doc, err := h.service.GetDocumentByID(r.Context(), id)
	if err != nil {
		log.Warn("Документ не найден", zap.Int("doc_id", id))
		helpers.Error(w, http.StatusNotFound, "Документ не найден")
		return
	}

	if user.Role != "admin" {
		if !doc.IsPublic {
			log.Warn("Попытка доступа к закрытому документу", zap.Int("user_id", userID), zap.Int("doc_id", id))
			helpers.Error(w, http.StatusForbidden, "Этот документ закрыт")
			return
		}
		if !isActiveSub(user) && !doc.AllowFreeDownload {
			log.Warn("Нет подписки и документ не free", zap.Int("user_id", userID), zap.Int("doc_id", id))
			helpers.Error(w, http.StatusForbidden, "Нет доступа — купите подписку")
			return
		}
	}

	f, err := os.Open(doc.Filepath)
	if err != nil {
		log.Error("Файл не найден на диске", zap.String("filepath", doc.Filepath), zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Файл не найден")
		return
	}
	defer f.Close()

	ctype := mime.TypeByExtension(strings.ToLower(filepath.Ext(doc.Filename)))
	if ctype == "" {
		buf := make([]byte, 512)
		n, _ := f.Read(buf)
		ctype = http.DetectContentType(buf[:n])
		_, _ = f.Seek(0, io.SeekStart)
	}
	if ctype == "" {
		ctype = "application/octet-stream"
	}

	encoded := url.PathEscape(doc.Filename)
	w.Header().Set("Content-Type", ctype)
	// Добавляем и filename и filename*, чтобы охватить больше клиентов
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", doc.Filename, encoded))

	if fi, err := f.Stat(); err == nil {
		w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
		if doc.IsPublic && (doc.AllowFreeDownload || user.Role == "admin") {
			w.Header().Set("Cache-Control", "private, max-age=3600")
		}
	}

	http.ServeContent(w, r, doc.Filename, doc.UploadedAt, f)

	log.Info("Документ успешно скачан",
		zap.Int("user_id", userID),
		zap.Int("doc_id", id),
		zap.String("role", user.Role),
		zap.Bool("active_sub", isActiveSub(user)),
		zap.Bool("is_public", doc.IsPublic),
		zap.Bool("free", doc.AllowFreeDownload),
	)
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
	log := logger.WithCtx(r.Context())

	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		log.Warn("Невалидный doc_id в DeleteDocument", zap.String("raw", idStr))
		helpers.Error(w, http.StatusBadRequest, "Некорректный id документа")
		return
	}

	log.Info("Запрос на удаление документа", zap.Int("doc_id", id))

	doc, err := h.service.GetDocumentByID(r.Context(), id)
	if err != nil {
		log.Warn("Документ не найден для удаления", zap.Int("doc_id", id))
		helpers.Error(w, http.StatusNotFound, "Документ не найден")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		log.Error("Ошибка при удалении документа из базы", zap.Error(err), zap.Int("doc_id", id))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка при удалении")
		return
	}

	if err := os.Remove(doc.Filepath); err != nil && !os.IsNotExist(err) {
		log.Error("Ошибка при удалении файла с диска", zap.String("filepath", doc.Filepath), zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Файл не удалось удалить")
		return
	}

	log.Info("Документ успешно удалён", zap.Int("doc_id", id))
	helpers.JSON(w, http.StatusOK, "Документ удалён")
}

// GetAllDocuments godoc
// @Summary Получить все документы (только для админа)
// @Tags admin-files
// @Security ApiKeyAuth
// @Produce json
// @Param limit query int false "Максимальное количество документов (по умолчанию 10, 0 = все)"
// @Success 200 {array} models.Document
// @Failure 500 {string} string "Ошибка сервера"
// @Router /api/admin/files [get]
func (h *DocumentHandler) GetAllDocuments(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	limit := 10
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			if v >= 0 {
				limit = v // 0 = все
			}
		} else {
			log.Warn("Невалидный limit, используется значение по умолчанию", zap.String("raw", raw))
		}
	}

	log.Info("Запрос списка всех документов (admin)", zap.Int("limit", limit))

	docs, err := h.service.GetAllDocuments(r.Context(), limit)
	if err != nil {
		log.Error("Ошибка получения всех документов", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка получения документов")
		return
	}

	log.Info("Список документов получен", zap.Int("count", len(docs)))
	helpers.JSON(w, http.StatusOK, map[string]any{"data": docs})
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
// @Router /api/documents/{id}/preview [get]
func (h *DocumentHandler) PreviewDocument(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		log.Warn("Невалидный id в PreviewDocument", zap.String("raw", idStr))
		helpers.Error(w, http.StatusBadRequest, "Невалидный id")
		return
	}

	doc, err := h.service.GetDocumentByID(r.Context(), id)
	if err != nil {
		log.Warn("Документ не найден (preview)", zap.Int("doc_id", id))
		helpers.Error(w, http.StatusNotFound, "Документ не найден")
		return
	}

	if !doc.IsPublic {
		log.Warn("Документ не публичный (preview запрещён)", zap.Int("doc_id", id))
		helpers.Error(w, http.StatusForbidden, "Документ недоступен для просмотра")
		return
	}

	resp := models.DocumentPreviewResponse{
		ID:          doc.ID,
		Title:       doc.Title,
		Description: doc.Description,
		Category:    doc.Category,
		SectionID:   doc.SectionID,
		UploadedAt:  doc.UploadedAt.Format("2006-01-02"),
		Message:     "Документ доступен только по подписке",
	}

	log.Info("Превью документа сформировано", zap.Int("doc_id", id))
	helpers.JSON(w, http.StatusOK, map[string]any{"item": resp})
}

// PreviewDocuments godoc
// @Summary Превью публичных документов (список, метаданные)
// @Tags public-documents
// @Produce json
// @Param page query int false "Номер страницы (по умолчанию 1)"
// @Param page_size query int false "Размер страницы (по умолчанию 10)"
// @Param category query string false "Категория"
// @Success 200 {object} map[string]interface{} "data, page, page_size, total, category"
// @Failure 500 {object} map[string]string
// @Router /api/documents/preview [get]
func (h *DocumentHandler) PreviewDocuments(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

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

	log.Info("Запрос превью документов",
		zap.Int("page", page), zap.Int("page_size", pageSize),
		zap.Int("offset", offset), zap.String("category", category),
	)

	docs, total, err := h.service.GetPublicDocumentsPaginated(r.Context(), pageSize, offset, category)
	if err != nil {
		log.Error("Ошибка получения превью документов", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка получения документов")
		return
	}

	previews := make([]models.DocumentPreviewResponse, 0, len(docs))
	for _, d := range docs {
		if !d.IsPublic {
			continue
		}
		previews = append(previews, models.DocumentPreviewResponse{
			ID:          d.ID,
			Title:       d.Title,
			Description: d.Description,
			Category:    d.Category,
			SectionID:   d.SectionID,
			UploadedAt:  d.UploadedAt.Format("2006-01-02"),
			Message:     "Документ доступен только по подписке",
		})
	}

	log.Info("Превью документов сформировано", zap.Int("count", len(previews)), zap.Int("total", total))
	helpers.JSON(w, http.StatusOK, map[string]any{
		"data":      previews,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"category":  category,
	})
}

// UpdateMyProfile godoc
// @Summary Обновить свои данные
// @Tags profile
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param input body models.UpdateUserRequest true "Данные для обновления"
// @Success 200 {string} string "Профиль обновлён"
// @Failure 400 {string} string "Ошибка запроса"
// @Failure 401 {string} string "Нет доступа"
// @Router /api/profile [patch]
func (h *AuthHandler) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok || userID == 0 {
		log.Warn("Нет доступа при обновлении профиля: отсутствует user_id")
		helpers.Error(w, http.StatusUnauthorized, "Нет доступа")
		return
	}

	var input models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Warn("Невалидный JSON при обновлении профиля", zap.Error(err))
		helpers.Error(w, http.StatusBadRequest, "Невалидный JSON")
		return
	}

	input.Role = nil // обычный пользователь не меняет роль

	if err := h.authService.UpdateUser(r.Context(), userID, &input); err != nil {
		log.Error("Ошибка обновления профиля", zap.Error(err), zap.Int("user_id", userID))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка обновления профиля")
		return
	}

	log.Info("Профиль обновлён", zap.Int("user_id", userID))
	helpers.JSON(w, http.StatusOK, map[string]string{"message": "Профиль обновлён"})
}

func isActiveSub(u *models.User) bool {
	if u == nil || !u.HasSubscription || u.SubscriptionExpiresAt == nil {
		return false
	}
	return u.SubscriptionExpiresAt.After(time.Now().UTC())
}
