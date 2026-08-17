package target

import (
	"sort"
	"strings"
)

// Language is the source language a rule kind builds.
type Language string

const (
	// LanguageC covers C and C++ — cc_* rules do not distinguish them, and
	// neither does the toolchain the operator cares about (gcc and g++ ship
	// and version together).
	LanguageC Language = "C"
	// LanguageGo covers the go_* rules.
	LanguageGo Language = "Go"
)

// LanguageOf reports which language a rule kind builds, and whether it builds
// one at all — a genrule or a filegroup does not.
func LanguageOf(kind Kind) (Language, bool) {
	switch {
	case strings.HasPrefix(string(kind), "go_"):
		return LanguageGo, true
	case strings.HasPrefix(string(kind), "cc_"):
		return LanguageC, true
	default:
		return "", false
	}
}

// Languages returns the distinct languages built by a set of rule kinds, sorted
// alphabetically — so a package holding both is always reported as C then Go
// rather than in whatever order its targets happened to be indexed.
func Languages(kinds []Kind) []Language {
	seen := map[Language]bool{}
	for _, kind := range kinds {
		if language, ok := LanguageOf(kind); ok {
			seen[language] = true
		}
	}
	found := make([]Language, 0, len(seen))
	for language := range seen {
		found = append(found, language)
	}
	sort.Slice(found, func(i, j int) bool { return found[i] < found[j] })
	return found
}

// packagingKinds are the rules that produce a distributable artifact.
var packagingKinds = map[Kind]bool{
	"pkg_rpm":       true,
	"pkg_tar":       true,
	"pkg_tar_impl":  true,
	"pkg_deb":       true,
	"pkg_zip":       true,
	"pkg_files":     true,
	"pkg_filegroup": true,
}

// packagingSuffixes catch the other convention in this repo: a genrule named
// for the artifact it wraps, like //src/go/tools/basalt:basalt_rpm. The rule
// kind alone cannot tell those apart from any other genrule.
var packagingSuffixes = []string{"_rpm", "_tgz", "_tar", "_deb", "_appimage", "_zip"}

// IsPackaging reports whether a target produces an installable or distributable
// artifact, by rule kind or by the naming convention used for genrule-based
// packaging.
func IsPackaging(t Target) bool {
	if packagingKinds[t.Kind] {
		return true
	}
	for _, suffix := range packagingSuffixes {
		if strings.HasSuffix(t.Name, suffix) {
			return true
		}
	}
	return false
}
