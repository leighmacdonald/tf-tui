package main

import (
	"context"
	"errors"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	errFetchMetaProfile = errors.New("failed to fetch meta profile")
)

func NewAPIs(client *ClientWithResponses) APIs {
	return APIs{client: client}
}

type APIs struct {
	client *ClientWithResponses
}

func (a APIs) getMetaProfiles(ctx context.Context, steamIDs steamid.Collection) ([]MetaProfile, error) {
	if len(steamIDs) == 0 {
		return nil, nil
	}

	resp, errResp := a.client.MetaProfile(ctx, &MetaProfileParams{Steamids: steamIDs.ToStringSlice()})
	if errResp != nil {
		return nil, errors.Join(errResp, errFetchMetaProfile)
	}

	parsed, errParse := ParseMetaProfileResponse(resp)
	if errParse != nil {
		return nil, errors.Join(errResp, errParse)
	}

	return *parsed.JSON200, nil
}
