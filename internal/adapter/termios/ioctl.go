package termios

import (
	"syscall"
	"unsafe"
)

// Reads time out rather than blocking forever: VMIN=0 with VTIME=1 returns
// after a tenth of a second with whatever arrived, possibly nothing.
//
// This is what lets basalt hand the terminal to nano, claude or bazel. A reader
// blocked indefinitely in read(2) on the tty would steal the child's first
// keystroke, and there is no way to cancel a blocked read. With a timeout the
// reader can be paused between reads instead, and the longest a child can wait
// for basalt to let go is one poll interval.
//
// It also disambiguates a bare Escape from the start of an arrow key: a read
// that times out with a lone ESC pending proves no more bytes are coming.
const (
	pollVMIN  = 0
	pollVTIME = 1
)

// winsize mirrors the kernel's struct winsize for TIOCGWINSZ.
type winsize struct {
	rows   uint16
	cols   uint16
	xpixel uint16
	ypixel uint16
}

// getTermios reads the current line discipline settings.
func getTermios(fd uintptr) (syscall.Termios, error) {
	var t syscall.Termios
	if err := ioctl(fd, syscall.TCGETS, unsafe.Pointer(&t)); err != nil {
		return t, err
	}
	return t, nil
}

// setTermios installs line discipline settings.
func setTermios(fd uintptr, t syscall.Termios) error {
	return ioctl(fd, syscall.TCSETS, unsafe.Pointer(&t))
}

// getWinsize reads the terminal's dimensions.
func getWinsize(fd uintptr) (winsize, error) {
	var ws winsize
	if err := ioctl(fd, syscall.TIOCGWINSZ, unsafe.Pointer(&ws)); err != nil {
		return ws, err
	}
	return ws, nil
}

// ioctl issues a terminal ioctl, translating the errno into an error.
func ioctl(fd, request uintptr, arg unsafe.Pointer) error {
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, request, uintptr(arg)); errno != 0 {
		return errno
	}
	return nil
}

// makeRaw returns a copy of t with canonical input, echo and signal generation
// disabled — the equivalent of cfmakeraw(3).
//
// ISIG is cleared deliberately: Ctrl+C must arrive as the byte 0x03 for the key
// decoder to handle, rather than raising SIGINT and killing basalt before it
// can restore the terminal.
func makeRaw(t syscall.Termios) syscall.Termios {
	raw := t
	raw.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP |
		syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	raw.Cflag &^= syscall.CSIZE | syscall.PARENB
	raw.Cflag |= syscall.CS8
	raw.Cc[syscall.VMIN] = pollVMIN
	raw.Cc[syscall.VTIME] = pollVTIME
	return raw
}
