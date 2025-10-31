package tun2socks

import (
    "io"
    "log"
    "runtime"

    "gvisor.dev/gvisor/pkg/tcpip"
    "gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
    "gvisor.dev/gvisor/pkg/tcpip/header"
    "gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
    "gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
    "gvisor.dev/gvisor/pkg/tcpip/stack"
    "gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
    "gvisor.dev/gvisor/pkg/tcpip/transport/udp"
    "gvisor.dev/gvisor/pkg/waiter"
)

type Tun2socks struct {
    s  *stack.Stack
    th TransportHandler
}

func New(tun io.ReadWriteCloser, th TransportHandler) *Tun2socks {

    s := stack.New(stack.Options{
        // icmp.NewProtocol4,
        NetworkProtocols:   []stack.NetworkProtocolFactory{ipv4.NewProtocol, ipv6.NewProtocol},
        TransportProtocols: []stack.TransportProtocolFactory{tcp.NewProtocol, udp.NewProtocol},
    })

    if err := s.CreateNIC(1, &endpoint{tun: tun}); err != nil {
        panic(err)
    }

    s.SetPromiscuousMode(1, true)
    s.SetSpoofing(1, true)
    s.SetNICForwarding(1, ipv4.ProtocolNumber, true)
    s.SetNICForwarding(1, ipv6.ProtocolNumber, true)

    // a default route is required by udp sending
    s.SetRouteTable([]tcpip.Route{
        {NIC: 1, Destination: header.IPv4EmptySubnet},
        {NIC: 1, Destination: header.IPv6EmptySubnet},
    })

    t2s := &Tun2socks{th: th, s: s}

    tcpForwarder := tcp.NewForwarder(s, 0, 2<<10, func(r *tcp.ForwarderRequest) {
        var wq waiter.Queue
        ep, err := r.CreateEndpoint(&wq)
        if nil != err {
            r.Complete(true)
            return
        }
        defer r.Complete(false)
        conn := gonet.NewTCPConn(&wq, ep)
        log.Println("forward", conn.RemoteAddr().String(), conn.LocalAddr().String())
        if nil != t2s.th.TcpHandle(conn) {
            conn.Close()
        } else {
            runtime.SetFinalizer(conn, (*gonet.TCPConn).Close)
        }
    })

    udpForwarder := udp.NewForwarder(s, func(r *udp.ForwarderRequest) bool {
        var wq waiter.Queue
        ep, err := r.CreateEndpoint(&wq)
        if nil != err {
            // r.Complete(true)
            return false
        }
        // defer r.Complete(false)
        conn := gonet.NewUDPConn(&wq, ep)
        log.Println("udp", conn.RemoteAddr().String(), conn.LocalAddr().String())
        if nil != t2s.th.UdpHandle(conn) {
            conn.Close()
        } else {
            runtime.SetFinalizer(conn, (*gonet.UDPConn).Close)
        }
        return true
    })

    s.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpForwarder.HandlePacket)
    s.SetTransportProtocolHandler(udp.ProtocolNumber, udpForwarder.HandlePacket)

    return t2s
}

func (t2s *Tun2socks) Close() error {
    t2s.s.Close()
    t2s.s.Wait()
    return nil
}
