// Package keys decodes a raw terminal byte stream into key events.
//
// It is pure: no terminal, no syscalls, no timers. The adapter feeds it bytes
// and it reports what was pressed, which is what makes every binding in basalt
// testable without a pty.
package keys

// Kind identifies a key press.
type Kind int

const (
	// KindUnknown is an input basalt has no meaning for; callers ignore it.
	KindUnknown Kind = iota
	// KindRune is a printable character, carried in Key.Rune.
	KindRune
	KindEnter
	KindTab
	KindBackTab
	KindEscape
	KindBackspace
	KindUp
	KindDown
	KindLeft
	KindRight
	KindPageUp
	KindPageDown
	KindHome
	KindEnd
	// KindInterrupt is Ctrl+C.
	KindInterrupt
	// KindEOF is Ctrl+D.
	KindEOF
	// KindClearLine is Ctrl+U.
	KindClearLine
	// KindDeleteWord is Ctrl+W.
	KindDeleteWord
	// KindToggleMode is Ctrl+R, which switches literal and regex search.
	KindToggleMode
	// KindRedraw is Ctrl+L.
	KindRedraw
	// KindRefresh is F5, which forces a re-index.
	KindRefresh
)

// Key is a single decoded key press.
type Key struct {
	Kind Kind
	Rune rune
}

// IsRune reports whether the key carries a printable character.
func (k Key) IsRune() bool { return k.Kind == KindRune }
