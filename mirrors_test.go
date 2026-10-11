package mangoprovider

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/luevano/libmangal"
	"github.com/philippgille/gokv"
)

func TestMirrors(t *testing.T) {
	info := libmangal.ProviderInfo{ID: "mango-test", Website: "https://a/", Mirrors: []string{"https://b/", "https://c/"}}
	open := DefaultOptions().CacheStore
	store, err := openStore(func(bucket string) (gokv.Store, error) { return open("mango-test", bucket) }, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	down := map[string]bool{}
	var tried []string
	f := func(site string) error {
		tried = append(tried, site)
		if down[site] {
			return ErrSiteDown
		}
		return nil
	}
	cached := func() string { m, _, _ := store.GetMirror("mango-test"); return m }

	// the website works: nothing cached
	m := NewMirrors(info, "", nil)
	if err := m.Do(ctx, store, f); err != nil || !slices.Equal(tried, []string{"https://a/"}) || cached() != "" {
		t.Fatalf("tried %v, cached %q, %v", tried, cached(), err)
	}

	// a and b down: c works and is cached; a new run starts from it
	down["https://a/"], down["https://b/"] = true, true
	tried = nil
	m = NewMirrors(info, "", nil)
	if err := m.Do(ctx, store, f); err != nil || !slices.Equal(tried, []string{"https://a/", "https://b/", "https://c/"}) || cached() != "https://c/" {
		t.Fatalf("tried %v, cached %q, %v", tried, cached(), err)
	}
	tried = nil
	if err := NewMirrors(info, "", nil).Do(ctx, store, f); err != nil || tried[0] != "https://c/" {
		t.Fatalf("new run tried %v, want c first", tried)
	}

	// the preferred one comes before the website; when it works the cached one is dropped
	tried = nil
	m = NewMirrors(info, "https://b/", nil)
	down["https://b/"], down["https://c/"] = false, true
	if err := m.Do(ctx, store, f); err != nil || !slices.Equal(tried, []string{"https://c/", "https://b/"}) || cached() != "" {
		t.Fatalf("tried %v, cached %q, %v", tried, cached(), err)
	}

	// other errors don't move on
	other := errors.New("not found")
	tried = nil
	err = NewMirrors(info, "", nil).Do(ctx, store, func(site string) error { tried = append(tried, site); return other })
	if !errors.Is(err, other) || len(tried) != 1 {
		t.Errorf("tried %v, %v, want only the first with its error", tried, err)
	}

	// all down
	down["https://a/"], down["https://b/"], down["https://c/"] = true, true, true
	if err := NewMirrors(info, "", nil).Do(ctx, store, f); !errors.Is(err, ErrSiteDown) {
		t.Errorf("all down: %v", err)
	}
}
