package prettylog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"strconv"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	reset = "\033[0m"

	black        = 30
	red          = 31
	green        = 32
	yellow       = 33
	blue         = 34
	magenta      = 35
	cyan         = 36
	lightGray    = 37
	darkGray     = 90
	lightRed     = 91
	lightGreen   = 92
	lightYellow  = 93
	lightBlue    = 94
	lightMagenta = 95
	lightCyan    = 96
	white        = 97
)

func colorize(colorCode int, v string) string {
	return fmt.Sprintf("\033[%sm%s%s", strconv.Itoa(colorCode), v, reset)
}

type Handler struct {
	h        slog.Handler
	b        *bytes.Buffer
	m        *sync.Mutex
	file     *log.Logger
	fileLock *sync.Mutex
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.h.Enabled(ctx, level)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{h: h.h.WithAttrs(attrs), b: h.b, m: h.m}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{h: h.h.WithGroup(name), b: h.b, m: h.m}
}

const (
	timeFormat = "[2006-01-02 15:04:05]"
	// timeFormat = "[2006-01-02 15:04:05.000]"
)

func PrettylogNewHandler(opts *slog.HandlerOptions, logFile string) *Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	b := &bytes.Buffer{}

	// 이건 파일 크기로 로그
	fileLogger := log.New(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    10, // megabytes
		MaxBackups: 100,
		MaxAge:     28,   // days
		Compress:   true, // disabled by default
	}, "", log.LstdFlags)

	// 이건 시간으로 로그
	// initialLogFile := "/log/log-%Y-%m-%d-%H.log"
	// rotatelogs, err := rotatelogs.New(
	// 	initialLogFile,
	// 	//1일 뒤에는 삭제 1을 늘리면 날짜를 늘림
	// 	rotatelogs.WithMaxAge(1*24*time.Hour),
	// 	rotatelogs.WithRotationTime(1*time.Hour))
	// if err != nil {
	// 	panic(err)
	// }

	return &Handler{
		b: b,
		h: slog.NewJSONHandler(b, &slog.HandlerOptions{
			Level:       opts.Level,
			AddSource:   opts.AddSource,
			ReplaceAttr: suppressDefaults(opts.ReplaceAttr),
		}),
		m:        &sync.Mutex{},
		file:     fileLogger,
		fileLock: &sync.Mutex{},
	}
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {

	level := " " + r.Level.String() + ": "
	var colorizeLevel string

	switch r.Level {
	case slog.LevelDebug:
		colorizeLevel = colorize(lightGreen, level)
	case slog.LevelInfo:
		colorizeLevel = colorize(lightBlue, level)
	case slog.LevelWarn:
		colorizeLevel = colorize(lightYellow, level)
	case slog.LevelError:
		colorizeLevel = colorize(lightRed, level)
	}

	attrs, err := h.computeAttrs(ctx, r)
	if err != nil {
		return err
	}

	bytes, err := json.MarshalIndent(attrs, "", "  ")
	if err != nil {
		return fmt.Errorf("error when marshaling attrs: %w", err)
	}

	consoleOutput := fmt.Sprint(
		colorize(darkGray, r.Time.Format(timeFormat)),
		colorizeLevel,
		colorize(white, r.Message),
		colorize(darkGray, string(bytes)),
	)
	fileOutput := fmt.Sprint(
		// r.Time.Format(timeFormat),
		level,
		r.Message,
		string(bytes),
	)

	fmt.Println(consoleOutput)

	h.fileLock.Lock()
	defer h.fileLock.Unlock()
	h.file.Println(fileOutput)

	return nil
}

func (h *Handler) computeAttrs(
	ctx context.Context,
	r slog.Record,
) (map[string]any, error) {
	h.m.Lock()
	defer func() {
		h.b.Reset()
		h.m.Unlock()
	}()
	if err := h.h.Handle(ctx, r); err != nil {
		return nil, fmt.Errorf("error when calling inner handler's Handle: %w", err)
	}

	var attrs map[string]any
	err := json.Unmarshal(h.b.Bytes(), &attrs)
	if err != nil {
		return nil, fmt.Errorf("error when unmarshaling inner handler's Handle result: %w", err)
	}
	return attrs, nil
}

func suppressDefaults(
	next func([]string, slog.Attr) slog.Attr,
) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey ||
			a.Key == slog.LevelKey ||
			a.Key == slog.MessageKey {
			return slog.Attr{}
		}
		if next == nil {
			return a
		}
		return next(groups, a)
	}
}
