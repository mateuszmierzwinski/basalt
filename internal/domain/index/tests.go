package index

import (
	"strings"

	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
)

// testIndex answers "is there anything to test under this package" in constant
// time, by pre-expanding every test rule's package into the chain of packages
// that would reach it with a `/...` wildcard.
//
// Expanding once beats scanning the target list per lookup: the answer is
// needed on every cursor move, and the target set only changes on a re-index.
type testIndex struct {
	covered map[string]struct{}
}

// NewTestIndex builds an empty index. It is populated by Replace, which the
// session calls with the startup index and again after each re-index.
func NewTestIndex() MutableTestIndex {
	return &testIndex{covered: map[string]struct{}{}}
}

// Replace rebuilds the index from a target set.
//
// It is called only from the session loop, which is also the only caller of
// HasTestsUnder, so the map is never touched from two goroutines.
func (i *testIndex) Replace(index Index) {
	covered := make(map[string]struct{}, len(index.Targets))
	for _, entry := range index.Targets {
		if !isTestRule(entry.Kind) {
			continue
		}
		coverAncestors(covered, entry.Package)
	}
	i.covered = covered
}

// HasTestsUnder reports whether `bazel test <pkg>/...` would find anything.
func (i *testIndex) HasTestsUnder(pkg string) bool {
	_, ok := i.covered[pkg]
	return ok
}

// coverAncestors marks a test rule's package and every package above it, since
// a wildcard at any of those levels reaches the test.
func coverAncestors(covered map[string]struct{}, pkg string) {
	for current := pkg; strings.Contains(current, "/"); current = current[:strings.LastIndex(current, "/")] {
		covered[current] = struct{}{}
		if current == labelRoot {
			return
		}
	}
}

// labelRoot is the shortest package a wildcard is worth offering for. Walking
// above it would end at "//", which is the whole monorepo — not what selecting
// one target means.
const labelRoot = "//src"

const testKindSuffix = "_test"

func isTestRule(kind target.Kind) bool {
	return strings.HasSuffix(string(kind), testKindSuffix)
}
