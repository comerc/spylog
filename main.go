package spylog

import (
	"bytes"
	"context"
	"log/slog"
	"runtime"
	"strconv"
	"sync"
)

// ready for t.Parallel() and multiple t.Run()

var (
	handlerInstance *logHandler
	handlerOnce     sync.Once
)

func Init(logger *slog.Logger) {
	handlerOnce.Do(func() {
		// w := io.Discard // mute all logs
		// if flag.Lookup("test.v") != nil {
		// 	w = os.Stdout
		// }
		handlerInstance = &logHandler{
			handlers: make(map[string]map[string]*moduleLogHandler),
			handler:  logger.Handler(), // slog.NewTextHandler(w, nil),
		}
	})
}

type logHandler struct {
	mu       sync.Mutex
	current  sync.Map
	handlers map[string]map[string]*moduleLogHandler
	handler  slog.Handler
}

func GetModuleLogHandler(moduleName, testName string, init func()) *moduleLogHandler {
	h := handlerInstance
	h.mu.Lock()
	defer h.mu.Unlock()
	h.current.Store(getGID(), testName) // need for WithAttrs
	handlers, ok := h.handlers[moduleName]
	if !ok {
		handlers = make(map[string]*moduleLogHandler)
		h.handlers[moduleName] = handlers
	}
	handler, ok := handlers[testName]
	if !ok {
		handler = &moduleLogHandler{}
		h.handlers[moduleName][testName] = handler
	}
	slog.SetDefault(slog.New(h))
	init() // for slog.With("module", "name")
	return handler
}

func (h *logHandler) Handle(ctx context.Context, r slog.Record) error {
	h.handler.Handle(ctx, r)
	return nil
}

func (h *logHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	var module string
	for _, attr := range attrs {
		if attr.Key == "module" {
			module = attr.Value.String()
			break
		}
	}

	if module == "" {
		return h
	}

	if testName, ok := h.current.Load(getGID()); ok {
		if handler, exists := h.handlers[module][testName.(string)]; exists {
			return handler
		}
	}
	return h
}

func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

func (h *logHandler) WithGroup(name string) slog.Handler {
	return h
}

func (h *logHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

type moduleLogHandler struct {
	Records []*slog.Record
}

func (h *moduleLogHandler) Handle(ctx context.Context, r slog.Record) error {
	handlerInstance.handler.Handle(ctx, r)
	h.Records = append(h.Records, &r)
	return nil
}

func (h *moduleLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *moduleLogHandler) WithGroup(name string) slog.Handler {
	return h
}

func (h *moduleLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return handlerInstance.handler.Enabled(ctx, level)
}

func GetAttrValue(record *slog.Record, key string) string {
	var value string
	record.Attrs(func(attr slog.Attr) bool {
		if attr.Key == key {
			value = attr.Value.String()
			return false
		}
		return true
	})
	return value
}
