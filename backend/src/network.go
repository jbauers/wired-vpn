package main

import (
	"net"
)

// Get all valid hosts in a subnet.
func hosts(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); iterateHosts(ip) {
		ips = append(ips, ip.String())
	}

	// Remove network address and broadcast address.
	lenIPs := len(ips)
	switch {
	case lenIPs < 2:
		return ips, nil

	default:
		return ips[1 : len(ips)-1], nil
	}
}

func iterateHosts(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// Get the allowed IP. This may only be a /32 (for IPv4).
func getAllowedIP(ip string) []net.IPNet {
	_, ipnet, err := net.ParseCIDR("0.0.0.0/32")
	check(err)

	network := *ipnet
	network.IP = net.ParseIP(ip)

	allowedIP := []net.IPNet{network}

	return allowedIP
}
