package tap

import (
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/k8shop/water"
)

// func isIPv4(ip net.IP) bool {
// 	if ip.To4() != nil {
// 		return true
// 	}
// 	return false
// }
//
// func isIPv6(ip net.IP) bool {
// 	// To16() also valid for ipv4, ensure it's not an ipv4 address
// 	if ip.To4() != nil {
// 		return false
// 	}
// 	if ip.To16() != nil {
// 		return true
// 	}
// 	return false
// }

func OpenTunDevice(name, subnet string, dnsServers string) (io.ReadWriteCloser, error) {
// func OpenTunDevice(name, addr, gw, mask string, dnsServers []string, persist bool) (io.ReadWriteCloser, error) {
	tunDev, err := water.New(water.Config{
		DeviceType: water.TUN,
	})
	if err != nil {
		return nil, err
	}
	name = tunDev.Name()

	ip, ipn, err := net.ParseCIDR(config.PlatformSpecificParams.Network)
	if nil != err {
		return nil, err
	}

	ip = ip.To4()
	ipNet := ipn.IP.To4()
	if nil == ip || nil == ipNet || 4 != len(ipn.Mask) {
		return nil, errors.New("ipv4 supported only")
	}

	var gateway net.IP = make([]byte, 4)
	copy(gateway, ip)
	gateway[3] = 1 + ip[3]%254

	// var params string
	// if isIPv4(ip) {
	// 	params = fmt.Sprintf("%s inet %s netmask %s %s", name, addr, mask, gw)
	// } else if isIPv6(ip) {
	// 	prefixlen, err := strconv.Atoi(mask)
	// 	if err != nil {
	// 		return nil, errors.New(fmt.Sprintf("parse IPv6 prefixlen failed: %v", err))
	// 	}
	// 	params = fmt.Sprintf("%s inet6 %s/%d", name, addr, prefixlen)
	// } else {
	// 	return nil, errors.New("invalid IP address")
	// }

	out, err := exec.Command("ifconfig", name, "inet", ip.String(), "netmask", ipn.Mask.String(), gateway.String()).Output()
	// out, err := exec.Command("ifconfig", strings.Split(params, " ")...).Output()
	if err != nil {
		if len(out) != 0 {
			return nil, errors.New(fmt.Sprintf("%v, output: %s", err, out))
		}
		return nil, err
	}
	return tunDev, nil
}
