package main

import (
	"errors"
	"io"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/adrg/xdg"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	maxCacheAge  = time.Hour * 24
	cacheDirName = "cache"
)

var (
	errCacheMiss = errors.New("cache miss error")
	errCacheSet  = errors.New("cache set error")
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

func cachePath() string {
	cacheDir, found := os.LookupEnv("CACHE_DIR")
	if found && cacheDir != "" {
		return cacheDir
	}

	return path.Join(xdg.CacheHome, configDirName, cacheDirName)
}

func NewFilesystemCache() (*FilesystemCache, error) {
	if err := os.MkdirAll(cachePath(), 0o700); err != nil {
		tea.Println("Failed to make config root: " + err.Error())

		return nil, err
	}

	return &FilesystemCache{cacheDir: cachePath()}, nil
}

func (c *FilesystemCache) Set(steamID steamid.SteamID, variant CacheItemVariant, content []byte) error {
	file, errFile := os.Create(path.Join(c.cacheDir, cacheName(steamID, variant)))
	if errFile != nil {
		return errors.Join(errCacheSet, errFile)
	}

	defer file.Close()

	if _, err := file.Write(content); err != nil {
		return errors.Join(errCacheSet, err)
	}

	return nil
}

func (c *FilesystemCache) Get(steamID steamid.SteamID, variant CacheItemVariant) ([]byte, error) {
	file, errFile := os.Open(path.Join(c.cacheDir, cacheName(steamID, variant)))
	if errFile != nil {
		return nil, errors.Join(errCacheMiss, errFile)
	}

	stat, errStat := file.Stat()
	if errStat != nil {
		file.Close()
		return nil, errors.Join(errCacheMiss, errStat)
	}

	if time.Since(stat.ModTime()) > maxCacheAge {
		file.Close()
		if err := os.Remove(cacheName(steamID, variant)); err != nil {
			return nil, errors.Join(errCacheMiss, err)
		}
		return nil, errCacheMiss
	}

	body, errRead := io.ReadAll(file)
	if errRead != nil {
		file.Close()
		return nil, errors.Join(errCacheMiss, errRead)
	}

	file.Close()

	return body, nil
}

func cacheName(steamID steamid.SteamID, variant CacheItemVariant) string {
	return steamID.String() + "_" + strconv.Itoa(int(variant))
}
