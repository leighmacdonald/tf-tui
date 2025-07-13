package main

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

type MetaCache struct {
	profiles map[steamid.SteamID]MetaProfile
	mu       *sync.RWMutex
	queue    steamid.Collection
}

func NewMetaCache() *MetaCache {
	return &MetaCache{
		profiles: make(map[steamid.SteamID]MetaProfile),
		mu:       &sync.RWMutex{},
	}
}

func (m *MetaCache) Set(profiles []MetaProfile) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, profile := range profiles {
		sid := steamid.New(profile.SteamId)
		if !sid.Valid() {
			continue
		}

		m.profiles[sid] = profile
	}
}

func (m *MetaCache) Get(steamIDs steamid.Collection) ([]MetaProfile, error) {
	var profiles []MetaProfile
	for _, sid := range steamIDs {
		if data, found := m.profiles[sid]; found {
			profiles = append(profiles, data)
		}
	}

	return profiles, nil
}

func decodeJSON[T any](reader io.ReadCloser) (T, error) {
	var output T
	if err := json.NewDecoder(reader).Decode(&output); err != nil {
		return output, err
	}

	return output, nil
}
