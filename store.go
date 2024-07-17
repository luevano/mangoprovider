package mangoprovider

import (
	"github.com/luevano/libmangal/mangadata"
	"github.com/philippgille/gokv"
)

const (
	CacheBucketNameMangas   = "mangas"
	CacheBucketNameVolumes  = "volumes"
	CacheBucketNameChapters = "chapters"
)

// Store is a gokv.Store wrapper with special handling
// of search volumes/chapters as the manga needs to be
// re-setted when loading from the store, as the pointer
// of the manga will be different for all chapters and
// thus the SetMetadata method will not work.
type Store struct {
	openStore func(bucketName string) (gokv.Store, error)
	store     gokv.Store
}

func (s *Store) open(bucketName string) error {
	store, err := s.openStore(bucketName)
	s.store = store
	return err
}

func (s *Store) Close() error {
	if s.store == nil {
		return nil
	}
	return s.store.Close()
}

func (s *Store) SetMangas(cacheID string, mangas []mangadata.Manga) error {
	err := s.open(CacheBucketNameMangas)
	if err != nil {
		return err
	}
	defer s.Close()

	return s.store.Set(cacheID, mangas)
}

func (s *Store) GetMangas(cacheID string, query string, mangas *[]mangadata.Manga) (bool, error) {
	err := s.open(CacheBucketNameMangas)
	if err != nil {
		return false, err
	}
	defer s.Close()

	found, err := s.store.Get(cacheID, mangas)
	if err != nil {
		return false, err
	}
	if found {
		Log("cache: found mangas for query %q", query)
		return true, nil
	}
	return false, nil
}

func (s *Store) SetVolumes(cacheID string, volumes []mangadata.Volume) error {
	err := s.open(CacheBucketNameVolumes)
	if err != nil {
		return err
	}
	defer s.Close()

	return s.store.Set(cacheID, volumes)
}

func (s *Store) GetVolumes(cacheID string, manga Manga, volumes *[]mangadata.Volume) (bool, error) {
	err := s.open(CacheBucketNameVolumes)
	if err != nil {
		return false, err
	}
	defer s.Close()

	var foundVolumes []mangadata.Volume
	found, err := s.store.Get(cacheID, &foundVolumes)
	if err != nil {
		return false, err
	}
	if found {
		Log("cache: found volumes for manga %q", manga.String())
		// Need to re-set the manga, as the read data will point to a different address
		var updatedVolumes []mangadata.Volume
		for _, v := range foundVolumes {
			v := v.(*Volume)
			v.Manga_ = &manga
			updatedVolumes = append(updatedVolumes, v)
		}
		*volumes = updatedVolumes
		return true, nil
	}
	return false, nil
}

func (s *Store) SetChapters(cacheID string, chapters []mangadata.Chapter) error {
	err := s.open(CacheBucketNameChapters)
	if err != nil {
		return err
	}
	defer s.Close()

	return s.store.Set(cacheID, chapters)
}

func (s *Store) GetChapters(cacheID string, volume Volume, chapters *[]mangadata.Chapter) (bool, error) {
	err := s.open(CacheBucketNameChapters)
	if err != nil {
		return false, err
	}
	defer s.Close()

	var foundChapters []mangadata.Chapter
	found, err := s.store.Get(cacheID, &foundChapters)
	if err != nil {
		return false, err
	}
	if found {
		Log("cache: found chapters for volume %s", volume.String())
		// Need to re-set the volume, as the read data will point to a different address
		var updatedChapters []mangadata.Chapter
		for _, c := range foundChapters {
			c := c.(*Chapter)
			c.Volume_ = &volume
			updatedChapters = append(updatedChapters, c)
		}
		*chapters = updatedChapters
		return true, nil
	}
	return false, nil
}
