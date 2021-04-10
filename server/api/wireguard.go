package main

import (
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

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

// Takes a list of peer configs and applies the config to the server specified.
// Peers are not replaced, instead the peer configs indicate whether a peer
// should be removed or appended to the server. Rotating peers works by passing
// both the stale and new configs as part of the peer list, with the toRemove
// flag indicating what to do (see getPeerConfig).
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

// Generates WireGuard keys and returns them as strings.
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
