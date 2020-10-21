package main

import (
	"net"
)

// FIXME: Filter out broadcast, ensure within CIDR.
func incrementIP(currIP string) (newIP string) {
	ip := net.ParseIP(currIP)
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] > 0 {
			break
		}
	}

	return ip.String()
}

func stringInSlice(s string, list []string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}

	return false
}

func getAllowedIP(ip string) []net.IPNet {
	// The allowed IPv4 may only be a /32.
	_, ipnet, err := net.ParseCIDR("0.0.0.0/32")
	check(err)

	network := *ipnet
	network.IP = net.ParseIP(ip)

	return []net.IPNet{network}
}
