package internal

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
	maxCacheAge = time.Hour * 24
)

var (
	errCacheMiss = errors.New("cache miss error")
	errCacheSet  = errors.New("cache set error")
	errCacheDir  = errors.New("cache dir error")
)

type Cache interface {
	Get(steamID steamid.SteamID, variant CacheItemVariant) ([]byte, error)
	Set(steamID steamid.SteamID, variant CacheItemVariant, content []byte) error
}

type CacheItemVariant int

const (
	CacheMetaProfile CacheItemVariant = iota
)

type FilesystemCache struct {
	cacheDir string
}

func NewFilesystemCache() (FilesystemCache, error) {
	cachePath := config.PathCache(config.CacheDirName)
	if err := os.MkdirAll(cachePath, 0o700); err != nil {
		slog.Error("Failed to make config root", slog.String("error", err.Error()),
			slog.String("path", cachePath))

		return FilesystemCache{}, errors.Join(err, errCacheDir)
	}

	return FilesystemCache{cacheDir: cachePath}, nil
}

func (c FilesystemCache) Set(steamID steamid.SteamID, variant CacheItemVariant, content []byte) error {
	file, errFile := os.Create(path.Join(c.cacheDir, cacheName(steamID, variant)))
	if errFile != nil {
		return errors.Join(errCacheSet, errFile)
	}

	defer func(file io.Closer) {
		if err := file.Close(); err != nil {
			slog.Error("Failed to close cache file", slog.String("error", err.Error()))
		}
	}(file)

	if _, err := file.Write(content); err != nil {
		return errors.Join(errCacheSet, err)
	}

	return nil
}

func (c FilesystemCache) Get(steamID steamid.SteamID, variant CacheItemVariant) ([]byte, error) {
	file, errFile := os.Open(path.Join(c.cacheDir, cacheName(steamID, variant)))
	if errFile != nil {
		return nil, errors.Join(errCacheMiss, errFile)
	}

	stat, errStat := file.Stat()
	if errStat != nil {
		if err := file.Close(); err != nil {
			return nil, errors.Join(errCacheMiss, err, errStat)
		}

		return nil, errors.Join(errCacheMiss, errStat)
	}

	if time.Since(stat.ModTime()) > maxCacheAge {
		if err := file.Close(); err != nil {
			return nil, errors.Join(errCacheMiss, err)
		}

		if err := os.Remove(cacheName(steamID, variant)); err != nil {
			return nil, errors.Join(errCacheMiss, err)
		}

		return nil, errCacheMiss
	}

	body, errRead := io.ReadAll(file)
	if errRead != nil {
		if err := file.Close(); err != nil {
			return nil, errors.Join(errCacheMiss, err)
		}

		return nil, errors.Join(errCacheMiss, errRead)
	}

	if err := file.Close(); err != nil {
		return nil, errors.Join(errCacheMiss, err)
	}

	return body, nil
}

func cacheName(steamID steamid.SteamID, variant CacheItemVariant) string {
	return steamID.String() + "_" + strconv.Itoa(int(variant))
}
