package ui

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
