package filter

import "github.com/mateuszmierzwinski/basalt/internal/domain/target"

// Matcher filters targets by a query.
type Matcher interface {
	Filter(targets []target.Target, query string, mode Mode) Result
}
