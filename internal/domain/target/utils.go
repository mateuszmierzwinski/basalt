package target

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	labelPrefix    = "//"
	labelSeparator = ":"
	queryRuleWord  = "rule"
	queryFieldsLen = 3
)

// reject wraps a sentinel with the offending value, quoted. Every parse failure
// in this package reports the same way, so the shape is named once.
func reject(sentinel error, value string) error {
	return fmt.Errorf("%w: %q", sentinel, value)
}

// ParseLabelKind converts one line of `bazel query --output=label_kind` output
// — "<kind> rule <label>" — into a Target.
func ParseLabelKind(line string) (Target, error) {
	fields := strings.Fields(line)
	if len(fields) != queryFieldsLen || fields[1] != queryRuleWord {
		return Target{}, reject(ErrMalformedQueryLine, line)
	}
	return New(Kind(fields[0]), fields[2])
}

// New builds a Target from a rule kind and a fully-qualified label, rejecting
// anything that is not a well-formed absolute Bazel label.
func New(kind Kind, label string) (Target, error) {
	pkg, name, err := SplitLabel(label)
	if err != nil {
		return Target{}, err
	}
	return Target{
		Kind:    kind,
		Package: pkg,
		Name:    name,
		Label:   label,
		Display: DisplayName(pkg, name),
	}, nil
}

// SplitLabel divides "//path/to/pkg:name" into its package and target halves.
func SplitLabel(label string) (string, string, error) {
	if !strings.HasPrefix(label, labelPrefix) {
		return "", "", reject(ErrMalformedLabel, label)
	}
	pkg, name, found := strings.Cut(label, labelSeparator)
	if !found || name == "" || strings.Contains(name, labelSeparator) {
		return "", "", reject(ErrMalformedLabel, label)
	}
	if strings.TrimPrefix(pkg, labelPrefix) == "" {
		return "", "", reject(ErrMalformedLabel, label)
	}
	return pkg, name, nil
}

// DisplayName is the short form shown in the target list: the package's last
// path segment paired with the target name, e.g. "[falcon:linux]".
func DisplayName(pkg, name string) string {
	segments := strings.Split(strings.TrimPrefix(pkg, labelPrefix), "/")
	return "[" + segments[len(segments)-1] + labelSeparator + name + "]"
}

// Directory resolves a target's package to an absolute on-disk path, refusing
// any label that escapes the monorepo. Every action that opens a file or sets
// a working directory goes through here, so a poisoned cache entry such as
// "//../../etc:passwd" cannot walk basalt out of the tree.
func Directory(root string, t Target) (string, error) {
	relative := strings.TrimPrefix(t.Package, labelPrefix)
	resolved := filepath.Clean(filepath.Join(root, relative))
	if resolved != root && !strings.HasPrefix(resolved, root+string(filepath.Separator)) {
		return "", reject(ErrLabelEscapesRoot, t.Label)
	}
	return resolved, nil
}
