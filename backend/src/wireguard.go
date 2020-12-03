package main

import (
	"net"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func genKeys() (privkey string, pubkey string, psk string) {
	k, err := wgtypes.GeneratePrivateKey()
	check(err)

	sk, err := wgtypes.GenerateKey()
	check(err)

	privkey = k.String()
	pubkey = k.PublicKey().String()

	psk = sk.String()

	return privkey, pubkey, psk
}

func getPeerConfig(ip string, pubkey string, psk string, toRemove bool) (config wgtypes.PeerConfig) {
	key, err := wgtypes.ParseKey(pubkey)
	check(err)

	ppsk, err := wgtypes.ParseKey(psk)
	check(err)

	allowedIPs := getAllowedIP(ip)

	config = wgtypes.PeerConfig{PublicKey: key,
		Remove:            toRemove,
		PresharedKey:      &ppsk,
		ReplaceAllowedIPs: false,
		AllowedIPs:        allowedIPs}

	return config
}

func updateInterface(server Peer, peerList []wgtypes.PeerConfig) error {
	wc, err := wgctrl.New()
	check(err)

	key, err := wgtypes.ParseKey(server.PrivateKey)
	check(err)

	config := wgtypes.Config{PrivateKey: &key,
		ListenPort:   &server.Port,
		ReplacePeers: false,
		Peers:        peerList}

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


func getAvailableIP(ips []string) (ip string) {
	ip = serverIP
	for stringInSlice(ip, ips) {
		ip = iterIP(ip)
	}
	return ip
}

// FIXME: Filter out broadcast, ensure within CIDR.
func iterIP(currIP string) (newIP string) {
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

