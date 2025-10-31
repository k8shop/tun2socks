package tap

import (
    "io"
    "log"
    "net"
    "sync"
    "time"

    "cnfix/base"

    "github.com/k8shop/tun2socks"
)

var Rate RateLimit

type RateLimit struct {
    sync.Mutex
    count   int
    timeCut time.Time
}

func (rl *RateLimit) limit() {
    rl.Lock()
    defer rl.Unlock()

    if rl.count > 50 {
        // 开启转发会使该适配器上的 tcp/udp local addr bind 无效，引起网络请求的死循环
        // Set-NetIPInterface [-ifindex interface index] -Forwarding Disabled
        // netsh interface ip set interface interface=MSI forwarding=disabled
        // 创建Hyper-V虚拟交换机(外部网络)后，物理网卡的IPv4/Ipv6协议自动被禁用(可以再手动启用但是其中一个得改Mac地址)
        // 手动勾选启用物理网卡的协议，虚拟网卡有性能损耗，宽度测速从120M多降到80M多，还可以创建多个连接到它的适配器
        // Add-VMNetworkAdapter -Switch MSI -VM Name(添加到虚拟机)/-ManagementOS(允许操作系统共享此网络适配器,即添加到物理机)
        // Hyper-V虚拟交换机(相当于VirtualBox桥接网卡) 实现一个物理网卡创建多个网络适配器
        // 可以再多创建一个适配器，专门用于开启路由转发到虚拟机，但使用虚拟交换机更简洁
        // https://superuser.com/questions/1396033/windows-creating-virtual-interface-over-a-single-physical-interface
        log.Fatal("Too many connection requests! It seems Route run into dead loop. Please try binding a local address for socks5 agent.")
        // Set-VMNetworkAdapterVlan -ManagementOS -VMNetworkAdapterName Bridge -Access -VlanId 76
        // 使用 VLAN 有隔离的作用，安全、减轻广播风暴 (网络两端配置同样的ID即可通信)
        // sudo dhclient eth2
    }

    now := time.Now()
    if now.Before(rl.timeCut) {
        rl.count++
    } else {
        rl.count = 1
        rl.timeCut = now.Add(time.Millisecond * 50)
    }
}

type TransHandler struct {

}

func (th *TransHandler) TcpHandle(conn net.Conn) error {
    Rate.limit()
    addr := conn.LocalAddr().(*net.TCPAddr)
    // dialer, err := proxy.SOCKS5("tcp", cfgSocks5, nil, nil)
    // conn2, err := dialer.Dial("tcp", conn.RemoteAddr().String())
    conn2 := base.Cnfix.Dial("tcp", &base.Addr{
        IP:   addr.IP,
        Port: uint16(addr.Port),
    })

    if nil == conn2 {
        return io.ErrClosedPipe
    }

    base.ReadEachOther(conn, conn2, 0)

    return nil
}

func (th *TransHandler) UdpHandle(conn net.Conn) error {
    // Win10 App Store 好像对 UDP 有依赖，或者是它的部分服务拒绝使用 VPN 连接
    // 以前用别的 VPN 也遇到这问题，有些页面(如登录)显示 Internet 未连接
    // todo 仅过滤无聊广播包，允许正常 udp 数据
    // 考虑到不设 bind 可能会引起路由死循环
    // 必须保证 dns 要么是 127.0.0.1 要么是和本机网卡直连（不需要经过路由）
    addr := conn.LocalAddr().(*net.UDPAddr)
    if addr.Port == 53 {
        addr.IP = net.ParseIP(cfgDNS).To4()
    }

    Rate.limit()

    conn2 := base.Cnfix.Dial("udp", &base.Addr{
        IP:   addr.IP,
        Port: uint16(addr.Port),
    })

    if nil == conn2 {
        return io.ErrClosedPipe
    }

    base.ReadEachOther(conn2, conn, time.Minute)

    return nil
}

func open() io.Closer {

    tunDev, err := OpenTunDevice(cfgTunName, cfgNetwork, cfgDNS)
    if nil != err {
        log.Println(err)
        return nil
    }

    return tun2socks.New(tunDev, &TransHandler{})
}
