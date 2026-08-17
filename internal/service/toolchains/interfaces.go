package toolchains

import (
	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
	"github.com/mateuszmierzwinski/basalt/internal/domain/toolchain"
)

// Prober reports what the host's compilers are and what a given target's
// project asks for.
type Prober interface {
	// SummaryFor describes the toolchain situation for one target and the
	// languages of the package it belongs to.
	SummaryFor(selected target.Target, languages []target.Language) toolchain.Summary
}
