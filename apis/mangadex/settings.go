package mangadex

import (
	"strconv"

	"github.com/luevano/libmangal"
	mango "github.com/luevano/mangoprovider"
)

var settings = []libmangal.ProviderSetting{
	{
		Key:         "data_saver",
		Name:        "Data saver",
		Description: `Use MangaDex's "data-saver" (smaller, lower quality) page images.`,
		Kind:        libmangal.SettingBool,
		Default:     "false",
	},
	{
		Key:         "exclude_groups",
		Name:        "Excluded groups",
		Description: `Scanlation groups whose chapters are left out, comma separated: names (any case) or group ids (a name with a comma needs the id).`,
		Kind:        libmangal.SettingList,
	},
	{
		Key:         "prefer_groups",
		Name:        "Preferred groups",
		Description: `Scanlation groups to take a chapter from when there are several of the same number (avoid duplicate chapters), first listed first, comma separated names (any case) or group ids. Without one of them, the group with the most chapters in the volume.`,
		Kind:        libmangal.SettingList,
	},
}

// applySettings sets o's fields from the declared settings' values, if there's a hook.
func applySettings(hook libmangal.SettingsFunc, o *mango.MangaDexOptions) error {
	if hook == nil {
		return nil
	}
	values, err := providerInfo.SettingValues(hook(providerInfo.ID))
	if err != nil {
		return err
	}
	// checked by SettingValues
	o.DataSaver, _ = strconv.ParseBool(values["data_saver"])
	o.ExcludeGroups = libmangal.SplitList(values["exclude_groups"])
	o.PreferGroups = libmangal.SplitList(values["prefer_groups"])
	return nil
}
