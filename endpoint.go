package tun2socks

import (
    "io"
    "log"

    "gvisor.dev/gvisor/pkg/buffer"
    "gvisor.dev/gvisor/pkg/tcpip"
    "gvisor.dev/gvisor/pkg/tcpip/header"
    "gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
    "gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
    "gvisor.dev/gvisor/pkg/tcpip/stack"
)

// 1500 - ipHeader - tcpHeader - tcpTimestamp - packetCustomHeader
const PAYLOAD = 1500 - 20 - 20 - 12 - 8
const MTU = PAYLOAD + 20 + 20

type endpoint struct {
    tun io.ReadWriteCloser
}

func (e *endpoint) SetMTU(mtu uint32) {
    // TODO implement me
    panic("implement me")
}

func (e *endpoint) SetLinkAddress(addr tcpip.LinkAddress) {
    // TODO implement me
    panic("implement me")
}

func (e *endpoint) AddHeader(packetBuffer *stack.PacketBuffer) {
    // TODO implement me
    // panic("implement me")
}

func (e *endpoint) ParseHeader(packetBuffer *stack.PacketBuffer) bool {
    // TODO implement me
    panic("implement me")
}

func (e *endpoint) Close() {
    // TODO implement me
    // panic("implement me")
}

func (e *endpoint) SetOnCloseAction(f func()) {
    // TODO implement me
    // panic("implement me")
}

func (e *endpoint) ARPHardwareType() header.ARPHardwareType {
    panic(ErrNotSupported)
}

func (e *endpoint) MTU() uint32 {
    return MTU
}

func (e *endpoint) Capabilities() stack.LinkEndpointCapabilities {
    return stack.CapabilityNone
}

// Tun has none link-layer header, just returns 0.
func (e *endpoint) MaxHeaderLength() uint16 {
    return 0
}

// Mac address is required by ICMPv6
func (e *endpoint) LinkAddress() tcpip.LinkAddress {
    // https://stackoverflow.com/questions/21018729/generate-mac-address-in-go
    // mac[0] = (mac[0] | 2) & 0xfe // Set local bit, ensure unicast address
    return "\xFE\xFF\xFF\xFF\xFF\xFF"
}

func (e *endpoint) WritePackets(list stack.PacketBufferList) (n int, err tcpip.Error) {
    for _, pb := range list.AsSlice() {
        nw, err := e.tun.Write(pb.ToView().AsSlice())
        if nil != err {
            log.Println(err)
            return n, &tcpip.ErrInvalidEndpointState{}
        } else if nw != pb.Size() {
            panic(io.ErrShortWrite)
        }
        n += pb.Size()
        // pb.DecRef()
    }
    return
}

func (e *endpoint) Attach(dispatcher stack.NetworkDispatcher) {
    go func(tun io.Reader) {
        for {
            v := buffer.NewViewSize(MTU)
            nr, err := tun.Read(v.AsSlice())
            if nil != err {
                log.Println(err)
                break
            }

            v.CapLength(nr)

            pb := stack.NewPacketBuffer(stack.PacketBufferOptions{
                Payload: buffer.MakeWithView(v),
                OnRelease: func() {
                    // log.Println("test ok OnRelease")
                }})

            if 0x40 == v.AsSlice()[0]&0xf0 {
                dispatcher.DeliverNetworkPacket(ipv4.ProtocolNumber, pb)
                // log.Println("tcp4", v.Size())
            } else {
                // https://en.wikipedia.org/wiki/List_of_IP_Protocol_numbers
                // header.IPv6(v).TransportProtocol() == header.ICMPv6ProtocolNumber
                dispatcher.DeliverNetworkPacket(ipv6.ProtocolNumber, pb)
                // log.Println("tcp6", v.Size())
            }
            pb.DecRef()
        }
    }(e.tun)
}

func (e *endpoint) IsAttached() bool {
    panic(ErrNotSupported)
}

// Called by Stack.Wait()
// Once tun is closed, tun.Read(v) return error, the loop in Attach() break.
func (e *endpoint) Wait() {
    e.tun.Close()
}
