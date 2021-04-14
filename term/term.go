// +build linux

package term

import (
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/sys/unix"
)

func GetSize(fd uintptr) (uint16, uint16, error) {
	sz, err := unix.IoctlGetWinsize(int(fd), unix.TIOCGWINSZ)
	return sz.Col, sz.Row, err
}

func IsTerminal(fd uintptr) bool {
	_, err := unix.IoctlGetTermios(int(fd), unix.TCGETS)
	return err != unix.ENOTTY
}

func MakeRaw(fd uintptr) (func() error, error) {
	termios, err := unix.IoctlGetTermios(int(fd), unix.TCGETS)
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
	err = unix.IoctlSetTermios(int(fd), unix.TCSETS, termios)
	if err != nil {
		return nil, err
	}

	return func() error {
		return unix.IoctlSetTermios(int(fd), unix.TCSETS, &old)
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
