// +build linux

package main

import (
	"fmt"
	"net"
	"strconv"

	"github.com/milosgajdos/tenus"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// This is good enough on Linux. Note that a WireGuard interface named "wired0"
// has to be present, added with "ip link add dev wired0 type wireguard". It
// does not need to be further configured though, this will assign an IP and
// configure the device.

func configureInterface(wgInterface string, peer Peer) (error, tenus.Linker) {
	// FIXME: Just plowing through here, 'cus it's a lot
	// simpler and usually just works. Add error handling!

	// Configure Linux networking. Tenus is used as it works
	// with CAP_NET_ADMIN permissions, and we can avoid
	// running the CLI as root, which causes other issues.
	wired, _ := tenus.NewLinkFrom(wgInterface)
	tunHostIp, tunHostIpNet, _ := net.ParseCIDR(peer.IP)
	wired.SetLinkIp(tunHostIp, tunHostIpNet)

	// This resolves the hostname and returns the first IP address.
	// For our use case, this is ok. If we expect more than 1 IP for
	// this DNS entry, we would need to account for it.
	wgEndpoint, _ := net.ResolveUDPAddr("udp", peer.Endpoint+":"+strconv.Itoa(peer.Port))
	wgPublicKey, _ := wgtypes.ParseKey(peer.PublicKey)
	wgPSK, _ := wgtypes.ParseKey(peer.PSK)

	// Get the server's peer config.
	var peerList []wgtypes.PeerConfig
	peerConfig := wgtypes.PeerConfig{
		Endpoint:          wgEndpoint,
		PublicKey:         wgPublicKey,
		PresharedKey:      &wgPSK,
		Remove:            false,
		ReplaceAllowedIPs: true,
		AllowedIPs:        getAllowedIP(peer.AllowedIPs),
	}
	peerList = append(peerList, peerConfig)

	// Apply the server config.
	config := wgtypes.Config{
		Peers:        peerList,
		ReplacePeers: true,
	}

	wc, _ := wgctrl.New()
	wc.ConfigureDevice(wgInterface, config)

	// FIXME: Debugging.
	devices, _ := wc.Devices()
	for k, v := range devices {
		fmt.Println(k)
		fmt.Println(v)
	}

	return nil, wired
}

func setInterfaceUp(wired tenus.Linker) string {
	err := wired.SetLinkUp()
	if err != nil {
		return err.Error()
	}
	return "wired" // FIXME
}
