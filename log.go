package mangoprovider

import "github.com/luevano/libmangal"

var logger *libmangal.Logger

// Log calls (*libmangal.Logger).Log() which is set by libmangal via Provider.SetLogger.
//
// It is the logger created by libmangal and used internally. Mangal also plugs into the logger (via (*libmangal.Client).Logger())
// and uses it for info display.
// 
// It can also be set anywhere else, but usually it's something libmangal manages and since the Provider implementation
// handles the "setup" we can intercept it and use externally (like here in mangoprovider).
//
// The logger is set once the provider is loaded, meaning that it will not be available for scraper/scraper.go for example,
// but it will for scraper/search_mangas.go as by that point the provider has been loaded.
//
// TODO: provider mangal zerologger and use it when the *libmangal.Logger isn't available?
func Log(msg string) {
	// In case it is called before it is set
	if logger != nil {
		logger.Log(msg)
	}
}
