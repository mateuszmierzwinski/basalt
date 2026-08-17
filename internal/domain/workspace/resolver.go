package workspace

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mateuszmierzwinski/basalt/internal/port"
)

// reject wraps a sentinel with the path that failed validation. Every startup
// failure in this package reports the same way, so the shape is named once.
func reject(sentinel error, path string) error {
	return fmt.Errorf("%w: %s", sentinel, path)
}

type resolver struct {
	env   port.Environment
	probe port.DirectoryProbe
}

// NewResolver builds the workspace resolver, returning the interface rather
// than the concrete type (Rule 7.2).
func NewResolver(env port.Environment, probe port.DirectoryProbe) Resolver {
	return &resolver{env: env, probe: probe}
}

// Resolve reads $MONOREPO and proves it points at a usable checkout. basalt
// refuses to start otherwise: a wrong root would silently index nothing and
// run bazel in the wrong place.
func (r *resolver) Resolve() (Workspace, error) {
	root, err := r.readRoot()
	if err != nil {
		return Workspace{}, err
	}
	if err := r.verifySourceTree(root); err != nil {
		return Workspace{}, err
	}
	return Workspace{
		Root:       root,
		SourceRoot: SourceDirectory(root),
		CacheDir:   CacheDirectory(root),
	}, nil
}

func (r *resolver) readRoot() (string, error) {
	raw, ok := r.env.Lookup(monorepoEnvVar)
	if !ok || strings.TrimSpace(raw) == "" {
		return "", ErrMonorepoUnset
	}
	root := NormalizeRoot(raw)
	if !filepath.IsAbs(root) {
		return "", reject(ErrMonorepoNotAbsolute, root)
	}
	if !r.probe.IsDirectory(root) {
		return "", reject(ErrMonorepoNotDirectory, root)
	}
	return root, nil
}

func (r *resolver) verifySourceTree(root string) error {
	source := SourceDirectory(root)
	if !r.probe.IsDirectory(source) {
		return reject(ErrSourceRootMissing, source)
	}
	return nil
}
