package model

// ViewState tracks the common ui states that are shared between many models.
type ViewState struct {
	// Page is the active highest level page model. They represent a complete standalone "page" or "screen" that occupies the entire
	// page with the exception of the footer
	Page Page
	// Section defines which "section" or "tab" within the page is active.
	Section Section
	// KeyZone defines which area, usually defined with a active model.Container, is active and accepting user keyboard inputs.
	KeyZone KeyZone

	// --------- h
	// | Upper | e
	// |-------- i
	// | Lower | g
	// --------- h
	// W i d t h t
	Upper  int
	Lower  int
	Height int
	Width  int
}
