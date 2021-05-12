// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris zos

package term

import (
	"io"
	"os"
	"strconv"
	"unicode"
	"unicode/utf8"

	"golang.org/x/sys/unix"
)

const (
	maskChar = '*'
	linefeed = '\n'
	space    = ' '
)

type EchoMode uint8

const (
	EchoNormal EchoMode = iota // characters are printed to the screen as typed
	EchoNone                   // nothing is printed to the screen
	EchoMask                   // an * is printed to the screen for each character
)

// GetBytes gets input from a terminal and returns it as a slice of bytes,
// which does not include the final \n (if any).
// The echo parameter controls what is printed to the screen.
// If limit > 0, its the max. number of characters to get; if the number is
// reached the input will be submitted w/o typing enter.
// It panics if stdin and stdout are not connected to a terminal.
func GetBytes(echo EchoMode, limit uint8) ([]byte, error) {
	checkIsTerminal()
	result := []byte{}
	stdoutFd := int(os.Stdout.Fd())
	termios, err := unix.IoctlGetTermios(stdoutFd, termiosGet)
	if err != nil {
		return result, err
	}
	old := *termios
	defer unix.IoctlSetTermios(stdoutFd, termiosSet, &old)

	termios.Cc[unix.VMIN] = 1
	termios.Cc[unix.VTIME] = 0
	termios.Lflag &^= unix.ECHO | unix.ICANON
	termios.Iflag |= unix.ICRNL
	unix.IoctlSetTermios(stdoutFd, termiosSet, termios)

	vEof := termios.Cc[unix.VEOF]
	vErase := termios.Cc[unix.VERASE]
	vKill := termios.Cc[unix.VKILL]
	vWerase := termios.Cc[unix.VWERASE]

	var cnt int
loop:
	for {
		buf := []byte{0, 0, 0, 0}
		cnt, err = os.Stdin.Read(buf)
		if err != nil {
			return result, err
		}
		switch buf[0] {
		case vEof:
			if len(result) == 0 {
				err = io.EOF
			}
			break loop
		case linefeed:
			break loop
		case vErase:
			if len(result) > 0 {
				_, n := utf8.DecodeLastRune(result)
				result = erase(n, result, echo != EchoNone)
			}
		case vKill:
			result = erase(len(result), result, echo != EchoNone)
		case vWerase:
			if len(result) == 0 {
				break
			}
			flag := false
			var pos int
			for pos = len(result) - 1; pos >= 0; pos-- {
				if !flag && result[pos] != space {
					flag = true
					continue
				}
				if flag && result[pos] == space {
					break
				}
			}
			result = erase(len(result)-(pos+1), result, echo != EchoNone)
		default:
			if r, _ := utf8.DecodeRune(buf); unicode.IsGraphic(r) {
				if echo == EchoNormal {
					os.Stdout.Write(buf[:cnt])
				} else if echo == EchoMask {
					os.Stdout.Write([]byte{maskChar})
				}
				result = append(result, buf[:cnt]...)
				if limit > 0 && utf8.RuneCount(result) == int(limit) {
					break loop
				}
			}
		}
	}
	return result, err
}

func erase(n int, result []byte, echo bool) []byte {
	if echo {
		if x := utf8.RuneCount(result[len(result)-n:]); x > 0 {
			os.Stdout.Write([]byte{0x1B, '['})
			os.Stdout.Write([]byte(strconv.Itoa(x)))
			os.Stdout.Write([]byte{'D', 0x1B, '[', 'K'})
		}
	}
	return result[:len(result)-n]
}

// GetLine gets one line of input from a terminal.
// It panics if stdin and stdout are not connected to a terminal.
func GetLine() (string, error) {
	b, err := GetBytes(EchoNormal, 0)
	os.Stdout.Write([]byte{linefeed})
	return string(b), err
}

// GetPassword gets one line of input from a terminal
// with the input masked with an * character.
// It panics if stdin and stdout are not connected to a terminal.
func GetPassword() ([]byte, error) {
	b, err := GetBytes(EchoMask, 0)
	os.Stdout.Write([]byte{linefeed})
	return b, err
}

// GetChar gets one character from a terminal.
// It panics if stdin and stdout are not connected to a terminal.
func GetChar(echo bool) (rune, error) {
	var mode EchoMode
	if echo {
		mode = EchoNormal
	} else {
		mode = EchoNone
	}
	b, err := GetBytes(mode, 1)
	if err != nil {
		return 0, err
	}
	if len(b) == 0 {
		return linefeed, nil
	}
	r, _ := utf8.DecodeRune(b)
	return r, nil
}
