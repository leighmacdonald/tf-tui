package main

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

func downloadUserLists(userLists []UserList) ([]BDSchema, error) { //nolint:unparam
	waitGroup := sync.WaitGroup{}
	mutex := sync.Mutex{}
	// There is no context passed down to children in tea apps.
	ctx := context.Background()
	var lists []BDSchema
	for _, userList := range userLists {
		waitGroup.Add(1)

		go func(list UserList) {
			defer waitGroup.Done()

			reqContext, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()

			req, errReq := http.NewRequestWithContext(reqContext, http.MethodGet, list.URL, nil)
			if errReq != nil {
				slog.Error("Failed to create request", slog.String("error", errReq.Error()))

				return
			}

			resp, errResp := http.DefaultClient.Do(req)
			if errResp != nil {
				slog.Error("Failed to get response", slog.String("error", errResp.Error()))

				return
			}

			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				slog.Error("Failed to get response", slog.Int("status_code", resp.StatusCode))

				return
			}

			bdList, errUnmarshal := UnmarshalJSON[BDSchema](resp.Body)
			if errUnmarshal != nil {
				slog.Error("Failed to unmarshal", slog.String("error", errUnmarshal.Error()))

				return
			}

			mutex.Lock()
			lists = append(lists, bdList)
			mutex.Unlock()
		}(userList)
	}

	waitGroup.Wait()

	return lists, nil
}
