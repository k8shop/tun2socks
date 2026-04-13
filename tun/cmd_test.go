package tun

import (
    "bufio"
    "bytes"
    "io"
    "log"
    "os/exec"
    "strings"
    "testing"
)

func TestDevs(t *testing.T) {
    cmd := exec.Command(
        "powershell",
        "Get-NetAdapter | Format-Table InterfaceDescription -HideTableHeaders",
    )

    out, err := cmd.Output()
    if nil != err {
        log.Fatalln(err, out)
    }

    var descriptions []string
    br := bufio.NewReader(bytes.NewBuffer(out))
    for {
        line, _, err := br.ReadLine()
        if nil != err {
            if io.EOF == err {
                break
            }
            log.Fatalln("Get-NetAdapter", err)
        }
        description := string(bytes.TrimSpace(line))
        if strings.HasPrefix(description, "TAP-Windows Adapter V9") {
            descriptions = append(descriptions, description)
        }
    }

    if len(descriptions) == 0 {
        log.Fatalln("TAP-Windows Adapter not found")
    }
}
