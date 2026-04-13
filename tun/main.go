package tun

import (
    "log"
    "net"
)

var cfgTunName = "CnfixTun"

var cfgDNS = "127.0.0.1"
var cfgSocks5 = "127.0.0.1:1080"
var cfgNetwork = "192.88.99.66/24"

var cfgIp string
var cfgMask string
var cfgGateway string

// init 是个特殊的函数名，和 main 一样有特殊的用途
// 同一个包内，可以有多个 init 函数，由编辑器自己处理，不允许用户调用
func flagParse() {
    // levelog.Level = levelog.LevelSilly
    // config.DialBind = "192.168.0.150"

    // todo 类似 helm 那样传递 --a.b.c=value -a.b[0].c=value
    // flag.StringVar(&cfgDNS, "dns", "127.0.0.1", "127.0.0.1")
    // flag.StringVar(&cfgSocks5, "socks5", "127.0.0.1:1080", "127.0.0.1:1080")
    // flag.StringVar(&cfgNetwork, "network", "192.88.99.66/24", "192.88.99.66/24")
    // flag.Parse()

    ip, ipnet, err := net.ParseCIDR(cfgNetwork)
    if nil != err {
        log.Fatalln(err)
    }
    ip = ip.To4()
    // todo if nil==ip.To4() 处理 ipv6
    cfgIp = ip.String()
    cfgMask = net.IP(ipnet.Mask).String()

    ip[3] = 1 + ip[3]%254
    cfgGateway = ip.String()

    // cmd := exec.Command("netsh", "interface", "ipv4", "set", "interface", "本地连接", "metric=1")
    // http://www.xntutor.com/powershell/powershell-start-process.html
    // todo 合并到一个 bat 临时文件中，避免多次弹出授权窗口
    // todo 先查询再决定是否需要修改，避免每次都弹出授权窗口
    // todo -ExecutionPolicy bypass
    // todo tapinstall.exe help find
    // cmd := exec.Command(
    //     "powershell",
    //     "Start-Process -Verb runAs -FilePath route -ArgumentList 'add 0.0.0.0/0 192.168.200.254'",
    // )
    // fmt.Println(cmd.Run())
    // cmd = exec.Command(
    //     "powershell",
    //     "Start-Process -Verb runAs -FilePath netsh -ArgumentList 'interface ipv4 set interface 本地连接 metric=1'",
    // )
    // fmt.Println(cmd.Run())
    // 通过设置网卡的网关，让系统自动添加路由？算了，还不如用前面的方法，还更直接
    // netsh interface ipv4 set dnsservers name=本地连接 source=static address=127.0.0.1 register=none validate=no
    // netsh interface ipv4 set address name="本地连接" source=static address=192.168.200.100/24 gateway=192.168.200.254 gwmetric=1 store=active

    // 69.63.190.26 ???? 谷歌的 ip 为什么变成了 Facebook
    // 是 dns 服务器返回的，但这是本地解析结果，不应该传到远程服务端，使用任何 DNS 都是（谷歌竟然返回错误的证书）
    // 这种情况下，只能是把 dns 流量传到远程去解析 + 本地缓存，而不是换个 DNS 就可以
    // 是放到远程去解析，获得远程解析的结果，不是让远程代理请求 DNS
    // 另一种思路是，把域名和ip的关联保存下来，真正走代理的时候使用域名作为地址而不是IP，这就需要解包，对，就是这样，使用假IP回复
}
