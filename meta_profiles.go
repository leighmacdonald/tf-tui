package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/tfapi"
)

func NewMetaFetcher(client *tfapi.ClientWithResponses, cache Cache) *MetaFetcher {
	return &MetaFetcher{
		client: client,
		cache:  cache,
	}
}

type MetaFetcher struct {
	client *tfapi.ClientWithResponses
	cache  Cache
}

// MetaProfiles handles loading player MetaProfiles. It first attempts to load from a local filesystem cache
// and if any are missing or expired, they will be fetched from the api, and subsequently cached.
func (m *MetaFetcher) MetaProfiles(ctx context.Context, steamIDs steamid.Collection) ([]tfapi.MetaProfile, error) {
	if len(steamIDs) == 0 {
		return nil, nil
	}

	profiles, missing, errCached := m.cachedMetaProfiles(steamIDs)
	if errCached != nil {
		return nil, errCached
	}

	if len(missing) == 0 {
		return profiles, nil
	}

	updates, errUpdates := m.fetchMetaProfiles(ctx, missing)
	if errUpdates != nil {
		return profiles, errUpdates
	}

	return append(profiles, updates...), nil
}

func (m *MetaFetcher) cachedMetaProfiles(steamIDs steamid.Collection) ([]tfapi.MetaProfile, steamid.Collection, error) {
	var profiles []tfapi.MetaProfile //nolint:prealloc
	var missing steamid.Collection
	for _, steamID := range steamIDs {
		body, errGet := m.cache.Get(steamID, CacheMetaProfile)
		if errGet != nil {
			if !errors.Is(errGet, errCacheMiss) {
				return nil, nil, errors.Join(errGet, errFetchMetaProfile)
			}

			missing = append(missing, steamID)

			continue
		}

		cached, err := unmarshalJSON[tfapi.MetaProfile](bytes.NewReader(body))
		if err != nil {
			missing = append(missing, steamID)

			continue
		}

		profiles = append(profiles, cached)
	}

	return profiles, missing, nil
}

func (m *MetaFetcher) fetchMetaProfiles(ctx context.Context, steamIDs steamid.Collection) ([]tfapi.MetaProfile, error) {
	var profiles []tfapi.MetaProfile //nolint:prealloc
	resp, errResp := m.client.MetaProfile(ctx, &tfapi.MetaProfileParams{Steamids: strings.Join(steamIDs.ToStringSlice(), ",")})
	if errResp != nil {
		return nil, errors.Join(errResp, errFetchMetaProfile)
	}
	defer func(closer io.Closer) {
		if err := closer.Close(); err != nil {
			slog.Error("Failed to close response body", slog.String("error", err.Error()))
		}
	}(resp.Body)

	parsed, errParse := tfapi.ParseMetaProfileResponse(resp)
	if errParse != nil {
		return nil, errors.Join(errParse, errFetchMetaProfile)
	}

	for _, profile := range *parsed.JSON200 {
		var buf bytes.Buffer
		if errBody := json.NewEncoder(&buf).Encode(profile); errBody != nil {
			return nil, errors.Join(errBody, errFetchMetaProfile)
		}
		if errSet := m.cache.Set(steamid.New(profile.SteamId), CacheMetaProfile, buf.Bytes()); errSet != nil {
			return nil, errors.Join(errSet, errFetchMetaProfile)
		}

		profiles = append(profiles, profile)
	}

	return profiles, nil
}
