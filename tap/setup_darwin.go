package tap

import (
    "log"
    "os"
    "os/exec"
    "os/signal"
    "regexp"
    "strings"
    "syscall"
    "time"
)

func Open() (t2s io.Closer) {

    flagParse()
    check()
    t2s = open()
    setup()

    return
}

func CmdOut(bin string, args ...interface{}) []byte {
    var strs []string
    for _, arg := range args {
        switch value := arg.(type) {
        case string:
            strs = append(strs, value)
        case []string:
            strs = append(strs, value...)
        default:
            log.Fatalln("CmdOut", "not string argument")
        }
    }
    cmd := exec.Command(bin, strs...)
    out, err := cmd.Output()
    if nil != err {
        log.Fatalln(err, out)
    }
    return out
}

func check() {
    // 检查是否另一进程正在运行
}

func setup() {

    // 正常来说，上一次进程退出之前就删除了
    if nil == exec.Command("networksetup", "-removenetworkservice", "CnfixTemporary").Run() {
        log.Println("wait for removenetworkservice")
        time.Sleep(9 * time.Second)
    }

    out := CmdOut("networksetup", "-listnetworkserviceorder")
    var exp = regexp.MustCompile("(?im)^\\((\\*|\\d+)\\) (.+)\\n\\(Hardware Port: .+, Device: (\\w+)\\)")
    groups := exp.FindAllSubmatch(out, 255)
    if len(groups) == 0 || groups[0][1][0] == '*' {
        log.Fatalln("No network service available")
    }

    var ethernets []string
    for _, group := range groups {
        ethernets = append(ethernets, string(group[2]))
    }

    CmdOut("networksetup", "-duplicatenetworkservice", ethernets[0], "CnfixTemporary")
    defer CmdOut("networksetup", "-removenetworkservice", "CnfixTemporary")
    // if len(cfgDNS) > 0 { }
    CmdOut("networksetup", "-setdnsservers", "CnfixTemporary", cfgDNS)
    socks5info := strings.Split(cfgSocks5, ":")
    CmdOut("networksetup", "-setsocksfirewallproxy", "CnfixTemporary", socks5info[0], socks5info[1])
    CmdOut("networksetup", "-ordernetworkservices", "CnfixTemporary", ethernets)

    // var ip string
    var router string
    timeEnd := time.Now().Add(time.Second * 9)
    exp = regexp.MustCompile("(?im)^IP address: ([.\\d]+)\n.+\nRouter: ([.\\d]+)")
    for {
        group := exp.FindSubmatch(CmdOut("networksetup", "-getinfo", "CnfixTemporary"))
        if nil != group {
            // ip = string(group[1])
            router = string(group[2])
            // fmt.Println(ip, router)
            break
        }

        if time.Now().Before(timeEnd) {
            log.Println("wait for ordernetworkservices")
            time.Sleep(500 * time.Millisecond)
        } else {
            log.Fatalln("failed to ordernetworkservices")
        }
    }

    // 重复删除或者重复添加都是返回 0 不用担心
    CmdOut("route", "delete", "default", router)
    CmdOut("route", "add", "default", cfgGateway)
    CmdOut("route", "add", "default", router, "-ifscope", string(groups[0][3]))
    defer CmdOut("route", "delete", "default", "-ifscope", string(groups[0][3]))

    chSignal := make(chan os.Signal)
    signal.Notify(chSignal, os.Interrupt, syscall.SIGTERM)
    <-chSignal
}

// 这是 GUI 问题，如果桌面已退出，用 SSH 登录在后台启动时会出现这个提示
// _RegisterApplication(), FAILED TO establish the default connection to the WindowServer, _CGSDefaultConnection() is NULL.

// sudo systemsetup -setremotelogin on
//
// launchctl list | grep virtualbox
//
// less /var/log/system.log
//
// # 0/1 + 128/1 == default
// sudo route add 0/1 192.168.200.254
// sudo route add 128/1 192.168.200.254
//
// # 224 to 239
// # 224.0.0.0/4

// 列表中第一个是出口网卡
// networksetup -listnetworkserviceorder
// networksetup -listallnetworkservices
// 保存原 DNS
// networksetup -getdnsservers Ethernet

// DNS 无效会导致 route 命令很慢，route -n 或许没问题
// 修改 DNS 附带清除缓存效果，是最简单的办法
// sudo networksetup -setdnsservers Ethernet 192.168.99.1

// networksetup -getinfo Ethernet

// Mac不用跃点来作为路由优先级，而是用它 network service 自己的排序
// 默认路由只能有一个
// 带 -ifscope 添加的默认路由可以有多个，但不是像 Linux 那样的默认路由
// 只在时特别指定时用到，比如 nc -s ，net.DialTcp(,laddr,)
// sudo route add default 10.0.2.2
// sudo route add default 10.0.2.2 -ifscope en0
// sudo route delete default 10.0.2.2 -ifscope en0

// 重启网卡也会自动恢复默认路由，临时添加的路由也会自动消失
// sudo networksetup -setnetworkserviceenabled Ethernet off
// sudo networksetup -setnetworkserviceenabled Ethernet on

// sudo networksetup -duplicatenetworkservice Ethernet CnfixTemporary
// sudo networksetup -setsocksfirewallproxy CnfixTemporary 127.0.0.1 1080
// 在副本上修改 DNS 不影响原来设置
// sudo networksetup -setdnsservers CnfixTemporary 127.0.0.1
// sudo networksetup -ordernetworkservices CnfixTemporary Ethernet "Ethernet Adaptor (en1)"
// networksetup -listnetworkserviceorder

// 禁用掉下次还可以用，顺序也不会变
// 但还是删掉每次重建吧，难于维护
// networksetup -removenetworkservice CnfixTemporary
// sudo networksetup -setnetworkserviceenabled CnfixTemporary off

// networksetup -h | grep list
// networksetup -h | grep get

// networksetup -getadditionalroutes Ethernet
