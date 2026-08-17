package workspace

// Resolver turns the ambient environment into a validated Workspace.
type Resolver interface {
	Resolve() (Workspace, error)
}
