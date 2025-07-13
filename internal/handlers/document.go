package handlers

import (
	"edutalks/internal/logger"
	"edutalks/internal/middleware"
	"edutalks/internal/models"
	"edutalks/internal/services"
	"encoding/json"
	"fmt"
	"io"
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
// @Tags admin
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Файл документа"
// @Param description formData string false "Описание файла"
// @Param is_public formData bool false "Доступен по подписке?"
// @Success 200 {string} string "Файл загружен"
// @Failure 400 {string} string "Ошибка загрузки"
// @Router /api/admin/files/upload [post]
func (h *DocumentHandler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	logger.Log.Info("Запрос на загрузку документа")
	err := r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		logger.Log.Warn("Ошибка разбора формы при загрузке документа", zap.Error(err))
		http.Error(w, "Ошибка разбора формы", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		logger.Log.Warn("Файл не найден при загрузке", zap.Error(err))
		http.Error(w, "Файл не найден", http.StatusBadRequest)
		return
	}
	defer file.Close()

	description := r.FormValue("description")
	isPublic := strings.ToLower(r.FormValue("is_public")) == "true"

	userID := r.Context().Value(middleware.ContextUserID).(int)
	uploadDir := "uploaded"
	os.MkdirAll(uploadDir, os.ModePerm)

	filename := fmt.Sprintf("%d_%s", time.Now().Unix(), handler.Filename)
	fullPath := filepath.Join(uploadDir, filename)

	dst, err := os.Create(fullPath)
	if err != nil {
		logger.Log.Error("Ошибка при сохранении файла", zap.Error(err))
		http.Error(w, "Ошибка при сохранении файла", http.StatusInternalServerError)
		return
	}
	defer dst.Close()
	io.Copy(dst, file)

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
		http.Error(w, "Ошибка при сохранении документа", http.StatusInternalServerError)
		return
	}

	logger.Log.Info("Документ успешно загружен", zap.String("filename", handler.Filename), zap.Int("user_id", userID))
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Файл загружен"))
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
		http.Error(w, "Ошибка при получении документов", http.StatusInternalServerError)
		return
	}
	logger.Log.Info("Документы получены", zap.Int("count", len(docs)))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(docs)
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
		http.Error(w, "Пользователь не найден", http.StatusUnauthorized)
		return
	}

	idStr := mux.Vars(r)["id"]
	id, _ := strconv.Atoi(idStr)

	doc, err := h.service.GetDocumentByID(r.Context(), id)
	if err != nil {
		logger.Log.Warn("Документ не найден", zap.Int("doc_id", id))
		http.Error(w, "Документ не найден", http.StatusNotFound)
		return
	}

	if !user.HasSubscription {
		logger.Log.Warn("Попытка доступа к файлу без подписки", zap.Int("user_id", userID), zap.Int("doc_id", id))
		http.Error(w, "Нет доступа — купите подписку", http.StatusForbidden)
		return
	}

	if !doc.IsPublic {
		logger.Log.Warn("Попытка доступа к закрытому документу", zap.Int("user_id", userID), zap.Int("doc_id", id))
		http.Error(w, "Этот документ закрыт", http.StatusForbidden)
		return
	}

	fileBytes, err := os.ReadFile(doc.Filepath)
	if err != nil {
		logger.Log.Error("Файл не найден на диске", zap.String("filepath", doc.Filepath), zap.Error(err))
		http.Error(w, "Файл не найден", http.StatusInternalServerError)
		return
	}

	logger.Log.Info("Документ успешно скачан", zap.String("filename", doc.Filename), zap.Int("user_id", userID))
	w.Header().Set("Content-Disposition", "attachment; filename="+doc.Filename)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(fileBytes)
}

// DeleteDocument godoc
// @Summary Удаление документа (только для админа)
// @Tags admin
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
		http.Error(w, "Документ не найден", http.StatusNotFound)
		return
	}

	err = h.service.Delete(r.Context(), id)
	if err != nil {
		logger.Log.Error("Ошибка при удалении документа из базы", zap.Error(err), zap.Int("doc_id", id))
		http.Error(w, "Ошибка при удалении", http.StatusInternalServerError)
		return
	}

	if err := os.Remove(doc.Filepath); err != nil && !os.IsNotExist(err) {
		logger.Log.Error("Ошибка при удалении файла с диска", zap.String("filepath", doc.Filepath), zap.Error(err))
		http.Error(w, "Файл не удалось удалить", http.StatusInternalServerError)
		return
	}

	logger.Log.Info("Документ успешно удалён", zap.Int("doc_id", id))
	w.Write([]byte("Документ удалён"))
}
