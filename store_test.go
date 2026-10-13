package mangoprovider

import (
	"testing"

	"github.com/luevano/libmangal/mangadata"
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
