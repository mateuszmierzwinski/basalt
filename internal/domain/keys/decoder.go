package keys

import "unicode/utf8"

// escape is the byte that begins every multi-byte key sequence.
const escape = 0x1B

// Decode returns the key at the head of buf and the number of bytes it
// consumed.
//
// ok is false when buf holds the start of a sequence that is not yet complete —
// a lone ESC, or a CSI introducer with no final byte. The caller must read more
// input and try again, or call Flush once it is satisfied no more is coming.
// This is the ambiguity that makes a bare Escape indistinguishable from the
// first byte of an arrow key until more bytes arrive or time passes.
func Decode(buf []byte) (Key, int, bool) {
	if len(buf) == 0 {
		return Key{}, 0, false
	}
	if buf[0] == escape {
		return decodeEscape(buf)
	}
	if k, ok := controlKeys[buf[0]]; ok {
		return k, 1, true
	}
	return decodeRune(buf)
}

// Flush decodes buf knowing that no further input will arrive, which resolves a
// dangling ESC to the Escape key.
func Flush(buf []byte) (Key, int) {
	if len(buf) == 1 && buf[0] == escape {
		return Key{Kind: KindEscape}, 1
	}
	if k, n, ok := Decode(buf); ok {
		return k, n
	}
	// An incomplete sequence that will never complete is discarded whole
	// rather than leaked to the search field as stray characters.
	return Key{Kind: KindUnknown}, len(buf)
}

// decodeRune reads one UTF-8 character, reporting incompleteness when the
// buffer holds only part of a multi-byte sequence.
func decodeRune(buf []byte) (Key, int, bool) {
	r, size := utf8.DecodeRune(buf)
	if r == utf8.RuneError && size <= 1 {
		if !utf8.FullRune(buf) {
			return Key{}, 0, false
		}
		return Key{Kind: KindUnknown}, 1, true
	}
	return Key{Kind: KindRune, Rune: r}, size, true
}

// controlKeys maps the single-byte C0 controls basalt acts on.
//
// 0x08 (Ctrl+H) and 0x7F (DEL) both mean Backspace: terminals disagree about
// which one the Backspace key sends, and basalt must work on both.
var controlKeys = map[byte]Key{
	0x03: {Kind: KindInterrupt},
	0x04: {Kind: KindEOF},
	0x08: {Kind: KindBackspace},
	0x09: {Kind: KindTab},
	0x0A: {Kind: KindEnter},
	0x0C: {Kind: KindRedraw},
	0x0D: {Kind: KindEnter},
	0x12: {Kind: KindToggleMode},
	0x15: {Kind: KindClearLine},
	0x17: {Kind: KindDeleteWord},
	0x7F: {Kind: KindBackspace},
}
