package mangadex

import (
	"slices"
	"testing"

	mango "github.com/luevano/mangoprovider"
)

func TestApplySettings(t *testing.T) {
	if err := providerInfo.Validate(); err != nil {
		t.Fatal(err)
	}
	o := mango.DefaultMangaDexOptions()
	o.DataSaver = true
	if err := applySettings(nil, &o); err != nil || !o.DataSaver {
		t.Fatalf("no hook changed the options: %v, %+v", err, o)
	}
	hook := func(id string) map[string]string {
		if id != providerInfo.ID {
			t.Errorf("asked for %q", id)
		}
		return map[string]string{"exclude_groups": "A, b"}
	}
	if err := applySettings(hook, &o); err != nil {
		t.Fatal(err)
	}
	if o.DataSaver || !slices.Equal(o.ExcludeGroups, []string{"A", "b"}) || len(o.PreferGroups) != 0 {
		t.Errorf("options %+v", o)
	}
	bad := func(string) map[string]string { return map[string]string{"data_saver": "maybe"} }
	if err := applySettings(bad, &o); err == nil {
		t.Error("invalid value: no error")
	}
}
