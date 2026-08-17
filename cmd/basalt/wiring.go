package main

import (
	"os"

	"github.com/mateuszmierzwinski/basalt/internal/adapter/osenv"
	"github.com/mateuszmierzwinski/basalt/internal/adapter/osexec"
	"github.com/mateuszmierzwinski/basalt/internal/adapter/osfs"
	"github.com/mateuszmierzwinski/basalt/internal/adapter/systime"
	"github.com/mateuszmierzwinski/basalt/internal/cli"
	"github.com/mateuszmierzwinski/basalt/internal/domain/toolset"
	"github.com/mateuszmierzwinski/basalt/internal/domain/workspace"
)

// newApp is the composition root: the only place concrete adapters are named
// (Rule 7.3). Everything below here sees interfaces.
func newApp() cli.App {
	environment := osenv.New()
	store := osfs.NewStore()
	probe := osfs.NewProbe()
	clock := systime.New()
	commands := osexec.NewRunner()
	return cli.NewApp(cli.Dependencies{
		Resolver: workspace.NewResolver(environment, probe),
		Detector: toolset.NewDetector(environment),
		Indexers: newIndexerFactory(store, commands, clock),
		Session: newSessionFactory(SessionDeps{
			Store:       store,
			Probe:       probe,
			Runner:      osexec.NewInteractiveRunner(),
			Commands:    commands,
			Environment: environment,
			Clock:       clock,
		}),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
}
