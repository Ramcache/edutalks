package handlers

import (
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
	"time"

	"github.com/gorilla/mux"
)

type DocumentHandler struct {
	service *services.DocumentService
}

func NewDocumentHandler(s *services.DocumentService) *DocumentHandler {
	return &DocumentHandler{service: s}
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
	userID := r.Context().Value(middleware.ContextUserID).(int)

	// файл
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Не удалось прочитать файл", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// описание
	description := r.FormValue("description")
	isPublic := r.FormValue("is_public") == "true"

	// сохраняем файл на диск
	dstPath := filepath.Join("uploaded", fmt.Sprintf("%d_%s", time.Now().Unix(), header.Filename))
	dstFile, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "Не удалось сохранить файл", http.StatusInternalServerError)
		return
	}
	defer dstFile.Close()
	io.Copy(dstFile, file)

	// в БД
	doc := &models.Document{
		UserID:      userID,
		Filename:    header.Filename,
		Filepath:    dstPath,
		Description: description,
		IsPublic:    isPublic,
	}
	if err := h.service.Upload(r.Context(), doc); err != nil {
		http.Error(w, "Ошибка записи в БД", http.StatusInternalServerError)
		return
	}

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
	docs, err := h.service.GetPublicDocuments(r.Context())
	if err != nil {
		http.Error(w, "Ошибка при получении документов", http.StatusInternalServerError)
		return
	}
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
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	doc, err := h.service.GetDocumentByID(r.Context(), id)
	if err != nil || !doc.IsPublic {
		http.Error(w, "Документ не найден или недоступен", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, doc.Filepath)
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

	// Загружаем документ для проверки и удаления файла с диска
	doc, err := h.service.GetDocumentByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Документ не найден", http.StatusNotFound)
		return
	}

	// Удаляем из базы
	err = h.service.Delete(r.Context(), id)
	if err != nil {
		http.Error(w, "Ошибка при удалении", http.StatusInternalServerError)
		return
	}

	// Удаляем файл с диска
	if err := os.Remove(doc.Filepath); err != nil && !os.IsNotExist(err) {
		http.Error(w, "Файл не удалось удалить", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Документ удалён"))
}
