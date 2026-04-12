package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	lumberjackV2 "gopkg.in/natefinch/lumberjack.v2"
)

// dailyRotateWriter appends to files named {prefix}-{YYYY-MM-DD}{ext} using the configured IANA timezone (or time.Local).
// It swaps the underlying lumberjack when that zone's calendar date changes. Within the same day, max_size / max_backups / compress still apply.
// When max_age > 0, files in the log directory matching the daily naming pattern older than max_age calendar days are removed.
type dailyRotateWriter struct {
	mu sync.Mutex
	// cfg is read-only after creation.
	cfg *Config
	loc *time.Location

	dir    string
	prefix string
	ext    string

	day string
	lj  *lumberjackV2.Logger

	lastClean   time.Time
	dailyNameRE *regexp.Regexp
}

func newDailyRotateWriter(cfg *Config) *dailyRotateWriter {
	logDir := filepath.Dir(cfg.Filename)
	base := filepath.Base(cfg.Filename)
	ext := filepath.Ext(base)
	prefix := base[:len(base)-len(ext)]
	re := regexp.MustCompile(`^` + regexp.QuoteMeta(prefix) + `-(\d{4}-\d{2}-\d{2})`)
	return &dailyRotateWriter{
		cfg:         cfg,
		loc:         resolveLogLocation(cfg.Timezone),
		dir:         logDir,
		prefix:      prefix,
		ext:         ext,
		dailyNameRE: re,
	}
}

func (w *dailyRotateWriter) pathForDay(day string) string {
	return filepath.Join(w.dir, fmt.Sprintf("%s-%s%s", w.prefix, day, w.ext))
}

func (w *dailyRotateWriter) openForDay(day string) error {
	if w.lj != nil {
		_ = w.lj.Close()
		w.lj = nil
	}
	w.lj = &lumberjackV2.Logger{
		Filename:   w.pathForDay(day),
		MaxSize:    w.cfg.MaxSize,
		MaxBackups: w.cfg.MaxBackups,
		MaxAge:     w.cfg.MaxAge,
		Compress:   w.cfg.Compress,
		LocalTime:  true,
	}
	w.day = day
	return nil
}

func (w *dailyRotateWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	day := time.Now().In(w.loc).Format("2006-01-02")
	if w.lj == nil || w.day != day {
		if err := w.openForDay(day); err != nil {
			return 0, err
		}
		if w.cfg.MaxAge > 0 {
			w.cleanupOldDailyFiles()
			w.lastClean = time.Now()
		}
	} else if w.cfg.MaxAge > 0 && (w.lastClean.IsZero() || time.Since(w.lastClean) > time.Hour) {
		w.cleanupOldDailyFiles()
		w.lastClean = time.Now()
	}

	return w.lj.Write(p)
}

// Sync is a best-effort flush; lumberjack does not expose the underlying *os.File.
func (w *dailyRotateWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return nil
}

// Close releases the current lumberjack file handle.
func (w *dailyRotateWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.lj == nil {
		return nil
	}
	err := w.lj.Close()
	w.lj = nil
	return err
}

func (w *dailyRotateWriter) cleanupOldDailyFiles() {
	if w.cfg.MaxAge <= 0 {
		return
	}
	cutoff := time.Now().In(w.loc).AddDate(0, 0, -w.cfg.MaxAge)
	cutoffDay := time.Date(cutoff.Year(), cutoff.Month(), cutoff.Day(), 0, 0, 0, 0, w.loc)

	pattern := filepath.Join(w.dir, w.prefix+"-*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}
	for _, path := range matches {
		base := filepath.Base(path)
		if !isDailyLogFilename(base, w.ext) {
			continue
		}
		sub := w.dailyNameRE.FindStringSubmatch(base)
		if len(sub) < 2 {
			continue
		}
		t, err := time.ParseInLocation("2006-01-02", sub[1], w.loc)
		if err != nil {
			continue
		}
		dayStart := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, w.loc)
		if dayStart.Before(cutoffDay) {
			_ = os.Remove(path)
		}
	}
}

func isDailyLogFilename(base, ext string) bool {
	if strings.HasSuffix(base, ext+".gz") {
		return true
	}
	return strings.HasSuffix(base, ext)
}
