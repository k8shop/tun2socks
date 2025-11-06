package tun2socks

import (
    "io"
    "log"
    "net"
    "sync"
    "time"

    "gvisor.dev/gvisor/pkg/tcpip"
    "gvisor.dev/gvisor/pkg/tcpip/header"
    "gvisor.dev/gvisor/pkg/tcpip/stack"
)

type ICMPForwarder struct {
    t2s *Tun2socks
    mu sync.Mutex
    connMap map[stack.TransportEndpointID]*ICMPConn
}

type ICMPConn struct {
    forwarder *ICMPForwarder
    ipVersion int
    id stack.TransportEndpointID
    // mu sync.Mutex
    pbs chan *stack.PacketBuffer
}

func NewICMPForwarder(t2s *Tun2socks) *ICMPForwarder {
    return &ICMPForwarder{
        t2s: t2s,
        connMap: make(map[stack.TransportEndpointID]*ICMPConn),
    }
}

func (forwarder *ICMPForwarder) HandlePacket (id stack.TransportEndpointID, pb *stack.PacketBuffer) bool {
    forwarder.mu.Lock()
    defer forwarder.mu.Unlock()

    pb.IncRef()
    if conn, ok := forwarder.connMap[id]; ok {
        conn.pbs <- pb
    } else {
        conn = NewICMPConn(forwarder, id, pb)
        forwarder.t2s.th.HandleICMP(conn)
        forwarder.connMap[id] = conn
    }

    return true
}

func NewICMPConn(forwarder *ICMPForwarder, id stack.TransportEndpointID, pb *stack.PacketBuffer) *ICMPConn {
    // log.Println(bytes.Equal(header.IPv4(pb.ToView().AsSlice()).Payload(), pb.ToView().AsSlice()[20:]))
    // log.Println(bytes.Equal(pb.Network().Payload(), pb.ToView().AsSlice()[20:]))
    conn := &ICMPConn{
        forwarder: forwarder,
        ipVersion: header.IPVersion(pb.NetworkHeader().Slice()),
        id: id,
        pbs: make(chan *stack.PacketBuffer, 9),
    }
    conn.pbs <- pb
    return conn
}

func (conn *ICMPConn) Network() string {
    if conn.ipVersion == header.IPv4Version {
        return "ip4:icmp"
    } else {
        return "ip6:icmp"
    }
}

func (conn *ICMPConn) SourceAddress() net.IP {
    return conn.id.RemoteAddress.AsSlice()
}

func (conn *ICMPConn) DestinationAddress() net.IP {
    return conn.id.LocalAddress.AsSlice()
}

func (conn *ICMPConn) SetReadDeadline(time.Time) error {
    return nil
}

func (conn *ICMPConn) Read(buf []byte) (int, error) {
    var datagram []byte
    timer := time.NewTimer(3*time.Second)
    select {
    case pb := <- conn.pbs:
        timer.Stop()
        if nil == pb {
            return 0, io.ErrClosedPipe
        }
        datagram = pb.Network().Payload()
        defer pb.DecRef()
    case <- timer.C:
        return 0, ErrTimeout
    }

    if len(buf) < len(datagram) {
        return 0, io.ErrShortBuffer
    }

    return copy(buf, datagram), nil
}

func (conn *ICMPConn) Write(buf []byte) (int, error) {
    // switch header.IPVersion(buf) {
    switch conn.ipVersion {
    case header.IPv4Version:
        head := header.IPv4(buf)
        head.SetDestinationAddressWithChecksumUpdate(tcpip.AddrFrom4Slice(conn.SourceAddress()))
        log.Println(head.SourceAddress(), head.IsChecksumValid(), len(buf), conn.id)

    case header.IPv6Version:
        head := header.IPv6(buf)
        // IPv6 doesn't have checksum
        head.SetDestinationAddress(tcpip.AddrFrom16Slice(conn.SourceAddress()))
    default:
        return 0, net.UnknownNetworkError("")
    }

    // tun.Write()
    // s.WriteRawPacket(1, ipv4.ProtocolNumber, buffer.MakeWithData(buf))
    return conn.forwarder.t2s.tun.Write(buf)
}

func (conn *ICMPConn) Close() error {
    conn.forwarder.mu.Lock()
    delete(conn.forwarder.connMap, conn.id)
    conn.forwarder.mu.Unlock()

    // conn.mu.Lock()
    // defer conn.mu.Unlock()
    if conn.pbs == nil {
        return nil
    }

    close(conn.pbs)
    for {
        pb := <- conn.pbs
        if pb == nil {
            break
        }
        pb.DecRef()
    }
    conn.pbs = nil

    return nil
}
