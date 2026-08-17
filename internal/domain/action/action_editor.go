package action

import (
	"path/filepath"

	"github.com/mateuszmierzwinski/basalt/internal/domain/toolset"
	"github.com/mateuszmierzwinski/basalt/internal/port"
)

// nanoAction opens one Markdown file in the target's directory. One instance
// exists per file found, which is why it is constructed by the registry rather
// than living in the static map.
type nanoAction struct {
	file string
}

func (a *nanoAction) ID() string    { return "nano." + a.file }
func (a *nanoAction) Label() string { return "Nano > " + a.file }
func (a *nanoAction) Kind() Kind    { return KindEditor }

// Applies is true whenever the action exists at all: the registry only builds
// one for a file it has already seen on disk.
func (a *nanoAction) Applies(ctx Context) bool {
	return ctx.Tools.Has(toolset.ToolNano)
}

// Command opens the file by its base name with the working directory set to
// the package, so nothing constructs a path from the file name itself.
func (a *nanoAction) Command(ctx Context) (port.CommandSpec, error) {
	return toolCommand(ctx, toolset.ToolNano, ctx.Directory, filepath.Base(a.file))
}

// newNanoAction builds the Nano action for one Markdown file.
func newNanoAction(file string) Action {
	return &nanoAction{file: file}
}
