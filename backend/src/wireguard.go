package main

import (
	"log"

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

func updateInterface(server Peer, peerList []wgtypes.PeerConfig) {
	wc, err := wgctrl.New()
	check(err)

	key, err := wgtypes.ParseKey(server.PrivateKey)
	check(err)

	config := wgtypes.Config{PrivateKey: &key,
		ListenPort:   &server.Port,
		ReplacePeers: false,
		Peers:        peerList}

	err = wc.ConfigureDevice(server.Interface, config)
	check(err)

	log.Print("UPDATED: " + server.Interface)
}
