package mangoprovider

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/logger"
	"github.com/luevano/libmangal/progress"
)

// Logger logs through the libmangal logger with the provider ID as prefix, one per provider.
//
// It is the logger created by libmangal and used internally. Mangal also plugs into the logger
// (via (*lm.Client).Logger()) and uses it for info display. Messages logged before it's set are
// kept and logged once it is.
type Logger struct {
	mu      sync.Mutex
	prefix  string
	source  string
	name    string
	logger  *logger.Logger
	backlog []string
}

func NewLogger(prefix string) *Logger {
	name := strings.TrimSuffix(prefix, ": ")
	return &Logger{prefix: prefix, source: name, name: name}
}

// NewProviderLogger is the logger of a provider: logs prefixed with its ID, statuses with its name.
func NewProviderLogger(info libmangal.ProviderInfo) *Logger {
	l := NewLogger(info.ID + ": ")
	if info.Name != "" {
		l.name = info.Name
	}
	return l
}

// SetLogger sets the libmangal logger and logs the messages kept until now.
func (l *Logger) SetLogger(logger *logger.Logger) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logger = logger
	for _, msg := range l.backlog {
		l.logger.Log("[backlog] %s", msg)
	}
	l.backlog = nil
}

// Log logs the message, a nil Logger (e.g. in tests) doesn't log anything.
func (l *Logger) Log(format string, a ...any) {
	if l == nil {
		return
	}
	msg := l.prefix + fmt.Sprintf(format, a...)

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.logger == nil {
		l.backlog = append(l.backlog, msg)
		return
	}
	l.logger.Log("%s", msg)
}

// Status logs the message and reports it (prefixed with the provider name) to the progress
// reporter of ctx, for what the user waits on: mirror switches, challenges, rate limits.
func (l *Logger) Status(ctx context.Context, format string, a ...any) {
	if l == nil {
		return
	}
	l.Log(format, a...)
	progress.Report(ctx, progress.Status{Source: l.source, Message: l.name + ": " + fmt.Sprintf(format, a...)})
}

// Progress reports a counted step (e.g. a search page) without logging it.
func (l *Logger) Progress(ctx context.Context, done, total int, format string, a ...any) {
	if l == nil {
		return
	}
	progress.Report(ctx, progress.Status{Source: l.source, Message: l.name + ": " + fmt.Sprintf(format, a...), Done: done, Total: total})
}
