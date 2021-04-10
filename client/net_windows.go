// +build windows

package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"syscall"

	"golang.zx2c4.com/wireguard/windows/conf"
	//	"golang.zx2c4.com/wireguard/windows/manager"
)

// FIXME: This is an attempt of getting things to work on Windows.
// Currently this saves the config to disk, and it can be applied
// semi-successfully with the wg command. Unsurprisingly that may
// conflict with the official WireGuard app, which would do the
// real magic. WireGuard provides several ways to interact with
// it, not sure what's best - but it should be simple.
// - https://github.com/mullvad/mullvadvpn-app
// - https://github.com/WireGuard/wireguard-windows

func ipStringToCidr(ip string) []conf.IPCidr {
	var addresses []conf.IPCidr
	allowedIPslice := strings.Split(ip, "/")
	allowedIP := conf.IPCidr{
		IP:   net.ParseIP(allowedIPslice[0]),
		Cidr: uint8(24), // FIXME
	}
	addresses = append(addresses, allowedIP)
	return addresses
}

func configureInterface(wgInterface string, peer Peer) (error, conf.Config) {
	ok := AttachConsole(ATTACH_PARENT_PROCESS)
	if ok {
		fmt.Println("Okay, attached")
	}

	// Configure Windows networking.
	var config conf.Config

	privK, _ := conf.NewPrivateKeyFromString(peer.PrivateKey)
	wgPrivateKey := *privK

	var dns []net.IP
	dns = append(dns, net.ParseIP(peer.DNS))

	addresses := ipStringToCidr(peer.AllowedIPs)

	iface := conf.Interface{
		PrivateKey: wgPrivateKey,
		Addresses:  addresses,
	}

	pub, _ := conf.NewPrivateKeyFromString(peer.PublicKey)
	wgPublicKey := *pub

	psk, _ := conf.NewPrivateKeyFromString(peer.PSK)
	wgPSK := *psk

	wgEndpoint := conf.Endpoint{
		Host: peer.Endpoint,
		Port: uint16(peer.Port),
	}

	allowedIPs := ipStringToCidr("10.0.0.0/8")

	var peerList []conf.Peer
	peerConfig := conf.Peer{
		Endpoint:     wgEndpoint,
		PublicKey:    wgPublicKey,
		PresharedKey: wgPSK,
		AllowedIPs:   allowedIPs,
	}
	peerList = append(peerList, peerConfig)

	config.Name = wgInterface
	config.Interface = iface
	config.Peers = peerList

	bytes := []byte(config.ToWgQuick())
	err := ioutil.WriteFile("wired.conf", bytes, 0)

	// err := config.Save(true)
	return err, config
}

func setInterfaceUp(config conf.Config) string {
	if config.Name != "" {
		p, _ := config.Path()
		return p
	}
	return ""
}

const (
	ATTACH_PARENT_PROCESS = ^uint32(0) // (DWORD)-1
)

var (
	modkernel32 = syscall.NewLazyDLL("kernel32.dll")

	procAttachConsole = modkernel32.NewProc("AttachConsole")
)

func AttachConsole(dwParentProcess uint32) (ok bool) {
	r0, _, _ := syscall.Syscall(procAttachConsole.Addr(), 1, uintptr(dwParentProcess), 0, 0)
	ok = bool(r0 != 0)
	return
}
