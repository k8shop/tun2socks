package tun2socks

import (
    "io"
    "log/slog"

    "gvisor.dev/gvisor/pkg/tcpip"
    "gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
    "gvisor.dev/gvisor/pkg/tcpip/header"
    "gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
    "gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
    "gvisor.dev/gvisor/pkg/tcpip/stack"
    "gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
    "gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
    "gvisor.dev/gvisor/pkg/tcpip/transport/udp"
    "gvisor.dev/gvisor/pkg/waiter"
)

type Tun2socks struct {
    s  *stack.Stack
    th TransportHandler
    tun io.ReadWriteCloser
}

func New(tun io.ReadWriteCloser, th TransportHandler) *Tun2socks {

    s := stack.New(stack.Options{
        NetworkProtocols:   []stack.NetworkProtocolFactory{ipv4.NewProtocol, ipv6.NewProtocol},
        TransportProtocols: []stack.TransportProtocolFactory{tcp.NewProtocol, udp.NewProtocol, icmp.NewProtocol4, icmp.NewProtocol6},
    })

    if err := s.CreateNIC(1, NewEndpoint(tun)); err != nil {
        panic(err)
    }

    s.SetPromiscuousMode(1, true)
    s.SetSpoofing(1, true)
    s.SetNICForwarding(1, ipv4.ProtocolNumber, true)
    s.SetNICForwarding(1, ipv6.ProtocolNumber, true)

    s.SetRouteTable([]tcpip.Route{
        {NIC: 1, Destination: header.IPv4EmptySubnet},
        {NIC: 1, Destination: header.IPv6EmptySubnet},
    })

    t2s := &Tun2socks{th: th, s: s, tun: tun}

    tcpForwarder := tcp.NewForwarder(s, 0, 2<<10, func(r *tcp.ForwarderRequest) {
        var wq waiter.Queue
        ep, err := r.CreateEndpoint(&wq)
        if nil != err {
            slog.Error(err.String())
            r.Complete(true)
            return
        }
        defer r.Complete(false)
        t2s.th.HandleTCP(gonet.NewTCPConn(&wq, ep))
    })

    udpForwarder := udp.NewForwarder(s, func(r *udp.ForwarderRequest) bool {
        var wq waiter.Queue
        ep, err := r.CreateEndpoint(&wq)
        if nil != err {
            slog.Error(err.String())
            return false
        }
        t2s.th.HandleUDP(gonet.NewUDPConn(&wq, ep))
        return true
    })

    s.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpForwarder.HandlePacket)
    s.SetTransportProtocolHandler(udp.ProtocolNumber, udpForwarder.HandlePacket)

    icmpForwarder := NewICMPForwarder(t2s)
    s.SetTransportProtocolHandler(icmp.ProtocolNumber4, icmpForwarder.HandlePacket)
    s.SetTransportProtocolHandler(icmp.ProtocolNumber6, icmpForwarder.HandlePacket)

    return t2s
}

func (t2s *Tun2socks) Close() error {
    t2s.s.Destroy()
    return nil
}
