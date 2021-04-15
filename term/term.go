// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris zos

package term

import (
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/sys/unix"
)

// GetSize returns the size (width, height) of the terminal. It returns
// an error if the file descriptor fd is not connected to a terminal.
func GetSize(fd uintptr) (uint16, uint16, error) {
	sz, err := unix.IoctlGetWinsize(int(fd), unix.TIOCGWINSZ)
	return sz.Col, sz.Row, err
}

// IsTerminal returns whether the file descriptor fd is connected to a terminal.
func IsTerminal(fd uintptr) bool {
	_, err := unix.IoctlGetTermios(int(fd), termiosGet)
	return err != unix.ENOTTY
}

// MakeRaw puts the terminal into raw mode as decribed in section "Raw Mode"
// in the termios(3) manpage. It returns an error if the file descriptor fd
// is not connected to a terminal. The returned function can be used to restore
// the terminal to its previous state.
//   restore, err := term.MakeRaw(os.Stdout.Fd())
//   if err != nil {
//       panic(err)
//   }
//   defer restore()
func MakeRaw(fd uintptr) (func() error, error) {
	termios, err := unix.IoctlGetTermios(int(fd), termiosGet)
	if err != nil {
		return nil, err
	}
	old := *termios

	// From termios(3) manpage section "Raw mode":
	// termios_p->c_iflag &= ~(IGNBRK | BRKINT | PARMRK | ISTRIP
	//                 | INLCR | IGNCR | ICRNL | IXON);
	// termios_p->c_oflag &= ~OPOST;
	// termios_p->c_lflag &= ~(ECHO | ECHONL | ICANON | ISIG | IEXTEN);
	// termios_p->c_cflag &= ~(CSIZE | PARENB);
	// termios_p->c_cflag |= CS8;
	termios.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP |
		unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	termios.Oflag &^= unix.OPOST
	termios.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	termios.Cflag &^= unix.CSIZE | unix.PARENB
	termios.Cflag |= unix.CS8
	err = unix.IoctlSetTermios(int(fd), termiosSet, termios)
	if err != nil {
		return nil, err
	}

	return func() error {
		return unix.IoctlSetTermios(int(fd), termiosSet, &old)
	}, nil
}

func center(s string, w int) string {
	strLen := utf8.RuneCountInString(s)
	if strLen >= w {
		return s
	}
	left := (w - strLen) / 2
	right := w - strLen - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

func maxInt(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func checkIsTerminal() {
	if !(IsTerminal(os.Stdin.Fd()) && IsTerminal(os.Stdout.Fd())) {
		panic("STDIN and STDOUT must be connected to a terminal")
	}
}
