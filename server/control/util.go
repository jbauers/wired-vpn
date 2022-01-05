package main

import (
	"errors"
	"log"
	"net"
)

// Returns the WireGuard interface for a given group, or an empty string
// if the group can't be found.
func getGroupInterface(peers []Peer, group string) string {
	for _, p := range peers {
		for _, g := range p.Groups {
			if group == g {
				return p.Interface
			}
		}
	}
	return ""
}

// Accepts an IP and a CIDR as strings and returns
// a string merging the two.
func getIpCidrString(ip string, cidr string) string {
	_, ipnet, err := net.ParseCIDR(cidr)
	check(err)

	network := *ipnet
	network.IP = net.ParseIP(ip)

	return network.String()
}

// This function accepts a list of strings and returns the next IP not in this
// list. If we overflow the server CIDR, an error is returned.
func getAvailableIP(ips []string, cidr string) (string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	check(err)

	for stringInSlice(ip.String(), ips) {
		ip = iterIP(ip)
	}

	if !ipnet.Contains(ip) {
		return "", errors.New("Exhausted IP addresses.")
	}

	return ip.String(), nil
}

// Increments an IP, skipping broadcast addresses.
func iterIP(ip net.IP) net.IP {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] == 255 {
			ip[i-1]++
			ip[i] = 1
		}
		if ip[i] > 0 {
			break
		}
	}
	return ip
}

// Returns true if string in slice.
func stringInSlice(s string, list []string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// Panic on error.
func check(e error) {
	if e != nil {
		log.Panic(e)
	}
}
