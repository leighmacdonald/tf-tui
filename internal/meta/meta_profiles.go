package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/tf-tui/internal"
	"github.com/leighmacdonald/tf-tui/internal/encoding"
	"github.com/leighmacdonald/tf-tui/internal/tfapi"
)

var ErrFetchMetaProfile = errors.New("failed to fetch meta profile")

func New(client *tfapi.ClientWithResponses, cache internal.Cache) *MetaFetcher {
	return &MetaFetcher{
		client: client,
		cache:  cache,
	}
}

type MetaFetcher struct {
	client *tfapi.ClientWithResponses
	cache  internal.Cache
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
		body, errGet := m.cache.Get(steamID, internal.CacheMetaProfile)
		if errGet != nil {
			if !errors.Is(errGet, internal.ErrCacheMiss) {
				return nil, nil, errors.Join(errGet, ErrFetchMetaProfile)
			}

			missing = append(missing, steamID)

			continue
		}

		cached, err := encoding.UnmarshalJSON[tfapi.MetaProfile](bytes.NewReader(body))
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
		return nil, errors.Join(errResp, ErrFetchMetaProfile)
	}
	defer func(closer io.Closer) {
		if err := closer.Close(); err != nil {
			slog.Error("Failed to close response body", slog.String("error", err.Error()))
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		slog.Error("Failed to fetch profiles")

		return nil, ErrFetchMetaProfile
	}
	parsed, errParse := tfapi.ParseMetaProfileResponse(resp)
	if errParse != nil {
		return nil, errors.Join(errParse, ErrFetchMetaProfile)
	}

	for _, profile := range *parsed.JSON200 {
		var buf bytes.Buffer
		if errBody := json.NewEncoder(&buf).Encode(profile); errBody != nil {
			return nil, errors.Join(errBody, ErrFetchMetaProfile)
		}
		if errSet := m.cache.Set(steamid.New(profile.SteamId), internal.CacheMetaProfile, buf.Bytes()); errSet != nil {
			return nil, errors.Join(errSet, ErrFetchMetaProfile)
		}

		profiles = append(profiles, profile)
	}

	return profiles, nil
}
