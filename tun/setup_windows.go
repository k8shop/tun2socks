package tun

import (
    "bytes"
    "fmt"
    "io"
    "log/slog"

    "github.com/k8shop/systool"
    "github.com/k8shop/tun2socks"
)

func Open(handler tun2socks.TransportHandler) (t2s io.Closer) {

    flagParse()
    // check()
    t2s = open(handler)
    setup()

    return
}

/*
func addAdapter() {
    dir := filepath.Dir(os.Args[0])
    systool.CmdOut(
        "powershell",
        fmt.Sprintf(
            `Start-Process -Verb RunAs -FilePath '%s' -ArgumentList 'install "%s" tap0901' -Wait -WindowStyle Hidden`,
            dir+"/res/tap/bin/tapinstall.exe",
            dir+"/res/tap/driver/OemVista.inf",
        ),
    )
}

func findAdapter() string {
    // 每个命令默认的 Format 方式不一样，展示出来的属性就不一样
    // 很多隐藏的有用属性，需要改变 Format 方式才能看到
    // date | Format-List
    // Get-NetAdapter | Format-Table
    // Get-NetAdapter | Format-List
    // Get-NetAdapter | Format-Custom
    // Get-NetAdapter | Select-Object -Property ComponentID
    out := systool.CmdOut(
        "powershell",
        `Get-NetAdapter -InterfaceDescription "TAP-Windows Adapter V*" | Format-Table Name -HideTableHeaders`,
    )

    var name string
    br := bufio.NewReader(bytes.NewBuffer(out))
    for {
        line, _, err := br.ReadLine()
        if nil != err {
            if io.EOF == err {
                break
            }
            log.Fatal(err)
        }
        line = bytes.TrimSpace(line)
        if 0 == len(line) {
            continue
        }
        name = string(line)
        if name == cfgTunName {
            break
        }
    }
    return name
}

func check() {

    name := findAdapter()
    if len(name) == 0 {
        addAdapter()
        name = findAdapter()
        if len(name) == 0 {
            log.Fatalln("TAP-Windows Adapter install failed")
        }
    }

    if name != cfgTunName {
        // file, err := os.Create(os.TempDir() + "/cnfix-tap-rename.bat")
        // defer os.Remove(file.Name())
        // file.WriteString("netsh interface set interface name=\"")
        // // 使用 []byte 可以绕过编码转换问题
        // file.Write(names[0])
        // file.WriteString("\" newname=cnfix")
        // file.Close()
        // systool.CmdOut(
        //     file.Name(),
        //     // "powershell",
        //     // fmt.Sprintf("Start-Process -Verb RunAs -FilePath '%s' -Wait -WindowStyle Hidden", file.Name()),
        //     // 奇怪，执行 bat 本身不需要超管权限，使用 Start-Process 不加 -Verb RunAs 则失败
        // )

        systool.CmdOut(
            "cmd",
            "/c",
            // 参数包含空格，涉及双引号语法解析的，要借助 cmd 才可以
            fmt.Sprintf(`netsh interface set interface name="%s" newname=%s`, name, cfgTunName),
        )
    }

    dhcpEnabled := exec.Command(
        "powershell",
        "-Command",
        // Get-NetAdapter -Name CnfixTun | Get-NetIPAddress | where AddressFamily -eq IPv4
        "Get-NetAdapter -Name "+cfgTunName+" | Get-NetIPInterface -AddressFamily IPv4 -Dhcp Enabled",
    ).Run()

    // dnsMatch := exec.Command(
    //     "powershell",
    //     "-Command",
    //     // ? { 语句块 } 用于筛选？
    //     // @ { 语句块 } 用于？
    //     // % { 语句块 } 用于格式化输出
    //     // { 1,2,3 } 可包含多个子块，每个子块也是返回值
    //     // { 变量，常量，if else，字符串拼接，还可以抛出错误 }
    //     "Get-NetAdapter -Name "+cfgTunName+" | Get-DnsClientServerAddress -AddressFamily IPv4 | where ServerAddresses -Contains "+cfgDNS+" | measure | % { if($_.Count -ne 1){throw 'not found'} }",
    // ).Run()
    // fmt.Println(dnsMatch)

    // Tap-Windows 配置 IP/DNS 需要 DHCP 支持
    if nil != dhcpEnabled {
        systool.CmdBat(
            true,
            fmt.Sprintf("netsh interface ipv4 set address name=%s source=dhcp store=persistent", cfgTunName),
            fmt.Sprintf("netsh interface ipv4 set dnsservers name=%s source=dhcp register=none validate=no", cfgTunName),
        )
    }
}
*/

func setup() {

    out := systool.CmdOut(
        "powershell",
        fmt.Sprintf("Find-NetRoute -RemoteIPAddress %s | Format-Table IPAddress -HideTableHeaders", cfgGateway),
    )

    if cfgIp != string(bytes.TrimSpace(out)) {
        slog.Error("Wintun Adapter is Unavailable", "network", cfgNetwork)
        return
    }

    out = systool.CmdOut(
        "powershell",
        "Find-NetRoute -RemoteIPAddress 8.8.8.8 | Format-Table NextHop,InterfaceMetric -HideTableHeaders",
    )

    fields := bytes.Fields(out)

    if len(fields) != 2 {
        slog.Warn("None default routes found.")
    } else {
        // Metric 值的数字字符串
        // 懒得转换，直接判断是几位数即可
        // if cfgGateway == string(fields[0]) && len(fields[1]) == 1 {
        //     return
        // }
    }

    // _, err = fmt.Fprintf(file, "netsh interface ipv4 set address name=%s source=static address=%s gateway=%s gwmetric=1 store=active \r\n", cfgTunName, cfgNetwork, cfgGateway)
    // _, err = fmt.Fprintf(file, "netsh interface ipv4 set dnsservers name=%s source=static address=%s register=none validate=no \r\n", cfgTunName, cfgDNS)
    // _, err = fmt.Fprintf(file, "route -p delete 0.0.0.0 %s \r\n", cfgGateway)
    // _, err = file.Write("if %errorlevel% neq 0 exit /b %errorlevel% \r\n")

    systool.CmdBat(
        true,
        // 添加路由的两个命令，功能不一样，不可以混用
        // route add x.x.x.x/x gateway 网关转发
        // route add x.x.x.x/x localIP 在链路上
        fmt.Sprintf("route add 0.0.0.0/0 %s METRIC 1", cfgGateway),
        fmt.Sprintf("route -p add 0.0.0.0/0 %s METRIC 1", cfgGateway),
        fmt.Sprintf("netsh int ip set interface %s metric=0", cfgTunName),
        fmt.Sprintf("netsh int ip set interface %s metric=0 store=persistent", cfgTunName),
    )

    // setup ipv6
    // route -6 add ::/0 fec0::7
    // ip r get 2002:1b7a:3a9a::1
    // nc 2002:1b7a:3a9a::1 8080
    // curl [2002:1b7a:3a9a::1]
    // curl ipv6.jp.roxashome.com
    // nslookup qq.com ::1
    // Remove-NetIPAddress -InterfaceAlias CnfixTun -AddressFamily IPv6 -Confirm:$false
    // Remove-NetRoute -InterfaceAlias CnfixTun -AddressFamily IPv6 -Confirm:$false
    // todo 那个自动地址不能删的
    // New-NetIPAddress -InterfaceAlias CnfixTun -IPAddress fc00:7:6:5:4:3:2:1 -PrefixLength 112 -DefaultGateway fc00:7:6:5:4:3:2:1007
    // Set-NetIPInterface -InterfaceAlias CnfixTun -AddressFamily IPv6 -AutomaticMetric Disabled -InterfaceMetric 1
    // Set-NetRoute -InterfaceAlias CnfixTun -AddressFamily IPv6 -DestinationPrefix ::/0 -RouteMetric 1
    // Set-DnsClientServerAddress -InterfaceAlias CnfixTun -ServerAddresses ("::1")
}

// 读写注册表
// Get-ItemProperty -path "hkcu:Software\Microsoft\Windows\CurrentVersion\Internet Settings" -name ProxyServer
// Set-ItemProperty -path "hkcu:Software\Microsoft\Windows\CurrentVersion\Internet Settings" -name ProxyServer -value "127.0.0.1:1080" -type string
