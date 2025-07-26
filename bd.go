package main

import (
	"encoding/json"
	"net/http"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

type UserListManager struct {
	userLists []UserList
	lists     []BDSchema
}

func NewUserListManager(lists []UserList) *UserListManager {
	return &UserListManager{userLists: lists}
}

func (m *UserListManager) Sync() {
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	var lists []BDSchema
	for _, userList := range m.userLists {
		wg.Add(1)

		go func(list UserList) {
			defer wg.Done()

			req, errReq := http.NewRequest("GET", list.URL, nil)
			if errReq != nil {
				tea.Printf("Failed to create request: %v\n", errReq)
				return
			}

			resp, errResp := http.DefaultClient.Do(req)
			if errResp != nil {
				tea.Printf("Failed to get response: %v\n", errResp)
				return
			}
			if resp.StatusCode != 200 {
				tea.Printf("Failed to get response: %v\n", errResp)
				return
			}

			var bdList BDSchema
			if err := json.NewDecoder(resp.Body).Decode(&bdList); err != nil {
				tea.Printf("Failed to decode response: %v\n", errResp)
				return
			}
			mu.Lock()
			lists = append(lists, bdList)
			mu.Unlock()

		}(userList)
	}

	wg.Wait()

	m.lists = lists
}
