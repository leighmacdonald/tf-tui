package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
				tea.Printf("Failed to create request: %v\n", errReq)

				return
			}

			resp, errResp := http.DefaultClient.Do(req)
			if errResp != nil {
				tea.Printf("Failed to get response: %v\n", errResp)

				return
			}

			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				tea.Printf("Failed to get response: %v\n", errResp)

				return
			}

			var bdList BDSchema
			if err := json.NewDecoder(resp.Body).Decode(&bdList); err != nil {
				tea.Printf("Failed to decode response: %v\n", errResp)

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
