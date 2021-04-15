// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris zos

/*
Package term provides functions on unixoid systems for dealing with
POSIX compliant terminals/terminal emulators that also support
ANSI escape sequences.

It is only tested on Linux with the Xfce terminal emulator and the Linux console.

All inputs can be canceled with ^D (EOF).
*/
package term
