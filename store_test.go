package mangoprovider

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/luevano/libmangal/mangadata"
	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/syncmap"
)

// a zero Store (e.g. in tests) caches nothing instead of panicking
func TestZeroStore(t *testing.T) {
	var s Store
	var volumes []mangadata.Volume
	var chapters []mangadata.Chapter
	if err := s.SetVolumes("id", nil); err != nil {
		t.Error(err)
	}
	if found, err := s.GetVolumes("id", Manga{}, &volumes); found || err != nil {
		t.Errorf("volumes: %v, %v", found, err)
	}
	if err := s.SetChapters("id", nil); err != nil {
		t.Error(err)
	}
	if found, err := s.GetChapters("id", Volume{}, &chapters); found || err != nil {
		t.Errorf("chapters: %v, %v", found, err)
	}
}

// the buckets can't collide with another owner's, e.g. a lua provider with the id "mangas"
func TestBucketNames(t *testing.T) {
	var opened []string
	s, err := openStore(func(bucketName string) (gokv.Store, error) {
		opened = append(opened, bucketName)
		return syncmap.NewStore(syncmap.DefaultOptions), nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	want := []string{"mango-mangas", "mango-volumes", "mango-chapters", "mango-mirrors"}
	if !slices.Equal(opened, want) {
		t.Errorf("opened %v, want %v", opened, want)
	}
}

// cache hits go to Options.Debug, prefixed with the provider id
func TestStoreDebug(t *testing.T) {
	var lines []string
	options := DefaultOptions()
	options.Debug = func(format string, a ...any) { lines = append(lines, fmt.Sprintf(format, a...)) }
	loader := &Loader{
		ID: "mango-test", Name: "Test", Version: "0.1.0",
		Options: options,
		F:       func(context.Context, *Logger) (Functions, error) { return Functions{}, nil },
	}
	p, err := loader.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	store := p.(*Provider).store
	var mangas []mangadata.Manga
	if found, _ := store.GetMangas("q", "q", &mangas); found || len(lines) != 0 {
		t.Fatalf("miss: %t, %v", found, lines)
	}
	if err := store.SetMangas("q", []mangadata.Manga{&Manga{ID: "a"}}); err != nil {
		t.Fatal(err)
	}
	if found, _ := store.GetMangas("q", "q", &mangas); !found || !slices.Equal(lines, []string{"mango-test: mangas q cached"}) {
		t.Errorf("hit: %t, %v", found, lines)
	}
}
