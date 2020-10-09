package main

import (
	"net"
)

func incrementIP(currIP string) (newIP string) {
	ip := net.ParseIP(currIP)
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
	return ip.String()
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
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
