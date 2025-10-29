package input

// Direction defines the cardinal directions the users can use in the UI.
type Direction int

const (
	Up Direction = iota //nolint:varnamelen
	Down
	Left
	Right
)
