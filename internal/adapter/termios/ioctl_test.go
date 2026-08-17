package termios

import (
	"syscall"
	"testing"
)

// makeRaw is the one piece of this package that is pure enough to test without
// a real tty, and it is also the piece whose correctness matters most: getting
// it wrong leaves the operator's shell unusable.
func TestMakeRawDisablesEchoAndCanonicalMode(t *testing.T) {
	var original syscall.Termios
	original.Lflag = syscall.ECHO | syscall.ICANON | syscall.ISIG | syscall.IEXTEN | syscall.ECHONL

	raw := makeRaw(original)
	for name, flag := range map[string]uint32{
		"ECHO":   syscall.ECHO,
		"ICANON": syscall.ICANON,
		"ISIG":   syscall.ISIG,
		"IEXTEN": syscall.IEXTEN,
		"ECHONL": syscall.ECHONL,
	} {
		if raw.Lflag&flag != 0 {
			t.Fatalf("%s survived makeRaw", name)
		}
	}
}

// ISIG must be cleared so Ctrl+C arrives as a byte the decoder handles, rather
// than killing basalt before it can restore the terminal.
func TestMakeRawClearsSignalGeneration(t *testing.T) {
	var original syscall.Termios
	original.Lflag = syscall.ISIG
	if makeRaw(original).Lflag&syscall.ISIG != 0 {
		t.Fatal("ISIG survived — Ctrl+C would kill basalt before it restores the tty")
	}
}

func TestMakeRawDisablesInputProcessing(t *testing.T) {
	var original syscall.Termios
	original.Iflag = syscall.ICRNL | syscall.IXON | syscall.ISTRIP | syscall.INLCR

	raw := makeRaw(original)
	if raw.Iflag&(syscall.ICRNL|syscall.IXON|syscall.ISTRIP|syscall.INLCR) != 0 {
		t.Fatalf("input flags survived: %x", raw.Iflag)
	}
}

func TestMakeRawDisablesOutputPostProcessing(t *testing.T) {
	var original syscall.Termios
	original.Oflag = syscall.OPOST
	if makeRaw(original).Oflag&syscall.OPOST != 0 {
		t.Fatal("OPOST survived")
	}
}

func TestMakeRawSetsEightBitCharacters(t *testing.T) {
	var original syscall.Termios
	original.Cflag = syscall.PARENB
	raw := makeRaw(original)
	if raw.Cflag&syscall.CS8 == 0 {
		t.Fatal("CS8 was not set")
	}
	if raw.Cflag&syscall.PARENB != 0 {
		t.Fatal("PARENB survived")
	}
}

// Reads must time out rather than block forever, so the reader can be paused
// between reads and the tty handed to a child process.
func TestMakeRawUsesPollingReads(t *testing.T) {
	raw := makeRaw(syscall.Termios{})
	if raw.Cc[syscall.VMIN] != pollVMIN || raw.Cc[syscall.VTIME] != pollVTIME {
		t.Fatalf("VMIN/VTIME = %d/%d, want %d/%d",
			raw.Cc[syscall.VMIN], raw.Cc[syscall.VTIME], pollVMIN, pollVTIME)
	}
	if raw.Cc[syscall.VMIN] != 0 {
		t.Fatal("a blocking read would steal a child process's first keystroke")
	}
}

// makeRaw must not mutate its argument: the caller keeps it to restore with.
func TestMakeRawDoesNotMutateItsInput(t *testing.T) {
	var original syscall.Termios
	original.Lflag = syscall.ECHO | syscall.ICANON
	before := original.Lflag

	makeRaw(original)
	if original.Lflag != before {
		t.Fatal("makeRaw mutated the saved settings — Restore would put back raw mode")
	}
}

func BenchmarkMakeRaw(b *testing.B) {
	var original syscall.Termios
	original.Lflag = syscall.ECHO | syscall.ICANON
	for b.Loop() {
		makeRaw(original)
	}
}
