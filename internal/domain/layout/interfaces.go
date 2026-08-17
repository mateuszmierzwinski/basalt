package layout

// Renderer turns view state into a drawable frame.
type Renderer interface {
	Render(view View, width, height int) Frame
}
