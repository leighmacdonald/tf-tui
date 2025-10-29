package model

type Section int

const (
	SectionServers Section = iota
	SectionPlayers
	SectionBans
	SectionBD
	SectionComp
	SectionChat
	SectionConsole
)
