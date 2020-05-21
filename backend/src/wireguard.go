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

func generatePeerConfig(ip string, pubkey string, psk string) (config wgtypes.PeerConfig) {
	key, err := wgtypes.ParseKey(pubkey)
	check(err)

	ppsk, err := wgtypes.ParseKey(psk)
	check(err)

	allowedIPs = getAllowedIP(ip)

	config = wgtypes.PeerConfig{PublicKey: key,
		PresharedKey:      &ppsk,
		ReplaceAllowedIPs: true,
		AllowedIPs:        allowedIPs}

	log.Print(config)

	return config
}

func updateInterface(name string, port int, privkey string, peerList []wgtypes.PeerConfig) {
	wc, err := wgctrl.New()
	check(err)

	key, err := wgtypes.ParseKey(privkey)
	check(err)

	config := wgtypes.Config{PrivateKey: &key,
		ListenPort:   &port,
		ReplacePeers: true,
		Peers:        peerList}

	log.Print(peerList)

	err = wc.ConfigureDevice(name, config)
	check(err)

	devices, err := wc.Devices()
	check(err)
	log.Print(devices)
	for _, d := range devices {
		log.Print(d.Name)
		log.Print(d.Type)
		log.Print(d.PrivateKey)
		log.Print(d.PublicKey)
		log.Print(d.ListenPort)
		log.Print(d.FirewallMark)
		log.Print(d.Peers)
	}
}
