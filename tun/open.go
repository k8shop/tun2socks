package tun

import (
    "io"
    "log/slog"

    "github.com/k8shop/tun2socks"
)

func open(handler tun2socks.TransportHandler) io.Closer {

    tunDev, err := OpenTunDevice(cfgTunName, cfgNetwork, cfgDNS)
    if nil != err {
        slog.Error(err.Error())
        return nil
    }

    return tun2socks.New(tunDev, handler)
}
