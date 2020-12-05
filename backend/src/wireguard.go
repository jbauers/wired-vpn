package main

import (
	"errors"
	"net"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func genKeys() (privateKey string, publicKey string, presharedKey string) {
	keyPair, err := wgtypes.GeneratePrivateKey()
	check(err)

	psk, err := wgtypes.GenerateKey()
	check(err)

	privateKey = keyPair.String()
	publicKey = keyPair.PublicKey().String()
	presharedKey = psk.String()

	return privateKey, publicKey, presharedKey
}

func getPeerConfig(ip string, publicKey string, presharedKey string, toRemove bool) (peerConfig wgtypes.PeerConfig) {
	pub, err := wgtypes.ParseKey(publicKey)
	check(err)

	psk, err := wgtypes.ParseKey(presharedKey)
	check(err)

	allowedIPs := getAllowedIP(ip)

	peerConfig = wgtypes.PeerConfig{
		PublicKey:         pub,
		PresharedKey:      &psk,
		Remove:            toRemove,
		AllowedIPs:        allowedIPs,
		ReplaceAllowedIPs: false,
	}

	return peerConfig
}

func updateInterface(server Peer, peerList []wgtypes.PeerConfig) error {
	wc, err := wgctrl.New()
	check(err)

	privateKey, err := wgtypes.ParseKey(server.PrivateKey)
	check(err)

	config := wgtypes.Config{
		PrivateKey:   &privateKey,
		ListenPort:   &server.Port,
		Peers:        peerList,
		ReplacePeers: false,
	}

	err = wc.ConfigureDevice(server.Interface, config)
	return err
}

func getAllowedIP(ip string) []net.IPNet {
	// The allowed IPv4 may only be a /32.
	_, ipnet, err := net.ParseCIDR("0.0.0.0/32")
	check(err)

	network := *ipnet
	network.IP = net.ParseIP(ip)

	return []net.IPNet{network}
}

func getAvailableIP(ips []string) (string, error) {
	ip, ipnet, err := net.ParseCIDR(serverCIDR)
	check(err)

	for stringInSlice(ip.String(), ips) {
		ip = iterIP(ip)
	}

	if !ipnet.Contains(ip) {
		return "", errors.New("Exhausted IP addresses!")
	}

	return ip.String(), nil
}

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

func stringInSlice(s string, list []string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
