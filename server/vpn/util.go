package main

import (
	"log"
	"net"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// Takes a list of peer configs and applies the config to the server specified.
// Peers are not replaced, instead the peer configs indicate whether a peer
// should be removed or appended to the server. Rotating peers works by passing
// both the stale and new configs as part of the peer list, with the toRemove
// flag indicating what to do (see getPeerConfig).
func updateInterface(privateKey wgtypes.Key, peerList []wgtypes.PeerConfig) error {
	wc, err := wgctrl.New()
	check(err)

	port := *wgPort

	config := wgtypes.Config{
		PrivateKey:   &privateKey,
		ListenPort:   &port,
		Peers:        peerList,
		ReplacePeers: false,
	}

	err = wc.ConfigureDevice(*wgInterface, config)
	return err
}

// Takes the IP, public key, pre-shared key as strings, and a bool whether the
// peer should be removed or added to the interface, and returns the wgtypes
// peer config for this peer. This config is then applied as part of
// updateInterface, which expects a list of these peer configs.
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

func check(e error) {
	if e != nil {
		log.Panic(e)
	}
}
