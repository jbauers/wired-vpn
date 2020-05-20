package main

import (
	"net"
	"os"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var serverInterface = os.Getenv("WG_INTERFACE")
var serverCIDR = os.Getenv("WG_NETWORK")
var serverIP = os.Getenv("WG_SERVER_IP")
var serverPort = 51820
var network net.IPNet
var allowedIPs []net.IPNet

// Expiry of Redis keys for WireGuard key rotation.
var keyTTL = time.Duration(10 * time.Second)

// FIXME: Don't panic.
func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	privkey, pubkey := initServer()

	_, ipnet, err := net.ParseCIDR(serverCIDR)
	check(err)

	network = *ipnet
	allowedIPs = append(allowedIPs, network)

	rc := redisClient()
	pubsub := rc.Subscribe("clients")
	pubsub.Receive()

	ch := pubsub.Channel()
	for msg := range ch {
		// FIXME: Get all from Redis.
		var peerList []wgtypes.PeerConfig

		_, config := handleClient(msg.Payload)

		peerList = append(peerList, config)
		updateInterface(serverInterface, serverPort, privkey, peerList)

		rc.Publish(msg.Payload, pubkey)
	}
}
