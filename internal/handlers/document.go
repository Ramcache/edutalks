package handlers

import (
	"edutalks/internal/logger"
	"edutalks/internal/middleware"
	"edutalks/internal/models"
	"edutalks/internal/services"
	helpers "edutalks/internal/utils/helpres"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
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

// UploadDocument godoc
// @Summary Загрузка документа (только для админа)
// @Tags admin-files
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Файл документа"
// @Param description formData string false "Описание файла"
// @Param is_public formData bool false "Доступен по подписке?"
// @Success 201 {string} string "Файл загружен"
// @Failure 400 {string} string "Ошибка загрузки"
// @Router /api/admin/files/upload [post]
func (h *DocumentHandler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	logger.Log.Info("Запрос на загрузку документа")
	err := r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
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
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {
			logger.Log.Error("ошибка при закрытии файла: %v", zap.Error(err))
		}
	}(file)

	description := r.FormValue("description")
	isPublic := strings.ToLower(r.FormValue("is_public")) == "true"

	userID := r.Context().Value(middleware.ContextUserID).(int)
	uploadDir := "uploaded"
	err = os.MkdirAll(uploadDir, os.ModePerm)
	if err != nil {
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
	defer func(dst *os.File) {
		err := dst.Close()
		if err != nil {
			logger.Log.Error("ошибка при закрытии файла: %v", zap.Error(err))
		}
	}(dst)
	_, err = io.Copy(dst, file)
	if err != nil {
		return
	}

	doc := &models.Document{
		UserID:      userID,
		Filename:    handler.Filename,
		Filepath:    fullPath,
		Description: description,
		IsPublic:    isPublic,
		UploadedAt:  time.Now(),
	}

	logger.Log.Info("Сохраняем информацию о документе", zap.String("filename", handler.Filename), zap.Int("user_id", userID))
	err = h.service.Upload(r.Context(), doc)
	if err != nil {
		logger.Log.Error("Ошибка при сохранении документа в базе", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка при сохранении документа")
		return
	}

	logger.Log.Info("Документ успешно загружен", zap.String("filename", handler.Filename), zap.Int("user_id", userID))
	helpers.JSON(w, http.StatusCreated, "Файл загружен")
}

// ListPublicDocuments godoc
// @Summary Список доступных документов (по подписке)
// @Tags files
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {array} models.Document
// @Failure 500 {string} string "Ошибка сервера"
// @Router /api/files [get]
func (h *DocumentHandler) ListPublicDocuments(w http.ResponseWriter, r *http.Request) {
	logger.Log.Info("Запрос на получение списка публичных документов")
	docs, err := h.service.GetPublicDocuments(r.Context())
	if err != nil {
		logger.Log.Error("Ошибка при получении документов", zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Ошибка при получении документов")
		return
	}
	logger.Log.Info("Документы получены", zap.Int("count", len(docs)))
	helpers.JSON(w, http.StatusOK, docs)
}

// DownloadDocument godoc
// @Summary Скачать документ по ID
// @Tags files
// @Security ApiKeyAuth
// @Produce octet-stream
// @Param id path int true "ID документа"
// @Success 200 {file} file
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

	if !user.HasSubscription {
		logger.Log.Warn("Попытка доступа к файлу без подписки", zap.Int("user_id", userID), zap.Int("doc_id", id))
		helpers.Error(w, http.StatusForbidden, "Нет доступа — купите подписку")
		return
	}

	if !doc.IsPublic {
		logger.Log.Warn("Попытка доступа к закрытому документу", zap.Int("user_id", userID), zap.Int("doc_id", id))
		helpers.Error(w, http.StatusForbidden, "Этот документ закрыт")
		return
	}

	fileBytes, err := os.ReadFile(doc.Filepath)
	if err != nil {
		logger.Log.Error("Файл не найден на диске", zap.String("filepath", doc.Filepath), zap.Error(err))
		helpers.Error(w, http.StatusInternalServerError, "Файл не найден")
		return
	}

	logger.Log.Info("Документ успешно скачан", zap.String("filename", doc.Filename), zap.Int("user_id", userID))
	w.Header().Set("Content-Disposition", "attachment; filename="+doc.Filename)
	w.Header().Set("Content-Type", "application/octet-stream")
	_, err = w.Write(fileBytes)
	if err != nil {
		return
	}
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
