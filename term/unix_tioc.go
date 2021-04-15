// +build darwin dragonfly freebsd netbsd openbsd

package term

import "golang.org/x/sys/unix"

const (
	termiosGet = unix.TIOCGETA
	termiosSet = unix.TIOCSETA
)
