package mangoprovider

import "github.com/luevano/libmangal"

var logger *libmangal.Logger

// Log calls (*libmangal.Logger).Log() which is set by libmangal via Provider.SetLogger.
//
// It is the logger created by libmangal and used internally. Mangal also plugs into the logger (via (*libmangal.Client).Logger()) and uses it for info display.
// 
// It can also be set anywhere else, but usually it's something libmangal manages and since the Provider implementation
// handles the "setup" we can intercept it and use externally (like here in mangoprovider).
func Log(msg string) {
	logger.Log(msg)
}
