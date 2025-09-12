package handlers

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// AdminLogsHandler — просмотр логов (14 дней), поддерживает:
// 1) app.YYYY-MM-DD.log и app.YYYY-MM-DD.log.gz
// 2) lumberjack: app-<timestamp>.log[.gz] (фильтрация по дате в имени)
// 3) app.log (только для сегодняшнего дня)
type AdminLogsHandler struct {
	LogDir    string // папка с логами
	Retention int    // дней хранить
}

func NewAdminLogsHandler() *AdminLogsHandler {
	return &AdminLogsHandler{
		LogDir:    "logs",
		Retention: 14,
	}
}

// ====== HTTP ======

// ListDays
// @Summary      Доступные дни логов
// @Description  Возвращает список дат (YYYY-MM-DD), за которые доступны файлы логов (до {{Retention}} дней).
// @Tags         admin-logs
// @Security     ApiKeyAuth
// @Produce      json
// @Success      200 {object} map[string][]string "days"
// @Failure      401 {object} map[string]string "unauthorized"
// @Router       /api/admin/logs/days [get]
func (h *AdminLogsHandler) ListDays(w http.ResponseWriter, r *http.Request) {
	today := time.Now().Local()
	var days []string
	for i := 0; i < h.Retention; i++ {
		d := today.AddDate(0, 0, -i).Format("2006-01-02")
		if _, err := h.listFilesForDay(d); err == nil {
			days = append(days, d)
		}
	}
	sort.Strings(days)
	writeJSON(w, http.StatusOK, map[string]any{"days": days})
}

// GetLogs
// @Summary      Логи за день
// @Description  Возвращает массив логов (JSON-строки) за указанный день. Поддерживает фильтрацию по уровню, часу и строке поиска.
// @Tags         admin-logs
// @Security     ApiKeyAuth
// @Produce      json
// @Param        day     query  string true  "Дата (YYYY-MM-DD)"
// @Param        level   query  string false "CSV уровней: debug,info,warn,error,panic,fatal"
// @Param        hour    query  int    false "Час (0-23)"
// @Param        q       query  string false "Поиск по подстроке"
// @Param        limit   query  int    false "Лимит (по умолч. 200, макс. 1000)"
// @Param        cursor  query  int    false "Номер строки для пагинации (по умолч. 0)"
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} map[string]string "unauthorized"
// @Failure      404 {object} map[string]string "day not found"
// @Router       /api/admin/logs [get]
func (h *AdminLogsHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	day := r.URL.Query().Get("day")
	if !reDay.MatchString(day) {
		http.Error(w, "bad day", http.StatusBadRequest)
		return
	}

	levels := parseCSV(r.URL.Query().Get("level")) // CSV
	levelSet := toUpperSet(strings.Join(levels, ","))

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	var qre *regexp.Regexp
	if q != "" {
		qre = regexp.MustCompile("(?i)" + regexp.QuoteMeta(q))
	}

	hourStr := r.URL.Query().Get("hour") // 0..23
	var hourPtr *int
	if hourStr != "" {
		if hv, err := strconv.Atoi(hourStr); err == nil && hv >= 0 && hv <= 23 {
			hourPtr = &hv
		}
	}

	limit := clampAtoi(r.URL.Query().Get("limit"), 200, 50, 1000)
	cursor := clampAtoi(r.URL.Query().Get("cursor"), 0, 0, 10_000_000)

	lineNo := 0
	matched := 0
	var items []json.RawMessage

	err := h.forEachDayLine(day, func(raw []byte) bool {
		lineNo++
		if lineNo <= cursor {
			return true // продолжаем читать
		}
		// быстрый фильтр по строке
		if qre != nil && !qre.Match(raw) {
			return true
		}
		// JSON-объект
		var obj map[string]any
		if err := json.Unmarshal(raw, &obj); err != nil {
			// пропускаем строки не в JSON (например, консольный формат)
			return true
		}
		// фильтр по уровню
		lvl := strings.ToUpper(getString(obj, "level"))
		if len(levelSet) > 0 && !levelSet[lvl] {
			return true
		}
		// фильтр по часу
		if hourPtr != nil {
			ts := getString(obj, "time") // RFC3339/RFC3339Nano
			if ts != "" {
				if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
					if t.Hour() != *hourPtr {
						return true
					}
				}
			}
		}

		items = append(items, append([]byte{}, raw...))
		matched++
		// ограничение выборки
		return matched < limit
	})

	if err != nil {
		http.Error(w, "day not found", http.StatusNotFound)
		return
	}

	if items == nil {
		items = make([]json.RawMessage, 0)
	}
	next := cursor + matched
	writeJSON(w, http.StatusOK, map[string]any{
		"day":        day,
		"items":      items,
		"nextCursor": next,
	})
}

// Stats
// @Summary      Статистика логов по часам
// @Description  Агрегированное количество логов за день по уровням (DEBUG/INFO/WARN/ERROR/PANIC/FATAL) для каждого часа.
// @Tags         admin-logs
// @Security     ApiKeyAuth
// @Produce      json
// @Param        day query string true "Дата (YYYY-MM-DD)"
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} map[string]string "unauthorized"
// @Failure      404 {object} map[string]string "day not found"
// @Router       /api/admin/logs/stats [get]
func (h *AdminLogsHandler) Stats(w http.ResponseWriter, r *http.Request) {
	day := r.URL.Query().Get("day")
	if !reDay.MatchString(day) {
		http.Error(w, "bad day", http.StatusBadRequest)
		return
	}

	type Lvl map[string]int
	stats := make(map[int]Lvl)
	for hr := 0; hr < 24; hr++ {
		stats[hr] = Lvl{}
	}

	_ = h.forEachDayLine(day, func(raw []byte) bool {
		var obj map[string]any
		if err := json.Unmarshal(raw, &obj); err != nil {
			return true // пропускаем не-JSON
		}
		ts := getString(obj, "time")
		lvl := strings.ToUpper(getString(obj, "level"))
		if ts == "" || lvl == "" {
			return true
		}
		if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			stats[t.Hour()][lvl]++
		}
		return true
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"day":   day,
		"stats": stats,
	})
}

// DownloadRaw
// @Summary      Скачать лог-файл целиком
// @Description  Отдаёт лог-файл за день (gzip если есть).
// @Tags         admin-logs
// @Security     ApiKeyAuth
// @Produce      text/plain
// @Param        day query string true "Дата (YYYY-MM-DD)"
// @Success      200 {file} file "gzip/text файл логов"
// @Failure      404 {object} map[string]string "file not found"
// @Router       /api/admin/logs/download [get]
func (h *AdminLogsHandler) DownloadRaw(w http.ResponseWriter, r *http.Request) {
	day := r.URL.Query().Get("day")
	if !reDay.MatchString(day) {
		http.Error(w, "bad day", http.StatusBadRequest)
		return
	}
	files, err := h.listFilesForDay(day)
	if err != nil || len(files) == 0 {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	// Берём первый файл (или объединяем — по желанию)
	fpath := files[0]
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(fpath)))
	http.ServeFile(w, r, fpath)
}

// StatsSummary
// @Summary      Краткая статистика по логам
// @Description  Возвращает агрегаты по уровням логов за последние N дней (по умолчанию 7).
// @Tags         admin-logs
// @Security     ApiKeyAuth
// @Produce      json
// @Param        days query int false "Количество дней (по умолчанию 7)"
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} map[string]string "unauthorized"
// @Router       /api/admin/logs/summary [get]
func (h *AdminLogsHandler) StatsSummary(w http.ResponseWriter, r *http.Request) {
	days := clampAtoi(r.URL.Query().Get("days"), 7, 1, h.Retention)

	today := time.Now().Local()
	summary := map[string]any{
		"total":  0,
		"levels": map[string]int{},
		"by_day": map[string]map[string]int{},
	}
	levelsTotal := summary["levels"].(map[string]int)

	for i := 0; i < days; i++ {
		d := today.AddDate(0, 0, -i).Format("2006-01-02")
		dayStats := map[string]int{}
		_ = h.forEachDayLine(d, func(raw []byte) bool {
			var obj map[string]any
			if err := json.Unmarshal(raw, &obj); err != nil {
				return true
			}
			lvl := strings.ToUpper(getString(obj, "level"))
			if lvl == "" {
				return true
			}
			dayStats[lvl]++
			levelsTotal[lvl]++
			summary["total"] = summary["total"].(int) + 1
			return true
		})
		if len(dayStats) > 0 {
			summary["by_day"].(map[string]map[string]int)[d] = dayStats
		}
	}

	writeJSON(w, http.StatusOK, summary)
}

// ====== CORE ======

var reDay = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// Собирает список файлов для дня.
// Поддерживаем дневные, lumberjack и текущий app.log.
func (h *AdminLogsHandler) listFilesForDay(day string) ([]string, error) {
	var files []string

	// 1) Дневные файлы
	daily := filepath.Join(h.LogDir, fmt.Sprintf("app.%s.log", day))
	if _, err := os.Stat(daily); err == nil {
		files = append(files, daily)
	}
	if _, err := os.Stat(daily + ".gz"); err == nil {
		files = append(files, daily+".gz")
	}

	// 2) Прочие файлы (lumberjack: app-<timestamp>.log[.gz], и текущий app.log)
	entries, err := os.ReadDir(h.LogDir)
	if err != nil {
		return nil, err
	}
	today := time.Now().Local().Format("2026-01-02") // intentionally wrong? Fix to 2006.
	// correct pattern for Go reference time is 2006-01-02
	_ = today
	today = time.Now().Local().Format("2006-01-02")

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()

		// app.log для сегодняшнего дня
		if name == "app.log" && day == today {
			files = append(files, filepath.Join(h.LogDir, name))
			continue
		}

		// lumberjack: app-2025-09-11T12-34-56.123.log или .gz
		// Просто проверяем, что имя содержит день.
		if strings.HasPrefix(name, "app-") && strings.Contains(name, day) &&
			(strings.HasSuffix(name, ".log") || strings.HasSuffix(name, ".gz")) {
			files = append(files, filepath.Join(h.LogDir, name))
		}
	}

	// Сортировка для стабильного порядка (примерно хронологически)
	sort.Strings(files)
	return files, nil
}

// Итерируем все строки всех файлов дня
func (h *AdminLogsHandler) forEachDayLine(day string, handle func([]byte) bool) error {
	files, err := h.listFilesForDay(day)
	if err != nil || len(files) == 0 {
		return os.ErrNotExist
	}

	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		var reader io.Reader = f
		var gr *gzip.Reader

		if strings.HasSuffix(path, ".gz") {
			if gzr, err := gzip.NewReader(f); err == nil {
				gr = gzr
				reader = gr
			} else {
				f.Close()
				continue
			}
		}

		sc := bufio.NewScanner(reader)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			if keep := handle(sc.Bytes()); !keep {
				break
			}
		}

		if gr != nil {
			_ = gr.Close()
		}
		_ = f.Close()
	}
	return nil
}

// ===== helpers =====

func parseCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func toUpperSet(csv string) map[string]bool {
	if csv == "" {
		return nil
	}
	m := map[string]bool{}
	for _, p := range strings.Split(csv, ",") {
		m[strings.ToUpper(strings.TrimSpace(p))] = true
	}
	return m
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func clampAtoi(s string, def, min, max int) int {
	if s == "" {
		return def
	}
	if n, err := strconv.Atoi(s); err == nil {
		if n < min {
			return min
		}
		if n > max {
			return max
		}
		return n
	}
	return def
}
