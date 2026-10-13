package mangoprovider

import (
	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/logger"
)

// Logger is the logger of a provider, see logger.ProviderLogger.
type Logger = logger.ProviderLogger

// NewProviderLogger is the logger of a provider: logs prefixed with its ID, statuses with its name.
func NewProviderLogger(info libmangal.ProviderInfo) *Logger {
	return logger.NewProviderLogger(info.ID, info.Name)
}
