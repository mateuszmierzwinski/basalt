package index

// TestIndex is the read side of the test lookup: it answers whether a package
// has anything testable beneath it, which is what decides whether the action
// panel offers to test a target that is not itself a test.
type TestIndex interface {
	HasTestsUnder(pkg string) bool
}

// MutableTestIndex adds the write side, held by the session so a re-index
// replaces the answers without rebuilding the action provider around it.
type MutableTestIndex interface {
	TestIndex
	Replace(index Index)
}
