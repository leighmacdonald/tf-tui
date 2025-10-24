// Package cache implements a very trivial filesystem cache.
package cache

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal/config"
)

const (
	// How long until a entry is considered stale.
	maxCacheAge = time.Hour * 24 * 7 // Please dont hurt me
)

var (
	ErrCacheMiss = errors.New("cache miss error")
	errCacheSet  = errors.New("cache set error")
	errCacheDir  = errors.New("cache dir error")
)

type Cache interface {
	//
	Get(steamID steamid.SteamID, variant ItemVariant) ([]byte, error)
	Set(steamID steamid.SteamID, variant ItemVariant, content []byte) error
}

type ItemVariant int

const (
	CacheMetaProfile ItemVariant = iota
)

// Filesystem implements the default filesystem based Cache interface.
type Filesystem struct {
	cacheDir string
}

func New() (Filesystem, error) {
	cachePath := config.PathCache(config.CacheDirName)
	if err := os.MkdirAll(cachePath, 0o700); err != nil {
		slog.Error("Failed to make config root", slog.String("error", err.Error()),
			slog.String("path", cachePath))

		return Filesystem{}, errors.Join(err, errCacheDir)
	}

	return Filesystem{cacheDir: cachePath}, nil
}

func (c Filesystem) Set(steamID steamid.SteamID, variant ItemVariant, content []byte) error {
	file, errFile := os.Create(path.Join(c.cacheDir, cacheName(steamID, variant)))
	if errFile != nil {
		return errors.Join(errFile, errCacheSet)
	}

	defer func(file io.Closer) {
		if err := file.Close(); err != nil {
			slog.Error("Failed to close cache file", slog.String("error", err.Error()))
		}
	}(file)

	if _, err := file.Write(content); err != nil {
		return errors.Join(err, errCacheSet)
	}

	return nil
}

func (c Filesystem) Get(steamID steamid.SteamID, variant ItemVariant) ([]byte, error) {
	file, errFile := os.Open(path.Join(c.cacheDir, cacheName(steamID, variant)))
	if errFile != nil {
		return nil, errors.Join(errFile, ErrCacheMiss)
	}

	stat, errStat := file.Stat()
	if errStat != nil {
		if err := file.Close(); err != nil {
			return nil, errors.Join(errStat, err, ErrCacheMiss)
		}

		return nil, errors.Join(errStat, ErrCacheMiss)
	}

	if time.Since(stat.ModTime()) > maxCacheAge {
		if err := file.Close(); err != nil {
			return nil, errors.Join(err, ErrCacheMiss)
		}

		if err := os.Remove(cacheName(steamID, variant)); err != nil {
			return nil, errors.Join(err, ErrCacheMiss)
		}

		return nil, ErrCacheMiss
	}

	body, errRead := io.ReadAll(file)
	if errRead != nil {
		if err := file.Close(); err != nil {
			return nil, errors.Join(err, ErrCacheMiss)
		}

		return nil, errors.Join(errRead, ErrCacheMiss)
	}

	if err := file.Close(); err != nil {
		return nil, errors.Join(err, ErrCacheMiss)
	}

	return body, nil
}

func cacheName(steamID steamid.SteamID, variant ItemVariant) string {
	return steamID.String() + "_" + strconv.Itoa(int(variant))
}
