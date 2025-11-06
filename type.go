package tun2socks

import (
    "net"
)

type TransportHandler interface {
    HandleTCP(net.Conn)
    HandleUDP(net.Conn)
    HandleICMP(*ICMPConn)
}
