package sd

import (
	"net"
	"os"
	"strings"
)

func serviceHost() string {
	if address := strings.TrimSpace(os.Getenv("SERVICE_HOST")); address != "" {
		return address
	}

	if address := preferredIPv4("eth0", "em1"); address != "" {
		return address
	}

	if address := firstIPv4(); address != "" {
		return address
	}

	return "127.0.0.1"
}

func preferredIPv4(names ...string) string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range ifaces {
		for _, name := range names {
			if name == iface.Name {
				if address := interfaceIPv4(iface); address != "" {
					return address
				}
				break
			}
		}
	}
	return ""
}

func interfaceIPv4(iface net.Interface) string {
	if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
		return ""
	}

	if strings.HasPrefix(iface.Name, "docker") || strings.HasPrefix(iface.Name, "w-") {
		return ""
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		var ip net.IP
		switch address := addr.(type) {
		case *net.IPNet:
			ip = address.IP
		case *net.IPAddr:
			ip = address.IP
		}

		if ip == nil || ip.IsLoopback() {
			continue
		}
		if ip = ip.To4(); ip != nil {
			return ip.String()
		}
	}
	return ""
}

func firstIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if address := interfaceIPv4(iface); address != "" {
			return address
		}
	}
	return ""
}
