package tap

import (
    "io"

    "github.com/k8shop/water"
)

func OpenTunDevice(name, addr, gw, mask string, dnsServers []string, persist bool) (io.ReadWriteCloser, error) {
    cfg := water.Config{
        DeviceType: water.TUN,
        PlatformSpecificParams: water.PlatformSpecificParams{
            Name: name,
            // MultiQueue: true,
        },
    }

    return water.New(cfg)
}
