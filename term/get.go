// +build linux

package term

import (
	"bytes"
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
	EchoNormal EchoMode = iota
	EchoNone
	EchoMask
)

func GetBytes(echo EchoMode, limit uint8) ([]byte, error) {
	checkIsTerminal()
	result := []byte{}
	stdoutFd := int(os.Stdout.Fd())
	termios, err := unix.IoctlGetTermios(stdoutFd, unix.TCGETS)
	if err != nil {
		return result, err
	}
	old := *termios
	defer unix.IoctlSetTermios(stdoutFd, unix.TCSETS, &old)

	termios.Cc[unix.VMIN] = 1
	termios.Cc[unix.VTIME] = 0
	termios.Lflag &^= unix.ECHO | unix.ICANON
	termios.Oflag |= unix.ICRNL
	unix.IoctlSetTermios(stdoutFd, unix.TCSETS, termios)

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
		x := utf8.RuneCount(result[len(result)-n:])
		os.Stdout.Write([]byte{0x1B, '['})
		os.Stdout.Write([]byte(strconv.Itoa(x)))
		os.Stdout.Write([]byte{'D', 0x1B, '[', 'K'})
	}
	return result[:len(result)-n]
}

func GetRunes(echo EchoMode, limit uint8) ([]rune, error) {
	b, err := GetBytes(echo, limit)
	return bytes.Runes(b), err
}

func GetString(echo EchoMode, limit uint8) (string, error) {
	b, err := GetBytes(echo, limit)
	return string(b), err
}

func GetLine() (string, error) {
	s, err := GetString(EchoNormal, 0)
	os.Stdout.Write([]byte{linefeed})
	return s, err
}

func GetPassword() ([]byte, error) {
	b, err := GetBytes(EchoMask, 0)
	os.Stdout.Write([]byte{linefeed})
	return b, err
}

func GetChar(echo bool) (rune, error) {
	var mode EchoMode
	if echo {
		mode = EchoNormal
	} else {
		mode = EchoNone
	}
	s, err := GetString(mode, 1)
	if err != nil {
		return 0, err
	}
	if s == "" {
		return 0, nil
	}
	r, _ := utf8.DecodeRuneInString(s)
	return r, nil
}
