package toolchain

import (
	"strconv"
	"strings"
)

// ParseGoVersion extracts the dotted version from `go version` output.
//
// The Fedora toolchain reports a build variant — "go1.26.5-X:nodwarf5" — so the
// suffix is trimmed rather than carried into a comparison that would then find
// "5-X" where it expected a number.
func ParseGoVersion(output string) string {
	for _, field := range strings.Fields(output) {
		if !strings.HasPrefix(field, "go1") {
			continue
		}
		return trimVariant(strings.TrimPrefix(field, "go"))
	}
	return ""
}

// ParseGoDirective extracts the version from a go.mod's `go` line.
func ParseGoDirective(contents string) string {
	for _, line := range strings.Split(contents, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "go ") {
			continue
		}
		return trimVariant(strings.TrimSpace(strings.TrimPrefix(trimmed, "go ")))
	}
	return ""
}

// ParseCompilerVersion extracts the dotted version from a `gcc --version` or
// `g++ --version` banner, whose first line looks like
// "gcc (GCC) 16.1.1 20260515 (Red Hat 16.1.1-2)".
func ParseCompilerVersion(output string) string {
	first, _, _ := strings.Cut(output, "\n")
	for _, field := range strings.Fields(first) {
		if looksLikeVersion(field) {
			return field
		}
	}
	return ""
}

// looksLikeVersion accepts a dotted numeric run, which is what distinguishes
// "16.1.1" from the date and the parenthesised vendor string beside it.
func looksLikeVersion(field string) bool {
	parts := strings.Split(field, ".")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}
	return true
}

// trimVariant drops a distribution's build suffix from a version.
func trimVariant(version string) string {
	for _, separator := range []string{"-", "+", " "} {
		version, _, _ = strings.Cut(version, separator)
	}
	return version
}

// CompareVersions orders two dotted versions, returning -1, 0 or 1. Missing
// components count as zero, so "1.26" and "1.26.0" are equal.
func CompareVersions(left, right string) int {
	leftParts, rightParts := strings.Split(left, "."), strings.Split(right, ".")
	for i := 0; i < max(len(leftParts), len(rightParts)); i++ {
		difference := componentAt(leftParts, i) - componentAt(rightParts, i)
		if difference != 0 {
			return sign(difference)
		}
	}
	return 0
}

func componentAt(parts []string, index int) int {
	if index >= len(parts) {
		return 0
	}
	value, err := strconv.Atoi(parts[index])
	if err != nil {
		return 0
	}
	return value
}

func sign(value int) int {
	if value < 0 {
		return -1
	}
	return 1
}
