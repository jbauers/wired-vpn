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

// The allowed IPv4 for clients to be added to the server may only be a /32,
// but wgctrl expects a list. This function returns the list with a single
// entry from an IP string.
func getAllowedIP(ip string) []net.IPNet {
	_, ipnet, err := net.ParseCIDR("0.0.0.0/32")
	check(err)

	network := *ipnet
	network.IP = net.ParseIP(ip)

	return []net.IPNet{network}
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

// Log server info.
func printServerInfo(servers Servers) {
	for _, server := range servers.Peers {
		log.Printf("---------------------- Backend ready -----------------------")
		log.Printf(" Interface:  %s", server.Interface)
		log.Printf(" Network:    %s", server.CIDR)
		log.Printf(" Endpoint:   %s", server.Endpoint)
		log.Printf(" Port:       %d", server.Port)
		log.Printf(" PublicKey:  %s", server.PublicKey)
		log.Printf(" DNS:        %s", server.DNS)
		log.Printf(" Groups:     %s", server.Groups)
		log.Printf(" AllowedIPs: %s", server.AllowedIPs)
		log.Printf("------------------------------------------------------------")
	}
}
