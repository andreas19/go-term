// +build aix linux solaris zos

package term

import "golang.org/x/sys/unix"

const (
	termiosGet = unix.TCGETS
	termiosSet = unix.TCSETS
)
