package errorx

import (
	"golang.org/x/sys/unix"
)

const ECANCELED = unix.ECANCELED
const ECONNREFUSED = unix.ECONNREFUSED
const ECONNRESET = unix.ECONNRESET
const EHOSTUNREACH = unix.EHOSTUNREACH
const ETIMEDOUT = unix.ETIMEDOUT
