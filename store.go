package mangoprovider

import (
	"encoding/gob"
	"errors"
	"fmt"

	"github.com/luevano/libmangal/mangadata"
	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/encoding"
	"github.com/philippgille/gokv/syncmap"
)

// The cache buckets, prefixed so they can't be mistaken for another owner's (e.g. a Lua
// provider with the id "mangas").
const (
	CacheBucketNameMangas   = BundleID + "-mangas"
	CacheBucketNameVolumes  = BundleID + "-volumes"
	CacheBucketNameChapters = BundleID + "-chapters"
	// CacheBucketNameMirrors: the mirror that last worked by provider id, should expire.
	CacheBucketNameMirrors = BundleID + "-mirrors"
)

// LegacyCacheBucketNames: The buckets' names up to v0.21.0 ("mangas", "volumes", "chapters",
// "mirrors"), e.g. for migrating old caches. Nothing reads them anymore, the cache is disposable.
var LegacyCacheBucketNames = []string{"mangas", "volumes", "chapters", "mirrors"}

// the cached slices hold interfaces, gob needs the concrete types (same names as mangal's)
func init() {
	gob.RegisterName("mango-manga", &Manga{})
	gob.RegisterName("mango-volume", &Volume{})
	gob.RegisterName("mango-chapter", &Chapter{})
	gob.RegisterName("mango-page", &Page{})
	gob.RegisterName("provider-metadata", &mangadata.Metadata{})
}

// Store is a gokv.Store wrapper with special handling
// of search volumes/chapters as the manga needs to be
// re-setted when loading from the store, as the pointer
// of the manga will be different for all chapters and
// thus the SetMetadata method will not work.
//
// The buckets are opened once when the provider loads and closed with it.
type Store struct {
	mangas, volumes, chapters, mirrors gokv.Store
	log                                *Logger
}

func openStore(open func(bucketName string) (gokv.Store, error), log *Logger) (Store, error) {
	s := Store{log: log}
	for _, b := range []struct {
		name  string
		store *gokv.Store
	}{
		{CacheBucketNameMangas, &s.mangas},
		{CacheBucketNameVolumes, &s.volumes},
		{CacheBucketNameChapters, &s.chapters},
		{CacheBucketNameMirrors, &s.mirrors},
	} {
		st, err := open(b.name)
		if err != nil {
			return Store{}, errors.Join(fmt.Errorf("open cache %s: %w", b.name, err), s.Close())
		}
		*b.store = st
	}
	return s, nil
}

func (s Store) Close() error {
	var errs []error
	for _, st := range []gokv.Store{s.mangas, s.volumes, s.chapters, s.mirrors} {
		if st != nil {
			errs = append(errs, st.Close())
		}
	}
	return errors.Join(errs...)
}

// GetMirror: the mirror that last worked for the provider, if it hasn't expired.
func (s Store) GetMirror(providerID string) (string, bool, error) {
	var mirror string
	if s.mirrors == nil {
		return "", false, nil
	}
	found, err := s.mirrors.Get(providerID, &mirror)
	return mirror, found, err
}

func (s Store) SetMirror(providerID, mirror string) error {
	if s.mirrors == nil {
		return nil
	}
	return s.mirrors.Set(providerID, mirror)
}

func (s Store) DeleteMirror(providerID string) error {
	if s.mirrors == nil {
		return nil
	}
	return s.mirrors.Delete(providerID)
}

func (s Store) SetMangas(cacheID string, mangas []mangadata.Manga) error {
	if s.mangas == nil {
		return nil
	}
	return s.mangas.Set(cacheID, mangas)
}

func (s Store) GetMangas(cacheID string, query string, mangas *[]mangadata.Manga) (bool, error) {
	if s.mangas == nil {
		return false, nil
	}
	found, err := s.mangas.Get(cacheID, mangas)
	if err != nil {
		return false, err
	}
	if found {
		s.log.Log("cache: found mangas for query %q", query)
		return true, nil
	}
	return false, nil
}

func (s Store) SetVolumes(cacheID string, volumes []mangadata.Volume) error {
	if s.volumes == nil {
		return nil
	}
	return s.volumes.Set(cacheID, volumes)
}

func (s Store) GetVolumes(cacheID string, manga Manga, volumes *[]mangadata.Volume) (bool, error) {
	if s.volumes == nil {
		return false, nil
	}
	var foundVolumes []mangadata.Volume
	found, err := s.volumes.Get(cacheID, &foundVolumes)
	if err != nil {
		return false, err
	}
	if found {
		s.log.Log("cache: found volumes for manga %q", manga.String())
		// Need to re-set the manga, as the read data will point to a different address
		var updatedVolumes []mangadata.Volume
		for _, v := range foundVolumes {
			v, ok := v.(*Volume)
			if !ok {
				// e.g. written by an older version, fetch it again
				return false, nil
			}
			v.Manga_ = &manga
			updatedVolumes = append(updatedVolumes, v)
		}
		*volumes = updatedVolumes
		return true, nil
	}
	return false, nil
}

func (s Store) SetChapters(cacheID string, chapters []mangadata.Chapter) error {
	if s.chapters == nil {
		return nil
	}
	return s.chapters.Set(cacheID, chapters)
}

func (s Store) GetChapters(cacheID string, volume Volume, chapters *[]mangadata.Chapter) (bool, error) {
	if s.chapters == nil {
		return false, nil
	}
	var foundChapters []mangadata.Chapter
	found, err := s.chapters.Get(cacheID, &foundChapters)
	if err != nil {
		return false, err
	}
	if found {
		s.log.Log("cache: found chapters for volume %s", volume.String())
		// Need to re-set the volume, as the read data will point to a different address
		var updatedChapters []mangadata.Chapter
		for _, c := range foundChapters {
			c, ok := c.(*Chapter)
			if !ok {
				// e.g. written by an older version, fetch it again
				return false, nil
			}
			c.Volume_ = &volume
			updatedChapters = append(updatedChapters, c)
		}
		*chapters = updatedChapters
		return true, nil
	}
	return false, nil
}

// NewMemoryStore: A Store in memory, like DefaultOptions' CacheStore (e.g. for tests).
func NewMemoryStore() Store {
	s, _ := openStore(func(string) (gokv.Store, error) {
		return syncmap.NewStore(syncmap.Options{Codec: encoding.Gob}), nil
	}, nil)
	return s
}
