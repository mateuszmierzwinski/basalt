package tui

import (
	"sync"

	"github.com/mateuszmierzwinski/basalt/internal/domain/keys"
)

var (
	handlersOnce sync.Once
	handlers     map[keys.Kind]KeyHandler
)

// loadHandlers populates the package-scoped binding map exactly once (Rule 6.1,
// 6.2). Every value is a named struct implementing KeyHandler (Rule 6.3).
//
// The bindings are:
//
//	Tab / Shift+Tab   move between the target list and the action panel
//	Up/Down, PgUp/PgDn, Home/End   move within the focused panel
//	Enter             targets → focus actions; actions → run the action
//	Esc               back out: actions → targets → clear query → quit
//	Ctrl+C, Ctrl+D    quit
//	Ctrl+R            toggle literal and regex search
//	Ctrl+U, Ctrl+W    clear the query, delete the last word
//	Ctrl+L            repaint
//	F5                re-index from bazel
func loadHandlers() {
	handlers = map[keys.Kind]KeyHandler{
		keys.KindInterrupt:  &quitHandler{},
		keys.KindEOF:        &quitHandler{},
		keys.KindRedraw:     &redrawHandler{},
		keys.KindRefresh:    &refreshHandler{},
		keys.KindTab:        &focusHandler{forward: true},
		keys.KindBackTab:    &focusHandler{forward: false},
		keys.KindRight:      &focusHandler{forward: true},
		keys.KindLeft:       &focusHandler{forward: false},
		keys.KindUp:         &moveHandler{delta: -1},
		keys.KindDown:       &moveHandler{delta: 1},
		keys.KindPageUp:     &moveHandler{delta: -1, page: true},
		keys.KindPageDown:   &moveHandler{delta: 1, page: true},
		keys.KindHome:       &edgeHandler{toEnd: false},
		keys.KindEnd:        &edgeHandler{toEnd: true},
		keys.KindEnter:      &selectHandler{},
		keys.KindEscape:     &escapeHandler{},
		keys.KindToggleMode: &toggleModeHandler{},
		keys.KindBackspace:  &editHandler{transform: DeleteLastRune},
		keys.KindDeleteWord: &editHandler{transform: DeleteLastWord},
		keys.KindClearLine:  &editHandler{transform: ClearQuery},
	}
}
