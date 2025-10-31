package tun2socks

import (
    "net"
    "time"
)

type TransportHandler interface {
    TcpHandle(net.Conn) error
    UdpHandle(net.Conn) error
}

type HasReadDeadline interface {
    SetReadDeadline(time.Time) error
}
