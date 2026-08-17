// Package toolchain reports which compilers are available and which version a
// project asks for, so the header line can show required against available.
package toolchain

import "github.com/mateuszmierzwinski/basalt/internal/domain/target"

// Versions is what basalt knows about the host's compilers. Any field may be
// empty: a machine without g++ is a perfectly ordinary machine, and basalt says
// so rather than pretending.
type Versions struct {
	// GoAvailable is the host Go toolchain, e.g. "1.26.5".
	GoAvailable string
	// GCC and GXX are the C and C++ compilers, usually equal.
	GCC string
	GXX string
}

// Summary is the header line's content for one selection: the languages in
// play, what they need, and what the host offers.
type Summary struct {
	Languages   []target.Language
	GoRequired  string
	GoAvailable string
	GCC         string
	GXX         string
}

// GoSatisfied reports whether the available Go toolchain is at least the
// required one. An unknown version on either side counts as satisfied — basalt
// reports what it knows and does not raise an alarm it cannot substantiate.
//
// There is no native equivalent: nothing in the tree declares a required gcc,
// so there is no requirement to fall short of.
func (s Summary) GoSatisfied() bool {
	if s.GoRequired == "" || s.GoAvailable == "" {
		return true
	}
	return CompareVersions(s.GoAvailable, s.GoRequired) >= 0
}
