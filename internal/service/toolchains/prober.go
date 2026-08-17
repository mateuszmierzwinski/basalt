// Package toolchains probes the host's compilers once and resolves each
// target's declared requirement from the go.mod that governs it.
package toolchains

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
	"github.com/mateuszmierzwinski/basalt/internal/domain/toolchain"
	"github.com/mateuszmierzwinski/basalt/internal/port"
)

// probeTimeout bounds each `--version` call. A compiler that does not answer
// promptly is treated as absent rather than allowed to delay startup.
const probeTimeout = 5 * time.Second

const goModFile = "go.mod"

// Config is the prober's construction parameters.
type Config struct {
	Environment port.Environment
	Runner      port.CommandRunner
	Store       port.FileStore
	Root        string
}

type prober struct {
	store    port.FileStore
	root     string
	versions toolchain.Versions

	// mu guards required, which memoises the go.mod lookup per package — the
	// header is recomputed on every cursor move and the answer never changes
	// within a session.
	mu       sync.RWMutex
	required map[string]string
}

// New probes the host's compilers and returns the prober.
//
// Probing happens once, at startup: the versions cannot change under a running
// basalt in any way worth chasing, and shelling out on every keystroke to
// re-answer the same question would be absurd.
func New(ctx context.Context, config Config) Prober {
	return &prober{
		store:    config.Store,
		root:     config.Root,
		versions: probeVersions(ctx, config),
		required: map[string]string{},
	}
}

func probeVersions(ctx context.Context, config Config) toolchain.Versions {
	return toolchain.Versions{
		GoAvailable: toolchain.ParseGoVersion(runVersion(ctx, config, "go", "version")),
		GCC:         toolchain.ParseCompilerVersion(runVersion(ctx, config, "gcc", "--version")),
		GXX:         toolchain.ParseCompilerVersion(runVersion(ctx, config, "g++", "--version")),
	}
}

// runVersion asks one compiler for its version, returning empty when it is not
// installed or does not answer.
func runVersion(ctx context.Context, config Config, program string, args ...string) string {
	path, err := config.Environment.LookPath(program)
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	result, err := config.Runner.Run(ctx, port.CommandSpec{Path: path, Args: args, Dir: config.Root})
	if err != nil {
		return ""
	}
	return result.Stdout
}

// SummaryFor combines the host's compilers with what this target's project asks
// for.
func (p *prober) SummaryFor(selected target.Target, languages []target.Language) toolchain.Summary {
	summary := toolchain.Summary{
		Languages:   languages,
		GoAvailable: p.versions.GoAvailable,
		GCC:         p.versions.GCC,
		GXX:         p.versions.GXX,
	}
	if containsLanguage(languages, target.LanguageGo) {
		summary.GoRequired = p.goRequirement(selected)
	}
	return summary
}

// goRequirement finds the nearest go.mod above the target and reads its `go`
// directive — the module that actually governs it, rather than a repo-wide
// guess that would be wrong the moment one project pinned a different version.
func (p *prober) goRequirement(selected target.Target) string {
	directory, err := target.Directory(p.root, selected)
	if err != nil {
		return ""
	}
	if cached, ok := p.cachedRequirement(directory); ok {
		return cached
	}
	required := toolchain.ParseGoDirective(p.nearestGoMod(directory))
	p.mu.Lock()
	p.required[directory] = required
	p.mu.Unlock()
	return required
}

func (p *prober) cachedRequirement(directory string) (string, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	cached, ok := p.required[directory]
	return cached, ok
}

// nearestGoMod walks up from a package directory to the monorepo root, stopping
// at the first go.mod it can read.
func (p *prober) nearestGoMod(directory string) string {
	for current := directory; strings.HasPrefix(current, p.root); current = filepath.Dir(current) {
		if contents, err := p.store.Read(filepath.Join(current, goModFile)); err == nil {
			return string(contents)
		}
		if current == p.root {
			return ""
		}
	}
	return ""
}

func containsLanguage(languages []target.Language, want target.Language) bool {
	for _, language := range languages {
		if language == want {
			return true
		}
	}
	return false
}
