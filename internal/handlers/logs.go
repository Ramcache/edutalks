// internal/handlers/admin_logs.go
package handlers

import (
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
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

	"edutalks/internal/logger"
	"go.uber.org/zap"
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
	log := logger.WithCtx(r.Context())

	today := time.Now().Local()
	var days []string
	for i := 0; i < h.Retention; i++ {
		d := today.AddDate(0, 0, -i).Format("2006-01-02")
		if _, err := h.listFilesForDay(d); err == nil {
			days = append(days, d)
		}
	}
	// свежие сверху
	sort.Sort(sort.Reverse(sort.StringSlice(days)))

	log.Info("admin logs: список доступных дней",
		zap.Int("retention_days", h.Retention),
		zap.Int("days_count", len(days)),
	)

	writeJSON(w, http.StatusOK, map[string]any{"days": days})
}

// LogItem — структурированная запись лога для UI
type LogItem struct {
	Time    string         `json:"time,omitempty"`
	Level   string         `json:"level,omitempty"`
	Message string         `json:"msg,omitempty"`
	Fields  map[string]any `json:"fields,omitempty"`
}

// GetLogs
// @Summary      Логи за день
// @Description  Возвращает массив логов за указанный день. Поддерживает фильтрацию по уровню, часу и строке поиска.
// @Tags         admin-logs
// @Security     ApiKeyAuth
// @Produce      json
// @Param        day     query  string true  "Дата (YYYY-MM-DD)"
// @Param        level   query  string false "CSV уровней: debug,info,warn,error,panic,fatal"
// @Param        hour    query  int    false "Час (0-23)"
// @Param        q       query  string false "Поиск по подстроке"
// @Param        limit   query  int    false "Лимит (по умолч. 200, макс. 1000)"
// @Param        cursor  query  int    false "Номер строки для пагинации (по умолч. 0) — счётчик по файлу"
// @Param        order   query  string false "Порядок в выдаче: asc|desc (по умолчанию asc)"
// @Param        tail    query  int    false "Вернуть только последние N совпадений после сортировки (опц.)"
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} map[string]string "unauthorized"
// @Failure      404 {object} map[string]string "day not found"
// @Router       /api/admin/logs [get]
func (h *AdminLogsHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	day := r.URL.Query().Get("day")
	if !reDay.MatchString(day) {
		log.Warn("admin logs: некорректный параметр day", zap.String("day", day))
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
		} else {
			log.Warn("admin logs: некорректный час", zap.String("hour", hourStr))
		}
	}

	limit := clampAtoi(r.URL.Query().Get("limit"), 200, 50, 1000)
	cursor := clampAtoi(r.URL.Query().Get("cursor"), 0, 0, 10_000_000)
	order := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("order"))) // asc|desc
	tail := clampAtoi(r.URL.Query().Get("tail"), 0, 0, 1000)

	log.Info("admin logs: запрос логов",
		zap.String("day", day),
		zap.Strings("levels", levels),
		zap.Any("hour", hourPtr),
		zap.String("q", q),
		zap.Int("limit", limit),
		zap.Int("cursor", cursor),
		zap.String("order", order),
		zap.Int("tail", tail),
	)

	lineNo := 0
	matched := 0
	var items []LogItem

	err := h.forEachDayLineCtx(r.Context(), day, func(raw []byte) bool {
		lineNo++
		if lineNo <= cursor {
			return true // продолжаем читать
		}
		// быстрый фильтр по подстроке
		if qre != nil && !qre.Match(raw) {
			return true
		}
		// парсим JSON
		var obj map[string]any
		if err := json.Unmarshal(raw, &obj); err != nil {
			// пропускаем не-JSON (например, консольный формат)
			return true
		}

		// фильтр по уровню (с алиасами)
		lvl := normalizeLevel(getString(obj, "level"), obj)
		if len(levelSet) > 0 && !levelSet[lvl] {
			return true
		}

		// фильтр по часу
		if hourPtr != nil {
			if t, ok := extractTimeLocal(obj, raw); ok {
				if t.Hour() != *hourPtr {
					return true
				}
			}
		}

		items = append(items, toLogItem(obj))
		matched++
		// ограничение выборки
		return matched < limit
	})

	if err != nil {
		log.Warn("admin logs: файлы за день не найдены", zap.String("day", day), zap.Error(err))
		http.Error(w, "day not found", http.StatusNotFound)
		return
	}

	// сортировка
	if order == "desc" {
		for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
			items[i], items[j] = items[j], items[i]
		}
	}
	// «хвост»
	if tail > 0 && len(items) > tail {
		if order == "desc" {
			items = items[:tail]
		} else {
			items = items[len(items)-tail:]
		}
	}

	// курсор и флаг наличия ещё данных
	next := cursor + matched
	hasMore := matched >= limit // простая и честная эвристика

	log.Info("admin logs: логи отданы",
		zap.String("day", day),
		zap.Int("returned", len(items)),
		zap.Int("next_cursor", next),
		zap.Int("scanned_lines", lineNo),
		zap.Bool("has_more", hasMore),
	)

	writeJSON(w, http.StatusOK, map[string]any{
		"day":        day,
		"items":      items,
		"nextCursor": next,
		"hasMore":    hasMore,
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
	log := logger.WithCtx(r.Context())

	day := r.URL.Query().Get("day")
	if !reDay.MatchString(day) {
		log.Warn("admin logs: некорректный параметр day (stats)", zap.String("day", day))
		http.Error(w, "bad day", http.StatusBadRequest)
		return
	}

	// Массив уровней фиксируем для стабильной структуры ответа.
	allLvls := []string{"DEBUG", "INFO", "WARN", "ERROR", "PANIC", "FATAL"}

	type Lvl map[string]int
	stats := make(map[int]Lvl)
	for hr := 0; hr < 24; hr++ {
		stats[hr] = Lvl{}
		// Инициализируем нулями, чтобы фронт видел стабильную структуру
		for _, lv := range allLvls {
			stats[hr][lv] = 0
		}
	}

	linesScanned := 0

	err := h.forEachDayLineCtx(r.Context(), day, func(raw []byte) bool {
		linesScanned++

		var obj map[string]any
		if err := json.Unmarshal(raw, &obj); err != nil {
			// не JSON — пробуем вытащить время из строки регуляркой
			if t, ok := extractTimeFromRaw(raw); ok {
				stats[t.Hour()]["INFO"]++ // если вообще не знаем уровень — считаем INFO
			}
			return true
		}

		// время — максимально терпим к формату и ключам
		t, ok := extractTimeLocal(obj, raw)
		if !ok {
			// если даже из сырой строки не получилось — пропустим
			return true
		}

		// уровень с алиасами
		lvl := normalizeLevel(getString(obj, "level"), obj)
		if _, known := stats[t.Hour()][lvl]; !known {
			// на всякий случай, если встретится новый уровень
			stats[t.Hour()][lvl] = 0
		}
		stats[t.Hour()][lvl]++
		return true
	})
	if err != nil {
		http.Error(w, "day not found", http.StatusNotFound)
		return
	}

	log.Info("admin logs: статистика по дням сформирована",
		zap.String("day", day),
		zap.Int("scanned_lines", linesScanned),
	)

	writeJSON(w, http.StatusOK, map[string]any{
		"day":   day,
		"stats": stats,
	})
}

// DownloadLog
// @Summary      Скачать лог за день
// @Tags         admin-logs
// @Security     ApiKeyAuth
// @Produce      application/zip,text/plain
// @Param        day query string true "Дата (YYYY-MM-DD)"
// @Param        zip query int false "Если 1 — отдать ZIP со всеми файлами за день"
// @Success      200 {file} file "Лог-файл"
// @Failure      404 {object} map[string]string "file not found"
// @Router       /api/admin/logs/download [get]
func (h *AdminLogsHandler) DownloadLog(w http.ResponseWriter, r *http.Request) {
	log := logger.WithCtx(r.Context())

	day := r.URL.Query().Get("day")
	files, err := h.listFilesForDay(day)
	if err != nil || len(files) == 0 {
		log.Warn("admin logs: файл лога не найден для скачивания", zap.String("day", day))
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	// zip=1 — отдать все файлы дня одним архивом
	if r.URL.Query().Get("zip") == "1" && len(files) > 1 {
		filename := fmt.Sprintf("logs-%s.zip", day)
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		zw := zip.NewWriter(w)
		for _, p := range files {
			fw, err := zw.Create(filepath.Base(p))
			if err != nil {
				continue
			}
			src, err := os.Open(p)
			if err != nil {
				continue
			}
			_, _ = io.Copy(fw, src)
			_ = src.Close()
		}
		_ = zw.Close()
		log.Info("admin logs: скачан ZIP набора файлов", zap.String("day", day), zap.Int("files", len(files)))
		return
	}

	// иначе — первый (можно поменять стратегию на «самый новый»)
	fpath := files[0]
	log.Info("admin logs: скачивание файла лога", zap.String("day", day), zap.String("file", filepath.Base(fpath)))

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filepath.Base(fpath)+"\"")
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
	log := logger.WithCtx(r.Context())

	days := clampAtoi(r.URL.Query().Get("days"), 7, 1, h.Retention)

	today := time.Now().Local()
	allLvls := []string{"DEBUG", "INFO", "WARN", "ERROR", "PANIC", "FATAL"}
	summary := map[string]any{
		"total":  0,
		"levels": map[string]int{},
		"by_day": map[string]map[string]int{},
	}
	levelsTotal := summary["levels"].(map[string]int)
	for _, lv := range allLvls {
		levelsTotal[lv] = 0
	}

	for i := 0; i < days; i++ {
		d := today.AddDate(0, 0, -i).Format("2006-01-02")
		dayStats := map[string]int{}
		for _, lv := range allLvls {
			dayStats[lv] = 0
		}

		_ = h.forEachDayLineCtx(r.Context(), d, func(raw []byte) bool {
			var obj map[string]any
			if err := json.Unmarshal(raw, &obj); err != nil {
				if _, ok := extractTimeFromRaw(raw); ok {
					dayStats["INFO"]++
					levelsTotal["INFO"]++
					summary["total"] = summary["total"].(int) + 1
				}
				return true
			}
			lvl := normalizeLevel(getString(obj, "level"), obj)
			dayStats[lvl]++
			levelsTotal[lvl]++
			summary["total"] = summary["total"].(int) + 1
			return true
		})

		// Сохраняем день только если есть хоть какие-то логи
		nonZero := false
		for _, c := range dayStats {
			if c > 0 {
				nonZero = true
				break
			}
		}
		if nonZero {
			summary["by_day"].(map[string]map[string]int)[d] = dayStats
		}
	}

	log.Info("admin logs: краткая статистика сформирована",
		zap.Int("days", days),
		zap.Int("total", summary["total"].(int)),
	)
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
	today := time.Now().Local().Format("2006-01-02")

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

		// lumberjack: app-2025-09-11T12-34-56.123.log или .gz — проверяем, что имя содержит день.
		if strings.HasPrefix(name, "app-") && strings.Contains(name, day) &&
			(strings.HasSuffix(name, ".log") || strings.HasSuffix(name, ".gz")) {
			files = append(files, filepath.Join(h.LogDir, name))
			continue
		}

		// Перестраховка: иногда встречаются app_YYYY-MM-DD.log
		if strings.Contains(name, strings.ReplaceAll(day, "-", "_")) &&
			(strings.HasSuffix(name, ".log") || strings.HasSuffix(name, ".gz")) {
			files = append(files, filepath.Join(h.LogDir, name))
		}
	}

	// Стабильный порядок (примерно хронологически)
	sort.Strings(files)
	return files, nil
}

// Итерируем все строки всех файлов дня, уважаем контекст запроса и не режем длинные строки
func (h *AdminLogsHandler) forEachDayLineCtx(ctx context.Context, day string, handle func([]byte) bool) error {
	files, err := h.listFilesForDay(day)
	if err != nil || len(files) == 0 {
		return os.ErrNotExist
	}

	for _, path := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		f, err := os.Open(path)
		if err != nil {
			// файл мог быть удалён/перемещён; пропускаем
			continue
		}

		var reader io.Reader = f
		var gr *gzip.Reader

		if strings.HasSuffix(path, ".gz") {
			if gzr, err := gzip.NewReader(f); err == nil {
				gr = gzr
				reader = gr
			} else {
				_ = f.Close()
				continue
			}
		}

		sc := bufio.NewScanner(reader)
		// 4 МБ на строку — чтобы не обрезать длинные JSON-записи
		buf := make([]byte, 0, 256*1024)
		sc.Buffer(buf, 4*1024*1024)

		for sc.Scan() {
			select {
			case <-ctx.Done():
				if gr != nil {
					_ = gr.Close()
				}
				_ = f.Close()
				return ctx.Err()
			default:
			}
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
	// популярный синоним
	if key == "msg" {
		if v, ok := m["message"]; ok {
			if s, ok := v.(string); ok {
				return s
			}
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

func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

// маппинг «сырая запись -> структурированная»
func toLogItem(obj map[string]any) LogItem {
	li := LogItem{
		Time:    getString(obj, "time"),
		Level:   normalizeLevel(getString(obj, "level"), obj),
		Message: getString(obj, "msg"),
		Fields:  map[string]any{},
	}

	// популярные «колоночные» поля
	keys := []string{
		"method", "path", "status", "url",
		"remote_ip", "ip", "user_id", "request_id", "req_id", "requestId", "rid",
		"stack", "error",
	}
	for _, k := range keys {
		if v, ok := obj[k]; ok {
			li.Fields[k] = v
		}
	}

	// прицепим остальное в details
	for k, v := range obj {
		if k == "time" || k == "level" || k == "msg" || k == "message" {
			continue
		}
		if _, ok := li.Fields[k]; ok {
			continue
		}
		li.Fields[k] = v
	}
	return li
}

// ===== парсинг времени и уровней с запасом прочности =====

var tsRE = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+\-]\d{2}:\d{2})`)

func extractTimeLocal(obj map[string]any, raw []byte) (time.Time, bool) {
	// 1) строковые ключи
	for _, k := range []string{"time", "ts", "timestamp"} {
		if s := getString(obj, k); s != "" {
			if t, ok := parseTimestamp(s); ok {
				return t.Local(), true
			}
		}
	}
	// 2) numeric unix(ts)
	if v, ok := obj["ts"]; ok {
		switch vv := v.(type) {
		case float64:
			// различаем секунды/миллисекунды по порядку величины
			if vv > 1e12 {
				sec := int64(vv) / 1000
				nsec := (int64(vv) % 1000) * int64(time.Millisecond)
				return time.Unix(sec, nsec).Local(), true
			}
			return time.Unix(int64(vv), 0).Local(), true
		case json.Number:
			if i, err := vv.Int64(); err == nil {
				if i > 1e12 {
					sec := i / 1000
					nsec := (i % 1000) * int64(time.Millisecond)
					return time.Unix(sec, nsec).Local(), true
				}
				return time.Unix(i, 0).Local(), true
			}
		}
	}
	// 3) из сырой строки регуляркой
	if t, ok := extractTimeFromRaw(raw); ok {
		return t.Local(), true
	}
	return time.Time{}, false
}

func extractTimeFromRaw(raw []byte) (time.Time, bool) {
	m := tsRE.Find(raw)
	if len(m) == 0 {
		return time.Time{}, false
	}
	if t, ok := parseTimestamp(string(m)); ok {
		return t, true
	}
	return time.Time{}, false
}

func normalizeLevel(lvl string, obj map[string]any) string {
	l := strings.ToUpper(strings.TrimSpace(lvl))
	if l != "" {
		return levelAlias(l)
	}
	// пробуем альтернативные ключи
	for _, k := range []string{"severity", "lvl"} {
		if s, ok := obj[k]; ok {
			if str, ok := s.(string); ok {
				return levelAlias(strings.ToUpper(strings.TrimSpace(str)))
			}
			// числовые уровни — сведём к приблизительным
			if num, ok := s.(float64); ok {
				return numericLevel(num)
			}
		}
	}
	return "INFO"
}

func levelAlias(l string) string {
	switch l {
	case "TRACE":
		return "DEBUG"
	case "WARNING":
		return "WARN"
	case "ERR":
		return "ERROR"
	case "CRITICAL":
		return "FATAL"
	default:
		return l
	}
}

func numericLevel(n float64) string {
	// очень условное соответствие
	switch {
	case n <= 10:
		return "DEBUG"
	case n <= 20:
		return "INFO"
	case n <= 30:
		return "WARN"
	case n <= 40:
		return "ERROR"
	case n <= 50:
		return "PANIC"
	default:
		return "FATAL"
	}
}

// normalizeRFC3339Frac приводит количество знаков после секунды к 9 (RFC3339Nano).
// Работает и с 'Z', и с часовыми смещениями (+03:00 / -07:00), и с запятой в долях.
func normalizeRFC3339Frac(s string) string {
	// смена запятой на точку, если вдруг
	if strings.Contains(s, ",") {
		s = strings.ReplaceAll(s, ",", ".")
	}
	// ищем точку между секундами и таймзоной
	// сначала находим позицию 'T' (или пробела), чтобы не зацепить дату
	sep := strings.IndexByte(s, 'T')
	if sep == -1 {
		sep = strings.IndexByte(s, ' ')
		if sep == -1 {
			return s
		}
	}
	dot := strings.IndexByte(s[sep:], '.')
	if dot == -1 {
		// нет долей секунд — оставляем как есть
		return s
	}
	dot += sep

	// ищем начало таймзоны после точки: 'Z' или '+' или '-'
	zoneIdx := -1
	for i := dot + 1; i < len(s); i++ {
		switch s[i] {
		case 'Z', '+', '-':
			zoneIdx = i
			goto done
		}
	}
done:
	if zoneIdx == -1 {
		// сомнительная строка, не трогаем
		return s
	}

	frac := s[dot+1 : zoneIdx]
	if len(frac) == 9 {
		return s
	}
	if len(frac) > 9 {
		frac = frac[:9]
	} else {
		frac = frac + strings.Repeat("0", 9-len(frac))
	}
	return s[:dot+1] + frac + s[zoneIdx:]
}

func parseTimestamp(s string) (time.Time, bool) {
	// Сначала пробуем нормализовать до RFC3339Nano (ровно 9 знаков в долях)
	sn := normalizeRFC3339Frac(s)
	// Базовые варианты
	layouts := []string{
		time.RFC3339Nano,                      // 2006-01-02T15:04:05.999999999Z07:00
		time.RFC3339,                          // 2006-01-02T15:04:05Z07:00 (без долей)
		"2006-01-02 15:04:05Z07:00",           // с пробелом, без долей
		"2006-01-02 15:04:05.000000000Z07:00", // с пробелом, с долями (ровно 9)
		"2006-01-02 15:04:05",                 // вообще без таймзоны
		"2006-01-02T15:04:05",                 // без таймзоны (ISO, без долей)
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, sn); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
