package mangoprovider

import (
	"fmt"

	"github.com/luevano/libmangal"
)

var (
	logBacklog []string
	logger     *libmangal.Logger
)

// Log calls (*libmangal.Logger).Log(msg) when set, else it appends the message to the backlog.
// Once the logger is set, it logs all backlog on the next log.
//
// It is the logger created by libmangal and used internally. Mangal also plugs into the logger (via (*libmangal.Client).Logger())
// and uses it for info display.
func Log(msg string) {
	switch {
	case logger != nil:
		if len(logBacklog) != 0 {
			for _, msg := range logBacklog {
				logger.Log(fmt.Sprintf("[logBacklog] %s", msg))
			}
			logBacklog = nil
		}
		logger.Log(msg)
	default:
		logBacklog = append(logBacklog, msg)
	}
}
