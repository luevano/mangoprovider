package mangaplus

import (
	"context"
	"testing"

	"github.com/luevano/libmangal"
	"github.com/luevano/mangoplus"
	mango "github.com/luevano/mangoprovider"
)

// the web api needs no secret: no device, its own cache keys, the browser user agent
func TestWebAPI(t *testing.T) {
	options := mango.DefaultOptions()
	options.UserAgent = "browser"
	options.MangaPlus.API = mangoplus.APIWeb
	provider, err := Loader(options).Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer provider.Close()
	account, ok, err := provider.(libmangal.AccountProvider).Account(context.Background())
	if err != nil || !ok || account.Status != "web api (no device)" {
		t.Errorf("account: %+v, %t, %v", account, ok, err)
	}

	plusOptions := mangoplus.DefaultOptions()
	plusOptions.API, plusOptions.WebUserAgent = mangoplus.APIWeb, "browser"
	client, err := mangoplus.NewPlusClient(plusOptions)
	if err != nil {
		t.Fatal(err)
	}
	p := &plus{client: client}
	if got := p.cacheKey("mid:1"); got != "mid:1-web" {
		t.Errorf("cache key %q", got)
	}
	if got := p.client.ImageHeaders("https://img.test/a.jpg")["User-Agent"]; got != "browser" {
		t.Errorf("image user agent %q", got)
	}
	if got := (&plus{}).cacheKey("mid:1"); got != "mid:1" {
		t.Errorf("app cache key %q", got)
	}
}
