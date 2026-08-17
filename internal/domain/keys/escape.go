package keys

// maxSequenceLength bounds how long an escape sequence may be before basalt
// gives up and discards it. Real sequences are a handful of bytes; the cap is a
// resource guard against a hostile or malfunctioning terminal (Rule 25).
const maxSequenceLength = 32

// decodeEscape handles every sequence introduced by ESC.
//
// Two introducers matter in practice: CSI ("\x1b[") for cursor and navigation
// keys, and SS3 ("\x1bO") which some terminals use for Home/End and for the
// arrows in application cursor mode.
func decodeEscape(buf []byte) (Key, int, bool) {
	if len(buf) < 2 {
		return Key{}, 0, false
	}
	switch buf[1] {
	case '[':
		return decodeCSI(buf)
	case 'O':
		return decodeSS3(buf)
	case escape:
		// Two Escapes in quick succession arrive as one read. Consuming both
		// as a single meta-key would swallow the pair, and Escape is basalt's
		// back-out-and-quit key — so only the first byte is taken and the
		// second is decoded on the next pass.
		return Key{Kind: KindEscape}, 1, true
	default:
		// ESC followed by anything else is a Meta-prefixed key, which basalt
		// does not bind. Consume both bytes so it cannot leak into the search.
		return Key{Kind: KindUnknown}, 2, true
	}
}

// decodeSS3 handles "\x1bO<final>".
func decodeSS3(buf []byte) (Key, int, bool) {
	if len(buf) < 3 {
		return Key{}, 0, false
	}
	if k, ok := finalBytes[buf[2]]; ok {
		return k, 3, true
	}
	return Key{Kind: KindUnknown}, 3, true
}

// decodeCSI handles "\x1b[<params><final>", covering both the letter finals
// (arrows, Home, End, Shift+Tab) and the numeric "~" forms (PageUp, F5).
func decodeCSI(buf []byte) (Key, int, bool) {
	end, ok := findFinalByte(buf)
	if !ok {
		return Key{}, 0, false
	}
	final := buf[end]
	if final == '~' {
		return numericKey(string(buf[2:end])), end + 1, true
	}
	if k, known := finalBytes[final]; known {
		return k, end + 1, true
	}
	return Key{Kind: KindUnknown}, end + 1, true
}

// findFinalByte locates the terminating byte of a CSI sequence, which is the
// first byte in the range 0x40–0x7E after the parameter bytes.
//
// The scan is bounded by maxSequenceLength so a stream of parameter bytes with
// no terminator cannot make the decoder buffer without limit.
func findFinalByte(buf []byte) (int, bool) {
	for i := 2; i < len(buf) && i < maxSequenceLength; i++ {
		if buf[i] >= 0x40 && buf[i] <= 0x7E {
			return i, true
		}
	}
	if len(buf) >= maxSequenceLength {
		return maxSequenceLength - 1, true
	}
	return 0, false
}

// numericKey maps the "\x1b[<n>~" family by its parameter.
func numericKey(param string) Key {
	if k, ok := tildeParams[param]; ok {
		return k
	}
	return Key{Kind: KindUnknown}
}

// finalBytes maps the letter-terminated CSI and SS3 sequences. "Z" is
// Shift+Tab, which walks the panel focus backwards.
var finalBytes = map[byte]Key{
	'A': {Kind: KindUp},
	'B': {Kind: KindDown},
	'C': {Kind: KindRight},
	'D': {Kind: KindLeft},
	'H': {Kind: KindHome},
	'F': {Kind: KindEnd},
	'Z': {Kind: KindBackTab},
}

// tildeParams maps the numeric CSI sequences terminated by "~".
var tildeParams = map[string]Key{
	"1":  {Kind: KindHome},
	"4":  {Kind: KindEnd},
	"5":  {Kind: KindPageUp},
	"6":  {Kind: KindPageDown},
	"7":  {Kind: KindHome},
	"8":  {Kind: KindEnd},
	"15": {Kind: KindRefresh},
}
