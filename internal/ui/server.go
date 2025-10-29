package ui

import "github.com/leighmacdonald/tf-tui/internal/tf"

type Server struct {
	Hostname  string
	Players   Players
	Game      string
	Region    string
	Map       string
	Ping      float64
	Tags      []string
	LogSecret int
}

type Snapshot struct {
	HostPort string
	Server   Server
	Status   tf.Status
	// TODO only send these once
	PluginsSM   []tf.GamePlugin
	PluginsMeta []tf.GamePlugin
	CVars       tf.CVarList
}

func (s Snapshot) AvgPing() float64 {
	if len(s.Status.Players) == 0 {
		return 0
	}

	var pings float64
	for _, player := range s.Status.Players {
		pings += float64(player.Ping)
	}

	return pings / float64(len(s.Status.Players))

}
