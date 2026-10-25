package mangaplus

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/luevano/mangoplus"
	mango "github.com/luevano/mangoprovider"
)

func TestApplySettings(t *testing.T) {
	if err := providerInfo.Validate(); err != nil {
		t.Fatal(err)
	}
	if !providerInfo.Traits.PartialChapters || !providerInfo.Traits.AvoidUnattended {
		t.Errorf("traits %+v", providerInfo.Traits)
	}
	file := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(file, []byte("s3cret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	log := mango.NewProviderLogger(providerInfo)

	o := mango.DefaultMangaPlusOptions()
	id := o.AndroidID
	values := map[string]string{"api": "web", "request_interval": "1s", "secret_file": file}
	if err := applySettings(func(string) map[string]string { return values }, &o, log); err != nil {
		t.Fatal(err)
	}
	if o.API != mangoplus.APIWeb || o.RequestInterval != time.Second || o.Quality != "super_high" ||
		o.Viewer != mangoplus.ViewerApp || o.AndroidID != id || o.Secret != "s3cret" {
		t.Errorf("options %+v", o)
	}

	// the app's saved secrets win over the settings
	o = mango.DefaultMangaPlusOptions()
	o.Secret, o.FreeSecret = "saved", "saved free"
	values = map[string]string{"secret_file": file, "free_secret": "other", "android_id": "abc"}
	if err := applySettings(func(string) map[string]string { return values }, &o, log); err != nil {
		t.Fatal(err)
	}
	if o.Secret != "saved" || o.FreeSecret != "saved free" || o.AndroidID != "abc" {
		t.Errorf("options %+v", o)
	}

	// a missing file is the free tier, not an error
	o = mango.DefaultMangaPlusOptions()
	values = map[string]string{"secret_file": filepath.Join(t.TempDir(), "none")}
	if err := applySettings(func(string) map[string]string { return values }, &o, log); err != nil || o.Secret != "" {
		t.Errorf("missing file: %v, %q", err, o.Secret)
	}

	values = map[string]string{"quality": "ultra"}
	if err := applySettings(func(string) map[string]string { return values }, &o, log); err == nil {
		t.Error("invalid quality: no error")
	}
}
