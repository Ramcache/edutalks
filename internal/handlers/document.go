package handlers

import (
	"edutalks/internal/middleware"
	"edutalks/internal/models"
	"edutalks/internal/services"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type DocumentHandler struct {
	service *services.DocumentService
}

func NewDocumentHandler(s *services.DocumentService) *DocumentHandler {
	return &DocumentHandler{service: s}
}

// UploadDocument godoc
// @Summary Загрузка документа
// @Tags files
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Документ"
// @Success 200 {string} string "Файл загружен"
// @Failure 400 {string} string "Ошибка загрузки"
// @Router /api/files/upload [post]
func (h *DocumentHandler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.ContextUserID).(int)

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Не удалось прочитать файл", http.StatusBadRequest)
		return
	}
	defer file.Close()

	dstPath := filepath.Join("uploaded", fmt.Sprintf("%d_%s", time.Now().Unix(), header.Filename))
	dstFile, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "Не удалось сохранить файл", http.StatusInternalServerError)
		return
	}
	defer dstFile.Close()
	io.Copy(dstFile, file)

	doc := &models.Document{
		UserID:   userID,
		Filename: header.Filename,
		Filepath: dstPath,
	}

	if err := h.service.Upload(r.Context(), doc); err != nil {
		http.Error(w, "Ошибка записи в БД", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Файл загружен"))
}
