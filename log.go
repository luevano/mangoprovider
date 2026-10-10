package mangoprovider

import (
	"fmt"
	"sync"

	"github.com/luevano/libmangal/logger"
)

// Logger logs through the libmangal logger with the provider ID as prefix, one per provider.
//
// It is the logger created by libmangal and used internally. Mangal also plugs into the logger
// (via (*lm.Client).Logger()) and uses it for info display. Messages logged before it's set are
// kept and logged once it is.
type Logger struct {
	mu      sync.Mutex
	prefix  string
	logger  *logger.Logger
	backlog []string
}

func NewLogger(prefix string) *Logger {
	return &Logger{prefix: prefix}
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
