package tun2socks

import "errors"

type netError struct {
    errMessage  string
    isTimeout   bool
    isTemporary bool
}

func (e *netError) Error() string {
    return e.errMessage
}

func (e *netError) Timeout() bool {
    return e.isTimeout
}

func (e *netError) Temporary() bool {
    return e.isTemporary
}

//
var ErrTimeout = &netError{"io timeout", true, true}
var ErrNotSupported = errors.New("not supported")

// var ErrClosedPipe = io.ErrClosedPipe
// Keep log silent just like io.EOF
var ErrClosedPipe error = nil
