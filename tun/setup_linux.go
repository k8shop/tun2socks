package tun

import (
    "fmt"
    "log"
    "os"
    "os/exec"
    "os/signal"
    "syscall"
)

func main(){
    flagParse()
    open()
    setup()
}

func setup() {

    // Linux 的 DNS 配置最终是以 /etc/resolv.conf 文件为准的
    // systemd-resolved、dnsmasq、resolvconf 等都会修改这个文件

    // options timeout:2 attempts:3 rotate single-request-reopen
    // http://man7.org/linux/man-pages/man5/resolv.conf.5.html

    // systemctl restart systemd-resolved.service
    // 将导致 /etc/resolv.conf 重新生成，类似 resolvconf -u

    // todo 非 systemd-resolved 的情况
    // todo 有这文件也不足以说明服务可用，WSL2 Ubuntu 就是这样
    if _, err := os.Stat("/etc/systemd/resolved.conf"); os.IsNotExist(err) {
        log.Fatalln("dns not managed by systemd-resolved")
    }

    tempFile, err := os.Create("/tmp/cnfix-tap-setup.sh")
    if nil != err {
        log.Fatalln("setup", err)
    }
    defer os.Remove(tempFile.Name())
    defer tempFile.Close()

    _, err = fmt.Fprintf(tempFile, `set -e
ip addr add %s/%s dev cnfix
ip link set dev cnfix up
ip route add default via %s metric 1
systemctl stop systemd-resolved.service
echo nameserver %s > /etc/resolv.conf
`, cfgIp, cfgMask, cfgGateway, cfgDNS)
    if nil != err {
        log.Fatalln("setup", err)
    }

    defer exec.Command("systemctl", "start", "systemd-resolved.service").Run()

    cmd := exec.Command("sh", tempFile.Name())
    out, err := cmd.Output()
    if nil != err {
        log.Fatalln("setup", err, out)
    }

    chSignal := make(chan os.Signal)
    signal.Notify(chSignal, os.Interrupt, syscall.SIGTERM)
    <-chSignal
    // os.Exit(0) 会导致 defer 得不到执行
}

// 2020/05/06 05:10:31 socks connect tcp 192.168.99.1:1080->169.254.169.254:80: unknown error host unreachable
// 2020/05/06 05:10:32 socks connect tcp 192.168.99.1:1080->169.254.169.254:80: unknown error host unreachable
// 2020/05/06 05:10:33 socks connect tcp 192.168.99.1:1080->169.254.169.254:80: unknown error host unreachable
// 2020/05/06 05:10:34 socks connect tcp 192.168.99.1:1080->169.254.169.254:80: unknown error host unreachable
// 2020/05/06 05:10:35 socks connect tcp 192.168.99.1:1080->169.254.169.254:80: unknown error host unreachable
// 2020/05/06 05:10:36 socks connect tcp 192.168.99.1:1080->169.254.169.254:80: unknown error host unreachable
