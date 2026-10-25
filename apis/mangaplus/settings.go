package mangaplus

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangoplus"
	mango "github.com/luevano/mangoprovider"
)

var defaults = mango.DefaultMangaPlusOptions()

var settings = []libmangal.ProviderSetting{
	{
		Key:         "quality",
		Name:        "Image quality",
		Description: "Page image quality.",
		Kind:        libmangal.SettingEnum,
		Default:     defaults.Quality,
		Values:      []string{"low", "high", "super_high"},
	},
	{
		Key:         "api",
		Name:        "API",
		Description: `"app" (the official app's, with the secret file's device or a registered free one), "web" (the website's, no device or secret: only what the website shows, e.g. a series' first and latest chapters; 1 request/s) or "auto" (web when there's no secret and no free device yet).`,
		Kind:        libmangal.SettingEnum,
		Default:     string(defaults.API),
		Values:      []string{string(mangoplus.APIApp), string(mangoplus.APIWeb), string(mangoplus.APIAuto)},
	},
	{
		Key:         "viewer",
		Name:        "Chapter viewer",
		Description: `"app" (same as the official app, needed for subscription images) or "legacy" (deprecated, old endpoint).`,
		Kind:        libmangal.SettingEnum,
		Default:     string(defaults.Viewer),
		Values:      []string{string(mangoplus.ViewerApp), string(mangoplus.ViewerLegacy)},
	},
	{
		Key:         "request_interval",
		Name:        "Request interval",
		Description: `Minimum time between API requests, e.g. "500ms" or "1s".`,
		Kind:        libmangal.SettingDuration,
		Default:     defaults.RequestInterval.String(),
		Warning:     `Lowering or removing it ("0s") risks getting the device locked out for 10+ minutes.`,
	},
	{
		Key:         "os_version",
		Name:        "OS version",
		Description: "The OS version sent to the API.",
		Kind:        libmangal.SettingString,
		Default:     defaults.OSVersion,
	},
	{
		Key:         "app_version",
		Name:        "App version",
		Description: "The app version sent to the API.",
		Kind:        libmangal.SettingString,
		Default:     defaults.AppVersion,
	},
	{
		Key:         "android_id",
		Name:        "Android ID",
		Description: "The Android ID sent to the API, random when empty.",
		Kind:        libmangal.SettingSecret,
	},
	{
		Key:         "free_secret",
		Name:        "Free device secret",
		Description: "Device secret of a registered free device (free chapters only). Only used when the app has no free device saved and there's no secret file.",
		Kind:        libmangal.SettingSecret,
	},
	{
		Key:         "secret_file",
		Name:        "Secret file",
		Description: "File with the device secret from the official app, for subscription chapters. Only used when the app has no secret saved; empty registers a free device.",
		Kind:        libmangal.SettingPath,
	},
}

// applySettings sets o's fields from the declared settings' values, if there's a hook.
// The secrets the app already gave (o.Secret, o.FreeSecret) win over the settings.
func applySettings(hook libmangal.SettingsFunc, o *mango.MangaPlusOptions, log *mango.Logger) error {
	if hook == nil {
		return nil
	}
	values, err := providerInfo.SettingValues(hook(providerInfo.ID))
	if err != nil {
		return err
	}
	o.Quality = values["quality"]
	o.API = mangoplus.API(values["api"])
	o.Viewer = mangoplus.Viewer(values["viewer"])
	// checked by SettingValues
	o.RequestInterval, _ = time.ParseDuration(values["request_interval"])
	o.OSVersion = values["os_version"]
	o.AppVersion = values["app_version"]
	if id := values["android_id"]; id != "" {
		o.AndroidID = id
	}
	if o.FreeSecret == "" {
		o.FreeSecret = values["free_secret"]
	}
	if path := values["secret_file"]; path != "" && o.Secret == "" {
		data, err := os.ReadFile(path)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			log.Log("secret file %q doesn't exist, using the free tier", path)
		case err != nil:
			log.Log("couldn't read secret file %q, using the free tier: %s", path, err)
		default:
			o.Secret = strings.TrimSpace(string(data))
		}
	}
	return nil
}
