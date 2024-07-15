package mangoprovider

import (
	"fmt"

	"github.com/luevano/libmangal/logger"
)

var (
	logBacklog []string
	logger_    *logger.Logger
	prefix_    string
)

// Log calls (*lm.Logger).Log(msg) when set, else it appends the message to the backlog.
// Once the logger is set, it logs all backlog on the next log.
//
// It is the logger created by libmangal and used internally. Mangal also plugs into the logger (via (*lm.Client).Logger())
// and uses it for info display.
func Log(format string, a ...any) {
	_fmt := prefix_ + format
	switch {
	case logger_ != nil:
		if len(logBacklog) != 0 {
			for _, msg := range logBacklog {
				logger_.Log("[logBacklog] %s", msg)
			}
			logBacklog = nil
		}
		logger_.Log(_fmt, a...)
	default:
		logBacklog = append(logBacklog, fmt.Sprintf(_fmt, a...))
	}
}
