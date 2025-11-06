package tun2socks

import (
    "fmt"
    "io"
    "log/slog"

    "gvisor.dev/gvisor/pkg/buffer"
    "gvisor.dev/gvisor/pkg/tcpip"
    "gvisor.dev/gvisor/pkg/tcpip/header"
    "gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
    "gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
    "gvisor.dev/gvisor/pkg/tcpip/stack"
)

type endpoint struct {
    mtu uint32
    tun io.ReadWriteCloser
    closeAction func()
    stopGoroutine chan any
}

func NewEndpoint(tun io.ReadWriteCloser) *endpoint {
    return &endpoint{tun: tun, mtu: 1500}
}

// space reserved for front header.
// Tun has none link-layer header, just returns 0.
func (e *endpoint) MaxHeaderLength() uint16 {
    return 0
}

func (e *endpoint) AddHeader(packetBuffer *stack.PacketBuffer) {
    // add link-layer header
}

func (e *endpoint) ParseHeader(packetBuffer *stack.PacketBuffer) bool {
    // stack.WriteRawPacket
    // check if header is correct
    return true
}

func (e *endpoint) ARPHardwareType() header.ARPHardwareType {
    panic(ErrNotSupported)
}

func (e *endpoint) MTU() uint32 {
    return e.mtu
}

func (e *endpoint) SetMTU(mtu uint32) {
    e.mtu = mtu
}

func (e *endpoint) Capabilities() stack.LinkEndpointCapabilities {
    return stack.CapabilityLoopback |
        stack.CapabilitySaveRestore |
        stack.CapabilityDisconnectOk |
        stack.CapabilityResolutionRequired |
        stack.CapabilityTXChecksumOffload |
        stack.CapabilityRXChecksumOffload
}

func (e *endpoint) SetLinkAddress(addr tcpip.LinkAddress) {
    panic(ErrNotSupported)
}

// Mac address is required by ICMPv6
func (e *endpoint) LinkAddress() tcpip.LinkAddress {
    // https://stackoverflow.com/questions/21018729/generate-mac-address-in-go
    // mac[0] = (mac[0] | 2) & 0xfe // Set local bit, ensure unicast address
    return "\xFE\xFF\xFF\xFF\xFF\xFF"
}

func (e *endpoint) WritePackets(list stack.PacketBufferList) (n int, err tcpip.Error) {
    var bs []byte
    for _, pb := range list.AsSlice() {

        slices := pb.AsSlices()
        if len(slices) == 1 {
            bs = slices[0]
        } else {
            // Naming convention
            // AsXxx: no-copy
            // ToXxx: copy
            bs = pb.ToView().AsSlice()
        }

        nw, err := e.tun.Write(bs)
        n += nw
        if nil != err {
            slog.Error(err.Error())
            return n, &tcpip.ErrInvalidEndpointState{}
        } else if nw != pb.Size() {
            panic(io.ErrShortWrite)
        }
    }
    return
}

func (e *endpoint) Attach(dispatcher stack.NetworkDispatcher) {
    // Attach is called with a nil dispatcher when the endpoint's NIC is being removed.
    // See more docs of interface:
    // stack.NetworkLinkEndpoint
    if nil == dispatcher {
        e.stopGoroutine = make(chan any)
        e.tun.Close()
        return
    }

    go func(tun io.Reader) {
        for {
            v := buffer.NewViewSize(int(e.mtu))
            nr, err := tun.Read(v.AsSlice())
            if nil != err {
                // Once tun is closed, tun.Read() ends with error, the loop break.
                if nil != e.stopGoroutine {
                    close(e.stopGoroutine)
                    slog.Info("tun closed")
                } else {
                    slog.Error(err.Error())
                }
                break
            }

            v.CapLength(nr)

            pb := stack.NewPacketBuffer(stack.PacketBufferOptions{Payload: buffer.MakeWithView(v)})

            switch header.IPVersion(v.AsSlice()) {
            // 0x40 == v.AsSlice()[0]&0xf0
            case header.IPv4Version:
                dispatcher.DeliverNetworkPacket(ipv4.ProtocolNumber, pb)
            // 0x60 == v.AsSlice()[0]&0xf0
            case header.IPv6Version:
                dispatcher.DeliverNetworkPacket(ipv6.ProtocolNumber, pb)
            default:
                slog.Error(fmt.Sprintf("ProtocolNumber Unknown %X", v.AsSlice()[0]))
                // https://en.wikipedia.org/wiki/List_of_IP_Protocol_numbers
                // header.IPv6(v).TransportProtocol() == header.ICMPv6ProtocolNumber
            }

            pb.DecRef()
        }
    }(e.tun)
}

func (e *endpoint) IsAttached() bool {
    panic(ErrNotSupported)
}

func (e *endpoint) Wait() {
    <- e.stopGoroutine
}

func (e *endpoint) SetOnCloseAction(f func()) {
    if nil != f {
        e.closeAction = f
    }
}

func (e *endpoint) Close() {
    e.closeAction()
}
